package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// socket.go represents de AF_XDP socket itself

type Socket struct {
	fd   int
	umem *UMEM

	fillRing ring
	// fill ring holds plain addresses, not full descs, it represents the "window"
	// inside the fill ring, but only addresses, not the full descriptors
	fillDescs []uint64

	rxRing ring
	// rx ring hold. A XDPDesc represents a complete XDP descriptor
	rxDescs []unix.XDPDesc
}

// newSocket creates the AF_XDP socket, attaches it to the given UMEM,
// configures the fill and rx rings, and binds it to a specific interface + queue.
// This is the part that actually turns 'some mmap'd memory" into a working channel that
// receives real packets.
func newSocket(ifindex, queueID int) (*Socket, error) {
	// creates socket
	fd, err := syscall.Socket(unix.AF_XDP, syscall.SOCK_RAW, 0)
	if err != nil {
		return nil, fmt.Errorf("socket AF_XDP: %w", err)
	}

	// Register the UMEM with THIS socket's fd, a UMEM belongs to one
	// "owning" socket, which is the that creates fill/completion queues.
	u, err := newUMEM(fd)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}

	// custom configs for this socket
	if err := unix.SetsockoptInt(fd, unix.SOL_XDP, unix.XDP_UMEM_FILL_RING, ringNumDescs); err != nil {
		return nil, fmt.Errorf("setsockopt XDP_UMEM_FILL_RING: %w", err)
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_XDP, unix.XDP_UMEM_COMPLETION_RING, ringNumDescs); err != nil {
		return nil, fmt.Errorf("setsockopt XDP_UMEM_COMPLETION_RING: %w", err)
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_XDP, unix.XDP_RX_RING, ringNumDescs); err != nil {
		return nil, fmt.Errorf("setsockopt XDP_RX_RING: %w", err)
	}

	// XDP_MMAP_OFFSETS tells us WHERE inside the rings mmap region the producer
	// index, consumer index, and descriptor array each live
	// these offsets are not fixed, they depend on kernel version,
	// so we alway ask for them rather then hardcode the values
	var offsets unix.XDPMmapOffsets
	optlen := unsafe.Sizeof(offsets)
	// Same reasoning as XDP_UMEM_REG: there's no typed wrapper for a
	// struct valued getsockopt in golang.org/x/sys/unix, so we drop to
	// the raw syscall. The key difference from setsockopt: optlen here is
	// a POINTER, not a plain value the kernel writes back how many
	// bytes it actually filled, since in principle it could write less
	// than we allocated space for.
	_, _, errno := unix.Syscall6(
		unix.SYS_GETSOCKOPT,
		uintptr(fd),
		uintptr(unix.SOL_XDP),
		uintptr(unix.XDP_MMAP_OFFSETS),
		uintptr(unsafe.Pointer(&offsets)),
		uintptr(unsafe.Pointer(&optlen)),
		0,
	)
	if errno != 0 {
		return nil, fmt.Errorf("getsockopt XDP_MMAP_OFFSETS: %w", errno)
	}

	// MAP_SHARED: any write on this region is visible to other that also have mapped this region (the kernel).
	// 		This is what mades the ring "shared" on a real UMEM ring, without this flag it would be MAP_PRIVATE,
	// 		and therefore the kernel would never be able to see what was written on the fill ring.
	// MAP_POPULATE: pre-loads all memory pages now, at the mmap call time, without loading on demand (which is default mode)
	// 		This matters since the rx ring is accessed on a high frequency loop. Without MAP_POPULATE,
	// 		the first received package would pay the cost of a lot of page faults, the type of latency that a
	// 		kernel bypass software would like to evict.

	fillMap, err := syscall.Mmap(fd, unix.XDP_UMEM_PGOFF_FILL_RING,
		int(offsets.Fr.Desc)+ringNumDescs*8, //
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED|syscall.MAP_POPULATE,
	)
	if err != nil {
		return nil, fmt.Errorf("mmap fill ring: %w", err)
	}

	rxMap, err := syscall.Mmap(fd, unix.XDP_PGOFF_RX_RING,
		int(offsets.Rx.Desc)+ringNumDescs*int(unsafe.Sizeof(unix.XDPDesc{})),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED|syscall.MAP_POPULATE,
	)
	if err != nil {
		return nil, fmt.Errorf("mmap rx ring: %w", err)
	}

	sock := &Socket{
		fd:   fd,
		umem: u,
		fillRing: ring{
			producer: (*uint32)(unsafe.Pointer(&fillMap[offsets.Fr.Producer])),
			consumer: (*uint32)(unsafe.Pointer(&fillMap[offsets.Fr.Consumer])),
			mask:     ringNumDescs - 1,
		},
		rxRing: ring{
			producer: (*uint32)(unsafe.Pointer(&rxMap[offsets.Rx.Producer])),
			consumer: (*uint32)(unsafe.Pointer(&rxMap[offsets.Rx.Consumer])),
			mask:     ringNumDescs - 1,
		},
	}

	sock.fillDescs = unsafe.Slice(
		(*uint64)(unsafe.Pointer(&fillMap[offsets.Fr.Desc])),
		ringNumDescs,
	)
	sock.rxDescs = unsafe.Slice(
		(*unix.XDPDesc)(unsafe.Pointer(&rxMap[offsets.Rx.Desc])),
		ringNumDescs,
	)

	// Bind ties this socket to a specific interface + hardware queue
	// Only packets landing on that exact queue (and redirected there by
	// our XDP program) will ever reach this socket
	sa := &unix.SockaddrXDP{
		Ifindex: uint32(ifindex),
		QueueID: uint32(queueID),
	}
	if err := unix.Bind(fd, sa); err != nil {
		return nil, fmt.Errorf("bind AF_XDP socket: %w", err)
	}

	return sock, nil
}

// fill pushes every UMEM frame address onto the fill ring, telling the
// kernel "these are all free, use them for the next incomin packets".
// Called once at startup then again each loop iteration for frames we
// just finished reading (see readLoop in main.go)
func (s *Socket) fill(addrs []uint64) int {
	prod := *s.fillRing.producer

	for _, addr := range addrs {
		s.fillDescs[prod&s.fillRing.mask] = addr
		prod++
	}

	*s.fillRing.producer = prod

	return len(addrs)
}

// receive drains the rx ring, returning descriptors the kernel has
// filled with real packet data. Each descriptors Addr points into the
// UMEM; use umem.frameAt(desc.Addr, desc.Len) to get the actual bytes
func (s *Socket) receive() []unix.XDPDesc {
	prod := *s.rxRing.producer
	cons := *s.rxRing.consumer

	n := prod - cons
	if n == 0 {
		return nil
	}

	out := make([]unix.XDPDesc, 0, n)
	for range n {
		out = append(out, s.rxDescs[cons&s.rxRing.mask])
		cons++
	}

	*s.rxRing.consumer = cons

	return out
}

func (s *Socket) close() error {
	s.umem.close()
	return syscall.Close(s.fd)
}

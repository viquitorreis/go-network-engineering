package main

import (
	"fmt"
	"syscall"

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

	// todo

	return sock, nil
}

// fill pushes every UMEM frame address onto the fill ring, telling the
// kernel "these are all free, use them for the next incomin packets".
// Called once at startup then again each loop iteration for frames we
// just finished reading (see readLoop in main.go)
func (s *Socket) fill(addrs []uint64) int {

}

// receive drains the rx ring, returning descriptors the kernel has
// filled with real packet data. Each descriptors Addr points into the
// UMEM; use umem.frameAt(desc.Addr, desc.Len) to get the actual bytes
func (s *Socket) receive() []unix.XDPDesc {

}

func (s *Socket) close() error {
}

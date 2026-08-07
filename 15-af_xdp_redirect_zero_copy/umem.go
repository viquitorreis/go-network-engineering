package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	numFrames = 4096 // total frames the UMEM holds
	frameSize = 2048 // total bytes per frame (must fit the largest expected packet)
)

// UMEM is the shared memory region between kernel and userspace
// where the actual packet bytes live. Both the fill/completion rings (which move
// frame *ownership*) and the rx/tx rings (which move frame *data*)
// reference offsets into this same buffer, nothing is copied between rings,
// only frame addresses are.
type UMEM struct {
	buffer []byte // the raw mmap'd region backing every frame
}

// newUMEM allocates the memory region and registers it with the socket
// via XDP_UMEM_REG setsockopt (ref: https://docs.kernel.org/networking/af_xdp.html#umem).
// This must happen before any ring is configured, since the rings only make sense
// relative to this region.
func newUMEM(fd int) (*UMEM, error) {
	size := numFrames * frameSize

	// - Mmap creates a mapping space on the virtual memmory address
	//  of the calling process. (ref: https://man7.org/linux/man-pages/man2/mmap.2.html)
	// - MAP_ANONYMOUS: not backed by a file, just plain memory
	// - MAP_POPULATE: pre-fault all pages now, so the first packet
	//  received doesn't pay a page-fault penaulty.
	// - syscall.PROT_READ|syscall.PROT_WRITE arent syscalls themselves but protection flags
	// 	 passed to mmap, telling the linux kernel the permissions that process would have
	//	 on the specific mapping region, on this case read and write perms.
	buf, err := syscall.Mmap(-1, 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS|syscall.MAP_POPULATE,
	)
	if err != nil {
		return nil, fmt.Errorf("mmap umem: %w", err)
	}

	reg := unix.XDPUmemReg{
		Addr: uint64(uintptr(unsafe.Pointer(&buf[0]))),
		Len:  uint64(len(buf)),
		Size: uint32(frameSize),
	}

	// there's no typed wrapper for XDP_UMEM_REG in golang.org/x/sys/unix
	// the following option takes a whole struct, so we drop down
	// to the raw setsockopt syscall, passing a pointer to the struct and its
	// size directly.
	// - unix.Syscall6 is a direct syscall with max 6 args, but setsockopt(2)
	// always have 5 args (fd, level, optname, optval, optlen), so the 6th
	// will always be 0 here
	// -
	_, _, errno := unix.Syscall6(
		unix.SYS_SETSOCKOPT,
		uintptr(fd),
		uintptr(unix.SOL_XDP),
		uintptr(unix.XDP_UMEM_REG),
		uintptr(unsafe.Pointer(&reg)), // points to the beginning at the struct "safely" on the kernel layer
		unsafe.Sizeof(reg),            // the kernel needs to know where the bytes from the start pointer finish
		0,
	)
	if errno != 0 {
		// mummap is a linux syscall that removes any virtual memory mapping created before using mmap
		syscall.Munmap(buf)
		return nil, fmt.Errorf("setsockopt XDP_UMEM_REG: %w", errno)
	}

	return &UMEM{
		buffer: buf,
	}, nil
}

// frameAt returns the byte slice for the frame at the given UMEM offset.
// ie the actual packet bytes once a descriptor tells us where they are.
func (u *UMEM) frameAt(addr uint64, length uint32) []byte {
	return u.buffer[addr : addr+uint64(length)]
}

// close removes any virtual memory mapping created for this process before
func (u *UMEM) close() error {
	return syscall.Munmap(u.buffer)
}

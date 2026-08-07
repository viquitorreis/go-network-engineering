package main

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
	// use syscall.Mmap
	return &UMEM{
		buffer: buf,
	}, nil
}

// frameAt returns the byte slice for the frame at the given UMEM offset.
// ie the actual packet bytes once a descriptor tells us where they are.
func (u *UMEM) frameAt(addr uint64, length uint32) []byte {
}

// close removes any virtual memory mapping created for this process before
func (u *UMEM) close() error {
}

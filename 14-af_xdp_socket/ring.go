package main

// ring.go is used for the 4 rings needed for AF_XDP (fill and completion, tx and rx)
// ref: https://docs.kernel.org/networking/af_xdp.html#rings

const ringNumDescs = 64 // number of slots in each ring

type ring struct {
	producer *uint32
	consumer *uint32
	mask     uint32 // numDecs - 1, used for wrap-around indexing
}

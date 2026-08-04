package main

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux counter xdp_count_by_port.c -- -I. -Iheaders

import (
	"fmt"
	"log"
)

func main() {
	// todo: https://ebpf-go.dev/guides/getting-started/#compile-ebpf-c-and-generate-scaffolding-using-bpf2go
}

func printCounts(objs *counterObjects) {
	var key uint16
	var value uint64
	iter := objs.PortCount.Iterate()

	fmt.Println("--- counting by port ---")
	for iter.Next(&key, &value) {
		fmt.Printf("port %d: %d packages\n", key, value)
	}
	if err := iter.Err(); err != nil {
		log.Printf("error iterating map: %v", err)
	}
}

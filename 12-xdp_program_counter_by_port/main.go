package main

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux counter xdp_count_by_port.c -- -I. -Iheaders

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cilium/ebpf/link"
)

func main() {
	var ifaceName string

	if len(os.Args) > 0 {
		ifaceName = os.Args[1]
	} else {
		ifaceName = "lo"
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Fatalf("interface %s not found: %v", ifaceName, err)
	}

	var objs counterObjects
	if err := loadCounterObjects(&objs, nil); err != nil {
		log.Fatalf("error loading eBPF objects (verifier rejected?): %v", err)
	}
	defer objs.Close()

	xdpLink, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.CountByPort,
		Interface: iface.Index,
	})
	if err != nil {
		log.Fatalf("error attaching XDP: %v", err)
	}
	defer xdpLink.Close()

	log.Printf("XDP attached in %s, counting UDP packets by port...", ifaceName)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			log.Println("finishing, detaching XDP...")
			return
		case <-ticker.C:
			printCounts(&objs)
		}
	}
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

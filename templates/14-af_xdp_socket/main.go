package main

//go:generate

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf/link"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: sudo ./af_xdp_receiver <ifname>")
	}
	ifname := os.Args[1]

	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		log.Fatalf("interface %s not found: %v", ifname, err)
	}

	// Load the compiled XDP program (from bpf2go) and attach it to the
	// interface. This is the kernel-side piece from Part 1 — it decides
	// WHICH packets get redirected, our socket just receives them.
	var objs afxdpObjects
	if err := loadAfxdpObjects(&objs, nil); err != nil {
		log.Fatalf("loading eBPF objects: %v", err)
	}
	defer objs.Close()

	l, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.XdpRedirectToSocket,
		Interface: iface.Index,
	})
	if err != nil {
		log.Fatalf("attaching XDP: %v", err)
	}
	defer l.Close()

	// newSocket now creates its own UMEM internally, bound to the
	// socket's fd no external umem parameter (see the fix above).
	const queueID = 0
	sock, err := newSocket(iface.Index, queueID)
	if err != nil {
		log.Fatalf("creating AF_XDP socket: %v", err)
	}
	defer sock.close()

	// Register this socket into the XSKMAP so the XDP program knows
	// "queue 0 has an active AF_XDP socket, redirect here".
	if err := objs.XsksMap.Put(uint32(queueID), uint32(sock.fd)); err != nil {
		log.Fatalf("registering socket in xsks_map: %v", err)
	}

	// Prime the fill ring before the loop starts, without this, the
	// kernel has nowhere to put the fist packets that arrive.
	initial := make([]uint64, numFrames)
	for i := range initial {
		initial[i] = uint64(i * frameSize)
	}

	sock.fill(initial)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		os.Exit(0)
	}()

	log.Printf("AF_XDP socket bound to %s queue %d, waiting for packets...", ifname, queueID)

	for {
		descs := sock.receive()
		for _, d := range descs {
			frame := sock.umem.frameAt(d.Addr, d.Len)
			log.Printf("packet: %d bytes, first bytes: %x", d.Len, frame[:min(16, len(frame))])
		}

		if len(descs) > 0 {
			// Return the frames we just read back to the fill ring so
			// the kernel can reuse them for future packets.
			addrs := make([]uint64, len(descs))
			for i, d := range descs {
				addrs[i] = d.Addr
			}

			sock.fill(addrs)
		}
	}
}

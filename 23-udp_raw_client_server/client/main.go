package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"udpraw/types"
)

const numMsg = 100

func main() {
	addr, err := net.ResolveUDPAddr("udp4", ":8080")
	if err != nil {
		log.Fatalf("err resolving udp addr: %v", err)
	}

	udpDial, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		log.Fatalf("err dialing udp: %v", err)
	}
	defer udpDial.Close()

	done := make(chan struct{}, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	total, dropped := 0, 0

	go func() {
		<-sigCh
		done <- struct{}{}

		log.Printf("Total msg: %d. Total dropped: %d", total, dropped)

		log.Println("bye")
	}()

	select {
	case <-done:
		return
	default:
	}

	received := make(map[int]bool)

	for n := range numMsg {
		msg := types.Message{Cmd: types.MSGCmd, Content: n}.ToDatagram()
		if _, err := udpDial.Write(msg); err != nil {
			log.Println("err writing to server", err)
			return
		}

		total++
	}

	udpDial.SetReadDeadline(time.Now().Add(time.Second * 2)) // dead line read

	for {
		// pseudo conn -> reading from it
		buf := make([]byte, 512)
		n, _, err := udpDial.ReadFromUDP(buf)
		if err != nil {
			// timeout or any error
			break
		}

		msgNum := types.ParseMsg(buf[:n])
		received[msgNum.Content] = true

		log.Println("server reply", string(buf))
	}

	// not the best, ok...
	for n := 0; n < numMsg; n++ {
		if !received[n] {
			dropped++
		}
	}

	log.Printf("Total msg: %d. Total dropped: %d", total, dropped)

	time.Sleep(time.Second * 5)
}

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	nodeName  = "NodeA"
	localAddr = "127.0.0.1:9001"
	peerAddr  = "127.0.0.1:9002"
)

func main() {
	localUDPAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		panic(err)
	}

	peerUDPAddr, err := net.ResolveUDPAddr("udp", peerAddr)
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", localUDPAddr)
	if err != nil {
		panic(err)
	}

	quit := make(chan struct{})
	defer func() {
		close(quit)
		_ = conn.Close()
	}()

	fmt.Printf("%s listening on %s, peer is %s\n", nodeName, localAddr, peerAddr)
	fmt.Println("Press Ctrl+C to stop.")

	go receiveLoop(conn, quit)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	sendHello(conn, peerUDPAddr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			sendHello(conn, peerUDPAddr)
		case <-stop:
			fmt.Printf("\n%s stopped\n", nodeName)
			return
		}
	}
}

func receiveLoop(conn *net.UDPConn, quit <-chan struct{}) {
	buf := make([]byte, 1024)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-quit:
				return
			default:
				fmt.Printf("read error: %v\n", err)
				return
			}
		}

		fmt.Printf("received from %s: %s\n", remoteAddr.String(), string(buf[:n]))
	}
}

func sendHello(conn *net.UDPConn, peer *net.UDPAddr) {
	message := []byte("Hello")
	n, err := conn.WriteToUDP(message, peer)
	if err != nil {
		fmt.Printf("send error: %v\n", err)
		return
	}

	fmt.Printf("sent %d bytes to %s: %s\n", n, peer.String(), string(message))
}

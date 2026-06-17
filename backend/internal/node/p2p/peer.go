package p2p

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"netweaver-backend/pkg/config"
)

type Config struct {
	NodeName  string
	LocalAddr string
	PeerAddr  string
	Message   string
	Interval  time.Duration
}

func Profile(name string) (Config, error) {
	switch strings.ToLower(name) {
	case "a", "nodea":
		return Config{
			NodeName:  "NodeA",
			LocalAddr: config.DefaultNodeAAddr,
			PeerAddr:  config.DefaultNodeBAddr,
			Message:   config.DefaultP2PMessage,
			Interval:  2 * time.Second,
		}, nil
	case "b", "nodeb":
		return Config{
			NodeName:  "NodeB",
			LocalAddr: config.DefaultNodeBAddr,
			PeerAddr:  config.DefaultNodeAAddr,
			Message:   config.DefaultP2PMessage,
			Interval:  2 * time.Second,
		}, nil
	default:
		return Config{}, fmt.Errorf("unknown p2p profile %q", name)
	}
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.NodeName == "" {
		cfg.NodeName = "Node"
	}
	if cfg.Message == "" {
		cfg.Message = config.DefaultP2PMessage
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
	}

	localUDPAddr, err := net.ResolveUDPAddr("udp", cfg.LocalAddr)
	if err != nil {
		return err
	}

	peerUDPAddr, err := net.ResolveUDPAddr("udp", cfg.PeerAddr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", localUDPAddr)
	if err != nil {
		return err
	}

	quit := make(chan struct{})
	defer func() {
		close(quit)
		_ = conn.Close()
	}()

	fmt.Printf("%s listening on %s, peer is %s\n", cfg.NodeName, cfg.LocalAddr, cfg.PeerAddr)
	fmt.Println("Press Ctrl+C to stop.")

	go receiveLoop(conn, quit)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	sendMessage(conn, peerUDPAddr, cfg.Message)

	for {
		select {
		case <-ticker.C:
			sendMessage(conn, peerUDPAddr, cfg.Message)
		case <-ctx.Done():
			fmt.Printf("\n%s stopped\n", cfg.NodeName)
			return nil
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

func sendMessage(conn *net.UDPConn, peer *net.UDPAddr, message string) {
	data := []byte(message)
	n, err := conn.WriteToUDP(data, peer)
	if err != nil {
		fmt.Printf("send error: %v\n", err)
		return
	}

	fmt.Printf("sent %d bytes to %s: %s\n", n, peer.String(), string(data))
}

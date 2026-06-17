package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"netweaver-backend/internal/node/p2p"
	noderuntime "netweaver-backend/internal/node/runtime"
	"netweaver-backend/internal/node/tun"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 || strings.HasPrefix(args[0], "-") || args[0] == "tun" {
		if len(args) > 0 && args[0] == "tun" {
			args = args[1:]
		}
		if err := tun.Run(args); err != nil {
			log.Fatal(err)
		}
		return
	}

	switch args[0] {
	case "run":
		if err := runAgent(args[1:]); err != nil {
			log.Fatal(err)
		}
	case "p2p":
		if err := runP2P(args[1:]); err != nil {
			log.Fatal(err)
		}
	default:
		printUsage()
		os.Exit(2)
	}
}

func runAgent(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return noderuntime.Run(ctx, args)
}

func runP2P(args []string) error {
	fs := flag.NewFlagSet("node p2p", flag.ExitOnError)
	profile := fs.String("profile", "A", "p2p profile: A or B")
	name := fs.String("name", "", "node display name")
	localAddr := fs.String("local", "", "local UDP address")
	peerAddr := fs.String("peer", "", "peer UDP address")
	message := fs.String("message", "", "message sent to peer")
	interval := fs.Duration("interval", 2*time.Second, "send interval")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := p2p.Profile(*profile)
	if err != nil {
		return err
	}
	if *name != "" {
		cfg.NodeName = *name
	}
	if *localAddr != "" {
		cfg.LocalAddr = *localAddr
	}
	if *peerAddr != "" {
		cfg.PeerAddr = *peerAddr
	}
	if *message != "" {
		cfg.Message = *message
	}
	if *interval > 0 {
		cfg.Interval = *interval
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return p2p.Run(ctx, cfg)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/node [tun flags]")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/node tun [tun flags]")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/node run -controller http://127.0.0.1:8080 -tun-name tuno -tun-auto-config -tun-auto-cleanup")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/node p2p -profile A")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/node p2p -profile B")
}

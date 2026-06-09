package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"netweaver-backend/internal/controller/api"
	"netweaver-backend/internal/controller/relay"
	"netweaver-backend/internal/controller/state"
	"netweaver-backend/pkg/config"
)

func main() {
	addr := flag.String("addr", config.DefaultControllerAddr, "HTTP listen address")
	relayAddr := flag.String("relay-addr", fmt.Sprintf(":%d", config.DefaultRelayPort), "UDP relay listen address; empty disables relay")
	flag.Parse()

	if strings.TrimSpace(*relayAddr) != "" {
		relayServer, err := relay.Listen(strings.TrimSpace(*relayAddr))
		if err != nil {
			log.Fatalf("start udp relay: %v", err)
		}
		defer relayServer.Close()

		go func() {
			log.Printf("udp relay listening on %s", relayServer.Addr().String())
			if err := relayServer.Serve(context.Background()); err != nil {
				log.Printf("udp relay stopped: %v", err)
			}
		}()
	}

	router := api.NewRouter(state.NewRegistry())
	if err := router.Run(*addr); err != nil {
		log.Fatal(err)
	}
}

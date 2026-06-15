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

func logControllerStartupWarnings() {
	if config.UsesDefaultDashboardCredential() {
		log.Printf("warning: dashboard is still using default credentials (%s / %s); set %s and %s before real deployment", config.DefaultDashboardUsername, config.DefaultDashboardPassword, config.EnvDashboardUsername, config.EnvDashboardPassword)
	}

	if config.UsesDefaultJWTSecret() {
		log.Printf("warning: dashboard JWT is still using the development default; set %s before real deployment", config.EnvJWTSecret)
	}

	if config.UsesDefaultNodePSK() {
		log.Printf("warning: node API is still using the development PSK; set %s before real deployment", config.EnvNodePSK)
	}

	if config.RelayAddr() == config.DefaultRelayAddr {
		log.Printf("warning: relay public address is still %s; set %s to the controller server IP before multi-machine deployment", config.DefaultRelayAddr, config.EnvRelayAddr)
	}

	if config.UsesDefaultSTUNServers() {
		log.Printf("warning: STUN servers are still using the default public list; set %s if your environment needs a dedicated STUN plan", config.EnvSTUNServers)
	}
}

func main() {
	addr := flag.String("addr", config.DefaultControllerAddr, "HTTP listen address")
	relayAddr := flag.String("relay-addr", fmt.Sprintf(":%d", config.DefaultRelayPort), "UDP relay listen address; empty disables relay")
	flag.Parse()

	logControllerStartupWarnings()

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

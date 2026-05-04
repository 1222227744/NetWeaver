package main

import (
	"flag"
	"log"

	"netweaver-backend/internal/controller/api"
	"netweaver-backend/internal/controller/state"
	"netweaver-backend/pkg/config"
)

func main() {
	addr := flag.String("addr", config.DefaultControllerAddr, "HTTP listen address")
	flag.Parse()

	router := api.NewRouter(state.NewRegistry())
	if err := router.Run(*addr); err != nil {
		log.Fatal(err)
	}
}

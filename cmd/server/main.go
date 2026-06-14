package main

import (
	"flag"
	"log"

	"github.com/fdcs99/biligo/internal/config"
	"github.com/fdcs99/biligo/internal/httpapi"
	"github.com/fdcs99/biligo/internal/store"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := store.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	router := httpapi.NewRouter(db)
	log.Printf("biligo server listening on %s", cfg.Server.Addr)
	if err := router.Run(cfg.Server.Addr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}

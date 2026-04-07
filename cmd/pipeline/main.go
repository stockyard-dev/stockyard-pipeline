package main

import (
	"flag"
	"fmt"
	"github.com/stockyard-dev/stockyard-pipeline/internal/server"
	"github.com/stockyard-dev/stockyard-pipeline/internal/store"
	"log"
	"net/http"
	"os"
)

func main() {
	portFlag := flag.String("port", "", "")
	dataFlag := flag.String("data", "", "")
	flag.Parse()
	port := os.Getenv("PORT")
	if port == "" {
		port = "9140"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./pipeline-data"
	}

	if *portFlag != "" {
		port = *portFlag
	}
	if *dataFlag != "" {
		dataDir = *dataFlag
	}
	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("pipeline: open database: %v", err)
	}
	defer db.Close()

	srv := server.New(db, server.DefaultLimits(), dataDir)

	fmt.Printf("\n  Pipeline — Self-hosted data pipeline orchestrator\n")
	fmt.Printf("  Questions? hello@stockyard.dev\n")
	fmt.Printf("  ─────────────────────────────────\n")
	fmt.Printf("  Dashboard:  http://localhost:%s/ui\n", port)
	fmt.Printf("  API:        http://localhost:%s/api\n", port)
	fmt.Printf("  Data:       %s\n", dataDir)
	fmt.Printf("  ─────────────────────────────────\n\n")

	log.Printf("pipeline: listening on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("pipeline: %v", err)
	}
}

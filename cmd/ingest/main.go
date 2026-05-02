package main

import (
	"context"
	"flag"
	"log"
	"proferti-be/internal/config"
	"proferti-be/internal/db"
	"proferti-be/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	start := flag.String("start", "", "Tanggal mulai (e.g. 2026-01-01)")
	end := flag.String("end", "", "Tanggal akhir (e.g. 2026-01-31)")
	flag.Parse()

	log.Printf("🚀 Menjalankan Ingest Data Banjir (GFM)... Rentang: %s s/d %s\n", *start, *end)
	err = worker.RunFullIngestionCycle(ctx, pool, *start, *end)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("✅ Ingest data banjir selesai.")
}

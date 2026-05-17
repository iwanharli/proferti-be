package main

import (
	"context"
	"flag"
	"log"
	"time"

	"proferti-be/internal/config"
	"proferti-be/internal/db"
	"proferti-be/internal/worker"
)

func main() {
	start := flag.String("start", "", "Tanggal mulai YYYY-MM-DD (kosong = 60 hari terakhir)")
	end := flag.String("end", "", "Tanggal akhir  YYYY-MM-DD (kosong = hari ini)")
	flag.Parse()

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

	if *start == "" {
		log.Println("🚀 Ingest GFM — mode: 60 hari terakhir (mode harian)")
	} else {
		log.Printf("🚀 Ingest GFM — mode: backfill %s s/d %s", *start, *end)
	}

	t0 := time.Now()
	if err := worker.RunFullIngestionCycle(ctx, pool, *start, *end); err != nil {
		log.Fatal(err)
	}
	log.Printf("✅ Selesai dalam %s.", time.Since(t0).Round(time.Second))
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"proferti-be/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load Environment
	_ = godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://iwanharli@localhost:5432/db_proferti?sslmode=disable"
	}

	// 2. Database Connection
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// 3. Define Range (Last 60 Days)
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -60)
	
	startStr := startTime.Format("2006-01-02")
	endStr := endTime.Format("2006-01-02")

	fmt.Printf("🚀 Starting Real GFM Ingestion Cycle...\n")
	fmt.Printf("📅 Range: %s to %s (Last 60 days)\n", startStr, endStr)
	fmt.Println("🛰️  Fetching real Sentinel-1 flood data from Copernicus GFM API...")

	// 4. Run Ingestion
	err = worker.RunFullIngestionCycle(ctx, pool, startStr, endStr)
	
	if err != nil {
		log.Fatalf("❌ Ingestion process failed: %v", err)
	}

	fmt.Println("\n✅ Real GFM Ingestion completed successfully!")
}

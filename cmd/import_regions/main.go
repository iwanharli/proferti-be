package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Open CSV
	csvFile, err := os.Open("internal/t_regions_202605011823.csv")
	if err != nil {
		log.Fatalf("Unable to open CSV: %v", err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = ';' // Semicolon delimiter
	reader.LazyQuotes = true

	// Skip header
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Error reading header: %v", err)
	}
	fmt.Printf("Headers: %v\n", header)

	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading record: %v", err)
			continue
		}

		// code;name;lat;lng;path;status;geom;created_at;updated_at
		code := record[0]
		name := record[1]
		lat, _ := strconv.ParseFloat(record[2], 64)
		lng, _ := strconv.ParseFloat(record[3], 64)
		path := record[4]
		status := record[5]
		geom := record[6] // WKT MultiPolygon
		createdAtStr := record[7]
		updatedAtStr := record[8]

		createdAt, _ := time.Parse("2006-01-02 15:04:05", createdAtStr)
		updatedAt, _ := time.Parse("2006-01-02 15:04:05", updatedAtStr)

		// Insert with geom casting
		query := `
			INSERT INTO regions (code, name, lat, lng, path, status, geom, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, ST_GeomFromText($7, 4326), $8, $9)
			ON CONFLICT (code) DO UPDATE SET
				name = EXCLUDED.name,
				lat = EXCLUDED.lat,
				lng = EXCLUDED.lng,
				path = EXCLUDED.path,
				status = EXCLUDED.status,
				geom = EXCLUDED.geom,
				updated_at = EXCLUDED.updated_at
		`

		_, err = conn.Exec(ctx, query, code, name, lat, lng, path, status, geom, createdAt, updatedAt)
		if err != nil {
			log.Printf("Error inserting record %s: %v", code, err)
			continue
		}

		count++
		if count%50 == 0 {
			fmt.Printf("Processed %d records...\n", count)
		}
	}

	fmt.Printf("Successfully imported %d regions!\n", count)
}

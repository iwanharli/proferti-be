package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"proferti-be/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type RegionData struct {
	Kode   string
	Name   string
	Lat    float64
	Lng    float64
	Path   string
	Status int16
}

func main() {
	_ = godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://iwanharli@localhost:5432/db_proferti?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	fmt.Println("Starting regions seeder...")

	// NEW: Truncate and restart identity
	fmt.Println("Cleaning up regions table...")
	_, err = pool.Exec(ctx, "TRUNCATE regions RESTART IDENTITY CASCADE")
	if err != nil {
		log.Fatalf("Failed to truncate regions: %v", err)
	}

	// Directories to scan (Limited to prov and kab to save disk space)
	dirs := []string{
		"data/regions/prov",
		"data/regions/kab",
	}

	var allRegions []RegionData

	for _, dir := range dirs {
		fmt.Printf("Scanning directory: %s\n", dir)
		files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
		if err != nil {
			log.Printf("Error scanning %s: %v", dir, err)
			continue
		}

		for _, file := range files {
			regions, err := parseSQLFile(file)
			if err != nil {
				log.Printf("    Error parsing %s: %v", file, err)
			}
			allRegions = append(allRegions, regions...)
		}
	}

	// Sort by kode ASC
	fmt.Printf("Sorting %d regions by kode...\n", len(allRegions))
	sort.Slice(allRegions, func(i, j int) bool {
		return allRegions[i].Kode < allRegions[j].Kode
	})

	fmt.Println("Inserting regions...")
	totalImported := 0
	for _, r := range allRegions {
		wkt, err := pathToWKT(r.Path)
		if err != nil {
			// log.Printf("      Failed to convert path to WKT for %s: %v", r.Kode, err)
			continue
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO regions (kode, name, lat, lng, path, status, geom)
			VALUES ($1, $2, $3, $4, $5, 1, ST_GeomFromText($6, 4326))
		`, r.Kode, r.Name, r.Lat, r.Lng, r.Path, wkt)

		if err != nil {
			if strings.Contains(err.Error(), "non-closed rings") {
				continue
			}
			log.Printf("      Failed to insert %s: %v", r.Kode, err)
			continue
		}
		totalImported++
		if totalImported%500 == 0 {
			fmt.Printf("  Inserted %d records...\n", totalImported)
		}
	}

	fmt.Printf("\nDone! Seeded %d regions into the regions table.\n", totalImported)

	// NEW: Execute demo data seeder
	fmt.Println("\nSeeding demo data (Users, Developers, Projects)...")
	demoSeederPath := filepath.Join("cmd", "dummy_seed", "seeder_baru.sql")
	demoContent, err := os.ReadFile(demoSeederPath)
	if err != nil {
		log.Printf("Warning: Could not read demo seeder file at %s: %v", demoSeederPath, err)
	} else {
		_, err = pool.Exec(ctx, string(demoContent))
		if err != nil {
			log.Printf("Error executing demo seeder: %v", err)
		} else {
			fmt.Println("Demo data seeded successfully!")
		}
	}

	// Triggering real GFM Ingestion for the last 2 months (default)
	fmt.Println("\nTriggering real GFM Ingestion (API Process) for the last 2 months...")
	err = worker.RunFullIngestionCycle(ctx, pool, "", "")
	if err != nil {
		log.Printf("Warning: GFM Ingestion failed: %v", err)
	} else {
		fmt.Println("GFM Ingestion completed successfully!")
	}
}

func parseSQLFile(filename string) ([]RegionData, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Regex to find VALUES ('kode', 'name', lat, lng, 'path')
	re := regexp.MustCompile(`\('([^']+)',\s*'([^']+)',\s*([\d.-]+),\s*([\d.-]+),\s*'([^']+)'\)`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	var regions []RegionData
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		kode := match[1]
		name := match[2]
		var lat, lng float64
		fmt.Sscanf(match[3], "%f", &lat)
		fmt.Sscanf(match[4], "%f", &lng)
		path := match[5]

		regions = append(regions, RegionData{
			Kode: kode,
			Name: name,
			Lat:  lat,
			Lng:  lng,
			Path: path,
		})
	}

	return regions, nil
}

func pathToWKT(path string) (string, error) {
	var raw interface{}
	if err := json.Unmarshal([]byte(path), &raw); err != nil {
		return "", err
	}

	var multiPoly [][][][]float64

	if l1, ok := raw.([]interface{}); ok {
		if len(l1) == 0 { return "", fmt.Errorf("empty path") }
		if l2, ok := l1[0].([]interface{}); ok {
			if len(l2) == 0 { return "", fmt.Errorf("empty path L2") }
			if _, ok := l2[0].(float64); ok {
				// Level 2 (not expected)
			} else if l3, ok := l2[0].([]interface{}); ok {
				if len(l3) == 0 { return "", fmt.Errorf("empty path L3") }
				if _, ok := l3[0].(float64); ok {
					// Level 3: Polygon
					var poly [][][]float64
					if err := json.Unmarshal([]byte(path), &poly); err == nil {
						multiPoly = append(multiPoly, poly)
					}
				} else {
					// Level 4: MultiPolygon
					if err := json.Unmarshal([]byte(path), &multiPoly); err != nil {
						return "", err
					}
				}
			}
		}
	}

	if len(multiPoly) == 0 {
		return "", fmt.Errorf("invalid path structure")
	}

	var sb strings.Builder
	sb.WriteString("MULTIPOLYGON(")
	for i, poly := range multiPoly {
		if i > 0 { sb.WriteString(",") }
		sb.WriteString("(")
		for j, ring := range poly {
			if j > 0 { sb.WriteString(",") }
			sb.WriteString("(")
			for k, pt := range ring {
				if k > 0 { sb.WriteString(",") }
				if len(pt) >= 2 {
					sb.WriteString(fmt.Sprintf("%f %f", pt[1], pt[0]))
				}
			}
			sb.WriteString(")")
		}
		sb.WriteString(")")
	}
	sb.WriteString(")")

	return sb.String(), nil
}

package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL wajib diisi (lihat .env.example)")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("db ping: ", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	dir := "migrations"
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		if err := goose.Up(db, dir); err != nil {
			log.Fatal(err)
		}
		log.Println("goose: up OK")
	case "down":
		if err := goose.Down(db, dir); err != nil {
			log.Fatal(err)
		}
		log.Println("goose: down OK (satu versi)")
	case "status":
		if err := goose.Status(db, dir); err != nil {
			log.Fatal(err)
		}
	case "version":
		if err := goose.Version(db, dir); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("perintah tidak dikenal: %q — pakai: up | down | status | version", cmd)
	}
}

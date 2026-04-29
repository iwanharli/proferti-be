package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	db := os.Getenv("DATABASE_URL")
	if db == "" {
		return nil, fmt.Errorf("DATABASE_URL wajib diisi")
	}

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	return &Config{DatabaseURL: db, HTTPAddr: addr}, nil
}

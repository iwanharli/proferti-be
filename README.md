# Proferti Backend (Go)

Backend layanan marketplace properti Proferti menggunakan bahasa pemrograman Go.

## Teknologi
- **Framework**: [Chi Router v5](https://github.com/go-chi/chi)
- **Database**: PostgreSQL with [pgx v5](https://github.com/jackc/pgx)
- **Migrations**: [Goose](https://github.com/pressly/goose)
- **Environment**: [godotenv](https://github.com/joho/godotenv)

## Struktur Database
Proyek ini menggunakan standarisasi prefix untuk tabel:
- `t_` : Tabel utama bisnis (misal: `t_projects`, `t_developers`, `t_users`).
- `na_`: Tabel khusus NextAuth/Auth.js (misal: `na_accounts`, `na_sessions`).

## Persiapan
1. Pastikan Go (v1.23+) dan PostgreSQL sudah terinstal.
2. Buat database di PostgreSQL:
   ```sql
   CREATE DATABASE db_proferti;
   ```
3. Salin `.env.example` menjadi `.env` dan sesuaikan `DATABASE_URL`:
   ```bash
   DATABASE_URL=postgresql://user:password@localhost:5432/db_proferti?sslmode=disable
   ```

## Menjalankan Migrasi
Untuk membuat struktur tabel:
```bash
go run cmd/migrate/main.go up
```
Untuk melihat status migrasi:
```bash
go run cmd/migrate/main.go status
```

## Memasukkan Dummy Data (Seeding)
Gunakan script SQL yang sudah disediakan:
```bash
psql -d db_proferti -f seed.sql
```

## Menjalankan Server
```bash
go run cmd/server/main.go
```
API akan berjalan di `http://localhost:8080` (default).

## Endpoint API
- `GET /health` - Cek status server
- `GET /api/projects` - List semua proyek
- `GET /api/projects/{id}` - Detail proyek
- `GET /api/developers` - List developer

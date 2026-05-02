package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"proferti-be/internal/config"
	"proferti-be/internal/db"
	"proferti-be/internal/handlers"
	"proferti-be/internal/worker"

	"github.com/robfig/cron/v3"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	api := &handlers.API{Pool: pool}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000", "http://localhost:3001",
			"http://127.0.0.1:3000", "http://127.0.0.1:3001",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
	}))

	r.Get("/health", handlers.Health)

	r.Route("/api", func(rt chi.Router) {
		// ── Auth ──────────────────────────────────────────────────
		rt.Post("/auth/login", api.Login)
		rt.Post("/auth/register", api.Register)
		rt.Post("/auth/oauth-sync", api.OAuthSync)

		// ── Projects ──────────────────────────────────────────────
		rt.Get("/projects/meta", api.GetProjectsMeta)
		rt.Get("/regions/geojson", api.GetRegionsGeoJSON)
		rt.Get("/regions/detect", api.DetectRegion)
		
		// ── GFM Flood Monitoring ─────────────────────────────────
		rt.Get("/gfm/scenes", api.ListGFMScenes)
		rt.Get("/gfm/scenes/geojson", api.GetGFMScenesGeoJSON)
		rt.Get("/gfm/summary", api.GetGFMSummary)
		rt.Get("/gfm/risk-summary", api.GetGFMRiskSummary)
		rt.Post("/gfm/ingest", api.TriggerGFMIngestion)
		rt.Get("/flood-mvt/{z}/{x}/{y}.pbf", api.GetFloodMVT)

		rt.Get("/projects", api.ListProjects)
		rt.Post("/projects", api.CreateProject)
		rt.Get("/projects/{id}", api.GetProject)
		rt.Put("/projects/{id}", api.UpdateProject)
		rt.Delete("/projects/{id}", api.DeleteProject)

		// Unit Types (scoped to project)
		rt.Get("/projects/{id}/unit-types", api.ListUnitTypes)
		rt.Post("/projects/{id}/unit-types", api.CreateUnitType)

		// Gallery (scoped to project)
		rt.Post("/projects/{id}/gallery", api.AddGallery)

		// Unit (scoped to project)
		rt.Get("/projects/{id}/units", api.ListProjectUnits)
		rt.Post("/projects/{id}/units", api.CreateUnit)

		// ── Unit (standalone update) ──────────────────────────────
		rt.Patch("/units/{id}/status", api.PatchUnitStatus)

		// ── Unit Types (standalone update/delete/get) ─────────────────
		rt.Get("/unit-types/{id}", api.GetUnitType)
		rt.Put("/unit-types/{id}", api.UpdateUnitType)
		rt.Delete("/unit-types/{id}", api.DeleteUnitType)

		// ── Gallery (standalone delete) ───────────────────────────
		rt.Delete("/gallery/{id}", api.DeleteGallery)

		// ── Developers ────────────────────────────────────────────
		rt.Get("/developers", api.ListDevelopers)
		rt.Get("/developers/me", api.GetMyDeveloper)
		rt.Get("/developers/{id}", api.GetDeveloper)
		rt.Post("/developers/register", api.RegisterDeveloper)

		// ── Leads ─────────────────────────────────────────────────
		rt.Post("/leads", api.SubmitLead)
		rt.Get("/leads", api.ListMyLeads)
		rt.Patch("/leads/{id}/status", api.PatchLeadStatus)
		rt.Get("/leads/{id}/notes", api.ListLeadNotes)
		rt.Post("/leads/{id}/notes", api.AddLeadNote)

		// ── Locations ─────────────────────────────────────────────
		rt.Get("/locations", api.ListLocations)

		// ── Upload ────────────────────────────────────────────────
		rt.Post("/upload", api.UploadFile)
	})

	// Serve static files from uploads directory
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "uploads"))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(filesDir)))

	// ── Automated Flood Monitoring (Daily at 02:00 AM) ────────
	c := cron.New()
	_, err = c.AddFunc("0 2 * * *", func() {
		log.Println("⏰ Starting scheduled daily flood ingestion cycle...")
		worker.RunFullIngestionCycle(context.Background(), pool, "", "")
	})
	if err != nil {
		log.Printf("❌ Failed to schedule cron job: %v", err)
	} else {
		c.Start()
		log.Println("✅ Daily flood monitoring scheduled for 02:00 AM")
	}

	log.Printf("proferti-be listening on %s", cfg.HTTPAddr)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shCtx)
}

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/joho/godotenv"
	"github.com/you/auth-service/internal/db"
	"github.com/you/auth-service/internal/handlers"
	authmw "github.com/you/auth-service/internal/middleware"
	"github.com/you/auth-service/internal/mailer"
)

func main() {
	_ = godotenv.Load() // load .env if present (ignored in production)

	ctx := context.Background()

	// Database
	database, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Pool.Close()

	// Background cleanup: inactive devices every 24h
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			if err := database.CleanupInactiveDevices(ctx); err != nil {
				log.Printf("cleanup error: %v", err)
			}
		}
	}()

	h := &handlers.Handler{
		DB:     database,
		Mailer: mailer.New(),
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)

	// Health check (used by Render)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Public routes with rate limiting
	r.Group(func(r chi.Router) {
		// 5 requests / minute / IP on auth endpoints
		r.Use(httprate.LimitByIP(5, time.Minute))
		r.Post("/auth/request-login", h.RequestLogin)
		r.Get("/auth/login", h.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authmw.RequireAuth(database))
		r.Post("/auth/logout", h.Logout)
		r.Get("/auth/me", h.Me)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("auth-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

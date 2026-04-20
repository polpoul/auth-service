package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/joho/godotenv"
	"github.com/you/auth-service/internal/allowlist"
	"github.com/you/auth-service/internal/db"
	"github.com/you/auth-service/internal/handlers"
	authmw "github.com/you/auth-service/internal/middleware"
	"github.com/you/auth-service/internal/mailer"
)

func corsMiddleware(allowedOrigins string) func(http.Handler) http.Handler {
	origins := strings.Split(allowedOrigins, ",")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			for _, allowed := range origins {
				if strings.TrimSpace(allowed) == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					break
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	database, err := db.New(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			if err := database.CleanupInactiveDevices(context.Background()); err != nil {
				log.Printf("cleanup error: %v", err)
			}
		}
	}()

	// Chargement de l'allowlist — retourne un *Allowlist (pointeur)
	// La vérification relit le fichier à chaque requête (hot reload)
	al, err := allowlist.Load(os.Getenv("ALLOWLIST_PATH"))
	if err != nil {
		log.Fatalf("failed to load allowlist: %v", err)
	}
	if al != nil {
		log.Printf("allowlist configured at: %s (hot reload enabled)", os.Getenv("ALLOWLIST_PATH"))
	} else {
		log.Printf("no allowlist configured, all emails allowed")
	}

	h := &handlers.Handler{
		DB:        database,
		Mailer:    mailer.New(),
		Allowlist: al, // *allowlist.Allowlist — nil si pas de fichier configuré
	}

	allowedOrigins := os.Getenv("CORS_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "https://vivalink.top"
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(corsMiddleware(allowedOrigins))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(5, time.Minute))
		r.Post("/auth/request-login", h.RequestLogin)
		r.Get("/auth/login", h.Login)
	})

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
	log.Printf("CORS origins: %s", allowedOrigins)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

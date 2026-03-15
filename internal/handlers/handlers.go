package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/you/auth-service/internal/db"
	"github.com/you/auth-service/internal/mailer"
	authmw "github.com/you/auth-service/internal/middleware"
	"github.com/you/auth-service/internal/tokens"
)

type Handler struct {
	DB     *db.DB
	Mailer *mailer.Mailer
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// POST /auth/request-login
func (h *Handler) RequestLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" || !strings.Contains(email, "@") {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if min := 500 * time.Millisecond; elapsed < min {
			time.Sleep(min - elapsed)
		}
	}()

	if _, err := h.DB.FindOrCreateUser(r.Context(), email); err != nil {
		log.Printf("FindOrCreateUser error: %v", err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "if this email exists, a link has been sent"})
		return
	}

	magicToken, err := tokens.GenerateMagicToken(email, os.Getenv("JWT_SECRET"))
	if err != nil {
		log.Printf("GenerateMagicToken error: %v", err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "if this email exists, a link has been sent"})
		return
	}

	magicURL := fmt.Sprintf("%s?token=%s", os.Getenv("APP_URL"), magicToken)

	if err := h.Mailer.SendMagicLink(email, magicURL); err != nil {
		log.Printf("SendMagicLink error: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "if this email exists, a link has been sent"})
}

// GET /auth/login?token=XXXX
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	email, err := tokens.ValidateMagicToken(tokenStr, os.Getenv("JWT_SECRET"))
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	user, err := h.DB.FindUserByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	deviceToken, tokenHash, err := tokens.GenerateDeviceToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	deviceName := r.UserAgent()
	if deviceName == "" {
		deviceName = "unknown"
	}

	if _, err := h.DB.CreateDevice(r.Context(), user.ID, tokenHash, &deviceName); err != nil {
		log.Printf("CreateDevice error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"device_token": deviceToken,
		"user_id":      user.ID,
	})
}

// POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	tokenHash := authmw.GetTokenHash(r.Context())
	if err := h.DB.RevokeDevice(r.Context(), tokenHash); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// GET /auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := authmw.GetUserID(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{
		"user_id": userID,
	})
}

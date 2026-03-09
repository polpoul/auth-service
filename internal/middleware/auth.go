package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/you/auth-service/internal/db"
	"github.com/you/auth-service/internal/tokens"
)

type contextKey string

const ContextUserID contextKey = "user_id"
const ContextTokenHash contextKey = "token_hash"

func RequireAuth(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			rawToken := strings.TrimPrefix(authHeader, "Bearer ")
			tokenHash := tokens.HashToken(rawToken)

			device, err := database.FindDeviceByTokenHash(r.Context(), tokenHash)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Update last_seen asynchronously (non-blocking)
			go func() {
				_ = database.UpdateLastSeen(context.Background(), tokenHash)
			}()

			ctx := context.WithValue(r.Context(), ContextUserID, device.UserID)
			ctx = context.WithValue(ctx, ContextTokenHash, tokenHash)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts authenticated user ID from context.
func GetUserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(ContextUserID).(uuid.UUID)
	return id
}

// GetTokenHash extracts the current device token hash from context.
func GetTokenHash(ctx context.Context) string {
	h, _ := ctx.Value(ContextTokenHash).(string)
	return h
}

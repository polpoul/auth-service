package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/you/auth-service/internal/openapps"
)

// Ces tests ne couvrent que les chemins qui répondent AVANT tout accès à la base ou à l'envoi
// d'email : DB et Mailer sont nil, une requête qui irait trop loin ferait planter le test.

func newHandler(t *testing.T, limiter *openapps.EmailLimiter) *Handler {
	t.Helper()
	policy, err := openapps.Parse("portail", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	return &Handler{OpenApps: policy, OpenAppLimiter: limiter}
}

func requestLogin(h *Handler, body, secret string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/auth/request-login", strings.NewReader(body))
	if secret != "" {
		req.Header.Set(appSecretHeader, secret)
	}
	rec := httptest.NewRecorder()
	h.RequestLogin(rec, req)
	return rec
}

func TestOpenAppWithoutSecretIsForbidden(t *testing.T) {
	h := newHandler(t, openapps.NewEmailLimiter(3, time.Hour))
	body := `{"email":"eleve@example.com","app":"portail"}`
	if rec := requestLogin(h, body, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("sans secret : attendu 403, reçu %d", rec.Code)
	}
	if rec := requestLogin(h, body, "mauvais"); rec.Code != http.StatusForbidden {
		t.Fatalf("mauvais secret : attendu 403, reçu %d", rec.Code)
	}
}

func TestOpenAppOverLimitGetsGenericAnswerAndNoLink(t *testing.T) {
	// Limite à 0 : toute demande est au-dessus de la limite. DB nil prouve qu'on ne va pas plus loin.
	h := newHandler(t, openapps.NewEmailLimiter(0, time.Hour))
	rec := requestLogin(h, `{"email":"eleve@example.com","app":"portail"}`, "s3cret")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), genericLoginMessage) {
		t.Fatalf("attendu la réponse générique 200, reçu %d %s", rec.Code, rec.Body.String())
	}
}

func TestInvalidEmailIsRejectedBeforeAnythingElse(t *testing.T) {
	h := newHandler(t, nil)
	if rec := requestLogin(h, `{"email":"pas-un-email","app":"portail"}`, "s3cret"); rec.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, reçu %d", rec.Code)
	}
}

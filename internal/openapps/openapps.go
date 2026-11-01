// Package openapps gère les applications « ouvertes » : celles pour lesquelles l'allowlist est
// sautée parce que l'application contrôle elle-même qui peut se connecter (ex. le portail
// stagiaire, qui n'envoie un lien qu'aux adresses inscrites).
//
// Sauter l'allowlist sans autre garde-fou permettrait à n'importe qui de faire envoyer un email
// à n'importe quelle adresse. Deux gardes-fous sont donc imposés :
//   - un secret partagé : seule l'application légitime peut demander un lien pour une application ouverte ;
//   - une limite par adresse : une même boîte ne reçoit pas de rafale de liens.
package openapps

import (
	"crypto/subtle"
	"errors"
	"strings"
	"sync"
	"time"
)

// Policy : applications ouvertes et secret exigé pour les utiliser.
type Policy struct {
	apps   map[string]bool
	secret string
}

// Parse lit la liste séparée par des virgules (ex. "portail"). Une liste non vide sans secret est
// une erreur : une application ouverte sans secret serait ouverte à tous.
func Parse(apps, secret string) (*Policy, error) {
	p := &Policy{apps: map[string]bool{}, secret: secret}
	for _, app := range strings.Split(apps, ",") {
		if app = strings.TrimSpace(app); app != "" {
			p.apps[app] = true
		}
	}
	if len(p.apps) > 0 && secret == "" {
		return nil, errors.New("OPEN_APPS défini sans OPEN_APP_SECRET : refusé")
	}
	return p, nil
}

// IsOpen : l'application est-elle ouverte ? Le nom vide ne l'est jamais.
func (p *Policy) IsOpen(app string) bool {
	return p != nil && app != "" && p.apps[app]
}

// SecretOK compare en temps constant le secret reçu à celui attendu.
func (p *Policy) SecretOK(given string) bool {
	if p == nil || p.secret == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(given), []byte(p.secret)) == 1
}

// EmailLimiter limite le nombre de demandes par adresse sur une fenêtre glissante. En mémoire :
// un redémarrage remet les compteurs à zéro, ce qui suffit contre une rafale.
type EmailLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	now    func() time.Time
	hits   map[string][]time.Time
}

func NewEmailLimiter(max int, window time.Duration) *EmailLimiter {
	return &EmailLimiter{max: max, window: window, now: time.Now, hits: map[string][]time.Time{}}
}

// Allow enregistre la demande et dit si elle est autorisée. Une demande refusée n'est pas comptée :
// un refus répété ne prolonge pas indéfiniment le blocage.
func (l *EmailLimiter) Allow(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)

	recent := l.hits[email][:0:0]
	for _, t := range l.hits[email] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= l.max {
		l.hits[email] = recent
		return false
	}
	l.hits[email] = append(recent, now)
	l.prune(cutoff)
	return true
}

// prune retire les adresses sans demande récente, pour que la table ne grossisse pas indéfiniment.
func (l *EmailLimiter) prune(cutoff time.Time) {
	if len(l.hits) < 1000 {
		return
	}
	for email, times := range l.hits {
		if len(times) == 0 || !times[len(times)-1].After(cutoff) {
			delete(l.hits, email)
		}
	}
}

package allowlist

import (
	"encoding/json"
	"fmt"
	"os"
)

// Allowlist contient le chemin du fichier et permet une lecture dynamique
type Allowlist struct {
	path string
}

// Load initialise l'allowlist avec le chemin du fichier.
// Si le chemin est vide, retourne nil (pas de restriction).
func Load(path string) (*Allowlist, error) {
	if path == "" {
		return nil, nil
	}
	// Vérifier que le fichier est lisible au démarrage (optionnel mais utile pour détecter les erreurs tôt)
	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("checking allowlist path: %w", err)
	}
	return &Allowlist{path: path}, nil
}

// read lit et parse le fichier JSON à chaque appel
func (al *Allowlist) read() (map[string][]string, error) {
	data, err := os.ReadFile(al.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]string{}, nil
		}
		return nil, fmt.Errorf("reading allowlist: %w", err)
	}
	var entries map[string][]string
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing allowlist: %w", err)
	}
	return entries, nil
}

// IsAllowed vérifie si l'email est autorisé pour le service donné.
// Relit le fichier à chaque appel pour prendre en compte les modifications à chaud.
// Si l'allowlist est nil (pas de fichier configuré), tout le monde est autorisé.
// Si app est vide, vérifie juste que l'email existe dans la liste.
func (al *Allowlist) IsAllowed(email, app string) bool {
	if al == nil {
		return true
	}
	entries, err := al.read()
	if err != nil {
		// En cas d'erreur de lecture, on refuse par sécurité
		return false
	}
	services, ok := entries[email]
	if !ok {
		return false
	}
	if app == "" {
		return true
	}
	for _, s := range services {
		if s == app {
			return true
		}
	}
	return false
}

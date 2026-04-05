package allowlist

import (
	"encoding/json"
	"fmt"
	"os"
)

// Allowlist maps email -> liste de services autorisés
type Allowlist map[string][]string

// Load charge l'allowlist depuis un fichier JSON.
// Si le chemin est vide ou le fichier absent, retourne nil (pas de restriction).
func Load(path string) (Allowlist, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading allowlist: %w", err)
	}
	var al Allowlist
	if err := json.Unmarshal(data, &al); err != nil {
		return nil, fmt.Errorf("parsing allowlist: %w", err)
	}
	return al, nil
}

// IsAllowed vérifie si l'email est autorisé pour le service donné.
// Si l'allowlist est nil (pas de fichier configuré), tout le monde est autorisé.
// Si app est vide, on vérifie juste que l'email existe dans la liste.
func (al Allowlist) IsAllowed(email, app string) bool {
	if al == nil {
		return true
	}
	services, ok := al[email]
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

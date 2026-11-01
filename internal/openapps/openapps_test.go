package openapps

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	if _, err := Parse("portail", ""); err == nil {
		t.Fatal("une application ouverte sans secret doit être refusée")
	}
	p, err := Parse(" portail , autre ,", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsOpen("portail") || !p.IsOpen("autre") {
		t.Fatal("les applications listées doivent être ouvertes")
	}
	if p.IsOpen("axio") || p.IsOpen("") {
		t.Fatal("axio et le nom vide ne doivent jamais être ouverts")
	}
	empty, err := Parse("", "")
	if err != nil || empty.IsOpen("portail") {
		t.Fatal("sans OPEN_APPS, aucune application n'est ouverte et ce n'est pas une erreur")
	}
}

func TestSecretOK(t *testing.T) {
	p, _ := Parse("portail", "s3cret")
	if !p.SecretOK("s3cret") {
		t.Fatal("le bon secret doit passer")
	}
	for _, bad := range []string{"", "S3cret", "s3cret ", "autre"} {
		if p.SecretOK(bad) {
			t.Fatalf("%q ne doit pas passer", bad)
		}
	}
	var none *Policy
	if none.SecretOK("") || none.IsOpen("portail") {
		t.Fatal("une politique absente n'ouvre rien")
	}
}

func TestEmailLimiter(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	l := NewEmailLimiter(3, time.Hour)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("a@x.fr") {
			t.Fatalf("la demande %d doit passer", i+1)
		}
	}
	if l.Allow("a@x.fr") {
		t.Fatal("la 4e demande dans l'heure doit être refusée")
	}
	if !l.Allow("b@x.fr") {
		t.Fatal("une autre adresse n'est pas concernée")
	}
	now = now.Add(61 * time.Minute)
	if !l.Allow("a@x.fr") {
		t.Fatal("après la fenêtre, la demande doit repasser")
	}
}

func TestEmailLimiterRefusalsDoNotExtendBlock(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	l := NewEmailLimiter(1, time.Hour)
	l.now = func() time.Time { return now }
	l.Allow("a@x.fr")
	for i := 0; i < 5; i++ {
		now = now.Add(10 * time.Minute)
		l.Allow("a@x.fr") // refusées
	}
	now = now.Add(11 * time.Minute) // 61 min après la première demande acceptée
	if !l.Allow("a@x.fr") {
		t.Fatal("les refus ne doivent pas prolonger le blocage")
	}
}

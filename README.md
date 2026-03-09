# auth-service

Service d'authentification minimaliste par magic link.  
Stack : **Go** + **Supabase** (PostgreSQL + SMTP) + **Render** (hébergement gratuit)

---

## Structure

```
auth-service/
├── cmd/server/main.go          # point d'entrée
├── internal/
│   ├── db/db.go                # queries PostgreSQL
│   ├── handlers/handlers.go    # routes HTTP
│   ├── middleware/auth.go      # middleware Bearer token
│   ├── mailer/mailer.go        # envoi magic link SMTP
│   └── tokens/tokens.go        # JWT + device token
├── migrations/001_init.sql     # à coller dans Supabase SQL Editor
├── Dockerfile
├── render.yaml
└── .env.example
```

---

## 1. Supabase

1. Créer un projet sur [supabase.com](https://supabase.com)
2. Dans **SQL Editor**, exécuter `migrations/001_init.sql`
3. Récupérer la **DATABASE_URL** : Settings → Database → Connection string (mode `URI`)
4. Activer SMTP : Settings → Auth → SMTP Settings → noter host, port, user, pass

---

## 2. Développement local

```bash
cp .env.example .env
# remplir toutes les valeurs

go mod tidy
go run ./cmd/server
```

Test rapide :
```bash
# Demander un magic link
curl -X POST http://localhost:8080/auth/request-login \
  -H "Content-Type: application/json" \
  -d '{"email":"toi@example.com"}'

# Valider le token reçu par email
curl "http://localhost:8080/auth/login?token=XXXX"
# → { "device_token": "...", "user_id": "..." }

# Appel authentifié
curl http://localhost:8080/auth/me \
  -H "Authorization: Bearer DEVICE_TOKEN"

# Déconnexion
curl -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer DEVICE_TOKEN"
```

---

## 3. Déploiement sur Render (gratuit)

### Étape 1 — Pusher sur GitHub

```bash
git init && git add . && git commit -m "init"
git remote add origin https://github.com/toi/auth-service.git
git push -u origin main
```

### Étape 2 — Créer le service

1. [render.com](https://render.com) → **New > Web Service**
2. Connecter le repo GitHub
3. Render détecte le `render.yaml` automatiquement

### Étape 3 — Renseigner les secrets dans le dashboard Render

| Clé | Valeur |
|-----|--------|
| `DATABASE_URL` | `postgresql://postgres:...@db.xxx.supabase.co:5432/postgres` |
| `JWT_SECRET` | `openssl rand -hex 32` |
| `APP_URL` | `https://auth-service.onrender.com` |
| `SMTP_HOST` | depuis Supabase SMTP settings |
| `SMTP_PORT` | `587` |
| `SMTP_USER` | depuis Supabase SMTP settings |
| `SMTP_PASS` | depuis Supabase SMTP settings |
| `SMTP_FROM` | ton adresse email d'envoi |

### Étape 4 — Deploy

Render build l'image Docker et expose le service sur `https://auth-service.onrender.com`.

> ⚠️ Plan gratuit : le service se met en veille après 15 min d'inactivité.  
> Il se réveille automatiquement à la prochaine requête (~1s de délai).  
> Pour un auth perso, c'est totalement acceptable.

---

## 4. Intégration dans une autre app

Pour chaque requête authentifiée :
```
Authorization: Bearer DEVICE_TOKEN
```

Vérification depuis ton app :
```
GET https://auth-service.onrender.com/auth/me
Authorization: Bearer DEVICE_TOKEN
→ { "user_id": "uuid" }
```

---

## API

| Méthode | Route | Auth | Description |
|---------|-------|------|-------------|
| GET | `/health` | — | Health check Render |
| POST | `/auth/request-login` | — | Envoie magic link |
| GET | `/auth/login?token=` | — | Valide token → retourne device_token |
| POST | `/auth/logout` | Bearer | Révoque le device |
| GET | `/auth/me` | Bearer | Retourne user_id |

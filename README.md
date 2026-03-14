# auth-service

Service d'authentification minimaliste par magic link.  
Stack : **Go** + **MySQL** + **Mailjet** (SMTP) + **VPS Ionos** via Docker + Traefik

---

## Structure

```
auth-service/
├── cmd/server/main.go              # point d'entrée
├── internal/
│   ├── db/db.go                    # queries MySQL
│   ├── handlers/handlers.go        # routes HTTP
│   ├── middleware/auth.go          # middleware Bearer token
│   ├── mailer/mailer.go            # envoi magic link SMTP
│   └── tokens/tokens.go            # JWT + device token
├── migrations/001_init.sql         # migration MySQL
├── .github/workflows/docker.yml   # CI/CD GitHub Actions
├── Dockerfile
└── .env.example
```

---

## 1. Prérequis

- Go 1.22+
- Docker + Docker Compose sur le VPS
- Compte [Mailjet](https://mailjet.com) pour l'envoi des emails
- Réseau Docker `web` créé sur le VPS : `docker network create web`

---

## 2. Développement local

```bash
cp .env.example .env
# remplir toutes les valeurs (voir .env.example)

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

## 3. Déploiement sur le VPS

### Étape 1 — Pusher sur GitHub

```bash
git add . && git commit -m "update"
git push
```

GitHub Actions build et publie automatiquement l'image sur `ghcr.io/polpoul/auth-service:latest`.  
Watchtower met à jour le container sur le VPS toutes les 5 minutes.

### Étape 2 — Configurer les secrets sur le VPS

Dans `/home/pascal/traefik/.env` :

```
MYSQL_ROOT_PASSWORD=...
MYSQL_PASSWORD=...
JWT_SECRET=...         # générer avec : openssl rand -hex 32
SMTP_HOST=in-v3.mailjet.com
SMTP_PORT=587
SMTP_USER=<clé API Mailjet>
SMTP_PASS=<clé secrète Mailjet>
SMTP_FROM=auth@vivalink.top
```

### Étape 3 — Lancer les services

```bash
cd ~/traefik
docker-compose up -d
```

### Étape 4 — Vérifier

```bash
# Test local (avant DNS configuré)
curl -k https://localhost/health -H "Host: auth.vivalink.top"
# → ok

# Test public (après DNS configuré)
curl https://auth.vivalink.top/health
# → ok
```

### Étape 5 — DNS sur Ionos

Ajouter un enregistrement :
- **Type** : `A`
- **Nom** : `auth`
- **Valeur** : `87.106.43.140`

---

## 4. Migration MySQL

Le fichier `migrations/001_init.sql` est exécuté automatiquement par le container MySQL au premier démarrage via `docker-entrypoint-initdb.d`.

Il crée la base `vivalink_db` avec les tables `users` et `devices`.

---

## 5. Intégration dans une autre app

Pour chaque requête authentifiée :

```
Authorization: Bearer DEVICE_TOKEN
```

Vérification depuis un service backend :

```
GET https://auth.vivalink.top/auth/me
Authorization: Bearer DEVICE_TOKEN
→ 200 : { "user_id": "uuid" }
→ 401 : token invalide ou révoqué
```

---

## API

| Méthode | Route | Auth | Description |
|---------|-------|------|-------------|
| GET | `/health` | — | Health check |
| POST | `/auth/request-login` | — | Envoie magic link |
| GET | `/auth/login?token=` | — | Valide token → retourne device_token |
| POST | `/auth/logout` | Bearer | Révoque le device |
| GET | `/auth/me` | Bearer | Retourne user_id |

---

## Variables d'environnement

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | DSN MySQL — `user:pass@tcp(host:port)/db?parseTime=true` |
| `JWT_SECRET` | Secret pour signer les magic link tokens |
| `APP_URL` | URL publique du service (ex: `https://auth.vivalink.top`) |
| `PORT` | Port d'écoute (défaut : `8080`) |
| `SMTP_HOST` | Serveur SMTP |
| `SMTP_PORT` | Port SMTP (587) |
| `SMTP_USER` | Utilisateur SMTP |
| `SMTP_PASS` | Mot de passe SMTP |
| `SMTP_FROM` | Adresse d'expédition |

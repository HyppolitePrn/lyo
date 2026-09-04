# Lyo — StreamPulse

Plateforme de streaming audio en temps réel.

**English version:** [`README.md`](README.md) · **Documentation complète :** [`docs/`](docs/README.md)

Un diffuseur émet de l'audio en direct depuis une application mobile ; les auditeurs se connectent sans
aucune configuration. Le backend diffuse les trames audio vers N auditeurs simultanés selon un patron
hub / fan-out. L'accès aux fonctionnalités est gouverné par les rôles et pilotable à chaud par des
interrupteurs de fonctionnalités.

---

## Pile technique

| Couche | Technologie | Version |
|---|---|---|
| API backend | Go + routeur chi | 1.26 |
| Base de données | PostgreSQL (pgx/v5) | 17 |
| Mobile | Flutter + provider + just_audio | 3.41 |
| Conteneurs | Docker + Docker Compose | — |
| Observabilité | slog (JSON), OpenTelemetry (traces, métriques, logs via OTLP), Grafana + Loki ✅ *(le client mobile ne propage pas encore `traceparent` : la trace démarre donc au backend — voir [ADR 009](docs/adr/009-observability-otlp.md))* | — |

L'API REST est **contract-first** : la source de vérité est `backend/api/openapi.yaml`. Les types serveur
sont générés par `oapi-codegen` — ne jamais modifier `internal/api/api.gen.go` à la main.

---

## Démarrage local

**Prérequis :** Go 1.26, Flutter 3.41, Docker + Docker Compose, `golangci-lint` v2.

```bash
# 1. Copier la configuration
cp .env.example .env   # renseigner JWT_SECRET (32 caractères minimum)

# 2. Démarrer PostgreSQL et la stack d'observabilité
docker compose -f docker/docker-compose.yml up -d

# 3. Démarrer le backend (migrations appliquées automatiquement au boot)
cd backend && go run ./cmd/server

# 4. Lancer l'application mobile
#    (par défaut : émulateur Android → hôte sur 10.0.2.2:8080)
cd mobile && flutter run

# Pour un appareil physique ou un hôte personnalisé :
cd mobile && flutter run --dart-define=API_BASE_URL=http://192.168.x.x:8080
```

**Services locaux**

| Service | URL | Identifiants |
|---|---|---|
| API backend | `http://localhost:8080` | — |
| pgAdmin | `http://localhost:5050` | `some@one.com` / `someone` |
| Grafana | `http://localhost:3000` | `admin` / `admin` |
| Prometheus | `http://localhost:9090` | — |
| Mailpit (capture des e-mails) | `http://localhost:8025` | — |
| MinIO (stockage S3 local) | `http://localhost:9001` | voir `.env` |

pgAdmin tourne en mode bureau (`PGADMIN_CONFIG_SERVER_MODE: "False"`) : c'est un conteneur de confort de
développement, absent de `docker-compose.prod.yml`. Ajoutez le serveur `postgres` (hôte `postgres`, port
`5432`) avec les identifiants du `.env`.

---

## Architecture

### Backend

```
cmd/server/main.go        ← racine de composition : pool DB, services, routeur, arrêt gracieux
internal/api/             ← handlers HTTP (StrictServerInterface généré par oapi-codegen)
internal/streaming/       ← moteur de fan-out (Hub) + handlers WebSocket ingest/listen
internal/auth/            ← signature et vérification JWT, hiérarchie de rôles
internal/features/        ← interrupteurs de fonctionnalités en base, initialisés au démarrage
pkg/middleware/           ← Authenticate (permissif) + RequireRole
backend/migrations/       ← fichiers SQL numérotés, appliqués via golang-migrate
```

Le middleware `Authenticate` place les claims JWT dans le contexte mais **ne rejette jamais** : chaque
handler contrôle lui-même le rôle requis via `middleware.ClaimsFromContext`. Ce choix contre-intuitif
permet à une même route de servir un visiteur anonyme et un utilisateur authentifié — voir
[ADR 007](docs/adr/007-permissive-auth-middleware.md).

Les endpoints WebSocket utilisent un repli d'authentification par paramètre `?token=`, car un handshake
WebSocket ne peut pas transporter d'en-tête personnalisé.

Flux de données en direct :

```
Diffuseur   →  WS /streams/{id}/ingest  →  Hub.Broadcast()
                                                 ↓ fan-out (shards de 100)
Auditeur(s) ←  WS /streams/{id}/listen  ←  Hub.Subscribe()
```

Si le buffer d'un auditeur est plein, la trame est **abandonnée pour cet auditeur** — jamais pour le
diffuseur ni pour les autres. C'est un back-pressure assumé, prouvé par le test
`TestHub_BroadcastDropsChunksForSlowListener`.

### Mobile

```
features/<nom>/
  providers/<nom>_notifier.dart   ← ChangeNotifier, logique métier
  services/<nom>_service.dart     ← appels ApiClient bruts
  screens/ + widgets/             ← interface, lit l'état via context.watch / context.read
```

Les notifiers sont enregistrés une seule fois dans `main.dart` sous un `MultiProvider`. Ils ne s'appellent
jamais entre eux : celui qui a besoin du jeton d'authentification le reçoit **en paramètre de méthode**,
lu au point d'appel depuis `AuthNotifier`. Voir [ADR 004](docs/adr/004-provider-over-riverpod.md) pour
les raisons du passage de Riverpod à `provider`.

`ApiClient` dérive son URL de base de `--dart-define=API_BASE_URL` à la compilation. Le lecteur alimente
`just_audio` en trames AAC binaires reçues par WebSocket via un `StreamAudioSource` personnalisé. Le
diffuseur capture le micro avec le paquet `record` (AAC-LC, 44,1 kHz, mono, 128 kbit/s).

---

## Rôles

| Rôle | Accès |
|---|---|
| `anonymous` | Parcourir et écouter les directs publics (sans jeton) |
| `user` | Écouter, favoris, playlists |
| `broadcaster` | Démarrer et arrêter des directs, publier des pistes |
| `admin` | Tout ce qui précède + gestion des comptes et des interrupteurs |

La hiérarchie est **ordinale** : on écrit toujours `claims.Role.AtLeast(auth.RoleBroadcaster)`, jamais une
égalité — un administrateur hérite ainsi mécaniquement de tous les droits inférieurs.

---

## Interrupteurs de fonctionnalités

Initialisés dans `internal/features/seed.go`.

| Interrupteur | Défaut | Implémenté |
|---|---|---|
| `live_streaming` | **actif** | ✅ |
| `track_uploads` | **actif** | ✅ |
| `playlists` | **actif** | ✅ |
| `favorites` | **actif** | ✅ |
| `chat_websocket` | inactif | ❌ interrupteur seul |
| `recommendations` | inactif | ❌ interrupteur seul |
| `offline_mode` | inactif | ❌ interrupteur seul |
| `transcoding` | inactif | ❌ interrupteur seul |

> Le client mobile lit actuellement ces valeurs depuis une constante de compilation
> (`core/features/feature_flags_provider.dart`) ; le chargement depuis l'API à l'exécution est inscrit à
> la feuille de route. D'ici là, basculer un interrupteur côté serveur modifie le comportement de l'API
> mais pas l'interface mobile.

---

## Qualité — obligatoire avant toute pull request

```bash
# Backend
cd backend && golangci-lint run ./... && go build ./... && go test -race ./...

# Mobile
cd mobile && flutter analyze && flutter test
```

La CI applique en plus un **seuil de couverture de 80 %** (hors code généré) : une PR qui fait passer la
couverture sous ce seuil échoue. Voir [`docs/plan-de-tests.md`](docs/plan-de-tests.md).

---

## Déploiement

### TLS et reverse proxy

En production, **Caddy est le seul conteneur à publier des ports**. Il termine TLS sur le 443, redirige le
80, et relaie vers le backend et Grafana sur le réseau Docker privé — aucun des deux ne publie plus de
port. Les certificats sont obtenus et renouvelés automatiquement via ACME : il n'y a pas de tâche de
renouvellement à maintenir. Le raisonnement, et ce que ce choix ne couvre **pas**, sont dans
l'[ADR 012](docs/adr/012-tls-reverse-proxy.md).

| Variable | Rôle |
|---|---|
| `LYO_DOMAIN` | Nom d'hôte public. Les certificats sont émis pour lui **et** pour `grafana.<domaine>` |
| `ACME_EMAIL` | Adresse de contact pour les avis d'expiration |
| `TRUSTED_PROXY` | À `true` derrière le proxy, pour que l'API lise l'IP client dans `X-Forwarded-For`. `APP_ENV=production` refuse de démarrer sans |
| `CORS_ALLOWED_ORIGINS` | Origines navigateur autorisées, séparées par des virgules |

| URL | Sert |
|---|---|
| `https://<LYO_DOMAIN>` | L'API, WebSockets `/streams/{id}/ingest` et `/listen` comprises (`wss://`) |
| `https://grafana.<LYO_DOMAIN>` | Grafana |
| `https://<LYO_DOMAIN>/internal/*` | Rien — 404 au niveau du proxy ; le webhook d'alerte n'est joignable que depuis le réseau interne |

Les deux enregistrements DNS doivent pointer vers l'hôte **avant** le premier démarrage, sinon le défi ACME
échoue. Laissé à `LYO_DOMAIN=localhost`, Caddy signe avec son autorité interne, ce qui permet de faire
tourner la pile de production sur un poste local.

### Backend — automatique à la fusion sur `main`

Toute modification de `backend/**` poussée sur `main` déclenche la CI complète (lint → tests → seuil de
couverture → build), puis :

1. construction d'une image Docker publiée sur GHCR sous `latest` **et** `sha-<commit>` ;
2. connexion SSH au VPS, `docker compose pull backend && up -d` — la pile entière et pas seulement le backend, puisque celui-ci ne publie plus de port et n'est joignable qu'une fois Caddy démarré (Compose ne recrée que les services modifiés).

Le double étiquetage par SHA rend le **retour arrière** immédiat : redéployer l'image du commit précédent.

### Application mobile — sur tag de version

```bash
git tag v1.2.0
git push origin v1.2.0
```

Construit un APK et un AAB signés et publie une Release GitHub. La version de `pubspec.yaml` est dérivée
du tag (`1.2.0+<numéro de run>`).

### Secrets GitHub requis

`VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `VPS_DEPLOY_PATH`, `VPS_URL`, `KEYSTORE_BASE64`,
`KEYSTORE_PASSWORD`, `KEY_PASSWORD`, `KEY_ALIAS`.

> ℹ️ `VPS_URL` doit être une URL `https://` : la CI de release échoue sinon. Le trafic en clair n'est
> autorisé que dans les builds *debug* et *profile* — un APK signé pointant vers du HTTP serait refusé par
> Android dès l'API 28. Voir [`docs/architecture/securite.md`](docs/architecture/securite.md).

---

## Contribuer

**Branches :** `type/description-courte` — en miroir du type de commit (`feat/broadcaster-screen`,
`fix/jwt-expiry`, `ci/lint-step`).

**Commits :** `type(scope): message` — types : `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`.

Détails complets : [`CONTRIBUTING.md`](CONTRIBUTING.md).

---

## Documentation

Index complet : **[`docs/README.md`](docs/README.md)**.

| Document | Contenu |
|---|---|
| [Cahier des charges](docs/cahier-des-charges.fr.md) | Contexte, faisabilité et ROI, périmètre, spécifications, risques, feuille de route, KPI, conformité, bilan réflexif |
| [Architecture](docs/architecture/README.md) | Diagrammes UML et BPMN, modèle de données, sécurité, déploiement — chaque schéma doublé d'une alternative textuelle |
| [Plan de tests et cahier de recette](docs/plan-de-tests.md) | Stratégie, couverture mesurée, scénarios R-01 à R-15 |
| [User stories](docs/user-stories.md) | 20 stories avec critères d'acceptation |
| [Veille technologique](docs/veille-technologique.fr.md) | Plan de veille, analyse concurrentielle, comparatif des transports |
| [Guide d'utilisation](docs/guide-utilisateur.fr.md) | Auditeur, diffuseur, administrateur |
| [Plan de formation](docs/plan-formation.fr.md) | Par public, y compris adaptations au handicap |
| [Décisions d'architecture](docs/adr/) | 10 ADR, dont un explicitement remplacé |

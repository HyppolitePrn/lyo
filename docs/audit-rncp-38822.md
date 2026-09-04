# Audit RNCP 38822 — Projet Lyo / StreamPulse

**Version 2 — 2026-09-03** (v1 : 2026-09-02)
**Branche auditée :** `main` — dernier commit `e4141b6`
**Blocs couverts :** Bloc 3 (sujet principal — StreamPulse), Bloc 2 et Bloc 1 (rattrapages sur le même projet)

> ⚠️ **Aucun correctif n'a été appliqué par cet audit.** État des lieux + liste d'actions.
>
> - **PARTIE A — Ce qui manque et nécessite du code / des fichiers** (§2 à §5)
> - **PARTIE B — Ce qui ne nécessite pas d'implémentation : à argumenter à l'oral** (§6 à §8)
>
> Règle stricte RNCP : **un seul critère « non acquis » invalide le bloc entier.**

---

## 0. Ce qui a changé depuis la v1 — bilan

Cinq PR ont été mergées depuis l'audit initial (#34 → #39). **Sur les 8 blocages majeurs identifiés en v1, 5 sont entièrement résolus et 1 partiellement.** C'est une progression massive, en particulier sur le cœur du Bloc 3.

### 🟢 Résolu — Observabilité (le blocage n°1 de la v1)

L'écart le plus grave est comblé, et bien au-delà de ce qui était demandé.

| v1 | v2 |
|---|---|
| Aucune dépendance `go.opentelemetry.io` | **16 modules OTel** : traces, métriques, logs, instrumentation runtime |
| `Obs.OTLPEndpoint` configuré mais jamais lu | `observability.Setup()` appelé **en première ligne** de `main()`, avant tout le reste |
| Pas de middleware de trace | `middleware.Trace()` monté dans la chaîne chi |
| Logs jamais envoyés à Loki | Bridge **`otelslog`** → pipeline logs OTLP → Loki |
| Traces exportées vers `debug` (jetées) | **Tempo** ajouté (dev + prod), datasource Grafana |
| Zéro dashboard | **2 dashboards** provisionnés : `lyo-streaming.json`, `lyo-performance.json` |
| Aucune alerte | **Alerting complet** : `rules.yml`, `contact-points.yml`, `notification-policies.yml` |
| Aucune métrique métier | **Les 6 métriques métier recommandées**, à l'identique : `lyo.listeners.active`, `lyo.streams.live`, `lyo.chunks.dropped`, `lyo.stream.bytes`, `lyo.listener.disconnect`, `lyo.broadcast.duration` |

**Bonus non demandé, et excellent pour le jury :** le package `internal/incident` avec le webhook `POST /internal/alerts` qui reçoit les alertes Grafana, les persiste et notifie par mail. Cela ferme la boucle **mesure → alerte → incident → traçabilité** et couvre très fortement les critères **Ce3.3.x** et **Ce3.5.2**. C'est le genre d'initiative qui distingue un projet.

### 🟢 Résolu — Couverture de tests (blocage n°3)

| Package | v1 | v2 |
|---|---|---|
| `internal/streaming` | **0,0 %** | **96,3 %** ✅ |
| `internal/auth` | **0,0 %** | **86,4 %** ✅ |
| `pkg/middleware` | **0,0 %** | **96,6 %** ✅ |
| `internal/features` | 0,0 % | **100 %** ✅ |
| `pkg/config` | 0,0 % | 90,5 % ✅ |
| `pkg/mailer` | 0,0 % | 100 % ✅ |
| `internal/storage` | 0,0 % | 93,1 % ✅ |
| `internal/user/usecase` | 0,0 % | 100 % ✅ |
| `internal/playlist` | 5,5 % | 100 % ✅ |
| `internal/api` | 15,0 % | 79,4 % 🟡 |
| `internal/user` | 24,5 % | 96,2 % ✅ |
| `internal/track` | 28,8 % | 98,6 % ✅ |
| `internal/passwordreset` | 50,0 % | 92,4 % ✅ |
| `internal/incident` | — | 80,0 % ✅ |
| `internal/observability` | — | **55,0 %** 🟠 |
| `cmd/server` | 0,0 % | **12,2 %** 🟠 |

Et surtout : **le seuil de 80 % est désormais appliqué en CI** (avec exclusion propre et justifiée du code généré `api.gen.go`). `internal/streaming/ws_test.go` teste le parcours WebSocket de bout en bout — c'était la recommandation n°1 de la v1.

### 🟢 Résolu — Accessibilité (blocage n°6)

`grep Semantics|semanticLabel mobile/lib` : **0 → 107 occurrences**. Orientation portrait verrouillée. Ce point comptait pour les trois blocs à la fois.

### 🟢 Résolu — Documentation (blocage n°8)

Quasiment tout ce que réclamaient les critères Ce3.6.x et le Bloc 1 a été écrit :

| Livrable | État |
|---|---|
| `docs/cahier-des-charges.fr.md` (630 lignes) + `.en.md` | ✅ Les 13 sections attendues, **bilingue** |
| `docs/veille-technologique.fr.md` | ✅ Plan de veille + analyse concurrentielle + **apport concret** (les 3 volets de C1.2) |
| `docs/user-stories.md` | ✅ 5 épopées + traçabilité |
| `docs/plan-de-tests.md` | ✅ Typologie, matrice, **cahier de recette** (Ce3.2.2) |
| `docs/plan-formation.fr.md` | ✅ Publics, dispositifs, modalités adaptées (Ce3.6.5) |
| `docs/guide-utilisateur.fr.md` + `.en.md` | ✅ Manuel utilisateur bilingue (Ce3.6.1) |
| `docs/architecture/` | ✅ 6 documents : MCD, diagrammes de séquence, **BPMN**, sécurité, déploiement |
| `docs/supervision.md` | ✅ Runbook d'exploitation |
| ADR | ✅ **3 → 11**, avec un index |
| `README.fr.md`, `CONTRIBUTING.md`, `SECURITY.md` | ✅ |

### 🟢 Résolu — Incohérence ADR / Riverpod

`ADR 002` est passé en **Superseded by ADR 004**, l'ADR 004 explique le revirement, et le README + le CLAUDE.md annoncent désormais correctement `provider`. Le piège d'oral est désamorcé.

### 🟢 Résolu — Contrôle du volume

`setVolume` implémenté, avec deux fichiers de test dédiés (`volume_control_test.dart`, `volume_controller_test.dart`).

### 🟡 Partiellement résolu — Rôle Admin (blocage n°4)

**Fait :** `/admin/supervision`, `/admin/incidents`, `/admin/incidents/{id}` côté backend + un module mobile complet `features/admin/` (écran de supervision, notifier, service, modèle, test).

**Toujours cassé :** `ListFeatureFlags` et `ToggleFeatureFlag` dans `internal/api/handlers.go` **renvoient encore `errNotImplemented` → HTTP 501**. Ces deux endpoints sont pourtant publiés dans le contrat OpenAPI. Voir §2.1.

### 🔴 Non traité — les 3 points restants de la v1

- ~~**TLS / reverse proxy**~~ : ✅ traité depuis — Caddy en terminaison TLS, seul conteneur publiant des ports (§3.1)
- **Scans de sécurité en CI + Dependabot** : toujours absents (§3.2)
- **Feature flags mobiles codés en dur** : toujours une `static const Map` (§2.2)

---

## 1. ⚠️ Point bloquant immédiat : tout ce travail n'est pas commité

`git status` sur `main` montre **20 fichiers non suivis ou modifiés** :

```
 M CLAUDE.md, README.md, docs/adr/002-state-management-riverpod.md
?? CONTRIBUTING.md, SECURITY.md, README.fr.md
?? docs/README.md, docs/architecture/, docs/audit-rncp-38822.md
?? docs/adr/004…011, docs/adr/README.md
?? docs/cahier-des-charges.{fr,en}.md, docs/guide-utilisateur.{fr,en}.md
?? docs/plan-de-tests.md, docs/plan-formation.fr.md
?? docs/user-stories.md, docs/veille-technologique.fr.md
```

**L'intégralité de la documentation — dont le cahier des charges, livrable principal évalué du Bloc 1 — n'existe que sur ce disque.** Elle n'est ni poussée, ni sauvegardée, ni visible par un jury qui consulte le dépôt.

C'est aussi une occasion manquée sur le plan de l'évaluation : le critère **Ce3.1.1** porte sur *« la mise en place du dépôt de code et du système de gestion des versions … le suivi de l'historique des modifications »*. Un dossier de documentation qui apparaît d'un seul bloc, ou pas du tout, ne raconte rien.

**À faire tout de suite**, sur une branche puis en PR pour rester cohérent avec la méthodologie affichée :

```bash
git switch -c docs/rncp-deliverables
git add CLAUDE.md README.md README.fr.md CONTRIBUTING.md SECURITY.md docs/
git commit -m "docs: add RNCP deliverables (cahier des charges, architecture, ADR, guides)"
```

> Note : `.envrc` est aussi non suivi. Vérifier qu'il ne contient aucun secret avant tout `git add` large — `.gitignore` ne le couvre pas explicitement.

---

# PARTIE A — À implémenter

---

## 2. Fonctionnalités et cohérence produit

### 2.1 🔴 Les deux endpoints de feature flags renvoient toujours 501

```go
// backend/internal/api/handlers.go
func (h *Handlers) ListFeatureFlags(...)  { return nil, errNotImplemented }
func (h *Handlers) ToggleFeatureFlag(...) { return nil, errNotImplemented }
```

Le service `features` est complet et **testé à 100 %**, le repository existe, le contrat OpenAPI publie les deux routes. **Il ne manque que le câblage** — c'est probablement l'action au meilleur rapport effort/valeur de toute cette liste.

Le risque est double : un jury qui teste l'API sur une route documentée obtient un `501`, et le README présente le système de flags comme *« contrôlé au runtime »* alors que rien ne peut le modifier.

### 2.2 🔴 Les feature flags mobiles restent codés en dur

`mobile/lib/core/features/feature_flags_provider.dart` est toujours une `static const Map<String, bool>`, sans aucun appel réseau.

À noter : **le README documente honnêtement cette limite** (« loading them from `GET /features` at runtime is on the roadmap ») — c'est la bonne posture, et cela réduit fortement le risque à l'oral. Mais la boucle reste ouverte : même une fois §2.1 corrigé, basculer un flag côté admin **ne changerait rien dans l'application**.

**À faire :** exposer un `GET /features` public (flags activés uniquement), le charger au démarrage et après login, transformer `FeatureFlags` en `ChangeNotifier` avec repli sur les valeurs actuelles en cas d'échec réseau.

### 2.3 🟠 Gestion des interruptions audio — toujours absente

`grep interruptionEventStream|becomingNoisyEventStream mobile/lib` → **0 résultat**.

Le sujet demande explicitement *« la gestion des interruptions (appels entrants, notifications) »*. `player_notifier.dart` configure bien `AudioSessionConfiguration.music()` mais ne s'abonne à aucun événement :
- `session.interruptionEventStream` → pause sur appel entrant, reprise à la fin
- `session.becomingNoisyEventStream` → pause au débranchement du casque

C'est une vingtaine de lignes, et c'est un item nommément listé dans les enjeux techniques du sujet.

### 2.4 🟠 Pas de gestion des utilisateurs par l'admin

Le sujet définit : *« Admin : **Gestion des utilisateurs** et accès aux metrics globales »*. La seconde moitié est désormais couverte (supervision), **la première ne l'est pas** : aucun `GET /admin/users`, aucun changement de rôle, aucune suspension ni suppression de compte.

### 2.5 🟠 Droit à l'effacement (RGPD) — pas d'endpoint

`/users/me` supporte `GET` et `PATCH`, mais **pas `DELETE`**. Le droit à l'effacement est le droit RGPD le plus souvent testé par un jury, et le cahier des charges comporte désormais une section 12 « Conformité RGPD » — l'écart entre ce qui est écrit et ce qui est implémenté devient donc visible.

**À faire :** `DELETE /users/me` (anonymisation ou suppression en cascade), plus une purge des `password_reset_tokens` expirés (politique de rétention).

### 2.6 🟠 Les 5 points bonus restent à 0

`chat_websocket`, `recommendations`, `offline_mode`, `transcoding` sont toujours de simples lignes dans `seed.go`, toutes à `false`. Kubernetes absent.

**Priorisation inchangée :** le **chat WebSocket** reste le meilleur choix — il réutilise le pattern `Hub` déjà écrit et **déjà testé à 96 %**, donc le coût réel est faible. Second choix : **`recommendations`**, qui couvrirait en plus le critère C2.5 du Bloc 2 (voir §7).

### 2.7 🟠 Pas de logique de file d'attente (Queue)

Inchangé depuis la v1 : le CRUD playlist est complet, mais il n'y a ni réordonnancement (`PATCH .../tracks/reorder`), ni lecture continue piste suivante/précédente. Le sujet demande *« CRUD complet des playlists avec logique de file d'attente (Queue) »*.

---

## 3. Sécurité et industrialisation

### 3.1 ✅ TLS en production — traité

> **Mise à jour.** Ce point était le principal écart technique de l'audit ; il est corrigé. `docker/Caddyfile` termine TLS avec des certificats ACME renouvelés automatiquement, et Caddy est le **seul** conteneur à publier des ports : le backend et Grafana ne publient plus rien et ne sont joignables qu'à travers lui. Le trafic en clair n'est autorisé que dans les manifestes Android *debug* et *profile*, et la CI de release refuse une `VPS_URL` qui ne serait pas en `https://`. Décision et limites : [ADR 012](adr/012-tls-reverse-proxy.md). Le constat d'origine est conservé ci-dessous.

**Constat initial.** `docker-compose.prod.yml` publiait le backend en direct (`${SERVER_PORT:-8080}:8080`), sans reverse proxy. Aucune trace de Caddy, Nginx ou Traefik dans `docker/`.

Le sujet exige nommément *« Mise en place de protocoles sécurisés (TLS) »*. Conséquences concrètes du constat initial :
- les JWT et les identifiants transitaient en clair ;
- l'APK de release construit avec `API_BASE_URL=${{ secrets.VPS_URL }}` aurait été bloqué par Android (cleartext interdit depuis API 28) si l'URL était en `http://` ;
- les WebSockets étaient en `ws://` et non `wss://`.

**Reste à vérifier au déploiement :** que `S3_PUBLIC_ENDPOINT` pointe bien vers une URL `https://` — le stockage objet n'est pas derrière le proxy.

### 3.2 🔴 Aucune analyse de sécurité automatisée

`.github/` ne contient toujours **que** `workflows/`. Absents :
- `govulncheck ./...` (vulnérabilités des dépendances Go)
- `gosec` / linters de sécurité golangci-lint
- scan Trivy de l'image publiée sur GHCR
- **`.github/dependabot.yml`** (Go modules, pub, Actions, Docker)
- `CODEOWNERS`, `pull_request_template.md`, `ISSUE_TEMPLATE/`

Critère **Ce3.1.4**. À noter l'ironie : `SECURITY.md` a été écrit, mais aucun outil ne vérifie quoi que ce soit. Un jury verra le décalage.

### 3.3 🟠 Pas de rate limiting

Inchangé : aucun `httprate` ni équivalent. `/auth/login`, `/auth/register`, `/auth/forgot-password` restent exposés au bruteforce et au spam mail.

### 3.4 🟠 Pas de smoke test ni de rollback après déploiement

Le job `deploy` fait toujours `pull && up -d` puis s'arrête. Aucun `curl /health`, aucun retour arrière. Si une migration échoue au boot, le déploiement est signalé « vert ».

Les images sont pourtant déjà taguées `sha-${{ github.sha }}` : **le rollback est à quelques lignes près gratuit**. Critères Ce3.4.3 / Ce3.4.4 / Ce3.5.4.

### 3.5 🟠 Pas d'environnement de staging

Un seul environnement. Critère **Ce3.4.2** (*« livraison sur l'ensemble des plateformes d'intégration »*).

### 3.6 🟠 CORS en dur dans `main.go`

`AllowedOrigins: ["http://localhost:*", "http://127.0.0.1:*"]` avec `AllowCredentials: true` reste codé en dur — contradiction directe avec le « zéro hardcoding » revendiqué. → `CORS_ALLOWED_ORIGINS`.

### 3.7 🟠 Signature des commits

`git log --pretty='%G?'` renvoie **`E` sur les 12 derniers commits** — y compris les plus récents, alors que les tout premiers de la v1 affichaient `G`. La signature semble avoir été perdue en cours de route.

Les livrables communs exigent nommément *« commits signés »*. → Vérifier le badge « Verified » sur GitHub, reconfigurer la clé GPG/SSH et `git config --global commit.gpgsign true`.

---

## 4. Tests — ce qui reste

La situation est devenue bonne. Il reste quatre écarts, tous secondaires.

### 4.1 🟠 Deux packages sous le seuil

- `internal/observability` — **55,0 %** : c'est le package qui porte le cœur du Bloc 3. Un jury pourrait relever qu'il est le moins testé du projet. Cibler `otel.go` (chemins d'erreur du `Setup`, `shutdown`) et `prometheus.go`.
- `cmd/server` — **12,2 %** : acceptable pour un `main`, mais `runMigrations` mériterait un test.

Le total pondéré passe le seuil CI de 80 %, donc ce n'est pas bloquant — c'est un point de finition.

### 4.2 🟠 Toujours aucun test d'intégration

`grep -rl "go:build integration" backend` → **aucun fichier**.

L'écart de documentation est corrigé : le `CLAUDE.md` déclarait la convention `backend/internal/*_integration_test.go` alors qu'aucun fichier ne l'appliquait. La ligne a été retirée et remplacée par un constat explicite — les repositories sont testés avec une doublure `PgxPool`, et l'absence de suite d'intégration est renvoyée vers le [plan de tests](plan-de-tests.md). Une convention documentée mais jamais appliquée est plus risquée que pas de convention du tout.

Le manque de fond reste entier : rien ne prouve aujourd'hui que le **vrai SQL** et les **migrations** fonctionnent. Les écrire (testcontainers-go ou service Postgres GitHub Actions) demeure la bonne réponse.

### 4.3 🟠 Aucun benchmark ni test de charge

`grep "func Benchmark" backend` → **0 résultat**. `goleak` absent.

Ce n'est pas seulement un manque de test : **c'est ce qui empêche de répondre à la question d'oral la plus probable du sujet** (« combien coûte en CPU le streaming de 100 flux ? », §6.1). Avec `internal/streaming` à 96 % de couverture, ajouter `BenchmarkHubBroadcast` avec 10/100/1000 abonnés représente quelques dizaines de lignes pour un argument décisif.

Ajouter aussi `go.uber.org/goleak` dans le package streaming : cela **prouve** l'absence de fuite de goroutines à la déconnexion, ce que le sujet demande explicitement (*« éviter les fuites de mémoire »*).

### 4.4 🟠 Mobile : pas de seuil de couverture ni de test d'accessibilité

- `mobile.yml` génère `lcov.info` et l'archive, mais **ne bloque sur aucun seuil** (contrairement au backend, désormais).
- Aucun test d'accessibilité automatisé (`meetsGuideline(textContrastGuideline)`, `androidTapTargetGuideline`). Avec 107 `Semantics` posés, c'est dommage : le test transformerait un effort déjà fait en **preuve automatisée**, ce qui est bien plus fort devant un jury.
- Toujours aucun test de `PlayerNotifier` / `BroadcasterNotifier` sur coupure réseau — or le sujet demande la robustesse du state management *« même en cas de changement d'état réseau »*.

---

## 5. Observabilité — le dernier maillon manquant

L'instrumentation est excellente. Il reste **un** trou, et il est précisément sur la phrase du sujet.

### 5.1 🟠 La trace ne descend pas jusqu'à la base de données

`grep otelpgx backend/go.mod` → **absent**. Le pool `pgxpool` n'est pas instrumenté.

Le sujet écrit : *« Un étudiant doit pouvoir suivre le cheminement d'une requête **de l'application mobile jusqu'à la base de données** »*. Aujourd'hui la trace couvre HTTP → handler → service, puis **s'arrête avant le SQL**.

**À faire :** `github.com/exaring/otelpgx` — un `poolCfg.ConnConfig.Tracer = otelpgx.NewTracer()` dans `main.go`, et les spans SQL apparaissent dans Tempo. C'est une ligne de code pour fermer littéralement la phrase du référentiel.

### 5.2 🟠 La trace ne commence pas au mobile

Aucun header `traceparent` n'est émis par `ApiClient`. La trace démarre donc au backend, et la partie « de l'application mobile » de la phrase ci-dessus n'est pas satisfaite non plus.

**À faire :** générer et propager un `traceparent` W3C depuis `core/api/api_client.dart`. Avec §5.1, la démonstration devient complète et spectaculaire en soutenance : **un clic dans Tempo, du geste de l'utilisateur jusqu'à la requête SQL.**

### 5.3 🟢 Rien d'autre à signaler

Traces (Tempo), métriques (Prometheus), logs (Loki via `otelslog`), instrumentation runtime, 2 dashboards, alerting provisionné, webhook d'incidents. C'est au niveau attendu, voire au-dessus.

---

## 5bis. Accessibilité et i18n

- **Semantics : ✅ 107 occurrences.** L'écart de la v1 est comblé.
- **Reste à vérifier** (non automatisé aujourd'hui) : contrastes ≥ 4,5:1 sur `lyo_tokens.dart`, cibles tactiles ≥ 48 dp, comportement à `textScaler` 200 %, annonces d'état du player aux lecteurs d'écran.
- **i18n : toujours absente.** `flutter_localizations` n'est pas dans `pubspec.yaml`, l'UI reste monolingue en dur. Le livrable « FR/EN » est désormais satisfait côté **documentation** (cahier des charges, guide utilisateur, README) — l'interface reste un axe d'amélioration à mentionner à l'oral plutôt qu'un blocage.

---

# PARTIE B — À argumenter à l'oral

---

## 6. Bloc 3 — sujets purement oraux

### 6.1 Scalabilité et justification des coûts 🔴 **question quasi certaine, toujours sans chiffres**

Le sujet la pose textuellement : *« Combien nous coûte en CPU le streaming de 100 flux simultanés ? »* et *« prouver que le serveur peut encaisser N auditeurs simultanés avec une consommation mémoire minimale »*.

**C'est le seul point de la Partie B qui mérite un vrai (petit) investissement technique**, et il est maintenant très bon marché : le hub est testé à 96 %, l'infrastructure de mesure existe.

1. `go test -bench=. -benchmem ./internal/streaming/` sur `Hub.Broadcast` à 10 / 100 / 1000 abonnés
2. Charge k6/vegeta sur `/listen` + `docker stats` pour le RSS
3. Profil `pprof` heap et CPU
4. **Bonus gratuit :** vous avez désormais `lyo.listeners.active` et `lyo.chunks.dropped` dans Grafana — faire la démonstration de charge **en direct sur le dashboard** est nettement plus impressionnant qu'un tableau de chiffres.

**Raisonnement à tenir** (à recaler avec vos mesures) :
> « Un auditeur = une goroutine (~4 Ko de pile) + un canal bufferisé de `STREAM_BUFFER_SIZE` (65 536 octets par défaut). Le coût dominant est le buffer, pas la goroutine : ~64 Ko × N, soit ~64 Mo pour 1 000 auditeurs. Le CPU est dominé par les copies mémoire et les écritures socket, pas par le calcul : on ne décode ni ne transcode, on relaie des trames AAC opaques. À 128 kbit/s, 100 flux × 10 auditeurs = 1 000 × 16 Ko/s ≈ 16 Mo/s, soit ~128 Mbit/s : **c'est la bande passante, et non le CPU, qui sature en premier**. Le levier de coût est donc l'egress réseau — ce qui plaide pour un CDN/edge, pas pour plus de CPU. »

**Limite à assumer :** hub en mémoire, processus unique ⇒ **pas de scalabilité horizontale sans état partagé** (bus Redis Pub/Sub ou NATS, routage sticky par `streamID`). Assumer cette limite est bien mieux vu que de prétendre le contraire.

### 6.2 Posture Tech Lead — « pourquoi ce choix »

Les **11 ADR** couvrent désormais l'essentiel de l'argumentaire. Les points à savoir défendre au-delà de ce qui est écrit :

| Choix | Alternative écartée | Argument |
|---|---|---|
| Go | Node.js / Python | Goroutines ~2 Ko de pile vs threads OS, pas de GIL, binaire statique → image alpine minimale, GC à faible latence |
| Fan-out bufferisé + drop | Mutex sur slice partagé | Un auditeur lent ne doit **jamais** bloquer le diffuseur : on dégrade **cet** auditeur plutôt que **tout le monde**. Back-pressure assumé — et **mesuré** via `lyo.chunks.dropped` |
| Sharding par 100 auditeurs | Une goroutine par auditeur par chunk | Compromis parallélisme / coût d'ordonnancement |
| WebSocket | HLS / WebRTC / RTMP | HLS = 6–30 s de latence ; WebRTC = complexité SFU/TURN démesurée pour de l'audio unidirectionnel ; WS = un port, traverse les proxys, latence sub-seconde |
| OTel + collector | Client Prometheus direct | Un seul standard pour traces/métriques/logs, backends interchangeables sans toucher au code |
| Webhook d'incidents | Alertes Grafana seules | Persiste l'incident, le rend interrogeable par l'API et traçable — boucle SRE complète |

### 6.3 Kubernetes — pourquoi ne pas l'avoir fait

« Pour un service à processus unique avec état en mémoire (le Hub) sur un VPS, K8s ajoute un plan de contrôle à opérer sans bénéfice : la scalabilité horizontale resterait bloquée par l'état en mémoire tant que le bus de messages n'existe pas. Le prérequis à K8s ici n'est pas l'orchestration, c'est de rendre le Hub distribuable. » — Réponse de Tech Lead, à conserver telle quelle.

### 6.4 Gestion des ressources et fuites mémoire

À montrer dans le code : `Hub.Done()`, `defer cancel()`, arrêt gracieux 15 s, `defer pool.Close()`, et le `shutdown` OTel avec son propre timeout. **Ce qui manque encore comme preuve :** `goleak` (§4.3).

### 6.5 Preuve de fluidité 60 FPS

Toujours aucune capture DevTools. → Timeline en mode profile sur appareil physique pendant une écoute live, à déposer dans `docs/`. Argumentaire : décodage audio hors thread UI dans `just_audio`, `notifyListeners()` ne redessine que les sous-arbres abonnés, aucun octet audio manipulé sur le thread UI.

### 6.6 Ce3.3.x — surveillance continue

**Ce critère est devenu facile à défendre.** Raconter la boucle complète, qui existe maintenant réellement : dashboards → règle d'alerte Grafana → webhook `/internal/alerts` → incident persisté + mail → consultation via `/admin/incidents` et l'écran mobile de supervision → correctif → redéploiement. Les feature flags comme rollback fonctionnel instantané complètent le tableau.
**Point faible à ne pas éluder :** la surveillance des **vulnérabilités** (Dependabot, govulncheck) n'existe pas encore — c'est la moitié manquante de Ce3.3.4.

### 6.7 RGPD

Le cahier des charges comporte désormais une section 12 dédiée. À l'oral : minimisation, base légale, sécurité, droits des personnes — avec **deux écarts à assumer honnêtement** : pas de `DELETE /users/me` (droit à l'effacement, §2.5), pas de purge des tokens expirés. Les corriger avant la soutenance serait préférable : deux petits endpoints qui transforment une faiblesse en argument.

---

## 7. Bloc 2 (rattrapage) — transposition à défendre

Le Bloc 2 impose NextJS/Firebase/Swift-Kotlin ; le projet est en Go + Flutter. Le risque est de **cadrage**, pas technique. **À clarifier impérativement avec l'école.**

| Critère | État après v2 | Reste à faire |
|---|---|---|
| **C2.1** Architecture | 🟢 Ce2.1.3 (UML/BPMN) désormais **couvert** par `docs/architecture/` ; Ce2.1.5 (accessibilité) couvert ; Ce2.1.4 « conforme aux exigences de sécurité » couvert depuis la terminaison TLS (§3.1) | Ce2.1.6 impact environnemental : couvert par la section 8 du cahier des charges, à savoir défendre |
| **C2.2** Intégrité du code | 🟡 Lint, tests race, **seuil de couverture 80 %** ✅ (= Ce2.2.2 « indicateurs de référence ») | 🔴 Ce2.2.1 « sécurité **shift-left**, dès le début du cycle » ← §3.2, non couvert |
| **C2.3** Front-end | 🟢 Design system, responsive, **107 Semantics** (Ce2.3.5) | Test d'accessibilité automatisé (§4.4) pour en faire une preuve |
| **C2.4** Back-end | 🟡 Go/chi/PostgreSQL/S3/JWT, OWASP partiellement | Savoir mapper l'OWASP Top 10 : A01 = rôles ✅ ; A02 = hachage ✅ et TLS ✅ ; A03 = requêtes paramétrées pgx ✅ ; A07 = JWT ✅ mais **rate limiting ❌** |
| **C2.5** Données massives | 🔴 **Toujours non couvert** | **Le critère le plus à risque du Bloc 2.** Implémenter la fonctionnalité bonus `recommendations` (historique d'écoute → co-occurrence) créerait un vrai pipeline collecte → stockage → analyse, et cocherait à la fois C2.5 **et** les points bonus /5. Repli argumentaire : le streaming *est* de la donnée temps réel à haut débit, avec collecte (ingest), traitement (fan-out) et **métriques agrégées désormais réelles** — l'argument est plus crédible qu'en v1 grâce à l'observabilité |

---

## 8. Bloc 1 (rattrapage) — l'essentiel est écrit

**C'est le bloc qui a le plus progressé.** Le livrable principal évalué RNCP — le cahier des charges — existe désormais en 630 lignes, bilingue, avec les 13 sections attendues.

| Élément | Critère | État |
|---|---|---|
| Contexte et objectifs | C1.1 | ✅ §1 |
| Étude de faisabilité technique / organisationnelle / **financière** + ROI | C1.1 | ✅ §2 — **vérifier que le volet financier et le ROI sont chiffrés**, c'est le plus souvent creux |
| Veille technologique et concurrentielle | C1.2 | ✅ `veille-technologique.fr.md` couvre les **3 volets** : plan, outils/méthodes, apport opérationnel |
| Architecture macro | C1.3 | ✅ `docs/architecture/` (6 documents) |
| Spécifications fonctionnelles | C1.3 | ✅ `user-stories.md`, 5 épopées + traçabilité |
| Accessibilité | C1.3 | ✅ §7 du cahier des charges + 107 Semantics dans le code |
| Numérique responsable | C1.3, C1.5 | ✅ §8 |
| Analyse des risques | C1.1, C1.4 | ✅ §9 — vérifier que le **risque droits d'auteur sur l'audio diffusé** y figure, il est spécifique à ce projet |
| Feuille de route, planning, budget | C1.4 | ✅ §10 |
| **Outil de pilotage (Kanban / GitHub Project)** | C1.4 | 🔴 **Toujours aucune preuve.** Les captures d'écran de l'outil sont attendues à l'oral |
| KPI | C1.4, C1.6 | ✅ §11 — et désormais **mesurables réellement** via Grafana, ce qui est un atout considérable pour C1.6 |
| Conformité RGPD / ANSSI / ITIL | C1.6 | ✅ §12, avec les réserves du §6.7 |
| Bilan / analyse réflexive | C1.6, C1.7 | ✅ §13 |

### Les deux points Bloc 1 restants

1. **🔴 Outil de pilotage** — c'est le seul livrable Bloc 1 encore totalement absent. Créer un GitHub Project, y rejouer les épopées de `user-stories.md` en colonnes (Backlog / En cours / Revue / Fait), et le peupler avec l'historique réel des PR #12 à #39. Une demi-journée, et cela couvre un critère explicite de C1.4.
2. **🟢 C1.5 / C1.7 — récits de pilotage.** L'historique Git est devenu une mine. Préparer 3 récits « problème → décision → correction → résultat » :
   - `c6a9995` migration Riverpod → Provider, **puis** `ADR 002` passé en *Superseded* et `ADR 004` écrit pour justifier le revirement — cas d'école parfait de C1.7 : identification, correction, **et mise à jour de la documentation**
   - `090672c` refactor vers l'interface `PgxPool` **pour rendre le code testable**, suivi de l'ajout du seuil de couverture en CI — une décision d'architecture au service de la qualité
   - `feeaa22` observabilité + supervision — d'un manque identifié à une boucle SRE complète

---

## 9. Plan d'action recommandé — v2

### Aujourd'hui (30 minutes)

1. **Commiter et pousser toute la documentation** — §1. Rien d'autre n'a de valeur tant que ce n'est pas fait.

### Priorité haute (1–2 jours)

2. **Implémenter `ListFeatureFlags` / `ToggleFeatureFlag`** — §2.1, quelques dizaines de lignes, le service est prêt et testé à 100 %
3. ~~**TLS via Caddy** en prod + `https://` / `wss://` côté mobile~~ — §3.1, ✅ fait
4. **`otelpgx` + `traceparent` mobile** — §5.1, §5.2 : ferme littéralement la phrase du référentiel « du mobile jusqu'à la base »
5. **Scans de sécurité CI + `dependabot.yml`** — §3.2

### Priorité moyenne (2–3 jours)

6. **Benchmarks `Hub.Broadcast` + `goleak` + test de charge** — §4.3, §6.1 : c'est la réponse à la question d'oral la plus probable
7. **Gestion des interruptions audio** — §2.3
8. **`DELETE /users/me` + purge des tokens** — §2.5
9. **Smoke test `/health` + rollback** après déploiement — §3.4
10. **Rate limiting** sur les routes `/auth/*` — §3.3
11. **GitHub Project** rétro-alimenté — §8

### Finitions

12. Couverture `internal/observability` (55 % → 80 %) — §4.1
13. Tests d'intégration sur un vrai PostgreSQL — §4.2 (la convention fantôme du CLAUDE.md, elle, est retirée)
14. Seuil de couverture mobile + test d'accessibilité automatisé — §4.4
15. Capture DevTools 60 FPS — §6.5
16. Resigner les commits — §3.7

### Si du temps reste — les 5 points bonus

17. **Chat WebSocket** (réutilise le `Hub` déjà testé à 96 %) — §2.6
18. **Recommandations** (couvre aussi le C2.5 du Bloc 2) — §2.6, §7

---

## 10. Les 12 questions d'oral à préparer

| # | Question | Réponse | État |
|---|---|---|---|
| 1 | « Combien coûte en CPU le streaming de 100 flux ? » | §6.1 | 🔴 **chiffres manquants** |
| 2 | « Montrez une trace du mobile jusqu'à la base. » | §5.1, §5.2 | 🟡 trace partielle — s'arrête avant le SQL |
| 3 | « Erreur technique vs dégradation d'expérience sur ce dashboard ? » | Les 6 métriques métier + `lyo-performance.json` | 🟢 **prêt** |
| 4 | « Quelle est votre couverture de tests ? » | 80 % appliqué en CI, streaming 96 % | 🟢 **prêt** |
| 5 | « Que se passe-t-il si un auditeur est trop lent ? » | Back-pressure + drop, **mesuré** par `lyo.chunks.dropped` | 🟢 **prêt** |
| 6 | « Comment scalez-vous à 10 000 auditeurs ? » | Assumer la limite de l'état en mémoire | 🟢 prêt |
| 7 | « Pourquoi Provider et pas Riverpod ? » | ADR 004 | 🟢 **prêt** |
| 8 | « Comment un utilisateur aveugle utilise-t-il l'app ? » | 107 Semantics | 🟡 prêt, mais sans preuve automatisée |
| 9 | « Où sont vos secrets ? » | Env, `.gitignore`, GitHub Secrets | 🟢 prêt |
| 10 | « Comment revenez-vous en arrière après un déploiement cassé ? » | Images taguées par SHA | 🔴 **pas de rollback automatisé** |
| 11 | « Êtes-vous conforme RGPD ? » | Cahier des charges §12 | 🟡 pas de droit à l'effacement |
| 12 | « Qu'est-ce que vous feriez différemment ? » | Analyse réflexive §13 | 🟢 prêt — **ne jamais répondre « rien »** |

---

## Annexe — commandes de vérification

```bash
# Couverture réelle (v2 : tout ≥ 80 % sauf observability et cmd/server)
cd backend && go test -cover ./...

# Le seuil CI, reproduit en local
cd backend && go test -coverprofile=coverage.out ./... \
  && grep -v 'internal/api/api.gen.go' coverage.out > coverage.nogen.out \
  && go tool cover -func=coverage.nogen.out | tail -1

# OTel : 16 modules attendus
cd backend && grep -c "go.opentelemetry.io" go.mod

# Le trou restant : la trace ne descend pas au SQL
cd backend && grep -c otelpgx go.mod      # attendu : 0 aujourd'hui, > 0 après correction

# Handlers encore non implémentés
cd backend && grep -B2 "errNotImplemented" internal/api/handlers.go

# Accessibilité mobile (v1 : 0 → v2 : 107)
grep -rn "Semantics\|semanticLabel" mobile/lib | wc -l

# Interruptions audio : toujours 0
grep -rn "interruptionEventStream\|becomingNoisyEventStream" mobile/lib | wc -l

# Documentation non commitée — à vérifier en premier
git status --short

# Signature des commits (E = non vérifiable)
git log -12 --pretty='%h %G? %s'

# Pipeline CI complet en local
cd backend && golangci-lint run ./... && go build ./... && go test -race ./...
cd mobile  && flutter analyze && flutter test
```

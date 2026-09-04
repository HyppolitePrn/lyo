# Diagrammes de séquence (UML)

## 1. Inscription et authentification

```mermaid
sequenceDiagram
    autonumber
    actor U as Utilisateur
    participant M as App mobile (AuthNotifier)
    participant A as API Go — /auth/*
    participant S as user.Service
    participant DB as PostgreSQL
    participant J as auth.Service (JWT)

    U->>M: Saisit e-mail + mot de passe
    M->>A: POST /auth/register {username, email, password}
    A->>A: ctx, cancel := context.WithTimeout(ctx, 10s)
    A->>S: Register(...)
    S->>S: bcrypt.GenerateFromPassword(password, DefaultCost)
    S->>DB: INSERT INTO users (…, password_hash, role='user')
    DB-->>S: user{id, role}
    S->>J: Issue(userID, role)
    J-->>S: TokenPair{access (15 min), refresh (7 j)}
    S-->>A: user + tokens
    A-->>M: 201 {accessToken, refreshToken, user}
    M->>M: Décode le champ "role" du payload JWT (_jwtRole)
    M-->>U: Redirection vers l'accueil

    Note over M,A: Requêtes suivantes : en-tête Authorization: Bearer ...
    M->>A: GET /users/me (Bearer)
    A->>A: middleware.Authenticate — permissif :<br/>place les claims en contexte, ne rejette jamais
    A->>A: handler : ClaimsFromContext + claims.Role.AtLeast(RoleUser)
    A-->>M: 200 profil  /  401 si aucun claim

    Note over M,A: À expiration de l'access token
    M->>A: POST /auth/refresh {refreshToken}
    A->>J: Verify(refresh) puis Issue(...)
    A-->>M: 200 nouvelle paire de jetons
```

**Alternative textuelle.** L'utilisateur saisit ses identifiants dans l'application. Le client appelle
`POST /auth/register`. Le handler dérive d'abord un contexte à échéance de 10 secondes, puis délègue au
service utilisateur qui hache le mot de passe avec bcrypt et insère la ligne avec le rôle `user` par
défaut. Le service JWT émet une paire de jetons : un jeton d'accès valable 15 minutes et un jeton de
rafraîchissement valable 7 jours. Le client décode localement le champ `role` du payload pour adapter
son interface, sans jamais faire confiance à cette valeur pour l'autorisation — celle-ci est toujours
revérifiée côté serveur. Les requêtes suivantes portent l'en-tête `Authorization: Bearer`. Le middleware
d'authentification est volontairement permissif : il enrichit le contexte quand un jeton valide est
présent, mais ne rejette jamais lui-même ; c'est chaque handler qui exige le rôle nécessaire. À
expiration, le client échange son jeton de rafraîchissement contre une nouvelle paire.

## 2. Diffusion live de bout en bout — le cœur du système

```mermaid
sequenceDiagram
    autonumber
    actor B as Diffuseur
    participant MB as App diffuseur (record + WS)
    participant API as API REST
    participant SVC as streaming.Service
    participant HUB as Hub (goroutines + channels)
    participant DB as PostgreSQL
    participant ML as App auditeur (just_audio)
    actor L as Auditeur

    B->>MB: « Démarrer le live »
    MB->>API: POST /streams {title, description} (Bearer)
    API->>API: Vérifie flag live_streaming + rôle ≥ broadcaster
    API->>SVC: StartStream(broadcasterID, …)
    SVC->>DB: HasLive(broadcasterID)
    SVC->>DB: INSERT INTO streams (status='live')
    Note right of DB: index unique partiel :<br/>refus si un live existe déjà
    SVC->>HUB: NewHub(bufferSize, logger)
    SVC-->>API: stream{id}
    API-->>MB: 201 {id}

    MB->>API: WS GET /streams/{id}/ingest?token=…
    API->>API: Auth par query param (le WS ne porte pas d'en-tête)
    loop Toutes les ~N ms
        MB->>HUB: Trame binaire AAC-LC ADTS (44,1 kHz, mono, 128 kbit/s)
        HUB->>HUB: Broadcast(ctx, chunk)
    end

    L->>ML: Ouvre le live
    ML->>API: WS GET /streams/{id}/listen?token=…
    API->>HUB: Subscribe(listenerID) → chan Chunk (bufferisé)
    HUB-->>ML: Trames audio
    ML->>ML: StreamController → _WsAudioSource → just_audio

    Note over HUB: Fan-out par shards de 100 auditeurs.<br/>Si le canal d'un auditeur est plein :<br/>select ... default → chunk DROPPÉ pour lui seul,<br/>le diffuseur n'est jamais bloqué.

    B->>MB: « Arrêter le live »
    MB->>API: DELETE /streams/{id}
    API->>SVC: EndStream(id, broadcasterID)
    SVC->>DB: UPDATE streams SET status='ended', ended_at=now()
    SVC->>HUB: Close() → ctx.Done() fermé
    HUB-->>MB: Boucle d'ingest terminée proprement
    HUB-->>ML: Canaux fermés, auditeurs déconnectés
```

**Alternative textuelle.** Le diffuseur demande la création d'un live par l'API REST. Le handler vérifie
successivement que le feature flag `live_streaming` est actif et que l'appelant a au moins le rôle
`broadcaster`, puis le service crée l'enregistrement en base — la base refusant un second live simultané
via un index unique partiel — et instancie un Hub en mémoire. Le diffuseur ouvre alors une connexion
WebSocket d'ingestion, authentifiée par paramètre de requête puisqu'un handshake WebSocket ne transporte
pas d'en-tête personnalisé, et y pousse des trames AAC-LC brutes capturées par le micro. Chaque trame est
diffusée par le Hub. Un auditeur ouvre de son côté une connexion d'écoute ; le Hub lui alloue un canal
bufferisé et l'ajoute à sa table d'abonnés. La diffusion s'effectue en parallèle par lots de 100
auditeurs, une goroutine par lot ; si le canal d'un auditeur est saturé, la trame est abandonnée pour
**cet auditeur seul**, jamais pour les autres, et le diffuseur n'est jamais ralenti : c'est un
back-pressure assumé. À l'arrêt, le service marque le live comme terminé en base et ferme le contexte du
Hub, ce qui termine proprement la boucle d'ingestion et déconnecte tous les auditeurs.

## 3. Publication d'une piste enregistrée (upload S3 présigné)

```mermaid
sequenceDiagram
    autonumber
    actor B as Diffuseur
    participant M as App mobile
    participant API as API Go
    participant S3 as MinIO / S3
    participant DB as PostgreSQL

    B->>M: Choisit un fichier audio
    M->>API: POST /tracks/upload-url {filename, contentType} (Bearer)
    API->>API: flag track_uploads + rôle ≥ broadcaster
    API->>S3: Génère une URL PUT présignée (courte durée)
    API-->>M: 200 {uploadUrl, publicUrl}
    M->>S3: PUT uploadUrl (octets audio)
    Note over API,S3: Les octets ne transitent JAMAIS par l'API :<br/>pas de saturation mémoire ni de bande passante backend
    S3-->>M: 200
    M->>API: POST /tracks {title, artist, audioUrl: publicUrl, durationSeconds}
    API->>DB: INSERT INTO tracks
    API-->>M: 201 track
```

**Alternative textuelle.** Pour publier une piste, l'application demande d'abord à l'API une URL
d'upload présignée à durée de vie courte. Le fichier est ensuite envoyé **directement** au stockage objet
par le téléphone : les octets audio ne traversent jamais le backend, qui ne porte donc ni la charge
mémoire ni la bande passante de l'upload. Une fois le transfert terminé, le client enregistre les
métadonnées de la piste via un second appel REST.

## 4. Réinitialisation du mot de passe

```mermaid
sequenceDiagram
    autonumber
    actor U as Utilisateur
    participant M as App mobile
    participant API as API Go
    participant PR as passwordreset.Service
    participant DB as PostgreSQL
    participant SMTP as Serveur SMTP

    U->>M: « Mot de passe oublié »
    M->>API: POST /auth/forgot-password {email}
    API->>PR: RequestReset(email)
    PR->>DB: SELECT user WHERE email = …
    alt Compte inexistant
        PR-->>API: (aucune action)
        Note right of API: Réponse identique dans les deux cas :<br/>pas d'énumération de comptes
    else Compte existant
        PR->>PR: Génère un jeton aléatoire, stocke SHA-256(jeton)
        PR->>DB: INSERT password_reset_tokens (expires_at, used_at=NULL)
        PR->>SMTP: E-mail contenant un lien deep-link avec le jeton en clair
    end
    API-->>M: 202 « Si un compte existe, un e-mail a été envoyé »

    U->>M: Ouvre le lien (deep link app_links)
    M->>API: POST /auth/reset-password {token, newPassword}
    API->>PR: Reset(token, newPassword)
    PR->>DB: SELECT WHERE token_hash = SHA-256(token) AND expires_at > now() AND used_at IS NULL
    PR->>DB: UPDATE users SET password_hash = bcrypt(newPassword)
    PR->>DB: UPDATE password_reset_tokens SET used_at = now()
    API-->>M: 200
```

**Alternative textuelle.** La demande de réinitialisation répond toujours de la même façon, que le compte
existe ou non, afin de ne pas permettre l'énumération d'adresses e-mail. Quand le compte existe, un jeton
aléatoire est généré ; seule son empreinte est conservée en base, le jeton en clair n'existant que dans
l'e-mail envoyé. Le lien ouvre l'application par deep link. La validation exige simultanément une
empreinte correspondante, une date d'expiration non dépassée et un jeton non encore consommé ; l'usage
est unique.

---

## English summary

Four UML sequence diagrams: (1) registration/login with bcrypt hashing, a 15-minute access token and a
7-day refresh token, and a deliberately permissive auth middleware whose role checks live in the
handlers; (2) the end-to-end live pipeline, from broadcaster ingest WebSocket through the in-memory Hub
to listener fan-out, including the non-blocking drop policy that degrades one slow listener rather than
the whole broadcast; (3) presigned-S3 track upload, where audio bytes bypass the API entirely; (4)
password reset with hashed, expiring, single-use tokens and an anti-enumeration constant response.

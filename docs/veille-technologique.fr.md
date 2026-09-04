# Veille technologique et analyse concurrentielle

**Version :** 1.0 · **Date :** 2026-09-03

> Ce document comporte **trois volets** : un *plan de veille* (sources, outils, fréquence, méthode de
> tri), une *analyse concurrentielle*, et la démonstration de ce que la veille a **concrètement apporté**
> à la solution. Le troisième est le plus important : il est traité au § 4.

---

## 1. Plan de veille

### 1.1 Objectifs

| Axe de veille | Question à laquelle il répond | Périodicité |
|---|---|---|
| **Sécurité** | Une dépendance que j'utilise est-elle vulnérable ? | Continue (automatisée) |
| **Écosystème Go** | Une évolution du langage ou de la bibliothèque standard rend-elle une dépendance inutile ? | Mensuelle |
| **Écosystème Flutter / Dart** | Une rupture d'API ou une dépréciation menace-t-elle le build mobile ? | À chaque version majeure |
| **Streaming temps réel** | Existe-t-il une meilleure approche que le WebSocket pour ce cas d'usage ? | Trimestrielle |
| **Observabilité** | Les standards (OpenTelemetry) évoluent-ils vers une meilleure intégration ? | Trimestrielle |
| **Concurrence produit** | Le marché a-t-il déplacé les attentes des utilisateurs ? | Semestrielle |
| **Réglementaire** | RGPD, accessibilité (RGAA/WCAG), droits d'auteur sur l'audio diffusé | Semestrielle |

### 1.2 Sources retenues

| Source | Type | Axe | Pourquoi cette source |
|---|---|---|---|
| Notes de version Go (`go.dev/doc/devel/release`) | Officielle | Go | Source primaire, sans intermédiaire |
| Go vulnerability database (`pkg.go.dev/vuln`) | Officielle | Sécurité | Base officielle interrogée par `govulncheck` |
| GitHub Security Advisories | Officielle | Sécurité | Couvre Go, Dart et les images Docker |
| Flutter release notes + `flutter_deprecations` | Officielle | Mobile | Anticipation des ruptures d'API |
| Spécification OpenTelemetry + blog CNCF | Officielle / communautaire | Observabilité | Le standard, pas un fournisseur |
| Documentation OWASP (Top 10, ASVS, cheat sheets) | Référentiel | Sécurité | Grille de lecture réutilisable |
| Guides et recommandations ANSSI | Institutionnelle | Sécurité, conformité | Référentiel attendu en France |
| CNIL — délibérations et guides développeurs | Institutionnelle | RGPD | Interprétation faisant autorité |
| RFC IETF (6455 WebSocket, 7519 JWT, 8216 HLS) | Normative | Streaming, auth | Comprendre les protocoles à la source plutôt que par des tutoriels |
| Blogs d'ingénierie (Twitch, Discord, Cloudflare) | Retour d'expérience | Streaming à grande échelle | Chiffres réels de production |
| Changelogs des dépendances directes | Technique | Toutes | Impact immédiat sur le projet |

### 1.3 Outils et méthode

| Outil | Rôle | Statut |
|---|---|---|
| **Dependabot** | Ouverture automatique de PR de mise à jour (Go modules, pub, Actions, Docker) | 🚧 à activer |
| **`govulncheck`** | Détection des vulnérabilités **réellement atteignables** dans le code, pas seulement présentes dans l'arbre de dépendances | 🚧 à intégrer en CI |
| **Trivy** | Analyse de l'image Docker publiée | 🚧 à intégrer en CI |
| **`go list -m -u all` / `flutter pub outdated`** | Inventaire manuel de l'obsolescence | ✅ à la demande |
| **Watch GitHub** sur les dépôts critiques (chi, pgx, just_audio, coder/websocket) | Notification des releases | ✅ |
| **Flux RSS agrégé** (notes de version, avis de sécurité) | Lecture hebdomadaire regroupée | ✅ |

**Méthode de tri.** Chaque information collectée passe par une grille à trois questions :

1. **Est-ce que cela nous concerne ?** La dépendance ou le composant est-il réellement utilisé, et le
   code vulnérable est-il atteignable depuis nos chemins d'exécution ? (`govulncheck` répond
   automatiquement à cette question, contrairement à un simple audit de `go.sum`.)
2. **Quel est l'impact si nous ne faisons rien ?** Sécurité exploitable · rupture de build à terme ·
   dette technique · simple confort.
3. **Quel est le coût de l'action ?**

Le croisement impact × coût produit trois issues : **traiter immédiatement** (vulnérabilité exploitable,
rupture de build), **inscrire au backlog** avec une issue datée, ou **consigner et ne rien faire** en
notant la décision — une veille dont on décide de ne pas agir reste une veille, à condition que la
décision soit tracée.

**Restitution.** Une entrée par décision dans le journal ci-dessous ; les décisions structurantes
donnent lieu à un ADR dans `docs/adr/`.

---

## 2. Analyse concurrentielle

### 2.1 Panorama

| Solution | Positionnement | Latence typique | Modèle | Ce qu'on en retient |
|---|---|---|---|---|
| **Twitch (audio)** | Live vidéo/audio à très grande échelle | 2–10 s (LL-HLS) | Publicité + abonnements | La latence faible est un **argument produit** différenciant, pas un détail technique |
| **Spotify Live / Greenroom** *(arrêté)* | Audio social adossé à un catalogue | ~ 5 s | Abonnement | Un service audio social sans communauté préexistante ne tient pas : l'échec valide un périmètre resserré |
| **Clubhouse** | Audio social conversationnel | < 1 s (WebRTC) | Levées de fonds | La latence sub-seconde n'est indispensable qu'en **conversation bidirectionnelle** ; notre usage est unidirectionnel |
| **Mixlr** | Diffusion audio pour créateurs et lieux de culte | 5–20 s | Abonnement diffuseur | Cible directe : le diffuseur individuel qui veut diffuser en trois gestes |
| **Icecast / Shoutcast** | Serveur de radio libre, historique | 5–30 s (HTTP progressif) | Auto-hébergement | Robuste et éprouvé, mais aucune application mobile ni gestion de comptes intégrée : c'est un serveur, pas un produit |
| **Radio France, podcasts** | Différé | Sans objet | Public / publicitaire | Confirme que le différé est un marché distinct du direct |

### 2.2 Positionnement de Lyo

Lyo se situe volontairement entre Mixlr et Icecast : la **simplicité d'usage mobile** d'un produit grand
public, avec la **maîtrise technique** d'un serveur auto-hébergé. Sa proposition tient en une phrase :
*diffuser en direct depuis un téléphone, en trois gestes, sans matériel ni configuration serveur, avec
une latence inférieure à deux secondes.*

**Ce que Lyo ne cherche pas à faire**, et pourquoi : pas de vidéo (multiplie le coût de transcodage et de
bande passante par un ordre de grandeur), pas de conversation bidirectionnelle (imposerait WebRTC et une
infrastructure SFU/TURN), pas de catalogue musical sous licence (le risque juridique sur les droits
d'auteur est le premier risque métier identifié — voir la matrice des risques du cahier des charges).

### 2.3 Comparatif d'architectures de transport

Cette comparaison est **l'arbitrage technique fondateur du projet**.

| Critère | **WebSocket** *(retenu)* | HLS / LL-HLS | WebRTC | RTMP |
|---|---|---|---|---|
| Latence | **< 1 s** | 6–30 s (2–5 s en LL-HLS) | < 500 ms | 2–5 s |
| Complexité serveur | Faible — un port, une boucle de lecture | Moyenne — segmentation, playlists, stockage | **Élevée** — signalisation, SFU, STUN/TURN, NAT | Moyenne |
| Traversée des pare-feux et proxys | **Excellente** (HTTP upgrade sur 443) | Excellente | Moyenne (UDP souvent bloqué) | Faible (port 1935 souvent bloqué) |
| Adapté au navigateur | Oui | Oui | Oui | Non (déprécié avec Flash) |
| Mise à l'échelle par CDN | Non nativement | **Oui — c'est sa force** | Non | Non |
| Adéquation au besoin | **Audio unidirectionnel temps réel** | Diffusion massive tolérante à la latence | Conversation bidirectionnelle | Ingestion vers un service tiers |

**Décision.** WebSocket. La latence est le cœur de la proposition de valeur, ce qui élimine HLS.
Le flux est unidirectionnel : la complexité de WebRTC (signalisation, traversée de NAT, SFU) ne serait
pas payée en retour. RTMP est un protocole d'ingestion, mal adapté à la distribution vers des clients
mobiles. WebSocket offre la latence recherchée pour une complexité minimale, et se replie proprement
derrière un proxy HTTPS.

**Limite assumée** — c'est le corollaire honnête de ce choix : le WebSocket ne se distribue pas par CDN.
La mise à l'échelle horizontale exigera un bus de messages (Redis Pub/Sub ou NATS) entre instances, avec
routage par identifiant de flux. Ce point est inscrit à la feuille de route et documenté en ADR.

---

## 3. Journal de veille et décisions

| Date | Constat issu de la veille | Décision | Trace |
|---|---|---|---|
| 2026-04 | HLS ajoute 6 à 30 s de latence, incompatible avec la promesse « direct » | Transport WebSocket retenu | ADR 003 |
| 2026-04 | Les frameworks Go (Gin, Echo) imposent leurs propres types de handler | chi retenu, compatible `net/http` | ADR 001 |
| 2026-04 | La rédaction manuelle des handlers dérive du contrat au fil des versions | Approche OpenAPI-first avec génération de code | ADR 005 |
| 2026-04 | Les ORM Go masquent le SQL et créent des requêtes N+1 difficiles à diagnostiquer | `pgx/v5` en SQL explicite | ADR 008 |
| 2026-06 | `gorilla/websocket` a connu une période sans mainteneur ; `coder/websocket` propose une API alignée sur `context` | `coder/websocket` retenu | ADR 006 |
| 2026-07 | Riverpod impose une réécriture des conventions pour un gain limité sur des états simples | Migration vers `provider` / `ChangeNotifier` | ADR 004 |
| 2026-08 | Une seule stack d'observabilité neutre s'impose : OpenTelemetry | OTLP retenu comme unique protocole d'export | ADR 009 |
| 2026-09 | `govulncheck` évalue l'**atteignabilité** du code vulnérable, là où un audit de `go.sum` produit du bruit | Intégration en CI planifiée | Feuille de route |

---

## 4. Apport concret de la veille au produit

C'est le volet que le référentiel exige explicitement et qui est le plus souvent oublié. Quatre exemples
où la veille a **changé le code**, et pas seulement alimenté une réflexion :

**① La politique d'abandon de trame vient d'un retour d'expérience externe.** Les publications
d'ingénierie sur les systèmes de diffusion à grande échelle décrivent invariablement le même mode de
défaillance : un consommateur lent qui, par effet de contre-pression, finit par bloquer le producteur et
donc **tous** les autres consommateurs. Cette lecture a directement produit la branche `default` du
`select` dans `Hub.Broadcast` : la trame est abandonnée pour l'auditeur saturé, jamais pour la
diffusion. Cette décision est aujourd'hui **couverte par un test dédié**
(`TestHub_BroadcastDropsChunksForSlowListener`) — la veille a produit du code, puis une preuve.

**② La contrainte « un seul live » est descendue au niveau de la base.** La lecture des analyses sur les
conditions de concurrence de type TOCTOU a montré que la vérification `HasLive()` suivie de `Create()`
laissait une fenêtre exploitable par deux requêtes simultanées. Un index unique partiel PostgreSQL
(migration 000006) rend la règle inviolable, même avec plusieurs instances du backend. Le commentaire de
la migration documente explicitement ce raisonnement.

**③ Le seuil de couverture exclut le code généré.** Les retours de la communauté Go sur les métriques de
couverture montrent qu'inclure le code généré produit un indicateur qui mesure le générateur plutôt que
le projet. Le job de CI retire donc `internal/api/api.gen.go` du profil avant d'appliquer le seuil de
80 %, avec un commentaire justificatif dans le workflow.

**④ L'upload passe par une URL présignée.** La documentation OWASP et les guides de conception S3
convergent sur le fait qu'un upload transitant par l'API expose le serveur à une saturation mémoire et
bande passante triviale à provoquer. Le fichier part donc directement du téléphone vers le stockage
objet, l'API ne délivrant qu'une autorisation à portée et durée limitées.

---

## English summary

The technology-watch plan defines seven watch axes with their cadence, a set of primary sources
(official release notes, the Go vulnerability database, OWASP, ANSSI, CNIL, IETF RFCs, large-scale
engineering blogs), a tooling stack (Dependabot, `govulncheck`, Trivy, repository watches), and a
three-question triage grid producing one of three outcomes: act now, backlog with a dated issue, or
record the decision not to act. The competitive analysis positions Lyo between Mixlr and Icecast — mobile
simplicity with self-hosted control — and states explicitly what the product refuses to do and why. The
founding architectural comparison (WebSocket vs HLS vs WebRTC vs RTMP) selects WebSocket for sub-second
unidirectional audio and openly records its cost: no CDN distribution, hence a message bus is required
before horizontal scaling. The final section shows four cases where the watch changed actual code: the
non-blocking drop policy, the database-level single-live constraint, the coverage threshold excluding
generated code, and presigned uploads.

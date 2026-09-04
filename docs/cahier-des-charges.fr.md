# Cahier des charges — Lyo / StreamPulse

**Version :** 1.0 · **Date :** 2026-09-03 · **Auteur :** Hyppolite Pernot
**Version anglaise :** [`cahier-des-charges.en.md`](cahier-des-charges.en.md)

> **Convention de lecture.** ✅ = livré et vérifiable dans le dépôt · 🚧 = planifié, inscrit à la feuille
> de route (§ 9). Ce document ne présente jamais comme acquis ce qui ne l'est pas : la crédibilité de
> l'ensemble de la documentation en dépend.

## Sommaire

1. [Contexte et objectifs](#1-contexte-et-objectifs)
2. [Étude de faisabilité](#2-étude-de-faisabilité)
3. [Veille technologique et concurrentielle](#3-veille-technologique-et-concurrentielle)
4. [Architecture macroscopique](#4-architecture-macroscopique)
5. [Spécifications fonctionnelles](#5-spécifications-fonctionnelles)
6. [Spécifications techniques](#6-spécifications-techniques)
7. [Accessibilité et inclusion](#7-accessibilité-et-inclusion)
8. [Numérique responsable](#8-numérique-responsable)
9. [Analyse des risques](#9-analyse-des-risques)
10. [Feuille de route et pilotage](#10-feuille-de-route-et-pilotage)
11. [Indicateurs de performance (KPI)](#11-indicateurs-de-performance-kpi)
12. [Conformité : RGPD, ANSSI, ITIL](#12-conformité--rgpd-anssi-itil)
13. [Bilan et analyse réflexive](#13-bilan-et-analyse-réflexive)

---

## 1. Contexte et objectifs

### 1.1 Contexte

L'audio en direct est le seul format qui se consomme **en faisant autre chose** : en conduisant, en
marchant, en travaillant. Cette caractéristique explique la vitalité du marché (podcast, radio en ligne,
audio social), mais aussi son point de friction principal : les plateformes existantes sont soit trop
lourdes pour un diffuseur individuel (configuration serveur, encodeur, matériel), soit trop tolérantes à
la latence pour porter une véritable expérience de direct.

Deux constats fondent le projet :

- **La latence est une caractéristique produit, pas un détail technique.** Un délai de 20 secondes rend
  impossible toute interaction avec l'audience : la réaction arrive après que le sujet a changé.
- **La barrière d'entrée du diffuseur est de nature technique, pas éditoriale.** Diffuser depuis un
  téléphone en trois gestes n'a rien d'évident aujourd'hui pour un utilisateur non technicien.

### 1.2 Problématique

> Comment permettre à un diffuseur individuel de démarrer une diffusion audio en direct depuis son
> téléphone, en quelques secondes et sans configuration, tout en garantissant à ses auditeurs une
> latence inférieure à deux secondes et un service qui ne se dégrade pas globalement quand un auditeur
> se dégrade individuellement ?

### 1.3 Objectifs

| Objectif | Nature | Indicateur de succès |
|---|---|---|
| O1 — Diffuser en direct depuis un mobile sans configuration | Produit | Moins de 3 actions entre l'ouverture de l'application et le début de la diffusion |
| O2 — Latence perçue inférieure à 2 s | Technique | Mesure de bout en bout en recette |
| O3 — Isolation des défaillances : un auditeur lent ne dégrade que lui-même | Technique | Prouvé par test automatisé |
| O4 — Consommation mémoire maîtrisée et prévisible par auditeur | Technique | Coût mémoire linéaire et borné, mesurable |
| O5 — Livraison continue et réversible | Industriel | Toute fusion sur `main` déployée automatiquement ; retour arrière en une commande |
| O6 — Observabilité distinguant erreurs techniques et dégradation d'expérience | Exploitation | Tableau de bord séparant les deux familles d'indicateurs 🚧 |
| O7 — Application utilisable en situation de handicap | Inclusion | Parcours principal validé au lecteur d'écran 🚧 |

### 1.4 Public cible

| Public | Besoin | Volume estimé en phase pilote |
|---|---|---|
| **Diffuseur individuel** (créateur, animateur, association, lieu de culte) | Diffuser sans matériel ni compétence serveur | 10 à 50 comptes |
| **Auditeur** | Écouter immédiatement, sans compte obligatoire | 100 à 1 000 sessions |
| **Administrateur** | Exploiter, surveiller, couper une fonctionnalité défaillante | 1 à 2 comptes |

### 1.5 Périmètre

| Dans le périmètre (MVP) | Hors périmètre, et pourquoi |
|---|---|
| Diffusion audio en direct unidirectionnelle | Vidéo — coût de transcodage et de bande passante d'un ordre de grandeur supérieur |
| Pistes enregistrées, playlists, favoris | Conversation bidirectionnelle — imposerait WebRTC et une infrastructure SFU/TURN |
| Comptes, rôles, réinitialisation de mot de passe | Catalogue musical sous licence — risque juridique majeur sur les droits d'auteur |
| Application Android | iOS — Flutter le permet, mais le coût du compte développeur et de la chaîne de signature est hors budget du pilote |
| Observabilité et déploiement continu | Kubernetes — voir la justification en § 2.1 |

---

## 2. Étude de faisabilité

### 2.1 Faisabilité technique

| Verrou technique | Analyse | Conclusion |
|---|---|---|
| Diffuser un flux vers N auditeurs sans dégradation croisée | Un modèle de concurrence à goroutines et canaux bufferisés permet un fan-out sans verrou global ; l'abandon de trame borne la dégradation à l'auditeur concerné | **Levé** — implémenté et prouvé par test |
| Latence inférieure à 2 s | WebSocket sur trames AAC relayées sans transcodage ; aucune segmentation ni mise en tampon intermédiaire | **Levé** |
| Coût mémoire prévisible | Une goroutine (quelques kilo-octets de pile) et un canal borné par auditeur ; le buffer domine et est configurable (`STREAM_BUFFER_SIZE`) | **Levé** — coût linéaire et plafonné |
| Mise à l'échelle horizontale | L'état des Hubs est **en mémoire du processus** : deux instances ne partagent pas leurs auditeurs | **Non levé — limite assumée.** Un bus de messages (Redis Pub/Sub, NATS) avec routage par identifiant de flux est le prérequis. Inscrit à la feuille de route. |
| Application unique pour deux plateformes | Flutter permet un socle commun ; la capture micro et la lecture en arrière-plan s'appuient sur des bibliothèques matures | **Levé** pour Android |

> **Pourquoi pas Kubernetes.** L'orchestration n'est pas le facteur limitant de ce système : la mise à
> l'échelle horizontale est bloquée par l'état en mémoire du Hub, pas par la capacité à démarrer des
> réplicas. Introduire Kubernetes ajouterait un plan de contrôle à exploiter sans lever le verrou réel.
> Le prérequis à Kubernetes ici n'est pas l'orchestration, c'est de rendre le Hub distribuable. Docker
> Compose sur un VPS unique est donc l'outil proportionné au besoin actuel.

### 2.2 Faisabilité organisationnelle

| Ressource | Disponibilité | Conséquence sur la conduite du projet |
|---|---|---|
| Équipe | **1 développeur** | Facteur de risque principal (*bus factor* de 1). Compensé par : documentation systématique des décisions en ADR, automatisation maximale de la chaîne de livraison, conventions strictes de nommage et de commits. |
| Durée | Avril à septembre 2026, à temps partiel | Périmètre MVP resserré, priorisation par valeur |
| Compétences | Go, Flutter, Docker, CI/CD acquises ; observabilité et sécurité réseau en cours d'acquisition | Ces deux domaines concentrent l'essentiel de la dette identifiée |
| Outillage | GitHub (dépôt, Actions, Releases, Projects), VPS mutualisé | Coût marginal nul pour le pilote |

### 2.3 Faisabilité financière et retour sur investissement

**Coûts d'infrastructure (hypothèse pilote : 50 diffuseurs, 500 auditeurs simultanés en pointe)**

| Poste | Hypothèse | Coût mensuel estimé |
|---|---|---|
| VPS (4 vCPU, 8 Go RAM, 200 Mbit/s) | Backend + PostgreSQL + observabilité | 20 – 40 € |
| Stockage objet (pistes enregistrées) | 100 Go | 2 – 5 € |
| **Bande passante sortante** | 500 auditeurs × 128 kbit/s × 3 h/j ≈ **2,6 To/mois** | 0 € si incluse dans le forfait VPS, **20 – 250 €** au-delà selon l'opérateur |
| Nom de domaine + certificat TLS | Let's Encrypt gratuit | ~ 1 € |
| CI/CD, registre d'images | GitHub Actions, dépôt public | 0 € |
| **Total** | | **≈ 25 à 300 € / mois** |

**Le poste de coût dominant est la bande passante sortante, pas le calcul.** Ce résultat oriente
directement l'architecture : optimiser le CPU n'apporterait rien, alors qu'une distribution en périphérie
(CDN) ou une réduction de débit deviendraient les seuls leviers réels en cas de montée en charge. Le
constat justifie aussi le refus de la vidéo, qui multiplierait ce poste par 10 à 50.

**Coûts de développement (valorisation, projet réalisé en formation)**

| Poste | Estimation |
|---|---|
| Conception et développement backend | ≈ 200 h |
| Développement mobile | ≈ 150 h |
| Industrialisation (CI/CD, Docker, observabilité) | ≈ 60 h |
| Tests et documentation | ≈ 90 h |
| **Total** | **≈ 500 h** — soit 20 000 à 25 000 € valorisés au coût moyen d'un développeur |

**Retour sur investissement.** Le pilote n'a pas de modèle de revenus : le retour est ici de nature
**pédagogique et patrimoniale**. Dans une hypothèse de commercialisation, l'analyse serait la suivante :

| Scénario | Hypothèse | Seuil de rentabilité |
|---|---|---|
| Abonnement diffuseur | 5 € / mois / diffuseur | **6 à 60 diffuseurs payants** couvrent l'infrastructure ; ≈ 350 diffuseurs sur 12 mois amortissent le développement |
| Coût marginal d'un auditeur supplémentaire | ≈ 0,17 Go / heure d'écoute | **Quasi nul en calcul, non nul en bande passante** — d'où un modèle qui doit facturer le diffuseur, jamais l'auditeur |

**Conclusion de faisabilité.** Le projet est faisable techniquement dans son périmètre MVP, avec un
verrou de mise à l'échelle horizontale identifié et documenté. Il est faisable financièrement à l'échelle
pilote pour un coût d'infrastructure marginal. Le risque principal est organisationnel : une seule
personne.

---

## 3. Veille technologique et concurrentielle

Traitée intégralement dans **[`veille-technologique.fr.md`](veille-technologique.fr.md)** : plan de
veille (7 axes, sources, outils, méthode de tri en trois questions), analyse concurrentielle (Twitch,
Spotify Live, Clubhouse, Mixlr, Icecast/Shoutcast), comparatif d'architectures de transport
(WebSocket / HLS / WebRTC / RTMP), journal des décisions, et surtout les quatre cas où la veille a
directement modifié le code.

**Synthèse de l'arbitrage fondateur.** WebSocket est retenu : la latence est le cœur de la proposition de
valeur (ce qui écarte HLS), le flux est unidirectionnel (ce qui rend la complexité de WebRTC non
rentable), et le protocole traverse proxys et pare-feux sur le port 443. Contrepartie assumée et
documentée : pas de distribution par CDN, donc un bus de messages est le prérequis à toute mise à
l'échelle horizontale.

---

## 4. Architecture macroscopique

Détail complet, diagrammes et alternatives textuelles dans **[`architecture/`](architecture/)** :
[vue d'ensemble et découpage en couches](architecture/README.md) ·
[modèle de données](architecture/modele-de-donnees.md) ·
[diagrammes de séquence](architecture/diagrammes-de-sequence.md) ·
[processus BPMN](architecture/processus-bpmn.md) ·
[schéma de sécurité](architecture/securite.md) ·
[déploiement et CI/CD](architecture/deploiement.md).

**Principes structurants**

| Principe | Traduction concrète |
|---|---|
| Contrat d'abord | `backend/api/openapi.yaml` est la source de vérité ; les types serveur sont **générés**, jamais écrits à la main ; la CI régénère et compile, ce qui rend toute dérive contrat/code impossible à fusionner |
| Séparation en couches | présentation → métier → infrastructure, dépendances strictement descendantes |
| Le temps réel ne passe pas par le contrat REST | Les deux endpoints WebSocket sont hors OpenAPI, spécification qui ne modélise pas les flux bidirectionnels persistants ; ils sont montés directement sur le routeur et documentés séparément |
| Dégradation locale, jamais globale | Buffer borné par auditeur + abandon de trame |
| Règles métier critiques portées par la base | Index unique partiel pour « un seul live par diffuseur » |
| Le code inactif est déployable | Feature flags en base : livrer n'est pas activer |
| Aucune valeur en dur | Configuration 100 % par variables d'environnement, échec au démarrage si un secret manque |

---

## 5. Spécifications fonctionnelles

Les besoins détaillés sont formalisés en user stories avec critères d'acceptation dans
**[`user-stories.md`](user-stories.md)** (20 stories, 5 épopées, table de traçabilité vers les endpoints,
les scénarios de recette et les feature flags).

### 5.1 Matrice des droits par rôle

| Fonctionnalité | `anonymous` | `user` | `broadcaster` | `admin` |
|---|:---:|:---:|:---:|:---:|
| Parcourir et écouter les lives publics | ✅ | ✅ | ✅ | ✅ |
| Écouter les pistes publiées | ✅ | ✅ | ✅ | ✅ |
| Créer un compte, gérer son profil | — | ✅ | ✅ | ✅ |
| Favoris | — | ✅ | ✅ | ✅ |
| Playlists | — | ✅ | ✅ | ✅ |
| Démarrer / arrêter un live | — | — | ✅ | ✅ |
| Publier / supprimer une piste | — | — | ✅ (les siennes) | ✅ (toutes) |
| Gérer les feature flags | — | — | — | 🚧 |
| Gérer les comptes, consulter les indicateurs globaux | — | — | — | 🚧 |

La hiérarchie est **ordinale** : un contrôle s'écrit toujours « au moins ce rôle », jamais « exactement
ce rôle ». Un administrateur hérite mécaniquement de tous les droits inférieurs.

### 5.2 Niveaux de performance cibles

| Indicateur | Cible | Mesure | Statut |
|---|---|---|---|
| Latence audio de bout en bout | **< 2 s** | Chronométrage en recette (R-06) | ✅ atteint en conditions nominales |
| Délai de démarrage du son | < 3 s | Recette R-06 | ✅ |
| Auditeurs simultanés par instance | **≥ 500** | `STREAM_MAX_LISTENERS` par défaut ; à confirmer par test de charge | 🚧 à mesurer |
| Empreinte mémoire par auditeur | ≤ 70 Ko (buffer 64 Ko + pile de goroutine) | Profil `pprof` | 🚧 à mesurer |
| Latence API (p95) | < 300 ms | Métriques HTTP | 🚧 |
| Disponibilité | 99 % en pilote | Sonde `/health` | 🚧 |
| Taux de trames abandonnées | < 1 % des trames diffusées | Métrique `lyo_chunks_dropped_total` | 🚧 |
| Fluidité de l'interface | 60 FPS pendant l'écoute | Capture Flutter DevTools en mode profil | 🚧 à produire |

### 5.3 Règles de gestion

| # | Règle | Où elle est appliquée |
|---|---|---|
| RG-01 | Un diffuseur ne peut avoir qu'un seul live actif | Index unique partiel PostgreSQL |
| RG-02 | Seul le propriétaire d'un live peut y pousser de l'audio | Vérification dans le handler d'ingestion, testée |
| RG-03 | Un live terminé n'accepte ni ingestion ni écoute | Vérification de statut, testée |
| RG-04 | Une piste ne peut être supprimée que par son propriétaire ou un administrateur | Contrôle de propriété dans le handler |
| RG-05 | Une playlist est privée par défaut | Valeur par défaut `is_public = false` |
| RG-06 | L'ordre des pistes d'une playlist est la file d'attente de lecture | Tableau ordonné `track_ids` |
| RG-07 | Un jeton de réinitialisation est à usage unique et expirant | `used_at` + `expires_at` |
| RG-08 | Une fonctionnalité désactivée répond 503, jamais 404 ni 500 | Contrôle du flag en tête de handler |
| RG-09 | Un auditeur saturé perd ses propres trames, jamais celles des autres | Branche `default` du `select` dans `Hub.Broadcast` |

---

## 6. Spécifications techniques

### 6.1 Pile technologique et justification

| Couche | Choix | Alternative écartée | Justification |
|---|---|---|---|
| Langage backend | **Go 1.26** | Node.js, Python | Concurrence native (goroutine ≈ 2 Ko de pile initiale contre ~1 Mo pour un thread OS), absence de verrou global d'interpréteur, binaire statique de quelques mégaoctets permettant une image alpine minimale, ramasse-miettes à faible latence adapté au temps réel |
| Routeur HTTP | **chi** | Gin, Echo, `net/http` seul | Compatible `http.Handler` : aucun verrouillage propriétaire, tout l'écosystème de middlewares standard fonctionne (ADR 001) |
| Contrat d'API | **OpenAPI + oapi-codegen** | Handlers écrits à la main | Contrat unique et versionné, types générés : la dérive silencieuse entre mobile et backend devient impossible (ADR 005) |
| Transport temps réel | **WebSocket (`coder/websocket`)** | HLS, WebRTC, RTMP | Voir § 3 et ADR 003, ADR 006 |
| Base de données | **PostgreSQL 17 + pgx/v5** | ORM (GORM) | SQL explicite, pas de requête cachée ni de N+1, `pgxpool` performant, types PostgreSQL avancés (tableaux, ENUM, index partiels) réellement exploités (ADR 008) |
| Migrations | **golang-migrate + `embed.FS`** | Scripts manuels | Migrations embarquées dans le binaire, appliquées au démarrage, chacune réversible |
| Stockage audio | **S3 / MinIO, URL présignées** | Upload transitant par l'API | Le backend ne porte jamais les octets audio (ADR 010) |
| Mobile | **Flutter 3.41 + `provider`** | Natif Swift/Kotlin, Riverpod | Base de code unique ; `provider` retenu après retour d'expérience (ADR 004) |
| Audio mobile | `just_audio`, `audio_service`, `record` | — | Décodage hors thread d'interface, lecture en arrière-plan, capture AAC-LC |
| Observabilité | **slog (JSON) ✅ + OpenTelemetry 🚧** | Format propriétaire | Standard neutre, non lié à un fournisseur (ADR 009) |
| Conteneurisation | Docker multi-étapes alpine | Image complète | Image finale minimale, surface d'attaque réduite |
| CI/CD | GitHub Actions | Jenkins, GitLab CI | Intégré au dépôt, secrets chiffrés, coût nul en dépôt public |

### 6.2 Contraintes techniques

| Contrainte | Origine | Traitement |
|---|---|---|
| Un handshake WebSocket ne transporte pas d'en-tête personnalisé | RFC 6455 / API navigateur | Repli d'authentification par paramètre `?token=`, avec vérification identique du jeton — testé |
| L'état des Hubs vit en mémoire du processus | Choix de conception assumé | Une seule instance ; bus de messages requis avant toute réplication |
| Android bloque le trafic HTTP en clair depuis l'API 28 | Plateforme | **Rend TLS obligatoire pour une distribution réelle** — reverse proxy Caddy et certificats ACME automatiques ; le trafic en clair n'est autorisé que dans les builds *debug* et *profile* |
| Le débit sortant est le facteur limitant avant le CPU | Résultat de l'étude de coûts (§ 2.3) | Aucun transcodage : les trames AAC sont relayées telles quelles |
| Timeouts obligatoires par requête | Politique interne | `context.WithTimeout` en tête de handler, 503 sur dépassement |

### 6.3 Environnements

| Environnement | Composition | Déclenchement |
|---|---|---|
| Local | Stack complète : Postgres, MinIO, Mailpit, pgAdmin, observabilité | `docker compose up -d` |
| Intégration (CI) | Runner GitHub | Chaque pull request |
| Production | VPS, 6 conteneurs, réseau Docker privé | Fusion sur `main` |
| Pré-production | — | 🚧 planifié |

---

## 7. Accessibilité et inclusion

**Engagement.** L'application vise le niveau **WCAG 2.1 AA**, référentiel sur lequel s'aligne le RGAA.
L'accessibilité est traitée comme une exigence fonctionnelle (user story US-18, scénario de recette
R-14), et non comme une finition.

| Exigence | Traduction dans le produit | Statut |
|---|---|---|
| Compatibilité lecteur d'écran | Libellé accessible sur tout contrôle icône seule (lecture/pause, favori, micro, mini-lecteur) | 🚧 |
| Annonce des changements d'état | Le passage lecture/pause et la connexion à un live sont annoncés vocalement | 🚧 |
| Contraste | Ratio ≥ 4,5:1 sur les textes, vérifié sur les jetons de design | 🚧 à auditer |
| Cibles tactiles | ≥ 48 × 48 dp | 🚧 à auditer |
| Redimensionnement du texte | Interface utilisable à 200 %, sans troncature ni superposition | 🚧 |
| Information jamais portée par la seule couleur | L'indicateur « en direct » associe une pastille **et** un texte | 🚧 |
| Navigation cohérente | Structure d'écrans stable, retour arrière prévisible | ✅ |
| Preuve automatisée | Test Flutter `meetsGuideline` (contraste, cibles tactiles) exécuté en CI | 🚧 |

**Accessibilité de la documentation** — traitée dès à présent :

- **Chaque diagramme est doublé d'une alternative textuelle** décrivant intégralement son contenu, ce
  qui rend la documentation d'architecture exploitable au lecteur d'écran.
- Les diagrammes sont écrits en Mermaid, donc en **texte** : ils sont lisibles, indexables et
  versionnables, contrairement à une image exportée.
- Hiérarchie de titres stricte et sans saut de niveau ; tableaux avec en-têtes explicites.
- Aucune information portée par la seule couleur : les statuts sont écrits (« implémenté », « planifié »)
  autant que symbolisés.
- Documentation disponible en **français et en anglais** (résumé anglais en fin de chaque document,
  versions intégrales pour les documents principaux).

---

## 8. Numérique responsable

### 8.1 Sobriété de l'architecture

| Levier | Effet | Statut |
|---|---|---|
| **Mutualisation du flux** | Un flux entrant est fan-outé vers N auditeurs : **une seule** ingestion et un seul traitement, au lieu de N pipelines indépendants. C'est le gain de sobriété le plus important de l'architecture. | ✅ |
| **Aucun transcodage** | Les trames AAC sont relayées telles quelles. Le transcodage adaptatif est le poste CPU dominant des plateformes de streaming ; nous ne le payons pas. | ✅ (par conception) |
| **Go plutôt qu'un environnement interprété** | Empreinte mémoire et CPU par requête inférieure d'un ordre de grandeur à une pile Node.js ou Python équivalente | ✅ |
| **Image alpine multi-étapes** | Image finale de quelques dizaines de mégaoctets : moins de stockage, moins de transfert à chaque déploiement, moins de surface d'attaque | ✅ |
| **Un seul VPS plutôt qu'un cluster** | Pas de plan de contrôle Kubernetes à alimenter en permanence pour un service mono-processus | ✅ |
| **Buffers bornés** | La consommation mémoire est plafonnée et prévisible ; pas de croissance non maîtrisée sous charge | ✅ |
| **Upload direct vers le stockage objet** | Les octets audio ne sont pas copiés deux fois à travers le backend | ✅ |
| **Extinction des ressources en fin de live** | `Hub.Close()` libère goroutines et canaux ; aucune ressource ne survit à son usage | ✅ |

### 8.2 Sobriété côté terminal

| Levier | Effet | Statut |
|---|---|---|
| Audio uniquement | La consommation d'énergie et de données d'un flux audio est un à deux ordres de grandeur sous celle de la vidéo | ✅ |
| Débit fixe à 128 kbit/s | ≈ 57 Mo par heure d'écoute — compatible avec un forfait mobile modeste | ✅ |
| Décodage hors thread d'interface | Pas de réveil CPU inutile, moins de consommation batterie | ✅ |
| Reconstruction ciblée de l'interface | Seuls les sous-arbres abonnés sont redessinés lors d'un changement d'état | ✅ |
| Mode hors-ligne (cache de playlist) | Éviterait de retélécharger les mêmes pistes | 🚧 |

### 8.3 Ce qui n'est pas fait, et qui devrait l'être

- Aucune mesure réelle d'empreinte carbone ni de consommation électrique : les affirmations ci-dessus
  sont argumentées par conception, pas mesurées. Une mesure de la consommation électrique du VPS sous
  charge serait la preuve manquante.
- Aucune politique de purge des données : les pistes et les lives terminés s'accumulent indéfiniment.
  Une politique de rétention réduirait à la fois le stockage et l'exposition RGPD (§ 12).

---

## 9. Analyse des risques

Cotation : Probabilité (P) et Impact (I) de 1 (faible) à 4 (critique) ; Criticité = P × I.

| # | Risque | Nature | P | I | Crit. | Mitigation | Statut |
|---|---|---|:-:|:-:|:-:|---|---|
| **R1** | **Trafic non chiffré** : les jetons JWT circulent en clair ; Android bloque par ailleurs le trafic HTTP en clair depuis l'API 28 | Technique / sécurité | 4 | 4 | **16** | Reverse proxy Caddy avec TLS Let's Encrypt automatique, `wss://`, port 8080 non publié, trafic en clair refusé dans l'APK de release | ✅ traité ([ADR 012](adr/012-tls-reverse-proxy.md)) |
| **R2** | **Droits d'auteur** sur l'audio diffusé par les utilisateurs | Juridique / métier | 3 | 4 | **12** | Conditions d'utilisation transférant la responsabilité au diffuseur, procédure de signalement et de retrait, journalisation des diffusions, absence de catalogue sous licence fournie par la plateforme | 🚧 |
| **R3** | **Bus factor de 1** : une seule personne maîtrise l'ensemble | Humain | 4 | 3 | **12** | Documentation systématique (ADR, ce cahier des charges, guides), automatisation intégrale de la chaîne de livraison, conventions strictes | ✅ atténué |
| **R4** | **Impossibilité de mise à l'échelle horizontale** : l'état des Hubs est en mémoire | Technique | 3 | 4 | **12** | Limite documentée et assumée ; bus Redis Pub/Sub ou NATS avec routage par flux ; à ce stade, mise à l'échelle verticale uniquement | 🚧 documenté |
| **R5** | **Bruteforce et spam** sur les endpoints d'authentification | Sécurité | 3 | 3 | **9** | Limitation de débit sur `/auth/login`, `/auth/register`, `/auth/forgot-password` | 🚧 |
| **R6** | **Coût de bande passante** hors de contrôle en cas de succès | Financier | 2 | 4 | **8** | Le coût est modélisé (§ 2.3) ; plafonnement du nombre d'auditeurs (`STREAM_MAX_LISTENERS`), alerte sur le débit sortant, modèle de revenus facturant le diffuseur | ✅ partiellement |
| **R7** | **Vulnérabilité d'une dépendance** non détectée | Sécurité | 3 | 3 | **9** | `govulncheck`, `gosec`, Trivy et Dependabot en CI | 🚧 |
| **R8** | **Déploiement défectueux non détecté** : le conteneur redémarre en boucle sans que personne ne le sache | Exploitation | 3 | 3 | **9** | Smoke test sur `/health` après déploiement, échec du job, retour arrière sur le tag SHA précédent | 🚧 |
| **R9** | **Perte de données** en cas de défaillance du VPS | Exploitation | 2 | 4 | **8** | Volume persistant ✅ ; sauvegarde automatisée et restauration testée 🚧 | 🚧 |
| **R10** | **Non-conformité RGPD** : absence de suppression de compte et de politique de rétention | Réglementaire | 3 | 3 | **9** | `DELETE /users/me`, purge des jetons expirés, politique de confidentialité (§ 12) | 🚧 |
| **R11** | **Application inutilisable en situation de handicap** | Réglementaire / inclusion | 3 | 3 | **9** | Libellés accessibles, contrastes, cibles tactiles, test automatisé (§ 7) | 🚧 |
| **R12** | **Dérive entre documentation et code** | Qualité | 2 | 3 | **6** | Contrat OpenAPI régénéré et compilé en CI ; ADR mis à jour ou marqué obsolète lors de tout revirement ; statuts ✅/🚧 explicites dans toute la documentation | ✅ atténué |
| **R13** | **Dépendance à un fournisseur unique** (VPS, GHCR) | Stratégique | 2 | 2 | **4** | Tout est conteneurisé et décrit en `docker-compose` : la migration vers un autre hébergeur est mécanique | ✅ |
| **R14** | **Défaillance du stockage objet** rendant les pistes inaccessibles | Technique | 2 | 3 | **6** | Le service reste disponible pour le direct même si le stockage est indisponible ; dégradation partielle et non totale | ✅ par conception |

**Les trois risques à traiter en premier**, par criticité : R1 (chiffrement en transit), puis R2 (droits
d'auteur) et R3 (bus factor) à égalité avec R4 (mise à l'échelle).

---

## 10. Feuille de route et pilotage

### 10.1 Méthodologie

Approche **Agile / DevOps adaptée à une équipe d'une personne** :

- Une branche par unité de travail, nommée `type/description` en miroir du type de commit.
- Commits conventionnels (`feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`) — **66 commits**, dont
  29 `feat` et 7 `fix`.
- Une pull request par unité de travail, avec la CI comme relecteur automatique systématique
  (**35 PR ouvertes à ce jour**).
- Versions taguées `vX.Y.Z` déclenchant une Release GitHub automatisée (**v1.0.0 à v1.1.0**).
- Décisions structurantes consignées en ADR au moment où elles sont prises.

### 10.2 Réalisé

| Jalon | Période | Contenu | État |
|---|---|---|---|
| J1 — Socle | Avril 2026 | Initialisation, schéma de base, contrat OpenAPI, CI backend et mobile | ✅ |
| J2 — Authentification | Avril–Mai 2026 | JWT, rôles, inscription et connexion, écrans mobiles | ✅ |
| J3 — Diffusion live | Mai–Juin 2026 | Hub, fan-out, WebSocket ingest/listen, diffuseur et lecteur mobiles | ✅ |
| J4 — Déploiement continu | Juin 2026 | GHCR, déploiement VPS, releases mobiles signées | ✅ |
| J5 — Contenu | Juillet–Août 2026 | Pistes, upload S3 présigné, playlists, favoris, profil, réinitialisation de mot de passe | ✅ |
| J6 — Qualité | Septembre 2026 | 293 tests, couverture ≥ 80 % sur tous les packages métier, seuil bloquant en CI (PR #35) | ✅ |
| J7 — Documentation | Septembre 2026 | Cahier des charges, architecture, user stories, plan de tests, guides, ADR | ✅ ce lot |

### 10.3 Reste à faire, par ordre de valeur

| Priorité | Lot | Contenu | Risque / critère levé | Charge estimée |
|:-:|---|---|---|---|
| ~~**1**~~ | ~~**TLS et sécurisation du transport**~~ | Reverse proxy Caddy, `wss://`, port 8080 non publié, CORS externalisé en variable d'environnement | R1 | ✅ livré |
| **2** | **Instrumentation OpenTelemetry** | Traces (jusqu'à la base via `otelpgx`), métriques techniques et **métriques métier** (auditeurs actifs, trames abandonnées, octets diffusés, déconnexions brutales), propagation `traceparent` depuis le mobile | O6 | 3 j |
| **3** | **Tableaux de bord et alertes** | Tempo comme backend de traces, tableau de bord Grafana provisionné (auditeurs, débit, taux d'erreurs, latence), règles d'alerte, collecte des logs vers Loki avec identifiant de trace | O6, R8 | 2 j |
| **4** | **Accessibilité** | Libellés accessibles, contrastes, cibles tactiles, annonces d'état, test automatisé | O7, R11 | 2 j |
| **5** | **Sécurité de la chaîne** | `govulncheck`, `gosec`, Trivy, Dependabot, limitation de débit | R5, R7 | 1,5 j |
| **6** | **Administration** | Implémentation des handlers de feature flags, gestion des comptes, indicateurs globaux, écran mobile admin, **chargement des flags depuis l'API côté mobile** | US-15 à US-17 | 3 j |
| **7** | **Conformité RGPD** | `DELETE /users/me`, export de données, purge des jetons expirés, politique de confidentialité | R10 | 1,5 j |
| **8** | **Fiabilité du déploiement** | Smoke test post-déploiement, retour arrière automatique, environnement de pré-production | R8 | 1,5 j |
| **9** | **Finitions du lecteur** | Contrôle du volume, gestion des interruptions (appel entrant, casque débranché), file d'attente de playlist avec réordonnancement | US-06, US-08, US-10 | 2 j |
| **10** | **Mesures de charge** | Benchmarks du fan-out à 10/100/1000 auditeurs, profils `pprof`, test de charge, capture 60 FPS | O4, § 5.2 | 2 j |
| 11 | Chat WebSocket *(bonus)* | Second Hub textuel réutilisant le même patron | Fonctionnalité supplémentaire | 3 j |
| 12 | Recommandations *(bonus)* | Historique d'écoute → recommandation par co-occurrence : collecte, stockage, analyse | Fonctionnalité supplémentaire | 4 j |

### 10.4 Outil de pilotage

Le suivi est assuré par **GitHub** comme outil unique : issues pour les unités de travail, Projects
(vue Kanban : *À faire* → *En cours* → *En revue* → *Terminé*) pour la visualisation du flux, pull
requests numérotées pour la traçabilité des décisions, et Releases pour la communication de version.

**L'historique Git constitue lui-même la preuve du pilotage** : chaque fonctionnalité est rattachable à
sa branche, sa PR, sa revue et son déploiement. Les captures du tableau Kanban sont présentées en
soutenance.

---

## 11. Indicateurs de performance (KPI)

### 11.1 KPI de projet

| Indicateur | Définition | Valeur actuelle | Cible |
|---|---|---|---|
| Nombre de pull requests fusionnées | Unités de travail livrées | 35 | — |
| Répartition des commits par type | Part de `feat` sur le total | 29 `feat` / 66 commits ≈ 44 % | ≥ 40 % (le reste étant correctifs, tests, documentation) |
| Ratio correctifs / fonctionnalités | `fix` ÷ `feat` | 7 / 29 ≈ **0,24** | < 0,3 — un ratio bas indique des fonctionnalités livrées stables |
| Délai de livraison (*lead time*) | De la fusion au déploiement en production | < 10 min (automatisé) | < 15 min |
| Taux de réussite de la CI | Builds verts ÷ builds totaux | à mesurer | > 90 % |
| Couverture de tests backend | Hors code généré | **≥ 80 % sur tous les packages métier** | ≥ 80 %, **bloquant** |
| Nombre de tests automatisés | Fonctions de test backend | **293** | croissant |
| Décisions documentées | ADR rédigés | 10 | 1 par décision structurante |
| Fréquence de déploiement | Déploiements en production par mois | ≈ 6 | ≥ 4 |

### 11.2 KPI de produit et d'exploitation

| Indicateur | Définition | Source | Statut |
|---|---|---|---|
| Auditeurs simultanés | `lyo_listeners_active` par flux | Métrique métier | 🚧 |
| Lives simultanés | `lyo_streams_live` | Métrique métier | 🚧 |
| **Taux de trames abandonnées** | `lyo_chunks_dropped_total` rapporté aux trames diffusées — **indicateur d'expérience**, distinct des erreurs techniques | Métrique métier | 🚧 |
| Déconnexions brutales | `lyo_listener_disconnect_total{reason="abrupt"}` | Métrique métier | 🚧 |
| Débit sortant | `rate(lyo_stream_bytes_total[1m])` — pilote directement le coût | Métrique métier | 🚧 |
| Durée moyenne de diffusion | `lyo_broadcast_duration_seconds` | Métrique métier | 🚧 |
| Taux d'erreurs techniques | Part des réponses 5xx — **indicateur technique** | Métrique HTTP | 🚧 |
| Latence API p50 / p95 / p99 | Histogramme de durée des requêtes | Métrique HTTP | 🚧 |
| Disponibilité | Part de sondes `/health` réussies | Sonde | 🚧 |
| Taux d'absence de plantage (mobile) | Sessions sans crash | Outillage mobile | 🚧 |

> **Distinction essentielle, exigée par le sujet.** Une erreur 500 est une **défaillance technique** : du
> code a échoué. Une déconnexion brutale ou une trame abandonnée est une **dégradation d'expérience** : le
> système fonctionne comme conçu, mais l'utilisateur en souffre. Ces deux familles répondent à des
> décisions différentes — la première appelle un correctif, la seconde appelle un dimensionnement — et
> doivent donc être visualisées séparément sur le tableau de bord.

---

## 12. Conformité : RGPD, ANSSI, ITIL

### 12.1 RGPD

**Données traitées**

| Donnée | Finalité | Base légale | Conservation |
|---|---|---|---|
| Adresse e-mail | Identification, réinitialisation du mot de passe | Exécution du contrat | Durée de vie du compte |
| Nom d'utilisateur | Affichage public | Exécution du contrat | Durée de vie du compte |
| Empreinte du mot de passe (bcrypt) | Authentification | Exécution du contrat | Durée de vie du compte |
| Rôle | Habilitation | Exécution du contrat | Durée de vie du compte |
| Favoris, playlists | Personnalisation du service | Exécution du contrat | Durée de vie du compte |
| Contenus publiés (lives, pistes) | Objet même du service | Exécution du contrat | Jusqu'à suppression par l'auteur |
| Jetons de réinitialisation | Sécurité du parcours de récupération | Intérêt légitime | Expiration courte, puis purge 🚧 |

**Principe de minimisation.** Aucun nom réel, aucune date de naissance, aucune donnée de localisation,
aucun traceur publicitaire, aucun identifiant tiers. La collecte se limite au strict nécessaire au
fonctionnement du service — c'est une décision de conception, pas une conséquence.

**Droits des personnes**

| Droit | État | Moyen |
|---|---|---|
| Accès | ✅ | `GET /users/me` |
| Rectification | ✅ | `PATCH /users/me` |
| **Effacement** | 🚧 | `DELETE /users/me` à implémenter — les clés étrangères en `ON DELETE CASCADE` rendent l'opération atomique et complète |
| Portabilité | 🚧 | Export JSON des données du compte à implémenter |
| Opposition / limitation | 🚧 | Suspension de compte |
| Information | 🚧 | Politique de confidentialité à rédiger et à exposer dans l'application |

**Sécurité du traitement.** Mots de passe hachés avec bcrypt ; jetons de réinitialisation stockés hachés,
expirants et à usage unique ; base de données non exposée sur le réseau public ; secrets hors du code
source ; journalisation ne contenant jamais de mot de passe ; chiffrement en transit assuré de bout en
bout par le reverse proxy TLS, flux audio en direct compris.

**Sous-traitants** (au sens de l'article 28) : hébergeur du VPS, fournisseur de stockage objet,
fournisseur SMTP, GitHub pour la chaîne de fabrication.

**Violation de données.** Procédure à formaliser : détection par les alertes, qualification, notification
à la CNIL sous 72 heures, information des personnes concernées si le risque est élevé. 🚧

### 12.2 Recommandations ANSSI appliquées

| Recommandation | Application | Statut |
|---|---|---|
| Secrets hors du code source | Variables d'environnement, GitHub Secrets, `.gitignore` vérifié | ✅ |
| Moindre privilège | Rôles ordinaux, `GITHUB_TOKEN` restreint à `packages: write`, Postgres non publié | ✅ |
| Cloisonnement réseau | Réseau Docker privé, publication minimale des ports | ✅ |
| Journalisation | Logs structurés horodatés avec identifiant de corrélation | ✅ |
| Défense en profondeur | Rôle **et** propriété **et** feature flag vérifiés indépendamment ; contrainte métier portée en base | ✅ |
| Chiffrement des flux | TLS terminé par Caddy, certificats ACME renouvelés automatiquement | ✅ |
| Maintien en condition de sécurité | Analyse automatisée des dépendances | 🚧 |
| Sauvegarde et restauration | Volume persistant ✅ ; sauvegarde automatisée et restauration testée 🚧 | 🚧 |

### 12.3 Inspiration ITIL

| Processus ITIL | Transposition dans le projet |
|---|---|
| Gestion des changements | Toute modification passe par une pull request, la CI et un déploiement automatisé et tracé |
| Gestion des versions | Versions sémantiques taguées, Releases GitHub, images taguées par SHA de commit |
| Gestion des incidents | Alertes 🚧 → diagnostic par tableaux de bord et journaux → correctif ou coupure par feature flag → post-mortem en issue |
| Gestion des problèmes | Toute anomalie corrigée est accompagnée d'un **test de non-régression** : la cause racine est verrouillée, pas seulement le symptôme |
| Gestion de la configuration | Configuration intégralement en variables d'environnement, documentée dans `.env.example` |
| Gestion de la continuité | Retour arrière par image SHA ; migrations réversibles ; coupure fonctionnelle par flag |

---

## 13. Bilan et analyse réflexive

### 13.1 Prévisionnel contre réalisé

| Élément | Prévu | Réalisé | Écart et analyse |
|---|---|---|---|
| Diffusion live | MVP | ✅ Livré et testé à 96,6 % de couverture | Conforme — le cœur technique a été traité en premier, ce qui était le bon ordre |
| Contenu (pistes, playlists, favoris) | MVP | ✅ Livré | Conforme |
| CI/CD | MVP | ✅ Livré dès avril | **En avance** — investir tôt dans l'automatisation a produit des intérêts composés sur tout le projet |
| Observabilité | MVP | 🚧 Infrastructure déployée, **application non instrumentée** | **Écart le plus important.** Analyse en § 13.3 |
| Tests | ≥ 80 % | ✅ Atteint tardivement (septembre) | **Écart de calendrier.** Analyse en § 13.3 |
| Sécurité du transport (TLS) | Implicite | ✅ Livré tardivement (septembre) | Sous-estimé : classé « infrastructure » alors que c'est une exigence produit bloquante sur Android |
| Accessibilité | Implicite | 🚧 Non fait | Non planifié explicitement, donc non fait — c'est la leçon principale du projet |
| Administration | MVP | 🚧 Contrat défini, implémentation absente | Priorisé après le cœur métier, à tort : c'est aussi une fonctionnalité d'exploitation |

### 13.2 Trois décisions correctives assumées

Ces trois épisodes relèvent du pilotage et de l'ajustement de plan en cours de projet.

**① Migration de Riverpod vers Provider** (commit `c6a9995`). Riverpod avait été choisi sur sa réputation
et documenté en ADR 002. À l'usage, les états de l'application se sont révélés simples et essentiellement
locaux à une fonctionnalité : la puissance de Riverpod n'était pas exploitée, tandis que ses conventions
alourdissaient chaque ajout. Décision : migrer vers `provider` / `ChangeNotifier`. Le point important
n'est pas la migration mais son traitement documentaire — l'ADR 002 a été passé en **Superseded** et un
ADR 004 explique le revirement. Un ADR qui documente un changement d'avis assumé vaut mieux qu'un ADR
devenu faux, et c'est exactement ce que la démarche exige : documenter le *pourquoi*, y compris le
pourquoi d'un retour en arrière.

**② Descente de la contrainte « un seul live » dans la base** (migration 000006). La règle était portée
par le code : `HasLive()` puis `Create()`. Deux requêtes simultanées pouvaient passer entre les deux.
Plutôt que d'ajouter un verrou applicatif, la contrainte a été confiée à un index unique partiel
PostgreSQL. La règle est devenue inviolable, y compris avec plusieurs instances, et le code s'est
simplifié.

**③ Persistance de la lecture à la sortie de l'écran du lecteur** (commit `fb25144`). Un défaut détecté à
l'usage — quitter l'écran du lecteur coupait le son — a révélé un couplage entre le cycle de vie de l'UI
et celui de la lecture. Le correctif a séparé les deux, ce qui a permis dans la foulée le mini-lecteur
persistant. Un défaut a produit une amélioration fonctionnelle.

### 13.3 Ce que je ferais différemment

**Instrumenter l'observabilité en même temps que la première fonctionnalité, et non après.** La stack
d'observabilité a été déployée tôt, ce qui a créé une illusion d'avancement : les conteneurs tournaient,
les datasources étaient provisionnées, mais l'application n'émettait rien. La leçon est qu'une
infrastructure d'observabilité sans instrumentation applicative ne vaut rien — et qu'il est plus facile
d'instrumenter un handler au moment où on l'écrit que d'en instrumenter trente-six après coup. Le bon
réflexe aurait été de traiter la première trace de bout en bout comme un critère d'acceptation du premier
endpoint.

**Écrire les tests dans la même PR que le code, dès le premier jour.** La couverture est passée de moins
de 15 % à plus de 80 % en un seul lot tardif (PR #35). Le résultat est bon, mais la méthode est
mauvaise : pendant plusieurs mois, chaque refactorisation s'est faite sans filet, et une partie des tests
a dû être écrite en redécouvrant l'intention du code. Le seuil de couverture bloquant en CI, ajouté par
cette même PR, est précisément le mécanisme qui aurait dû exister dès le premier commit — une règle qui
n'est pas mécanisée n'est pas une règle.

**Traiter l'accessibilité comme une exigence fonctionnelle, pas comme une finition.** Aucune ligne
d'accessibilité n'a été écrite parce qu'aucune user story ne l'exigeait. C'est une démonstration nette
qu'une exigence non écrite est une exigence non livrée. La correction adoptée est structurelle :
l'accessibilité est désormais la user story US-18 avec des critères d'acceptation vérifiables et le
scénario de recette R-14, et sera vérifiée par un test automatisé — au même titre que n'importe quelle
fonctionnalité.

### 13.4 Ce que je referais à l'identique

- **Le contrat OpenAPI d'abord.** Aucune dérive entre le mobile et le backend en six mois : le générateur
  et la CI l'ont rendue structurellement impossible.
- **Le CI/CD dès la première semaine.** Chaque jour suivant en a bénéficié ; le déploiement n'a jamais
  été un événement redouté.
- **Les feature flags en base.** Livrer du code inactif, puis l'activer sans redéploiement, offre un
  coupe-circuit immédiat qu'aucune procédure de retour arrière n'égale en rapidité.
- **La politique d'abandon de trame.** C'est la décision d'architecture dont je suis le plus satisfait :
  elle est simple, elle borne la dégradation à celui qui la subit, et elle est prouvée par un test.
- **Documenter les décisions au moment où elles sont prises.** Les ADR écrits sur le coup sont justes ;
  ceux reconstitués a posteriori auraient été des rationalisations.

---

## Annexes

| Document | Contenu |
|---|---|
| [`architecture/`](architecture/) | Diagrammes UML et BPMN, modèle de données, sécurité, déploiement |
| [`user-stories.md`](user-stories.md) | 20 user stories avec critères d'acceptation et traçabilité |
| [`plan-de-tests.md`](plan-de-tests.md) | Stratégie de test, couverture mesurée, cahier de recette R-01 à R-15 |
| [`veille-technologique.fr.md`](veille-technologique.fr.md) | Plan de veille, analyse concurrentielle, journal des décisions |
| [`guide-utilisateur.fr.md`](guide-utilisateur.fr.md) | Manuel utilisateur (auditeur, diffuseur, administrateur) |
| [`plan-formation.fr.md`](plan-formation.fr.md) | Plan de formation adapté à la diversité des publics |
| [`adr/`](adr/) | 10 décisions d'architecture, dont une explicitement remplacée |
| [`../README.md`](../README.md) / [`../README.fr.md`](../README.fr.md) | Guide développeur, EN et FR |

# User stories — Lyo / StreamPulse

**Version :** 1.0 · **Date :** 2026-09-03

Format : *En tant que **\<rôle\>**, je veux **\<action\>** afin de **\<bénéfice\>***, suivi de critères
d'acceptation vérifiables. Chaque story porte un identifiant stable, référencé par le cahier de recette
(`docs/plan-de-tests.md`) et par les pull requests.

**Statut :** ✅ livré · 🚧 planifié (voir la feuille de route du cahier des charges).

---

## Épopée 1 — Découverte et accès (rôle `anonymous`)

### US-01 ✅ — Écouter sans créer de compte
**En tant que** visiteur, **je veux** parcourir et écouter les lives publics sans créer de compte,
**afin de** juger de l'intérêt du service avant de m'engager.

- Étant donné que je n'ai aucun compte, quand j'ouvre l'application, alors la liste des lives en cours s'affiche.
- Quand je sélectionne un live, alors la lecture démarre sans demande de connexion.
- Les actions réservées (favori, playlist, diffusion) invitent à se connecter plutôt que d'échouer silencieusement.

> *Choix d'architecture associé :* le middleware d'authentification permissif, qui permet à une même
> route de servir un visiteur anonyme et un utilisateur authentifié (voir ADR 007).

### US-02 ✅ — Créer un compte
**En tant que** visiteur, **je veux** créer un compte avec un e-mail et un mot de passe, **afin d'**
accéder aux favoris et aux playlists.

- Un e-mail ou un nom d'utilisateur déjà pris est refusé avec un message clair.
- Le mot de passe n'est jamais stocké en clair (empreinte bcrypt).
- L'inscription connecte immédiatement, sans seconde saisie.

### US-03 ✅ — Rester connecté
**En tant qu'** utilisateur, **je veux** que ma session persiste entre deux ouvertures de l'application,
**afin de** ne pas ressaisir mes identifiants à chaque usage.

- La session survit à une fermeture complète de l'application.
- L'expiration du jeton d'accès est traitée de façon transparente par le jeton de rafraîchissement.
- La déconnexion efface les jetons du terminal.

### US-04 ✅ — Récupérer un accès perdu
**En tant qu'** utilisateur, **je veux** réinitialiser mon mot de passe par e-mail, **afin de** récupérer
mon compte sans assistance.

- La réponse est identique que le compte existe ou non (pas d'énumération de comptes).
- Le lien ouvre directement l'application (deep link).
- Le lien expire et **ne fonctionne qu'une seule fois**.

---

## Épopée 2 — Écoute (rôle `user`)

### US-05 ✅ — Écouter un live en temps réel
**En tant qu'** auditeur, **je veux** écouter un live avec une latence faible, **afin de** vivre
l'expérience « en direct ».

- Le son démarre en moins de 3 secondes après ouverture.
- La latence perçue reste inférieure à 2 secondes.
- Si ma connexion est trop lente, **la diffusion des autres auditeurs n'est pas affectée** : ma dégradation reste locale.

### US-06 ✅ — Écouter une piste enregistrée avec contrôle de lecture
**En tant qu'** auditeur, **je veux** lire une piste avec lecture/pause et barre de progression,
**afin de** naviguer dans le contenu.

- Position et durée sont affichées ; le déplacement du curseur repositionne la lecture.
- 🚧 Un réglage de volume est disponible dans le lecteur.

### US-07 ✅ — Continuer d'écouter en faisant autre chose
**En tant qu'** auditeur, **je veux** que la lecture continue écran verrouillé ou dans une autre
application, **afin de** ne pas être contraint de rester sur l'écran du lecteur.

- La lecture se poursuit écran verrouillé.
- Les commandes système (notification, écran de verrouillage) permettent lecture/pause.
- Le mini-lecteur reste accessible lors de la navigation dans l'application.

### US-08 🚧 — Ne pas être gêné par les interruptions
**En tant qu'** auditeur, **je veux** que la lecture se mette en pause lors d'un appel entrant ou du
débranchement de mes écouteurs, **afin de** ne pas manquer de contenu ni déranger mon entourage.

- Un appel entrant met la lecture en pause ; la fin de l'appel la reprend.
- Débrancher le casque met la lecture en pause (le son ne bascule pas brutalement sur le haut-parleur).

### US-09 ✅ — Retrouver ce que j'aime
**En tant qu'** utilisateur, **je veux** mettre en favori des pistes, des lives et des playlists,
**afin de** les retrouver rapidement.

- L'ajout et le retrait sont réversibles et immédiats à l'écran.
- Les favoris survivent au redémarrage de l'application.

### US-10 ✅ — Organiser mon écoute en playlists
**En tant qu'** utilisateur, **je veux** créer des playlists et y ajouter des pistes, **afin de**
préparer une écoute continue.

- Création, renommage et suppression sont disponibles.
- L'ordre d'ajout est conservé : il constitue la file d'attente de lecture.
- Une playlist est privée par défaut.
- 🚧 Réordonnancement des pistes et enchaînement automatique piste suivante / précédente.

---

## Épopée 3 — Diffusion (rôle `broadcaster`)

### US-11 ✅ — Diffuser en direct depuis mon téléphone
**En tant que** diffuseur, **je veux** démarrer un live depuis mon téléphone en quelques secondes,
**afin de** diffuser sans matériel ni configuration.

- Un titre et une description suffisent à démarrer.
- La permission micro est demandée explicitement, avec un message d'aide en cas de refus.
- Le live apparaît dans la liste publique en moins de 5 secondes.
- Un diffuseur ne peut pas avoir deux lives simultanés (contrainte garantie par la base de données).

### US-12 ✅ — Arrêter proprement un live
**En tant que** diffuseur, **je veux** arrêter mon live d'un geste, **afin de** libérer les ressources et
informer mes auditeurs.

- Le live passe au statut « terminé » et disparaît de la liste des lives.
- Les auditeurs sont déconnectés proprement et informés.
- Aucune goroutine ni connexion ne subsiste côté serveur.

### US-13 ✅ — Publier une piste enregistrée
**En tant que** diffuseur, **je veux** téléverser un fichier audio, **afin de** proposer du contenu
réécoutable.

- La progression du transfert est visible.
- Le fichier part directement vers le stockage objet, sans transiter par l'API.
- La piste devient listable et ajoutable en playlist dès sa publication.
- 🚧 Reprise ou nouvelle tentative explicite en cas d'échec réseau.

### US-14 ✅ — Supprimer mon contenu
**En tant que** diffuseur, **je veux** supprimer une de mes pistes, **afin de** garder la maîtrise de ce
que je publie.

- Seuls le propriétaire et un administrateur peuvent supprimer.
- L'objet audio correspondant est supprimé du stockage.

---

## Épopée 4 — Administration (rôle `admin`)

### US-15 🚧 — Activer ou couper une fonctionnalité sans redéployer
**En tant qu'** administrateur, **je veux** basculer un feature flag, **afin de** couper immédiatement une
fonctionnalité défaillante sans attendre un déploiement.

- La liste des flags et leur état sont consultables.
- Le basculement prend effet pour tous les clients sans redémarrage du serveur.
- Un flag désactivé fait répondre 503 à l'API et disparaître l'entrée correspondante de l'interface.

### US-16 🚧 — Gérer les comptes
**En tant qu'** administrateur, **je veux** consulter les comptes et modifier un rôle, **afin de**
promouvoir un diffuseur ou traiter un abus.

- La liste est paginée.
- Le changement de rôle est effectif au prochain rafraîchissement de jeton.
- Un compte peut être suspendu ou supprimé.

### US-17 🚧 — Surveiller la santé de la plateforme
**En tant qu'** administrateur, **je veux** consulter les indicateurs globaux, **afin de** détecter une
dégradation avant que les utilisateurs ne la signalent.

- Les auditeurs connectés, le débit de diffusion, le taux d'erreurs et la latence sont visualisables.
- Les erreurs **techniques** (5xx) sont distinguées des dégradations d'**expérience** (déconnexions
  brutales, trames abandonnées).
- Une anomalie déclenche une alerte notifiée.

---

## Épopée 5 — Qualités transverses

### US-18 🚧 — Utiliser l'application avec un lecteur d'écran
**En tant qu'** utilisateur aveugle ou malvoyant, **je veux** que chaque contrôle soit annoncé et
actionnable au lecteur d'écran, **afin d'** utiliser l'application de façon autonome.

- Tout bouton icône seule porte un libellé accessible explicite.
- Les changements d'état de lecture sont annoncés.
- Les cibles tactiles mesurent au moins 48 × 48 dp.
- Les contrastes respectent le ratio 4,5:1 (WCAG AA).
- L'interface reste utilisable à 200 % de taille de police, sans texte tronqué.
- Aucune information n'est portée par la seule couleur (ex. l'indicateur « en direct » est aussi textuel).

### US-19 ✅ — Être protégé dans mes données
**En tant qu'** utilisateur, **je veux** que mes données soient minimales et protégées, **afin de**
faire confiance au service.

- Seuls un nom d'utilisateur, un e-mail et un mot de passe haché sont collectés.
- Le mot de passe n'apparaît dans aucun journal.
- ✅ Chiffrement en transit (TLS/`wss://`) sur l'ensemble des échanges avec le serveur.
- 🚧 Suppression de compte en autonomie (droit à l'effacement).

### US-20 ✅ — Bénéficier de corrections rapides
**En tant qu'** utilisateur, **je veux** que les correctifs arrivent vite et sans régression, **afin de**
disposer d'un service fiable.

- Toute fusion sur la branche principale déclenche lint, tests, contrôle de couverture, build et déploiement.
- Une version défectueuse peut être remplacée par la précédente, chaque image étant taguée par son commit.
- Toute anomalie corrigée est accompagnée d'un test de non-régression.

---

## Traçabilité

| Story | Endpoints / composants | Recette | Feature flag |
|---|---|---|---|
| US-01 | `GET /streams`, `GET /tracks`, WS `/listen` | R-06 | `live_streaming` |
| US-02, US-03 | `POST /auth/register`, `/auth/login`, `/auth/refresh` | R-01, R-02 | — |
| US-04 | `POST /auth/forgot-password`, `/auth/reset-password` | R-03 | — |
| US-05 | WS `/streams/{id}/listen`, `Hub.Subscribe` | R-06, R-07 | `live_streaming` |
| US-06, US-07, US-08 | `RecordedPlayerNotifier`, `LyoAudioHandler` | R-13 | — |
| US-09 | `/users/me/favorites/**` | R-10 | `favorites` |
| US-10 | `/playlists/**` | R-10 | `playlists` |
| US-11, US-12 | `POST /streams`, WS `/ingest`, `DELETE /streams/{id}` | R-05, R-08 | `live_streaming` |
| US-13, US-14 | `POST /tracks/upload-url`, `POST /tracks`, `DELETE /tracks/{id}` | R-09 | `track_uploads` |
| US-15 | `/admin/features/**` | R-12 | — |
| US-16, US-17 | 🚧 à ajouter au contrat OpenAPI | — | — |
| US-18 | Ensemble de l'interface mobile | R-14 | — |
| US-19 | `internal/user`, `internal/auth` | R-02b, R-11 | — |
| US-20 | `.github/workflows/**` | R-15 | — |

---

## English summary

Twenty user stories grouped into five epics (discovery and access, listening, broadcasting,
administration, cross-cutting qualities), each written in the standard "As a role, I want an action, so
that a benefit" form with verifiable acceptance criteria and a delivered/planned status. A traceability
table links every story to its endpoints, its acceptance-test scenario and its governing feature flag.
Accessibility (US-18), data protection (US-19) and delivery reliability (US-20) are treated as
first-class user stories rather than as non-functional footnotes.

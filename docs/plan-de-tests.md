# Plan de tests et cahier de recette — Lyo / StreamPulse

**Version :** 1.0 · **Date :** 2026-09-03

> Ce document définit la **stratégie de test**, la **planification itérative** des tests en parallèle du
> développement, et le **cahier de recette** avec ses scénarios numérotés et leurs résultats attendus.

---

## 1. Objectifs et politique de qualité

| Objectif | Indicateur | Seuil | Vérification |
|---|---|---|---|
| Aucune régression fonctionnelle livrée | Tests unitaires et d'intégration verts | 100 % | Bloquant en CI |
| Couverture du code métier | `go tool cover`, hors code généré | **≥ 80 %** | **Bloquant en CI** |
| Absence d'accès concurrent non protégé | `go test -race` | 0 alerte | Bloquant en CI |
| Absence de dérive contrat / code | `go generate ./internal/api/` puis compilation | 0 diff | Bloquant en CI |
| Qualité statique | `golangci-lint run ./...` / `flutter analyze` | 0 erreur | Bloquant en CI |

**Principe directeur.** Le test est écrit **dans la même pull request** que le code qu'il couvre. Une PR
qui fait baisser la couverture sous le seuil ne peut pas être fusionnée : la garde n'est pas une
convention, elle est mécanisée par le job d'intégration continue.

---

## 2. Typologie des tests

| Niveau | Portée | Outils | Où | Exécution |
|---|---|---|---|---|
| **Unitaire — backend** | Une fonction, un service, un middleware, isolé par doublure | `testing`, doublures maison, `-race` | `backend/**/*_test.go` | Chaque PR |
| **Composant / HTTP** | Un handler complet via `httptest`, du routage à la réponse JSON | `net/http/httptest` | `internal/api/handlers_*_test.go` | Chaque PR |
| **Bout en bout WebSocket** | Chaîne réelle ingest → Hub → listen sur une vraie connexion | `httptest.Server` + client WS | `internal/streaming/ws_test.go` | Chaque PR |
| **Concurrence** | Comportement sous accès simultanés | détecteur de compétition Go | `hub_test.go` | Chaque PR |
| **Unitaire — mobile** | Notifiers, services, décodage de réponses | `flutter_test`, `mocktail` | `mobile/test/**` | Chaque PR |
| **Intégration base de données** 🚧 | Repositories sur un PostgreSQL réel, migrations up/down | tag `//go:build integration` + service Postgres en CI | à créer | Job dédié |
| **Recette fonctionnelle** | Parcours utilisateur complet, manuelle | Cahier de recette § 6 | — | Avant chaque release |
| **Accessibilité** 🚧 | Contrastes, tailles de cibles tactiles | `meetsGuideline` (Flutter) | à créer | Chaque PR |
| **Performance / charge** 🚧 | Coût CPU et mémoire du fan-out | `go test -bench -benchmem`, `pprof` | à créer | À la demande |

---

## 3. Matrice fonctionnalité × type de test

| Fonctionnalité | Unitaire | Composant HTTP | Bout en bout | Concurrence | Recette |
|---|:---:|:---:|:---:|:---:|:---:|
| Inscription / connexion | ✅ | ✅ | — | — | ✅ R-01, R-02 |
| Émission et vérification JWT | ✅ | — | — | — | — |
| Hiérarchie de rôles | ✅ | ✅ | — | — | ✅ R-11 |
| Middleware d'authentification | ✅ | ✅ | — | — | — |
| Réinitialisation de mot de passe | ✅ | ✅ | — | — | ✅ R-03 |
| Profil utilisateur | ✅ | ✅ | — | — | ✅ R-04 |
| Création / fin de live | ✅ | ✅ | ✅ | — | ✅ R-05, R-08 |
| Fan-out audio (Hub) | ✅ | — | ✅ | ✅ | ✅ R-06 |
| Auditeur lent / abandon de trame | ✅ | — | — | ✅ | ✅ R-07 |
| Upload de piste (URL présignée) | ✅ | ✅ | — | — | ✅ R-09 |
| Playlists et file d'attente | ✅ | ✅ | — | — | ✅ R-10 |
| Favoris | ✅ | ✅ | — | — | ✅ R-10 |
| Feature flags | ✅ | ✅ | — | — | ✅ R-12 |
| Configuration / démarrage | ✅ | — | — | — | — |
| Lecture en arrière-plan 🚧 | — | — | — | — | ✅ R-13 |
| Accessibilité 🚧 | — | — | — | — | ✅ R-14 |

---

## 4. État de la couverture

Mesure du **2026-09-03** sur la branche `refactor/tests-and-add-misssing-tests` (PR #35), commande
`go test -cover ./...` :

| Package | Couverture | Objectif ≥ 80 % |
|---|---:|:---:|
| `internal/features` | 100,0 % | ✅ |
| `internal/observability` | 100,0 % | ✅ |
| `internal/playlist` | 100,0 % | ✅ |
| `internal/user/usecase` | 100,0 % | ✅ |
| `pkg/config` | 100,0 % | ✅ |
| `pkg/mailer` | 100,0 % | ✅ |
| `pkg/middleware` | 100,0 % | ✅ |
| `internal/track` | 98,6 % | ✅ |
| **`internal/streaming`** | **96,6 %** | ✅ |
| `internal/user` | 96,2 % | ✅ |
| `internal/storage` | 93,1 % | ✅ |
| `internal/passwordreset` | 92,4 % | ✅ |
| **`internal/auth`** | **86,4 %** | ✅ |
| `internal/api` | 80,7 % | ✅ |
| `cmd/server` | 14,9 % | ⚠️ racine de composition — voir ci-dessous |

**Volumétrie :** 293 fonctions de test réparties sur 26 fichiers.

**Sur `cmd/server`.** Ce package est la racine de composition : il ne contient pas de logique métier mais
du câblage (pool de connexions, instanciation des services, montage du routeur, arrêt gracieux). Sa
couverture faible est **assumée et argumentée** : ce qui s'y teste utilement — le parsing de
configuration, le chargement des migrations, l'arrêt gracieux — l'est déjà ; le reste ne se valide que
par le démarrage réel du binaire, couvert par le job de build et la recette. Le seuil de 80 % s'applique
au total du projet hors code généré, mesure sous laquelle le projet est conforme.

**Exclusion du code généré.** `internal/api/api.gen.go` est produit par `oapi-codegen` à partir du
contrat OpenAPI. Il n'est pas écrit à la main, il est régénéré et vérifié à chaque exécution de la CI :
le tester reviendrait à tester le générateur. Il est donc retiré du profil avant le calcul du seuil, ce
qui est explicité dans le workflow.

---

## 5. Ce que les tests prouvent réellement

Cette section est destinée à la soutenance : elle relie chaque test à la **propriété système** qu'il
démontre, plutôt qu'à une simple ligne de code.

### 5.1 Le moteur de streaming (`internal/streaming`)

| Test | Propriété démontrée |
|---|---|
| `TestHub_BroadcastReachesEveryListener` | Aucune trame n'est perdue en régime nominal |
| `TestHub_BroadcastAcrossMultipleShards` | Le découpage en lots de 100 auditeurs n'altère pas la diffusion — la parallélisation est correcte au-delà d'un shard |
| **`TestHub_BroadcastDropsChunksForSlowListener`** | **Propriété centrale du sujet :** un auditeur saturé voit sa trame abandonnée sans que le diffuseur ni les autres auditeurs soient bloqués. C'est la preuve du back-pressure. |
| `TestHub_BroadcastStopsOnCancelledContext` | L'annulation de contexte est respectée : pas de travail orphelin |
| `TestHub_CloseFiresDone` | L'arrêt d'un live termine les boucles d'ingestion — pas de goroutine résiduelle |
| `TestHub_UnsubscribeClosesChannel` | Le désabonnement libère effectivement les ressources |
| **`TestHub_ConcurrentSubscribeBroadcastUnsubscribe`** | Exécuté sous `-race` : abonnement, diffusion et désabonnement simultanés ne provoquent ni compétition de données ni envoi sur canal fermé |
| `TestIngestToListen_ForwardsBinaryChunksAndIgnoresText` | **Test bout en bout sur vraies connexions WebSocket** : une trame binaire poussée à l'ingestion ressort à l'écoute, et les trames texte sont ignorées |
| `TestEndStream_DisconnectsListener` | La fin d'un live déconnecte proprement les auditeurs |
| `TestListen_UnsubscribesOnDisconnect` | Une déconnexion d'auditeur nettoie son abonnement |

### 5.2 Sécurité et autorisation

| Test | Propriété démontrée |
|---|---|
| `TestVerify_RejectsNonHMACSigningMethod` | L'attaque `alg: none` / substitution d'algorithme est bloquée |
| `TestVerify_WrongSecret`, `TestVerify_Malformed`, `TestVerify_Expired` | Un jeton forgé, altéré ou expiré est rejeté |
| `TestAuthenticate_PermissiveOnBadOrMissingToken` | Le middleware permissif se comporte comme spécifié : il laisse passer sans claims, il ne rejette pas |
| `TestRequireRole` | La hiérarchie ordinale des rôles est respectée |
| `TestIngest_RejectsAnonymous`, `_RejectsPlainUser`, `_RejectsForeignBroadcaster` | **Défense en profondeur :** l'endpoint d'ingestion refuse l'anonyme, l'utilisateur simple, **et le diffuseur qui n'est pas propriétaire du live** |
| `TestIngest_RejectsEndedStream`, `TestListen_RejectsEndedStream` | On ne peut ni alimenter ni écouter un live terminé |
| `TestListen_AllowsAnonymous` | L'écoute publique reste ouverte aux visiteurs — le comportement voulu, testé explicitement |
| `TestIngest_RejectsInvalidQueryToken` | Le repli d'authentification par paramètre de requête valide bien le jeton |
| `TestIngest_MissingHubIs503`, `TestIngest_RepoErrorIs500` | Les codes d'erreur distinguent l'indisponibilité de la faute technique |
| `TestRegisterHandler_IgnoresClientSuppliedRole`, `TestRegister_AlwaysCreatesPlainUser` | Un champ `role` dans le corps d'inscription n'atteint jamais le compte créé : l'inscription produit toujours un `user` |
| `TestUpdateUserByIDHandler_IgnoresClientSuppliedRole` | `PATCH /users/me` ne modifie que `username` et `email` — jamais le rôle |
| `TestRefreshTokenHandler_UsesStoredRoleNotTokenRole` | Le rafraîchissement relit le rôle en base : un jeton émis quand le compte était admin ne renouvelle plus des privilèges d'admin après rétrogradation |
| `TestRefreshTokenHandler_DeletedUser` | Un jeton de rafraîchissement qui survit à son compte n'est plus une identité valide |
| `TestAdminEndpoint_RejectsTokenSignedWithAnotherSecret` | Forger un rôle suppose de forger la signature, donc de connaître `JWT_SECRET` |
| `TestLoad_RejectsShortJWTSecret`, `TestLoad_ProductionRequiresProxy` | La configuration refuse de démarrer avec une clé HMAC trop courte, ou en production hors du reverse proxy TLS |

---

## 6. Cahier de recette

**Modalités.** Recette exécutée avant chaque tag de version, sur un appareil Android physique connecté au
backend de production, avec trois comptes de test (`user`, `broadcaster`, `admin`). Résultat consigné :
*conforme* / *non conforme* + numéro d'anomalie.

**Critères d'entrée en recette :** CI verte, couverture ≥ 80 %, aucune anomalie bloquante ouverte,
migrations appliquées sur l'environnement de recette.
**Critères de sortie :** 100 % des scénarios de sévérité *bloquante* conformes, aucune anomalie
bloquante ou majeure ouverte.

| # | Scénario | Prérequis | Étapes | Résultat attendu | Sévérité |
|---|---|---|---|---|---|
| **R-01** | Inscription | Aucun compte avec cet e-mail | Ouvrir l'app → « Créer un compte » → saisir nom, e-mail, mot de passe → valider | Compte créé, connexion automatique, arrivée sur l'accueil, rôle `user` | Bloquante |
| **R-02** | Connexion et persistance de session | Compte existant | Se connecter → fermer complètement l'app → rouvrir | Session restaurée sans nouvelle saisie ; jeton expiré rafraîchi de façon transparente | Bloquante |
| **R-02b** | Connexion refusée | Compte existant | Saisir un mot de passe erroné | Message d'erreur explicite, aucune indication sur l'existence du compte, aucun accès accordé | Bloquante |
| **R-03** | Mot de passe oublié | Compte existant | « Mot de passe oublié » → saisir l'e-mail → ouvrir le lien reçu → définir un nouveau mot de passe | Message identique que le compte existe ou non ; le lien ouvre l'app ; nouveau mot de passe fonctionnel ; **le même lien réutilisé est refusé** | Bloquante |
| **R-04** | Modification du profil | Connecté | Profil → modifier le nom d'utilisateur → enregistrer | Modification persistée, visible après redémarrage | Majeure |
| **R-05** | Démarrer un live | Compte `broadcaster` | Écran diffuseur → titre → « Démarrer » → autoriser le micro | Live créé et visible dans la liste publique en moins de 5 s ; indicateur de diffusion actif | Bloquante |
| **R-05b** | Live unique par diffuseur | Un live déjà actif | Tenter d'en démarrer un second | Refus explicite, le premier live n'est pas interrompu | Majeure |
| **R-05c** | Micro refusé | Compte `broadcaster` | Refuser la permission micro | Message d'aide orientant vers les réglages ; aucun plantage | Majeure |
| **R-06** | Écouter un live | Un live actif | Depuis un second appareil : ouvrir le live | Audio audible en moins de 3 s, latence perçue inférieure à 2 s, sans coupure sur 5 minutes | Bloquante |
| **R-07** | Réseau dégradé | En cours d'écoute | Activer le mode avion 5 s puis le désactiver | État d'erreur explicite puis reprise ou possibilité de relance manuelle ; aucun plantage ni blocage de l'interface | Majeure |
| **R-08** | Fin de live | Live actif avec un auditeur | Le diffuseur arrête le live | Le diffuseur revient à l'écran précédent ; l'auditeur est informé de la fin ; le live disparaît de la liste des lives | Bloquante |
| **R-09** | Publier une piste | Compte `broadcaster` | Upload → choisir un fichier → renseigner titre et artiste → publier | Barre de progression, piste visible dans la liste publique, lecture possible | Bloquante |
| **R-10** | Playlists et favoris | Compte `user`, au moins 2 pistes | Créer une playlist → y ajouter 2 pistes → mettre une piste en favori → retirer un favori | Playlist listée avec ses pistes dans l'ordre d'ajout ; favoris cohérents après redémarrage | Majeure |
| **R-11** | Cloisonnement des rôles | Compte `user` | Tenter d'accéder à l'écran diffuseur puis d'appeler `POST /streams` avec ce jeton | L'entrée n'est pas proposée dans l'interface ; l'appel direct à l'API renvoie 403 | Bloquante |
| **R-12** | Feature flag 🚧 | Compte `admin` | Désactiver `live_streaming` → rafraîchir l'app | La fonctionnalité disparaît de l'interface et l'API répond 503, sans redéploiement | Majeure |
| **R-13** | Lecture en arrière-plan et interruptions | En cours d'écoute | Verrouiller l'écran ; passer sur une autre app ; recevoir un appel ; débrancher le casque | La lecture continue écran verrouillé, les commandes système sont disponibles ; l'appel met en pause puis la lecture reprend ; le débranchement du casque met en pause 🚧 | Majeure |
| **R-14** | Accessibilité 🚧 | Lecteur d'écran activé (TalkBack) | Parcourir l'accueil, lancer et arrêter une lecture au lecteur d'écran ; passer la taille de police à 200 % | Chaque contrôle est annoncé par un libellé compréhensible ; les changements d'état de lecture sont annoncés ; aucun texte tronqué ni superposé | Majeure |
| **R-15** | Déploiement | — | Fusionner une PR sur `main` | Image publiée sur GHCR, conteneur relancé, `GET /health` renvoie 200 en moins de 2 minutes | Bloquante |

---

## 7. Planification itérative

Les tests ne constituent pas une phase, ils sont **répartis sur chaque itération** :

| Moment | Activité de test | Automatisé |
|---|---|---|
| Rédaction de la user story | Définition des critères d'acceptation, qui deviennent les cas de test | Non |
| Pendant le développement | Tests unitaires écrits avec le code, dans la même PR | Oui, en local |
| À l'ouverture de la PR | Lint, tests, race, seuil de couverture, build | **Oui — bloquant** |
| Revue de PR | Vérification que les cas d'erreur, et pas seulement le cas nominal, sont testés | Non |
| Avant un tag de version | Cahier de recette § 6 | Non |
| Après déploiement | `GET /health` ; smoke test automatisé 🚧 | Partiel |
| En continu, en production | Tableaux de bord et alertes comme tests permanents en conditions réelles 🚧 | 🚧 |

**Gestion des anomalies.** Une anomalie détectée donne lieu à : (1) une issue décrivant la reproduction,
(2) **un test qui échoue et qui reproduit le défaut**, (3) le correctif, (4) le test qui passe. Ce test
de non-régression reste dans la suite : le défaut ne peut plus revenir silencieusement. C'est ainsi que
la suite de tests assure le suivi de la performance des versions du code au fil des livraisons.

---

## 8. Reste à faire

| Élément | Ce que cela apporte | Priorité |
|---|---|---|
| Tests d'intégration sur PostgreSQL réel (tag `integration`) + job CI dédié | Valider les repositories et les migrations sur une vraie base | Haute |
| Tests mobiles sur `PlayerNotifier` / `BroadcasterNotifier`, incluant la coupure WebSocket | Prouver la robustesse du state management aux changements d'état réseau | Haute |
| Test d'accessibilité automatisé (contrastes, cibles tactiles) | Preuve automatisée de la conformité WCAG AA | Haute |
| Seuil de couverture côté mobile (lcov) | Symétrie du garde-fou backend/mobile | Moyenne |
| Benchmarks `BenchmarkHubBroadcast` à 10 / 100 / 1000 auditeurs + profils `pprof` | Justification des coûts | Moyenne |
| `go.uber.org/goleak` pour prouver l'absence de fuite de goroutine | Preuve de la libération des ressources à la déconnexion | Moyenne |
| Test de charge N connexions d'écoute simultanées | Scalabilité | Moyenne |
| Vérification automatisée du profil TLS après déploiement (redirection 80→443, HSTS, `/internal/*` en 404) | Prouver en continu la posture du reverse proxy, aujourd'hui vérifiée à la main (voir README, « TLS et reverse proxy ») | Moyenne |

---

## English summary

This test plan defines the quality policy (100 % green tests, ≥ 80 % coverage enforced as a **blocking**
CI gate excluding generated code, race detector always on, contract/code drift detection through OpenAPI
regeneration), the test taxonomy, a feature × test-type matrix, the measured per-package coverage
(293 test functions; streaming 96.6 %, auth 86.4 %, middleware and config 100 %), a mapping from each
critical test to the **system property it proves** — most notably the non-blocking drop policy and the
end-to-end WebSocket ingest-to-listen path — and a numbered acceptance-test book (R-01 to R-15) with
entry and exit criteria. Tests are written in the same pull request as the code they cover, and every
bug fix ships with a failing-then-passing regression test.

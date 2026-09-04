# Plan de formation des utilisateurs

**Version :** 1.0 · **Date :** 2026-09-03

> Objectif : un plan de formation adapté à la diversité du public, y compris les personnes présentant un
> handicap, avec des stratégies d'enseignement différenciées par groupe d'utilisateurs.

---

## 1. Principes

Trois principes gouvernent ce plan :

1. **La formation doit être proportionnée au besoin réel.** Un auditeur qui veut écouter une émission ne
   doit avoir *aucune* formation à suivre : si l'application exige une explication pour être utilisée, le
   problème est dans l'application, pas dans l'utilisateur. La formation est donc concentrée sur les
   profils dont les tâches sont réellement complexes — diffuseur et administrateur.
2. **Un même contenu, plusieurs canaux.** Chaque notion est disponible sous au moins deux formes parmi :
   in-app, texte, vidéo, audio. Cela répond à la fois à la diversité des situations de handicap et à la
   diversité des préférences d'apprentissage.
3. **Tout support est accessible par construction.** Une vidéo sans sous-titres, une capture d'écran sans
   texte alternatif ou un tutoriel qui désigne un bouton par sa seule couleur exclut une partie du
   public. Ce sont des critères de conformité du support, pas des options.

---

## 2. Publics et besoins

| Public | Profil type | Compétences numériques | Besoin de formation | Durée cible |
|---|---|---|---|---|
| **Auditeur grand public** | Toute personne disposant d'un smartphone | Variable, souvent faible | **Aucune** — l'interface doit suffire ; un guide de secours doit exister | 0 min |
| **Diffuseur amateur** | Créateur, animateur associatif, responsable d'un lieu de culte | Moyenne | Démarrer un direct, comprendre l'incidence de sa connexion sur la qualité, publier une piste | 15 min |
| **Diffuseur régulier** | Usage hebdomadaire ou plus | Moyenne à bonne | Bonnes pratiques audio, gestion des incidents en direct, organisation du contenu | 45 min |
| **Administrateur / exploitant** | Personne technique | Bonne | Interrupteurs de fonctionnalités, lecture des tableaux de bord, procédure d'incident, gestion des comptes | 2 h |
| **Utilisateur en situation de handicap** | Transversal à tous les profils | Variable ; souvent expert de ses propres outils d'assistance | Les **spécificités** de l'application avec son outil d'assistance, jamais l'usage de l'outil lui-même | Selon profil |
| **Nouveau développeur** | Contributeur au projet | Élevée | Architecture, conventions, chaîne de livraison | 1 j |

---

## 3. Dispositif par public

### 3.1 Auditeur — objectif : zéro formation

| Support | Forme | Accessibilité |
|---|---|---|
| **Interface auto-explicative** | Libellés explicites plutôt qu'icônes seules, messages d'erreur qui indiquent quoi faire et pas seulement ce qui a échoué | Libellés vocalisés 🚧 |
| **Onboarding de 3 écrans** 🚧 | À la première ouverture : écouter, mettre en favori, créer une playlist. Passable à tout moment | Texte lisible au lecteur d'écran, aucune information portée par la seule couleur |
| **Guide d'utilisation** | [`guide-utilisateur.fr.md`](guide-utilisateur.fr.md) — chapitre 3 | Structure de titres, étapes numérotées, tableau de résolution des problèmes |
| **Section « Résolution des problèmes »** | Tableau symptôme → cause → action | Format tabulaire annoncé correctement par les lecteurs d'écran |

### 3.2 Diffuseur — objectif : réussir sa première diffusion

**Parcours en trois temps (≈ 15 minutes)**

| Étape | Contenu | Support | Durée |
|---|---|---|---|
| 1. Comprendre | Ce qu'est un direct, la différence avec une piste enregistrée, la règle « un seul direct à la fois » | Guide § 4 + fiche récapitulative d'une page | 3 min |
| 2. Faire | Diffusion d'essai avec un second appareil, en suivant une liste de vérification | Liste de vérification imprimable ci-dessous | 7 min |
| 3. Approfondir | Qualité audio, incidence de la connexion, gestion d'un incident en direct | Tutoriel vidéo sous-titré 🚧 + version audio décrite 🚧 | 5 min |

**Liste de vérification avant diffusion** — support de formation réutilisable à chaque émission :

- [ ] Batterie chargée ou téléphone branché
- [ ] Wi-Fi ou bonne couverture mobile
- [ ] Casque-micro branché (évite le retour audio)
- [ ] Notifications en mode silencieux
- [ ] Titre explicite renseigné
- [ ] Test d'écoute effectué depuis un second appareil
- [ ] Durée prévue annoncée aux auditeurs

**Erreurs fréquentes traitées explicitement en formation** — enseigner les pièges est plus efficace
qu'enseigner le chemin nominal :

| Erreur | Conséquence | Ce qu'il faut faire |
|---|---|---|
| Refuser la permission micro | La diffusion ne démarre pas | Réglages Android → Applications → Lyo → Autorisations |
| Diffuser sur un réseau mobile faible | Qualité dégradée **pour tous** les auditeurs | Vérifier la connexion avant, pas pendant |
| Fermer l'application pendant le direct | Interruption de la diffusion | Garder l'application au premier plan |
| Oublier d'arrêter le direct | Le direct reste ouvert et bloque le suivant | Toujours utiliser « Arrêter » |
| Titre vague (« test », « live ») | Les auditeurs ne trouvent pas l'émission | Titre explicite et daté si récurrent |

### 3.3 Administrateur — objectif : savoir réagir à un incident

| Module | Contenu | Support | Durée |
|---|---|---|---|
| A1 — Rôles et habilitations | Hiérarchie des rôles, promotion d'un diffuseur, conséquences d'un changement de rôle | Guide § 5 + [matrice des droits](cahier-des-charges.fr.md#51-matrice-des-droits-par-rôle) | 20 min |
| A2 — Interrupteurs de fonctionnalités | Rôle de coupe-circuit, effet immédiat, cas d'usage | Guide § 5.1 + exercice pratique | 20 min |
| A3 — Lecture des tableaux de bord | **Distinguer erreur technique et dégradation d'expérience** ; interpréter le taux de trames abandonnées | Tableau de bord Grafana 🚧 + fiche d'interprétation | 40 min |
| A4 — Procédure d'incident | Détection → qualification → décision (correctif ou coupure par flag) → communication → post-mortem | Fiche de procédure ci-dessous | 40 min |

**Fiche de procédure d'incident** — support de formation et aide-mémoire d'exploitation :

1. **Constater** : quelle alerte, quel indicateur, depuis quand ?
2. **Qualifier** : est-ce une **erreur technique** (le code a échoué) ou une **dégradation
   d'expérience** (le système fonctionne comme conçu, mais les utilisateurs en souffrent) ? *Cette
   question détermine tout le reste.*
3. **Contenir** : la fonctionnalité concernée peut-elle être coupée par interrupteur ? Si oui, le faire
   **avant** de chercher la cause — le rétablissement prime sur le diagnostic.
4. **Décider** : correctif à déployer, retour arrière sur la version précédente, ou attente ?
5. **Communiquer** : informer les utilisateurs concernés.
6. **Capitaliser** : ouvrir une issue, écrire un **test de non-régression**, mettre à jour la
   documentation si la procédure s'est révélée insuffisante.

### 3.4 Utilisateurs en situation de handicap

**Posture.** Une personne utilisant un lecteur d'écran depuis des années le maîtrise mieux que nous. La
formation ne porte donc **jamais** sur l'outil d'assistance : elle porte sur les particularités de
l'application, sur ce qui fonctionne, et — honnêtement — sur ce qui ne fonctionne pas encore.

| Situation | Adaptation du support | Adaptation de l'application |
|---|---|---|
| **Déficience visuelle** | Guide texte structuré, lisible au lecteur d'écran ; alternative textuelle de chaque schéma ; version audio des tutoriels ; aucune consigne du type « le bouton bleu en haut à droite » | Libellés vocalisés, annonces d'état, contrastes 🚧 |
| **Déficience auditive** | Tutoriels vidéo **sous-titrés** et accompagnés d'une transcription texte ; toute information sonore doublée visuellement | Sans objet pour la formation ; à noter que le contenu audio diffusé n'est pas transcrit — limite du produit à assumer |
| **Déficience motrice** | Supports ne supposant aucun geste précis ni chronométré | Cibles tactiles ≥ 48 × 48 dp, aucune action nécessitant un geste complexe ou maintenu 🚧 |
| **Déficience cognitive** | Étapes courtes et numérotées, une action par étape, vocabulaire constant (un même élément est toujours nommé pareil), pictogrammes accompagnés de texte | Parcours linéaires, absence de délai contraint |
| **Faible littératie numérique** | Vocabulaire du quotidien, glossaire systématique, formation par la pratique guidée plutôt que par la lecture | Messages d'erreur formulés en langage courant |

**Engagements de conformité des supports**

- Toute vidéo est **sous-titrée** et accompagnée d'une **transcription texte**.
- Toute capture d'écran porte un **texte alternatif** décrivant son contenu, pas seulement son sujet.
- Tout schéma est doublé d'une **description textuelle intégrale** — c'est déjà le cas de l'ensemble des
  diagrammes d'architecture de ce dépôt.
- Aucune instruction ne repose sur la couleur, la forme ou la position seules.
- Les documents sont fournis en **Markdown**, format texte lisible par tout outil d'assistance, plutôt
  qu'en PDF non balisé.

### 3.5 Nouveau développeur

| Étape | Support | Durée |
|---|---|---|
| Vue d'ensemble | [`README.fr.md`](../README.fr.md) et [`architecture/`](architecture/) | 1 h |
| Comprendre le *pourquoi* des choix | [`adr/`](adr/) — 10 décisions, dont une explicitement remplacée | 1 h |
| Installer l'environnement | README § « Démarrage local » | 1 h |
| Conventions et qualité | [`CONTRIBUTING.md`](../CONTRIBUTING.md), [`plan-de-tests.md`](plan-de-tests.md) | 1 h |
| Première contribution | Une issue étiquetée « bonne première contribution » | 4 h |

---

## 4. Modalités et évaluation

| Modalité | Public | Justification |
|---|---|---|
| **Autoformation par la documentation** | Tous | Disponible en permanence, sans contrainte d'horaire, accessible aux outils d'assistance |
| **Onboarding in-app** 🚧 | Auditeur | Formation au moment du besoin, sans support externe |
| **Tutoriel vidéo sous-titré** 🚧 | Diffuseur | Adapté à l'apprentissage par imitation d'un geste |
| **Pratique guidée** (diffusion d'essai) | Diffuseur | La compétence visée est un geste, elle s'acquiert en le faisant |
| **Atelier** | Administrateur | La procédure d'incident se travaille par mise en situation |

**Évaluation de la formation**

| Indicateur | Cible | Mesure |
|---|---|---|
| Taux de réussite de la première diffusion | > 90 % | Part des comptes diffuseurs ayant un direct de plus de 2 minutes |
| Recours au guide de dépannage | En baisse | Consultations de la section correspondante |
| Erreurs récurrentes | En baisse | Motifs de refus (permission micro, direct déjà actif) |
| Délai de résolution d'incident | < 30 min | Suivi des incidents |
| Retours d'accessibilité | Traités sous 15 j | Issues étiquetées `accessibility` |

**Boucle d'amélioration.** Chaque question récurrente d'utilisateur est traitée dans cet ordre :
(1) *l'interface peut-elle rendre la question inutile ?* — si oui, c'est un correctif produit, pas un
ajout de documentation ; (2) sinon, la documentation est complétée ; (3) si la question revient malgré
tout, le support de formation est revu. **Une documentation qui grossit indéfiniment est le symptôme
d'une interface qui devrait être corrigée.**

---

## English summary

The training plan is governed by three principles: training must be proportionate (a listener should
need none — if the app requires explanation, the app is the problem), each topic exists in at least two
delivery formats, and every support material is accessible by construction rather than by retrofit. It
defines six audiences with their real training needs and durations, a three-stage broadcaster path built
around a reusable pre-broadcast checklist and an explicit "common mistakes" table, a four-module
administrator curriculum centred on the incident procedure — whose pivotal step is telling a technical
failure apart from an experience degradation — and per-disability adaptations that never teach the
assistive tool itself, only the application's specifics and its current honest limitations. Effectiveness
is measured, and every recurring user question is first considered a product defect before it is
considered a documentation gap.

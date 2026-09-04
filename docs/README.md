# Documentation — Lyo / StreamPulse

**FR / EN.** La documentation technique est disponible en français et en anglais : les documents
principaux ont une version intégrale dans chaque langue, les autres comportent un résumé anglais en fin
de document. · *Technical documentation is available in French and English: the main documents have a
full version in each language, the others carry an English summary at the end.*

**Convention de statut · Status convention :** ✅ livré / delivered · 🚧 planifié / planned.

---

## Cadrage projet · Project framing

| Document | FR | EN | Contenu |
|---|:--:|:--:|---|
| **Cahier des charges** · Requirements specification | [`cahier-des-charges.fr.md`](cahier-des-charges.fr.md) | [`cahier-des-charges.en.md`](cahier-des-charges.en.md) | Contexte, faisabilité technique/organisationnelle/**financière et ROI**, périmètre, spécifications, accessibilité, numérique responsable, **analyse des risques**, feuille de route, **KPI**, conformité, **bilan réflexif** |
| **Veille technologique et concurrentielle** | [`veille-technologique.fr.md`](veille-technologique.fr.md) | *(résumé EN inclus)* | Plan de veille, sources, outils, méthode de tri, analyse concurrentielle, comparatif WebSocket/HLS/WebRTC/RTMP, **apport concret de la veille au code** |
| **User stories** | [`user-stories.md`](user-stories.md) | *(résumé EN inclus)* | 20 stories, 5 épopées, critères d'acceptation, traçabilité |

## Architecture

| Document | Contenu | Notation |
|---|---|---|
| [`architecture/README.md`](architecture/README.md) | Vue d'ensemble, découpage en couches | — |
| [`architecture/modele-de-donnees.md`](architecture/modele-de-donnees.md) | MCD, diagramme de classes, dictionnaire de données, choix de modélisation | UML / MCD |
| [`architecture/diagrammes-de-sequence.md`](architecture/diagrammes-de-sequence.md) | Authentification, **streaming de bout en bout**, upload S3, réinitialisation de mot de passe | UML séquence |
| [`architecture/processus-bpmn.md`](architecture/processus-bpmn.md) | Diffuser un live, publier une piste, parcours auditeur, administrer | BPMN |
| [`architecture/securite.md`](architecture/securite.md) | Zones de confiance, chaîne d'autorisation, secrets, **mapping OWASP Top 10**, ANSSI | Flux + tableaux |
| [`architecture/deploiement.md`](architecture/deploiement.md) | Topologie VPS, conteneurs, chaîne CI/CD, stratégie de retour arrière | UML déploiement |

> **Accessibilité de la documentation.** Tous les diagrammes sont en Mermaid — donc en texte, versionnés
> et diffables — et **chacun est accompagné d'une alternative textuelle intégrale**, ce qui rend
> l'ensemble exploitable au lecteur d'écran. Aucune information n'est portée par la seule couleur.

## Qualité et exploitation

| Document | Contenu |
|---|---|
| [`plan-de-tests.md`](plan-de-tests.md) | Stratégie, typologie, matrice fonctionnalité × test, **couverture mesurée**, ce que chaque test prouve, **cahier de recette R-01 à R-15**, planification itérative |
| [`supervision.md`](supervision.md) | Exploitation de la supervision : démarrage de la stack, tableaux de bord, métriques clés, règles d'alerte, réponse à incident, vérification de bout en bout |
| [`adr/`](adr/) | 11 décisions d'architecture, dont une explicitement remplacée |

## Guides

| Document | FR | EN | Public |
|---|:--:|:--:|---|
| **Guide d'utilisation** · User guide | [`guide-utilisateur.fr.md`](guide-utilisateur.fr.md) | [`guide-utilisateur.en.md`](guide-utilisateur.en.md) | Auditeur, diffuseur, administrateur |
| **Plan de formation** | [`plan-formation.fr.md`](plan-formation.fr.md) | *(résumé EN inclus)* | Formateur, exploitant |
| **Guide développeur** · Developer guide | [`../README.fr.md`](../README.fr.md) | [`../README.md`](../README.md) | Contributeur |
| **Contribuer** · Contributing | — | [`../CONTRIBUTING.md`](../CONTRIBUTING.md) | Contributeur |
| **Sécurité** · Security policy | — | [`../SECURITY.md`](../SECURITY.md) | Chercheur en sécurité |

---

## Par où commencer

| Vous êtes… | Lisez dans cet ordre |
|---|---|
| **Utilisateur** | [Guide d'utilisation](guide-utilisateur.fr.md) |
| **Évaluateur / jury** | [Cahier des charges](cahier-des-charges.fr.md) → [Architecture](architecture/README.md) → [Plan de tests](plan-de-tests.md) → [ADR](adr/) |
| **Nouveau développeur** | [README](../README.fr.md) → [Architecture](architecture/README.md) → [ADR](adr/) → [CONTRIBUTING](../CONTRIBUTING.md) |
| **Exploitant** | [Déploiement](architecture/deploiement.md) → [Sécurité](architecture/securite.md) → [Plan de formation § 3.3](plan-formation.fr.md) |

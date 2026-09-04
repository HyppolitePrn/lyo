# Processus métier (BPMN)

Les processus ci-dessous sont modélisés en BPMN simplifié (couloirs, événements, tâches, passerelles).
Chaque diagramme est suivi de son équivalent textuel.

## 1. Diffuser un live

```mermaid
flowchart TB
    subgraph Diffuseur
        A(("Début :<br/>intention de diffuser")) --> B["Se connecter à l'application"]
        B --> C{"Rôle ≥ broadcaster ?"}
        C -- Non --> C1["Afficher : accès réservé aux diffuseurs"] --> Z1((("Fin — refusé")))
        C -- Oui --> D["Saisir titre et description"]
        D --> E["Autoriser l'accès au micro"]
        E --> E1{"Permission accordée ?"}
        E1 -- Non --> E2["Message d'aide : activer le micro dans les réglages"] --> Z1
        E1 -- Oui --> F["Démarrer la capture AAC"]
        F --> G["Diffuser (WebSocket ingest)"]
        G --> H{"Arrêter ?"}
        H -- Non --> G
        H -- Oui --> I["Clôturer le live"]
    end
    subgraph Systeme["Système"]
        D --> S1{"Un live déjà actif<br/>pour ce diffuseur ?"}
        S1 -- Oui --> S2["Refus 409 — un seul live à la fois"] --> Z1
        S1 -- Non --> S3["Créer le stream (status = live)<br/>+ instancier le Hub"]
        S3 --> F
        G --> S4["Fan-out vers N auditeurs<br/>drop non bloquant si buffer plein"]
        I --> S5["status = ended, ended_at = now()<br/>Hub.Close() → libération des goroutines"]
        S5 --> Z2((("Fin — live archivé")))
    end
```

**Alternative textuelle.** Le processus débute par l'intention de diffuser. Le système vérifie d'abord
l'habilitation : un utilisateur sans le rôle `broadcaster` est refusé. Le diffuseur saisit ensuite le
titre et la description du live ; le système contrôle qu'aucun live n'est déjà actif pour ce compte et
refuse sinon. La permission d'accès au micro est demandée à l'OS ; en cas de refus, un message d'aide
oriente vers les réglages. La capture audio démarre alors et la diffusion s'effectue en continu par
WebSocket, le système opérant le fan-out vers tous les auditeurs avec abandon non bloquant des trames
pour les auditeurs saturés. À la demande d'arrêt, le live est marqué terminé, horodaté, et les
ressources du Hub sont libérées.

## 2. Publier une piste enregistrée

```mermaid
flowchart TB
    subgraph Diffuseur
        A(("Début")) --> B["Sélectionner un fichier audio"]
        B --> C["Saisir titre, artiste, durée"]
        C --> D["Envoyer le fichier vers le stockage objet"]
        D --> E{"Transfert réussi ?"}
        E -- Non --> E1["Proposer une nouvelle tentative"] --> D
        E -- Oui --> F["Confirmer la publication"]
    end
    subgraph Systeme["Système"]
        C --> S1{"Flag track_uploads actif<br/>et rôle ≥ broadcaster ?"}
        S1 -- Non --> S2["503 fonctionnalité désactivée / 403 interdit"] --> Z1((("Fin — refusé")))
        S1 -- Oui --> S3["Générer une URL PUT présignée<br/>à durée de vie courte"]
        S3 --> D
        F --> S4["Enregistrer les métadonnées de la piste"]
        S4 --> S5["La piste devient listable et ajoutable en playlist"]
        S5 --> Z2((("Fin — piste publiée")))
    end
```

**Alternative textuelle.** Le diffuseur sélectionne un fichier et renseigne les métadonnées. Le système
vérifie le feature flag et le rôle, puis délivre une URL d'upload présignée à durée de vie courte. Le
téléphone envoie le fichier directement au stockage objet, avec possibilité de nouvelle tentative en cas
d'échec réseau. Une fois le transfert confirmé, les métadonnées sont enregistrées et la piste devient
listable publiquement et ajoutable à une playlist.

## 3. Écouter et organiser (parcours auditeur)

```mermaid
flowchart LR
    A(("Ouvrir l'app")) --> B{"Authentifié ?"}
    B -- Non --> C["Parcourir les lives publics<br/>(rôle anonymous)"]
    B -- Oui --> D["Accueil personnalisé :<br/>lives, pistes, favoris, playlists"]
    C --> E["Écouter un live"]
    D --> E
    D --> F["Écouter une piste enregistrée"]
    E --> G{"Interruption système<br/>(appel, casque débranché) ?"}
    G -- Oui --> H["Mettre en pause,<br/>reprendre à la fin de l'interruption"] --> E
    G -- Non --> I["Lecture continue,<br/>y compris en arrière-plan"]
    F --> J["Ajouter aux favoris"]
    F --> K["Ajouter à une playlist<br/>(position = fin de file d'attente)"]
    I --> Z((("Fin de session")))
```

**Alternative textuelle.** À l'ouverture, un visiteur non authentifié peut parcourir et écouter les lives
publics avec le rôle `anonymous` ; un utilisateur authentifié obtient en plus un accueil personnalisé
avec ses favoris et playlists. Pendant l'écoute, les interruptions système (appel entrant, débranchement
du casque) déclenchent une pause puis une reprise. La lecture se poursuit en arrière-plan. Une piste
peut être mise en favori ou ajoutée en fin de file d'attente d'une playlist.

## 4. Administrer (rôle admin)

```mermaid
flowchart TB
    A(("Besoin d'exploitation")) --> B{"Nature de l'action"}
    B -- "Activer / désactiver une fonctionnalité" --> C["Basculer un feature flag"]
    C --> C1["Effet immédiat, sans redéploiement,<br/>pour tous les clients"]
    B -- "Surveiller la plateforme" --> D["Consulter les tableaux de bord<br/>et les alertes"]
    D --> D1{"Anomalie détectée ?"}
    D1 -- Oui --> D2["Décision : rollback de version<br/>ou coupure de la fonctionnalité par flag"]
    D2 --> C
    D1 -- Non --> Z((("Fin")))
    B -- "Gérer un compte" --> E["Modifier un rôle / suspendre un compte"]
    E --> Z
    C1 --> Z
```

**Alternative textuelle.** L'administrateur dispose de trois familles d'actions. Il peut basculer un
feature flag, avec effet immédiat sur tous les clients et sans redéploiement — c'est aussi le mécanisme
de coupure d'urgence d'une fonctionnalité défaillante. Il peut surveiller la plateforme via les tableaux
de bord et les alertes, et déclencher soit un retour à la version précédente, soit une coupure
fonctionnelle par flag. Il peut enfin gérer les comptes (changement de rôle, suspension).

> **État d'implémentation.** Le basculement de flag et la gestion de comptes sont décrits au contrat
> OpenAPI (`/admin/features`, `/admin/features/{name}/toggle`) ; leur implémentation et l'écran
> d'administration mobile figurent à la feuille de route (voir `docs/cahier-des-charges.fr.md`, § Feuille
> de route). Ce document décrit le processus cible, pas un existant.

---

## English summary

Four BPMN-style process models with swimlanes: going live (role check, single-live constraint,
microphone permission, non-blocking fan-out, graceful teardown), publishing a recorded track (feature
flag gate, presigned upload, metadata registration), the listener journey (anonymous browsing, audio
interruption handling, favourites and playlist queueing), and the admin loop (feature-flag kill switch,
dashboard-driven incident decision, account management). The admin lane describes the target process;
its implementation status is tracked in the roadmap.

# Guide d'utilisation — Lyo

**Version :** 1.0 · **Date :** 2026-09-03 · **English version:** [`guide-utilisateur.en.md`](guide-utilisateur.en.md)

Ce guide s'adresse aux **utilisateurs de l'application**. Pour l'installation et le développement, voir
le [guide développeur](../README.fr.md).

> **Note de lecture.** Ce document est conçu pour être lisible à l'écran comme au lecteur d'écran :
> chaque étape est numérotée, chaque bouton est désigné par son libellé, et aucune instruction ne repose
> sur une couleur ou une position à l'écran seule. Les fonctionnalités signalées 🚧 ne sont pas encore
> disponibles dans la version actuelle.

---

## Sommaire

- [1. Qu'est-ce que Lyo ?](#1-quest-ce-que-lyo-)
- [2. Premiers pas](#2-premiers-pas)
- [3. Guide de l'auditeur](#3-guide-de-lauditeur)
- [4. Guide du diffuseur](#4-guide-du-diffuseur)
- [5. Guide de l'administrateur](#5-guide-de-ladministrateur)
- [6. Accessibilité](#6-accessibilité)
- [7. Vos données personnelles](#7-vos-données-personnelles)
- [8. Résolution des problèmes](#8-résolution-des-problèmes)
- [9. Glossaire](#9-glossaire)

---

## 1. Qu'est-ce que Lyo ?

Lyo est une application d'**audio en direct**. Elle permet :

- d'**écouter** des émissions diffusées en direct par d'autres utilisateurs, avec un délai inférieur à
  deux secondes — ce qui rend le direct réellement vivant ;
- d'**écouter des pistes enregistrées** publiées par les diffuseurs, et de les organiser en playlists ;
- de **diffuser vous-même en direct** depuis votre téléphone, sans matériel ni configuration, si votre
  compte dispose du rôle diffuseur.

**Vous n'avez pas besoin de compte pour écouter les directs publics.** Un compte devient nécessaire pour
les favoris, les playlists et la diffusion.

### Les quatre profils

| Profil | Ce que vous pouvez faire |
|---|---|
| **Visiteur** (sans compte) | Parcourir et écouter les directs publics et les pistes publiées |
| **Auditeur** (compte standard) | Tout ce qui précède, plus les favoris, les playlists et un profil |
| **Diffuseur** | Tout ce qui précède, plus démarrer des directs et publier des pistes |
| **Administrateur** | Tout ce qui précède, plus la gestion des fonctionnalités, des comptes et le suivi de la plateforme |

---

## 2. Premiers pas

### 2.1 Installer l'application

1. Rendez-vous sur la page des versions du projet (GitHub Releases).
2. Téléchargez le fichier `.apk` de la version la plus récente.
3. Sur votre téléphone Android, ouvrez le fichier téléchargé.
4. Si Android vous demande d'autoriser l'installation depuis cette source, acceptez, puis relancez
   l'installation.
5. Ouvrez Lyo.

### 2.2 Créer un compte

1. Sur l'écran d'accueil, choisissez **« Créer un compte »**.
2. Renseignez un **nom d'utilisateur** — il sera visible publiquement.
3. Renseignez votre **adresse e-mail** — elle sert à vous identifier et à récupérer votre compte.
4. Choisissez un **mot de passe**. Conseil : au moins 12 caractères, idéalement une phrase facile à
   retenir pour vous et difficile à deviner pour autrui.
5. Validez avec **« S'inscrire »**.

Vous êtes connecté immédiatement, sans seconde saisie.

**Si un message vous indique que l'e-mail ou le nom d'utilisateur est déjà pris :** ils doivent être
uniques. Choisissez-en un autre, ou connectez-vous si le compte est déjà le vôtre.

### 2.3 Se connecter

1. Choisissez **« Se connecter »**.
2. Saisissez votre e-mail et votre mot de passe.
3. Validez.

Votre session est **conservée** : vous n'aurez pas à ressaisir vos identifiants à chaque ouverture, y
compris après fermeture complète de l'application.

### 2.4 Mot de passe oublié

1. Sur l'écran de connexion, choisissez **« Mot de passe oublié »**.
2. Saisissez votre adresse e-mail et validez.
3. Un message confirme l'envoi. *Par sécurité, ce message est identique que le compte existe ou non :
   c'est volontaire, cela empêche un tiers de découvrir si une adresse est inscrite.*
4. Ouvrez l'e-mail reçu et touchez le lien : l'application s'ouvre directement sur l'écran de
   réinitialisation.
5. Saisissez votre nouveau mot de passe et validez.

**Le lien expire et ne fonctionne qu'une seule fois.** Si vous le réutilisez, refaites simplement une
demande. Si vous ne recevez rien : vérifiez vos courriers indésirables, puis vérifiez l'adresse saisie.

---

## 3. Guide de l'auditeur

### 3.1 Trouver quelque chose à écouter

L'écran d'accueil présente deux ensembles :

- les **directs en cours** — signalés par une pastille accompagnée de la mention « En direct » (jamais
  par la couleur seule) ;
- les **pistes enregistrées** publiées par les diffuseurs.

Touchez un élément pour l'ouvrir.

### 3.2 Écouter un direct

1. Touchez un direct dans la liste.
2. Le son démarre en quelques secondes ; l'écran affiche le titre et le diffuseur.
3. Pour quitter l'écoute, revenez en arrière puis fermez le lecteur.

**Si le son se coupe :** votre connexion est probablement trop lente. L'application privilégie le direct
sur la continuité — plutôt que d'accumuler du retard, elle laisse tomber ce qui n'a pas pu être reçu à
temps. Vous restez ainsi synchronisé avec l'émission en cours. Rapprochez-vous d'un meilleur réseau et
relancez.

### 3.3 Écouter une piste enregistrée

1. Touchez une piste dans la liste.
2. Utilisez **Lecture / Pause** pour contrôler la lecture.
3. Utilisez la **barre de progression** pour vous déplacer dans la piste : la position et la durée
   totale sont affichées.
4. 🚧 Un réglage de volume sera disponible dans une prochaine version. En attendant, utilisez les
   boutons de volume de votre téléphone.

### 3.4 Continuer à écouter en faisant autre chose

- La lecture **continue** si vous verrouillez l'écran ou passez à une autre application.
- Les commandes **Lecture / Pause** restent disponibles depuis la notification et l'écran de
  verrouillage.
- Dans l'application, un **mini-lecteur** reste affiché en bas de l'écran : touchez-le pour revenir au
  lecteur complet.

🚧 *Prochainement :* mise en pause automatique lors d'un appel entrant et lors du débranchement du casque.

### 3.5 Favoris

- Touchez l'icône de favori sur une piste, un direct ou une playlist pour l'ajouter.
- Touchez-la de nouveau pour la retirer.
- Retrouvez l'ensemble dans l'onglet **Favoris**.

Vos favoris sont enregistrés sur votre compte : vous les retrouvez sur tout appareil où vous vous
connectez.

### 3.6 Playlists

**Créer une playlist**

1. Ouvrez l'onglet **Playlists**.
2. Choisissez **« Nouvelle playlist »**.
3. Renseignez un titre et, si vous le souhaitez, une description.
4. Validez.

Une playlist est **privée par défaut** : vous seul la voyez.

**Ajouter une piste**

1. Depuis une piste, choisissez **« Ajouter à une playlist »**.
2. Sélectionnez la playlist de destination.

La piste est ajoutée **à la fin** : l'ordre d'ajout est l'ordre de lecture.

**Modifier ou supprimer** — depuis le détail de la playlist : renommer, retirer une piste, ou supprimer
la playlist. 🚧 Le réordonnancement des pistes arrivera dans une prochaine version.

---

## 4. Guide du diffuseur

> Ce chapitre nécessite un compte disposant du rôle **diffuseur**. Si l'écran de diffusion ne vous est
> pas proposé, demandez à un administrateur de faire évoluer votre rôle.

### 4.1 Démarrer un direct

1. Ouvrez l'écran **Diffuser**.
2. Renseignez un **titre** — c'est ce que les auditeurs verront dans la liste. Soyez explicite.
3. Ajoutez une **description** si vous le souhaitez.
4. Touchez **« Démarrer »**.
5. À la première utilisation, Android demande l'autorisation d'accéder au **microphone** : acceptez.
6. L'écran indique que la diffusion est en cours. Votre direct apparaît dans la liste publique en
   quelques secondes.

**Bonnes pratiques**

- Testez avec un second appareil avant une diffusion importante.
- Privilégiez le Wi-Fi ou une bonne couverture mobile : votre flux montant conditionne la qualité perçue
  par **tous** vos auditeurs.
- Un casque-micro améliore nettement la qualité et évite l'effet de retour.
- Gardez l'application ouverte pendant la diffusion.

### 4.2 Arrêter un direct

Touchez **« Arrêter »**. Le direct disparaît de la liste publique, vos auditeurs sont informés, et les
ressources serveur sont libérées.

**Vous ne pouvez avoir qu'un seul direct actif à la fois.** Si le démarrage est refusé pour cette raison,
arrêtez d'abord le direct en cours.

### 4.3 Publier une piste enregistrée

1. Ouvrez l'écran **Publier une piste**.
2. Choisissez **« Sélectionner un fichier »** et prenez un fichier audio dans votre téléphone.
3. Renseignez le **titre** et l'**artiste**.
4. Touchez **« Publier »**.
5. Une barre de progression indique l'avancement du transfert. Restez sur l'écran jusqu'à la fin.

Votre piste est ensuite écoutable par tous et peut être ajoutée à des playlists.

**En cas d'échec du transfert**, vérifiez votre connexion et recommencez. Un fichier volumineux sur un
réseau mobile lent peut prendre plusieurs minutes.

### 4.4 Supprimer une piste

Depuis le détail d'une de vos pistes, choisissez **« Supprimer »**. Le fichier audio est également
supprimé du stockage. **Cette action est irréversible.**

---

## 5. Guide de l'administrateur

> 🚧 Les fonctions décrites ci-dessous sont spécifiées et partiellement disponibles côté serveur ;
> l'écran d'administration mobile est en cours de développement. Ce chapitre décrit le fonctionnement
> cible.

### 5.1 Activer ou désactiver une fonctionnalité

Chaque grande fonctionnalité de Lyo est gouvernée par un **interrupteur** (feature flag) modifiable sans
mise à jour de l'application ni redémarrage du serveur.

| Interrupteur | Ce qu'il gouverne |
|---|---|
| `live_streaming` | Diffusion et écoute en direct |
| `track_uploads` | Publication de pistes par les diffuseurs |
| `playlists` | Création et gestion des playlists |
| `favorites` | Mise en favori |
| `chat_websocket` | Discussion entre auditeurs *(non implémenté)* |
| `recommendations` | Recommandations *(non implémenté)* |
| `offline_mode` | Écoute hors connexion *(non implémenté)* |
| `transcoding` | Adaptation automatique du débit *(non implémenté)* |

**Usage principal :** si une fonctionnalité dysfonctionne, la désactiver la retire immédiatement de
l'interface et fait répondre « temporairement indisponible » au serveur, **sans attendre un correctif ni
une nouvelle version**. C'est le moyen de réaction le plus rapide dont dispose l'exploitant.

### 5.2 Gérer les comptes

Consulter la liste des comptes, promouvoir un utilisateur au rôle diffuseur, suspendre ou supprimer un
compte. Un changement de rôle prend effet au prochain rafraîchissement de session de l'utilisateur.

### 5.3 Surveiller la plateforme

Les tableaux de bord distinguent deux familles d'indicateurs, et cette distinction commande deux
réactions différentes :

- **Erreurs techniques** (réponses en erreur du serveur) → *quelque chose est cassé* → un correctif est
  nécessaire ;
- **Dégradation d'expérience** (déconnexions brutales, trames audio abandonnées) → *le système
  fonctionne comme prévu mais les utilisateurs en souffrent* → c'est un problème de dimensionnement ou de
  capacité réseau.

---

## 6. Accessibilité

### Ce qui fonctionne aujourd'hui

- La navigation suit une structure d'écrans stable et un retour arrière prévisible.
- L'information « En direct » est portée par un **texte**, pas uniquement par une couleur.
- La lecture en arrière-plan permet d'utiliser l'application sans rester sur l'écran du lecteur, ce qui
  bénéficie notamment aux personnes utilisant une navigation vocale.

### 🚧 Ce qui est en cours

L'application ne satisfait pas encore l'ensemble des critères d'accessibilité visés (WCAG 2.1 niveau AA).
Les travaux engagés portent sur : les libellés vocalisés de tous les boutons à icône, l'annonce des
changements d'état de lecture, la vérification des contrastes, la taille minimale des zones tactiles
(48 × 48 dp) et le bon comportement de l'interface à 200 % de taille de police. Cet état est présenté
ici en toute transparence plutôt que passé sous silence.

### Conseils immédiats

- **Taille du texte :** l'application suit le réglage système de votre téléphone (Réglages → Affichage →
  Taille de police).
- **Lecture au casque :** les commandes des écouteurs (lecture, pause) sont prises en charge par le
  système.
- **Écoute prolongée :** la lecture en arrière-plan évite de garder l'écran allumé, ce qui économise la
  batterie et réduit la fatigue visuelle.

**Un problème d'accessibilité ?** Signalez-le via une issue GitHub : ces signalements sont traités en
priorité.

---

## 7. Vos données personnelles

**Ce que Lyo collecte :** votre nom d'utilisateur, votre adresse e-mail, une empreinte chiffrée de votre
mot de passe (jamais le mot de passe lui-même), votre rôle, vos favoris et vos playlists, ainsi que les
contenus que vous publiez.

**Ce que Lyo ne collecte pas :** votre nom réel, votre date de naissance, votre position géographique.
Aucun traceur publicitaire, aucun partage avec un tiers à des fins commerciales.

**Vos droits**

| Droit | Comment l'exercer |
|---|---|
| Consulter vos données | Écran **Profil** |
| Corriger vos données | Écran **Profil**, puis **Modifier** |
| Supprimer votre compte | 🚧 Bientôt disponible dans l'application ; en attendant, par demande à l'administrateur |
| Récupérer vos données | 🚧 Export prévu |

**Sécurité.** Votre mot de passe est stocké sous forme d'empreinte irréversible : personne, pas même
l'administrateur, ne peut le lire. ✅ Les échanges entre l'application et le serveur sont chiffrés (TLS) —
y compris le flux audio en direct — donc l'écoute sur un réseau Wi-Fi public ne les expose pas.

---

## 8. Résolution des problèmes

| Symptôme | Cause probable | Que faire |
|---|---|---|
| « Impossible de se connecter au serveur » | Réseau indisponible ou serveur inaccessible | Vérifiez votre connexion, puis réessayez dans quelques instants |
| Le son se coupe pendant un direct | Connexion trop lente : les trames non reçues à temps sont abandonnées pour préserver la synchronisation | Changez de réseau, ou réessayez plus tard |
| Le direct n'apparaît pas dans la liste | La liste n'est pas rafraîchie, ou le direct est terminé | Tirez vers le bas pour rafraîchir |
| « Un direct est déjà en cours » | Vous avez déjà un direct actif | Arrêtez-le avant d'en démarrer un autre |
| La diffusion ne démarre pas | Autorisation micro refusée | Réglages Android → Applications → Lyo → Autorisations → Microphone |
| Le lien de réinitialisation ne fonctionne pas | Lien expiré ou déjà utilisé | Refaites une demande de réinitialisation |
| Déconnecté sans raison apparente | Session expirée | Reconnectez-vous |
| Le transfert de piste échoue | Connexion instable ou fichier volumineux | Passez en Wi-Fi et recommencez |

---

## 9. Glossaire

| Terme | Définition |
|---|---|
| **Direct (live)** | Émission diffusée et écoutée en temps réel, sans enregistrement préalable |
| **Diffuseur** | Utilisateur autorisé à émettre des directs et à publier des pistes |
| **Piste** | Fichier audio enregistré, réécoutable à tout moment |
| **Playlist** | Liste ordonnée de pistes ; l'ordre est celui de la lecture |
| **Favori** | Marque-page personnel sur une piste, un direct ou une playlist |
| **Latence** | Délai entre le moment où le diffuseur parle et celui où vous l'entendez. Lyo vise moins de 2 secondes |
| **Interrupteur de fonctionnalité** | Réglage permettant à un administrateur d'activer ou de couper une fonctionnalité sans mise à jour |
| **Rôle** | Niveau d'autorisation d'un compte : visiteur, auditeur, diffuseur ou administrateur |

# User Guide — Lyo

**Version:** 1.0 · **Date:** 2026-09-03 · **Version française :** [`guide-utilisateur.fr.md`](guide-utilisateur.fr.md)

This guide is for **people using the application**. For installation and development, see the
[developer guide](../README.md).

> **Reading note.** This document is written to work on screen and with a screen reader: every step is
> numbered, every control is named by its label, and no instruction relies on colour or on-screen
> position alone. Features marked 🚧 are not available in the current version.

---

## Contents

- [1. What is Lyo?](#1-what-is-lyo)
- [2. Getting started](#2-getting-started)
- [3. Listener guide](#3-listener-guide)
- [4. Broadcaster guide](#4-broadcaster-guide)
- [5. Administrator guide](#5-administrator-guide)
- [6. Accessibility](#6-accessibility)
- [7. Your personal data](#7-your-personal-data)
- [8. Troubleshooting](#8-troubleshooting)
- [9. Glossary](#9-glossary)

---

## 1. What is Lyo?

Lyo is a **live audio** application. It lets you:

- **listen** to shows broadcast live by other users, with under two seconds of delay — which is what
  makes a live show actually feel live;
- **listen to recorded tracks** published by broadcasters, and organise them into playlists;
- **broadcast live yourself** from your phone, with no hardware and no configuration, if your account has
  the broadcaster role.

**You do not need an account to listen to public live streams.** An account is required for favourites,
playlists and broadcasting.

### The four profiles

| Profile | What you can do |
|---|---|
| **Visitor** (no account) | Browse and listen to public live streams and published tracks |
| **Listener** (standard account) | All of the above, plus favourites, playlists and a profile |
| **Broadcaster** | All of the above, plus starting live streams and publishing tracks |
| **Administrator** | All of the above, plus managing features, accounts and platform monitoring |

---

## 2. Getting started

### 2.1 Install the application

1. Go to the project's releases page (GitHub Releases).
2. Download the `.apk` file of the most recent version.
3. On your Android phone, open the downloaded file.
4. If Android asks you to allow installation from this source, accept and start the installation again.
5. Open Lyo.

### 2.2 Create an account

1. On the welcome screen, choose **"Create an account"**.
2. Enter a **username** — it will be publicly visible.
3. Enter your **email address** — it identifies you and lets you recover your account.
4. Choose a **password**. Tip: at least 12 characters, ideally a phrase that is easy for you to remember
   and hard for anyone else to guess.
5. Confirm with **"Sign up"**.

You are signed in immediately, with no second step.

**If you are told the email or username is already taken:** both must be unique. Pick another one, or
sign in if the account is already yours.

### 2.3 Sign in

1. Choose **"Sign in"**.
2. Enter your email and password.
3. Confirm.

Your session is **kept**: you will not have to type your credentials again on every launch, even after
fully closing the app.

### 2.4 Forgotten password

1. On the sign-in screen, choose **"Forgot password"**.
2. Enter your email address and confirm.
3. A message confirms the request. *For security, this message is identical whether the account exists or
   not: this is deliberate and prevents anyone from discovering whether an address is registered.*
4. Open the email you receive and tap the link: the app opens straight onto the reset screen.
5. Enter your new password and confirm.

**The link expires and works only once.** If you reuse it, simply request a new one. If nothing arrives,
check your spam folder, then check the address you entered.

---

## 3. Listener guide

### 3.1 Find something to listen to

The home screen shows two sets:

- **live streams in progress** — marked with a dot **and** the word "Live" (never colour alone);
- **recorded tracks** published by broadcasters.

Tap an item to open it.

### 3.2 Listen to a live stream

1. Tap a live stream in the list.
2. Sound starts within a few seconds; the screen shows the title and the broadcaster.
3. To stop listening, go back and close the player.

**If the sound cuts out:** your connection is probably too slow. The app favours liveness over
continuity — rather than accumulating delay, it drops what could not arrive in time, which keeps you in
sync with the show as it happens. Move to a better network and try again.

### 3.3 Listen to a recorded track

1. Tap a track in the list.
2. Use **Play / Pause** to control playback.
3. Use the **progress bar** to move within the track: position and total duration are displayed.
4. 🚧 A volume control will arrive in a future version. Until then, use your phone's volume buttons.

### 3.4 Keep listening while doing something else

- Playback **continues** if you lock the screen or switch to another app.
- **Play / Pause** controls remain available from the notification and the lock screen.
- Inside the app, a **mini player** stays visible at the bottom of the screen: tap it to return to the
  full player.

🚧 *Coming soon:* automatic pause on an incoming call and when headphones are unplugged.

### 3.5 Favourites

- Tap the favourite icon on a track, live stream or playlist to add it.
- Tap it again to remove it.
- Find them all in the **Favourites** tab.

Favourites are stored on your account, so you find them again on any device you sign in from.

### 3.6 Playlists

**Create a playlist**

1. Open the **Playlists** tab.
2. Choose **"New playlist"**.
3. Enter a title and, if you wish, a description.
4. Confirm.

A playlist is **private by default**: only you can see it.

**Add a track**

1. From a track, choose **"Add to playlist"**.
2. Select the destination playlist.

The track is added **at the end**: the order in which you add tracks is the order they play in.

**Edit or delete** — from the playlist detail screen: rename it, remove a track, or delete the playlist.
🚧 Reordering tracks will arrive in a future version.

---

## 4. Broadcaster guide

> This chapter requires an account with the **broadcaster** role. If the broadcast screen is not offered
> to you, ask an administrator to upgrade your role.

### 4.1 Start a live stream

1. Open the **Broadcast** screen.
2. Enter a **title** — this is what listeners see in the list, so make it explicit.
3. Add a **description** if you wish.
4. Tap **"Start"**.
5. On first use, Android asks for **microphone** permission: accept.
6. The screen indicates that you are broadcasting. Your stream appears in the public list within seconds.

**Good practice**

- Test with a second device before an important broadcast.
- Prefer Wi-Fi or good mobile coverage: your upstream quality determines what **all** your listeners hear.
- A headset microphone noticeably improves quality and avoids feedback.
- Keep the app open while broadcasting.

### 4.2 Stop a live stream

Tap **"Stop"**. The stream disappears from the public list, your listeners are informed, and server
resources are released.

**You can only have one active live stream at a time.** If starting is refused for that reason, stop the
current one first.

### 4.3 Publish a recorded track

1. Open the **Publish a track** screen.
2. Choose **"Select a file"** and pick an audio file from your phone.
3. Enter the **title** and **artist**.
4. Tap **"Publish"**.
5. A progress bar shows the transfer. Stay on the screen until it completes.

Your track is then playable by everyone and can be added to playlists.

**If the transfer fails**, check your connection and try again. A large file on a slow mobile network can
take several minutes.

### 4.4 Delete a track

From the detail screen of one of your tracks, choose **"Delete"**. The audio file is removed from storage
as well. **This cannot be undone.**

---

## 5. Administrator guide

> 🚧 The functions below are specified and partly available on the server side; the mobile admin screen is
> under development. This chapter describes the target behaviour.

### 5.1 Turn a feature on or off

Every major Lyo feature is governed by a **switch** (feature flag) that can be changed without updating
the app or restarting the server.

| Switch | What it governs |
|---|---|
| `live_streaming` | Live broadcasting and listening |
| `track_uploads` | Track publishing by broadcasters |
| `playlists` | Playlist creation and management |
| `favorites` | Favouriting |
| `chat_websocket` | Listener chat *(not implemented)* |
| `recommendations` | Recommendations *(not implemented)* |
| `offline_mode` | Offline listening *(not implemented)* |
| `transcoding` | Automatic bitrate adaptation *(not implemented)* |

**Main use:** if a feature misbehaves, switching it off removes it from the interface immediately and
makes the server answer "temporarily unavailable", **without waiting for a fix or a new release**. It is
the fastest reaction available to whoever operates the platform.

### 5.2 Manage accounts

View the account list, promote a user to broadcaster, suspend or delete an account. A role change takes
effect at the user's next session refresh.

### 5.3 Monitor the platform

Dashboards separate two families of indicators, and that separation drives two different responses:

- **Technical errors** (server error responses) → *something is broken* → a fix is needed;
- **Experience degradation** (abrupt disconnects, dropped audio chunks) → *the system is behaving as
  designed but users are suffering* → this is a capacity or network sizing problem.

---

## 6. Accessibility

### What works today

- Navigation follows a stable screen structure with predictable back behaviour.
- "Live" status is conveyed by **text**, not colour alone.
- Background playback means you do not have to stay on the player screen, which particularly helps people
  using voice navigation.

### 🚧 Work in progress

The application does not yet meet all of its target accessibility criteria (WCAG 2.1 level AA). Work
under way covers spoken labels for every icon-only button, announcements of playback state changes,
contrast verification, minimum touch target size (48 × 48 dp) and correct behaviour at 200 % font size.
This status is stated openly here rather than left unsaid.

### Immediate tips

- **Text size:** the app follows your phone's system setting (Settings → Display → Font size).
- **Headphone playback:** headset controls (play, pause) are handled by the system.
- **Long listening sessions:** background playback avoids keeping the screen on, saving battery and
  reducing eye strain.

**Found an accessibility problem?** Report it through a GitHub issue — these reports are handled as a
priority.

---

## 7. Your personal data

**What Lyo collects:** your username, your email address, an irreversible hash of your password (never
the password itself), your role, your favourites and playlists, and the content you publish.

**What Lyo does not collect:** your real name, date of birth or location. No advertising trackers, no
commercial sharing with third parties.

**Your rights**

| Right | How to exercise it |
|---|---|
| View your data | **Profile** screen |
| Correct your data | **Profile** screen, then **Edit** |
| Delete your account | 🚧 Coming to the app; until then, by request to the administrator |
| Retrieve your data | 🚧 Export planned |

**Security.** Your password is stored as an irreversible hash: nobody, not even the administrator, can
read it. ✅ Traffic between the app and the server is encrypted (TLS) — the live audio stream included — so
using the app on a public Wi-Fi network does not expose it.

---

## 8. Troubleshooting

| Symptom | Likely cause | What to do |
|---|---|---|
| "Cannot reach the server" | Network unavailable or server unreachable | Check your connection, then retry shortly |
| Sound cuts out during a live stream | Connection too slow: chunks that cannot arrive in time are dropped to preserve sync | Switch networks, or try later |
| The live stream is not in the list | The list is stale, or the stream has ended | Pull down to refresh |
| "A live stream is already running" | You already have an active stream | Stop it before starting another |
| Broadcasting will not start | Microphone permission denied | Android Settings → Apps → Lyo → Permissions → Microphone |
| The reset link does not work | Link expired or already used | Request a new reset |
| Signed out unexpectedly | Session expired | Sign in again |
| Track upload fails | Unstable connection or large file | Switch to Wi-Fi and try again |

---

## 9. Glossary

| Term | Definition |
|---|---|
| **Live stream** | A show broadcast and listened to in real time, with no prior recording |
| **Broadcaster** | A user allowed to run live streams and publish tracks |
| **Track** | A recorded audio file, playable at any time |
| **Playlist** | An ordered list of tracks; the order is the playback order |
| **Favourite** | A personal bookmark on a track, live stream or playlist |
| **Latency** | The delay between the broadcaster speaking and you hearing it. Lyo targets under 2 seconds |
| **Feature switch** | A setting letting an administrator enable or disable a feature without an update |
| **Role** | An account's permission level: visitor, listener, broadcaster or administrator |

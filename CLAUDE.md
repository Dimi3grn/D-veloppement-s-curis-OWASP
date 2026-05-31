# NoteVault — Projet Développement Sécurisé (Ynov B2, Intro Cybersécurité 2025-2026)

> **Contexte pour toute session Claude Code ouverte dans ce dossier.**
> Ce projet est un travail scolaire noté. L'étudiant doit **comprendre et pouvoir
> réexpliquer à l'oral** chaque protection. Ne JAMAIS coder une protection sans
> l'expliquer : pour chaque sécurisation, fournir le triptyque
> **(1) le risque / la faille → (2) la protection → (3) la ligne de code concernée**.
> Date de l'oral : **mardi 2 juin 2026**. Soutenance 15-20 min.

## Le sujet

**NoteVault** : application web de **prise de notes** en texte libre, sécurisée.
- Un utilisateur crée / lit / édite / supprime ses notes.
- Partage à 3 niveaux : **privé** / **partagé** (avec un autre utilisateur) / **public**.
- Espace **admin** : gestion des comptes + page de journaux (logs).
- Le partage à 3 niveaux crée des frontières de contrôle d'accès riches à démontrer.
- Le texte libre est le terrain idéal pour démontrer le **XSS**.

## La stack (et pourquoi)

- **Backend : Go** (`net/http`, bibliothèque standard). Choisi car explicite : on écrit
  nous-mêmes chaque protection → on peut la montrer et l'expliquer à l'oral. Pas de
  framework qui « cache » la sécurité (contrairement à Django/Spring).
- **Frontend : React + Vite** (JavaScript). Rendu Markdown via **`marked`** +
  nettoyage anti-XSS via **`DOMPurify`** (voir `src/lib/markdown.js`).
- **Base de données : SQLite** via `modernc.org/sqlite` (driver **100% Go**, pas besoin
  de compilateur C sur Windows). Accès via `database/sql` + **requêtes paramétrées**.
- **Mots de passe : bcrypt** (`golang.org/x/crypto/bcrypt`).

## Décision d'authentification : COOKIE DE SESSION (pas JWT)

Auth par **cookie de session httpOnly + Secure + SameSite**, session stockée côté serveur.
Raisons (à savoir réexpliquer à l'oral) :
- App **mono-serveur** → l'avantage « stateless » du JWT ne sert à rien ici.
- Besoin de **révocation immédiate** (admin désactive un compte, déconnexion) → trivial
  avec une session serveur, compliqué avec un JWT.
- Cookie **httpOnly** = invisible au JavaScript → **un XSS ne peut pas voler la session**
  (synergie avec la démo XSS). Un JWT en localStorage serait volable par XSS.
- À l'oral, expliquer « j'ai écarté le JWT car inadapté à CE contexte » = bon point
  sur le critère compréhension technique. Le JWT reste valable ailleurs (microservices,
  OAuth/OIDC, mobile).

## Direction artistique

Style **« Éditorial sombre »** (inspiré d'Outline), `src/App.css` issu d'un brief
Claude Design : fond près de l'encre (#17181b), **titres en serif Source Serif 4**
(chargée via `@import` Google Font en tête de `App.css` ; la pile système prend le
relais si on la retire), bordures fines (1px) au lieu d'ombres, coins 3px, **zéro
transform au survol** (transitions de couleur uniquement), un seul accent bleu-encre
désaturé (#5a82c2) pour liens/actif/focus, largeur de lecture bridée à 680px, focus
clavier net, scrollbars discrètes, responsive < 720px. Surtout PAS de style « template
IA » (arrondis partout, violet, emojis, ombres). Anciennes maquettes dans `design/`.

Les notes s'écrivent en **Markdown brut** (mode édition) et s'affichent en **rendu
nettoyé** (mode lecture), comme Outline/Notion.

## Architecture

```
React (navigateur, port 5173)  --/api/*-->  Go API (port 8080)  -->  SQLite
                          (proxy Vite en dev, évite CORS)
```
Règle d'or : **on ne fait jamais confiance au front**. Tous les contrôles de sécurité
sont côté serveur (Go).

## Structure des dossiers

```
NoteVault/
├── CLAUDE.md            <- ce fichier
├── backend/             (Go)
│   ├── go.mod           (module "notevault")
│   ├── main.go
│   └── internal/
│       ├── auth/        login, sessions, bcrypt
│       ├── handlers/    routes (notes, users, admin)
│       ├── middleware/  auth, contrôle d'accès, headers, rate-limit
│       └── db/          connexion + requêtes SQL
└── frontend/            (React + Vite)
    ├── vite.config.js   (proxy /api -> :8080)
    └── src/
        ├── pages/       Login, Notes, Admin
        └── api/         appels au back
```

## Mapping OWASP Top 10 (cible : 8/8)

| Catégorie | Démonstration sur NoteVault |
|---|---|
| **A01 Broken Access Control** ⭐ | IDOR : un user ne peut pas lire `/api/notes/{id}` d'un autre ; respect des niveaux privé/partagé/public ; `/api/admin` réservé admin |
| **A02 Cryptographic Failures** | Mots de passe hachés bcrypt ; cookie Secure ; HTTPS |
| **A03 Injection** | SQLi : requêtes paramétrées (`?`) ; XSS : échappement (React) + démo `dangerouslySetInnerHTML` contrôlée |
| **A04 Insecure Design** | Moindre privilège, rate-limiting login |
| **A05 Security Misconfiguration** | Headers HTTP sécurisés, mode debug coupé, erreurs génériques, CORS strict |
| **A07 Authentication Failures** | Politique de mot de passe, blocage après N essais, sessions sécurisées |
| **A09 Logging & Monitoring Failures** | Journaux admin (connexions, modifs de notes) |

## Comment lancer le projet

```bash
# Terminal 1 — backend
cd backend
go run main.go            # http://localhost:8080

# Terminal 2 — frontend
cd frontend
npm install               # la première fois
npm run dev               # http://localhost:5173
```

## Feuille de route (état)

1. [x] Squelette projet (Go + React qui tournent, proxy, /api/health)
2. [x] Base de données SQLite + schéma (users, notes, sessions, note_shares, logs)
3. [x] Authentification (bcrypt + sessions cookie) — A02, A07
4. [x] CRUD notes + contrôle d'accès — A01
5. [x] Partage de notes (privé/partagé/public)
6. [x] Protections injection — A03 (SQLi + XSS via DOMPurify)
7. [x] Espace admin + journaux — A09
8. [x] Durcissement (headers, rate-limit, CORS) — A04, A05, A07
   - [ ] BONUS restant : Docker + HTTPS (Caddy), scan A06 (govulncheck/npm audit)
9. [x] Fiche orale `SOUTENANCE.md` (risque/protection/code/démo par catégorie)

## État : MVP complet et sécurisé (8 catégories OWASP couvertes)
A01 · A02 · A03 · A04 · A05 · A07 · A09 (+ A08 partiel). Voir `SOUTENANCE.md`.

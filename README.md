# 🔐 NoteVault — Application de prise de notes sécurisée

Projet **Développement Sécurisé** — Ynov B2, *Introduction à la cybersécurité* (2025-2026).
Application web conçue autour des principes de l'**OWASP Top 10**.

NoteVault est une application de **prise de notes en Markdown** : un utilisateur écrit ses
notes, les lit en rendu, et peut les partager (privé / partagé / public). Un espace
**administrateur** permet de gérer les comptes et de consulter un journal de sécurité.

L'objectif du projet n'est pas la richesse fonctionnelle, mais la **démonstration de
mécanismes de sécurité** et la capacité à les expliquer.

---

## ✨ Fonctionnalités

- **Authentification** (inscription / connexion / déconnexion) par session.
- **Notes en Markdown** : édition en texte brut, lecture en rendu nettoyé.
- **Partage à 3 niveaux** : privé, partagé avec un utilisateur précis, ou public.
- **Contrôle d'accès** strict : chacun ne voit que ce qu'il a le droit de voir.
- **Espace admin** : gestion des comptes (activer / désactiver) + **journal de sécurité**.

---

## 🛠️ Stack technique

| Couche | Choix | Pourquoi |
|---|---|---|
| Backend | **Go** (`net/http`, lib standard) | Code explicite : chaque protection est écrite et visible, pas cachée par un framework. |
| Frontend | **React + Vite** | SPA, rendu Markdown via `marked` + nettoyage `DOMPurify`. |
| Base de données | **SQLite** (`modernc.org/sqlite`, pur Go) | Fichier unique, zéro installation ; `database/sql` force les requêtes paramétrées. |
| Mots de passe | **bcrypt** | Hachage salé, lent, à sens unique. |
| Auth | **Cookie de session httpOnly** | Résiste au vol par XSS et permet la révocation immédiate (vs JWT). |

---

## 🛡️ Sécurité — couverture OWASP Top 10

| Catégorie | Protection mise en place | Où |
|---|---|---|
| **A01 Broken Access Control** | Middleware d'auth + vérification d'*ownership* + RBAC ; réponses 404 (ne révèle pas l'existence) | `backend/internal/auth/middleware.go`, `backend/internal/notes/handlers.go` |
| **A02 Cryptographic Failures** | Mots de passe **bcrypt** (salés) ; token de session via `crypto/rand` | `backend/internal/auth/password.go`, `session.go` |
| **A03 Injection** | **Requêtes SQL paramétrées** (`?`) ; **XSS** neutralisé par DOMPurify | `backend/internal/db/*.go`, `frontend/src/lib/markdown.js` |
| **A04 Insecure Design** | Rate-limiting du login, moindre privilège, *deny by default* | `backend/internal/middleware/ratelimit.go` |
| **A05 Security Misconfiguration** | En-têtes de sécurité (CSP, X-Frame-Options…), CORS en liste blanche, erreurs génériques | `backend/internal/middleware/security.go` |
| **A07 Authentication Failures** | Politique de mot de passe, messages génériques, anti-énumération, sessions révocables | `backend/internal/auth/handlers.go` |
| **A09 Logging & Monitoring** | Journal des événements de sécurité (sans donnée sensible) | `backend/internal/db/logs.go` |

> Détails, fiches par catégorie et **script de démonstration** : voir [`SOUTENANCE.md`](./SOUTENANCE.md).

---

## 🏗️ Architecture

```
React (navigateur, :5173)  ──/api/*──►  API Go (:8080)  ──►  SQLite (fichier)
                       (proxy Vite en dev, évite CORS)
```

Règle d'or : **on ne fait jamais confiance au front.** Tous les contrôles de sécurité
sont côté serveur (Go).

---

## 🚀 Installation et lancement

**Prérequis :** [Go](https://go.dev/) ≥ 1.24 et [Node.js](https://nodejs.org/) ≥ 20.

```bash
# 1. Backend (terminal 1)
cd backend
go run main.go                 # http://localhost:8080

# 2. Frontend (terminal 2)
cd frontend
npm install                    # la première fois seulement
npm run dev                    # http://localhost:5173
```

Ouvre ensuite **http://localhost:5173**.

### Compte administrateur

Un compte admin est créé automatiquement au premier démarrage. Ses identifiants
proviennent de variables d'environnement (aucun secret en dur dans le code) :

```bash
# définir le mot de passe admin au lancement (sinon un repli de dev s'applique)
ADMIN_EMAIL=admin@notevault.local ADMIN_PASSWORD=VotreMotDePasse go run main.go
```

Le mot de passe effectivement utilisé est affiché dans la console au démarrage.

---

## 📁 Structure du projet

```
.
├── backend/                 # API Go
│   ├── main.go              # point d'entrée, routes, middlewares
│   └── internal/
│       ├── auth/            # inscription, connexion, sessions, bcrypt
│       ├── db/              # connexion SQLite + requêtes (paramétrées)
│       ├── notes/           # CRUD des notes + contrôle d'accès
│       ├── admin/           # gestion des comptes + journaux
│       └── middleware/      # headers de sécurité, CORS, rate-limit
├── frontend/                # application React + Vite
│   └── src/
│       ├── components/      # Auth, NotesApp, NoteEditor, Admin
│       ├── lib/markdown.js  # rendu Markdown + nettoyage XSS
│       └── api/client.js    # appels à l'API
├── design/                  # maquettes de la direction artistique
├── SOUTENANCE.md            # fiches OWASP + script de démo (oral)
└── README.md
```

---

## 📝 Note

Projet pédagogique. Les comptes de test et la base `notevault.db` sont à régénérer
(supprimer `backend/notevault.db` puis relancer) avant toute présentation.

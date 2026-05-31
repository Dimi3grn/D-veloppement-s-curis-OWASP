# NoteVault — Antisèche de soutenance (Développement sécurisé, Ynov B2)

> But de l'oral (15-20 min) : présenter l'app, démontrer les fonctionnalités,
> **expliquer avec tes propres mots les protections OWASP** et montrer la partie
> du code concernée. La note récompense la **compréhension**, pas le par-cœur.

---

## 0. Lancer le projet (avant l'oral)

```bash
# Terminal 1 — backend Go
cd backend
go run main.go            # http://localhost:8080

# Terminal 2 — frontend React
cd frontend
npm install               # la première fois seulement
npm run dev               # http://localhost:5173
```

**Comptes de démonstration :**

| Email | Mot de passe | Rôle |
|---|---|---|
| alice@test.com | motdepasse123 | user (a des notes) |
| bob@test.com | motdepasse456 | user |
| admin@notevault.local | défini par la variable d'env `ADMIN_PASSWORD` (repli dev : `changeme-dev-only`) | admin |

> Comptes de test jetables. L'admin est créé au 1er démarrage : pour fixer son mot de
> passe, lance le backend avec `ADMIN_PASSWORD=...` (sinon le repli de dev s'applique).
> Le mot de passe admin affiché dans la console au démarrage est celui réellement utilisé.

---

## 1. Présentation (≈1 min)

NoteVault = appli de **prise de notes en Markdown**, avec partage (privé / partagé /
public) et un espace **admin**. Stack : **Go** (backend/API), **React** (frontend),
**SQLite** (base). Le backend est le **gardien** : on ne fait jamais confiance au front.

**Pourquoi Go + cookies de session (et pas un framework + JWT) ?**
- Go n'écrit aucune sécurité « magique » : **je montre et j'explique chaque ligne**.
- **Cookie de session httpOnly** plutôt que JWT, parce que : mono-serveur (le « stateless »
  du JWT ne sert à rien ici), besoin de **révocation immédiate** (désactivation de compte),
  et un cookie httpOnly **résiste au vol par XSS**. Savoir *quand ne pas* utiliser le JWT
  est un choix d'ingénieur.

---

## 2. Les fiches OWASP (le cœur de la note)

Pour chaque faille : **le risque → la protection → le fichier → la démo**.

### A01 — Broken Access Control ⭐ (la plus importante)
- **Risque** : changer un id dans l'URL (`/api/notes/1`) pour lire les données d'autrui (IDOR).
- **Protection** : (1) middleware `RequireAuth` (aucune route sans session), (2) on filtre
  toujours sur l'identité **prouvée par la session**, jamais sur un paramètre du client,
  (3) vérification d'**ownership** sur chaque note. On renvoie **404** (pas 403) pour ne
  pas révéler l'existence de la ressource. Rôles (RBAC) via `RequireAdmin`.
- **Code** : `backend/internal/auth/middleware.go`, `backend/internal/notes/handlers.go` (blocs `CONTRÔLE D'ACCÈS` + `canRead`).
- **Démo** : connecté en Bob, ouvrir une note privée d'Alice → « Note introuvable ». Désactiver
  le compte de Bob depuis l'admin → sa session est coupée **instantanément**.

### A02 — Cryptographic Failures
- **Risque** : mots de passe en clair → base volée = tous les comptes compromis.
- **Protection** : **bcrypt** (sel aléatoire unique + lent + à sens unique). Token de session
  généré avec **`crypto/rand`** (imprévisible). Cookie `Secure` en prod (HTTPS).
- **Code** : `backend/internal/auth/password.go`, `backend/internal/auth/session.go` (`generateToken`).
- **Démo** : `grep "motdepasse123" backend/notevault.db` → **0 résultat**. Montrer 2 hash
  différents pour le même mot de passe (le sel).

### A03 — Injection (SQL + XSS)
- **Risque SQLi** : `' OR '1'='1` pour contourner une requête.
- **Protection SQLi** : **requêtes paramétrées** (`?`) partout — code et donnée séparés.
- **Risque XSS** : du texte libre rendu en HTML peut exécuter du JS chez le lecteur.
- **Protection XSS** : **DOMPurify** nettoie le HTML rendu (retire `<script>`, `onerror`, etc.).
- **Code** : tous les `?` dans `backend/internal/db/*.go` ; `frontend/src/lib/markdown.js`.
- **Démo** : créer une note avec `<img src=x onerror="alert(document.cookie)">`. Avec DOMPurify :
  rien ne se passe (l'attribut `onerror` est supprimé). **Et** `document.cookie` ne contient pas
  la session (httpOnly) → même un XSS ne volerait rien.

### A04 — Insecure Design
- **Risque** : conception qui facilite l'abus (force brute, privilèges trop larges).
- **Protection** : **rate-limiting** du login (5 essais/min/IP → 429), **moindre privilège**
  (rôle `user` par défaut), **deny-by-default** (tout est refusé sauf autorisation explicite).
- **Code** : `backend/internal/middleware/ratelimit.go`.
- **Démo** : envoyer 6 connexions ratées de suite → la 6e renvoie `429 Too Many Requests`.

### A05 — Security Misconfiguration
- **Risque** : en-têtes manquants, CORS trop ouvert, erreurs trop détaillées.
- **Protection** : en-têtes de sécurité sur **toutes** les réponses (`X-Frame-Options: DENY`
  anti-clickjacking, `X-Content-Type-Options: nosniff`, `Referrer-Policy`, **CSP**) ;
  **CORS en liste blanche** (une seule origine, jamais `*`) ; messages d'erreur **génériques**.
- **Code** : `backend/internal/middleware/security.go`.
- **Démo** : `curl -i http://localhost:8080/api/health` → voir les en-têtes. `curl` avec une
  `Origin` pirate → aucun en-tête CORS renvoyé.

### A07 — Identification & Authentication Failures
- **Risque** : force brute, vol de session, énumération des comptes, mots de passe faibles.
- **Protection** : politique de mot de passe (≥ 8), **messages génériques** (« Identifiants
  invalides » que l'email existe ou non), `dummyHash` **anti-énumération par le temps**,
  cookie **httpOnly + SameSite**, session **révocable**, rate-limiting.
- **Code** : `backend/internal/auth/handlers.go` (`Login`), `session.go` (le cookie).
- **Démo** : se tromper d'email vs de mot de passe → **même message**. Déconnexion → l'ancien
  cookie est mort.

### A09 — Security Logging & Monitoring Failures
- **Risque** : sans journal, une attaque passe inaperçue et est inanalysable.
- **Protection** : on journalise les **événements de sécurité** (connexions OK/KO, accès
  refusés, actions admin) avec qui/quand/IP, **sans jamais stocker de mot de passe**.
- **Code** : `backend/internal/db/logs.go` ; appels dans `auth/handlers.go` et `auth/middleware.go`.
- **Démo** : espace admin → **Journal de sécurité** : on voit la tentative d'accès admin
  d'Alice refusée, la désactivation de Bob, etc.

> **A08** (intégrité) est partiellement couvert : `SameSite` (anti-CSRF), lockfiles de
> dépendances. **A10 (SSRF)** : non applicable — l'app ne va chercher aucune URL fournie
> par l'utilisateur (à dire si on te pose la question).

---

## 3. Script de démo en direct (ordre conseillé)

1. **Connexion** (alice) → montrer l'app, créer/éditer une note en Markdown.
2. **A01** : en Bob, tenter d'ouvrir une note privée d'Alice via l'URL → refusé.
3. **A03 XSS** : note avec `<img onerror=...>` → neutralisée ; expliquer le lien httpOnly.
4. **A07** : mauvais identifiants → message générique ; déconnexion → session morte.
5. **A04** : 6 logins ratés → 429.
6. **Admin (A09 + A01)** : journal de sécurité, désactiver Bob → sa session saute.
7. **A02** : `grep` du mot de passe dans la base → introuvable, montrer le hash.
8. **A05** : `curl -i` → en-têtes de sécurité.

---

## 4. Questions probables du jury + réponses courtes

- **« Pourquoi pas de JWT ? »** → Mono-serveur, besoin de révocation, et httpOnly protège du
  vol XSS. Le JWT est très bien ailleurs (microservices, OAuth, mobile).
- **« 404 au lieu de 403 ? »** → Pour ne pas révéler qu'une ressource existe.
- **« DOMPurify suffit ? »** → C'est la barrière principale ; la CSP et le cookie httpOnly
  sont des barrières supplémentaires (défense en profondeur).
- **« bcrypt vs SHA-256 ? »** → SHA-256 est rapide (mauvais pour des mots de passe) et sans
  sel ; bcrypt est lent, salé, conçu pour ça.
- **« Le rate-limit en mémoire ? »** → Suffisant pour un mono-serveur ; en prod multi-serveur
  on le mettrait dans Redis.

---

## 5. Pistes d'amélioration (à mentionner = maturité)
- Conteneurisation **Docker** (non-root) + **HTTPS** via reverse proxy (Caddy) → renforce A05.
- Scan de dépendances **A06** : `govulncheck` (Go) + `npm audit` (front).
- Jeton **CSRF** explicite en plus de `SameSite`.
- Rotation de l'identifiant de session après connexion (anti session-fixation).

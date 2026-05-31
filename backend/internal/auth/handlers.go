package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"notevault/internal/db"
)

// Handler regroupe les routes d'authentification. Il détient la connexion à la base.
type Handler struct {
	DB *sql.DB
	// SecureCookies : true en production (HTTPS), false en dev local (HTTP).
	SecureCookies bool
}

// writeJSON est un petit utilitaire pour renvoyer une réponse JSON avec un code HTTP.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError renvoie un message d'erreur générique. On reste VOLONTAIREMENT vague
// dans les messages liés à l'auth (protection A07) pour ne pas aider un attaquant.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// registerRequest = la forme attendue du corps JSON de l'inscription.
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register gère POST /api/register : création d'un compte utilisateur.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// 1) On décode le JSON envoyé par le front.
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Requête invalide")
		return
	}

	// 2) VALIDATION DES ENTRÉES (ne jamais faire confiance au front).
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !strings.Contains(req.Email, "@") || len(req.Email) > 254 {
		writeError(w, http.StatusBadRequest, "Email invalide")
		return
	}
	// Politique de mot de passe (protection A07) : longueur minimale.
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "Le mot de passe doit faire au moins 8 caractères")
		return
	}

	// 3) On vérifie que l'email n'est pas déjà pris.
	existing, err := db.GetUserByEmail(h.DB, req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "Cet email est déjà utilisé")
		return
	}

	// 4) On HACHE le mot de passe (jamais stocké en clair) — protection A02.
	hash, err := HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// 5) On crée l'utilisateur avec le rôle "user" par défaut (moindre privilège - A04).
	id, err := db.CreateUser(h.DB, req.Email, hash, "user")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// 6) On répond. On ne renvoie JAMAIS le hash.
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":    id,
		"email": req.Email,
	})
}

// loginRequest = la forme attendue du corps JSON de la connexion.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login gère POST /api/login : vérifie les identifiants et ouvre une session.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Requête invalide")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// On cherche l'utilisateur.
	user, err := db.GetUserByEmail(h.DB, req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Vérification du mot de passe.
	var passwordOK bool
	if user != nil && user.IsActive {
		passwordOK = CheckPassword(user.PasswordHash, req.Password)
	} else {
		// L'email n'existe pas (ou compte désactivé) : on lance quand même un bcrypt
		// "à vide" pour que le temps de réponse soit identique (anti-énumération, A07).
		CheckPassword(dummyHash, req.Password)
		passwordOK = false
	}

	// MESSAGE GÉNÉRIQUE (A07) : on ne dit JAMAIS "email inconnu" ou "mauvais mot de
	// passe". Un seul message pour les deux cas => l'attaquant n'apprend rien.
	if !passwordOK {
		// Journal de sécurité (A09) : on trace l'échec, sans le mot de passe.
		db.AddLog(h.DB, req.Email, "login_failed", "identifiants invalides", r.RemoteAddr)
		writeError(w, http.StatusUnauthorized, "Identifiants invalides")
		return
	}

	// Identifiants corrects => on crée une session.
	token, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	expiresAt := time.Now().Add(sessionDuration)
	if err := db.CreateSession(h.DB, token, user.ID, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// On dépose le cookie de session (httpOnly + Secure + SameSite).
	h.setSessionCookie(w, token, expiresAt)

	// Journal de sécurité (A09) : connexion réussie.
	db.AddLog(h.DB, user.Email, "login_success", "", r.RemoteAddr)

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	})
}

// Logout gère POST /api/logout : détruit la session côté serveur ET le cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		// On supprime la session en base => le token devient inutilisable (révocation).
		db.DeleteSession(h.DB, cookie.Value)
	}
	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Déconnecté"})
}

// Me gère GET /api/me : renvoie l'utilisateur connecté (ou 401 si personne).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := h.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Non connecté")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	})
}

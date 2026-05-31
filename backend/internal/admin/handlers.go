// Package admin regroupe les routes réservées aux administrateurs.
package admin

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"notevault/internal/auth"
	"notevault/internal/db"
)

type Handler struct {
	DB *sql.DB
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// userView = vue publique d'un utilisateur (SANS le hash du mot de passe).
type userView struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

// ListUsers : GET /api/admin/users — liste tous les comptes.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := db.ListUsers(h.DB)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	// On ne renvoie JAMAIS le password_hash au front (minimisation des données).
	out := make([]userView, 0, len(users))
	for _, u := range users {
		out = append(out, userView{ID: u.ID, Email: u.Email, Role: u.Role, IsActive: u.IsActive})
	}
	writeJSON(w, http.StatusOK, out)
}

// setActive factorise activer/désactiver un compte.
func (h *Handler) setActive(w http.ResponseWriter, r *http.Request, active bool) {
	admin, _ := auth.UserFromContext(r) // l'admin qui agit

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Identifiant invalide")
		return
	}

	// Garde-fou : un admin ne peut pas se désactiver lui-même (anti auto-blocage).
	if !active && id == admin.ID {
		writeError(w, http.StatusBadRequest, "Vous ne pouvez pas désactiver votre propre compte")
		return
	}

	target, err := db.GetUserByID(h.DB, id)
	if err != nil || target == nil {
		writeError(w, http.StatusNotFound, "Utilisateur introuvable")
		return
	}

	if err := db.SetUserActive(h.DB, id, active); err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if !active {
		// RÉVOCATION : on supprime toutes ses sessions => déconnexion immédiate.
		// (C'est l'avantage des sessions serveur dont on a parlé à l'étape 3.)
		_ = db.DeleteUserSessions(h.DB, id)
		db.AddLog(h.DB, admin.Email, "account_disabled", "compte désactivé : "+target.Email, r.RemoteAddr)
	} else {
		db.AddLog(h.DB, admin.Email, "account_enabled", "compte réactivé : "+target.Email, r.RemoteAddr)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Compte mis à jour"})
}

// DisableUser : POST /api/admin/users/{id}/disable
func (h *Handler) DisableUser(w http.ResponseWriter, r *http.Request) { h.setActive(w, r, false) }

// EnableUser : POST /api/admin/users/{id}/enable
func (h *Handler) EnableUser(w http.ResponseWriter, r *http.Request) { h.setActive(w, r, true) }

// ListLogs : GET /api/admin/logs — les 200 derniers événements de sécurité.
func (h *Handler) ListLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := db.ListLogs(h.DB, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

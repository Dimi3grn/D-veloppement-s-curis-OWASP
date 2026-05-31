package auth

import (
	"context"
	"net/http"

	"notevault/internal/db"
)

// contextKey est un type privé pour éviter les collisions de clés dans le contexte.
type contextKey string

const userContextKey contextKey = "user"

// RequireAuth est un MIDDLEWARE : il enveloppe un handler et n'exécute celui-ci
// que si une session valide est présente. Sinon il renvoie 401.
//
// C'est le principe "deny by default" (A01/A04) : par défaut, on refuse ;
// on n'autorise que si l'utilisateur est authentifié.
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := h.CurrentUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Authentification requise")
			return
		}
		// On range l'utilisateur dans le "contexte" de la requête, pour que le
		// handler suivant puisse savoir QUI fait la requête, sans re-vérifier.
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// UserFromContext récupère l'utilisateur injecté par RequireAuth.
// Les handlers protégés l'utilisent pour connaître l'auteur de la requête.
func UserFromContext(r *http.Request) (*db.User, bool) {
	user, ok := r.Context().Value(userContextKey).(*db.User)
	return user, ok
}

// RequireAdmin : middleware qui exige une session valide ET le rôle "admin".
// Il enveloppe RequireAuth (donc auth obligatoire), puis vérifie le rôle.
// C'est le contrôle d'accès basé sur les rôles (RBAC) - lié à A01.
func (h *Handler) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return h.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, _ := UserFromContext(r)
		if user.Role != "admin" {
			// On journalise la tentative d'accès non autorisé (A09).
			db.AddLog(h.DB, user.Email, "access_denied", "tentative d'accès admin", r.RemoteAddr)
			writeError(w, http.StatusForbidden, "Accès réservé aux administrateurs")
			return
		}
		next(w, r)
	})
}

package auth

import (
	"crypto/rand"    // génère de l'aléatoire CRYPTOGRAPHIQUE (imprévisible)
	"encoding/base64"
	"net/http"
	"time"

	"notevault/internal/db"
)

// sessionCookieName : le nom du cookie déposé dans le navigateur.
const sessionCookieName = "session"

// sessionDuration : durée de vie d'une session.
const sessionDuration = 24 * time.Hour

// generateToken crée un identifiant de session aléatoire et imprévisible.
//
// IMPORTANT (A02) : on utilise crypto/rand, PAS math/rand.
//   - math/rand est prévisible (un attaquant pourrait deviner les prochains tokens).
//   - crypto/rand est conçu pour la sécurité : impossible à prédire.
// 32 octets = 256 bits d'entropie => impossible à deviner par force brute.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// base64url : transforme les octets en texte utilisable dans un cookie.
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// setSessionCookie dépose le cookie de session dans la réponse.
func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:    sessionCookieName,
		Value:   token,
		Path:    "/",
		Expires: expires,

		// --- LES 3 PROTECTIONS CLÉS DU COOKIE ---

		// HttpOnly : le cookie est INVISIBLE au JavaScript (document.cookie ne le voit pas).
		// => même un XSS ne peut PAS voler la session. C'est notre choix anti-XSS.
		HttpOnly: true,

		// Secure : le cookie n'est envoyé QUE sur HTTPS (chiffré).
		// En dev local (HTTP), on le met à false sinon le navigateur ne l'enverrait pas.
		// En production (HTTPS), il DOIT être true.
		Secure: h.SecureCookies,

		// SameSite=Lax : le cookie n'est pas envoyé lors de requêtes venant d'un
		// AUTRE site => protection de base contre le CSRF (on renforcera à l'étape 8).
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie efface le cookie (à la déconnexion).
func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // -1 => supprime le cookie immédiatement
	})
}

// CurrentUser lit le cookie de session de la requête et renvoie l'utilisateur connecté.
// Renvoie (nil, false) si personne n'est connecté. Sera réutilisé par le middleware (étape 4).
func (h *Handler) CurrentUser(r *http.Request) (*db.User, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, false // pas de cookie => pas connecté
	}
	user, err := db.GetUserBySession(h.DB, cookie.Value)
	if err != nil || user == nil {
		return nil, false // session invalide / expirée / compte désactivé
	}
	return user, true
}

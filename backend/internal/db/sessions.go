package db

import (
	"database/sql"
	"time"
)

// CreateSession enregistre une nouvelle session côté serveur.
func CreateSession(database *sql.DB, id string, userID int64, expiresAt time.Time) error {
	_, err := database.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expiresAt,
	)
	return err
}

// GetUserBySession renvoie l'utilisateur associé à un token de session,
// À CONDITION que la session ne soit pas expirée ET que le compte soit actif.
// Renvoie (nil, nil) si la session est invalide/expirée/le compte désactivé.
//
// C'est ici que se joue la RÉVOCATION : si l'admin désactive le compte
// (is_active = 0), cette requête ne renvoie plus rien => accès coupé instantanément.
func GetUserBySession(database *sql.DB, token string) (*User, error) {
	u := &User{}
	err := database.QueryRow(
		`SELECT u.id, u.email, u.password_hash, u.role, u.is_active, u.created_at
		   FROM sessions s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.id = ? AND s.expires_at > ? AND u.is_active = 1`,
		token, time.Now(),
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// DeleteSession supprime UNE session (utilisé à la déconnexion).
func DeleteSession(database *sql.DB, id string) error {
	_, err := database.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteUserSessions supprime TOUTES les sessions d'un utilisateur
// (utilisé quand l'admin désactive un compte : on le déconnecte partout).
func DeleteUserSessions(database *sql.DB, userID int64) error {
	_, err := database.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

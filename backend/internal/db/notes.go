package db

import (
	"database/sql"
	"time"
)

// Note représente une note. UserID = le propriétaire (clé du contrôle d'accès).
type Note struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Visibility string    `json:"visibility"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateNote crée une note appartenant à userID.
func CreateNote(database *sql.DB, userID int64, title, content, visibility string) (int64, error) {
	res, err := database.Exec(
		`INSERT INTO notes (user_id, title, content, visibility) VALUES (?, ?, ?, ?)`,
		userID, title, content, visibility,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetNoteByID récupère une note par son id (sans filtre de propriétaire :
// c'est le HANDLER qui décidera ensuite si l'utilisateur a le droit de la voir).
func GetNoteByID(database *sql.DB, id int64) (*Note, error) {
	n := &Note{}
	err := database.QueryRow(
		`SELECT id, user_id, title, content, visibility, created_at, updated_at
		   FROM notes WHERE id = ?`, id,
	).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Visibility, &n.CreatedAt, &n.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}

// ListNotesByUser renvoie UNIQUEMENT les notes du propriétaire donné.
// Le filtre "WHERE user_id = ?" est déjà une protection d'accès (A01).
func ListNotesByUser(database *sql.DB, userID int64) ([]Note, error) {
	rows, err := database.Query(
		`SELECT id, user_id, title, content, visibility, created_at, updated_at
		   FROM notes WHERE user_id = ? ORDER BY updated_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Visibility, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// ListPublicNotes renvoie toutes les notes PUBLIQUES, SAUF celles de l'utilisateur
// courant (pour ne pas les afficher en double avec ses propres notes).
func ListPublicNotes(database *sql.DB, excludeUserID int64) ([]Note, error) {
	rows, err := database.Query(
		`SELECT id, user_id, title, content, visibility, created_at, updated_at
		   FROM notes WHERE visibility = 'public' AND user_id != ? ORDER BY updated_at DESC`,
		excludeUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Visibility, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// UpdateNote met à jour le contenu d'une note (le contrôle d'accès est fait avant, par le handler).
func UpdateNote(database *sql.DB, id int64, title, content, visibility string) error {
	_, err := database.Exec(
		`UPDATE notes SET title = ?, content = ?, visibility = ?, updated_at = CURRENT_TIMESTAMP
		  WHERE id = ?`,
		title, content, visibility, id,
	)
	return err
}

// DeleteNote supprime une note (contrôle d'accès fait avant, par le handler).
func DeleteNote(database *sql.DB, id int64) error {
	_, err := database.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}

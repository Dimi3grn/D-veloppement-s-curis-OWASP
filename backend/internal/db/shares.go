package db

import "database/sql"

// ShareNoteWithUser partage une note avec un utilisateur (sans erreur si déjà partagé).
func ShareNoteWithUser(database *sql.DB, noteID, userID int64) error {
	// "INSERT OR IGNORE" : si la paire existe déjà, on n'échoue pas.
	_, err := database.Exec(
		`INSERT OR IGNORE INTO note_shares (note_id, user_id) VALUES (?, ?)`,
		noteID, userID,
	)
	return err
}

// UnshareNoteWithUser retire le partage d'une note pour un utilisateur.
func UnshareNoteWithUser(database *sql.DB, noteID, userID int64) error {
	_, err := database.Exec(
		`DELETE FROM note_shares WHERE note_id = ? AND user_id = ?`,
		noteID, userID,
	)
	return err
}

// IsNoteSharedWith indique si une note est partagée avec un utilisateur donné.
// C'est ce que le contrôle d'accès interroge pour les notes "shared".
func IsNoteSharedWith(database *sql.DB, noteID, userID int64) (bool, error) {
	var one int
	err := database.QueryRow(
		`SELECT 1 FROM note_shares WHERE note_id = ? AND user_id = ?`,
		noteID, userID,
	).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListNotesSharedWithUser renvoie les notes que d'AUTRES ont partagées avec userID.
func ListNotesSharedWithUser(database *sql.DB, userID int64) ([]Note, error) {
	rows, err := database.Query(
		`SELECT n.id, n.user_id, n.title, n.content, n.visibility, n.created_at, n.updated_at
		   FROM notes n
		   JOIN note_shares s ON s.note_id = n.id
		  WHERE s.user_id = ? AND n.visibility = 'shared'
		  ORDER BY n.updated_at DESC`,
		userID,
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

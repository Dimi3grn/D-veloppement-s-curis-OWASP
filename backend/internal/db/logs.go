package db

import (
	"database/sql"
	"time"
)

// LogEntry = une ligne du journal de sécurité.
type LogEntry struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UserEmail string    `json:"user_email"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	IP        string    `json:"ip"`
}

// AddLog enregistre un événement de sécurité. On ne logue JAMAIS de mot de passe
// ni de donnée sensible (protection A09 : journaliser sans créer de fuite).
func AddLog(database *sql.DB, userEmail, action, detail, ip string) {
	// On ignore l'erreur volontairement : un échec de log ne doit pas casser
	// la requête de l'utilisateur. (En prod on remonterait ça à un système dédié.)
	_, _ = database.Exec(
		`INSERT INTO logs (user_email, action, detail, ip) VALUES (?, ?, ?, ?)`,
		userEmail, action, detail, ip,
	)
}

// ListLogs renvoie les derniers événements (les plus récents d'abord).
func ListLogs(database *sql.DB, limit int) ([]LogEntry, error) {
	rows, err := database.Query(
		`SELECT id, created_at, user_email, action, detail, ip
		   FROM logs ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []LogEntry{}
	for rows.Next() {
		var l LogEntry
		var email, detail, ip sql.NullString
		if err := rows.Scan(&l.ID, &l.CreatedAt, &email, &l.Action, &detail, &ip); err != nil {
			return nil, err
		}
		l.UserEmail = email.String
		l.Detail = detail.String
		l.IP = ip.String
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

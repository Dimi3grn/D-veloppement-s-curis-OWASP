package db

import (
	"database/sql"
	"time"
)

// User représente une ligne de la table users (sans jamais exposer le hash au front).
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
}

// CreateUser insère un nouvel utilisateur et renvoie son ID.
// NOTE : on reçoit déjà le HASH (jamais le mot de passe en clair ici).
func CreateUser(database *sql.DB, email, passwordHash, role string) (int64, error) {
	// Requête PARAMÉTRÉE : les "?" sont des trous remplis séparément par le driver.
	// Même si email contient une attaque SQL, elle reste une simple donnée.
	res, err := database.Exec(
		`INSERT INTO users (email, password_hash, role) VALUES (?, ?, ?)`,
		email, passwordHash, role,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetUserByEmail récupère un utilisateur par son email, ou (nil, nil) s'il n'existe pas.
func GetUserByEmail(database *sql.DB, email string) (*User, error) {
	u := &User{}
	err := database.QueryRow(
		`SELECT id, email, password_hash, role, is_active, created_at
		   FROM users WHERE email = ?`, // encore un "?" : anti-injection
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil // pas trouvé, mais ce n'est pas une "erreur" technique
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ListUsers renvoie tous les utilisateurs (pour l'espace admin).
func ListUsers(database *sql.DB) ([]User, error) {
	rows, err := database.Query(
		`SELECT id, email, password_hash, role, is_active, created_at
		   FROM users ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// SetUserActive active (true) ou désactive (false) un compte.
func SetUserActive(database *sql.DB, id int64, active bool) error {
	_, err := database.Exec(`UPDATE users SET is_active = ? WHERE id = ?`, active, id)
	return err
}

// GetUserByID récupère un utilisateur par son ID (utile pour les sessions).
func GetUserByID(database *sql.DB, id int64) (*User, error) {
	u := &User{}
	err := database.QueryRow(
		`SELECT id, email, password_hash, role, is_active, created_at
		   FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

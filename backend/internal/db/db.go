// Package db gère la connexion à la base SQLite et la création des tables.
package db

import (
	"database/sql" // l'interface standard de Go pour parler à une base SQL

	_ "modernc.org/sqlite" // le driver SQLite. Le "_" signifie : on importe ce
	// paquet UNIQUEMENT pour son effet de bord (il s'enregistre tout seul comme
	// driver "sqlite" auprès de database/sql). On n'appelle aucune fonction dessus.
)

// schema décrit les tables. "IF NOT EXISTS" => on peut relancer sans erreur.
const schema = `
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT    NOT NULL UNIQUE,          -- identifiant de connexion
    password_hash TEXT    NOT NULL,                 -- le hash bcrypt, JAMAIS le mot de passe en clair
    role          TEXT    NOT NULL DEFAULT 'user',  -- 'user' ou 'admin'
    is_active     INTEGER NOT NULL DEFAULT 1,       -- 1 = actif, 0 = désactivé par l'admin
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,                    -- le PROPRIÉTAIRE de la note
    title      TEXT    NOT NULL,
    content    TEXT    NOT NULL,                    -- texte libre (terrain à XSS, sécurisé plus tard)
    visibility TEXT    NOT NULL DEFAULT 'private',  -- 'private' | 'shared' | 'public'
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)      -- lien d'intégrité vers users
);

-- Sessions stockées CÔTÉ SERVEUR. Le cookie du navigateur ne contient que l'id
-- (un token aléatoire). Avantage : on peut SUPPRIMER une ligne ici = déconnexion
-- immédiate / révocation (impossible avec un JWT auto-porteur).
CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT    PRIMARY KEY,                 -- token aléatoire (crypto/rand)
    user_id    INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,                  -- date d'expiration de la session
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Partage ciblé : quelle note est partagée avec quel utilisateur.
-- Une ligne = "la note X est accessible à l'utilisateur Y".
CREATE TABLE IF NOT EXISTS note_shares (
    note_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,                        -- le destinataire du partage
    PRIMARY KEY (note_id, user_id),                  -- pas de doublon
    FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE, -- supprimer la note nettoie les partages
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Journal des événements de SÉCURITÉ (A09 - Logging & Monitoring).
-- On y trace : connexions réussies/échouées, accès refusés, actions admin.
-- IMPORTANT : on n'y stocke JAMAIS de donnée sensible (pas de mot de passe).
CREATE TABLE IF NOT EXISTS logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_email TEXT,                                 -- qui (peut être inconnu)
    action     TEXT NOT NULL,                        -- ex: login_success, access_denied
    detail     TEXT,                                 -- précision lisible
    ip         TEXT                                  -- adresse IP de la requête
);
`

// Open ouvre la base (la crée si le fichier n'existe pas) et applique le schéma.
func Open(path string) (*sql.DB, error) {
	// sql.Open ne se connecte pas vraiment tout de suite ; il prépare le "pool".
	// Premier argument = nom du driver ("sqlite", enregistré par l'import plus haut).
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Ping force une vraie connexion pour vérifier que tout va bien.
	if err := database.Ping(); err != nil {
		return nil, err
	}

	// On active la vérification des clés étrangères (désactivée par défaut dans SQLite).
	if _, err := database.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, err
	}

	// On crée les tables si elles n'existent pas encore.
	if _, err := database.Exec(schema); err != nil {
		return nil, err
	}

	return database, nil
}

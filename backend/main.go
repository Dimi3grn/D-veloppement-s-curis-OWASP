package main

import (
	"database/sql"  // le type *sql.DB
	"encoding/json" // pour transformer des structures Go en JSON
	"log"           // pour afficher des messages dans la console
	"net/http"      // le serveur web (bibliothèque standard, zéro dépendance)
	"os"            // lecture des variables d'environnement
	"time"          // durées (fenêtre du rate-limiter)

	"notevault/internal/admin"      // espace admin + journaux
	"notevault/internal/auth"       // inscription, connexion, sessions
	"notevault/internal/db"         // notre package base de données
	"notevault/internal/middleware" // headers de sécurité, CORS, rate-limit
	"notevault/internal/notes"      // CRUD des notes + contrôle d'accès
)

// getenvDefault renvoie la variable d'environnement key, ou fallback si elle est vide.
func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// seedAdmin crée un compte administrateur au premier démarrage s'il n'en existe
// pas encore. Les identifiants viennent des variables d'environnement
// ADMIN_EMAIL / ADMIN_PASSWORD (avec un repli de dev) : AUCUN secret en dur
// dans le code versionné (bonne pratique A02/A05).
func seedAdmin(database *sql.DB) {
	adminEmail := getenvDefault("ADMIN_EMAIL", "admin@notevault.local")
	adminPassword := getenvDefault("ADMIN_PASSWORD", "changeme-dev-only")

	existing, err := db.GetUserByEmail(database, adminEmail)
	if err != nil {
		log.Println("seedAdmin: erreur de vérification :", err)
		return
	}
	if existing != nil {
		return // l'admin existe déjà, rien à faire
	}

	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		log.Println("seedAdmin: erreur de hachage :", err)
		return
	}
	if _, err := db.CreateUser(database, adminEmail, hash, "admin"); err != nil {
		log.Println("seedAdmin: erreur de création :", err)
		return
	}
	log.Printf("Compte admin créé => %s / %s (à changer en production)", adminEmail, adminPassword)
}

// healthHandler répond à la question "le serveur est-il vivant ?".
// w  = ce qu'on RENVOIE au navigateur (la réponse)
// r  = ce qu'on REÇOIT du navigateur (la requête)
func healthHandler(w http.ResponseWriter, r *http.Request) {
	// On annonce au navigateur que la réponse est du JSON.
	w.Header().Set("Content-Type", "application/json")

	// On écrit la réponse : {"status":"ok"}
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "NoteVault API",
	})
}

func main() {
	// On ouvre (et crée si besoin) la base de données SQLite.
	// Le fichier "notevault.db" sera créé à côté de l'exécutable.
	database, err := db.Open("notevault.db")
	if err != nil {
		log.Fatal("Impossible d'ouvrir la base de données : ", err)
	}
	defer database.Close() // sera fermée proprement à l'arrêt du programme
	log.Println("Base de données prête (notevault.db)")

	// On s'assure qu'un compte admin existe (créé au premier lancement).
	seedAdmin(database)

	// Un "mux" (multiplexeur) est l'aiguilleur : il regarde l'URL demandée
	// et appelle la bonne fonction. C'est le routeur intégré de Go.
	mux := http.NewServeMux()

	// "GET /api/health" => seules les requêtes GET sur cette URL déclenchent
	// healthHandler. Préciser la méthode (GET) est déjà une bonne pratique.
	mux.HandleFunc("GET /api/health", healthHandler)

	// Routes d'authentification. On donne la base de données au handler.
	// SecureCookies=false en dev local (HTTP). À passer à true en production (HTTPS).
	authHandler := &auth.Handler{DB: database, SecureCookies: false}
	mux.HandleFunc("POST /api/register", authHandler.Register)

	// Le login est protégé par un rate-limiter : max 5 tentatives / minute / IP.
	loginLimiter := middleware.NewLoginLimiter(5, time.Minute)
	mux.HandleFunc("POST /api/login", loginLimiter.Middleware(authHandler.Login))
	mux.HandleFunc("POST /api/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/me", authHandler.Me)

	// Routes des notes. CHAQUE route est emballée dans RequireAuth :
	// impossible d'y accéder sans session valide (deny by default).
	notesHandler := &notes.Handler{DB: database}
	mux.HandleFunc("POST /api/notes", authHandler.RequireAuth(notesHandler.Create))
	mux.HandleFunc("GET /api/notes", authHandler.RequireAuth(notesHandler.List))
	mux.HandleFunc("GET /api/notes/shared", authHandler.RequireAuth(notesHandler.ListShared))
	mux.HandleFunc("GET /api/notes/public", authHandler.RequireAuth(notesHandler.ListPublic))
	mux.HandleFunc("GET /api/notes/{id}", authHandler.RequireAuth(notesHandler.Get))
	mux.HandleFunc("PUT /api/notes/{id}", authHandler.RequireAuth(notesHandler.Update))
	mux.HandleFunc("DELETE /api/notes/{id}", authHandler.RequireAuth(notesHandler.Delete))
	mux.HandleFunc("POST /api/notes/{id}/share", authHandler.RequireAuth(notesHandler.Share))
	mux.HandleFunc("DELETE /api/notes/{id}/share", authHandler.RequireAuth(notesHandler.Unshare))

	// Routes ADMIN. Toutes emballées dans RequireAdmin :
	// session valide + rôle "admin" obligatoires (sinon 403 + log).
	adminHandler := &admin.Handler{DB: database}
	mux.HandleFunc("GET /api/admin/users", authHandler.RequireAdmin(adminHandler.ListUsers))
	mux.HandleFunc("POST /api/admin/users/{id}/disable", authHandler.RequireAdmin(adminHandler.DisableUser))
	mux.HandleFunc("POST /api/admin/users/{id}/enable", authHandler.RequireAdmin(adminHandler.EnableUser))
	mux.HandleFunc("GET /api/admin/logs", authHandler.RequireAdmin(adminHandler.ListLogs))

	// On enveloppe TOUT le routeur dans les middlewares de sécurité :
	// chaque réponse passe par CORS (origine du front autorisée) puis par
	// SecurityHeaders (en-têtes de sécurité ajoutés à toutes les réponses).
	handler := middleware.SecurityHeaders(
		middleware.CORS("http://localhost:5173", mux),
	)

	// On démarre le serveur sur le port 8080.
	// log.Fatal arrête le programme et affiche l'erreur si le serveur plante.
	log.Println("Serveur NoteVault démarré sur http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

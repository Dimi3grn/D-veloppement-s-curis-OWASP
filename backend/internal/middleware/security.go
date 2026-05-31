// Package middleware regroupe les protections transversales (headers, CORS, rate-limit).
package middleware

import "net/http"

// SecurityHeaders ajoute des en-têtes de sécurité à TOUTES les réponses (A05).
// Chaque en-tête ferme une porte d'attaque précise.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// Empêche le navigateur de "deviner" le type d'un fichier (MIME sniffing),
		// une technique utilisée pour faire exécuter un fichier déguisé.
		h.Set("X-Content-Type-Options", "nosniff")

		// Interdit d'afficher le site dans une <iframe> => anti-clickjacking
		// (un attaquant ne peut pas piéger l'utilisateur en superposant notre site).
		h.Set("X-Frame-Options", "DENY")

		// Ne transmet pas l'URL d'origine quand on suit un lien vers un autre site.
		h.Set("Referrer-Policy", "no-referrer")

		// Content-Security-Policy : liste blanche des sources autorisées.
		// Ici on n'autorise que notre propre origine et on interdit l'embarquement.
		// (C'est une 2e barrière anti-XSS, en plus du nettoyage DOMPurify.)
		h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'")

		// En production HTTPS, on ajouterait aussi HSTS :
		// h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		next.ServeHTTP(w, r)
	})
}

// CORS n'autorise QUE l'origine du front (liste blanche), cookies compris.
//
// Rappel (A05) : par défaut, un navigateur interdit à un site A d'appeler l'API
// d'un site B. CORS permet d'ouvrir CETTE porte, mais on l'ouvre le moins possible :
// une seule origine autorisée, jamais "*".
func CORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == allowedOrigin {
			h := w.Header()
			// On renvoie l'origine précise (JAMAIS "*" : interdit avec des cookies).
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true") // autorise l'envoi du cookie
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			h.Set("Vary", "Origin")
		}
		// Requête de "préflight" (OPTIONS) : le navigateur demande la permission
		// avant d'envoyer la vraie requête. On répond tout de suite.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

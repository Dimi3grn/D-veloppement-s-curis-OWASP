package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// LoginLimiter limite le nombre de tentatives de connexion PAR ADRESSE IP
// sur une fenêtre de temps (protection A07/A04 contre la force brute).
//
// Principe : on garde, pour chaque IP, la liste des horodatages de ses tentatives
// récentes. Si elle dépasse le seuil dans la fenêtre, on refuse (429).
type LoginLimiter struct {
	mu     sync.Mutex            // protège la map contre les accès concurrents
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		hits:   make(map[string][]time.Time),
		max:    max,
		window: window,
	}
}

// clientIP extrait l'adresse IP (sans le port) de la requête.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// allow enregistre une tentative pour cette IP et indique si elle est autorisée.
func (l *LoginLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// On ne conserve que les tentatives encore dans la fenêtre de temps.
	recent := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= l.max {
		l.hits[ip] = recent
		return false // trop de tentatives
	}
	l.hits[ip] = append(recent, now)
	return true
}

// Middleware enveloppe un handler (typiquement le login) avec la limitation.
func (l *LoginLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests) // 429
			w.Write([]byte(`{"error":"Trop de tentatives. Réessayez dans une minute."}`))
			return
		}
		next(w, r)
	}
}

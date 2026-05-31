// Package notes gère le CRUD des notes et le contrôle d'accès (A01).
package notes

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"notevault/internal/auth"
	"notevault/internal/db"
)

type Handler struct {
	DB *sql.DB
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// validVisibility vérifie que la visibilité fait partie des valeurs autorisées.
func validVisibility(v string) bool {
	return v == "private" || v == "shared" || v == "public"
}

type noteRequest struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	Visibility string `json:"visibility"`
}

// Create : POST /api/notes — crée une note appartenant à l'utilisateur connecté.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r) // garanti présent grâce au middleware RequireAuth

	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Requête invalide")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		writeError(w, http.StatusBadRequest, "Titre requis (max 200 caractères)")
		return
	}
	if len(req.Content) > 50000 {
		writeError(w, http.StatusBadRequest, "Contenu trop long")
		return
	}
	if req.Visibility == "" {
		req.Visibility = "private" // par défaut : privé (moindre exposition)
	}
	if !validVisibility(req.Visibility) {
		writeError(w, http.StatusBadRequest, "Visibilité invalide")
		return
	}

	// On force user.ID comme propriétaire : impossible de créer une note "au nom"
	// de quelqu'un d'autre, car on ne fait pas confiance à un éventuel champ du front.
	id, err := db.CreateNote(h.DB, user.ID, req.Title, req.Content, req.Visibility)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

// List : GET /api/notes — renvoie UNIQUEMENT les notes de l'utilisateur connecté.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)

	// Le filtre se fait sur user.ID (l'identité prouvée par la session),
	// JAMAIS sur un paramètre fourni par le client => pas d'IDOR possible ici.
	notes, err := db.ListNotesByUser(h.DB, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

// Get : GET /api/notes/{id} — récupère UNE note, avec contrôle d'accès.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Identifiant invalide")
		return
	}

	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	if note == nil {
		writeError(w, http.StatusNotFound, "Note introuvable")
		return
	}

	// >>> CONTRÔLE D'ACCÈS (A01) <<<
	allowed, err := h.canRead(note, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	// 404 (et non 403) si pas autorisé : on ne révèle pas l'existence de la note.
	if !allowed {
		writeError(w, http.StatusNotFound, "Note introuvable")
		return
	}

	writeJSON(w, http.StatusOK, note)
}

// canRead centralise la règle d'accès en LECTURE d'une note (A01).
// Autorisé si : propriétaire OU note publique OU (note partagée ET partagée avec moi).
func (h *Handler) canRead(note *db.Note, userID int64) (bool, error) {
	if note.UserID == userID {
		return true, nil // le propriétaire a toujours accès
	}
	if note.Visibility == "public" {
		return true, nil // tout le monde (connecté) peut lire une note publique
	}
	if note.Visibility == "shared" {
		// accès seulement si la note a été explicitement partagée avec cet utilisateur
		return db.IsNoteSharedWith(h.DB, note.ID, userID)
	}
	return false, nil // "private" et pas propriétaire => refus
}

// Update : PUT /api/notes/{id} — modifie une note, RÉSERVÉ au propriétaire.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Identifiant invalide")
		return
	}

	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	// CONTRÔLE D'ACCÈS : seul le propriétaire peut modifier. Sinon 404 (on ne révèle rien).
	if note == nil || note.UserID != user.ID {
		writeError(w, http.StatusNotFound, "Note introuvable")
		return
	}

	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Requête invalide")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 200 {
		writeError(w, http.StatusBadRequest, "Titre requis (max 200 caractères)")
		return
	}
	if !validVisibility(req.Visibility) {
		writeError(w, http.StatusBadRequest, "Visibilité invalide")
		return
	}

	if err := db.UpdateNote(h.DB, id, req.Title, req.Content, req.Visibility); err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Note mise à jour"})
}

// Delete : DELETE /api/notes/{id} — supprime une note, RÉSERVÉ au propriétaire.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Identifiant invalide")
		return
	}

	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	// CONTRÔLE D'ACCÈS : seul le propriétaire peut supprimer.
	if note == nil || note.UserID != user.ID {
		writeError(w, http.StatusNotFound, "Note introuvable")
		return
	}

	if err := db.DeleteNote(h.DB, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Note supprimée"})
}

type shareRequest struct {
	Email string `json:"email"`
}

// Share : POST /api/notes/{id}/share — le PROPRIÉTAIRE partage sa note avec un email.
func (h *Handler) Share(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Identifiant invalide")
		return
	}

	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	// CONTRÔLE D'ACCÈS : seul le propriétaire peut décider de partager sa note.
	if note == nil || note.UserID != user.ID {
		writeError(w, http.StatusNotFound, "Note introuvable")
		return
	}

	var req shareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Requête invalide")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	target, err := db.GetUserByEmail(h.DB, req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	if target == nil {
		writeError(w, http.StatusNotFound, "Utilisateur introuvable")
		return
	}
	if target.ID == user.ID {
		writeError(w, http.StatusBadRequest, "Vous ne pouvez pas partager avec vous-même")
		return
	}

	if err := db.ShareNoteWithUser(h.DB, note.ID, target.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	// On bascule la note en "shared" pour que le partage prenne effet.
	if note.Visibility != "shared" {
		if err := db.UpdateNote(h.DB, note.ID, note.Title, note.Content, "shared"); err != nil {
			writeError(w, http.StatusInternalServerError, "Erreur serveur")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Note partagée avec " + target.Email})
}

// Unshare : DELETE /api/notes/{id}/share — le propriétaire retire un partage.
func (h *Handler) Unshare(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Identifiant invalide")
		return
	}

	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	if note == nil || note.UserID != user.ID {
		writeError(w, http.StatusNotFound, "Note introuvable")
		return
	}

	var req shareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Requête invalide")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	target, err := db.GetUserByEmail(h.DB, req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	if target == nil {
		writeError(w, http.StatusNotFound, "Utilisateur introuvable")
		return
	}

	if err := db.UnshareNoteWithUser(h.DB, note.ID, target.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Partage retiré"})
}

// ListPublic : GET /api/notes/public — toutes les notes publiques (visibles par tous).
func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)

	notes, err := db.ListPublicNotes(h.DB, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

// ListShared : GET /api/notes/shared — les notes que d'AUTRES ont partagées avec moi.
func (h *Handler) ListShared(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)

	notes, err := db.ListNotesSharedWithUser(h.DB, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

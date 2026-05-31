// Couche d'accès à l'API. TOUT passe par ici => un seul endroit à comprendre/maintenir.

// request() est l'appel générique. Les autres fonctions s'appuient dessus.
async function request(path, { method = 'GET', body } = {}) {
  const res = await fetch(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    // credentials:'include' => le navigateur joint le COOKIE de session à la requête.
    // Sans ça, le serveur ne saurait pas qui nous sommes.
    credentials: 'include',
    body: body ? JSON.stringify(body) : undefined,
  })

  // On essaie de lire le JSON (peut être vide sur certaines réponses).
  const data = await res.json().catch(() => null)

  if (!res.ok) {
    // On propage le message d'erreur renvoyé par le backend.
    throw new Error(data?.error || 'Erreur réseau')
  }
  return data
}

// --- Authentification ---
export const register = (email, password) =>
  request('/api/register', { method: 'POST', body: { email, password } })

export const login = (email, password) =>
  request('/api/login', { method: 'POST', body: { email, password } })

export const logout = () => request('/api/logout', { method: 'POST' })

export const me = () => request('/api/me')

// --- Notes ---
export const listNotes = () => request('/api/notes')
export const listShared = () => request('/api/notes/shared')
export const listPublic = () => request('/api/notes/public')
export const getNote = (id) => request(`/api/notes/${id}`)

export const createNote = (note) =>
  request('/api/notes', { method: 'POST', body: note })

export const updateNote = (id, note) =>
  request(`/api/notes/${id}`, { method: 'PUT', body: note })

export const deleteNote = (id) =>
  request(`/api/notes/${id}`, { method: 'DELETE' })

// --- Partage ---
export const shareNote = (id, email) =>
  request(`/api/notes/${id}/share`, { method: 'POST', body: { email } })

export const unshareNote = (id, email) =>
  request(`/api/notes/${id}/share`, { method: 'DELETE', body: { email } })

// --- Admin (réservé au rôle admin côté serveur) ---
export const listAdminUsers = () => request('/api/admin/users')
export const disableUser = (id) => request(`/api/admin/users/${id}/disable`, { method: 'POST' })
export const enableUser = (id) => request(`/api/admin/users/${id}/enable`, { method: 'POST' })
export const listLogs = () => request('/api/admin/logs')

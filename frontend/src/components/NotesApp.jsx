import { useState, useEffect } from 'react'
import * as api from '../api/client'
import NoteEditor from './NoteEditor'
import Admin from './Admin'

export default function NotesApp({ user, onLogout }) {
  const [myNotes, setMyNotes] = useState([])
  const [sharedNotes, setSharedNotes] = useState([])
  const [publicNotes, setPublicNotes] = useState([])
  const [selected, setSelected] = useState(null)
  const [creating, setCreating] = useState(false)
  const [view, setView] = useState('notes') // 'notes' ou 'admin'

  async function refresh() {
    const [mine, shared, pub] = await Promise.all([api.listNotes(), api.listShared(), api.listPublic()])
    setMyNotes(mine)
    setSharedNotes(shared)
    setPublicNotes(pub)
  }

  useEffect(() => { refresh() }, [])

  async function handleLogout() {
    await api.logout()
    onLogout()
  }

  function openNote(note) { setCreating(false); setSelected(note) }
  function closeEditor() { setSelected(null); setCreating(false) }
  async function afterChange() { await refresh(); closeEditor() }

  const canEditSelected = selected && selected.user_id === user.id

  return (
    <div>
      <header className="topbar">
        <span className="brand">NoteVault</span>
        <div className="topbar-right">
          {user.role === 'admin' && (
            <>
              <button className="link" onClick={() => setView('notes')}>Notes</button>
              <button className="link" onClick={() => setView('admin')}>Admin</button>
            </>
          )}
          <span className="user">{user.email}{user.role === 'admin' && ' · admin'}</span>
          <button className="link" onClick={handleLogout}>Déconnexion</button>
        </div>
      </header>

      {view === 'admin' && user.role === 'admin' ? (
        <div className="content"><Admin currentUser={user} /></div>
      ) : (
      <div className="layout">
        <aside className="sidebar">
          <button className="new-btn" onClick={() => { setSelected(null); setCreating(true) }}>
            + Nouvelle note
          </button>

          <div className="col-label">Mes notes</div>
          {myNotes.length === 0 && <p className="empty">Aucune note.</p>}
          <ul>
            {myNotes.map((n) => (
              <li key={n.id} className={selected?.id === n.id ? 'active' : ''} onClick={() => openNote(n)}>
                <span className="note-title">{n.title}</span>
                <span className="badge">{visibilityMark(n.visibility)}</span>
              </li>
            ))}
          </ul>

          <div className="col-label">Partagées avec moi</div>
          {sharedNotes.length === 0 && <p className="empty">Rien pour l'instant.</p>}
          <ul>
            {sharedNotes.map((n) => (
              <li key={n.id} className={selected?.id === n.id ? 'active' : ''} onClick={() => openNote(n)}>
                <span className="note-title">{n.title}</span>
              </li>
            ))}
          </ul>

          <div className="col-label">Notes publiques</div>
          {publicNotes.length === 0 && <p className="empty">Aucune note publique.</p>}
          <ul>
            {publicNotes.map((n) => (
              <li key={n.id} className={selected?.id === n.id ? 'active' : ''} onClick={() => openNote(n)}>
                <span className="note-title">{n.title}</span>
              </li>
            ))}
          </ul>
        </aside>

        <main className="content">
          {creating && (
            <NoteEditor note={null} canEdit={true} onSaved={afterChange} onDeleted={afterChange} onClose={closeEditor} />
          )}
          {selected && (
            <NoteEditor
              key={selected.id}
              note={selected}
              canEdit={canEditSelected}
              onSaved={afterChange}
              onDeleted={afterChange}
              onClose={closeEditor}
            />
          )}
          {!creating && !selected && (
            <div className="placeholder">Sélectionnez une note ou créez-en une nouvelle.</div>
          )}
        </main>
      </div>
      )}
    </div>
  )
}

// Petit indicateur sobre de visibilité (texte discret, pas d'emoji criard).
function visibilityMark(v) {
  if (v === 'public') return 'publique'
  if (v === 'shared') return 'partagée'
  return ''
}

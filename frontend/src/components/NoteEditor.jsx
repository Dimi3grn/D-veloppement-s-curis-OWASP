import { useState } from 'react'
import * as api from '../api/client'
import { renderMarkdown, renderMarkdownUNSAFE } from '../lib/markdown'

const VISIBILITY_LABEL = {
  private: 'Privée',
  shared: 'Partagée',
  public: 'Publique',
}

// NoteEditor : affiche une note en LECTURE (markdown rendu) et permet de passer
// en ÉDITION (markdown brut) si on en est le propriétaire.
export default function NoteEditor({ note, canEdit, onSaved, onDeleted, onClose }) {
  const isNew = !note
  // Nouvelle note => on démarre en édition. Note existante => on démarre en lecture.
  const [mode, setMode] = useState(isNew ? 'edit' : 'read')
  const [title, setTitle] = useState(note?.title || '')
  const [content, setContent] = useState(note?.content || '')
  const [visibility, setVisibility] = useState(note?.visibility || 'private')
  const [shareEmail, setShareEmail] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  async function handleSave() {
    setError(''); setMessage('')
    try {
      if (isNew) await api.createNote({ title, content, visibility })
      else await api.updateNote(note.id, { title, content, visibility })
      onSaved()
    } catch (err) { setError(err.message) }
  }

  async function handleDelete() {
    if (!confirm('Supprimer cette note ?')) return
    try { await api.deleteNote(note.id); onDeleted() }
    catch (err) { setError(err.message) }
  }

  async function handleShare() {
    setError(''); setMessage('')
    try {
      const res = await api.shareNote(note.id, shareEmail)
      setMessage(res.message)
      setShareEmail('')
      setVisibility('shared')
    } catch (err) { setError(err.message) }
  }

  // ---------- MODE LECTURE ----------
  if (mode === 'read') {
    return (
      <>
        <div className="doc">
          <div className="doc-head">
            <h1 className="doc-title">{title}</h1>
            <div className="doc-actions">
              {canEdit && <button className="ghost" onClick={() => setMode('edit')}>Éditer</button>}
              <button className="link" onClick={onClose}>Fermer</button>
            </div>
          </div>
          <div className="doc-meta">{VISIBILITY_LABEL[note.visibility]}</div>

          {/* RENDU MARKDOWN. dangerouslySetInnerHTML insère du HTML : c'est sûr
              UNIQUEMENT parce que renderMarkdown() a nettoyé le contenu avec
              DOMPurify (protection XSS - A03). Sans ce nettoyage, ce serait une faille. */}
          <div
            className="markdown"
            dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
          />
        </div>

        {canEdit && (
          <div className="share-box">
            <h3>Partager avec un utilisateur</h3>
            <div className="share-row">
              <input type="email" placeholder="email@exemple.com" value={shareEmail} onChange={(e) => setShareEmail(e.target.value)} />
              <button onClick={handleShare}>Partager</button>
            </div>
            {error && <p className="error">{error}</p>}
            {message && <p className="success">{message}</p>}
          </div>
        )}
      </>
    )
  }

  // ---------- MODE ÉDITION ----------
  return (
    <div className="doc">
      <div className="doc-head">
        <input
          className="doc-title-input"
          placeholder="Titre de la note"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          maxLength={200}
        />
        <div className="doc-actions">
          <button onClick={handleSave}>Enregistrer</button>
          <button className="link" onClick={onClose}>Fermer</button>
        </div>
      </div>

      <div className="edit-toolbar">
        <span className="hint">Markdown : # titre · **gras** · - liste · `code`</span>
      </div>

      <textarea
        className="md-editor"
        placeholder="Écrivez en Markdown…"
        value={content}
        onChange={(e) => setContent(e.target.value)}
      />

      <div className="visibility-row">
        <label style={{ margin: 0 }}>Visibilité</label>
        <select value={visibility} onChange={(e) => setVisibility(e.target.value)}>
          <option value="private">Privée (vous seul)</option>
          <option value="shared">Partagée (personnes choisies)</option>
          <option value="public">Publique (tous les connectés)</option>
        </select>
        {!isNew && <button className="danger" onClick={handleDelete} style={{ marginLeft: 'auto' }}>Supprimer</button>}
      </div>

      {error && <p className="error">{error}</p>}
    </div>
  )
}

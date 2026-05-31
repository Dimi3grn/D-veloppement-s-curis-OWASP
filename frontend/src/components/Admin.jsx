import { useState, useEffect } from 'react'
import * as api from '../api/client'
import './admin.css'

// Traduction lisible des actions du journal.
const ACTION_LABEL = {
  login_success: 'connexion réussie',
  login_failed: 'connexion échouée',
  access_denied: 'accès refusé',
  account_disabled: 'compte désactivé',
  account_enabled: 'compte réactivé',
}

export default function Admin({ currentUser }) {
  const [users, setUsers] = useState([])
  const [logs, setLogs] = useState([])
  const [error, setError] = useState('')

  async function refresh() {
    setError('')
    try {
      const [u, l] = await Promise.all([api.listAdminUsers(), api.listLogs()])
      setUsers(u)
      setLogs(l)
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => { refresh() }, [])

  async function toggleActive(u) {
    try {
      if (u.is_active) await api.disableUser(u.id)
      else await api.enableUser(u.id)
      await refresh()
    } catch (e) {
      setError(e.message)
    }
  }

  function actionClass(action) {
    if (action === 'login_failed' || action === 'access_denied' || action === 'account_disabled') return 'admin-action bad'
    if (action === 'login_success' || action === 'account_enabled') return 'admin-action good'
    return 'admin-action'
  }

  return (
    <div className="doc admin">
      <h1 className="doc-title">Administration</h1>
      <div className="doc-meta">Gestion des comptes et journal de sécurité</div>
      {error && <p className="error">{error}</p>}

      <h2 className="admin-h2">Comptes ({users.length})</h2>
      <table className="admin-table">
        <thead>
          <tr><th>Email</th><th>Rôle</th><th>État</th><th></th></tr>
        </thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.id}>
              <td>{u.email}</td>
              <td>{u.role}</td>
              <td className={u.is_active ? 'state-on' : 'state-off'}>
                {u.is_active ? 'actif' : 'désactivé'}
              </td>
              <td>
                {u.id !== currentUser.id ? (
                  <button className={u.is_active ? 'danger' : 'ghost'} onClick={() => toggleActive(u)}>
                    {u.is_active ? 'Désactiver' : 'Réactiver'}
                  </button>
                ) : (
                  <span className="admin-ip">vous</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2 className="admin-h2">Journal de sécurité ({logs.length})</h2>
      <table className="admin-table">
        <thead>
          <tr><th>Date</th><th>Utilisateur</th><th>Action</th><th>Détail</th><th>IP</th></tr>
        </thead>
        <tbody>
          {logs.map((l) => (
            <tr key={l.id}>
              <td>{new Date(l.created_at).toLocaleString('fr-FR')}</td>
              <td>{l.user_email || '—'}</td>
              <td><span className={actionClass(l.action)}>{ACTION_LABEL[l.action] || l.action}</span></td>
              <td>{l.detail}</td>
              <td className="admin-ip">{l.ip}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

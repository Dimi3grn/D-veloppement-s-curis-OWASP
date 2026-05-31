import { useState, useEffect } from 'react'
import * as api from './api/client'
import Auth from './components/Auth'
import NotesApp from './components/NotesApp'
import './App.css'

export default function App() {
  const [user, setUser] = useState(null)   // l'utilisateur connecté (null = déconnecté)
  const [loading, setLoading] = useState(true)

  // Au chargement, on demande au backend "qui suis-je ?" via le cookie de session.
  // Si une session valide existe déjà, on reste connecté (pas besoin de re-login).
  useEffect(() => {
    api.me()
      .then((u) => setUser(u))
      .catch(() => setUser(null)) // pas connecté : normal
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="center">Chargement...</div>

  // Aiguillage : connecté => l'app de notes, sinon => l'écran d'authentification.
  return user
    ? <NotesApp user={user} onLogout={() => setUser(null)} />
    : <Auth onLogin={setUser} />
}

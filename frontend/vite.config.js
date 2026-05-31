import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // PROXY DE DÉVELOPPEMENT
    // Toute requête du front commençant par "/api" est transférée
    // au backend Go (http://localhost:8080) SANS que le navigateur
    // ne voie un changement d'origine. Ça évite les problèmes CORS
    // pendant le développement. En production, c'est un reverse proxy
    // (ex: Nginx) qui jouera ce rôle.
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})

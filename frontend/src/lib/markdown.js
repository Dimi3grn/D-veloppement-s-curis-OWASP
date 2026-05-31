import { marked } from 'marked'
import DOMPurify from 'dompurify'

// renderMarkdown transforme du Markdown en HTML SÛR à afficher.
//
// Le problème (A03 - XSS) :
//   Le Markdown autorise le HTML brut. Donc si un utilisateur écrit
//   <img src=x onerror="alert(document.cookie)">  ou  <script>...</script>
//   dans sa note, marked() le transforme en HTML actif. Affiché tel quel,
//   ce code S'EXÉCUTERAIT dans le navigateur du lecteur = faille XSS.
//
// La protection :
//   DOMPurify.sanitize() analyse le HTML et SUPPRIME tout ce qui est dangereux
//   (balises <script>, attributs onerror/onclick, liens javascript:, etc.),
//   en gardant le HTML inoffensif (titres, listes, gras, liens http...).
export function renderMarkdown(md) {
  const rawHtml = marked.parse(md || '')   // 1) Markdown -> HTML (peut contenir du danger)
  const safeHtml = DOMPurify.sanitize(rawHtml) // 2) on retire tout ce qui est dangereux
  return safeHtml
}

// ⚠️ VERSION VOLONTAIREMENT VULNÉRABLE — NE PAS UTILISER EN PRODUCTION.
// Sert uniquement à DÉMONTRER la faille XSS à l'oral (étape 6) :
// on affiche le HTML sans le nettoyer => le code injecté s'exécute.
export function renderMarkdownUNSAFE(md) {
  return marked.parse(md || '') // pas de DOMPurify => XSS possible
}

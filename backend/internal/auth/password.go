// Package auth gère l'inscription, la connexion, les sessions et les mots de passe.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword transforme un mot de passe en clair en un hash bcrypt.
//
// Pourquoi bcrypt (protection A02 - Cryptographic Failures) :
//   - Il intègre un SEL aléatoire unique => deux personnes avec le même mot de
//     passe ont des hash DIFFÉRENTS (impossible de comparer / d'utiliser des
//     "rainbow tables").
//   - Il est LENT volontairement (facteur de coût) => casser par force brute
//     coûte très cher à l'attaquant.
//   - On ne peut PAS revenir au mot de passe d'origine depuis le hash (à sens unique).
func HashPassword(plain string) (string, error) {
	// DefaultCost (=10) = bon compromis sécurité/vitesse. Plus haut = plus sûr mais plus lent.
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword vérifie qu'un mot de passe en clair correspond à un hash stocké.
// bcrypt re-hache le mot de passe fourni (avec le sel contenu dans le hash) et compare.
// Renvoie true si ça correspond.
func CheckPassword(hash, plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}

// dummyHash : un hash bcrypt valide "bidon", calculé une seule fois au démarrage.
//
// À quoi ça sert (protection A07 - anti-énumération d'utilisateurs) :
// quand quelqu'un tente de se connecter avec un email QUI N'EXISTE PAS, on lance
// quand même une vérification bcrypt contre ce hash bidon. Pourquoi ? Pour que le
// temps de réponse soit IDENTIQUE, que l'email existe ou non. Sinon, un attaquant
// mesurant le temps de réponse pourrait deviner quels emails sont enregistrés.
var dummyHashBytes, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)
var dummyHash = string(dummyHashBytes)

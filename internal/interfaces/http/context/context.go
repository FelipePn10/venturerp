package contextkey

type UserKeyType struct{}

var UserKey = UserKeyType{}

// SessionKeyType carrega o início da sessão lido do token. É o que o "manter
// conectado" precisa para renovar a sessão sem estender o prazo para sempre:
// a renovação conta o teto a partir do login de verdade, não da última renovação.
type SessionKeyType struct{}

var SessionKey = SessionKeyType{}

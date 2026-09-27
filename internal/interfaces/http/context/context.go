package contextkey

type UserKeyType struct{}

var UserKey = UserKeyType{}

// SessionKeyType carrega o início da sessão lido do token. É o que o "manter
// conectado" precisa para renovar a sessão sem estender o prazo para sempre:
// a renovação conta o teto a partir do login de verdade, não da última renovação.
type SessionKeyType struct{}

var SessionKey = SessionKeyType{}

// ItemCodeTranslatedKeyType marca que o middleware de compatibilidade JÁ trocou
// o código público do item pela chave interna no corpo/query da requisição.
//
// Sem essa marca, o caso de uso resolve o código de novo — e, numa base em que
// códigos comerciais numéricos convivem com chaves internas ("1", "5", "8" ao
// lado dos internos 1..29), a chave interna 5 é reinterpretada como o código
// comercial "5" e vira outro item. O lançamento responde 201 gravando no item
// ERRADO. Foi assim na base de produção da Tecnofer: um orçamento de RN-01001
// gravava CHAPA AÇO CARBONO 3MM.
type ItemCodeTranslatedKeyType struct{}

var ItemCodeTranslatedKey = ItemCodeTranslatedKeyType{}

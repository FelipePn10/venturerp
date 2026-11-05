package contextkey

import (
	"context"
	"strings"
)

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

// translatedItemCodesKeyType guarda QUAIS chaves internas o middleware de
// compatibilidade produziu nesta requisição — corpo, query string e caminho.
//
// A marca booleana acima diz "algo foi traduzido", e isso não basta. Ela só era
// ligada quando a tradução acontecia no CORPO: na query string e no caminho a
// tradução acontecia igual, mas sem marca. Na Tecnofer, `GET /api/routing/routes
// ?item_code=RN-01001` virava `item_code=5`, o caso de uso relia o 5 como o
// comercial "5" (CHAPA AÇO CARBONO 3MM) e a lista de roteiros da VENT0202 vinha
// vazia — o usuário recriava a alternativa 1, que já existia, e levava 409.
//
// E uma marca única para a requisição inteira erra para o outro lado: um campo
// que o middleware deixa de propósito sem tradução (parent_code/child_code da
// estrutura, item-base do cadastro) passaria a ser lido como chave interna. Por
// isso o registro é por VALOR: só o valor que o middleware produziu é chave.
type translatedItemCodesKeyType struct{}

var translatedItemCodesKey = translatedItemCodesKeyType{}

// WithTranslatedItemCodes devolve um contexto que registra os códigos dados como
// chaves internas já traduzidas, somando-os aos que já estavam registrados.
// Também é o jeito certo de um caso de uso repassar a outro uma chave interna que
// leu do banco (ex.: o item da venda recorrente vira item do pedido).
func WithTranslatedItemCodes(ctx context.Context, codes ...string) context.Context {
	if len(codes) == 0 {
		return ctx
	}
	anterior, _ := ctx.Value(translatedItemCodesKey).(map[string]struct{})
	novo := make(map[string]struct{}, len(anterior)+len(codes))
	for c := range anterior {
		novo[c] = struct{}{}
	}
	for _, c := range codes {
		if c = strings.TrimSpace(c); c != "" {
			novo[c] = struct{}{}
		}
	}
	return context.WithValue(ctx, translatedItemCodesKey, novo)
}

// IsTranslatedItemCode diz se o código deve ser lido como CHAVE INTERNA: o
// middleware o produziu nesta requisição, ou o chamador ligou a marca geral
// (ItemCodeTranslatedKey) para o contexto inteiro.
func IsTranslatedItemCode(ctx context.Context, code string) bool {
	if marcado, _ := ctx.Value(ItemCodeTranslatedKey).(bool); marcado {
		return true
	}
	codes, _ := ctx.Value(translatedItemCodesKey).(map[string]struct{})
	_, ok := codes[strings.TrimSpace(code)]
	return ok
}

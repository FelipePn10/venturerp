package contextkey

import (
	"context"
	"testing"
)

func TestRegistroDeChaveTraduzidaEhPorValor(t *testing.T) {
	ctx := WithTranslatedItemCodes(context.Background(), "5", " 12 ")
	if !IsTranslatedItemCode(ctx, "5") || !IsTranslatedItemCode(ctx, "12") {
		t.Fatal("as chaves registradas deveriam ser reconhecidas (com ou sem espaços)")
	}
	if IsTranslatedItemCode(ctx, "2") {
		t.Fatal("chave não registrada não pode ser lida como chave interna")
	}
	// Somar um registro não altera o contexto anterior (o mapa é copiado).
	depois := WithTranslatedItemCodes(ctx, "2")
	if !IsTranslatedItemCode(depois, "2") || !IsTranslatedItemCode(depois, "5") {
		t.Fatal("o novo registro deveria somar ao anterior")
	}
	if IsTranslatedItemCode(ctx, "2") {
		t.Fatal("o contexto anterior não pode ver o registro feito depois")
	}
	if got := WithTranslatedItemCodes(ctx); got != ctx {
		t.Fatal("sem códigos, o contexto volta igual")
	}
}

func TestMarcaGeralContinuaValendoParaTudo(t *testing.T) {
	ctx := context.WithValue(context.Background(), ItemCodeTranslatedKey, true)
	if !IsTranslatedItemCode(ctx, "qualquer") {
		t.Fatal("a marca geral (demanda/ordem planejada) vale para qualquer código")
	}
	if IsTranslatedItemCode(context.Background(), "5") {
		t.Fatal("sem marca nem registro, nada é chave interna")
	}
}

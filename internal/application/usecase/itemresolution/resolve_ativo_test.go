package itemresolution

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type repoDeItens struct{ itens map[string]*itementity.Item }

func (r repoDeItens) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*itementity.Item, error) {
	if item, ok := r.itens[string(code)]; ok {
		return item, nil
	}
	return nil, itemrepo.ErrNotFound
}

func item(codigo, nome string, saude types.Health) *itementity.Item {
	bc, _ := valueobject.NewBusinessCode(codigo)
	return &itementity.Item{Code: 1, BusinessCode: bc, Name: nome, Health: saude}
}

// Esconder o item inativo na tela não basta: integração, importação de planilha
// e uma janela aberta antes da inativação chegam pelo mesmo endpoint.
func TestResolveActiveRecusaItemInativo(t *testing.T) {
	repo := repoDeItens{itens: map[string]*itementity.Item{
		"RN-01001": item("RN-01001", "CHAPA GALVANIZADA", types.ACTIVE),
		"RN-09999": item("RN-09999", "PERFIL DESCONTINUADO", types.INACTIVE),
		"BU-050":   item("BU-050", "BUCHA FANTASMA", types.GHOST),
	}}
	ctx := context.Background()

	if _, err := ResolveActive(ctx, repo, request.TextCode("RN-01001")); err != nil {
		t.Fatalf("item ativo foi recusado: %v", err)
	}

	// FANTASMA é item de planejamento que a estrutura atravessa: continua válido.
	if _, err := ResolveActive(ctx, repo, request.TextCode("BU-050")); err != nil {
		t.Fatalf("item fantasma foi recusado: %v", err)
	}

	_, err := ResolveActive(ctx, repo, request.TextCode("RN-09999"))
	if err == nil {
		t.Fatal("item inativo entrou em um lançamento")
	}
	// A mensagem precisa dizer QUAL item e O QUE fazer — quem recebe isso está
	// no meio de um pedido e não sabe de cadastro.
	for _, esperado := range []string{"RN-09999", "PERFIL DESCONTINUADO", "inativo", "reative-o"} {
		if !strings.Contains(err.Error(), esperado) {
			t.Fatalf("mensagem não explica o problema (%q não aparece): %s", esperado, err)
		}
	}

	// O caminho de manutenção continua abrindo o inativo, senão não há como
	// reativá-lo nem cadastrar a conversão de unidade dele.
	if _, err := Resolve(ctx, repo, request.TextCode("RN-09999")); err != nil {
		t.Fatalf("Resolve não pode recusar item inativo: %v", err)
	}
}

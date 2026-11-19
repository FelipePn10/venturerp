package item_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structrepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

type ativacaoAuth struct{ ports.AuthService }

func (ativacaoAuth) FindItemByCode(context.Context) bool { return true }

type ativacaoEstrutura struct {
	structrepo.ItemStructureRepository
	filhos int
}

func (e ativacaoEstrutura) GetAllDirectChildren(context.Context, int64) ([]*structentity.ItemStructure, error) {
	out := make([]*structentity.ItemStructure, e.filhos)
	for i := range out {
		out[i] = &structentity.ItemStructure{IsActive: true}
	}
	return out, nil
}

type ativacaoRoteiro struct {
	routingrepo.RoutingRepository
	roteiros int
	externas []*routingentity.ExternalOp
}

func (r ativacaoRoteiro) ListRoutesByItem(context.Context, int64) ([]*routingentity.ManufacturingRoute, error) {
	out := make([]*routingentity.ManufacturingRoute, r.roteiros)
	for i := range out {
		out[i] = &routingentity.ManufacturingRoute{IsActive: true}
	}
	return out, nil
}
func (r ativacaoRoteiro) GetExternalOpsByItem(context.Context, int64) ([]*routingentity.ExternalOp, error) {
	return r.externas, nil
}

type ativacaoFontes struct{ com map[int64]bool }

func (f ativacaoFontes) GetPreferredSupplier(_ context.Context, item int64) (int64, bool, error) {
	return 1, f.com[item], nil
}

type ativacaoFatores struct{ defasadas, sem []string }

func (f ativacaoFatores) ConferirDoPai(context.Context, int64) ([]string, []string, error) {
	return f.defasadas, f.sem, nil
}

func temProblema(r *ItemActivationReport, trecho string) bool {
	for _, i := range r.Issues {
		if strings.Contains(i, trecho) {
			return true
		}
	}
	return false
}

func TestProntidaoPegaOsCasosDaTecnofer(t *testing.T) {
	item := func(tipo types.TypeItem) *entity.Item {
		return &entity.Item{Code: 500, Engineering: entity.Engineering{Type: tipo}}
	}

	// BU-120/50: comprado com estrutura e roteiro — o MRP compraria a bucha.
	uc := &ValidateItemActivationUseCase{Auth: ativacaoAuth{}, StructureRepo: ativacaoEstrutura{filhos: 2},
		RoutingRepo: ativacaoRoteiro{roteiros: 1}, Suppliers: ativacaoFontes{com: map[int64]bool{500: true}}}
	r, err := uc.validate(context.Background(), item(types.COMPRADO), 500)
	if err != nil {
		t.Fatal(err)
	}
	if r.Ready || !temProblema(r, "cadastrado como comprado") || !temProblema(r, "2 componente(s)") || !temProblema(r, "roteiro") {
		t.Fatalf("comprado com processo passou: %+v", r)
	}

	// Fabricado: fator velho, operação externa comprando o próprio produto e
	// operação sem fornecedor.
	zinc, servico := int64(500), int64(900)
	semForn := int64(901)
	forn := int64(7)
	uc = &ValidateItemActivationUseCase{Auth: ativacaoAuth{}, StructureRepo: ativacaoEstrutura{filhos: 1},
		RoutingRepo: ativacaoRoteiro{roteiros: 1, externas: []*routingentity.ExternalOp{
			{OperationName: "ZINCAGEM", ServiceItemCode: &zinc, SupplierID: &forn},
			{OperationName: "PINTURA", ServiceItemCode: &semForn},
			{OperationName: "TÊMPERA", ServiceItemCode: &servico},
			{OperationName: "SOLDA FORA"},
		}},
		Suppliers: ativacaoFontes{com: map[int64]bool{900: true}},
		Fatores:   ativacaoFatores{defasadas: []string{"TUBO 101 104 MM gravado como 104 M (fator 1)"}}}
	r, err = uc.validate(context.Background(), item(types.FABRICADO), 500)
	if err != nil {
		t.Fatal(err)
	}
	for _, trecho := range []string{"unidade velha", "ZINCAGEM usa o próprio produto", "PINTURA sem fornecedor"} {
		if !temProblema(r, trecho) {
			t.Errorf("faltou %q: %+v", trecho, r.Issues)
		}
	}
	if temProblema(r, "TÊMPERA") {
		t.Errorf("TÊMPERA tem fonte do item de serviço e foi acusada: %v", r.Issues)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "SOLDA FORA") {
		t.Errorf("operação sem item de serviço deveria virar aviso: %v", r.Warnings)
	}

	// Comprado sem estrutura nem roteiro: nada a acusar por isso.
	uc = &ValidateItemActivationUseCase{Auth: ativacaoAuth{}, StructureRepo: ativacaoEstrutura{},
		RoutingRepo: ativacaoRoteiro{}, Suppliers: ativacaoFontes{com: map[int64]bool{500: true}}}
	if r, _ = uc.validate(context.Background(), item(types.COMPRADO), 500); !r.Ready {
		t.Fatalf("comprado limpo reprovado: %+v", r)
	}
}

type ativacaoUnidades struct{ faltam []string }

func (a ativacaoUnidades) UnidadesDeCompraSemConversao(context.Context, int64) ([]string, error) {
	return a.faltam, nil
}

// Chapas 8 e 9 da Tecnofer: compra em UN (fonte de suprimento), estoque em KG
// e só CHAPA→KG cadastrada. Antes saíam liberadas e só falhavam na aprovação
// do pedido; agora a liberação acusa a unidade sem conversão.
func TestLiberacaoBarraCompraSemConversao(t *testing.T) {
	item := &entity.Item{Code: 500, Engineering: entity.Engineering{Type: types.COMPRADO}}
	base := func(u conferidorDeUnidadesDeCompra) *ValidateItemActivationUseCase {
		return &ValidateItemActivationUseCase{Auth: ativacaoAuth{}, StructureRepo: ativacaoEstrutura{}, RoutingRepo: ativacaoRoteiro{},
			Suppliers: ativacaoFontes{com: map[int64]bool{500: true}}, UnidadesDeCompra: u}
	}
	r, err := base(ativacaoUnidades{faltam: []string{"UN→KG (fonte de suprimento)"}}).validate(context.Background(), item, 500)
	if err != nil {
		t.Fatal(err)
	}
	if r.Ready || !temProblema(r, "compra sem conversão para a unidade de estoque: UN→KG (fonte de suprimento)") {
		t.Fatalf("compra sem conversão passou na liberação: %+v", r)
	}
	if r, _ = base(ativacaoUnidades{}).validate(context.Background(), item, 500); !r.Ready {
		t.Fatalf("com conversão cadastrada deveria liberar: %+v", r)
	}
}

package itemresolution

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// A base da Tecnofer: códigos comerciais NUMÉRICOS ("1", "5", "8" — chapas)
// convivendo com as chaves internas 1..29. O comercial "5" é a CHAPA 3MM, cuja
// chave interna é 2; a chave interna 5 é a BALANÇA RN-01001.
type baseComCodigosAmbiguos struct{}

var (
	chapa3mm = &itementity.Item{Code: valueobject.ItemCode(2), BusinessCode: "5", Name: "CHAPA AÇO CARBONO 3MM"}
	balanca  = &itementity.Item{Code: valueobject.ItemCode(5), BusinessCode: "RN-01001", Name: "BALANÇA ASA DELTA"}
)

func (baseComCodigosAmbiguos) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*itementity.Item, error) {
	switch string(code) {
	case "5":
		return chapa3mm, nil
	case "RN-01001":
		return balanca, nil
	}
	return nil, itemrepo.ErrNotFound
}

func (baseComCodigosAmbiguos) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	switch int64(code) {
	case 2:
		return chapa3mm, nil
	case 5:
		return balanca, nil
	}
	return nil, itemrepo.ErrNotFound
}

// ⚠️ Defeito de produção: o middleware de compatibilidade troca o código público
// pela chave interna ANTES do caso de uso, e o caso de uso resolvia de novo pelo
// comercial primeiro. "RN-01001" virava 5, 5 era relido como o comercial "5" e o
// lançamento gravava CHAPA AÇO CARBONO 3MM respondendo 201.
func TestCodigoJaTraduzidoResolvePelaChaveInterna(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextkey.ItemCodeTranslatedKey, true)
	item, err := Resolve(ctx, baseComCodigosAmbiguos{}, request.TextCode("5"))
	if err != nil {
		t.Fatal(err)
	}
	if int64(item.Code) != 5 || item.BusinessCode != "RN-01001" {
		t.Fatalf("com a marca de tradução, 5 é a CHAVE INTERNA: veio %s (chave %d)", item.BusinessCode, int64(item.Code))
	}
}

// Sem a marca, o valor veio do cliente e é código comercial: o "5" é a chapa.
// É o contrato que a tela usa e não pode mudar.
func TestSemMarcaOCodigoNumericoSegueSendoComercial(t *testing.T) {
	item, err := Resolve(context.Background(), baseComCodigosAmbiguos{}, request.TextCode("5"))
	if err != nil {
		t.Fatal(err)
	}
	if int64(item.Code) != 2 || item.BusinessCode != "5" {
		t.Fatalf("sem marca, 5 é o código COMERCIAL: veio %s (chave %d)", item.BusinessCode, int64(item.Code))
	}
}

// Código alfanumérico não é ambíguo: com ou sem marca, resolve pelo comercial.
func TestCodigoAlfanumericoResolveIgualComOuSemMarca(t *testing.T) {
	for _, ctx := range []context.Context{
		context.Background(),
		context.WithValue(context.Background(), contextkey.ItemCodeTranslatedKey, true),
	} {
		item, err := Resolve(ctx, baseComCodigosAmbiguos{}, request.TextCode("RN-01001"))
		if err != nil {
			t.Fatal(err)
		}
		if int64(item.Code) != 5 {
			t.Fatalf("RN-01001 deveria resolver na chave 5, veio %d", int64(item.Code))
		}
	}
}

// Marca ligada e chave interna inexistente: cai no comercial em vez de recusar.
func TestComMarcaCaiNoComercialQuandoAChaveNaoExiste(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextkey.ItemCodeTranslatedKey, true)
	item, err := Resolve(ctx, baseComCodigosAmbiguos{}, request.TextCode("RN-01001"))
	if err != nil {
		t.Fatal(err)
	}
	if int64(item.Code) != 5 {
		t.Fatalf("esperava cair no comercial, veio %d", int64(item.Code))
	}
}

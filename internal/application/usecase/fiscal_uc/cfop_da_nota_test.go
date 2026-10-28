package fiscal_uc

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type catalogoFake map[int32]*entity.CFOP

func (c catalogoFake) GetCFOPByCode(_ context.Context, code int32) (*entity.CFOP, error) {
	if x, ok := c[code]; ok {
		return x, nil
	}
	return nil, sql.ErrNoRows
}

func TestConferirCFOPs(t *testing.T) {
	dia := func(s string) *time.Time { d, _ := time.Parse("2006-01-02", s); return &d }
	oficial := func(code int32, o entity.CFOPOficial) *entity.CFOP {
		o.Oficial, o.NFe = true, o.NFe || !o.Oficial
		if o.InicioVigencia == nil {
			o.InicioVigencia = dia("2006-01-01")
		}
		return &entity.CFOP{Code: code, IsActive: true, Oficial: o}
	}
	cat := catalogoFake{
		5101: oficial(5101, entity.CFOPOficial{}),
		1201: oficial(1201, entity.CFOPOficial{Devolucao: true}),
		5999: oficial(5999, entity.CFOPOficial{FimVigencia: dia("2019-12-31")}),
		5301: {Code: 5301, IsActive: true, Oficial: entity.CFOPOficial{Oficial: true, NFe: false}},
		5102: {Code: 5102, IsActive: false},
	}
	nota := &entity.FiscalExit{Finalidade: 1, DataEmissao: time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)}
	linha := func(seq int, cfop string) *entity.FiscalExitItem {
		return &entity.FiscalExitItem{Sequence: seq, Cfop: cfop}
	}
	conferir := func(e *entity.FiscalExit, items []*entity.FiscalExitItem, p *focusnfe.NFEPayload) map[string]string {
		t.Helper()
		pend, err := conferirCFOPs(context.Background(), cat, e, items, p)
		if err != nil {
			t.Fatal(err)
		}
		return niveis(pend)
	}

	if n := conferir(nota, []*entity.FiscalExitItem{linha(1, "5101")}, nil); len(n) != 0 {
		t.Fatalf("CFOP oficial vigente não pode ter pendência: %v", n)
	}
	n := conferir(nota, []*entity.FiscalExitItem{linha(1, "5888"), linha(2, "5999"), linha(3, "5301"), linha(4, "5102"), linha(5, "51")}, nil)
	for _, c := range []string{"itens.cfop.5888", "itens.cfop.5999", "itens.cfop.5301", "itens.cfop.5102", "itens.cfop.51"} {
		if n[c] != NivelImpede {
			t.Errorf("%s deveria impedir: %v", c, n)
		}
	}

	// Devolução: sem referência impede; com a nota referenciada (na nota ou por item) passa.
	dev := &entity.FiscalExit{Finalidade: 4, DataEmissao: nota.DataEmissao}
	if n := conferir(dev, []*entity.FiscalExitItem{linha(1, "1201")}, nil); n["itens.cfop.1201"] != NivelImpede {
		t.Fatalf("devolução sem referência deveria impedir: %v", n)
	}
	chave := "41261012345678000190550010000001231000001234"
	if n := conferir(&entity.FiscalExit{Finalidade: 4, DataEmissao: nota.DataEmissao, NFeReferenciada: &chave}, []*entity.FiscalExitItem{linha(1, "1201")}, nil); len(n) != 0 {
		t.Fatalf("devolução referenciada não pode ter pendência: %v", n)
	}
	porItem := &focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{DFeRefChave: chave}}}
	if n := conferir(dev, []*entity.FiscalExitItem{linha(1, "1201")}, porItem); len(n) != 0 {
		t.Fatalf("devolução referenciada por item não pode ter pendência: %v", n)
	}
	// Sem catálogo (rotas antigas), não confere.
	if pend, _ := conferirCFOPs(context.Background(), nil, nota, []*entity.FiscalExitItem{linha(1, "5888")}, nil); pend != nil {
		t.Fatal("sem catálogo não deveria conferir")
	}
}

func TestCfopDaLinha(t *testing.T) {
	casos := []struct {
		cfop     string
		comprado bool
		quer     string
	}{
		{"5101", false, "5101"},
		{"5101", true, "5102"},
		{"6101", true, "6102"},
		{"6107", true, "6108"},
		{"5401", true, "5403"},
		{"5102", false, "5102"}, // revenda escolhida fica, mesmo em item fabricado
		{"5949", true, "5949"},
		{"7101", true, "7102"}, // exportação de mercadoria de terceiros
		{"", true, ""},
	}
	for _, c := range casos {
		if got := cfopDaLinha(c.cfop, c.comprado); got != c.quer {
			t.Errorf("cfopDaLinha(%q, %v) = %q, quer %q", c.cfop, c.comprado, got, c.quer)
		}
	}
}

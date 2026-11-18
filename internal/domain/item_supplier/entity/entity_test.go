package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestDirectBillingDependencies(t *testing.T) {
	s, err := NewItemPreferredSupplier(1, 2, 3, "", 2, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	s.DirectBilling = true
	if s.Validate() == nil {
		t.Fatal("direct billing without preferred supplier accepted")
	}
	s.IsPreferred = true
	s.ThirdPartyOrder = true
	if err = s.Validate(); err != nil {
		t.Fatalf("valid dependency rejected: %v", err)
	}
	if s.Ranking != 1 {
		t.Fatalf("preferred ranking=%d", s.Ranking)
	}
}

func TestPoliticaDeCompraECota(t *testing.T) {
	novo := func() *ItemPreferredSupplier {
		s, err := NewItemPreferredSupplier(1, 2, 3, "", 2, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := novo()
	s.MinOrderQty = decimal.NewFromInt(-1)
	if s.Validate() == nil {
		t.Fatal("mínimo negativo aceito")
	}
	s = novo()
	s.OrderMultiple = decimal.NewFromInt(-5)
	if s.Validate() == nil {
		t.Fatal("múltiplo negativo aceito")
	}
	for _, cota := range []int64{0, -10, 101} {
		s = novo()
		c := decimal.NewFromInt(cota)
		s.QuotaPercent = &c
		if s.Validate() == nil {
			t.Fatalf("cota %d%% aceita", cota)
		}
	}
	s = novo()
	c := decimal.NewFromInt(100)
	s.QuotaPercent = &c
	if err := s.Validate(); err != nil {
		t.Fatalf("cota 100%% recusada: %v", err)
	}
	de, ate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	s = novo()
	s.ValidFrom, s.ValidUntil = &de, &ate
	if s.Validate() == nil {
		t.Fatal("vigência invertida aceita")
	}
}

func TestSituacaoDoVinculo(t *testing.T) {
	hoje := time.Date(2026, 11, 17, 15, 0, 0, 0, time.UTC)
	ontem, amanha := hoje.AddDate(0, 0, -1), hoje.AddDate(0, 0, 1)
	s, _ := NewItemPreferredSupplier(1, 2, 3, "", 1, uuid.New())
	if got := s.Situacao(hoje); got != SituacaoVigente {
		t.Fatalf("sem vigência = %s", got)
	}
	s.ValidUntil = &hoje
	if got := s.Situacao(hoje); got != SituacaoVigente {
		t.Fatalf("vence hoje = %s, o último dia ainda vale", got)
	}
	s.ValidUntil = &ontem
	if got := s.Situacao(hoje); got != SituacaoVencido {
		t.Fatalf("venceu ontem = %s", got)
	}
	s.ValidUntil, s.ValidFrom = nil, &amanha
	if got := s.Situacao(hoje); got != SituacaoFuturo {
		t.Fatalf("começa amanhã = %s", got)
	}
	s.Blocked = true
	if got := s.Situacao(hoje); got != SituacaoBloqueado {
		t.Fatalf("bloqueado = %s", got)
	}
}

func TestEventoDeBloqueioExigeMotivo(t *testing.T) {
	if _, err := NewBlockEvent(1, 2, AcaoBloqueio, "  ", uuid.New()); err == nil {
		t.Fatal("bloqueio sem motivo aceito")
	}
	if _, err := NewBlockEvent(1, 2, "APAGAR", "x", uuid.New()); err == nil {
		t.Fatal("ação inválida aceita")
	}
	if _, err := NewBlockEvent(1, 2, AcaoDesbloqueio, "voltou a entregar", uuid.New()); err != nil {
		t.Fatal(err)
	}
}

package purchase_order_uc

import (
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func TestConferirVidaUtil(t *testing.T) {
	hoje := time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC)
	minimo := 30
	validade := func(d int) *time.Time { v := time.Date(2026, 11, 6+d, 0, 0, 0, 0, time.UTC); return &v }
	regra := RegraDoItem{VidaMinimaDias: &minimo}
	if err := conferirVidaUtil(regra, validade(29), hoje, 1); err == nil {
		t.Fatal("lote com 29 dias não entra quando o mínimo é 30")
	}
	if err := conferirVidaUtil(regra, validade(30), hoje, 1); err != nil {
		t.Fatal(err)
	}
	if err := conferirVidaUtil(regra, nil, hoje, 1); err != nil {
		t.Fatal("sem data não há o que conferir")
	}
	if err := conferirVidaUtil(RegraDoItem{}, validade(1), hoje, 1); err != nil {
		t.Fatal("sem mínimo no cadastro não há regra")
	}
}

func TestDocumentosDoRecebimento(t *testing.T) {
	texto := func(s string) *string { return &s }
	item := request.ReceivePurchaseOrderItemDTO{PurchaseOrderItemCode: 1}
	if err := conferirDocumentosDoRecebimento(RegraDoItem{}, item); err != nil {
		t.Fatal("sem exigência não há o que cobrar")
	}
	if err := conferirDocumentosDoRecebimento(RegraDoItem{ExigeChecklist: true}, item); err == nil {
		t.Fatal("check list exigido e não confirmado")
	}
	item.ChecklistConfirmed = true
	if err := conferirDocumentosDoRecebimento(RegraDoItem{ExigeChecklist: true}, item); err != nil {
		t.Fatal(err)
	}
	item.Lot, item.HeatNumber = texto("L1"), texto("  ")
	if err := conferirDocumentosDoRecebimento(RegraDoItem{ExigeCertificado: true}, item); err == nil {
		t.Fatal("corrida em branco não vale")
	}
	item.HeatNumber, item.Certificate = texto("C-77"), texto("CERT-9")
	if err := conferirDocumentosDoRecebimento(RegraDoItem{ExigeCertificado: true}, item); err != nil {
		t.Fatal(err)
	}
}

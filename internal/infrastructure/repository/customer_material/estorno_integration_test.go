//go:build integration

// Testes do estorno da baixa quando a NF-e de retorno é cancelada (migração
// 000372).
//
// O documento 6 da Usimac exige, no cancelamento da nota, estornar as
// movimentações relacionadas. O defeito que estes testes impedem é o pior deste
// módulo: o saldo afirmando que o material voltou ao cliente enquanto ele continua
// no pátio da empresa.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run Estorno ./internal/infrastructure/repository/customer_material/
package customer_material_test

import (
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

const motivoEstorno = "cancelamento da NF-e 5911: quantidade errada na nota"

// notaBaixada fatura a nota inteira sobre a primeira linha da remessa.
func (a *ambiente) notaBaixada(t *testing.T, r *entity.Remessa, notaID int64, qtd string) {
	t.Helper()
	if _, err := a.repo.RegistrarMovimentosDaNota(a.ctx, notaID, []domrepo.MovimentoDaNota{{
		RemittanceItemID: r.Itens[0].ID, MovementType: entity.MovimentoRetorno,
		Quantity: qtd, UnitValue: "178.283333", CFOP: "5902",
	}}, a.userID); err != nil {
		t.Fatalf("baixando a nota %d: %v", notaID, err)
	}
}

// TestEstornoDevolveOSaldoDoCliente: o caso central. Cancelar a nota sem estornar
// deixa o sistema afirmando que o material voltou.
func TestEstornoDevolveOSaldoDoCliente(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	nota := int64(910_001 + testutilUnico())
	a.notaBaixada(t, r, nota, "6")

	depoisDaBaixa := a.recarregar(t, r.ID)
	if got := depoisDaBaixa.Itens[0].BalanceQty.String(); got != "0" {
		t.Fatalf("saldo após a baixa = %s, esperado 0", got)
	}
	if depoisDaBaixa.Status != entity.StatusEncerrada {
		t.Fatalf("situação após a baixa = %s, esperado ENCERRADA", depoisDaBaixa.Status)
	}

	estornados, err := a.repo.EstornarMovimentosDaNota(a.ctx, nota, a.userID, motivoEstorno)
	if err != nil {
		t.Fatalf("estorno recusado: %v", err)
	}
	if len(estornados) != 1 {
		t.Fatalf("estorno devolveu %d movimento(s), esperado 1", len(estornados))
	}
	if !estornados[0].Estornado() {
		t.Fatal("o movimento devolvido não está marcado como estornado")
	}
	if estornados[0].ReversalReason == nil || *estornados[0].ReversalReason != motivoEstorno {
		t.Fatalf("motivo do estorno = %v", estornados[0].ReversalReason)
	}
	if estornados[0].ReversedBy == nil || *estornados[0].ReversedBy != a.userID {
		t.Fatalf("autor do estorno = %v", estornados[0].ReversedBy)
	}

	depois := a.recarregar(t, r.ID)
	if got := depois.Itens[0].BalanceQty.String(); got != "6" {
		t.Fatalf("saldo após o estorno = %s, esperado 6 — o material continua no pátio", got)
	}
	if got := depois.Itens[0].QtyReturned.String(); got != "0" {
		t.Fatalf("quantidade retornada após o estorno = %s, esperado 0", got)
	}
	// A remessa havia encerrado sozinha ao zerar; com saldo de volta tem de reabrir,
	// senão ela sai da fila de cobrança do prazo fiscal com material aqui dentro.
	if depois.Status != entity.StatusAberta {
		t.Fatalf("situação após o estorno = %s, esperado ABERTA", depois.Status)
	}
}

// TestEstornoEhIdempotente: o estorno é chamado no cancelamento da nota e pode ser
// repetido à mão quando o cancelamento falha no meio. Devolver duas vezes inventaria
// material que não existe.
func TestEstornoEhIdempotente(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "10")
	primeira := int64(920_001 + testutilUnico())
	segunda := int64(921_001 + testutilUnico())
	// DUAS notas sobre a mesma linha, e estorna só a primeira. Com uma nota só, um
	// estorno em dobro levaria o consumo a zero — que é onde ele já ia parar — e o
	// saldo daria o valor certo por acidente. Com a segunda nota de pé, devolver
	// duas vezes passa do saldo e aparece.
	a.notaBaixada(t, r, primeira, "4")
	a.notaBaixada(t, r, segunda, "3")

	if _, err := a.repo.EstornarMovimentosDaNota(a.ctx, primeira, a.userID, motivoEstorno); err != nil {
		t.Fatalf("primeiro estorno: %v", err)
	}
	repetido, err := a.repo.EstornarMovimentosDaNota(a.ctx, primeira, a.userID, motivoEstorno)
	if err != nil {
		t.Fatalf("repetir o estorno deveria devolver o estado atual: %v", err)
	}
	if len(repetido) != 1 {
		t.Fatalf("a repetição devolveu %d movimento(s), esperado 1", len(repetido))
	}
	item := a.recarregar(t, r.ID).Itens[0]
	if got := item.QtyReturned.String(); got != "3" {
		t.Fatalf("quantidade retornada = %s, esperado 3 — a segunda nota continua de pé", got)
	}
	if got := item.BalanceQty.String(); got != "7" {
		t.Fatalf("saldo = %s, esperado 7 — a repetição devolveu de novo", got)
	}
}

// TestEstornoNaoReabreEncerramentoManual: encerrar com saldo é decisão aprovada de
// pessoa. Desfazê-la em silêncio por causa de uma nota cancelada trocaria um
// problema de saldo por um de governança.
func TestEstornoNaoReabreEncerramentoManual(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "10")
	nota := int64(930_001 + testutilUnico())
	a.notaBaixada(t, r, nota, "4")

	if err := a.repo.Encerrar(a.ctx, r.ID, "cliente abriu mão do saldo restante", a.userID); err != nil {
		t.Fatalf("encerramento recusado: %v", err)
	}
	if _, err := a.repo.EstornarMovimentosDaNota(a.ctx, nota, a.userID, motivoEstorno); err != nil {
		t.Fatalf("estorno recusado: %v", err)
	}
	depois := a.recarregar(t, r.ID)
	if depois.Status != entity.StatusEncerrada {
		t.Fatalf("situação = %s: o encerramento aprovado não pode ser desfeito pelo estorno", depois.Status)
	}
	// O saldo volta mesmo assim: a quantidade física é fato, a situação é decisão.
	if got := depois.Itens[0].BalanceQty.String(); got != "10" {
		t.Fatalf("saldo = %s, esperado 10", got)
	}
}

// TestEstornoDeixaANotaSemBaixa: é o que impede autorizar na SEFAZ uma nota cuja
// baixa foi desfeita, e o que faz um novo faturamento da mesma nota baixar de novo
// em vez de achar que já baixou.
func TestEstornoDeixaANotaSemBaixa(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	nota := int64(940_001 + testutilUnico())
	a.notaBaixada(t, r, nota, "6")

	vivos, err := a.repo.MovimentosDaNota(a.ctx, nota)
	if err != nil {
		t.Fatalf("lendo os movimentos da nota: %v", err)
	}
	if len(vivos) != 1 {
		t.Fatalf("antes do estorno a nota tinha %d movimento(s), esperado 1", len(vivos))
	}

	if _, err := a.repo.EstornarMovimentosDaNota(a.ctx, nota, a.userID, motivoEstorno); err != nil {
		t.Fatalf("estorno recusado: %v", err)
	}
	depois, err := a.repo.MovimentosDaNota(a.ctx, nota)
	if err != nil {
		t.Fatalf("lendo os movimentos após o estorno: %v", err)
	}
	if len(depois) != 0 {
		t.Fatalf("a nota ainda conta %d baixa(s) depois do estorno", len(depois))
	}
}

// TestEstornoExigeAutorEMotivo: estorno sem motivo é estorno que ninguém audita.
func TestEstornoExigeAutorEMotivo(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	nota := int64(950_001 + testutilUnico())
	a.notaBaixada(t, r, nota, "6")

	if _, err := a.repo.EstornarMovimentosDaNota(a.ctx, nota, "", motivoEstorno); err == nil {
		t.Fatal("estorno sem autor foi aceito")
	}
	if _, err := a.repo.EstornarMovimentosDaNota(a.ctx, nota, a.userID, "   "); err == nil {
		t.Fatal("estorno sem motivo foi aceito")
	}
	if got := a.recarregar(t, r.ID).Itens[0].BalanceQty.String(); got != "0" {
		t.Fatalf("saldo = %s: a recusa não pode ter devolvido saldo", got)
	}
}

// TestEstornoNaoVazaEntreEmpresas: estornar a nota da vizinha devolveria saldo de
// material que não é dela.
func TestEstornoNaoVazaEntreEmpresas(t *testing.T) {
	a := montar(t)
	outra := montar(t)
	r := a.remessa(t, "6")
	nota := int64(960_001 + testutilUnico())
	a.notaBaixada(t, r, nota, "6")

	estornados, err := outra.repo.EstornarMovimentosDaNota(outra.ctx, nota, outra.userID, motivoEstorno)
	if err != nil {
		t.Fatalf("estorno pela empresa vizinha: %v", err)
	}
	if len(estornados) != 0 {
		t.Fatalf("a vizinha estornou %d movimento(s) da nota alheia", len(estornados))
	}
	if got := a.recarregar(t, r.ID).Itens[0].BalanceQty.String(); got != "0" {
		t.Fatalf("saldo da dona = %s, esperado 0 — a vizinha devolveu saldo alheio", got)
	}
	// Controle positivo: a dona estorna a mesma nota, então o teste acima não passou
	// porque o estorno simplesmente não funciona.
	if propria, err := a.repo.EstornarMovimentosDaNota(a.ctx, nota, a.userID, motivoEstorno); err != nil || len(propria) != 1 {
		t.Fatalf("a dona não conseguiu estornar a própria nota: %v (%d movimentos)", err, len(propria))
	}
}

// TestEstornoEntraNaTrilha: o cancelamento tem de ficar registrado no histórico da
// operação, como o documento 6 pede.
func TestEstornoEntraNaTrilha(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	nota := int64(970_001 + testutilUnico())
	a.notaBaixada(t, r, nota, "6")
	if _, err := a.repo.EstornarMovimentosDaNota(a.ctx, nota, a.userID, motivoEstorno); err != nil {
		t.Fatalf("estorno recusado: %v", err)
	}

	var achou bool
	for _, ev := range a.trilha(t, r.ID) {
		if ev.EntityType != "customer_material_movements" || ev.Acao != "UPDATE" {
			continue
		}
		if ev.Motivo != nil && strings.Contains(*ev.Motivo, "cancelamento da NF-e") &&
			contem(ev.Alterados, "reversed_at") {
			achou = true
			break
		}
	}
	if !achou {
		t.Fatal("o estorno não aparece na trilha da remessa")
	}
}

// testutilUnico dá um número de nota distinto por execução. A nota não tem FK, mas
// número repetido na mesma base confundiria a leitura de uma falha.
func testutilUnico() int64 {
	return testutil.UniqueCode() % 10_000
}

//go:build integration

package stock_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// Transferência: saída e entrada na mesma operação, pelo mesmo custo; o
// estorno de uma perna desfaz as duas, e não se estorna duas vezes.
func TestIntegration_TransferenciaEEstorno(t *testing.T) {
	e := novoAmbienteEstoque(t)
	if err := e.mover(t, e.item, e.almox, "IN", 10, 7, ""); err != nil {
		t.Fatal(err)
	}
	res, err := e.repo.TransferirEntreAlmoxarifados(e.ctx, stockentity.TransferenciaEstoque{
		ItemCode: e.item, DeAlmox: e.almox, ParaAlmox: e.livre, Quantidade: dec("4"), Ator: e.ator,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Saida.UnitPrice.Equal(decimal.NewFromFloat(7)) || !res.Entrada.UnitPrice.Equal(decimal.NewFromFloat(7)) || *res.Entrada.ReferenceCode != res.Saida.ID {
		t.Fatalf("as pernas levam o custo médio e se referenciam: %+v / %+v", res.Saida, res.Entrada)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 6 {
		t.Fatalf("origem 10 − 4 = 6, veio %v", q)
	}
	if q, c := e.saldo(t, e.item, e.livre); q != 4 || c != 7 {
		t.Fatalf("destino 4 a custo 7, veio %v / %v", q, c)
	}
	if _, err := e.repo.TransferirEntreAlmoxarifados(e.ctx, stockentity.TransferenciaEstoque{
		ItemCode: e.item, DeAlmox: e.almox, ParaAlmox: e.livre, Quantidade: dec("50"), Ator: e.ator,
	}); !ehValidacao(err) {
		t.Fatalf("transferir mais que o saldo deveria ser recusado: %v", err)
	}
	if q, _ := e.saldo(t, e.item, e.livre); q != 4 {
		t.Fatalf("transferência recusada não deixa entrada solta no destino: %v", q)
	}

	estornos, err := e.repo.EstornarMovimento(e.ctx, res.Saida.ID, e.ator, "transferido errado")
	if err != nil || len(estornos) != 2 {
		t.Fatalf("estornar a transferência desfaz as duas pernas: %v %v", estornos, err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 10 {
		t.Fatalf("origem volta a 10, veio %v", q)
	}
	if q, _ := e.saldo(t, e.item, e.livre); q != 0 {
		t.Fatalf("destino volta a 0, veio %v", q)
	}
	if _, err := e.repo.EstornarMovimento(e.ctx, res.Entrada.ID, e.ator, "de novo"); !errors.As(err, new(*errorsuc.ConflictError)) {
		t.Fatalf("estorno duplo deveria dar conflito: %v", err)
	}
	if _, err := e.repo.EstornarMovimento(e.ctx, estornos[0].ID, e.ator, "estorno do estorno"); !ehValidacao(err) {
		t.Fatalf("estorno não se estorna: %v", err)
	}
	if _, err := e.repo.EstornarMovimento(e.ctx, res.Saida.ID, e.ator, " "); !ehValidacao(err) {
		t.Fatalf("estorno exige motivo: %v", err)
	}
}

// Estorno de movimento manual devolve o saldo pelo mesmo valor; movimento de
// documento (nota) é recusado; estornar entrada já consumida é barrado pelo saldo.
func TestIntegration_EstornoDeMovimentoManual(t *testing.T) {
	e := novoAmbienteEstoque(t)
	entrada := &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "IN", Quantity: decimal.NewFromFloat(5), UnitPrice: decimal.NewFromFloat(3), CreatedBy: e.ator}
	if _, err := e.repo.CreateMovement(e.ctx, entrada); err != nil {
		t.Fatal(err)
	}
	if err := e.mover(t, e.item, e.almox, "OUT", 4, 0, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.EstornarMovimento(e.ctx, entrada.ID, e.ator, "lançada errado"); !ehValidacao(err) || !strings.Contains(err.Error(), "saldo insuficiente") {
		t.Fatalf("estornar entrada já consumida deixaria saldo negativo: %v", err)
	}
	if err := e.mover(t, e.item, e.almox, "IN", 4, 3, ""); err != nil {
		t.Fatal(err)
	}
	rev, err := e.repo.EstornarMovimento(e.ctx, entrada.ID, e.ator, "lançada errado")
	if err != nil || len(rev) != 1 || rev[0].MovementType != "OUT" || !rev[0].TotalPrice.Equal(decimal.NewFromFloat(15)) {
		t.Fatalf("estorno manual: saída de 5 por 15: %+v %v", rev, err)
	}
	ref := stockentity.ReferenceTypeNFEntry
	code := int64(99)
	nota := &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "IN", Quantity: decimal.NewFromFloat(1), UnitPrice: decimal.NewFromFloat(1), CreatedBy: e.ator, ReferenceType: &ref, ReferenceCode: &code}
	if _, err := e.repo.CreateMovement(e.ctx, nota); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.EstornarMovimento(e.ctx, nota.ID, e.ator, "x"); !ehValidacao(err) || !strings.Contains(err.Error(), "documento") {
		t.Fatalf("movimento de nota se desfaz pela nota: %v", err)
	}
}

// Kardex: saldo anterior, saldo acumulado linha a linha e saldo final; e a
// posição numa data passada ignora o que veio depois.
func TestIntegration_KardexEPosicaoEmData(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 2, "")
	_ = e.mover(t, e.item, e.almox, "OUT", 3, 0, "")
	_ = e.mover(t, e.item, e.almox, "IN", 5, 4, "")
	// O primeiro movimento vai para 40 dias atrás: fica no saldo anterior.
	testutil.Exec(t, pool, `UPDATE stock_movements SET created_at = NOW() - INTERVAL '40 days'
		WHERE id = (SELECT MIN(id) FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2)`, e.empresa, e.item)
	hoje := time.Now()
	k, err := e.repo.Kardex(e.ctx, e.item, nil, hoje.AddDate(0, 0, -10), hoje)
	if err != nil {
		t.Fatal(err)
	}
	if !k.SaldoAnterior.Equal(dec("10")) || !k.ValorAnterior.Equal(decimal.NewFromFloat(20)) {
		t.Fatalf("saldo anterior 10 (R$ 20): %v %v", k.SaldoAnterior, k.ValorAnterior)
	}
	if len(k.Linhas) != 2 || !k.Linhas[0].Saldo.Equal(dec("7")) || !k.Linhas[1].Saldo.Equal(dec("12")) {
		t.Fatalf("saldos acumulados 7 e 12: %+v", k.Linhas)
	}
	if !k.SaldoFinal.Equal(dec("12")) || !k.ValorFinal.Equal(decimal.NewFromFloat(34)) || !k.Entradas.Equal(dec("5")) || !k.Saidas.Equal(dec("3")) {
		t.Fatalf("final 12 / R$ 34, entradas 5, saídas 3: %+v", k)
	}
	pos, err := e.repo.PosicaoEmData(e.ctx, hoje.AddDate(0, 0, -20), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pos) != 1 || !pos[0].Quantidade.Equal(dec("10")) {
		t.Fatalf("há 20 dias só existia a entrada de 10: %+v", pos)
	}
	// Segunda unidade: item estocado em UN com compra em CX (1 CX = 5 UN).
	testutil.Exec(t, pool, `UPDATE items SET supplies_purchase_uom='CX' WHERE code=$1`, e.item)
	testutil.Exec(t, pool, `INSERT INTO item_unit_conversions(item_code,from_uom,to_uom,factor,created_by) VALUES($1,'CX','UN',5,$2)`, e.item, e.ator)
	pos, err = e.repo.PosicaoEmData(e.ctx, hoje, &e.almox)
	if err != nil || len(pos) != 1 || pos[0].QuantidadeAlt == nil || !pos[0].QuantidadeAlt.Equal(dec("2.4")) || *pos[0].UnidadeAlternativa != "CX" {
		t.Fatalf("12 UN = 2,4 CX: %+v %v", pos, err)
	}
}

// ATP: inspeção não é prometível.
func TestIntegration_ATPIgnoraInspecao(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	testutil.Exec(t, pool, `UPDATE warehouse SET location='INSPECAO' WHERE id=$1`, e.livre)
	_ = e.mover(t, e.item, e.almox, "IN", 6, 1, "")
	_ = e.mover(t, e.item, e.livre, "IN", 4, 1, "")
	saldos, err := e.repo.SaldosComSituacao(e.ctx, e.item)
	if err != nil || len(saldos) != 2 {
		t.Fatalf("dois saldos: %+v %v", saldos, err)
	}
	for _, s := range saldos {
		if s.WarehouseID == e.livre && (s.Prometivel || s.Situacao != "INSPECAO") {
			t.Fatalf("inspeção não é prometível: %+v", s)
		}
		if s.WarehouseID == e.almox && (!s.Prometivel || s.Situacao != "LIVRE") {
			t.Fatalf("almoxarifado normal é livre: %+v", s)
		}
	}
}

// Reposição: item abaixo do mínimo, descontando o que já está pedido.
func TestIntegration_ReposicaoPeloMinimo(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_minimum_stock=20, planning_safety_stock=5, planning_multiple_lot=10 WHERE code=$1`, e.item)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_minimum_stock=3 WHERE code=$1`, e.outro)
	_ = e.mover(t, e.item, e.almox, "IN", 8, 1, "")
	_ = e.mover(t, e.outro, e.almox, "IN", 9, 1, "")
	sug, err := e.repo.SugestoesDeReposicao(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sug) != 1 || sug[0].ItemCode != e.item {
		t.Fatalf("só o item abaixo do mínimo: %+v", sug)
	}
	// 20 + 5 − 8 = 17 → múltiplo de 10 = 20.
	if !sug[0].Sugerido.Equal(dec("20")) || !sug[0].PontoPedido.Equal(dec("20")) || sug[0].Motivo != "estoque mínimo do cadastro" {
		t.Fatalf("sugestão 20 pelo mínimo: %+v", sug[0])
	}
}

// Análise: consumo, cobertura, giro e item parado; transferência não é consumo.
func TestIntegration_AnaliseDeEstoque(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	_ = e.mover(t, e.item, e.almox, "IN", 12, 10, "")
	_ = e.mover(t, e.item, e.almox, "OUT", 6, 0, "")
	if _, err := e.repo.TransferirEntreAlmoxarifados(e.ctx, stockentity.TransferenciaEstoque{ItemCode: e.item, DeAlmox: e.almox, ParaAlmox: e.livre, Quantidade: dec("2"), Ator: e.ator}); err != nil {
		t.Fatal(err)
	}
	_ = e.mover(t, e.outro, e.almox, "IN", 3, 5, "")
	testutil.Exec(t, pool, `UPDATE stock_movements SET created_at = NOW() - INTERVAL '200 days' WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.outro)
	an, err := e.repo.AnaliseDeEstoque(e.ctx, 6, 90)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range an {
		switch a.ItemCode {
		case e.item:
			if !a.ConsumoQtd.Equal(dec("6")) || !a.ConsumoMedioMensal.Equal(dec("1")) || a.CoberturaMeses == nil || *a.CoberturaMeses != 6 {
				t.Fatalf("consumo 6 (a transferência não conta), média 1/mês, cobertura 6 meses: %+v", a)
			}
		case e.outro:
			if !a.Parado || a.DiasSemMovimento == nil || *a.DiasSemMovimento < 199 {
				t.Fatalf("item sem movimento há 200 dias é parado: %+v", a)
			}
		}
	}
}

// Requisição ao almoxarifado: atendimento parcial e total no centro de custo,
// limite pelo pendente e cancelamento do resto.
func TestIntegration_RequisicaoAoAlmoxarifado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 2, "")
	_ = e.mover(t, e.outro, e.almox, "IN", 10, 2, "")
	cc := int32(310)
	req, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
		WarehouseID: e.almox, CostCenterCode: &cc, CreatedBy: e.ator,
		Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("4")}, {ItemCode: e.outro, Quantity: dec("2")}},
	})
	if err != nil || req.Code <= 0 || len(req.Itens) != 2 {
		t.Fatalf("criar: %+v %v", req, err)
	}
	if _, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[0].ID, Quantity: dec("5")}}, e.ator); !ehValidacao(err) {
		t.Fatalf("entregar mais que o pendente deveria ser recusado: %v", err)
	}
	r, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[0].ID, Quantity: dec("4")}}, e.ator)
	if err != nil || r.Status != stockentity.RequisicaoParcial {
		t.Fatalf("atendida uma linha: parcial. %+v %v", r, err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 6 {
		t.Fatalf("saída de 4: %v", q)
	}
	var centro *int32
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT cost_center_code FROM stock_movements WHERE enterprise_id=$1 AND reference_type='REQUISICAO_MATERIAL' AND reference_code=$2 LIMIT 1`,
		e.empresa, req.Code).Scan(&centro); err != nil || centro == nil || *centro != 310 {
		t.Fatalf("a saída leva o centro de custo da requisição: %v %v", centro, err)
	}
	if err := e.repo.CancelarRequisicaoMaterial(e.ctx, req.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[1].ID, Quantity: dec("1")}}, e.ator); !ehValidacao(err) {
		t.Fatalf("requisição cancelada não se atende: %v", err)
	}
	lista, err := e.repo.ListarRequisicoesMaterial(e.ctx, nil)
	if err != nil || len(lista) != 1 || lista[0].Status != stockentity.RequisicaoCancelada || !lista[0].Itens[0].IssuedQty.Equal(dec("4")) {
		t.Fatalf("lista: cancelada com o já entregue preservado: %+v %v", lista, err)
	}
}

// O dia do kardex e da posição é o da empresa (America/Sao_Paulo por padrão):
// o lançamento das 21:30 de Brasília é 00:30 do dia seguinte em UTC e sumia de
// "até hoje"; o de 00:30 do dia seguinte não pode entrar.
func TestIntegration_KardexDiaNoFusoDaEmpresa(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 2, "")
	_ = e.mover(t, e.item, e.almox, "IN", 5, 2, "")
	dia := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC) // só o calendário importa
	testutil.Exec(t, pool, `UPDATE stock_movements SET created_at = ($3::date + TIME '21:30') AT TIME ZONE 'America/Sao_Paulo'
		WHERE id = (SELECT MIN(id) FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2)`, e.empresa, e.item, dia)
	testutil.Exec(t, pool, `UPDATE stock_movements SET created_at = ($3::date + 1 + TIME '00:30') AT TIME ZONE 'America/Sao_Paulo'
		WHERE id = (SELECT MAX(id) FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2)`, e.empresa, e.item, dia)
	k, err := e.repo.Kardex(e.ctx, e.item, nil, dia, dia)
	if err != nil {
		t.Fatal(err)
	}
	if len(k.Linhas) != 1 || !k.SaldoFinal.Equal(dec("10")) {
		t.Fatalf("o dia 10/03 tem só o lançamento das 21:30 (saldo 10): %d linhas, saldo %s", len(k.Linhas), k.SaldoFinal)
	}
	pos, err := e.repo.PosicaoEmData(e.ctx, dia, &e.almox)
	if err != nil || len(pos) != 1 || !pos[0].Quantidade.Equal(dec("10")) {
		t.Fatalf("posição ao fim de 10/03 tem 10: %+v %v", pos, err)
	}
	// Datas em branco: padrão do mês corrente até hoje, sem erro.
	if _, err := e.repo.Kardex(e.ctx, e.item, nil, time.Time{}, time.Time{}); err != nil {
		t.Fatalf("kardex com período padrão: %v", err)
	}
}

// Transferência originada de um documento (a liberação da inspeção de
// recebimento): as duas pernas guardam a referência do documento, ficam na mesma
// transação e a entrada vale o que saiu — antes o suprimentos gravava a entrada
// sem preço e o valor liberado da inspeção sumia.
func TestIntegration_TransferenciaComReferenciaDeDocumento(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 4.5, "")
	ref, codigo := "RECEIVING_INSPECTION_ORDER", int64(777)
	r, err := e.repo.TransferirEntreAlmoxarifados(e.ctx, stockentity.TransferenciaEstoque{
		ItemCode: e.item, DeAlmox: e.almox, ParaAlmox: e.livre, Quantidade: dec("6.185567"), Ator: e.ator,
		ReferenciaTipo: &ref, ReferenciaCodigo: &codigo,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []*stockentity.StockMovement{r.Saida, r.Entrada} {
		if m.ReferenceType == nil || *m.ReferenceType != ref || m.ReferenceCode == nil || *m.ReferenceCode != codigo {
			t.Fatalf("perna %s sem a referência do documento: %v %v", m.MovementType, m.ReferenceType, m.ReferenceCode)
		}
	}
	if !r.Entrada.UnitPrice.Equal(r.Saida.UnitPrice) || !r.Entrada.TotalPrice.Equal(r.Saida.TotalPrice) || !r.Saida.TotalPrice.Equal(dec("27.8351")) {
		t.Fatalf("a entrada vale o que saiu (6,185567 × 4,50 = 27,8351): saída %s / entrada %s", r.Saida.TotalPrice, r.Entrada.TotalPrice)
	}
	destino, err := e.repo.GetBalance(e.ctx, e.item, "", e.livre)
	if err != nil || !destino.Quantity.Equal(dec("6.185567")) || !destino.TotalCost.Equal(dec("27.8351")) {
		t.Fatalf("destino com 6,185567 e R$ 27,8351: %+v %v", destino, err)
	}
	// Movimento de documento se desfaz pelo documento, não pelo estorno avulso.
	if _, err := e.repo.EstornarMovimento(e.ctx, r.Saida.ID, e.ator, "x"); err == nil {
		t.Fatal("o estorno avulso de uma transferência de documento deveria ser recusado")
	}
}

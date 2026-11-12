//go:build integration

// As consultas de impressão são SQL escrito à mão: este teste executa cada uma
// contra o banco migrado, para que coluna renomeada ou junção errada apareça
// aqui e não na hora de imprimir. Também confere que outra empresa não enxerga
// os documentos.
//
//	TEST_DATABASE_URL=... go test -tags=integration ./internal/infrastructure/repository/impressao/
package impressao_test

import (
	"context"
	"testing"
	"time"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/impressao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
)

func TestConsultasDeImpressaoExecutam(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	var empresa, codigo int64
	if err := pool.QueryRow(ctx, `SELECT id, code FROM enterprise ORDER BY id LIMIT 1`).Scan(&empresa, &codigo); err != nil {
		t.Fatalf("empresa semeada: %v", err)
	}
	c := impressao.New(pool)
	ten := uc.Tenant{ID: empresa, Code: codigo}
	hoje := time.Now()
	de := hoje.AddDate(0, -1, 0)

	if _, err := c.Emitente(ctx, ten); err != nil {
		t.Errorf("emitente: %v", err)
	}
	_ = c.NomeUsuario(ctx, uuid.New())
	checar := func(nome string, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("%s: %v", nome, err)
		}
	}
	_, err := c.PedidoVenda(ctx, ten, -1)
	checar("pedido de venda", err)
	_, err = c.NotaEntrada(ctx, ten, -1)
	checar("nota de entrada", err)
	_, err = c.Titulos(ctx, ten, true, uc.FiltroTitulos{De: &de, Ate: &hoje, SomenteAbertos: true})
	checar("contas a receber", err)
	_, err = c.Titulos(ctx, ten, false, uc.FiltroTitulos{})
	checar("contas a pagar", err)
	_, err = c.ContasBancarias(ctx, ten)
	checar("contas bancárias", err)
	_, err = c.PlanoMateriais(ctx, ten, -1)
	checar("plano de materiais", err)
	_, err = c.Capacidade(ctx, ten, -1)
	checar("capacidade", err)
	_, err = c.Programacao(ctx, ten, de, hoje, nil)
	checar("programação", err)
	_, err = c.Kardex(ctx, ten, uc.FiltroKardex{De: de, Ate: hoje})
	checar("kardex", err)
	_, err = c.CustoProducao(ctx, ten, []int64{}, &de, &hoje)
	checar("custo de produção", err)
	_, err = c.Romaneio(ctx, ten, -1)
	checar("romaneio", err)
	_, err = c.Proposta(ctx, ten, -1)
	checar("proposta", err)
	_, err = c.Cotacao(ctx, ten, -1)
	checar("cotação", err)
	_, err = c.FichaCliente(ctx, ten, -1)
	checar("ficha do cliente", err)

	// Outra empresa não enxerga os pedidos desta.
	var pedido int64
	if err := pool.QueryRow(ctx, `SELECT code FROM sales_orders WHERE enterprise_code=$1 LIMIT 1`, codigo).Scan(&pedido); err == nil {
		if p, err := c.PedidoVenda(ctx, uc.Tenant{ID: empresa + 9999, Code: codigo + 9999}, pedido); err != nil || p != nil {
			t.Fatalf("pedido %d vazou para outra empresa: %v %v", pedido, p, err)
		}
	}
}

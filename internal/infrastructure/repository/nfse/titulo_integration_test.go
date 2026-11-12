//go:build integration

package nfse_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	financialentity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/nfse"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O título da NFS-e (migration 000486) sai uma vez só, fica na empresa da
// nota e é cancelado com ela.
func TestTituloDaNFSeUmaVezSoEPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		_, _ = pool.Exec(context.Background(), `DELETE FROM contas_receber WHERE enterprise_id = ANY($1)`, ids)
		_, _ = pool.Exec(context.Background(), `DELETE FROM nfse WHERE enterprise_id = ANY($1)`, ids)
	})
	ator := testutil.Actor(t, pool)
	emissao := time.Now().Truncate(24 * time.Hour)
	venc := emissao.AddDate(0, 0, 30)
	nota, err := nfse.NewNFSeRepositoryPG(pool).Create(a.Ctx, &entity.NFSe{TipoRPS: 1, NaturezaOperacao: 1, DataEmissao: emissao,
		Status: entity.NFSeStatusRascunho, ItemListaServico: "14.01", Discriminacao: "SERVICO", CodigoMunicipio: "4106902",
		ValorServicos: decimal.NewFromInt(1000), ValorLiquido: decimal.NewFromInt(970), DataVencimento: &venc, CreatedBy: ator})
	if err != nil {
		t.Fatal(err)
	}
	if nota.DataVencimento == nil {
		t.Fatal("o vencimento da NFS-e não foi gravado")
	}
	fin := financial.NewFinancialRepositoryPG(pool).(*financial.FinancialRepositoryPG)
	doc := "NFS-e 1"
	titulo := func() *financialentity.ContaReceber {
		return &financialentity.ContaReceber{NumeroDocumento: &doc, DataLancamento: time.Now(), DataEmissao: emissao,
			DataVencimento: venc, ValorBruto: decimal.NewFromInt(970), CriadoPor: ator}
	}
	if gerou, err := fin.GerarTituloDaNFSe(a.Ctx, nota.ID, titulo()); err != nil || !gerou {
		t.Fatalf("primeiro título: gerou=%v err=%v", gerou, err)
	}
	if gerou, err := fin.GerarTituloDaNFSe(a.Ctx, nota.ID, titulo()); err != nil || gerou {
		t.Fatalf("segundo título da mesma nota: gerou=%v err=%v", gerou, err)
	}
	if l, err := fin.TitulosDaNFSe(b.Ctx, nota.ID); err != nil || len(l) != 0 {
		t.Fatalf("empresa B viu %d título(s) da NFS-e da A (%v)", len(l), err)
	}
	if err := fin.CancelarTitulosDaNFSe(b.Ctx, nota.ID); err != nil {
		t.Fatal(err)
	}
	l, err := fin.TitulosDaNFSe(a.Ctx, nota.ID)
	if err != nil || len(l) != 1 || l[0].Status != financialentity.ContaReceberStatusPendente || !l[0].ValorBruto.Equal(decimal.NewFromInt(970)) {
		t.Fatalf("título da A depois do cancelamento pela B: %+v %v", l, err)
	}
	if err := fin.CancelarTitulosDaNFSe(a.Ctx, nota.ID); err != nil {
		t.Fatal(err)
	}
	if l, _ := fin.TitulosDaNFSe(a.Ctx, nota.ID); len(l) != 0 {
		t.Fatalf("título continua ativo depois de cancelar a nota: %+v", l)
	}
}

package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	convservice "github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	uomrepo "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/uomgraph"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// FatorDeConversao resolve quanto 1 unidade `de` vale na unidade `para`, pelo
// motor de conversão (lote, item, padrão da empresa e cadeia entre eles).
func (r *FiscalRepositoryPG) FatorDeConversao(ctx context.Context, itemCode int64, de, para string) (*decimal.Decimal, error) {
	c, err := r.ConversaoDeUnidade(ctx, itemCode, "", time.Time{}, de, para)
	if err != nil || c == nil {
		return nil, err
	}
	f := c.Fator()
	return &f, nil
}

// ConversaoDeUnidade devolve o caminho inteiro (para gravar a origem e a
// explicação na linha da nota). `data` é a data da nota: conversão com
// vigência futura não vale para nota de hoje.
func (r *FiscalRepositoryPG) ConversaoDeUnidade(ctx context.Context, itemCode int64, mascara string, data time.Time, de, para string) (*convservice.Caminho, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	de, para = strings.ToUpper(strings.TrimSpace(de)), strings.ToUpper(strings.TrimSpace(para))
	if de == "" || para == "" {
		return nil, nil
	}
	c, err := uomgraph.Resolver(ctx, r.db, empresa, itemCode, strings.TrimSpace(mascara), "", data, de, para)
	if err != nil {
		return nil, fmt.Errorf("lendo a conversão %s→%s do item %d: %w", de, para, itemCode, err)
	}
	return c, nil
}

// NormalizarUnidades traduz o texto da nota (uCom/uTrib) para o catálogo.
func (r *FiscalRepositoryPG) NormalizarUnidades(ctx context.Context, fornecedor *int64, textos []string) (map[string]uomrepo.Resolucao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return uomgraph.Normalizar(ctx, r.db, empresa, fornecedor, textos)
}

// CustoDeReferencia: valor do produto por unidade de estoque na última nota
// aprovada do item. É a régua para desconfiar de um fator invertido.
func (r *FiscalRepositoryPG) CustoDeReferencia(ctx context.Context, itemCode int64, excetoNota int64) (*decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var v decimal.Decimal
	err = r.db.QueryRow(ctx, `SELECT round(fi.total_price / fi.quantidade_estoque, 6)
		  FROM fiscal_entry_items fi JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id
		 WHERE fi.item_code = $1 AND fe.enterprise_id = $2 AND fe.id <> $3
		   AND fe.status IN ('APPROVED','WRITTEN_OFF') AND fi.quantidade_estoque > 0 AND fi.total_price > 0
		 ORDER BY fe.data_entrada DESC, fe.id DESC LIMIT 1`, itemCode, empresa, excetoNota).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// FatorHistorico: o fator usado na última nota aprovada do mesmo fornecedor,
// para o mesmo item e a mesma unidade da nota.
func (r *FiscalRepositoryPG) FatorHistorico(ctx context.Context, fornecedor, itemCode int64, unidade string) (*decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var v decimal.Decimal
	err = r.db.QueryRow(ctx, `SELECT fi.fator_conversao
		  FROM fiscal_entry_items fi JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id
		 WHERE fi.item_code = $1 AND fe.enterprise_id = $2 AND fe.supplier_code = $3
		   AND fe.status IN ('APPROVED','WRITTEN_OFF') AND fi.fator_conversao > 0
		   AND upper(COALESCE(fi.unidade_normalizada, trim(fi.uom))) = $4
		 ORDER BY fe.data_entrada DESC, fe.id DESC LIMIT 1`, itemCode, empresa, fornecedor, strings.ToUpper(strings.TrimSpace(unidade))).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// PesoDoItem: peso líquido por unidade de estoque e a unidade do peso, do
// cadastro de engenharia do item (zero quando não informado).
func (r *FiscalRepositoryPG) PesoDoItem(ctx context.Context, itemCode int64) (decimal.Decimal, string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, "", err
	}
	var liquido *decimal.Decimal
	var unidade *string
	err = r.db.QueryRow(ctx, `SELECT NULLIF(engineering_weight->>'net','')::numeric, upper(NULLIF(trim(engineering_weight->>'unit'),''))
		  FROM items WHERE code = $1 AND enterprise_id = $2`, itemCode, empresa).Scan(&liquido, &unidade)
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, "", nil
	}
	if err != nil {
		return decimal.Zero, "", err
	}
	if liquido == nil || !liquido.IsPositive() {
		return decimal.Zero, "", nil
	}
	u := "KG"
	if unidade != nil {
		u = *unidade
	}
	return *liquido, u, nil
}

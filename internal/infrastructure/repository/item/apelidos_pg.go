package item

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// consultor é o que ApelidosDoCodigo precisa: serve o pool e a transação.
type consultor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// ApelidosDoCodigo diz de quem é um código que não é o de nenhum item: código
// anterior, referência do fabricante, código do fornecedor ou GTIN. Serve para a
// mensagem de "não encontrado" apontar o item certo. Não troca o código: o
// código anterior de um item pode ser o atual de outro, e dois fornecedores
// podem usar o mesmo código.
func ApelidosDoCodigo(ctx context.Context, db consultor, empresa int64, codigo string) ([]string, error) {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return nil, nil
	}
	rows, err := db.Query(ctx, `
		SELECT 'é o código anterior do item ' || i.business_code FROM items i
		 WHERE i.enterprise_id = $1 AND upper(btrim(i.previous_code)) = upper(btrim($2))
		UNION ALL
		SELECT 'é a referência do fabricante do item ' || i.business_code FROM items i
		 WHERE i.enterprise_id = $1 AND upper(btrim(i.manufacturer_part_number)) = upper(btrim($2))
		UNION ALL
		SELECT 'é o código do fornecedor ' || COALESCE(NULLIF(f.trade_name, ''), f.name, s.supplier_code::text) || ' para o item ' || i.business_code
		  FROM item_preferred_suppliers s
		  JOIN items i ON i.code = s.item_code AND i.enterprise_id = s.enterprise_id
		  LEFT JOIN suppliers f ON f.code = s.supplier_code AND f.enterprise_id = s.enterprise_id
		 WHERE s.enterprise_id = $1 AND upper(btrim(s.supplier_item_code)) = upper(btrim($2))
		UNION ALL
		SELECT 'é o GTIN do item ' || i.business_code FROM items i
		 WHERE i.enterprise_id = $1 AND i.accounting_gtin = btrim($2)
		UNION ALL
		SELECT 'é o GTIN da embalagem ' || g.uom || ' do item ' || i.business_code
		  FROM item_gtins g JOIN items i ON i.code = g.item_code AND i.enterprise_id = g.enterprise_id
		 WHERE g.enterprise_id = $1 AND g.gtin = btrim($2)
		LIMIT 5`, empresa, codigo)
	if err != nil {
		return nil, fmt.Errorf("procurando o código %s nos apelidos dos itens: %w", codigo, err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ApelidosDoCodigo, no repositório, usa a empresa do contexto.
func (r *RepositoryItemSQLC) ApelidosDoCodigo(ctx context.Context, codigo string) ([]string, error) {
	if r.pool == nil {
		return nil, nil
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return ApelidosDoCodigo(ctx, r.pool, empresa, codigo)
}

// linhaUnica é o que ConferirCodigoDoFornecedor precisa: serve pool e transação.
type linhaUnica interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ConferirCodigoDoFornecedor recusa o código do item no fornecedor que é o código
// atual de OUTRO item nosso: quem o digitasse num pedido ou numa nota levaria o
// outro item, sem aviso. O próprio item pode ter o mesmo código.
func ConferirCodigoDoFornecedor(ctx context.Context, db linhaUnica, empresa, itemCode int64, codigo string) error {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return nil
	}
	var outro, nome string
	err := db.QueryRow(ctx, `SELECT business_code, COALESCE(name, '') FROM items
		 WHERE enterprise_id = $1 AND upper(business_code) = upper($2) AND code <> $3 LIMIT 1`, empresa, codigo, itemCode).Scan(&outro, &nome)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("conferindo o código do fornecedor %s: %w", codigo, err)
	}
	return errorsuc.NewValidationError(fmt.Sprintf("o código %s do fornecedor é o código atual do nosso item %s (%s): quem digitasse %s num pedido ou numa nota levaria o item %s, e não este. Vincule este item sem o código do fornecedor ou peça ao fornecedor o código dele", codigo, outro, nome, codigo, outro))
}

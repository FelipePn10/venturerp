package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// LimiteDeAjusteDoUsuario lê o limite dos perfis ativos do usuário. Um perfil sem
// aquele limite libera a dimensão, como na união de permissões dos perfis.
func (r *StockRepositorySQLC) LimiteDeAjusteDoUsuario(ctx context.Context, usuario uuid.UUID) (*entity.LimiteDeAjuste, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var perfis int
	var semDocumento, semLinha bool
	var documento, linha *decimal.Decimal
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(BOOL_OR(t.max_document_value IS NULL),FALSE), MAX(t.max_document_value),
		        COALESCE(BOOL_OR(t.max_line_value IS NULL),FALSE), MAX(t.max_line_value)
		   FROM stock_tolerance_profiles t
		   JOIN acesso_perfis p ON p.enterprise_id=t.enterprise_id AND p.id=t.perfil_id AND p.ativo
		   JOIN acesso_usuario_perfis u ON u.enterprise_id=t.enterprise_id AND u.perfil_id=t.perfil_id
		  WHERE t.enterprise_id=$1 AND u.user_id=$2`, empresa, usuario).Scan(&perfis, &semDocumento, &documento, &semLinha, &linha)
	if err != nil {
		return nil, fmt.Errorf("lendo o limite de ajuste do usuário: %w", err)
	}
	if perfis == 0 {
		return nil, nil
	}
	l := &entity.LimiteDeAjuste{ValorDocumento: documento, ValorLinha: linha}
	if semDocumento {
		l.ValorDocumento = nil
	}
	if semLinha {
		l.ValorLinha = nil
	}
	return l, nil
}

// ToleranciasPorPerfil lista todos os perfis da empresa, com ou sem limite.
func (r *StockRepositorySQLC) ToleranciasPorPerfil(ctx context.Context) ([]entity.ToleranciaDoPerfil, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.nome, t.max_document_value, t.max_line_value
		   FROM acesso_perfis p
		   LEFT JOIN stock_tolerance_profiles t ON t.enterprise_id=p.enterprise_id AND t.perfil_id=p.id
		  WHERE p.enterprise_id=$1 AND p.ativo ORDER BY p.nome`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando as tolerâncias por perfil: %w", err)
	}
	defer rows.Close()
	out := []entity.ToleranciaDoPerfil{}
	for rows.Next() {
		var t entity.ToleranciaDoPerfil
		if err := rows.Scan(&t.PerfilID, &t.Nome, &t.ValorDocumento, &t.ValorLinha); err != nil {
			return nil, fmt.Errorf("lendo a tolerância do perfil: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SalvarToleranciaDoPerfil grava o limite. O perfil precisa ser da empresa: a chave
// estrangeira composta recusa o de outra.
func (r *StockRepositorySQLC) SalvarToleranciaDoPerfil(ctx context.Context, t entity.ToleranciaDoPerfil, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var existe bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM acesso_perfis WHERE enterprise_id=$1 AND id=$2)`, empresa, t.PerfilID).Scan(&existe); err != nil {
		return fmt.Errorf("conferindo o perfil: %w", err)
	}
	if !existe {
		return errorsuc.NewValidationError("o perfil não existe nesta empresa")
	}
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO stock_tolerance_profiles(enterprise_id,perfil_id,max_document_value,max_line_value,updated_by)
		 VALUES($1,$2,$3,$4,$5)
		 ON CONFLICT (enterprise_id,perfil_id) DO UPDATE
		   SET max_document_value=EXCLUDED.max_document_value, max_line_value=EXCLUDED.max_line_value,
		       updated_by=EXCLUDED.updated_by, updated_at=NOW()`,
		empresa, t.PerfilID, t.ValorDocumento, t.ValorLinha, ator); err != nil {
		return fmt.Errorf("gravando a tolerância do perfil: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) ExcluirToleranciaDoPerfil(ctx context.Context, perfil int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM stock_tolerance_profiles WHERE enterprise_id=$1 AND perfil_id=$2`, empresa, perfil); err != nil {
		return fmt.Errorf("apagando a tolerância do perfil: %w", err)
	}
	return nil
}

// Cadernos lista os cadernos do inventário com o avanço da digitação.
func (r *StockRepositorySQLC) Cadernos(ctx context.Context, inventario int64) ([]entity.CadernoInventario, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT n.number, n.warehouse_id, COUNT(l.id), COUNT(l.counted_qty), n.print_count, n.printed_at
		   FROM physical_inventory_notebooks n
		   JOIN physical_inventories i ON i.id=n.inventory_id AND i.enterprise_id=$1
		   LEFT JOIN physical_inventory_items l ON l.inventory_id=n.inventory_id AND l.notebook_number=n.number
		  WHERE n.inventory_id=$2
		  GROUP BY n.number, n.warehouse_id, n.print_count, n.printed_at
		  ORDER BY n.number`, empresa, inventario)
	if err != nil {
		return nil, fmt.Errorf("listando os cadernos: %w", err)
	}
	defer rows.Close()
	out := []entity.CadernoInventario{}
	for rows.Next() {
		var c entity.CadernoInventario
		if err := rows.Scan(&c.Numero, &c.Almoxarifado, &c.Linhas, &c.Contadas, &c.Impressoes, &c.ImpressoEm); err != nil {
			return nil, fmt.Errorf("lendo o caderno: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GerarCadernos reparte as linhas na ordem de quem anda pelo almoxarifado: endereço,
// item e lote. Linha contada não muda de caderno, então só gera antes da contagem.
func (r *StockRepositorySQLC) GerarCadernos(ctx context.Context, inventario int64, linhasPorCaderno int) ([]entity.CadernoInventario, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo a geração dos cadernos: %w", err)
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM physical_inventories WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, inventario, empresa).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("inventário não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o inventário: %w", err)
	}
	if status != entity.InventarioAberto {
		return nil, errorsuc.NewValidationError("os cadernos saem com o inventário aberto, antes da contagem")
	}
	var contadas int
	if err := tx.QueryRow(ctx, `SELECT COUNT(counted_qty) FROM physical_inventory_items WHERE inventory_id=$1`, inventario).Scan(&contadas); err != nil {
		return nil, fmt.Errorf("conferindo a contagem: %w", err)
	}
	if contadas > 0 {
		return nil, errorsuc.NewValidationError("a contagem já começou: os cadernos não podem mais ser refeitos")
	}
	rows, err := tx.Query(ctx, `SELECT id, warehouse_id FROM physical_inventory_items WHERE inventory_id=$1 ORDER BY warehouse_id, address, item_code, mask, lot`, inventario)
	if err != nil {
		return nil, fmt.Errorf("lendo as linhas do inventário: %w", err)
	}
	var ids, almox []int64
	for rows.Next() {
		var id, a int64
		if err := rows.Scan(&id, &a); err != nil {
			rows.Close()
			return nil, fmt.Errorf("lendo a linha do inventário: %w", err)
		}
		ids, almox = append(ids, id), append(almox, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errorsuc.NewValidationError("inventário sem linhas: inclua os itens antes de gerar os cadernos")
	}
	numeros := entity.DividirEmCadernos(almox, linhasPorCaderno)
	if _, err := tx.Exec(ctx, `DELETE FROM physical_inventory_notebooks WHERE inventory_id=$1`, inventario); err != nil {
		return nil, fmt.Errorf("refazendo os cadernos: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE physical_inventory_items l SET notebook_number=x.numero
		   FROM unnest($1::bigint[], $2::int[]) AS x(id, numero) WHERE l.id=x.id`, ids, numeros); err != nil {
		return nil, fmt.Errorf("gravando as linhas dos cadernos: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO physical_inventory_notebooks(inventory_id,number,warehouse_id)
		 SELECT $1, notebook_number, MIN(warehouse_id) FROM physical_inventory_items WHERE inventory_id=$1 GROUP BY notebook_number`, inventario); err != nil {
		return nil, fmt.Errorf("gravando os cadernos: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("gravando os cadernos: %w", err)
	}
	return r.Cadernos(ctx, inventario)
}

// MarcarCadernoImpresso conta a impressão: reimprimir é visível.
func (r *StockRepositorySQLC) MarcarCadernoImpresso(ctx context.Context, inventario int64, numero int, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE physical_inventory_notebooks n SET print_count=print_count+1, printed_at=NOW(), printed_by=$4
		  WHERE n.inventory_id=$1 AND n.number=$2
		    AND EXISTS (SELECT 1 FROM physical_inventories i WHERE i.id=n.inventory_id AND i.enterprise_id=$3)`,
		inventario, numero, empresa, ator)
	if err != nil {
		return fmt.Errorf("marcando a impressão do caderno: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("caderno não encontrado")
	}
	return nil
}

package itemgovernanca

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *PG) Embalagens(ctx context.Context, item int64) ([]dom.Embalagem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(gtin, ''), uom, package_quantity, description, parent_uom, parent_quantity FROM item_gtins
		 WHERE enterprise_id=$1 AND item_code=$2 ORDER BY package_quantity, uom`, empresa, item)
	if err != nil {
		return nil, fmt.Errorf("lendo os GTIN do item: %w", err)
	}
	defer rows.Close()
	out := []dom.Embalagem{}
	for rows.Next() {
		var e dom.Embalagem
		if err := rows.Scan(&e.GTIN, &e.Unidade, &e.Quantidade, &e.Descricao, &e.Contem, &e.ContemQuantidade); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PG) SalvarEmbalagens(ctx context.Context, item int64, lista []dom.Embalagem) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM item_gtins WHERE enterprise_id=$1 AND item_code=$2`, empresa, item); err != nil {
		return fmt.Errorf("limpando os GTIN do item: %w", err)
	}
	for _, e := range lista {
		if e.GTIN != "" {
			var dono string
			// O GTIN principal de outro item também conta: um código, um item.
			err := tx.QueryRow(ctx, `SELECT business_code FROM items WHERE enterprise_id=$1 AND accounting_gtin=$2 AND code<>$3 LIMIT 1`, empresa, e.GTIN, item).Scan(&dono)
			if err == nil {
				return errorsuc.NewValidationError(fmt.Sprintf("o GTIN %s já está no item %s", e.GTIN, dono))
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("conferindo o GTIN %s: %w", e.GTIN, err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO item_gtins (enterprise_id, item_code, gtin, uom, package_quantity, description, parent_uom, parent_quantity)
			VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8)`,
			empresa, item, e.GTIN, e.Unidade, e.Quantidade, e.Descricao, e.Contem, e.ContemQuantidade); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return errorsuc.NewValidationError(fmt.Sprintf("o GTIN %s já está em outro item desta empresa", e.GTIN))
			}
			return fmt.Errorf("gravando o GTIN %s: %w", e.GTIN, err)
		}
	}
	return tx.Commit(ctx)
}

func (r *PG) Revisoes(ctx context.Context, item int64) ([]dom.Revisao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT revision, valid_from, valid_until, reason, created_by, created_at FROM item_revisoes
		 WHERE enterprise_id=$1 AND item_code=$2 ORDER BY valid_from DESC`, empresa, item)
	if err != nil {
		return nil, fmt.Errorf("lendo as revisões do item: %w", err)
	}
	defer rows.Close()
	out := []dom.Revisao{}
	for rows.Next() {
		var v dom.Revisao
		if err := rows.Scan(&v.Revisao, &v.De, &v.Ate, &v.Motivo, &v.Por, &v.CriadaEm); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// CriarRevisao fecha a revisão aberta na véspera do início da nova e, se a nova já
// vale hoje, atualiza a revisão vigente do item — tudo junto.
func (r *PG) CriarRevisao(ctx context.Context, item int64, n dom.NovaRevisao, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE item_revisoes SET valid_until = $3::date - 1
		 WHERE enterprise_id=$1 AND item_code=$2 AND valid_until IS NULL`, empresa, item, n.De); err != nil {
		return fmt.Errorf("encerrando a revisão anterior: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO item_revisoes (enterprise_id, item_code, revision, valid_from, reason, created_by) VALUES ($1,$2,$3,$4::date,$5,$6)`,
		empresa, item, n.Revisao, n.De, n.Motivo, por); err != nil {
		return fmt.Errorf("gravando a revisão: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('venture.item_reason', $1, true)`, "Revisão "+n.Revisao+": "+n.Motivo); err != nil {
		return err
	}
	// A data de hoje é a do banco, no fuso da empresa: o processo pode rodar em UTC.
	if _, err := tx.Exec(ctx, `UPDATE items SET revision=$3 WHERE enterprise_id=$1 AND code=$2 AND $4::date <= CURRENT_DATE`, empresa, item, n.Revisao, n.De); err != nil {
		return fmt.Errorf("atualizando a revisão vigente: %w", err)
	}
	return tx.Commit(ctx)
}

// PromoverRevisoes leva a items.revision a revisão que começou a valer. Roda para
// todas as empresas: a revisão só atinge o item da própria empresa.
func (r *PG) PromoverRevisoes(ctx context.Context) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('venture.item_reason', 'Revisão vigente pela data', true)`); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `WITH vigente AS (
			SELECT DISTINCT ON (enterprise_id, item_code) enterprise_id, item_code, revision
			  FROM item_revisoes
			 WHERE valid_from <= CURRENT_DATE AND (valid_until IS NULL OR valid_until >= CURRENT_DATE)
			 ORDER BY enterprise_id, item_code, valid_from DESC)
		UPDATE items i SET revision = v.revision FROM vigente v
		 WHERE i.enterprise_id = v.enterprise_id AND i.code = v.item_code AND i.revision IS DISTINCT FROM v.revision`)
	if err != nil {
		return 0, fmt.Errorf("promovendo as revisões vigentes: %w", err)
	}
	return tag.RowsAffected(), tx.Commit(ctx)
}

// sinalDoMovimento dá o sinal do tipo; o ajuste já vem com o dele. Termina em "* "
// para multiplicar a coluna que o segue.
const sinalDoMovimento = `CASE WHEN m.movement_type IN ('IN','TRANSFER_IN','EP','EPP','EPE','ENTRADA','ADJUSTMENT') THEN 1
	WHEN m.movement_type IN ('OUT','TRANSFER_OUT','REP','SAIDA') THEN -1 ELSE 0 END * `

// SaldoDuasUnidades soma o saldo da unidade do estoque e os movimentos na segunda.
func (r *PG) SaldoDuasUnidades(ctx context.Context, item int64) (*dom.SaldoDuasUnidades, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	s := dom.SaldoDuasUnidades{Detalhes: []dom.SaldoPorLote{}}
	err = r.pool.QueryRow(ctx, `SELECT i.warehouse_unit_of_measurement::text, i.warehouse_secondary_uom,
			COALESCE((SELECT SUM(b.quantity) FROM stock_balances b WHERE b.enterprise_id=i.enterprise_id AND b.item_code=i.code), 0)::text,
			COALESCE((SELECT SUM(`+sinalDoMovimento+`m.secondary_quantity) FROM stock_movements m WHERE m.enterprise_id=i.enterprise_id AND m.item_code=i.code), 0)::text
		  FROM items i WHERE i.enterprise_id=$1 AND i.code=$2`, empresa, item).Scan(&s.Unidade, &s.SegundaUnidade, &s.Saldo, &s.SegundoSaldo)
	if err != nil {
		return nil, fmt.Errorf("lendo o saldo do item: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT m.warehouse_id, COALESCE(w.description, ''), NULLIF(m.lot, ''),
			SUM(`+sinalDoMovimento+`m.quantity)::text, COALESCE(SUM(`+sinalDoMovimento+`m.secondary_quantity), 0)::text
		  FROM stock_movements m LEFT JOIN warehouse w ON w.id = m.warehouse_id AND w.enterprise_id = m.enterprise_id
		 WHERE m.enterprise_id=$1 AND m.item_code=$2
		 GROUP BY m.warehouse_id, w.description, NULLIF(m.lot, '')
		HAVING SUM(`+sinalDoMovimento+`m.quantity) <> 0 OR COALESCE(SUM(`+sinalDoMovimento+`m.secondary_quantity), 0) <> 0
		 ORDER BY w.description, NULLIF(m.lot, '')`, empresa, item)
	if err != nil {
		return nil, fmt.Errorf("lendo o saldo por lote: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d dom.SaldoPorLote
		if err := rows.Scan(&d.AlmoxarifadoID, &d.Almoxarifado, &d.Lote, &d.Saldo, &d.SegundoSaldo); err != nil {
			return nil, err
		}
		s.Detalhes = append(s.Detalhes, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PG) AtributosLivres(ctx context.Context, item int64) ([]dom.AtributoLivre, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT name, value FROM item_atributos_livres WHERE enterprise_id=$1 AND item_code=$2 ORDER BY position, name`, empresa, item)
	if err != nil {
		return nil, fmt.Errorf("lendo os atributos livres do item: %w", err)
	}
	defer rows.Close()
	out := []dom.AtributoLivre{}
	for rows.Next() {
		var a dom.AtributoLivre
		if err := rows.Scan(&a.Nome, &a.Valor); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SalvarAtributosLivres troca a lista inteira do item.
func (r *PG) SalvarAtributosLivres(ctx context.Context, item int64, lista []dom.AtributoLivre) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM item_atributos_livres WHERE enterprise_id=$1 AND item_code=$2`, empresa, item); err != nil {
		return fmt.Errorf("limpando os atributos livres: %w", err)
	}
	for i, a := range lista {
		if _, err := tx.Exec(ctx, `INSERT INTO item_atributos_livres (enterprise_id, item_code, name, value, position) VALUES ($1,$2,$3,$4,$5)`,
			empresa, item, a.Nome, a.Valor, i); err != nil {
			return fmt.Errorf("gravando o atributo %s: %w", a.Nome, err)
		}
	}
	return tx.Commit(ctx)
}

// NomesDeAtributosLivres lista o que a empresa já usa, para "COR" não virar "CORES".
func (r *PG) NomesDeAtributosLivres(ctx context.Context) ([]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT name FROM item_atributos_livres WHERE enterprise_id=$1 ORDER BY name LIMIT 200`, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo os nomes de atributos livres: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ResumoDeAtributosLivres alimenta a busca de item das telas, que filtra no cliente.
// Apesar do nome, traz todos os apelidos do item, não só os atributos livres.
func (r *PG) ResumoDeAtributosLivres(ctx context.Context) ([]dom.ResumoDeAtributos, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Tudo por que se procura um item além do código e do nome: código anterior,
	// referência do fabricante, código do item em cada fornecedor, GTINs e os
	// atributos livres. As telas filtram a lista de itens no cliente com o texto e
	// mostram o tipo do código que achou o item.
	rows, err := r.pool.Query(ctx, `WITH a AS (
		SELECT i.code item, 'Código anterior' tipo, btrim(i.previous_code) valor, FALSE livre FROM items i
		 WHERE i.enterprise_id = $1 AND NULLIF(btrim(i.previous_code), '') IS NOT NULL
		UNION ALL SELECT i.code, 'Referência do fabricante', btrim(i.manufacturer_part_number), FALSE FROM items i
		 WHERE i.enterprise_id = $1 AND NULLIF(btrim(i.manufacturer_part_number), '') IS NOT NULL
		UNION ALL SELECT i.code, 'GTIN', btrim(i.accounting_gtin), FALSE FROM items i
		 WHERE i.enterprise_id = $1 AND NULLIF(btrim(i.accounting_gtin), '') IS NOT NULL
		UNION ALL SELECT i.code, 'GTIN tributável', btrim(i.accounting_gtin_tax), FALSE FROM items i
		 WHERE i.enterprise_id = $1 AND NULLIF(btrim(i.accounting_gtin_tax), '') IS NOT NULL
		UNION ALL SELECT DISTINCT s.item_code, 'Código do fornecedor ' || COALESCE(NULLIF(f.trade_name, ''), f.name, s.supplier_code::text), btrim(s.supplier_item_code), FALSE
		  FROM item_preferred_suppliers s LEFT JOIN suppliers f ON f.code = s.supplier_code AND f.enterprise_id = s.enterprise_id
		 WHERE s.enterprise_id = $1 AND NULLIF(btrim(s.supplier_item_code), '') IS NOT NULL
		UNION ALL SELECT g.item_code, 'GTIN da embalagem ' || g.uom, g.gtin, FALSE FROM item_gtins g
		 WHERE g.enterprise_id = $1 AND g.gtin IS NOT NULL
		UNION ALL SELECT l.item_code, l.name, l.value, TRUE FROM item_atributos_livres l WHERE l.enterprise_id = $1
	)
	SELECT i.business_code,
	       string_agg(CASE WHEN a.livre THEN a.tipo || ' ' || a.valor ELSE a.valor END, ' '),
	       json_agg(json_build_object('kind', a.tipo, 'value', a.valor))
	  FROM a JOIN items i ON i.code = a.item AND i.enterprise_id = $1
	 GROUP BY i.business_code`, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo o resumo de busca dos itens: %w", err)
	}
	defer rows.Close()
	out := []dom.ResumoDeAtributos{}
	for rows.Next() {
		var a dom.ResumoDeAtributos
		if err := rows.Scan(&a.Item, &a.Texto, &a.Apelidos); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CuidadosDosItens lista só os itens com cuidado cadastrado: são poucos, e as
// telas de recebimento, OF, separação e expedição os mostram em destaque.
func (r *PG) CuidadosDosItens(ctx context.Context) ([]dom.CuidadoDoItem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT business_code, COALESCE(name, ''), COALESCE(warehouse_hazardous, FALSE),
			NULLIF(btrim(warehouse_un_number), ''), NULLIF(btrim(warehouse_hazard_class), ''), NULLIF(btrim(warehouse_storage_conditions), '')
		  FROM items WHERE enterprise_id = $1
		   AND (COALESCE(warehouse_hazardous, FALSE) OR NULLIF(btrim(warehouse_storage_conditions), '') IS NOT NULL)
		 ORDER BY business_code`, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo os cuidados dos itens: %w", err)
	}
	defer rows.Close()
	out := []dom.CuidadoDoItem{}
	for rows.Next() {
		var c dom.CuidadoDoItem
		if err := rows.Scan(&c.Item, &c.Nome, &c.Perigoso, &c.ONU, &c.Classe, &c.Armazenagem); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Package itemgovernanca grava a governança do cadastro de item (status,
// visões liberadas, ajustes de campo obrigatório) e lê a ficha, o histórico e
// o uso do item — sempre da empresa da sessão.
package itemgovernanca

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/uomgraph"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

func (r *PG) Parametros(ctx context.Context) (dom.Parametros, error) {
	p := dom.Parametros{Visoes: append([]string{}, dom.Visoes...)}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return p, err
	}
	err = r.pool.QueryRow(ctx, `SELECT exige_liberacao, visoes FROM item_governanca_parametros WHERE enterprise_id=$1`, empresa).Scan(&p.ExigeLiberacao, &p.Visoes)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	return p, err
}

func (r *PG) SalvarParametros(ctx context.Context, p dom.Parametros, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO item_governanca_parametros (enterprise_id, exige_liberacao, visoes, updated_by) VALUES ($1,$2,$3,$4)
		ON CONFLICT (enterprise_id) DO UPDATE SET exige_liberacao=EXCLUDED.exige_liberacao, visoes=EXCLUDED.visoes, updated_by=EXCLUDED.updated_by, updated_at=NOW()`,
		empresa, p.ExigeLiberacao, p.Visoes, por)
	return err
}

func (r *PG) Politicas(ctx context.Context) ([]dom.RegraPolitica, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT status, processo, politica FROM item_status_politicas WHERE enterprise_id=$1`, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo a política do cadastro: %w", err)
	}
	defer rows.Close()
	out := []dom.RegraPolitica{}
	for rows.Next() {
		var p dom.RegraPolitica
		if err := rows.Scan(&p.Status, &p.Processo, &p.Politica); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PG) SalvarPoliticas(ctx context.Context, regras []dom.RegraPolitica, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, p := range regras {
		if _, err := tx.Exec(ctx, `INSERT INTO item_status_politicas (enterprise_id, status, processo, politica, updated_by) VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (enterprise_id, status, processo) DO UPDATE SET politica=EXCLUDED.politica, updated_by=EXCLUDED.updated_by, updated_at=NOW()`,
			empresa, p.Status, p.Processo, p.Politica, por); err != nil {
			return fmt.Errorf("gravando a política do cadastro: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *PG) Ajustes(ctx context.Context) (map[string]bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT campo, obrigatorio FROM item_regras_campo WHERE enterprise_id=$1`, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var c string
		var o bool
		if err := rows.Scan(&c, &o); err != nil {
			return nil, err
		}
		out[c] = o
	}
	return out, rows.Err()
}

// SalvarAjuste: nil volta ao padrão do catálogo.
func (r *PG) SalvarAjuste(ctx context.Context, campo string, obrigatorio *bool, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if obrigatorio == nil {
		_, err = r.pool.Exec(ctx, `DELETE FROM item_regras_campo WHERE enterprise_id=$1 AND campo=$2`, empresa, campo)
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO item_regras_campo (enterprise_id, campo, obrigatorio, updated_by) VALUES ($1,$2,$3,$4)
		ON CONFLICT (enterprise_id, campo) DO UPDATE SET obrigatorio=EXCLUDED.obrigatorio, updated_by=EXCLUDED.updated_by, updated_at=NOW()`,
		empresa, campo, *obrigatorio, por)
	return err
}

func (r *PG) Ficha(ctx context.Context, itemCode int64) (*dom.Ficha, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var dados []byte
	var parUM string
	f := dom.Ficha{}
	err = r.pool.QueryRow(ctx, `SELECT to_jsonb(i), i.engineering_type,
			EXISTS(SELECT 1 FROM item_structures s WHERE s.parent_code=i.code AND s.is_active),
			EXISTS(SELECT 1 FROM bom_headers h WHERE h.enterprise_id=i.enterprise_id AND h.item_code=i.code AND h.bom_type='MBOM' AND h.is_active),
			EXISTS(SELECT 1 FROM bom_headers h WHERE h.enterprise_id=i.enterprise_id AND h.item_code=i.code AND h.bom_type='MBOM' AND h.is_active
			       AND h.status='APPROVED' AND (h.valid_from IS NULL OR h.valid_from <= CURRENT_DATE)),
			COALESCE((SELECT EXISTS(SELECT 1 FROM item_structures s WHERE s.parent_code=i.code AND GREATEST(s.created_at, s.updated_at) > h.approved_at)
			                 OR (SELECT count(*) FROM item_structures s WHERE s.parent_code=i.code AND s.is_active AND (s.parent_mask IS NULL OR s.parent_mask = h.mask))
			                    <> (SELECT count(*) FROM bom_header_lines l WHERE l.bom_header_id=h.id)
			            FROM bom_headers h WHERE h.enterprise_id=i.enterprise_id AND h.item_code=i.code AND h.bom_type='MBOM' AND h.is_active AND h.status='APPROVED'
			            ORDER BY h.version DESC LIMIT 1), FALSE),
			EXISTS(SELECT 1 FROM manufacturing_routes m WHERE m.item_code=i.code AND m.enterprise_id=i.enterprise_id AND m.is_active),
			EXISTS(SELECT 1 FROM item_preferred_suppliers p WHERE p.item_code=i.code AND p.enterprise_id=i.enterprise_id AND p.is_active),
			EXISTS(SELECT 1 FROM item_standard_costs c WHERE c.item_code=i.code AND COALESCE(c.total_cost,0) > 0),
			upper(COALESCE(i.supplies_purchase_uom, '')) || '|' || i.warehouse_unit_of_measurement,
			COALESCE(i.supplies_purchase_uom, '') <> '' AND i.supplies_purchase_uom <> i.warehouse_unit_of_measurement::text,
			COALESCE((SELECT c.controls_lot FROM manufacturing_stock_item_controls c WHERE c.enterprise_id=i.enterprise_id AND c.item_code=i.code), FALSE),
			EXISTS(SELECT 1 FROM receiving_inspection_routes rt WHERE rt.enterprise_code=(SELECT e.code FROM enterprise e WHERE e.id=i.enterprise_id) AND rt.is_active AND (rt.item_code=i.code OR rt.item_code IS NULL)),
			EXISTS(SELECT 1 FROM item_engineering_drawings d WHERE d.enterprise_id=i.enterprise_id AND d.item_code=i.code),
			EXISTS(SELECT 1 FROM tools t WHERE t.enterprise_id=i.enterprise_id AND t.item_code=i.code AND t.is_active),
			COALESCE(fc.ncm, ''), COALESCE(fc.cod_clas_trib, ''), COALESCE(fc.cest, ''), COALESCE(fc.un_tributacao, ''), fc.default_origin,
			fc.ipi_rate::text, fc.default_icms_rate::text, fc.pis_rate::text, fc.cofins_rate::text
		  FROM items i
		  LEFT JOIN LATERAL (SELECT f.ncm, f.cod_clas_trib, f.cest, f.un_tributacao, f.default_origin, f.ipi_rate, f.default_icms_rate, f.pis_rate, f.cofins_rate FROM fiscal_classifications f
		       WHERE f.enterprise_id=i.enterprise_id AND f.is_active
		         AND (f.code::text=i.accounting_sale_fiscal_classification_code OR f.ncm=i.accounting_sale_fiscal_classification_code)
		       ORDER BY f.valid_from DESC, f.id DESC LIMIT 1) fc ON TRUE
		 WHERE i.code=$1 AND i.enterprise_id=$2`, itemCode, empresa).
		Scan(&dados, &f.Tipo, &f.TemEstrutura, &f.ControlaVersaoEstrutura, &f.TemMBOMAprovada, &f.EstruturaMudouAposAprov, &f.TemRoteiro, &f.TemFornecedor, &f.TemCustoPadrao, &parUM, &f.CompraEmOutraUM,
			&f.ControlaLote, &f.TemRotaInspecao, &f.TemDesenho, &f.TemFichaFerramenta, &f.Fiscal.NCM, &f.Fiscal.ClassTrib, &f.Fiscal.CEST, &f.Fiscal.Unidade, &f.Fiscal.Origem,
			&f.Fiscal.IPI, &f.Fiscal.ICMS, &f.Fiscal.PIS, &f.Fiscal.COFINS)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d", itemCode))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a ficha do item: %w", err)
	}
	if err := json.Unmarshal(dados, &f.Dados); err != nil {
		return nil, err
	}
	// Conversão da compra para o estoque pelo mesmo motor da compra e da nota:
	// direta, inversa, encadeada, padrão da empresa — vigente hoje. Antes só a
	// direta compra→estoque contava, e quem cadastrou "1 UN = 6 KG" (inversa
	// de KG→UN) via o item barrado na liberação.
	if compra, estoque, ok := strings.Cut(parUM, "|"); ok && f.CompraEmOutraUM && compra != "" {
		cam, err := uomgraph.Resolver(ctx, r.pool, empresa, itemCode, "", "", time.Time{}, compra, estoque)
		if err != nil {
			return nil, err
		}
		f.TemConversao = cam != nil
	}
	return &f, nil
}

func (r *PG) Situacao(ctx context.Context, itemCode int64) (*dom.Situacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	s := dom.Situacao{ItemCode: itemCode, Liberadas: map[string]dom.Marca{}}
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(i.business_code, i.code::text), COALESCE(i.name,''), i.engineering_type,
			COALESCE(g.status, 'LIBERADO'), g.motivo, g.substituto_code, si.business_code, g.alterado_em
		  FROM items i
		  LEFT JOIN item_governanca g ON g.item_code=i.code AND g.enterprise_id=i.enterprise_id
		  LEFT JOIN items si ON si.code=g.substituto_code AND si.enterprise_id=i.enterprise_id
		 WHERE i.code=$1 AND i.enterprise_id=$2`, itemCode, empresa).
		Scan(&s.Codigo, &s.Nome, &s.Tipo, &s.Status, &s.Motivo, &s.Substituto, &s.SubstitutoCd, &s.AlteradoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d", itemCode))
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT v.visao, v.liberado_por, COALESCE(u.name,''), v.liberado_em FROM item_visoes_liberadas v
		LEFT JOIN users u ON u.id=v.liberado_por WHERE v.enterprise_id=$1 AND v.item_code=$2`, empresa, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		var m dom.Marca
		if err := rows.Scan(&v, &m.Por, &m.Nome, &m.Em); err != nil {
			return nil, err
		}
		s.Liberadas[v] = m
	}
	return &s, rows.Err()
}

func evento(ctx context.Context, tx pgx.Tx, empresa, item int64, ev string, campo *string, antes, depois any, motivo *string, por *uuid.UUID) error {
	a, _ := json.Marshal(antes)
	d, _ := json.Marshal(depois)
	_, err := tx.Exec(ctx, `INSERT INTO item_historico (enterprise_id, item_code, evento, campo, antes, depois, motivo, por) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		empresa, item, ev, campo, a, d, motivo, por)
	return err
}

func str(s string) *string { return &s }

func (r *PG) IniciarCadastro(ctx context.Context, itemCode int64, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO item_governanca (enterprise_id, item_code, status, alterado_por) VALUES ($1,$2,'EM_CADASTRO',$3)
		ON CONFLICT (enterprise_id, item_code) DO NOTHING`, empresa, itemCode, por); err != nil {
		return err
	}
	if err := evento(ctx, tx, empresa, itemCode, "STATUS", str("status"), nil, dom.EmCadastro, nil, por); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// statusTx lê o status com trava (sem linha = LIBERADO).
func statusTx(ctx context.Context, tx pgx.Tx, empresa, item int64) (string, error) {
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2)`, item, empresa).Scan(&ok); err != nil {
		return "", err
	}
	if !ok {
		return "", errorsuc.NewNotFoundError(fmt.Sprintf("item %d", item))
	}
	var s string
	err := tx.QueryRow(ctx, `SELECT status FROM item_governanca WHERE enterprise_id=$1 AND item_code=$2 FOR UPDATE`, empresa, item).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return dom.Liberado, nil
	}
	return s, err
}

func gravarStatus(ctx context.Context, tx pgx.Tx, empresa, item int64, status string, motivo *string, subst *int64, por *uuid.UUID) error {
	_, err := tx.Exec(ctx, `INSERT INTO item_governanca (enterprise_id, item_code, status, motivo, substituto_code, alterado_por) VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (enterprise_id, item_code) DO UPDATE SET status=EXCLUDED.status, motivo=EXCLUDED.motivo, substituto_code=EXCLUDED.substituto_code,
		    alterado_por=EXCLUDED.alterado_por, alterado_em=NOW()`, empresa, item, status, motivo, subst, por)
	return err
}

func (r *PG) Liberar(ctx context.Context, itemCode int64, visao string, novoStatus string, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	atual, err := statusTx(ctx, tx, empresa, itemCode)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO item_visoes_liberadas (enterprise_id, item_code, visao, liberado_por) VALUES ($1,$2,$3,$4)
		ON CONFLICT DO NOTHING`, empresa, itemCode, visao, por)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("essa visão já está liberada")
	}
	if err := evento(ctx, tx, empresa, itemCode, "LIBERACAO", str(visao), nil, nil, nil, por); err != nil {
		return err
	}
	if novoStatus != "" && novoStatus != atual {
		if err := gravarStatus(ctx, tx, empresa, itemCode, novoStatus, nil, nil, por); err != nil {
			return err
		}
		if err := evento(ctx, tx, empresa, itemCode, "STATUS", str("status"), atual, novoStatus, str("todas as visões liberadas"), por); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PG) EstornarLiberacao(ctx context.Context, itemCode int64, visao, motivo string, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := statusTx(ctx, tx, empresa, itemCode); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM item_visoes_liberadas WHERE enterprise_id=$1 AND item_code=$2 AND visao=$3`, empresa, itemCode, visao)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("essa visão não está liberada")
	}
	if err := evento(ctx, tx, empresa, itemCode, "ESTORNO_LIBERACAO", str(visao), nil, nil, &motivo, por); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) MudarStatus(ctx context.Context, itemCode int64, esperado string, m dom.Mudanca) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	atual, err := statusTx(ctx, tx, empresa, itemCode)
	if err != nil {
		return err
	}
	if atual != esperado {
		return errorsuc.NewConflictError("o status do item mudou enquanto você editava: atualize a tela")
	}
	if err := m.Conferir(atual); err != nil {
		return err
	}
	if m.Substituto != nil {
		var ok bool
		var subStatus string
		err := tx.QueryRow(ctx, `SELECT TRUE, COALESCE(g.status,'LIBERADO') FROM items i LEFT JOIN item_governanca g ON g.item_code=i.code AND g.enterprise_id=i.enterprise_id
			WHERE i.code=$1 AND i.enterprise_id=$2`, *m.Substituto, empresa).Scan(&ok, &subStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("item substituto %d", *m.Substituto))
		}
		if err != nil {
			return err
		}
		if subStatus != dom.Liberado {
			return errorsuc.NewValidationError(fmt.Sprintf("o substituto precisa estar liberado (está %s)", dom.RotuloStatus(subStatus)))
		}
	}
	motivo := strings.TrimSpace(m.Motivo)
	if err := gravarStatus(ctx, tx, empresa, itemCode, m.Para, &motivo, m.Substituto, m.Por); err != nil {
		return err
	}
	if err := evento(ctx, tx, empresa, itemCode, "STATUS", str("status"), atual, m.Para, &motivo, m.Por); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) Historico(ctx context.Context, itemCode int64, limite int) ([]dom.Evento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if limite <= 0 || limite > 500 {
		limite = 200
	}
	rows, err := r.pool.Query(ctx, `SELECT h.id, h.evento, h.campo, h.antes, h.depois, h.motivo, h.por, COALESCE(u.name,''), h.ocorrido_em
		  FROM item_historico h LEFT JOIN users u ON u.id=h.por
		 WHERE h.enterprise_id=$1 AND h.item_code=$2 ORDER BY h.ocorrido_em DESC, h.id DESC LIMIT $3`, empresa, itemCode, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Evento{}
	for rows.Next() {
		var e dom.Evento
		var antes, depois []byte
		if err := rows.Scan(&e.ID, &e.Evento, &e.Campo, &antes, &depois, &e.Motivo, &e.Por, &e.Nome, &e.Em); err != nil {
			return nil, err
		}
		if len(antes) > 0 && string(antes) != "null" {
			e.Antes = antes
		}
		if len(depois) > 0 && string(depois) != "null" {
			e.Depois = depois
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PG) Uso(ctx context.Context, itemCode int64) (*dom.Uso, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	u := dom.Uso{EstruturasPai: []string{}}
	err = r.pool.QueryRow(ctx, `SELECT
		  COALESCE((SELECT SUM(quantity) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2), 0)::text,
		  (SELECT COUNT(DISTINCT so.code) FROM sales_order_items si JOIN sales_orders so ON so.code=si.sales_order_code
		    WHERE si.item_code=$2 AND so.enterprise_code=(SELECT code FROM enterprise WHERE id=$1)
		      AND si.status IN ('OPEN','PARTIAL')),
		  (SELECT COUNT(DISTINCT po.code) FROM purchase_order_items pi JOIN purchase_orders po ON po.code=pi.purchase_order_code
		    WHERE pi.item_code=$2 AND po.enterprise_code=(SELECT code FROM enterprise WHERE id=$1) AND pi.is_active AND po.is_active
		      AND po.status NOT IN ('CANCELLED','CANCELED','CLOSED') AND pi.requested_qty-pi.received_qty-pi.cancelled_qty > 0),
		  (SELECT COUNT(*) FROM production_orders o WHERE o.enterprise_id=$1 AND o.item_code=$2 AND o.status IN ('OPEN','IN_PROGRESS'))`,
		empresa, itemCode).Scan(&u.Saldo, &u.PedidosVenda, &u.PedidosCompra, &u.OrdensFabricacao)
	if err != nil {
		return nil, fmt.Errorf("lendo o uso do item: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT COALESCE(p.business_code, p.code::text) FROM item_structures s
		  JOIN items p ON p.code=s.parent_code AND p.enterprise_id=$1
		 WHERE s.child_code=$2 AND s.is_active ORDER BY 1 LIMIT 50`, empresa, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		u.EstruturasPai = append(u.EstruturasPai, c)
	}
	return &u, rows.Err()
}

func (r *PG) EmCadastro(ctx context.Context) ([]dom.Situacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT i.code, COALESCE(i.business_code, i.code::text), COALESCE(i.name,''), i.engineering_type, g.status, g.alterado_em
		  FROM item_governanca g JOIN items i ON i.code=g.item_code AND i.enterprise_id=g.enterprise_id
		 WHERE g.enterprise_id=$1 AND g.status='EM_CADASTRO' ORDER BY g.alterado_em, i.code LIMIT 500`, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Situacao{}
	for rows.Next() {
		s := dom.Situacao{Liberadas: map[string]dom.Marca{}}
		if err := rows.Scan(&s.ItemCode, &s.Codigo, &s.Nome, &s.Tipo, &s.Status, &s.AlteradoEm); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// visões liberadas de cada um
	lib, err := r.pool.Query(ctx, `SELECT item_code, visao, liberado_em FROM item_visoes_liberadas v
		 WHERE enterprise_id=$1 AND item_code IN (SELECT item_code FROM item_governanca WHERE enterprise_id=$1 AND status='EM_CADASTRO')`, empresa)
	if err != nil {
		return nil, err
	}
	defer lib.Close()
	idx := map[int64]int{}
	for i, s := range out {
		idx[s.ItemCode] = i
	}
	for lib.Next() {
		var item int64
		var v string
		var m dom.Marca
		if err := lib.Scan(&item, &v, &m.Em); err != nil {
			return nil, err
		}
		if i, ok := idx[item]; ok {
			out[i].Liberadas[v] = m
		}
	}
	return out, lib.Err()
}

func (r *PG) CandidatosParecidos(ctx context.Context, palavras []string, ncm string, excluir *int64) ([]dom.Candidato, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	padroes := make([]string, 0, len(palavras))
	for _, p := range palavras {
		if len(p) >= 3 {
			padroes = append(padroes, "%"+p+"%")
		}
	}
	if len(padroes) == 0 && strings.TrimSpace(ncm) == "" {
		return []dom.Candidato{}, nil
	}
	// translate tira os acentos para casar com as palavras normalizadas.
	rows, err := r.pool.Query(ctx, `SELECT i.code, COALESCE(i.business_code, i.code::text), COALESCE(NULLIF(i.name,''), i.pdm_description_technique, ''),
			COALESCE(i.accounting_sale_fiscal_classification_code, ''), COALESCE(g.status, 'LIBERADO')
		  FROM items i LEFT JOIN item_governanca g ON g.item_code=i.code AND g.enterprise_id=i.enterprise_id
		 WHERE i.enterprise_id=$1 AND ($4::bigint IS NULL OR i.code <> $4)
		   AND (lower(translate(COALESCE(i.name,'') || ' ' || COALESCE(i.pdm_description_technique,''), 'ÁÀÂÃÄÉÊÈËÍÎÌÏÓÔÕÒÖÚÛÙÜÇáàâãäéêèëíîìïóôõòöúûùüç', 'AAAAAEEEEIIIIOOOOOUUUUCaaaaaeeeeiiiiooooouuuuc')) LIKE ANY($2)
		        OR ($3 <> '' AND i.accounting_sale_fiscal_classification_code = $3))
		 LIMIT 300`, empresa, padroes, strings.TrimSpace(ncm), excluir)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Candidato{}
	for rows.Next() {
		var c dom.Candidato
		if err := rows.Scan(&c.ItemCode, &c.Codigo, &c.Nome, &c.NCM, &c.Status); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

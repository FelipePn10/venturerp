// Package serie grava números de série, base instalada e o perfil de
// pós-venda do item, sempre da empresa da sessão.
//
// As mesmas operações rodam no pool (telas) e dentro de outra transação: a
// entrega da OF cria as séries na transação dela e os efeitos da NF de saída
// mudam a série na transação dos efeitos — ou tudo entra, ou nada.
package serie

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	lotmask "github.com/FelipePn10/panossoerp/internal/domain/lot_mask/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/posvenda/garantia"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/serie"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// DB é o que o pool e a transação têm em comum.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PG struct {
	pool *pgxpool.Pool
	db   DB
}

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool, db: pool} }

// NewTx: o mesmo repositório dentro de uma transação já aberta.
func NewTx(tx pgx.Tx) *PG { return &PG{db: tx} }

var _ dom.Repository = (*PG)(nil)

// emTransacao: no pool abre a transação; já dentro de uma, usa a de fora.
func (r *PG) emTransacao(ctx context.Context, fn func(DB) error) error {
	if r.pool == nil {
		return fn(r.db)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ─── Perfil ──────────────────────────────────────────────────────────────────

func (r *PG) Perfil(ctx context.Context, itemCode int64) (*dom.Perfil, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	p := dom.Perfil{ItemCode: itemCode}
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(business_code,''), COALESCE(name,''), COALESCE(commercial_warranty_days,0) FROM items WHERE code=$1 AND enterprise_id=$2`,
		itemCode, empresa).Scan(&p.ItemCodigo, &p.ItemNome, &p.Garantia.DiasDoItem)
	p.Garantia.Inicio, p.Garantia.CobrePecas, p.Garantia.CobreMaoObra = garantia.InicioVenda, true, true
	err = r.db.QueryRow(ctx, `SELECT controla_serie, mascara_serie_id, observacoes, updated_at, updated_by,
			garantia_meses, garantia_inicio, cobre_pecas, cobre_mao_obra, cobre_deslocamento
		  FROM item_pos_venda WHERE enterprise_id=$1 AND item_code=$2`, empresa, itemCode).
		Scan(&p.ControlaSerie, &p.MascaraSerieID, &p.Observacoes, &p.UpdatedAt, &p.UpdatedBy,
			&p.Garantia.Meses, &p.Garantia.Inicio, &p.Garantia.CobrePecas, &p.Garantia.CobreMaoObra, &p.Garantia.CobreDeslocamento)
	if errors.Is(err, pgx.ErrNoRows) {
		return &p, nil // sem perfil: item sem série
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o perfil de pós-venda do item %d: %w", itemCode, err)
	}
	return &p, nil
}

func (r *PG) SalvarPerfil(ctx context.Context, p *dom.Perfil) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if p.MascaraSerieID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM lot_masks WHERE id=$1 AND enterprise_id=$2 AND is_active)`, *p.MascaraSerieID, empresa).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewValidationError(fmt.Sprintf("a máscara de série %d não existe nesta empresa ou está inativa", *p.MascaraSerieID))
		}
	}
	var existe bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2)`, p.ItemCode, empresa).Scan(&existe); err != nil {
		return err
	}
	if !existe {
		return errorsuc.NewNotFoundError(fmt.Sprintf("item %d", p.ItemCode))
	}
	g := p.Garantia
	if g.Inicio == "" {
		g.Inicio = garantia.InicioVenda
	}
	return r.db.QueryRow(ctx, `INSERT INTO item_pos_venda (enterprise_id, item_code, controla_serie, mascara_serie_id, observacoes, updated_by,
			garantia_meses, garantia_inicio, cobre_pecas, cobre_mao_obra, cobre_deslocamento)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (enterprise_id, item_code) DO UPDATE SET controla_serie=EXCLUDED.controla_serie,
			mascara_serie_id=EXCLUDED.mascara_serie_id, observacoes=EXCLUDED.observacoes, updated_by=EXCLUDED.updated_by,
			garantia_meses=EXCLUDED.garantia_meses, garantia_inicio=EXCLUDED.garantia_inicio, cobre_pecas=EXCLUDED.cobre_pecas,
			cobre_mao_obra=EXCLUDED.cobre_mao_obra, cobre_deslocamento=EXCLUDED.cobre_deslocamento, updated_at=NOW()
		RETURNING updated_at`, empresa, p.ItemCode, p.ControlaSerie, p.MascaraSerieID, p.Observacoes, p.UpdatedBy,
		g.Meses, g.Inicio, g.CobrePecas, g.CobreMaoObra, g.CobreDeslocamento).Scan(&p.UpdatedAt)
}

func (r *PG) Perfis(ctx context.Context) ([]dom.Perfil, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT p.item_code, COALESCE(i.business_code,''), COALESCE(i.name,''), p.controla_serie, p.mascara_serie_id,
			p.observacoes, p.updated_at, p.updated_by, p.garantia_meses, p.garantia_inicio, p.cobre_pecas, p.cobre_mao_obra, p.cobre_deslocamento,
			COALESCE(i.commercial_warranty_days,0)
		  FROM item_pos_venda p LEFT JOIN items i ON i.code = p.item_code AND i.enterprise_id = p.enterprise_id
		 WHERE p.enterprise_id=$1 ORDER BY i.business_code, p.item_code`, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Perfil{}
	for rows.Next() {
		var p dom.Perfil
		if err := rows.Scan(&p.ItemCode, &p.ItemCodigo, &p.ItemNome, &p.ControlaSerie, &p.MascaraSerieID, &p.Observacoes, &p.UpdatedAt, &p.UpdatedBy,
			&p.Garantia.Meses, &p.Garantia.Inicio, &p.Garantia.CobrePecas, &p.Garantia.CobreMaoObra, &p.Garantia.CobreDeslocamento, &p.Garantia.DiasDoItem); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ─── Séries ──────────────────────────────────────────────────────────────────

const colunas = `s.id, s.item_code, COALESCE(i.business_code,''), COALESCE(i.name,''), s.mask, s.numero_serie, s.situacao, s.origem, s.warehouse_id, s.lote,
	s.production_order_id, s.fiscal_entry_id, s.fiscal_exit_id, fe.numero_nf, s.customer_code, COALESCE(c.name,''),
	s.data_venda, s.data_instalacao, s.local_instalacao, s.observacoes, s.garantia_estendida_ate, s.garantia_estendida_motivo, s.created_at, s.updated_at`

const de = ` FROM numeros_serie s
	LEFT JOIN items i ON i.code = s.item_code AND i.enterprise_id = s.enterprise_id
	LEFT JOIN customers c ON c.code = s.customer_code AND c.enterprise_id = s.enterprise_id
	LEFT JOIN fiscal_exits fe ON fe.id = s.fiscal_exit_id AND fe.enterprise_id = s.enterprise_id`

func scan(row pgx.Row) (*dom.Serie, error) {
	var s dom.Serie
	err := row.Scan(&s.ID, &s.ItemCode, &s.ItemCodigo, &s.ItemNome, &s.Mask, &s.NumeroSerie, &s.Situacao, &s.Origem, &s.WarehouseID, &s.Lote,
		&s.ProductionOrderID, &s.FiscalEntryID, &s.FiscalExitID, &s.NumeroNF, &s.CustomerCode, &s.ClienteNome,
		&s.DataVenda, &s.DataInstalacao, &s.LocalInstalacao, &s.Observacoes, &s.EstendidaAte, &s.EstendidaMotivo, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.Serie, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := scan(r.db.QueryRow(ctx, `SELECT `+colunas+de+` WHERE s.enterprise_id=$1 AND s.id=$2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("série %d", id))
	}
	return s, err
}

func (r *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.Serie, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	limite := f.Limite
	if limite <= 0 || limite > 1000 {
		limite = 300
	}
	rows, err := r.db.Query(ctx, `SELECT `+colunas+de+`
		 WHERE s.enterprise_id=$1
		   AND ($2::bigint IS NULL OR s.item_code=$2)
		   AND ($3::bigint IS NULL OR s.customer_code=$3)
		   AND ($4='' OR s.situacao=$4)
		   AND ($5='' OR upper(s.numero_serie) LIKE '%'||upper($5)||'%' OR upper(COALESCE(i.name,'')) LIKE '%'||upper($5)||'%'
		        OR upper(COALESCE(c.name,'')) LIKE '%'||upper($5)||'%')
		 ORDER BY s.updated_at DESC, s.id DESC LIMIT $6`, empresa, f.ItemCode, f.CustomerCode, strings.ToUpper(f.Situacao), strings.TrimSpace(f.Busca), limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Serie{}
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *PG) Historico(ctx context.Context, id int64) ([]dom.Evento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT id, evento, situacao_antes, situacao_depois, customer_code, warehouse_id, referencia_tipo, referencia_id,
			motivo, ocorrido_em, created_by
		  FROM numeros_serie_eventos WHERE enterprise_id=$1 AND serie_id=$2 ORDER BY ocorrido_em, id`, empresa, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Evento{}
	for rows.Next() {
		var e dom.Evento
		if err := rows.Scan(&e.ID, &e.Evento, &e.SituacaoAntes, &e.SituacaoDepois, &e.CustomerCode, &e.WarehouseID, &e.ReferenciaTipo,
			&e.ReferenciaID, &e.Motivo, &e.OcorridoEm, &e.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func erroDeDuplicada(err error, item int64, serie string) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewValidationError(fmt.Sprintf("a série %s já existe para o item %d", serie, item))
	}
	return err
}

func evento(ctx context.Context, db DB, empresa, serieID int64, ev string, antes *string, depois string, cliente, almox *int64,
	refTipo string, refID *int64, motivo *string, por *uuid.UUID) error {
	var tipo *string
	if refTipo != "" {
		tipo = &refTipo
	}
	_, err := db.Exec(ctx, `INSERT INTO numeros_serie_eventos (enterprise_id, serie_id, evento, situacao_antes, situacao_depois, customer_code,
			warehouse_id, referencia_tipo, referencia_id, motivo, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, empresa, serieID, ev, antes, depois, cliente, almox, tipo, refID, motivo, por)
	if err != nil {
		return fmt.Errorf("gravando o histórico da série: %w", err)
	}
	return nil
}

// Cadastrar: série digitada (base legada já no cliente, ou estoque comprado).
func (r *PG) Cadastrar(ctx context.Context, s *dom.Serie, por *uuid.UUID, motivo *string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.emTransacao(ctx, func(db DB) error {
		var itemOK bool
		if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2)`, s.ItemCode, empresa).Scan(&itemOK); err != nil {
			return err
		}
		if !itemOK {
			return errorsuc.NewNotFoundError(fmt.Sprintf("item %d", s.ItemCode))
		}
		if s.CustomerCode != nil {
			var ok bool
			if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE code=$1 AND enterprise_id=$2)`, *s.CustomerCode, empresa).Scan(&ok); err != nil {
				return err
			}
			if !ok {
				return errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d", *s.CustomerCode))
			}
		}
		if s.WarehouseID != nil {
			var ok bool
			if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouse WHERE id=$1 AND enterprise_id=$2)`, *s.WarehouseID, empresa).Scan(&ok); err != nil {
				return err
			}
			if !ok {
				return errorsuc.NewNotFoundError(fmt.Sprintf("almoxarifado %d", *s.WarehouseID))
			}
		}
		err := db.QueryRow(ctx, `INSERT INTO numeros_serie (enterprise_id, item_code, mask, numero_serie, situacao, origem, warehouse_id, lote,
				fiscal_entry_id, customer_code, data_venda, data_instalacao, local_instalacao, observacoes, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$15) RETURNING id, created_at, updated_at`,
			empresa, s.ItemCode, s.Mask, s.NumeroSerie, s.Situacao, s.Origem, s.WarehouseID, s.Lote, s.FiscalEntryID, s.CustomerCode,
			s.DataVenda, s.DataInstalacao, s.LocalInstalacao, s.Observacoes, por).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return erroDeDuplicada(err, s.ItemCode, s.NumeroSerie)
		}
		return evento(ctx, db, empresa, s.ID, dom.EventoCadastro, nil, s.Situacao, s.CustomerCode, s.WarehouseID, "", nil, motivo, por)
	})
}

// Mudar aplica uma transição com a série travada e grava o evento junto.
func (r *PG) Mudar(ctx context.Context, id int64, m dom.Mudanca) (*dom.Serie, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.emTransacao(ctx, func(db DB) error {
		return mudar(ctx, db, empresa, id, m)
	})
	if err != nil {
		return nil, err
	}
	return r.Obter(ctx, id)
}

func mudar(ctx context.Context, db DB, empresa, id int64, m dom.Mudanca) error {
	var atual string
	var cliente *int64
	err := db.QueryRow(ctx, `SELECT situacao, customer_code FROM numeros_serie WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, id).Scan(&atual, &cliente)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("série %d", id))
	}
	if err != nil {
		return err
	}
	depois, err := dom.Transicao(atual, m.Evento)
	if err != nil {
		return err
	}
	if m.CustomerCode != nil {
		var ok bool
		if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE code=$1 AND enterprise_id=$2)`, *m.CustomerCode, empresa).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d", *m.CustomerCode))
		}
		cliente = m.CustomerCode
	}
	if m.LimparVenda {
		cliente = nil
	}
	_, err = db.Exec(ctx, `UPDATE numeros_serie SET situacao=$3,
			warehouse_id=COALESCE($4, warehouse_id),
			customer_code=$5,
			fiscal_exit_id=CASE WHEN $6 THEN NULL ELSE COALESCE($7, fiscal_exit_id) END,
			data_venda=CASE WHEN $6 THEN NULL ELSE COALESCE($8, data_venda) END,
			data_instalacao=COALESCE($9, data_instalacao),
			local_instalacao=COALESCE($10, local_instalacao),
			updated_by=$11, updated_at=NOW()
		WHERE enterprise_id=$1 AND id=$2`, empresa, id, depois, m.WarehouseID, cliente, m.LimparVenda, m.FiscalExitID, m.DataVenda,
		m.DataInstalacao, m.LocalInstalacao, m.Por)
	if err != nil {
		return fmt.Errorf("atualizando a série: %w", err)
	}
	return evento(ctx, db, empresa, id, m.Evento, &atual, depois, cliente, m.WarehouseID, m.ReferenciaTipo, m.ReferenciaID, m.Motivo, m.Por)
}

// AssistenciaTx: a série do cliente entra na assistência (OS na fábrica) ou
// volta ao cliente, dentro da transação da OS. Série que não está com o
// cliente no sistema (cadastrada depois, baixada) não muda: a OS segue.
func AssistenciaTx(ctx context.Context, tx pgx.Tx, empresa, serieID, osID int64, entrar bool, por *uuid.UUID) error {
	var atual string
	err := tx.QueryRow(ctx, `SELECT situacao FROM numeros_serie WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, serieID).Scan(&atual)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("série %d", serieID))
	}
	if err != nil {
		return err
	}
	ev, de := dom.EventoAssistencia, dom.NoCliente
	if !entrar {
		ev, de = dom.EventoRetornoAssistencia, dom.EmAssistencia
	}
	if atual != de {
		return nil
	}
	ref := osID
	return mudar(ctx, tx, empresa, serieID, dom.Mudanca{Evento: ev, ReferenciaTipo: "ASSISTENCIA_OS", ReferenciaID: &ref, Por: por})
}

// DevolucaoTx: as séries devolvidas pelo cliente voltam ao estoque do
// almoxarifado da devolução, na transação da devolução de venda. Série que não
// está mais com o cliente (em assistência, baixada) recusa a devolução.
func DevolucaoTx(ctx context.Context, tx pgx.Tx, empresa int64, series []int64, warehouseID, devolucaoID int64, por *uuid.UUID) error {
	motivo := "devolução de venda"
	ref := devolucaoID
	for _, id := range series {
		var atual, numero string
		err := tx.QueryRow(ctx, `SELECT situacao, numero_serie FROM numeros_serie WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, id).Scan(&atual, &numero)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("série %d", id))
		}
		if err != nil {
			return err
		}
		if atual != dom.NoCliente {
			return errorsuc.NewValidationError(fmt.Sprintf("a série %s está %s: só volta pela devolução a série que está com o cliente", numero, dom.Rotulo(atual)))
		}
		w := warehouseID
		if err := mudar(ctx, tx, empresa, id, dom.Mudanca{Evento: dom.EventoDevolucao, LimparVenda: true, WarehouseID: &w,
			ReferenciaTipo: "DEVOLUCAO_VENDA", ReferenciaID: &ref, Motivo: &motivo, Por: por}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE fiscal_exit_item_series SET ativo=FALSE WHERE enterprise_id=$1 AND serie_id=$2 AND ativo`, empresa, id); err != nil {
			return err
		}
	}
	return nil
}

// ─── NF de saída ─────────────────────────────────────────────────────────────

// LinhasComSerie: as linhas da nota cujo item controla série, com as séries
// escolhidas para cada uma.
func (r *PG) LinhasComSerie(ctx context.Context, fiscalExitID int64) ([]dom.LinhaNota, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT it.id, it.item_code, it.quantity
		  FROM fiscal_exit_items it
		  JOIN fiscal_exits fe ON fe.id = it.fiscal_exit_id AND fe.enterprise_id = $1
		  JOIN item_pos_venda p ON p.enterprise_id = $1 AND p.item_code = it.item_code AND p.controla_serie
		 WHERE it.fiscal_exit_id = $2
		 ORDER BY it.sequence, it.id`, empresa, fiscalExitID)
	if err != nil {
		return nil, err
	}
	linhas := []dom.LinhaNota{}
	for rows.Next() {
		var l dom.LinhaNota
		if err := rows.Scan(&l.FiscalExitItemID, &l.ItemCode, &l.Quantidade); err != nil {
			rows.Close()
			return nil, err
		}
		l.Series = []dom.Serie{}
		linhas = append(linhas, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range linhas {
		srows, err := r.db.Query(ctx, `SELECT `+colunas+de+`
			  JOIN fiscal_exit_item_series v ON v.serie_id = s.id AND v.ativo AND v.fiscal_exit_item_id = $2
			 WHERE s.enterprise_id = $1 ORDER BY s.numero_serie`, empresa, linhas[i].FiscalExitItemID)
		if err != nil {
			return nil, err
		}
		for srows.Next() {
			s, err := scan(srows)
			if err != nil {
				srows.Close()
				return nil, err
			}
			linhas[i].Series = append(linhas[i].Series, *s)
		}
		srows.Close()
		if err := srows.Err(); err != nil {
			return nil, err
		}
	}
	return linhas, nil
}

// DefinirSeriesDaLinha troca as séries escolhidas na linha (nota em rascunho).
func (r *PG) DefinirSeriesDaLinha(ctx context.Context, fiscalExitID, fiscalExitItemID int64, series []int64, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.emTransacao(ctx, func(db DB) error {
		var itemCode *int64
		var status string
		err := db.QueryRow(ctx, `SELECT it.item_code, fe.status FROM fiscal_exit_items it
			  JOIN fiscal_exits fe ON fe.id = it.fiscal_exit_id AND fe.enterprise_id = $1
			 WHERE it.id = $2 AND it.fiscal_exit_id = $3 FOR UPDATE OF fe`, empresa, fiscalExitItemID, fiscalExitID).Scan(&itemCode, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("linha %d da nota %d", fiscalExitItemID, fiscalExitID))
		}
		if err != nil {
			return err
		}
		if s := strings.ToUpper(status); s != "DRAFT" && s != "REJECTED" && s != "PENDENTE" {
			return errorsuc.NewValidationError("as séries só mudam com a nota em rascunho ou rejeitada")
		}
		if _, err := db.Exec(ctx, `UPDATE fiscal_exit_item_series SET ativo=FALSE WHERE enterprise_id=$1 AND fiscal_exit_item_id=$2 AND ativo`, empresa, fiscalExitItemID); err != nil {
			return err
		}
		for _, id := range series {
			var item int64
			var situacao, numero string
			err := db.QueryRow(ctx, `SELECT item_code, situacao, numero_serie FROM numeros_serie WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, id).Scan(&item, &situacao, &numero)
			if errors.Is(err, pgx.ErrNoRows) {
				return errorsuc.NewNotFoundError(fmt.Sprintf("série %d", id))
			}
			if err != nil {
				return err
			}
			if itemCode == nil || item != *itemCode {
				return errorsuc.NewValidationError(fmt.Sprintf("a série %s é de outro item", numero))
			}
			if situacao != dom.EmEstoque {
				return errorsuc.NewValidationError(fmt.Sprintf("a série %s está %s; só série em estoque sai na nota", numero, dom.Rotulo(situacao)))
			}
			_, err = db.Exec(ctx, `INSERT INTO fiscal_exit_item_series (enterprise_id, fiscal_exit_id, fiscal_exit_item_id, serie_id, created_by)
				VALUES ($1,$2,$3,$4,$5)`, empresa, fiscalExitID, fiscalExitItemID, id, por)
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23505" {
				var nota int64
				_ = db.QueryRow(ctx, `SELECT fe.numero_nf FROM fiscal_exit_item_series v JOIN fiscal_exits fe ON fe.id=v.fiscal_exit_id WHERE v.serie_id=$1 AND v.ativo`, id).Scan(&nota)
				return errorsuc.NewValidationError(fmt.Sprintf("a série %s já está na nota %d", numero, nota))
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// SaidaPelaNota: na autorização, as séries da nota vão para o cliente.
func (r *PG) SaidaPelaNota(ctx context.Context, fiscalExitID int64, cliente *int64, emissao time.Time, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.emTransacao(ctx, func(db DB) error {
		ids, err := seriesDaNota(ctx, db, empresa, fiscalExitID)
		if err != nil {
			return err
		}
		if len(ids) > 0 && cliente == nil {
			return errorsuc.NewValidationError("a nota tem item com série e não tem o cliente do cadastro: a série não tem a quem ser vinculada")
		}
		dia := time.Date(emissao.Year(), emissao.Month(), emissao.Day(), 0, 0, 0, 0, time.UTC)
		for _, id := range ids {
			if err := mudar(ctx, db, empresa, id, dom.Mudanca{Evento: dom.EventoVenda, CustomerCode: cliente, FiscalExitID: &fiscalExitID,
				DataVenda: &dia, ReferenciaTipo: "FISCAL_EXIT", ReferenciaID: &fiscalExitID, Por: por}); err != nil {
				return err
			}
		}
		return nil
	})
}

// EstornoDaNota: a nota cancelada devolve as séries ao estoque e as solta.
func (r *PG) EstornoDaNota(ctx context.Context, fiscalExitID int64, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.emTransacao(ctx, func(db DB) error {
		rows, err := db.Query(ctx, `SELECT id FROM numeros_serie WHERE enterprise_id=$1 AND fiscal_exit_id=$2 AND situacao=$3`, empresa, fiscalExitID, dom.NoCliente)
		if err != nil {
			return err
		}
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			motivo := "nota cancelada"
			if err := mudar(ctx, db, empresa, id, dom.Mudanca{Evento: dom.EventoEstornoVenda, LimparVenda: true, ReferenciaTipo: "FISCAL_EXIT",
				ReferenciaID: &fiscalExitID, Motivo: &motivo, Por: por}); err != nil {
				return err
			}
		}
		_, err = db.Exec(ctx, `UPDATE fiscal_exit_item_series SET ativo=FALSE WHERE enterprise_id=$1 AND fiscal_exit_id=$2 AND ativo`, empresa, fiscalExitID)
		return err
	})
}

func seriesDaNota(ctx context.Context, db DB, empresa, fiscalExitID int64) ([]int64, error) {
	rows, err := db.Query(ctx, `SELECT serie_id FROM fiscal_exit_item_series WHERE enterprise_id=$1 AND fiscal_exit_id=$2 AND ativo ORDER BY id`, empresa, fiscalExitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SeriesPorLinha: número das séries de cada linha (para o infAdProd do XML).
func (r *PG) SeriesPorLinha(ctx context.Context, fiscalExitID int64) (map[int64][]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT v.fiscal_exit_item_id, s.numero_serie FROM fiscal_exit_item_series v
		  JOIN numeros_serie s ON s.id = v.serie_id AND s.enterprise_id = v.enterprise_id
		 WHERE v.enterprise_id=$1 AND v.fiscal_exit_id=$2 AND v.ativo ORDER BY v.fiscal_exit_item_id, s.numero_serie`, empresa, fiscalExitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]string{}
	for rows.Next() {
		var linha int64
		var n string
		if err := rows.Scan(&linha, &n); err != nil {
			return nil, err
		}
		out[linha] = append(out[linha], n)
	}
	return out, rows.Err()
}

// ─── Produção ────────────────────────────────────────────────────────────────

// EntradaPelaProducaoTx cria as séries da entrega da OF na transação dela.
// Sem séries informadas e com máscara no perfil, gera uma por unidade pela
// máscara, travando as partes para a sequência não repetir nem pular.
func EntradaPelaProducaoTx(ctx context.Context, tx pgx.Tx, empresa, orderID, itemCode int64, mask *string, warehouseID int64,
	lote *string, quantidade decimal.Decimal, informadas []string, por *uuid.UUID, agora time.Time) ([]string, error) {
	var controla bool
	var mascara *int64
	err := tx.QueryRow(ctx, `SELECT controla_serie, mascara_serie_id FROM item_pos_venda WHERE enterprise_id=$1 AND item_code=$2`, empresa, itemCode).Scan(&controla, &mascara)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !controla) {
		if len(informadas) > 0 {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d não controla número de série (VATC0110)", itemCode))
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	series := informadas
	if len(strings.Join(series, "")) == 0 && quantidade.IsPositive() {
		if mascara == nil {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d controla número de série: informe as séries da entrega ou cadastre a máscara de série no perfil (VATC0110)", itemCode))
		}
		if series, err = gerarPelaMascara(ctx, tx, empresa, *mascara, int(quantidade.IntPart()), agora); err != nil {
			return nil, err
		}
	}
	series, err = dom.ConferirLista(series, quantidade)
	if err != nil {
		return nil, err
	}
	for _, n := range series {
		var id int64
		err := tx.QueryRow(ctx, `INSERT INTO numeros_serie (enterprise_id, item_code, mask, numero_serie, situacao, origem, warehouse_id, lote,
				production_order_id, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10) RETURNING id`, empresa, itemCode, mask, n, dom.EmEstoque, dom.OrigemProducao, warehouseID,
			lote, orderID, por).Scan(&id)
		if err != nil {
			return nil, erroDeDuplicada(err, itemCode, n)
		}
		almox := warehouseID
		if err := evento(ctx, tx, empresa, id, dom.EventoProducao, nil, dom.EmEstoque, nil, &almox, "PRODUCTION_ORDER", &orderID, nil, por); err != nil {
			return nil, err
		}
	}
	return series, nil
}

func gerarPelaMascara(ctx context.Context, tx pgx.Tx, empresa, mascaraID int64, n int, agora time.Time) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT p.id, p.sequence, p.part_type, COALESCE(p.value,''), COALESCE(p.size,0), COALESCE(p.date_format,''),
			p.zero_on_year_change, COALESCE(p.current_value,''), p.last_year
		  FROM lot_mask_parts p JOIN lot_masks m ON m.id = p.lot_mask_id AND m.enterprise_id = $1
		 WHERE p.lot_mask_id = $2 ORDER BY p.sequence FOR UPDATE OF p`, empresa, mascaraID)
	if err != nil {
		return nil, fmt.Errorf("lendo a máscara de série: %w", err)
	}
	var partes []lotmask.LotMaskPart
	for rows.Next() {
		var p lotmask.LotMaskPart
		if err := rows.Scan(&p.ID, &p.Sequence, &p.PartType, &p.Value, &p.Size, &p.DateFormat, &p.ZeroOnYearChange, &p.CurrentValue, &p.LastYear); err != nil {
			rows.Close()
			return nil, err
		}
		p.LotMaskID = mascaraID
		partes = append(partes, p)
	}
	rows.Close()
	if len(partes) == 0 {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a máscara de série %d não tem partes (VLOT0100)", mascaraID))
	}
	out := make([]string, 0, n)
	for k := 0; k < n; k++ {
		res, err := lotmask.Generate(partes, agora)
		if err != nil {
			return nil, err
		}
		for _, up := range res.Updates {
			for i := range partes {
				if partes[i].ID == up.PartID {
					ano := up.NewYear
					partes[i].CurrentValue, partes[i].LastYear = up.NewCurrent, &ano
				}
			}
		}
		out = append(out, res.Code)
	}
	for _, p := range partes {
		if _, err := tx.Exec(ctx, `UPDATE lot_mask_parts SET current_value=$2, last_year=$3 WHERE id=$1`, p.ID, p.CurrentValue, p.LastYear); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ─── Garantia ────────────────────────────────────────────────────────────────

func (r *PG) PorNumero(ctx context.Context, itemCode int64, numero string) (*dom.Serie, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := scan(r.db.QueryRow(ctx, `SELECT `+colunas+de+` WHERE s.enterprise_id=$1 AND s.item_code=$2 AND upper(s.numero_serie)=upper($3)`,
		empresa, itemCode, strings.TrimSpace(numero)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("série %s do item %d", numero, itemCode))
	}
	return s, err
}

// NotaDoCliente: a NF de saída autorizada de número tal para o cliente.
func (r *PG) NotaDoCliente(ctx context.Context, cliente int64, numeroNF int64) (*dom.NotaVendida, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var n dom.NotaVendida
	err = r.db.QueryRow(ctx, `SELECT id, numero_nf, data_emissao FROM fiscal_exits
		 WHERE enterprise_id=$1 AND customer_code=$2 AND numero_nf=$3 AND upper(status) IN ('AUTHORIZED','AUTORIZADA')
		 ORDER BY data_emissao DESC LIMIT 1`, empresa, cliente, numeroNF).Scan(&n.FiscalExitID, &n.NumeroNF, &n.Emissao)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("NF %d autorizada para o cliente %d", numeroNF, cliente))
	}
	return &n, err
}

// EstenderGarantia grava a garantia estendida da série, com o evento.
func (r *PG) EstenderGarantia(ctx context.Context, id int64, ate time.Time, motivo string, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.emTransacao(ctx, func(db DB) error {
		var situacao string
		var cliente *int64
		err := db.QueryRow(ctx, `UPDATE numeros_serie SET garantia_estendida_ate=$3, garantia_estendida_motivo=$4, updated_by=$5, updated_at=NOW()
			 WHERE enterprise_id=$1 AND id=$2 RETURNING situacao, customer_code`, empresa, id, ate, motivo, por).Scan(&situacao, &cliente)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("série %d", id))
		}
		if err != nil {
			return err
		}
		m := fmt.Sprintf("garantia estendida até %s: %s", ate.Format("02/01/2006"), motivo)
		return evento(ctx, db, empresa, id, dom.EventoGarantia, &situacao, situacao, cliente, nil, "", nil, &m, por)
	})
}

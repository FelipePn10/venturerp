package shipping_carrier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Uso conta, por tipo de documento, onde a transportadora está gravada.
// Pedido de compra e aviso de recebimento são da convenção `enterprise_code`;
// os demais, `enterprise_id`.
func (r *Repository) Uso(ctx context.Context, id int64) ([]domrepo.Uso, error) {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
SELECT 'clientes (transportadora padrão)', COUNT(*), COUNT(*) FILTER (WHERE is_active)
  FROM public.customers WHERE shipping_carrier_id = $1 AND enterprise_id = $2
UNION ALL
SELECT 'romaneios', COUNT(*), COUNT(*) FILTER (WHERE status NOT IN ('SHIPPED','CANCELLED'))
  FROM public.shipments WHERE shipping_carrier_id = $1 AND enterprise_id = $2
UNION ALL
SELECT 'cargas', COUNT(*), COUNT(*) FILTER (WHERE status NOT IN ('SHIPPED','CANCELLED'))
  FROM public.shipment_loads WHERE shipping_carrier_id = $1 AND enterprise_id = $2
UNION ALL
SELECT 'pedidos de compra', COUNT(*), COUNT(*) FILTER (WHERE status NOT IN ('RECEIVED','CANCELLED'))
  FROM public.purchase_orders
 WHERE (shipping_carrier_id = $1 OR redispatch_shipping_carrier_id = $1) AND enterprise_code = $3
UNION ALL
SELECT 'avisos de recebimento', COUNT(*), COUNT(*) FILTER (WHERE status NOT IN ('RELEASED','CANCELLED'))
  FROM public.receiving_notices WHERE shipping_carrier_id = $1 AND enterprise_code = $3
UNION ALL
SELECT 'notas fiscais', COUNT(*), 0
  FROM public.fiscal_exits WHERE shipping_carrier_id = $1 AND enterprise_id = $2
UNION ALL
SELECT 'ocorrências de entrega', COUNT(*), 0
  FROM public.shipping_carrier_occurrences WHERE carrier_id = $1 AND enterprise_id = $2`,
		id, empresaID, empresaCode)
	if err != nil {
		return nil, fmt.Errorf("conferir uso da transportadora: %w", err)
	}
	defer rows.Close()
	var out []domrepo.Uso
	for rows.Next() {
		var u domrepo.Uso
		if err := rows.Scan(&u.Documento, &u.Total, &u.Abertos); err != nil {
			return nil, err
		}
		if u.Total > 0 {
			out = append(out, u)
		}
	}
	return out, rows.Err()
}

func (r *Repository) PlacasEmUso(ctx context.Context, id int64) ([]string, error) {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
SELECT DISTINCT upper(regexp_replace(vehicle_plate, '[^A-Za-z0-9]', '', 'g')) FROM (
  SELECT vehicle_plate FROM public.shipment_loads
   WHERE shipping_carrier_id = $1 AND enterprise_id = $2 AND status NOT IN ('SHIPPED','CANCELLED')
  UNION ALL
  SELECT vehicle_plate FROM public.shipments
   WHERE shipping_carrier_id = $1 AND enterprise_id = $2 AND status NOT IN ('SHIPPED','CANCELLED')
) x WHERE COALESCE(vehicle_plate, '') <> ''`, id, empresaID)
	if err != nil {
		return nil, fmt.Errorf("placas em uso: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) Excluir(ctx context.Context, id int64) error {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, filho := range []string{"shipping_carrier_vehicles", "shipping_carrier_service_areas"} {
		if _, err := tx.Exec(ctx, `DELETE FROM public.`+filho+` WHERE carrier_id = $1 AND enterprise_id = $2`, id, empresaID); err != nil {
			return fmt.Errorf("excluir %s: %w", filho, err)
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM public.shipping_carriers WHERE id = $1 AND enterprise_id = $2`, id, empresaID)
	if err != nil {
		if emUso(err) {
			return errorsuc.NewConflictError("a transportadora está gravada em documentos: inative-a em vez de excluir")
		}
		return fmt.Errorf("excluir transportadora: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("transportadora %d não encontrada", id))
	}
	return tx.Commit(ctx)
}

func emUso(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23503"
}

func (r *Repository) HojeDaEmpresa(ctx context.Context) (time.Time, error) {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return time.Time{}, err
	}
	var hoje time.Time
	if err := r.pool.QueryRow(ctx, `
SELECT (now() AT TIME ZONE COALESCE((SELECT NULLIF(timezone,'') FROM public.enterprise_notification_settings WHERE enterprise_id=$1),'America/Sao_Paulo'))::date`,
		empresaID).Scan(&hoje); err != nil {
		return time.Time{}, fmt.Errorf("data de hoje na empresa: %w", err)
	}
	return hoje, nil
}

// ─── Tipos de veículo ────────────────────────────────────────────────────────

func (r *Repository) ListarTiposVeiculo(ctx context.Context) ([]*domrepo.TipoVeiculo, error) {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
SELECT t.id, t.code, t.description, t.wheel_type, t.body_type, t.axles, t.capacity_kg, t.capacity_m3, t.is_active,
       (SELECT COUNT(*) FROM public.shipping_carrier_vehicles v WHERE v.vehicle_type_id = t.id AND v.enterprise_id = t.enterprise_id)
  FROM public.shipping_vehicle_types t
 WHERE t.enterprise_id = $1
 ORDER BY t.code`, empresaID)
	if err != nil {
		return nil, fmt.Errorf("listar tipos de veículo: %w", err)
	}
	defer rows.Close()
	var out []*domrepo.TipoVeiculo
	for rows.Next() {
		var t domrepo.TipoVeiculo
		if err := rows.Scan(&t.ID, &t.Code, &t.Description, &t.WheelType, &t.BodyType, &t.Axles,
			&t.CapacityKg, &t.CapacityM3, &t.IsActive, &t.EmUso); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// SalvarTipoVeiculo grava pelo id; o código não muda depois de criado (é a
// chave que o usuário reconhece na frota).
func (r *Repository) SalvarTipoVeiculo(ctx context.Context, t *domrepo.TipoVeiculo) (*domrepo.TipoVeiculo, error) {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if t.ID > 0 {
		err = r.pool.QueryRow(ctx, `
UPDATE public.shipping_vehicle_types SET description=$3, wheel_type=$4, body_type=$5, axles=$6,
       capacity_kg=$7, capacity_m3=$8, is_active=$9, updated_at=NOW()
 WHERE id=$1 AND enterprise_id=$2 RETURNING id`,
			t.ID, empresaID, t.Description, t.WheelType, t.BodyType, t.Axles, t.CapacityKg, t.CapacityM3, t.IsActive).Scan(&t.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("tipo de veículo %d não encontrado", t.ID))
		}
	} else {
		err = r.pool.QueryRow(ctx, `
INSERT INTO public.shipping_vehicle_types (enterprise_id, code, description, wheel_type, body_type, axles, capacity_kg, capacity_m3, is_active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
			empresaID, t.Code, t.Description, t.WheelType, t.BodyType, t.Axles, t.CapacityKg, t.CapacityM3, t.IsActive).Scan(&t.ID)
	}
	if err != nil {
		if strings.Contains(err.Error(), "shipping_vehicle_types_unico") {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("já existe o tipo de veículo %s", t.Code))
		}
		return nil, fmt.Errorf("gravar tipo de veículo: %w", err)
	}
	return t, nil
}

func (r *Repository) ExcluirTipoVeiculo(ctx context.Context, id int64) error {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM public.shipping_vehicle_types WHERE id=$1 AND enterprise_id=$2`, id, empresaID)
	if err != nil {
		if emUso(err) {
			return errorsuc.NewConflictError("o tipo de veículo está em uso na frota: inative-o em vez de excluir")
		}
		return fmt.Errorf("excluir tipo de veículo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("tipo de veículo %d não encontrado", id))
	}
	return nil
}

// ─── Tipos de ocorrência ─────────────────────────────────────────────────────

// ocorrenciasPadrao é o que a empresa recebe na primeira consulta se ainda não
// tem nenhum tipo: empresa nova não fica sem poder registrar ocorrência.
var ocorrenciasPadrao = []struct {
	code, descricao string
	falha           bool
}{
	{"ATRASO", "Atraso na entrega", true},
	{"AVARIA", "Avaria da mercadoria", true},
	{"EXTRAVIO", "Extravio", true},
	{"ENTREGA_PARCIAL", "Entrega parcial", true},
	{"DEVOLUCAO", "Devolução", true},
	{"COBRANCA_INDEVIDA", "Cobrança indevida de frete", true},
	{"RECUSA_DESTINATARIO", "Recusa do destinatário", false},
	{"REENTREGA", "Reentrega a pedido do cliente", false},
	{"ELOGIO", "Elogio", false},
}

// semearOcorrencias grava os tipos padrão na empresa que ainda não tem nenhum.
// Roda antes de listar e antes do primeiro cadastro: sem isso, a empresa que
// cadastrasse um tipo próprio primeiro nunca receberia os padrões. Tipo padrão
// apagado não volta enquanto a empresa tiver ao menos um tipo.
func (r *Repository) semearOcorrencias(ctx context.Context, empresaID int64) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO public.shipping_occurrence_types (enterprise_id, code, description, counts_as_failure)
SELECT $1, p.code, p.descricao, p.falha
  FROM unnest($2::text[], $3::text[], $4::bool[]) AS p(code, descricao, falha)
 WHERE NOT EXISTS (SELECT 1 FROM public.shipping_occurrence_types WHERE enterprise_id = $1)
ON CONFLICT DO NOTHING`, empresaID, colunaPadrao(func(i int) string { return ocorrenciasPadrao[i].code }),
		colunaPadrao(func(i int) string { return ocorrenciasPadrao[i].descricao }), falhasPadrao())
	if err != nil {
		return fmt.Errorf("semear tipos de ocorrência: %w", err)
	}
	return nil
}

func colunaPadrao(f func(int) string) []string {
	out := make([]string, len(ocorrenciasPadrao))
	for i := range out {
		out[i] = f(i)
	}
	return out
}

func falhasPadrao() []bool {
	out := make([]bool, len(ocorrenciasPadrao))
	for i, p := range ocorrenciasPadrao {
		out[i] = p.falha
	}
	return out
}

func (r *Repository) ListarTiposOcorrencia(ctx context.Context) ([]*domrepo.TipoOcorrencia, error) {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.semearOcorrencias(ctx, empresaID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
SELECT t.id, t.code, t.description, t.counts_as_failure, t.is_active,
       (SELECT COUNT(*) FROM public.shipping_carrier_occurrences o WHERE o.enterprise_id = t.enterprise_id AND o.occurrence_type = t.code)
  FROM public.shipping_occurrence_types t
 WHERE t.enterprise_id = $1
 ORDER BY t.code`, empresaID)
	if err != nil {
		return nil, fmt.Errorf("listar tipos de ocorrência: %w", err)
	}
	defer rows.Close()
	var out []*domrepo.TipoOcorrencia
	for rows.Next() {
		var t domrepo.TipoOcorrencia
		if err := rows.Scan(&t.ID, &t.Code, &t.Description, &t.CountsAsFailure, &t.IsActive, &t.EmUso); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// SalvarTipoOcorrencia: o código não muda depois de criado — a ocorrência
// guarda o código, e trocá-lo deixaria o histórico sem tipo.
func (r *Repository) SalvarTipoOcorrencia(ctx context.Context, t *domrepo.TipoOcorrencia) (*domrepo.TipoOcorrencia, error) {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if t.ID > 0 {
		err = r.pool.QueryRow(ctx, `
UPDATE public.shipping_occurrence_types SET description=$3, counts_as_failure=$4, is_active=$5, updated_at=NOW()
 WHERE id=$1 AND enterprise_id=$2 RETURNING id`,
			t.ID, empresaID, t.Description, t.CountsAsFailure, t.IsActive).Scan(&t.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("tipo de ocorrência %d não encontrado", t.ID))
		}
	} else {
		if err := r.semearOcorrencias(ctx, empresaID); err != nil {
			return nil, err
		}
		err = r.pool.QueryRow(ctx, `
INSERT INTO public.shipping_occurrence_types (enterprise_id, code, description, counts_as_failure, is_active)
VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			empresaID, t.Code, t.Description, t.CountsAsFailure, t.IsActive).Scan(&t.ID)
	}
	if err != nil {
		if strings.Contains(err.Error(), "shipping_occurrence_types_unico") {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("já existe o tipo de ocorrência %s", t.Code))
		}
		return nil, fmt.Errorf("gravar tipo de ocorrência: %w", err)
	}
	return t, nil
}

// ExcluirTipoOcorrencia só apaga tipo nunca usado: a ocorrência guarda o
// código como texto, sem FK, então a conferência é feita aqui.
func (r *Repository) ExcluirTipoOcorrencia(ctx context.Context, id int64) error {
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
DELETE FROM public.shipping_occurrence_types t
 WHERE t.id=$1 AND t.enterprise_id=$2
   AND NOT EXISTS (SELECT 1 FROM public.shipping_carrier_occurrences o
                    WHERE o.enterprise_id = t.enterprise_id AND o.occurrence_type = t.code)`, id, empresaID)
	if err != nil {
		return fmt.Errorf("excluir tipo de ocorrência: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var existe bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM public.shipping_occurrence_types WHERE id=$1 AND enterprise_id=$2)`, id, empresaID).Scan(&existe); err != nil {
			return err
		}
		if existe {
			return errorsuc.NewConflictError("o tipo de ocorrência já foi usado em ocorrências: inative-o em vez de excluir")
		}
		return errorsuc.NewNotFoundError(fmt.Sprintf("tipo de ocorrência %d não encontrado", id))
	}
	return nil
}

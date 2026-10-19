package fiscal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// PendenciasPG guarda os efeitos de NF-e ainda não aplicados no ERP.
type PendenciasPG struct {
	db conexao
}

var _ repository.PendenciasNFeRepository = (*PendenciasPG)(nil)

func NewPendenciasPG(pool *pgxpool.Pool) *PendenciasPG { return &PendenciasPG{db: pool} }

// NewPendenciasTx: ligado à transação dos efeitos (marcar resolvida junto).
func NewPendenciasTx(tx pgx.Tx) *PendenciasPG { return &PendenciasPG{db: tx} }

func (r *PendenciasPG) Registrar(ctx context.Context, exitID int64, etapa string, por uuid.UUID) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var criador *uuid.UUID
	if por != uuid.Nil {
		criador = &por
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO fiscal_exit_pendencias (enterprise_id, fiscal_exit_id, etapa, criado_por)
		SELECT $1, f.id, $3, $4 FROM fiscal_exits f WHERE f.id = $2 AND f.enterprise_id = $1
		ON CONFLICT (fiscal_exit_id, etapa) WHERE situacao = 'PENDENTE' DO NOTHING`, e, exitID, etapa, criador)
	if err != nil {
		return fmt.Errorf("registrando a pendência da NF-e %d: %w", exitID, err)
	}
	return nil
}

// Reservar trava a pendência aberta (FOR UPDATE). Duas execuções simultâneas
// — a autorização e o reprocesso automático, ou duas consultas de status —
// esperam uma pela outra; a segunda encontra a pendência já resolvida e não
// aplica os efeitos de novo (antes baixava o estoque duas vezes).
func (r *PendenciasPG) Reservar(ctx context.Context, exitID int64, etapa string) (bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var id int64
	err = r.db.QueryRow(ctx, `
		SELECT id FROM fiscal_exit_pendencias
		 WHERE enterprise_id = $1 AND fiscal_exit_id = $2 AND etapa = $3 AND situacao = 'PENDENTE'
		 LIMIT 1 FOR UPDATE`, e, exitID, etapa).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reservando a pendência da NF-e %d: %w", exitID, err)
	}
	return true, nil
}

func (r *PendenciasPG) Resolver(ctx context.Context, exitID int64, etapa string) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE fiscal_exit_pendencias SET situacao = 'RESOLVIDA', resolvido_em = NOW(), ultimo_erro = NULL
		 WHERE enterprise_id = $1 AND fiscal_exit_id = $2 AND etapa = $3 AND situacao = 'PENDENTE'`, e, exitID, etapa)
	if err != nil {
		return fmt.Errorf("resolvendo a pendência da NF-e %d: %w", exitID, err)
	}
	return nil
}

// RegistrarFalha conta a tentativa, guarda o motivo e agenda a próxima com
// espera crescente (1, 2, 4… até 60 minutos).
func (r *PendenciasPG) RegistrarFalha(ctx context.Context, exitID int64, etapa, motivo string) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE fiscal_exit_pendencias
		   SET tentativas = tentativas + 1, ultimo_erro = $4,
		       proxima_tentativa = NOW() + LEAST(INTERVAL '60 minutes', INTERVAL '1 minute' * POWER(2, LEAST(tentativas, 6)))
		 WHERE enterprise_id = $1 AND fiscal_exit_id = $2 AND etapa = $3 AND situacao = 'PENDENTE'`, e, exitID, etapa, motivo)
	if err != nil {
		return fmt.Errorf("registrando a falha da pendência da NF-e %d: %w", exitID, err)
	}
	return nil
}

func (r *PendenciasPG) DaNota(ctx context.Context, exitID int64) ([]entity.PendenciaNFe, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT fiscal_exit_id, etapa, situacao, tentativas, COALESCE(ultimo_erro, ''), proxima_tentativa, criado_em, resolvido_em
		  FROM fiscal_exit_pendencias WHERE enterprise_id = $1 AND fiscal_exit_id = $2 ORDER BY criado_em DESC`, e, exitID)
	if err != nil {
		return nil, fmt.Errorf("listando as pendências da NF-e %d: %w", exitID, err)
	}
	defer rows.Close()
	out := []entity.PendenciaNFe{}
	for rows.Next() {
		var p entity.PendenciaNFe
		if err := rows.Scan(&p.FiscalExitID, &p.Etapa, &p.Situacao, &p.Tentativas, &p.UltimoErro, &p.ProximaTentativa, &p.CriadoEm, &p.ResolvidoEm); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Vencidas lista, de todas as empresas, as pendências na hora de tentar de
// novo — é a fila do processo em segundo plano, que roda cada uma na sessão da
// empresa dona.
func (r *PendenciasPG) Vencidas(ctx context.Context, limite int) ([]entity.PendenciaVencida, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.fiscal_exit_id, p.etapa, p.enterprise_id, e.code, p.criado_por
		  FROM fiscal_exit_pendencias p JOIN enterprise e ON e.id = p.enterprise_id
		 WHERE p.situacao = 'PENDENTE' AND p.proxima_tentativa <= $1
		 ORDER BY p.proxima_tentativa LIMIT $2`, time.Now(), limite)
	if err != nil {
		return nil, fmt.Errorf("lendo a fila de pendências de NF-e: %w", err)
	}
	defer rows.Close()
	out := []entity.PendenciaVencida{}
	for rows.Next() {
		var p entity.PendenciaVencida
		if err := rows.Scan(&p.FiscalExitID, &p.Etapa, &p.EnterpriseID, &p.EnterpriseCode, &p.CriadoPor); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

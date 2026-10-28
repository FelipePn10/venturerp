package enterprise

import (
	"context"
	"errors"
	"fmt"
	"math"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *repositoryEnterpriseSQLC) Create(
	ctx context.Context,
	enterprise *entity.Enterprise,
) (*entity.Enterprise, error) {
	// Com o pool, empresa (código livre), vínculo do criador e identidade
	// fiscal vão numa transação; sem ele, como antes (só a empresa).
	if r.pool != nil {
		return r.criarCompleta(ctx, enterprise)
	}
	return criar(ctx, r.q, enterprise)
}

func criar(ctx context.Context, q *sqlc.Queries, enterprise *entity.Enterprise) (*entity.Enterprise, error) {
	params := sqlc.CreateEnterpriseParams{
		Code:      int32(enterprise.Code),
		Name:      enterprise.Name,
		CreatedBy: pgutil.ToPgUUID(enterprise.CreatedBy),
	}
	// Note: ID is omitted — BIGSERIAL auto-generates it.

	dbEnterprise, err := q.CreateEnterprise(ctx, params)
	if err != nil {
		// code has a UNIQUE constraint: surface a typed conflict so the handler
		// returns 409 instead of a generic 500 on duplicate codes.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("já existe uma empresa com o código %d", enterprise.Code))
		}
		return nil, fmt.Errorf("create enterprise: %w", err)
	}

	return toEnterpriseEntity(sqlc.Enterprise(dbEnterprise)), nil
}

func (r *repositoryEnterpriseSQLC) criarCompleta(ctx context.Context, e *entity.Enterprise) (*entity.Enterprise, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if e.Code == 0 {
		// Próximo código livre, sem corrida entre dois cadastros simultâneos.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('enterprise_code'))`); err != nil {
			return nil, err
		}
		var proximo int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(code),0)+1 FROM enterprise`).Scan(&proximo); err != nil {
			return nil, err
		}
		if proximo > math.MaxInt32 {
			return nil, errorsuc.NewValidationError("não há código livre automático: informe o código da empresa")
		}
		e.Code = int(proximo)
	}
	criada, err := criar(ctx, sqlc.New(tx), e)
	if err != nil {
		return nil, err
	}
	if f := e.Fiscal; f != nil {
		padrao := func(v, p string) string {
			if v == "" {
				return p
			}
			return v
		}
		if _, err := tx.Exec(ctx, `INSERT INTO fiscal_configs (enterprise_id, cnpj_empresa, razao_social, trade_name, ie_empresa, regime_tributario, uf_empresa,
				municipio, codigo_municipio, cep, logradouro, numero, bairro, telefone, updated_by, complemento, email)
			VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12,$13,NULLIF($14,''),$15,NULLIF($16,''),NULLIF($17,''))`,
			criada.ID, f.CNPJ, f.RazaoSocial, f.NomeFantasia, f.IE, f.RegimeTributario, f.UF,
			padrao(f.Municipio, "A preencher"), padrao(f.CodigoMunicipio, "0000000"), padrao(f.CEP, "00000000"),
			padrao(f.Logradouro, "A preencher"), padrao(f.Numero, "S/N"), padrao(f.Bairro, "A preencher"), f.Telefone, e.CreatedBy,
			f.Complemento, f.Email); err != nil {
			return nil, fmt.Errorf("gravando a identidade fiscal da empresa: %w", err)
		}
		criada.Fiscal = f
	}
	return criada, tx.Commit(ctx)
}

func (r *repositoryEnterpriseSQLC) GetByCodeForUser(ctx context.Context, code int, userID uuid.UUID) (*entity.Enterprise, error) {
	row, err := r.q.GetEnterpriseByCodeForUser(ctx, sqlc.GetEnterpriseByCodeForUserParams{Code: int32(code), UserID: pgutil.ToPgUUID(userID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("empresa %d não encontrada", code))
	}
	if err != nil {
		return nil, fmt.Errorf("get enterprise by code %d: %w", code, err)
	}
	return toEnterpriseEntity(row), nil
}

func (r *repositoryEnterpriseSQLC) ListForUser(ctx context.Context, userID uuid.UUID) ([]*entity.Enterprise, error) {
	rows, err := r.q.ListEnterprisesForUser(ctx, pgutil.ToPgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list enterprises: %w", err)
	}
	out := make([]*entity.Enterprise, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEnterpriseEntity(row))
	}
	return out, nil
}

func toEnterpriseEntity(row sqlc.Enterprise) *entity.Enterprise {
	return &entity.Enterprise{
		ID:        int(row.ID),
		Code:      int(row.Code),
		Name:      row.Name,
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
	}
}

// Identificacao: a da empresa da sessão (nil sem configuração fiscal).
func (r *repositoryEnterpriseSQLC) Identificacao(ctx context.Context) (*entity.DadosFiscais, error) {
	if r.pool == nil {
		return nil, errors.New("repositório de empresa sem pool")
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var d entity.DadosFiscais
	err = r.pool.QueryRow(ctx, `SELECT cnpj_empresa, razao_social, COALESCE(trade_name,''), COALESCE(ie_empresa,''), regime_tributario, uf_empresa,
			COALESCE(municipio,''), COALESCE(codigo_municipio,''), COALESCE(cep,''), COALESCE(logradouro,''), COALESCE(numero,''),
			COALESCE(complemento,''), COALESCE(bairro,''), COALESCE(telefone,''), COALESCE(email,'')
		  FROM fiscal_configs WHERE enterprise_id = $1`, empresa).Scan(&d.CNPJ, &d.RazaoSocial, &d.NomeFantasia, &d.IE, &d.RegimeTributario, &d.UF,
		&d.Municipio, &d.CodigoMunicipio, &d.CEP, &d.Logradouro, &d.Numero, &d.Complemento, &d.Bairro, &d.Telefone, &d.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a identificação da empresa: %w", err)
	}
	return &d, nil
}

// AtualizarIdentificacao grava a identificação e o nome da empresa numa
// transação; os parâmetros fiscais (alíquotas, Focus, vencimentos) ficam como
// estão. Sem configuração fiscal ainda, cria com os padrões da tabela.
func (r *repositoryEnterpriseSQLC) AtualizarIdentificacao(ctx context.Context, d *entity.DadosFiscais, por uuid.UUID) error {
	if r.pool == nil {
		return errors.New("repositório de empresa sem pool")
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	nome := d.NomeFantasia
	if nome == "" {
		nome = d.RazaoSocial
	}
	if _, err := tx.Exec(ctx, `UPDATE enterprise SET name = $2 WHERE id = $1`, empresa, nome); err != nil {
		return fmt.Errorf("gravando o nome da empresa: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO fiscal_configs (enterprise_id, cnpj_empresa, razao_social, trade_name, ie_empresa, regime_tributario, uf_empresa,
			municipio, codigo_municipio, cep, logradouro, numero, complemento, bairro, telefone, email, updated_by, updated_at)
		VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12,NULLIF($13,''),$14,NULLIF($15,''),NULLIF($16,''),$17,NOW())
		ON CONFLICT (enterprise_id) DO UPDATE SET
			cnpj_empresa = EXCLUDED.cnpj_empresa, razao_social = EXCLUDED.razao_social, trade_name = EXCLUDED.trade_name,
			ie_empresa = EXCLUDED.ie_empresa, regime_tributario = EXCLUDED.regime_tributario, uf_empresa = EXCLUDED.uf_empresa,
			municipio = EXCLUDED.municipio, codigo_municipio = EXCLUDED.codigo_municipio, cep = EXCLUDED.cep,
			logradouro = EXCLUDED.logradouro, numero = EXCLUDED.numero, complemento = EXCLUDED.complemento,
			bairro = EXCLUDED.bairro, telefone = EXCLUDED.telefone, email = EXCLUDED.email,
			updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		empresa, d.CNPJ, d.RazaoSocial, d.NomeFantasia, d.IE, d.RegimeTributario, d.UF,
		d.Municipio, d.CodigoMunicipio, d.CEP, d.Logradouro, d.Numero, d.Complemento, d.Bairro, d.Telefone, d.Email, por); err != nil {
		return fmt.Errorf("gravando a identificação da empresa: %w", err)
	}
	return tx.Commit(ctx)
}

package nfse

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/nfse/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NFSeRepositoryPG struct {
	pool *pgxpool.Pool
}

var _ repository.NFSeRepository = (*NFSeRepositoryPG)(nil)

func NewNFSeRepositoryPG(pool *pgxpool.Pool) *NFSeRepositoryPG {
	return &NFSeRepositoryPG{pool: pool}
}

const nfseColumns = `id, numero_rps, serie_rps, tipo_rps, data_emissao, status, natureza_operacao,
	optante_simples, incentivador_cultural,
	tomador_cnpj_cpf, tomador_razao_social, tomador_email, tomador_logradouro, tomador_numero,
	tomador_complemento, tomador_bairro, tomador_codigo_municipio, tomador_uf, tomador_cep,
	item_lista_servico, codigo_tributario_municipio, discriminacao, codigo_municipio,
	valor_servicos, valor_deducoes, aliquota_iss, iss_retido, valor_iss, valor_liquido,
	focus_ref, numero_nfse, codigo_verificacao, url, xml_path,
	sales_order_code, notes, is_active, created_by, created_at, updated_at`

func scanNFSe(s interface{ Scan(...interface{}) error }) (*entity.NFSe, error) {
	var n entity.NFSe
	err := s.Scan(
		&n.ID, &n.NumeroRPS, &n.SerieRPS, &n.TipoRPS, &n.DataEmissao, &n.Status, &n.NaturezaOperacao,
		&n.OptanteSimples, &n.IncentivadorCultural,
		&n.TomadorCnpjCpf, &n.TomadorRazaoSocial, &n.TomadorEmail, &n.TomadorLogradouro, &n.TomadorNumero,
		&n.TomadorComplemento, &n.TomadorBairro, &n.TomadorCodigoMunicipio, &n.TomadorUF, &n.TomadorCEP,
		&n.ItemListaServico, &n.CodigoTributarioMunicipio, &n.Discriminacao, &n.CodigoMunicipio,
		&n.ValorServicos, &n.ValorDeducoes, &n.AliquotaISS, &n.IssRetido, &n.ValorISS, &n.ValorLiquido,
		&n.FocusRef, &n.NumeroNFSe, &n.CodigoVerificacao, &n.URL, &n.XMLPath,
		&n.SalesOrderCode, &n.Notes, &n.IsActive, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NFSeRepositoryPG) Create(ctx context.Context, n *entity.NFSe) (*entity.NFSe, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.nfse
		    (numero_rps, serie_rps, tipo_rps, data_emissao, status, natureza_operacao,
		     optante_simples, incentivador_cultural,
		     tomador_cnpj_cpf, tomador_razao_social, tomador_email, tomador_logradouro, tomador_numero,
		     tomador_complemento, tomador_bairro, tomador_codigo_municipio, tomador_uf, tomador_cep,
		     item_lista_servico, codigo_tributario_municipio, discriminacao, codigo_municipio,
		     valor_servicos, valor_deducoes, aliquota_iss, iss_retido, valor_iss, valor_liquido,
		     sales_order_code, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31)
		 RETURNING id, is_active, created_at, updated_at`,
		n.NumeroRPS, n.SerieRPS, n.TipoRPS, n.DataEmissao, n.Status, n.NaturezaOperacao,
		n.OptanteSimples, n.IncentivadorCultural,
		n.TomadorCnpjCpf, n.TomadorRazaoSocial, n.TomadorEmail, n.TomadorLogradouro, n.TomadorNumero,
		n.TomadorComplemento, n.TomadorBairro, n.TomadorCodigoMunicipio, n.TomadorUF, n.TomadorCEP,
		n.ItemListaServico, n.CodigoTributarioMunicipio, n.Discriminacao, n.CodigoMunicipio,
		n.ValorServicos, n.ValorDeducoes, n.AliquotaISS, n.IssRetido, n.ValorISS, n.ValorLiquido,
		n.SalesOrderCode, n.Notes, n.CreatedBy,
	).Scan(&n.ID, &n.IsActive, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating NFS-e: %w", err)
	}
	return n, nil
}

func (r *NFSeRepositoryPG) GetByID(ctx context.Context, id int64) (*entity.NFSe, error) {
	n, err := scanNFSe(r.pool.QueryRow(ctx, `SELECT `+nfseColumns+` FROM public.nfse WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("NFS-e %d não encontrada", id)
		}
		return nil, fmt.Errorf("getting NFS-e: %w", err)
	}
	return n, nil
}

func (r *NFSeRepositoryPG) List(ctx context.Context) ([]*entity.NFSe, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+nfseColumns+` FROM public.nfse WHERE is_active = true ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing NFS-e: %w", err)
	}
	defer rows.Close()
	var result []*entity.NFSe
	for rows.Next() {
		n, err := scanNFSe(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning NFS-e: %w", err)
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

func (r *NFSeRepositoryPG) UpdateStatus(ctx context.Context, id int64, status entity.NFSeStatus) (*entity.NFSe, error) {
	_, err := r.pool.Exec(ctx, `UPDATE public.nfse SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return nil, fmt.Errorf("updating NFS-e status: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *NFSeRepositoryPG) UpdateAuthorization(ctx context.Context, id int64, numeroNFSe, codigoVerificacao, url, focusRef string) (*entity.NFSe, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.nfse SET numero_nfse = $1, codigo_verificacao = $2, url = $3, focus_ref = $4,
		     status = 'AUTORIZADA', updated_at = NOW() WHERE id = $5`,
		numeroNFSe, codigoVerificacao, url, focusRef, id)
	if err != nil {
		return nil, fmt.Errorf("updating NFS-e authorization: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *NFSeRepositoryPG) SaveFocusLog(ctx context.Context, endpoint, method, reqBody, respBody string, statusCode, durationMs int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.focus_nfe_logs (fiscal_exit_id, endpoint, method, request_body, response_body, status_code, duration_ms)
		 VALUES (NULL, $1, $2, $3, $4, $5, $6)`,
		endpoint, method, reqBody, respBody, statusCode, durationMs)
	return err
}

package fiscal

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var _ repository.ReceivedDocumentsRepository = (*FiscalRepositoryPG)(nil)

func NewReceivedDocumentsRepositoryPG(r repository.FiscalRepository) repository.ReceivedDocumentsRepository {
	return r.(*FiscalRepositoryPG)
}

func (r *FiscalRepositoryPG) DFeVersion(ctx context.Context) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var v int64
	err = r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT dfe_ultima_versao FROM fiscal_configs WHERE enterprise_id=$1 LIMIT 1),0)`, empresa).Scan(&v)
	return v, err
}

func (r *FiscalRepositoryPG) UpsertReceivedDocuments(ctx context.Context, docs []repository.ReceivedDocument, maxVersao int64) (int, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	novos := 0
	for _, d := range docs {
		var inserido bool
		err := tx.QueryRow(ctx,
			`INSERT INTO fiscal_received_documents
				(enterprise_id, chave_acesso, cnpj_emitente, nome_emitente, numero_nf, serie, data_emissao, valor_total,
				 situacao, manifestacao, xml_completo, versao, fiscal_entry_id, synced_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,
			         (SELECT id FROM fiscal_entries WHERE enterprise_id=$1 AND chave_acesso=$2 AND is_active AND status <> 'CANCELLED' LIMIT 1), NOW())
			 ON CONFLICT (enterprise_id, chave_acesso) DO UPDATE SET
				cnpj_emitente=EXCLUDED.cnpj_emitente, nome_emitente=EXCLUDED.nome_emitente, numero_nf=EXCLUDED.numero_nf,
				serie=EXCLUDED.serie, data_emissao=EXCLUDED.data_emissao, valor_total=EXCLUDED.valor_total,
				situacao=EXCLUDED.situacao, manifestacao=COALESCE(EXCLUDED.manifestacao, fiscal_received_documents.manifestacao),
				xml_completo=EXCLUDED.xml_completo, versao=EXCLUDED.versao,
				fiscal_entry_id=COALESCE(fiscal_received_documents.fiscal_entry_id, EXCLUDED.fiscal_entry_id), synced_at=NOW()
			 RETURNING (xmax = 0)`,
			empresa, d.ChaveAcesso, nuloSeVazio(d.CNPJEmitente), truncarStr(d.NomeEmitente, 200), d.NumeroNF, d.Serie, d.DataEmissao, d.ValorTotal,
			nuloSeVazio(d.Situacao), d.Manifestacao, d.XMLCompleto, d.Versao).Scan(&inserido)
		if err != nil {
			return 0, fmt.Errorf("gravando NF-e recebida %s: %w", d.ChaveAcesso, err)
		}
		if inserido {
			novos++
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE fiscal_configs SET dfe_ultima_versao=GREATEST(dfe_ultima_versao,$2), dfe_sincronizado_em=NOW() WHERE enterprise_id=$1`,
		empresa, maxVersao); err != nil {
		return 0, err
	}
	return novos, tx.Commit(ctx)
}

func (r *FiscalRepositoryPG) ListReceivedDocuments(ctx context.Context, f repository.ReceivedDocumentsFilter) ([]repository.ReceivedDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if f.Limite <= 0 || f.Limite > 1000 {
		f.Limite = 300
	}
	busca := strings.TrimSpace(f.Busca)
	rows, err := r.pool.Query(ctx,
		`SELECT id, chave_acesso, COALESCE(cnpj_emitente,''), COALESCE(nome_emitente,''), numero_nf, serie, data_emissao, valor_total,
		        COALESCE(situacao,''), manifestacao, xml_completo, COALESCE(versao,0), fiscal_entry_id, synced_at
		   FROM fiscal_received_documents
		  WHERE enterprise_id=$1
		    AND (NOT $2 OR fiscal_entry_id IS NULL)
		    AND ($3 = '' OR nome_emitente ILIKE '%'||$3||'%' OR cnpj_emitente LIKE '%'||$3||'%' OR chave_acesso LIKE '%'||$3||'%' OR numero_nf::text = $3)
		    AND (NOT $5 OR (`+semManifestacaoConclusiva+` AND data_emissao + $7::int <= $6::date + $8::int))
		  ORDER BY data_emissao DESC NULLS LAST, id DESC LIMIT $4`, empresa, f.SomentePendentes, busca, f.Limite,
		f.SomentePrazo, dataOuHoje(f.Hoje), repository.PrazoManifestacaoDias, repository.AlertaPrazoDias)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []repository.ReceivedDocument
	for rows.Next() {
		var d repository.ReceivedDocument
		if err := rows.Scan(&d.ID, &d.ChaveAcesso, &d.CNPJEmitente, &d.NomeEmitente, &d.NumeroNF, &d.Serie, &d.DataEmissao, &d.ValorTotal,
			&d.Situacao, &d.Manifestacao, &d.XMLCompleto, &d.Versao, &d.FiscalEntryID, &d.SyncedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) SetManifestacao(ctx context.Context, chave, tipo string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE fiscal_received_documents SET manifestacao=$3 WHERE enterprise_id=$1 AND chave_acesso=$2`, empresa, chave, tipo)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("documento recebido não encontrado: sincronize as notas recebidas")
	}
	return nil
}

// semManifestacaoConclusiva: ciência não encerra o prazo; nota cancelada pelo
// emitente não precisa de manifestação.
const semManifestacaoConclusiva = `(manifestacao IS NULL OR manifestacao = 'ciencia') AND COALESCE(situacao,'') <> 'cancelada' AND data_emissao IS NOT NULL`

// dataOuHoje: a data vem do processo (parâmetro), nunca do CURRENT_DATE do
// banco — o banco roda em America/Sao_Paulo e o processo pode estar em UTC.
func dataOuHoje(t time.Time) time.Time {
	if t.IsZero() {
		t = time.Now()
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (r *FiscalRepositoryPG) ReservarSincronizacaoDFe(ctx context.Context, intervalo time.Duration) ([]repository.EmpresaDFe, error) {
	rows, err := r.pool.Query(ctx,
		`UPDATE fiscal_configs fc SET dfe_ultima_tentativa = NOW()
		   FROM enterprise e
		  WHERE e.id = fc.enterprise_id
		    AND fc.id IN (
		        SELECT id FROM fiscal_configs
		         WHERE dfe_sync_automatico
		           AND COALESCE(focus_nfe_token,'') <> '' AND COALESCE(cnpj_empresa,'') <> ''
		           AND (dfe_ultima_tentativa IS NULL OR dfe_ultima_tentativa < NOW() - make_interval(secs => $1))
		         FOR UPDATE SKIP LOCKED)
		 RETURNING fc.enterprise_id, e.code, fc.updated_by`, intervalo.Seconds())
	if err != nil {
		return nil, fmt.Errorf("reservando empresas para a sincronização DF-e: %w", err)
	}
	defer rows.Close()
	var out []repository.EmpresaDFe
	for rows.Next() {
		var d repository.EmpresaDFe
		var ator *uuid.UUID
		if err := rows.Scan(&d.EnterpriseID, &d.EnterpriseCode, &ator); err != nil {
			return nil, err
		}
		if ator != nil {
			d.Ator = *ator
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) RegistrarResultadoDFe(ctx context.Context, erro *string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE fiscal_configs SET dfe_ultima_tentativa=NOW(), dfe_ultimo_erro=$2 WHERE enterprise_id=$1`, empresa, erro)
	return err
}

func (r *FiscalRepositoryPG) DFeStatus(ctx context.Context, hoje time.Time) (*repository.DFeStatus, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	st := &repository.DFeStatus{}
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT dfe_sync_automatico FROM fiscal_configs WHERE enterprise_id=$1 LIMIT 1), FALSE),
		        (SELECT dfe_sincronizado_em FROM fiscal_configs WHERE enterprise_id=$1 LIMIT 1),
		        (SELECT dfe_ultima_tentativa FROM fiscal_configs WHERE enterprise_id=$1 LIMIT 1),
		        (SELECT dfe_ultimo_erro FROM fiscal_configs WHERE enterprise_id=$1 LIMIT 1),
		        (SELECT COUNT(*) FROM fiscal_received_documents WHERE enterprise_id=$1 AND `+semManifestacaoConclusiva+`
		            AND data_emissao + $3::int >= $2::date AND data_emissao + $3::int <= $2::date + $4::int),
		        (SELECT COUNT(*) FROM fiscal_received_documents WHERE enterprise_id=$1 AND `+semManifestacaoConclusiva+`
		            AND data_emissao + $3::int < $2::date)`,
		empresa, dataOuHoje(hoje), repository.PrazoManifestacaoDias, repository.AlertaPrazoDias,
	).Scan(&st.Automatico, &st.SincronizadoEm, &st.UltimaTentativa, &st.UltimoErro, &st.PrazoProximo, &st.PrazoVencido)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (r *FiscalRepositoryPG) SetDFeAutomatico(ctx context.Context, ativo bool) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE fiscal_configs SET dfe_sync_automatico=$2 WHERE enterprise_id=$1`, empresa, ativo)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("a empresa não tem configuração fiscal (VFIS0100)")
	}
	return nil
}

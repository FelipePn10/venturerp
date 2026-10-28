// Package recuperacao_uc: chamado → CAPA da qualidade (reclamação do cliente)
// e recuperação do custo da garantia junto ao fornecedor (pós-venda, Fase 3.5).
package recuperacao_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/capa_uc"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/recuperacao"
)

// AbridorDeCAPA: a CAPA da qualidade (capa_uc).
type AbridorDeCAPA interface {
	Abrir(ctx context.Context, dto capa_uc.AberturaDTO) (*capa_uc.CapaResponse, error)
}

type UseCase struct {
	Repo dom.Repository
	Auth ports.AuthService
	CAPA AbridorDeCAPA
}

func (uc *UseCase) ator(ctx context.Context) *uuid.UUID {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return nil
	}
	return &a
}

func limpo(s *string) *string {
	if s == nil {
		return nil
	}
	if v := strings.TrimSpace(*s); v != "" {
		return &v
	}
	return nil
}

// ─── Chamado → CAPA ──────────────────────────────────────────────────────────

type CAPADTO struct {
	CallItemCode *int64 `json:"call_item_code,omitempty"`
	Metodo       string `json:"metodo"`
	Responsavel  string `json:"responsavel,omitempty"`
	Prazo        string `json:"prazo,omitempty"`
}

// AbrirCAPA abre a CAPA de reclamação do cliente com o cliente, o item e a
// referência do chamado, e a liga ao chamado (um chamado, uma CAPA).
func (uc *UseCase) AbrirCAPA(ctx context.Context, callCode int64, dto CAPADTO) (*capa_uc.CapaResponse, error) {
	if uc.CAPA == nil {
		return nil, errorsuc.NewValidationError("a CAPA da qualidade não está disponível")
	}
	ch, err := uc.Repo.ChamadoParaCAPA(ctx, callCode)
	if err != nil {
		return nil, err
	}
	if ch.CapaID != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o chamado %d já tem a CAPA %d ligada", ch.CallNumber, *ch.CapaID))
	}
	var item *int64
	if dto.CallItemCode != nil {
		it, ok := ch.Itens[*dto.CallItemCode]
		if !ok {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d do chamado %d", *dto.CallItemCode, ch.CallNumber))
		}
		item = &it
	} else if len(ch.Itens) == 1 {
		for _, it := range ch.Itens {
			v := it
			item = &v
		}
	}
	ref := fmt.Sprintf("Chamado de assistência %d", ch.CallNumber)
	cliente := ch.CustomerCode
	metodo := strings.ToUpper(strings.TrimSpace(dto.Metodo))
	if metodo == "" {
		metodo = "8D"
	}
	capa, err := uc.CAPA.Abrir(ctx, capa_uc.AberturaDTO{Metodo: metodo, Tipo: "CORRETIVA", Origem: "RECLAMACAO_CLIENTE",
		Titulo: "Reclamação do cliente — " + ch.Assunto, ItemCode: item, CustomerCode: &cliente, Referencia: &ref,
		Responsavel: dto.Responsavel, Prazo: dto.Prazo})
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.VincularCAPA(ctx, callCode, capa.ID); err != nil {
		return nil, fmt.Errorf("a CAPA %d foi aberta, mas não ficou ligada ao chamado %d: %w", capa.Numero, ch.CallNumber, err)
	}
	return capa, nil
}

// ─── Recuperação junto ao fornecedor ─────────────────────────────────────────

type AberturaDTO struct {
	SupplierCode   int64           `json:"supplier_code"`
	OSID           *int64          `json:"os_id,omitempty"`
	CallCode       *int64          `json:"call_code,omitempty"`
	Motivo         string          `json:"motivo"`
	ValorReclamado decimal.Decimal `json:"valor_reclamado"`
}

func (uc *UseCase) Abrir(ctx context.Context, dto AberturaDTO) (*dom.Reclamacao, error) {
	r := &dom.Reclamacao{SupplierCode: dto.SupplierCode, OSID: dto.OSID, CallCode: dto.CallCode, Motivo: strings.TrimSpace(dto.Motivo),
		ValorReclamado: dto.ValorReclamado}
	if err := r.ConferirAbertura(); err != nil {
		return nil, err
	}
	if err := uc.Repo.Abrir(ctx, r, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, r.ID)
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*dom.Reclamacao, error) {
	return uc.Repo.Obter(ctx, id)
}

func (uc *UseCase) Listar(ctx context.Context, f dom.Filtro) ([]dom.Reclamacao, error) {
	f.Situacao = strings.ToUpper(strings.TrimSpace(f.Situacao))
	return uc.Repo.Listar(ctx, f)
}

type MudancaDTO struct {
	Situacao        string          `json:"situacao"`
	Resposta        *string         `json:"resposta,omitempty"`
	ValorRecuperado decimal.Decimal `json:"valor_recuperado"`
	Forma           *string         `json:"forma,omitempty"`
	Documento       *string         `json:"documento,omitempty"`
	RecuperadoEm    string          `json:"recuperado_em,omitempty"`
}

func (uc *UseCase) Mudar(ctx context.Context, id int64, dto MudancaDTO) (*dom.Reclamacao, error) {
	m := dom.Mudanca{Para: strings.ToUpper(strings.TrimSpace(dto.Situacao)), Resposta: limpo(dto.Resposta), ValorRecuperado: dto.ValorRecuperado,
		Forma: limpo(dto.Forma), Documento: limpo(dto.Documento), Por: uc.ator(ctx)}
	if m.Forma != nil {
		f := strings.ToUpper(*m.Forma)
		m.Forma = &f
	}
	if s := strings.TrimSpace(dto.RecuperadoEm); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			return nil, errorsuc.NewValidationError("data do ressarcimento inválida (use AAAA-MM-DD)")
		}
		m.RecuperadoEm = &t
	}
	if err := uc.Repo.Mudar(ctx, id, m); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, id)
}

package stock_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// AprovacoesDeInventario é o motor de aprovação visto pelo fechamento.
type AprovacoesDeInventario interface {
	TemPoliticaAtiva(ctx context.Context, t aprovacao.Tipo) (bool, error)
	Solicitar(ctx context.Context, t aprovacao.Tipo, documento int64, valor decimal.Decimal, descricao string) (*aprovacao.Solicitacao, error)
}

type CloseInventoryUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
	// Aprovacoes é ligado na montagem; sem ele o ajuste com alçada é recusado.
	Aprovacoes AprovacoesDeInventario
}

// Execute fecha o inventário lançando o ajuste de cada diferença, ou manda o
// ajuste para a alçada quando o inventário exige.
//
// Antes o fechamento só trocava a situação para CLOSED: a contagem nunca
// chegava ao saldo, e a conferência "todas as divergências ajustadas" nunca
// disparava porque a diferença não era calculada. As regras (item sem contagem,
// ajuste ao custo médio, transação única) estão no repositório, que trava o
// inventário enquanto lança.
func (uc *CloseInventoryUseCase) Execute(ctx context.Context, id int64, dto request.CloseInventoryDTO) (*response.InventoryCloseResponse, error) {
	if !uc.Auth.CanCloseInventory(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := uc.Repo.GetInventory(ctx, id)
	if err != nil {
		return nil, err
	}
	switch inv.Status {
	case entity.InventarioAberto:
		if inv.BlindCount {
			return nil, errorsuc.NewValidationError("contagem cega: encerre a contagem e confira as diferenças antes de fechar")
		}
	case entity.InventarioEmConferencia:
	case entity.InventarioAguardandoAprovacao:
		return nil, errorsuc.NewValidationError("o ajuste deste inventário está aguardando aprovação")
	default:
		return nil, errorsuc.NewValidationError("somente inventário em andamento pode ser fechado")
	}
	if inv.TotalItems == 0 {
		return nil, errorsuc.NewValidationError("inventário sem itens não pode ser fechado: conte ao menos um item")
	}
	motivo := strings.TrimSpace(dto.Reason)
	resumo, err := uc.Repo.ResumoDoFechamento(ctx, inv, dto.ZerarNaoContados)
	if err != nil {
		return nil, err
	}
	if resumo.NaoContados > 0 {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%d item(ns) ainda sem contagem: conte-os ou feche zerando os não contados", resumo.NaoContados))
	}
	limite, err := uc.Repo.LimiteDeAjusteDoUsuario(ctx, ator)
	if err != nil {
		return nil, err
	}
	if inv.ExigeAprovacao(resumo, limite) {
		return uc.enviarParaAprovacao(ctx, inv, resumo, dto.ZerarNaoContados, motivo)
	}
	res, err := uc.Repo.FecharInventario(ctx, id, ator, dto.ZerarNaoContados, motivo)
	if err != nil {
		return nil, err
	}
	return &response.InventoryCloseResponse{
		Adjustments: res.Ajustes, SurplusQty: num(res.Sobras), ShortageQty: num(res.Faltas),
		AdjustedValue: num(res.ValorAjustado), UncountedZeroed: res.NaoContadosZerados,
	}, nil
}

func (uc *CloseInventoryUseCase) enviarParaAprovacao(ctx context.Context, inv *entity.PhysicalInventory, r entity.ResumoFechamento, zerar bool, motivo string) (*response.InventoryCloseResponse, error) {
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do acerto: ele vai para quem aprova")
	}
	if uc.Aprovacoes == nil {
		return nil, errorsuc.NewValidationError("aprovação indisponível nesta instalação")
	}
	ativa, err := uc.Aprovacoes.TemPoliticaAtiva(ctx, aprovacao.AjusteInventario)
	if err != nil {
		return nil, err
	}
	if !ativa {
		return nil, errorsuc.NewValidationError("este inventário exige aprovação, mas a empresa não tem política ativa para ajuste de inventário: cadastre-a em Aprovações (VAPR0100)")
	}
	descricao := fmt.Sprintf("Inventário %d: %d linha(s) com diferença. %s", inv.Code, r.ComDiferenca, motivo)
	if _, err := uc.Aprovacoes.Solicitar(ctx, aprovacao.AjusteInventario, inv.ID, r.ValorAbsoluto, descricao); err != nil {
		return nil, err
	}
	if err := uc.Repo.AguardarAprovacaoDoInventario(ctx, inv.ID, zerar, motivo); err != nil {
		return nil, err
	}
	return &response.InventoryCloseResponse{PendingApproval: true}, nil
}

// EncerrarContagem leva o inventário à conferência.
func (uc *CloseInventoryUseCase) EncerrarContagem(ctx context.Context, id int64) (*response.PhysicalInventoryResponse, error) {
	if !uc.Auth.CanCloseInventory(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := uc.Repo.EncerrarContagem(ctx, id); err != nil {
		return nil, err
	}
	inv, err := uc.Repo.GetInventory(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPhysicalInventoryResponse(inv), nil
}

// Recontar devolve as linhas à contagem e diz quantas voltaram.
func (uc *CloseInventoryUseCase) Recontar(ctx context.Context, id int64, dto request.RecontagemInventarioDTO) (int, error) {
	if !uc.Auth.CanCloseInventory(ctx) {
		return 0, errorsuc.ErrUnauthorized
	}
	inv, err := uc.Repo.GetInventory(ctx, id)
	if err != nil {
		return 0, err
	}
	return uc.Repo.MarcarRecontagem(ctx, inv, dto.LineIDs)
}

// AprovacaoDeInventario aplica no inventário a decisão da alçada.
type AprovacaoDeInventario struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

var _ aprovacao.Efetivador = (*AprovacaoDeInventario)(nil)

// Efetivar lança os ajustes. É idempotente: inventário fechado não lança de novo.
func (a *AprovacaoDeInventario) Efetivar(ctx context.Context, documento, _ int64) error {
	inv, err := a.Repo.GetInventory(ctx, documento)
	if err != nil {
		return err
	}
	if inv.Status == entity.InventarioFechado {
		return nil
	}
	ator, err := a.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	motivo := ""
	if inv.CloseReason != nil {
		motivo = *inv.CloseReason
	}
	_, err = a.Repo.FecharInventario(ctx, documento, ator, inv.ZeroUncounted, motivo)
	return err
}

// Reprovar devolve o inventário à conferência, para recontar ou corrigir.
func (a *AprovacaoDeInventario) Reprovar(ctx context.Context, documento, _ int64, _ string) error {
	return a.Repo.DevolverInventarioParaConferencia(ctx, documento)
}

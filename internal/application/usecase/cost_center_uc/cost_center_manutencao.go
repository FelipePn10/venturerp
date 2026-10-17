package cost_center_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// Manutenção do centro de custo industrial (VCTB0102).
//
// Máquinas, funções, ordens, estruturas e rateios guardam o centro pelo código
// ou pelo id. Por isso:
//   - o código nunca muda;
//   - descrição, data de fim e situação mudam a qualquer momento;
//   - tipo, centro-pai, "é rateio" e data de início só mudam enquanto nada usa o
//     centro — trocá-los com uso reclassificaria custos já apurados;
//   - excluir só sem uso; com uso, a saída é desativar.

var tiposCentro = map[types.TypeCC]bool{
	types.CCAuxiliary: true, types.CCProductive: true, types.CCAdministrative: true, types.CCCommercial: true,
}

func descreverUsos(usos []repository.UsoCentroCusto) string {
	partes := make([]string, 0, len(usos))
	for _, u := range usos {
		partes = append(partes, fmt.Sprintf("%s (%d)", u.Onde, u.Quantidade))
	}
	return strings.Join(partes, ", ")
}

func naoEncontrado(err error) bool {
	var nf *errorsuc.NotFoundError
	return errors.As(err, &nf)
}

func mesmoCodigo(a, b *int32) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func mesmaData(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Format(time.DateOnly) == b.Format(time.DateOnly)
}

// conferirCentroPai: o pai existe na empresa, está ativo e não fica abaixo do
// próprio centro (o que faria um ciclo).
func conferirCentroPai(ctx context.Context, repo repository.CostCenterRepository, code int32, parent *int32) error {
	if parent == nil {
		return nil
	}
	if *parent == code {
		return errorsuc.NewValidationError("o centro não pode ser pai dele mesmo")
	}
	atual := *parent
	for i := 0; i < 50; i++ {
		p, err := repo.GetByCode(ctx, atual)
		if err != nil {
			if naoEncontrado(err) {
				if i == 0 {
					return errorsuc.NewValidationError(fmt.Sprintf("o centro-pai %d não existe nesta empresa", *parent))
				}
				return nil
			}
			return err
		}
		if i == 0 && !p.IsActive {
			return errorsuc.NewValidationError(fmt.Sprintf("o centro-pai %d está inativo — reative-o antes", *parent))
		}
		if p.ParentCode == nil {
			return nil
		}
		if *p.ParentCode == code {
			return errorsuc.NewValidationError(fmt.Sprintf("o centro-pai %d fica abaixo de %d: a árvore viraria um ciclo", *parent, code))
		}
		atual = *p.ParentCode
	}
	return nil
}

type ListCostCentersCadastroUseCase struct {
	Manut repository.CostCenterManutencaoRepository
	Auth  ports.AuthService
}

// Execute lista ativos e inativos (a tela de cadastro reativa os inativos).
func (uc *ListCostCentersCadastroUseCase) Execute(ctx context.Context) ([]*response.CostCenterResponse, error) {
	if !uc.Auth.CanListCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Manut.ListarComInativos(ctx)
	if err != nil {
		return nil, err
	}
	return toCostCenterResponses(list), nil
}

type UsoCostCenterUseCase struct {
	Repo  repository.CostCenterRepository
	Manut repository.CostCenterManutencaoRepository
	Auth  ports.AuthService
}

func (uc *UsoCostCenterUseCase) Execute(ctx context.Context, code int32) (*response.UsoCadastroResumoResponse, error) {
	if !uc.Auth.CanGetCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	cc, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	usos, err := uc.Manut.Usos(ctx, cc)
	if err != nil {
		return nil, err
	}
	out := &response.UsoCadastroResumoResponse{EmUso: len(usos) > 0, Usos: make([]response.UsoCadastroResponse, 0, len(usos))}
	for _, u := range usos {
		out.Usos = append(out.Usos, response.UsoCadastroResponse{Onde: u.Onde, Quantidade: u.Quantidade})
	}
	return out, nil
}

type UpdateCostCenterUseCase struct {
	Repo  repository.CostCenterRepository
	Manut repository.CostCenterManutencaoRepository
	Auth  ports.AuthService
}

func (uc *UpdateCostCenterUseCase) Execute(ctx context.Context, code int32, dto request.UpdateCostCenterDTO) (*response.CostCenterResponse, error) {
	if !uc.Auth.CanEditCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	atual, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	novo := repository.AlteracaoCentroCusto{
		ID:          atual.ID,
		Description: strings.TrimSpace(dto.Description),
		ParentCode:  dto.ParentCode,
		Type:        strings.ToUpper(strings.TrimSpace(dto.Type)),
		IsRatio:     atual.IsRatio,
		StartDate:   atual.StartDate,
		EndDate:     nil,
		IsActive:    atual.IsActive,
	}
	if novo.Description == "" {
		return nil, errorsuc.NewValidationError("a descrição é obrigatória")
	}
	if novo.Type == "" {
		novo.Type = string(atual.Type)
	}
	if !tiposCentro[types.TypeCC(novo.Type)] {
		return nil, errorsuc.NewValidationError("tipo inválido: use AUXILIARY, PRODUCTIVE, ADMINISTRATIVE ou COMMERCIAL")
	}
	if dto.IsRatio != nil {
		novo.IsRatio = *dto.IsRatio
	}
	if dto.IsActive != nil {
		novo.IsActive = *dto.IsActive
	}
	if strings.TrimSpace(dto.StartDate) != "" {
		d, msg := datetime.Obrigatoria("data de início", dto.StartDate)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		novo.StartDate = d
	}
	if dto.EndDate != nil && strings.TrimSpace(*dto.EndDate) != "" {
		d, msg := datetime.Obrigatoria("data de fim", *dto.EndDate)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		novo.EndDate = &d
	}
	if novo.EndDate != nil && novo.EndDate.Before(novo.StartDate) {
		return nil, errorsuc.NewValidationError("a data de fim não pode ser anterior à data de início")
	}

	inicio := atual.StartDate
	estrutural := novo.Type != string(atual.Type) || !mesmoCodigo(novo.ParentCode, atual.ParentCode) ||
		novo.IsRatio != atual.IsRatio || !mesmaData(&novo.StartDate, &inicio)
	if estrutural {
		usos, err := uc.Manut.Usos(ctx, atual)
		if err != nil {
			return nil, err
		}
		if len(usos) > 0 {
			return nil, errorsuc.NewConflictError(fmt.Sprintf(
				"o centro de custo %d já está em uso em %s: só a descrição, a data de fim e a situação podem mudar",
				atual.Code, descreverUsos(usos)))
		}
		if err := conferirCentroPai(ctx, uc.Repo, atual.Code, novo.ParentCode); err != nil {
			return nil, err
		}
	}

	if atual.IsActive && !novo.IsActive {
		filhos, err := uc.Manut.FilhosAtivos(ctx, atual.Code)
		if err != nil {
			return nil, err
		}
		if filhos > 0 {
			return nil, errorsuc.NewConflictError(fmt.Sprintf(
				"o centro de custo %d tem %d centro(s) filho(s) ativo(s): desative os filhos antes", atual.Code, filhos))
		}
	}
	if !atual.IsActive && novo.IsActive && novo.ParentCode != nil && !estrutural {
		pai, err := uc.Repo.GetByCode(ctx, *novo.ParentCode)
		if err != nil && !naoEncontrado(err) {
			return nil, err
		}
		if err == nil && !pai.IsActive {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("o centro-pai %d está inativo: reative-o antes", *novo.ParentCode))
		}
	}

	if err := uc.Manut.Alterar(ctx, novo); err != nil {
		return nil, err
	}
	salvo, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toCostCenterResponse(salvo), nil
}

type DeleteCostCenterUseCase struct {
	Repo  repository.CostCenterRepository
	Manut repository.CostCenterManutencaoRepository
	Auth  ports.AuthService
}

func (uc *DeleteCostCenterUseCase) Execute(ctx context.Context, code int32) error {
	if !uc.Auth.CanDeleteCostCenter(ctx) {
		return errorsuc.ErrUnauthorized
	}
	cc, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	usos, err := uc.Manut.Usos(ctx, cc)
	if err != nil {
		return err
	}
	if len(usos) > 0 {
		return errorsuc.NewConflictError(fmt.Sprintf(
			"o centro de custo %d não pode ser excluído: está em uso em %s. Desative-o para que não seja mais escolhido",
			cc.Code, descreverUsos(usos)))
	}
	return uc.Manut.Excluir(ctx, cc.ID)
}

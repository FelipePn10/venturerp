package financial_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

// Manutenção do plano de contas e do centro de custo financeiros.
//
// A regra é uma só para os dois: o que já foi lançado contra o cadastro continua
// apontando para ele pelo id. Por isso, com uso,
//   - descrição e situação (ativo/inativo) mudam à vontade — renomear e
//     desativar não mudam o significado do que já foi lançado;
//   - código, tipo, natureza e conta-pai NÃO mudam — trocá-los reclassificaria
//     títulos, notas e rateios antigos em silêncio (o DRE de um mês fechado
//     mudaria de linha);
//   - excluir é recusado — deixaria títulos e notas sem plano/centro.
// A recusa lista onde o cadastro é usado, para quem opera saber o que resolver.

var (
	tiposPlanoContas    = map[string]bool{"RECEITA": true, "DESPESA": true, "ATIVO": true, "PASSIVO": true, "PATRIMONIO": true}
	naturezasPlanoConta = map[string]bool{"DEBITO": true, "CREDITO": true}
)

func descreverUsos(usos []entity.UsoCadastro) string {
	partes := make([]string, 0, len(usos))
	for _, u := range usos {
		partes = append(partes, fmt.Sprintf("%s (%d)", u.Onde, u.Quantidade))
	}
	return strings.Join(partes, ", ")
}

func toUsoResumo(usos []entity.UsoCadastro) *response.UsoCadastroResumoResponse {
	out := &response.UsoCadastroResumoResponse{EmUso: len(usos) > 0, Usos: make([]response.UsoCadastroResponse, 0, len(usos))}
	for _, u := range usos {
		out.Usos = append(out.Usos, response.UsoCadastroResponse{Onde: u.Onde, Quantidade: u.Quantidade})
	}
	return out
}

// nivelDoCodigo segue a tela de cadastro: um nível por segmento ("3.1.01" → 3).
func nivelDoCodigo(codigo string) int32 {
	return int32(len(strings.Split(codigo, ".")))
}

func textoOuNulo(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func mesmoTexto(a, b *string) bool {
	a, b = textoOuNulo(a), textoOuNulo(b)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func ehNaoEncontrado(err error) bool {
	var nf *errorsuc.NotFoundError
	return errors.As(err, &nf)
}

// conferirContaPai: a conta-pai existe na empresa, está ativa e não é a própria
// conta nem descendente dela (o que faria um ciclo na árvore).
func conferirContaPai(ctx context.Context, repo repository.FinancialRepository, codigo string, parent *string) error {
	if parent == nil {
		return nil
	}
	if *parent == codigo {
		return errorsuc.NewValidationError("a conta não pode ser pai dela mesma")
	}
	atual := *parent
	for i := 0; i < 50; i++ {
		p, err := repo.GetPlanoContasPorCodigo(ctx, atual)
		if err != nil {
			if ehNaoEncontrado(err) {
				if atual == *parent {
					return errorsuc.NewValidationError(fmt.Sprintf("a conta-pai %s não existe no plano de contas desta empresa", *parent))
				}
				return nil
			}
			return err
		}
		if i == 0 && !p.IsActive {
			return errorsuc.NewValidationError(fmt.Sprintf("a conta-pai %s está inativa — reative-a antes", *parent))
		}
		if p.ParentCode == nil {
			return nil
		}
		if *p.ParentCode == codigo {
			return errorsuc.NewValidationError(fmt.Sprintf("a conta-pai %s fica abaixo de %s: a árvore viraria um ciclo", *parent, codigo))
		}
		atual = *p.ParentCode
	}
	return nil
}

// ---------- Plano de contas ----------

type ListPlanoContasCadastroUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

// Execute lista ativos e inativos — a tela de cadastro precisa dos inativos para
// reativar. As telas que escolhem plano continuam na listagem só de ativos.
func (uc *ListPlanoContasCadastroUseCase) Execute(ctx context.Context) ([]*response.PlanoContasResponse, error) {
	if !uc.Auth.CanListPlanoContas(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListPlanoContasComInativos(ctx)
	if err != nil {
		return nil, err
	}
	return toPlanoContasResponses(list), nil
}

type UsoPlanoContasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *UsoPlanoContasUseCase) Execute(ctx context.Context, id int64) (*response.UsoCadastroResumoResponse, error) {
	if !uc.Auth.CanListPlanoContas(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if _, err := uc.Repo.GetPlanoContas(ctx, id); err != nil {
		return nil, err
	}
	usos, err := uc.Repo.UsosPlanoContas(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUsoResumo(usos), nil
}

type UpdatePlanoContasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *UpdatePlanoContasUseCase) Execute(ctx context.Context, id int64, dto request.UpdatePlanoContasDTO) (*response.PlanoContasResponse, error) {
	if !uc.Auth.CanEditCadastroFinanceiro(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	atual, err := uc.Repo.GetPlanoContas(ctx, id)
	if err != nil {
		return nil, err
	}

	novo := *atual
	novo.Codigo = strings.TrimSpace(dto.Codigo)
	novo.Descricao = strings.TrimSpace(dto.Descricao)
	novo.Tipo = strings.ToUpper(strings.TrimSpace(dto.Tipo))
	novo.Natureza = strings.ToUpper(strings.TrimSpace(dto.Natureza))
	novo.ParentCode = textoOuNulo(dto.ParentCode)
	if novo.Codigo == "" {
		novo.Codigo = atual.Codigo
	}
	if novo.Tipo == "" {
		novo.Tipo = atual.Tipo
	}
	if novo.Natureza == "" {
		novo.Natureza = atual.Natureza
	}
	if dto.IsActive != nil {
		novo.IsActive = *dto.IsActive
	}
	if novo.Descricao == "" {
		return nil, errorsuc.NewValidationError("a descrição é obrigatória")
	}

	estrutural := novo.Codigo != atual.Codigo || novo.Tipo != atual.Tipo ||
		novo.Natureza != atual.Natureza || !mesmoTexto(novo.ParentCode, atual.ParentCode)
	if estrutural {
		if !tiposPlanoContas[novo.Tipo] {
			return nil, errorsuc.NewValidationError("tipo inválido: use RECEITA, DESPESA, ATIVO, PASSIVO ou PATRIMONIO")
		}
		if !naturezasPlanoConta[novo.Natureza] {
			return nil, errorsuc.NewValidationError("natureza inválida: use DEBITO ou CREDITO")
		}
		usos, err := uc.Repo.UsosPlanoContas(ctx, id)
		if err != nil {
			return nil, err
		}
		if len(usos) > 0 {
			return nil, errorsuc.NewConflictError(fmt.Sprintf(
				"a conta %s já está em uso em %s: só a descrição e a situação podem mudar. Para outra classificação, cadastre uma conta nova e desative esta",
				atual.Codigo, descreverUsos(usos)))
		}
		if novo.Codigo != atual.Codigo {
			if _, err := uc.Repo.GetPlanoContasPorCodigo(ctx, novo.Codigo); err == nil {
				return nil, errorsuc.NewConflictError(fmt.Sprintf("já existe uma conta com o código %s", novo.Codigo))
			} else if !ehNaoEncontrado(err) {
				return nil, err
			}
		}
		if err := conferirContaPai(ctx, uc.Repo, novo.Codigo, novo.ParentCode); err != nil {
			return nil, err
		}
		novo.Nivel = nivelDoCodigo(novo.Codigo)
	}

	if atual.IsActive && !novo.IsActive {
		filhas, err := uc.Repo.FilhasAtivasPlanoContas(ctx, atual.Codigo)
		if err != nil {
			return nil, err
		}
		if filhas > 0 {
			return nil, errorsuc.NewConflictError(fmt.Sprintf(
				"a conta %s tem %d conta(s) filha(s) ativa(s): desative as filhas antes", atual.Codigo, filhas))
		}
	}
	if !atual.IsActive && novo.IsActive && novo.ParentCode != nil && !estrutural {
		pai, err := uc.Repo.GetPlanoContasPorCodigo(ctx, *novo.ParentCode)
		if err != nil && !ehNaoEncontrado(err) {
			return nil, err
		}
		if err == nil && !pai.IsActive {
			return nil, errorsuc.NewConflictError(fmt.Sprintf(
				"a conta-pai %s está inativa: reative-a antes", *novo.ParentCode))
		}
	}

	salvo, err := uc.Repo.UpdatePlanoContas(ctx, &novo)
	if err != nil {
		return nil, err
	}
	return toPlanoContasResponse(salvo), nil
}

type DeletePlanoContasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *DeletePlanoContasUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanDeleteCadastroFinanceiro(ctx) {
		return errorsuc.ErrUnauthorized
	}
	atual, err := uc.Repo.GetPlanoContas(ctx, id)
	if err != nil {
		return err
	}
	usos, err := uc.Repo.UsosPlanoContas(ctx, id)
	if err != nil {
		return err
	}
	if len(usos) > 0 {
		return errorsuc.NewConflictError(fmt.Sprintf(
			"a conta %s não pode ser excluída: está em uso em %s. Desative-a para que não seja mais escolhida",
			atual.Codigo, descreverUsos(usos)))
	}
	return uc.Repo.DeletePlanoContas(ctx, id)
}

// ---------- Centro de custo ----------

type ListCentrosCustoCadastroUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListCentrosCustoCadastroUseCase) Execute(ctx context.Context) ([]*response.CentroCustoResponse, error) {
	if !uc.Auth.CanListCentrosCustoFinancial(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListCentrosCustoComInativos(ctx)
	if err != nil {
		return nil, err
	}
	return toCentroCustoResponses(list), nil
}

type UsoCentroCustoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *UsoCentroCustoUseCase) Execute(ctx context.Context, id int64) (*response.UsoCadastroResumoResponse, error) {
	if !uc.Auth.CanListCentrosCustoFinancial(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if _, err := uc.Repo.GetCentroCusto(ctx, id); err != nil {
		return nil, err
	}
	usos, err := uc.Repo.UsosCentroCusto(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUsoResumo(usos), nil
}

type UpdateCentroCustoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *UpdateCentroCustoUseCase) Execute(ctx context.Context, id int64, dto request.UpdateCentroCustoDTO) (*response.CentroCustoResponse, error) {
	if !uc.Auth.CanEditCadastroFinanceiro(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	atual, err := uc.Repo.GetCentroCusto(ctx, id)
	if err != nil {
		return nil, err
	}
	novo := *atual
	novo.Codigo = strings.TrimSpace(dto.Codigo)
	novo.Descricao = strings.TrimSpace(dto.Descricao)
	novo.Tipo = strings.TrimSpace(dto.Tipo)
	if novo.Codigo == "" {
		novo.Codigo = atual.Codigo
	}
	if novo.Tipo == "" {
		novo.Tipo = atual.Tipo
	}
	if dto.IsActive != nil {
		novo.IsActive = *dto.IsActive
	}
	if novo.Descricao == "" {
		return nil, errorsuc.NewValidationError("a descrição é obrigatória")
	}
	if novo.Codigo != atual.Codigo || novo.Tipo != atual.Tipo {
		usos, err := uc.Repo.UsosCentroCusto(ctx, id)
		if err != nil {
			return nil, err
		}
		if len(usos) > 0 {
			return nil, errorsuc.NewConflictError(fmt.Sprintf(
				"o centro de custo %s já está em uso em %s: só a descrição e a situação podem mudar",
				atual.Codigo, descreverUsos(usos)))
		}
	}
	salvo, err := uc.Repo.UpdateCentroCusto(ctx, &novo)
	if err != nil {
		return nil, err
	}
	return toCentroCustoResponse(salvo), nil
}

type DeleteCentroCustoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *DeleteCentroCustoUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanDeleteCadastroFinanceiro(ctx) {
		return errorsuc.ErrUnauthorized
	}
	atual, err := uc.Repo.GetCentroCusto(ctx, id)
	if err != nil {
		return err
	}
	usos, err := uc.Repo.UsosCentroCusto(ctx, id)
	if err != nil {
		return err
	}
	if len(usos) > 0 {
		return errorsuc.NewConflictError(fmt.Sprintf(
			"o centro de custo %s não pode ser excluído: está em uso em %s. Desative-o para que não seja mais escolhido",
			atual.Codigo, descreverUsos(usos)))
	}
	return uc.Repo.DeleteCentroCusto(ctx, id)
}

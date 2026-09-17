package aps_uc

import (
	"context"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	apsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
)

// familiaDeSetupRepo é o recorte do repositório usado pelas famílias de
// preparação. Declarado aqui, e não no pacote de domínio, porque é um cadastro
// de apoio da matriz — não muda o contrato de sequenciamento.
type familiaDeSetupRepo interface {
	ListarFamiliasDeSetup(context.Context) ([]apsrepo.FamiliaDeSetup, error)
	ItensDaFamiliaDeSetup(context.Context, string) ([]string, error)
	DefinirFamiliaDeSetup(context.Context, string, []int64) (int64, error)
}

func (uc *APSUseCase) familiaRepo() (familiaDeSetupRepo, error) {
	repo, ok := uc.repo.(familiaDeSetupRepo)
	if !ok {
		return nil, errorsuc.NewValidationError("o cadastro de famílias de preparação não está disponível")
	}
	return repo, nil
}

// ListarFamiliasDeSetup devolve as famílias em uso, com quantos itens cada uma
// carrega — é o número que mostra se a família cobre o que deveria cobrir.
func (uc *APSUseCase) ListarFamiliasDeSetup(ctx context.Context) ([]apsrepo.FamiliaDeSetup, error) {
	repo, err := uc.familiaRepo()
	if err != nil {
		return nil, err
	}
	return repo.ListarFamiliasDeSetup(ctx)
}

func (uc *APSUseCase) ItensDaFamiliaDeSetup(ctx context.Context, familia string) ([]string, error) {
	repo, err := uc.familiaRepo()
	if err != nil {
		return nil, err
	}
	nome := strings.TrimSpace(familia)
	if nome == "" {
		return nil, errorsuc.NewValidationError("informe a família")
	}
	return repo.ItensDaFamiliaDeSetup(ctx, nome)
}

// DefinirFamiliaDeSetup atribui a família a vários itens de uma vez. Família em
// branco limpa o vínculo — é como se desfaz um agrupamento errado.
func (uc *APSUseCase) DefinirFamiliaDeSetup(ctx context.Context, familia string, itens []int64) (int64, error) {
	repo, err := uc.familiaRepo()
	if err != nil {
		return 0, err
	}
	if len(itens) == 0 {
		return 0, errorsuc.NewValidationError("escolha ao menos um item")
	}
	// Em maiúsculas sem espaço nas pontas: "chapa 3mm" e "CHAPA 3MM" seriam duas
	// famílias diferentes, e a segunda ficaria sem regra nenhuma em silêncio.
	nome := strings.ToUpper(strings.TrimSpace(familia))
	if len(nome) > 60 {
		return 0, errorsuc.NewValidationError("o nome da família deve ter no máximo 60 caracteres")
	}
	return repo.DefinirFamiliaDeSetup(ctx, nome, itens)
}

// Package pdm_uc mantém o padrão descritivo de materiais e o aplica ao item.
package pdm_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/pdm"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type UseCase struct {
	Repo dom.Repository
	Auth ports.AuthService
}

// Previa é a descrição gerada e quem já tem a mesma combinação de atributos.
type Previa struct {
	dom.Resultado
	TemPadrao  bool     `json:"has_standard"`
	Duplicados []string `json:"duplicates"`
}

func (uc *UseCase) Modelo(ctx context.Context, group, modifier int32) (*dom.Modelo, error) {
	return uc.Repo.Modelo(ctx, group, modifier)
}

func (uc *UseCase) ModificadoresDoGrupo(ctx context.Context, group int32) ([]int32, error) {
	return uc.Repo.ModificadoresDoGrupo(ctx, group)
}

func (uc *UseCase) SalvarModificadoresDoGrupo(ctx context.Context, group int32, modificadores []int32) ([]int32, error) {
	if err := uc.autorizar(ctx); err != nil {
		return nil, err
	}
	if err := uc.Repo.SalvarModificadoresDoGrupo(ctx, group, modificadores); err != nil {
		return nil, err
	}
	return uc.Repo.ModificadoresDoGrupo(ctx, group)
}

func (uc *UseCase) SalvarAbreviacao(ctx context.Context, modifier int32, abreviacao *string) error {
	if err := uc.autorizar(ctx); err != nil {
		return err
	}
	if abreviacao != nil {
		s := strings.ToUpper(strings.TrimSpace(*abreviacao))
		abreviacao = &s
		if s == "" {
			abreviacao = nil
		} else if len([]rune(s)) > 20 {
			return errorsuc.NewValidationError("a abreviação tem no máximo 20 caracteres")
		}
	}
	return uc.Repo.SalvarAbreviacao(ctx, modifier, abreviacao)
}

func (uc *UseCase) SalvarAtributo(ctx context.Context, a dom.Atributo) (*dom.Atributo, error) {
	if err := uc.autorizar(ctx); err != nil {
		return nil, err
	}
	if err := a.Conferir(); err != nil {
		return nil, err
	}
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.SalvarAtributo(ctx, &a, por.String())
}

func (uc *UseCase) InativarAtributo(ctx context.Context, id int64) error {
	if err := uc.autorizar(ctx); err != nil {
		return err
	}
	return uc.Repo.InativarAtributo(ctx, id)
}

// Uso conta os itens de cada grupo e modificador.
func (uc *UseCase) Uso(ctx context.Context) (*dom.Uso, error) { return uc.Repo.Uso(ctx) }

// Previa monta a descrição sem gravar, para a tela mostrar enquanto se digita.
func (uc *UseCase) Previa(ctx context.Context, group, modifier int32, valores []valueobject.Attribute, excluir string) (*Previa, error) {
	m, err := uc.Repo.Modelo(ctx, group, modifier)
	if err != nil {
		return nil, err
	}
	r := m.Aplicar(valores)
	out := &Previa{Resultado: r, TemPadrao: m.TemPadrao(), Duplicados: []string{}}
	if out.TemPadrao && len(r.Erros) == 0 {
		if out.Duplicados, err = uc.Repo.ItensComAssinatura(ctx, r.Assinatura, excluir); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// AplicarNoItem confere o par grupo+modificador e, havendo padrão, troca a
// descrição técnica pela gerada. Item antigo que nunca passou pelo padrão e não
// trouxe atributos segue como está: corrigir o peso de um item não pode exigir
// recadastrar a descrição dele.
func (uc *UseCase) AplicarNoItem(ctx context.Context, item *entity.Item, alteracao bool) error {
	if item.PDM.GroupCode == 0 || item.PDM.ModifierCode == 0 {
		return nil
	}
	vinculos, err := uc.Repo.ModificadoresDoGrupo(ctx, item.PDM.GroupCode)
	if err != nil {
		return err
	}
	if len(vinculos) > 0 {
		achou := false
		for _, id := range vinculos {
			achou = achou || id == item.PDM.ModifierCode
		}
		if !achou {
			return errorsuc.NewValidationError(fmt.Sprintf("o modificador %d não pertence ao grupo %d do PDM: vincule em VITE0116", item.PDM.ModifierCode, item.PDM.GroupCode))
		}
	}
	m, err := uc.Repo.Modelo(ctx, item.PDM.GroupCode, item.PDM.ModifierCode)
	if err != nil {
		return err
	}
	if !m.TemPadrao() {
		item.PDM.Signature = nil
		return nil
	}
	if alteracao && len(item.PDM.Attributes) == 0 && item.PDM.Signature == nil {
		return nil
	}
	r := m.Aplicar(item.PDM.Attributes)
	if alteracao {
		r = m.AplicarNaAlteracao(item.PDM.Attributes)
	}
	if len(r.Erros) > 0 {
		return errorsuc.NewValidationError("PDM: " + strings.Join(r.Erros, "; "))
	}
	item.PDM.Attributes = r.Atributos
	item.PDM.DescriptionTechnique = r.Descricao
	assinatura := r.Assinatura
	item.PDM.Signature = &assinatura
	return nil
}

func (uc *UseCase) autorizar(ctx context.Context) error {
	if !uc.Auth.CanCreateItem(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

package service

import (
	"context"
	"testing"
	"time"

	cfgentity "github.com/FelipePn10/panossoerp/internal/domain/configurator/entity"
	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	str "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structqueryrepo "github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository"
	"github.com/google/uuid"
)

// repoFalso responde o mínimo para exercitar a derivação da máscara do filho.
type repoFalso struct {
	filhos          map[int64][]*str.ItemStructure
	perguntas       map[int64][]maskservice.ItemQuestion
	caracteristicas map[int64][]structqueryrepo.ItemCharacteristic
	regras          []structqueryrepo.EquivalentRule
	composicao      map[int64]string
	respostas       map[string][]maskvo.MaskAnswer // "item|mascara"
	criadas         map[string][]maskservice.ChildMaskAnswerInput
}

func (f *repoFalso) GetDirectChildrenForMask(_ context.Context, parent int64, _ string) ([]*str.ItemStructure, error) {
	return f.filhos[parent], nil
}
func (f *repoFalso) GetMaskAnswersByItemAndValue(_ context.Context, item int64, mask string) ([]maskvo.MaskAnswer, error) {
	return f.respostas[chave(item, mask)], nil
}
func (f *repoFalso) GetItemQuestions(_ context.Context, item int64) ([]maskservice.ItemQuestion, error) {
	return f.perguntas[item], nil
}
func (f *repoFalso) CreateMaskForItem(_ context.Context, item int64, mask string, answers []maskservice.ChildMaskAnswerInput, _ uuid.UUID) error {
	if f.criadas == nil {
		f.criadas = map[string][]maskservice.ChildMaskAnswerInput{}
	}
	f.criadas[chave(item, mask)] = answers
	criadas := make([]maskvo.MaskAnswer, 0, len(answers))
	for _, a := range answers {
		r, _ := maskvo.NewMaskAnswer(a.QuestionID, a.OptionID, int(a.Position), f.composicao[a.OptionID])
		criadas = append(criadas, r)
	}
	if f.respostas == nil {
		f.respostas = map[string][]maskvo.MaskAnswer{}
	}
	f.respostas[chave(item, mask)] = criadas
	return nil
}
func (f *repoFalso) ListItemCharacteristics(_ context.Context, item int64) ([]structqueryrepo.ItemCharacteristic, error) {
	return f.caracteristicas[item], nil
}
func (f *repoFalso) ListEquivalentRules(context.Context, int64) ([]structqueryrepo.EquivalentRule, error) {
	return f.regras, nil
}
func (f *repoFalso) GetVariableMaskComposition(_ context.Context, id int64) (string, error) {
	return f.composicao[id], nil
}
func (f *repoFalso) GetMaskAnswersWithNames(context.Context, int64, string) (map[string]float64, error) {
	return nil, nil
}

// Métodos que a interface exige e a derivação não usa.
func (f *repoFalso) ConsultChildren(context.Context, int64, string, *time.Time) ([]*str.ConsultRow, error) {
	return nil, nil
}
func (f *repoFalso) GetLatestMaskForItem(context.Context, int64) (string, error) { return "", nil }
func (f *repoFalso) GetWhereUsed(context.Context, int64, int) ([]*str.WhereUsedRow, error) {
	return nil, nil
}

func chave(item int64, mask string) string { return string(rune(item)) + "|" + mask }

func ptr(v int64) *int64 { return &v }

// TestDerivacaoDaMascaraDoFilho cobre as três origens aceitas para uma
// característica do filho e o caso em que não há nenhuma.
func TestDerivacaoDaMascaraDoFilho(t *testing.T) {
	const (
		pai                  = int64(1)
		filho                = int64(2)
		corDoPai             = int64(10)
		acabamentoPorRegra   = int64(20)
		puxadorPorPadrao     = int64(30)
		semOrigem            = int64(40)
		varBranco, varFosco  = int64(100), int64(200)
		varPuxadorPadrao     = int64(300)
		varAcabamentoDaRegra = int64(400)
	)
	respostaDoPai, _ := maskvo.NewMaskAnswer(corDoPai, varBranco, 1, "BR")

	novoRepo := func(perguntasDoFilho []maskservice.ItemQuestion, caracteristicas []structqueryrepo.ItemCharacteristic) *repoFalso {
		return &repoFalso{
			filhos: map[int64][]*str.ItemStructure{
				pai: {{ParentCode: pai, ChildCode: filho, Quantity: 1, Inherit: true}},
			},
			perguntas:       map[int64][]maskservice.ItemQuestion{filho: perguntasDoFilho},
			caracteristicas: map[int64][]structqueryrepo.ItemCharacteristic{filho: caracteristicas},
			regras: []structqueryrepo.EquivalentRule{{
				ChildItemCode: filho, ParentCharacteristicID: corDoPai, ParentOperator: cfgentity.OpEqual,
				ParentVariableID: ptr(varBranco), ChildCharacteristicID: acabamentoPorRegra,
				ChildVariableID: ptr(varAcabamentoDaRegra),
			}},
			composicao: map[int64]string{varBranco: "BR", varFosco: "FO", varPuxadorPadrao: "PX", varAcabamentoDaRegra: "AC"},
		}
	}

	t.Run("herda do pai, da regra de equivalência e da resposta padrão", func(t *testing.T) {
		repo := novoRepo(
			[]maskservice.ItemQuestion{{QuestionID: corDoPai, Position: 1}, {QuestionID: acabamentoPorRegra, Position: 2}, {QuestionID: puxadorPorPadrao, Position: 3}},
			[]structqueryrepo.ItemCharacteristic{
				{CharacteristicID: corDoPai, Code: "COR"},
				{CharacteristicID: acabamentoPorRegra, Code: "ACABAMENTO"},
				{CharacteristicID: puxadorPorPadrao, Code: "PUXADOR", DefaultVariableID: ptr(varPuxadorPadrao)},
			})
		nos, err := NewResolver(repo).Resolve(context.Background(), pai, "BR", []maskvo.MaskAnswer{respostaDoPai}, 1, map[int64]bool{}, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		if len(nos) != 1 {
			t.Fatalf("esperado 1 componente, obtido %d", len(nos))
		}
		no := nos[0]
		if no.ConfiguracaoIncompleta {
			t.Fatalf("configuração deveria fechar; faltantes=%v", no.CaracteristicasFaltantes)
		}
		if no.EffectiveMask == nil || *no.EffectiveMask != "BR#AC#PX" {
			t.Fatalf("máscara derivada errada: %v", no.EffectiveMask)
		}
	})

	t.Run("característica sem origem deixa a configuração incompleta", func(t *testing.T) {
		repo := novoRepo(
			[]maskservice.ItemQuestion{{QuestionID: corDoPai, Position: 1}, {QuestionID: semOrigem, Position: 2}},
			[]structqueryrepo.ItemCharacteristic{
				{CharacteristicID: corDoPai, Code: "COR"},
				{CharacteristicID: semOrigem, Code: "PROFUNDIDADE"},
			})
		nos, err := NewResolver(repo).Resolve(context.Background(), pai, "BR", []maskvo.MaskAnswer{respostaDoPai}, 1, map[int64]bool{}, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		no := nos[0]
		if !no.ConfiguracaoIncompleta {
			t.Fatal("deveria marcar configuração incompleta")
		}
		if len(no.CaracteristicasFaltantes) != 1 || no.CaracteristicasFaltantes[0] != "PROFUNDIDADE" {
			t.Fatalf("faltantes esperados [PROFUNDIDADE], obtido %v", no.CaracteristicasFaltantes)
		}
		if no.EffectiveMask != nil {
			t.Fatalf("sem configuração completa não pode haver máscara: %v", *no.EffectiveMask)
		}
	})
}

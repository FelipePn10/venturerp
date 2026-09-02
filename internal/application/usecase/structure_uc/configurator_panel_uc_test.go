package structure_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type panelItems struct{ item *itementity.Item }

func (f *panelItems) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*itementity.Item, error) {
	if f.item != nil && f.item.BusinessCode == code {
		return f.item, nil
	}
	return nil, itemrepo.ErrNotFound
}
func (f *panelItems) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	if f.item != nil && f.item.Code == code {
		return f.item, nil
	}
	return nil, itemrepo.ErrNotFound
}

type fakeConfigurator struct {
	panel      *response.StructureConfiguratorPanelResponse
	violations []response.StructureConfiguratorViolation
	mask       *response.CfgGeneratedMaskResponse
	maskErr    error
	vars       map[string]float64
	lastDTO    request.CfgGenerateMaskDTO
}

func (f *fakeConfigurator) StructurePanel(_ context.Context, itemCode int64, itemName string) (*response.StructureConfiguratorPanelResponse, error) {
	out := *f.panel
	out.ItemName = itemName
	return &out, nil
}
func (f *fakeConfigurator) ValidateCombination(context.Context, int64, []request.CfgMaskAnswerInput) ([]response.StructureConfiguratorViolation, error) {
	return f.violations, nil
}
func (f *fakeConfigurator) GenerateMask(_ context.Context, dto request.CfgGenerateMaskDTO) (*response.CfgGeneratedMaskResponse, error) {
	f.lastDTO = dto
	if f.maskErr != nil {
		return nil, f.maskErr
	}
	out := *f.mask
	out.Persisted = dto.Persist
	return &out, nil
}
func (f *fakeConfigurator) FormulaVariables(context.Context, []request.CfgMaskAnswerInput) map[string]float64 {
	return f.vars
}

type fakeResolver struct {
	tree *response.StructureTreeResponse
	err  error
	mask string
}

func (f *fakeResolver) Execute(_ context.Context, dto request.ResolveStructureQueryDTO) (*response.StructureTreeResponse, error) {
	f.mask = dto.Mask
	return f.tree, f.err
}

type panelAuth struct {
	ports.AuthService
	actor uuid.UUID
}

func (a *panelAuth) UserID(context.Context) (uuid.UUID, error) { return a.actor, nil }

func newPanelUC(t *testing.T) (*StructureConfiguratorUseCase, *fakeConfigurator, *fakeResolver) {
	t.Helper()
	cfg := &fakeConfigurator{
		panel: &response.StructureConfiguratorPanelResponse{
			Configurable: true,
			Questions: []response.StructureConfiguratorQuestion{
				{CharacteristicID: 1, Code: "COMPRIMENTO", Sequence: 1, UsedByFormula: true},
			},
		},
		mask: &response.CfgGeneratedMaskResponse{Mask: "1200#600", MaskHash: "abcd1234"},
		vars: map[string]float64{"COMPRIMENTO": 1200},
	}
	res := &fakeResolver{tree: &response.StructureTreeResponse{TotalNodes: 2}}
	items := &panelItems{item: &itementity.Item{Code: 900, BusinessCode: "QD-01", Name: "Quadro 01"}}
	return NewStructureConfiguratorUseCase(cfg, res, &panelAuth{actor: uuid.New()}, items), cfg, res
}

// O painel é aberto pelo código de negócio do item, como a tela o conhece.
func TestPanel_ReturnsItemBusinessCodeAndName(t *testing.T) {
	uc, _, _ := newPanelUC(t)
	got, err := uc.Panel(context.Background(), "QD-01")
	if err != nil {
		t.Fatalf("painel recusado: %v", err)
	}
	if got.ItemCode != "QD-01" || got.ItemName != "Quadro 01" {
		t.Fatalf("painel sem identidade do item: %+v", got)
	}
	if !got.Configurable || len(got.Questions) != 1 {
		t.Fatalf("painel sem perguntas: %+v", got)
	}
}

func TestPanel_UnknownItemIsRejected(t *testing.T) {
	uc, _, _ := newPanelUC(t)
	if _, err := uc.Panel(context.Background(), "NAO-EXISTE"); err == nil {
		t.Fatal("item inexistente devolveu painel")
	}
}

// Restrição violada barra a configuração antes de qualquer gravação.
func TestApply_RestrictionViolationBlocksBeforePersisting(t *testing.T) {
	uc, cfg, _ := newPanelUC(t)
	cfg.violations = []response.StructureConfiguratorViolation{
		{CharacteristicID: 1, Message: "Comprimento: a resposta precisa ser menor que 3000"},
	}
	_, err := uc.Apply(context.Background(), "QD-01", request.ApplyStructureConfigurationDTO{
		Answers: []request.CfgMaskAnswerInput{{CharacteristicID: 1, Value: "4000"}},
		Persist: true,
	})
	var violation *RestrictionViolationError
	if !errors.As(err, &violation) {
		t.Fatalf("esperado RestrictionViolationError, veio %T (%v)", err, err)
	}
	if len(violation.Violations) != 1 {
		t.Fatalf("violações = %d, quer 1", len(violation.Violations))
	}
	if cfg.lastDTO.ItemCode != 0 {
		t.Fatal("a máscara foi gerada apesar da restrição violada")
	}
}

// Configuração gravada devolve a estrutura já resolvida para aquela máscara.
func TestApply_PersistedReturnsResolvedStructure(t *testing.T) {
	uc, cfg, res := newPanelUC(t)
	got, err := uc.Apply(context.Background(), "QD-01", request.ApplyStructureConfigurationDTO{
		Answers: []request.CfgMaskAnswerInput{{CharacteristicID: 1, Value: "1200"}},
		Persist: true,
	})
	if err != nil {
		t.Fatalf("aplicação recusada: %v", err)
	}
	if !got.Persisted || got.Structure == nil || got.Structure.TotalNodes != 2 {
		t.Fatalf("estrutura não resolvida: %+v", got)
	}
	if res.mask != "1200#600" {
		t.Fatalf("estrutura resolvida com máscara %q", res.mask)
	}
	if got.Variables["COMPRIMENTO"] != 1200 {
		t.Fatalf("variáveis da fórmula ausentes: %+v", got.Variables)
	}
	if cfg.lastDTO.ItemCode != 900 || !cfg.lastDTO.Persist {
		t.Fatalf("máscara gerada com %+v", cfg.lastDTO)
	}
	if cfg.lastDTO.CreatedBy == uuid.Nil {
		t.Fatal("autor não veio do JWT")
	}
}

// Simulação não resolve a estrutura e avisa que nada foi gravado.
func TestApply_SimulationWarnsAndSkipsResolution(t *testing.T) {
	uc, _, res := newPanelUC(t)
	got, err := uc.Apply(context.Background(), "QD-01", request.ApplyStructureConfigurationDTO{
		Answers: []request.CfgMaskAnswerInput{{CharacteristicID: 1, Value: "1200"}},
	})
	if err != nil {
		t.Fatalf("simulação recusada: %v", err)
	}
	if got.Persisted || got.Structure != nil {
		t.Fatalf("simulação gravou/resolveu: %+v", got)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("simulação sem aviso de que nada foi gravado")
	}
	if res.mask != "" {
		t.Fatal("a estrutura foi resolvida numa simulação")
	}
}

func TestApply_EmptyAnswersIsValidationError(t *testing.T) {
	uc, _, _ := newPanelUC(t)
	_, err := uc.Apply(context.Background(), "QD-01", request.ApplyStructureConfigurationDTO{Persist: true})
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
}

// Falha ao resolver a estrutura não perde a configuração já gravada.
func TestApply_ResolutionFailureKeepsPersistedConfiguration(t *testing.T) {
	uc, _, res := newPanelUC(t)
	res.err = errors.New("máscara ainda propagando")
	res.tree = nil
	got, err := uc.Apply(context.Background(), "QD-01", request.ApplyStructureConfigurationDTO{
		Answers: []request.CfgMaskAnswerInput{{CharacteristicID: 1, Value: "1200"}},
		Persist: true,
	})
	if err != nil {
		t.Fatalf("aplicação recusada: %v", err)
	}
	if !got.Persisted || got.Structure != nil || len(got.Warnings) == 0 {
		t.Fatalf("resultado = %+v", got)
	}
}

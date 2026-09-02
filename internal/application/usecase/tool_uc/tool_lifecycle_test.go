package tool_uc

import (
	"context"
	"errors"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/tool/repository"
)

// lifecycleRepo é um repositório de ferramentas mínimo com o mesmo
// comportamento do banco: a inativação não avisa quando não encontra a linha.
type lifecycleRepo struct {
	repository.ToolRepository
	tools   map[int64]*entity.Tool
	serials map[int64]*entity.ToolSerial
	// silent reproduz o UPDATE que não altera nada e mesmo assim não falha.
	silent bool
}

func (r *lifecycleRepo) GetTool(_ context.Context, id int64) (*entity.Tool, error) {
	t, ok := r.tools[id]
	if !ok {
		return nil, errors.New("no rows in result set")
	}
	copied := *t
	return &copied, nil
}

func (r *lifecycleRepo) DeactivateTool(_ context.Context, id int64) error {
	if r.silent {
		return nil
	}
	if t, ok := r.tools[id]; ok {
		t.IsActive = false
	}
	return nil
}

func (r *lifecycleRepo) ResetToolLife(_ context.Context, id int64) (*entity.Tool, error) {
	t, ok := r.tools[id]
	if !ok {
		return nil, errors.New("no rows in result set")
	}
	if !r.silent {
		t.LifeUsed, t.IsActive = 0, true
	}
	copied := *t
	return &copied, nil
}

func (r *lifecycleRepo) GetToolSerial(_ context.Context, id int64) (*entity.ToolSerial, error) {
	s, ok := r.serials[id]
	if !ok {
		return nil, errors.New("no rows in result set")
	}
	copied := *s
	return &copied, nil
}

func (r *lifecycleRepo) DeactivateToolSerial(_ context.Context, id int64) error {
	if r.silent {
		return nil
	}
	if s, ok := r.serials[id]; ok {
		s.IsActive = false
	}
	return nil
}

func newLifecycleUC() (*ToolUseCase, *lifecycleRepo) {
	repo := &lifecycleRepo{
		tools:   map[int64]*entity.Tool{1: {ID: 1, Code: 10, Name: "Estampo", LifeLimit: 1000, LifeUsed: 940, IsActive: true}},
		serials: map[int64]*entity.ToolSerial{5: {ID: 5, ToolID: 1, SerialNumber: "S-01", IsActive: true}},
	}
	return New(repo), repo
}

func TestGetTool_UnknownIsNotFound(t *testing.T) {
	uc, _ := newLifecycleUC()
	_, err := uc.Get(context.Background(), 99)
	if _, ok := errorsuc.AsNotFound(err); !ok {
		t.Fatalf("esperado NotFoundError, veio %T (%v)", err, err)
	}
}

func TestDeactivateTool_PersistsAndIsIdempotencyAware(t *testing.T) {
	uc, repo := newLifecycleUC()
	if err := uc.Deactivate(context.Background(), 1); err != nil {
		t.Fatalf("inativação recusada: %v", err)
	}
	if repo.tools[1].IsActive {
		t.Fatal("ferramenta continua ativa")
	}
	if _, ok := errorsuc.AsConflict(uc.Deactivate(context.Background(), 1)); !ok {
		t.Fatal("segunda inativação deveria ser conflito")
	}
}

func TestDeactivateTool_UnknownIsNotFound(t *testing.T) {
	uc, _ := newLifecycleUC()
	if _, ok := errorsuc.AsNotFound(uc.Deactivate(context.Background(), 99)); !ok {
		t.Fatal("ferramenta inexistente não devolveu 404")
	}
}

// Um UPDATE que não altera nada não pode responder sucesso à tela — era esse o
// "inativar/zerar sem efeito" relatado.
func TestDeactivateTool_SilentUpdateIsReported(t *testing.T) {
	uc, repo := newLifecycleUC()
	repo.silent = true
	if err := uc.Deactivate(context.Background(), 1); err == nil {
		t.Fatal("inativação sem efeito respondeu sucesso")
	}
}

func TestResetToolLife_ZeroesUsage(t *testing.T) {
	uc, repo := newLifecycleUC()
	got, err := uc.ResetLife(context.Background(), 1)
	if err != nil {
		t.Fatalf("zeragem recusada: %v", err)
	}
	if got.LifeUsed != 0 || repo.tools[1].LifeUsed != 0 {
		t.Fatalf("vida útil não zerada: resposta=%v banco=%v", got.LifeUsed, repo.tools[1].LifeUsed)
	}
	if got.RemainingLife != 1000 || got.NeedsReplacement {
		t.Fatalf("vida restante após a troca: %+v", got)
	}
}

func TestResetToolLife_SilentUpdateIsReported(t *testing.T) {
	uc, repo := newLifecycleUC()
	repo.silent = true
	if _, err := uc.ResetLife(context.Background(), 1); err == nil {
		t.Fatal("zeragem sem efeito respondeu sucesso")
	}
}

func TestResetToolLife_UnknownIsNotFound(t *testing.T) {
	uc, _ := newLifecycleUC()
	if _, err := uc.ResetLife(context.Background(), 99); err == nil {
		t.Fatal("ferramenta inexistente aceita")
	}
}

func TestDeactivateSerial_PersistsAndReportsSilentUpdate(t *testing.T) {
	uc, repo := newLifecycleUC()
	if err := uc.DeactivateSerial(context.Background(), 5); err != nil {
		t.Fatalf("inativação de série recusada: %v", err)
	}
	if repo.serials[5].IsActive {
		t.Fatal("série continua ativa")
	}
	repo.serials[5].IsActive, repo.silent = true, true
	if err := uc.DeactivateSerial(context.Background(), 5); err == nil {
		t.Fatal("inativação de série sem efeito respondeu sucesso")
	}
	if _, ok := errorsuc.AsNotFound(uc.DeactivateSerial(context.Background(), 99)); !ok {
		t.Fatal("série inexistente não devolveu 404")
	}
}

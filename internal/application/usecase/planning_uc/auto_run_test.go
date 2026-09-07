package planning_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/google/uuid"
)

type repoFalso struct {
	travado      bool
	tentouTravar int
	iniciou      []RunRecord
	finalizou    []RunRecord
	cortes       []time.Time
	statusCorte  []string
	mudaram      int
}

func (r *repoFalso) DueSettings(context.Context, time.Time) ([]AutoRunSettings, error) {
	return nil, nil
}
func (r *repoFalso) Settings(context.Context, int64) (*AutoRunSettings, error) { return nil, nil }
func (r *repoFalso) TryLock(context.Context, int64) (bool, error) {
	r.tentouTravar++
	if r.travado {
		return false, nil
	}
	r.travado = true
	return true, nil
}
func (r *repoFalso) Unlock(context.Context, int64) error { r.travado = false; return nil }
func (r *repoFalso) ChangedItemsSince(context.Context, int64, time.Time) (int, error) {
	return r.mudaram, nil
}
func (r *repoFalso) StartRun(_ context.Context, rec RunRecord) (int64, error) {
	r.iniciou = append(r.iniciou, rec)
	return int64(len(r.iniciou)), nil
}
func (r *repoFalso) FinishRun(_ context.Context, _ int64, rec RunRecord) error {
	r.finalizou = append(r.finalizou, rec)
	return nil
}
func (r *repoFalso) SaveSnapshot(_ context.Context, _ int64, at time.Time, status string) error {
	r.cortes = append(r.cortes, at)
	r.statusCorte = append(r.statusCorte, status)
	return nil
}

// pipelineFalso devolve um resultado fixo, para o teste focar no agendamento.
type pipelineFalso struct {
	chamadas []request.RunPlanningPipelineDTO
	erro     error
}

func (p *pipelineFalso) Execute(_ context.Context, dto request.RunPlanningPipelineDTO) (*response.PlanningPipelineResponse, error) {
	p.chamadas = append(p.chamadas, dto)
	if p.erro != nil {
		return nil, p.erro
	}
	return &response.PlanningPipelineResponse{PlanCode: dto.PlanCode, Viable: true, MRPOrders: 3}, nil
}

func settings() AutoRunSettings {
	anterior := time.Date(2026, 9, 4, 2, 0, 0, 0, time.UTC)
	return AutoRunSettings{
		EnterpriseID: 1, IsEnabled: true, RunHour: 2, PlanCode: 10,
		InitialOrderNumber: 1, GenerateLLC: true, LastSnapshotAt: &anterior,
	}
}

// Enquanto um ciclo roda, outro disparo não pode entrar: duas execuções sobre
// as mesmas ordens planejadas se atropelariam.
func TestRunRecusaExecucaoConcorrente(t *testing.T) {
	repo := &repoFalso{travado: true}
	uc := &AutoRunPlanningUseCase{Pipeline: &pipelineFalso{}, Repo: repo, Now: func() time.Time { return time.Now() }}

	_, err := uc.Run(context.Background(), settings(), TriggerManual, nil)
	if err == nil {
		t.Fatal("segunda execução foi aceita com o cadeado tomado")
	}
	var conflito *errorsuc.ConflictError
	if !errors.As(err, &conflito) {
		t.Fatalf("erro deveria ser de conflito, veio %T: %v", err, err)
	}
	if len(repo.iniciou) != 0 {
		t.Fatal("registrou execução mesmo sem conseguir o cadeado")
	}
}

// O plano é obrigatório: sem ele não há o que calcular.
func TestRunExigePlanoConfigurado(t *testing.T) {
	uc := &AutoRunPlanningUseCase{Repo: &repoFalso{}}
	s := settings()
	s.PlanCode = 0
	if _, err := uc.Run(context.Background(), s, TriggerAutomatico, nil); err == nil {
		t.Fatal("aceitou execução sem plano")
	}
}

// O corte é gravado antes de o ciclo ler os dados e identifica o que mudou
// desde a execução anterior — é o que garante que nada se perca entre ciclos.
func TestRunRegistraCorteEItensAlterados(t *testing.T) {
	repo := &repoFalso{mudaram: 7}
	fixo := time.Date(2026, 9, 5, 2, 0, 0, 0, time.UTC)
	pipe := &pipelineFalso{}
	uc := &AutoRunPlanningUseCase{Pipeline: pipe, Repo: repo, Now: func() time.Time { return fixo }}
	ator := uuid.New()

	if _, err := uc.Run(context.Background(), settings(), TriggerManual, &ator); err != nil {
		t.Fatal(err)
	}
	// O pipeline recebe o mesmo instante do corte, para ler e sequenciar a
	// partir de um ponto único no tempo.
	if len(pipe.chamadas) != 1 || !pipe.chamadas[0].StartFrom.Equal(fixo) {
		t.Fatalf("pipeline não recebeu o corte: %+v", pipe.chamadas)
	}

	if len(repo.iniciou) != 1 {
		t.Fatalf("execuções registradas = %d, esperado 1", len(repo.iniciou))
	}
	rec := repo.iniciou[0]
	if !rec.SnapshotAt.Equal(fixo) {
		t.Fatalf("corte = %v, esperado %v", rec.SnapshotAt, fixo)
	}
	if rec.ChangedItems != 7 {
		t.Fatalf("itens alterados = %d, esperado 7", rec.ChangedItems)
	}
	if rec.Trigger != TriggerManual || rec.TriggeredBy == nil || *rec.TriggeredBy != ator {
		t.Fatalf("origem do disparo não registrada: %+v", rec)
	}
	if repo.travado {
		t.Fatal("o cadeado ficou preso após a execução")
	}
}

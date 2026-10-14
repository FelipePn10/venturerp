package planning_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
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
	pendentes    []AutoRunSettings
	salvos       []AutoRunSettings
	planos       map[int64]bool
}

func (r *repoFalso) DueSettings(context.Context, time.Time) ([]AutoRunSettings, error) {
	return r.pendentes, nil
}
func (r *repoFalso) SaveSettings(_ context.Context, s AutoRunSettings) error {
	r.salvos = append(r.salvos, s)
	return nil
}
func (r *repoFalso) ListRuns(context.Context, int64, int) ([]RunRecord, error) { return nil, nil }
func (r *repoFalso) PlanExists(_ context.Context, _ int64, plano int64) (bool, error) {
	return r.planos[plano], nil
}
func (r *repoFalso) Settings(_ context.Context, id int64) (*AutoRunSettings, error) {
	return &AutoRunSettings{EnterpriseID: id, RunHour: 2, InitialOrderNumber: 1}, nil
}
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
	sessoes  []any
	erro     error
}

type chaveSessao struct{}

func (p *pipelineFalso) Execute(ctx context.Context, dto request.RunPlanningPipelineDTO) (*response.PlanningPipelineResponse, error) {
	p.chamadas = append(p.chamadas, dto)
	p.sessoes = append(p.sessoes, ctx.Value(chaveSessao{}))
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

// A janela noturna roda sem ninguém logado: cada empresa precisa da própria
// sessão (empresa e responsável), senão o MRP não enxerga os dados dela nem tem
// em nome de quem gravar as ordens. Antes o pipeline recebia o contexto cru do
// agendador e toda execução automática falhava.
func TestRunDueRodaNaSessaoDeCadaEmpresa(t *testing.T) {
	quem := uuid.New()
	a, b := settings(), settings()
	b.EnterpriseID = 2
	a.UpdatedBy, b.UpdatedBy = &quem, &quem
	repo := &repoFalso{pendentes: []AutoRunSettings{a, b}}
	pipe := &pipelineFalso{}
	uc := &AutoRunPlanningUseCase{
		Pipeline: pipe, Repo: repo,
		Session: func(ctx context.Context, s AutoRunSettings) context.Context {
			return context.WithValue(ctx, chaveSessao{}, s.EnterpriseID)
		},
	}
	n, err := uc.RunDue(context.Background())
	if err != nil || n != 2 {
		t.Fatalf("executadas = %d (%v), esperado 2", n, err)
	}
	if len(pipe.sessoes) != 2 || pipe.sessoes[0] != int64(1) || pipe.sessoes[1] != int64(2) {
		t.Fatalf("pipeline sem a sessão da empresa: %+v", pipe.sessoes)
	}
	if repo.iniciou[0].TriggeredBy == nil || *repo.iniciou[0].TriggeredBy != quem || repo.iniciou[0].Trigger != TriggerAutomatico {
		t.Fatalf("execução automática sem responsável: %+v", repo.iniciou[0])
	}
}

type authFalsa struct {
	ports.AuthService
	empresa int64
	ator    uuid.UUID
	pode    bool
}

func (a *authFalsa) EnterpriseID(context.Context) (int64, error) { return a.empresa, nil }
func (a *authFalsa) UserID(context.Context) (uuid.UUID, error)   { return a.ator, nil }
func (a *authFalsa) CanRunMRPCalculation(context.Context) bool   { return a.pode }

// Ligar exige plano da própria empresa e horário válido; quem salva vira o
// responsável pela execução automática.
func TestUpdateSettingsValidaEGravaResponsavel(t *testing.T) {
	ator := uuid.New()
	repo := &repoFalso{planos: map[int64]bool{10: true}}
	uc := &AutoRunPlanningUseCase{Repo: repo, Auth: &authFalsa{empresa: 7, ator: ator, pode: true}}

	casos := []struct {
		nome string
		dto  request.UpdateAutoRunSettingsDTO
	}{
		{"hora fora do dia", request.UpdateAutoRunSettingsDTO{IsEnabled: true, RunHour: 24, PlanCode: 10}},
		{"minuto inválido", request.UpdateAutoRunSettingsDTO{IsEnabled: true, RunMinute: 60, PlanCode: 10}},
		{"ligada sem plano", request.UpdateAutoRunSettingsDTO{IsEnabled: true, RunHour: 2}},
		{"plano de outra empresa", request.UpdateAutoRunSettingsDTO{IsEnabled: true, RunHour: 2, PlanCode: 99}},
	}
	for _, c := range casos {
		if _, err := uc.UpdateSettings(context.Background(), c.dto); err == nil {
			t.Fatalf("%s: aceito", c.nome)
		}
	}
	if len(repo.salvos) != 0 {
		t.Fatalf("gravou parametrização inválida: %+v", repo.salvos)
	}

	if _, err := uc.UpdateSettings(context.Background(), request.UpdateAutoRunSettingsDTO{IsEnabled: true, RunHour: 3, RunMinute: 30, PlanCode: 10}); err != nil {
		t.Fatal(err)
	}
	s := repo.salvos[0]
	if s.EnterpriseID != 7 || s.UpdatedBy == nil || *s.UpdatedBy != ator || s.InitialOrderNumber != 1 {
		t.Fatalf("parametrização salva errada: %+v", s)
	}

	sem := &AutoRunPlanningUseCase{Repo: repo, Auth: &authFalsa{empresa: 7, pode: false}}
	if _, err := sem.UpdateSettings(context.Background(), request.UpdateAutoRunSettingsDTO{RunHour: 2}); !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("sem permissão: %v", err)
	}
}

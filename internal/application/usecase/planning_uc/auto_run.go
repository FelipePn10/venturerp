package planning_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/google/uuid"
)

// pipelineRunner é a parte do pipeline que a execução automática usa. É uma
// interface para o agendamento poder ser testado sem montar MRP, CRP e APS.
type pipelineRunner interface {
	Execute(context.Context, request.RunPlanningPipelineDTO) (*response.PlanningPipelineResponse, error)
}

// Trigger diz o que disparou a execução do planejamento.
type Trigger string

const (
	TriggerAutomatico Trigger = "AUTOMATICO"
	TriggerManual     Trigger = "MANUAL"
)

// AutoRunSettings é a janela noturna configurada para uma empresa.
type AutoRunSettings struct {
	EnterpriseID       int64
	EnterpriseCode     int64
	IsEnabled          bool
	RunHour            int
	RunMinute          int
	PlanCode           int64
	InitialOrderNumber int64
	GenerateLLC        bool
	LastSnapshotAt     *time.Time
	LastRunAt          *time.Time
	LastRunStatus      string
	// UpdatedBy é quem salvou a parametrização: a execução automática roda em
	// nome dele, como se ele tivesse clicado em "Calcular".
	UpdatedBy *uuid.UUID
}

// RunRecord é a linha de histórico de uma execução.
type RunRecord struct {
	ID           int64
	EnterpriseID int64
	PlanCode     int64
	Trigger      Trigger
	TriggeredBy  *uuid.UUID
	// TriggeredByName é o nome de quem disparou (ou de quem ligou a execução
	// automática), para o histórico não mostrar um identificador.
	TriggeredByName string
	SnapshotAt      time.Time
	StartedAt       time.Time
	FinishedAt      *time.Time
	ChangedItems    int
	MRPOrders       int
	CRPOverload     int
	Viable          *bool
	Status          string
	Notes           string
}

// PlanningRunRepository guarda a parametrização, o histórico e o cadeado que
// impede duas execuções simultâneas.
type PlanningRunRepository interface {
	// DueSettings devolve as empresas cuja janela noturna venceu em `now` e que
	// ainda não rodaram nesse dia.
	DueSettings(ctx context.Context, now time.Time) ([]AutoRunSettings, error)
	// Settings devolve a parametrização de uma empresa.
	Settings(ctx context.Context, enterpriseID int64) (*AutoRunSettings, error)
	// TryLock pega o cadeado da empresa. Devolve false quando outra execução
	// está em andamento — nunca bloqueia esperando.
	TryLock(ctx context.Context, enterpriseID int64) (bool, error)
	Unlock(ctx context.Context, enterpriseID int64) error
	// ChangedItemsSince conta os itens cujo estoque, ordem, estrutura ou demanda
	// mudou depois do corte informado.
	ChangedItemsSince(ctx context.Context, enterpriseID int64, since time.Time) (int, error)
	StartRun(ctx context.Context, rec RunRecord) (int64, error)
	FinishRun(ctx context.Context, id int64, rec RunRecord) error
	SaveSnapshot(ctx context.Context, enterpriseID int64, snapshotAt time.Time, status string) error
	// SaveSettings grava a parametrização da empresa sem tocar no corte nem na
	// última execução.
	SaveSettings(ctx context.Context, s AutoRunSettings) error
	// ListRuns devolve as últimas execuções da empresa, da mais recente para a
	// mais antiga.
	ListRuns(ctx context.Context, enterpriseID int64, limit int) ([]RunRecord, error)
	// PlanExists diz se o plano pertence à empresa.
	PlanExists(ctx context.Context, enterpriseID, planCode int64) (bool, error)
}

// AutoRunPlanningUseCase executa o planejamento com as garantias que um MRP
// precisa dar.
//
// Duas execuções nunca se sobrepõem: a segunda encontra o cadeado tomado e é
// registrada como pulada, em vez de disputar as mesmas ordens planejadas. E
// cada execução grava o instante em que leu os dados (o corte). Quem mexer em
// estoque, ordem ou estrutura durante a execução não corrompe o ciclo em
// andamento — a alteração é detectada pelo corte e replanejada na próxima
// rodada. É o mesmo princípio do planning file do SAP: o ciclo corrente
// permanece coerente e nada que mudou se perde.
type AutoRunPlanningUseCase struct {
	Pipeline pipelineRunner
	Repo     PlanningRunRepository
	Auth     ports.AuthService
	// Session monta a sessão da empresa para a execução automática, que roda
	// sem ninguém logado. Sem ela o MRP não sabe de qual empresa são os dados
	// nem em nome de quem gravar as ordens.
	Session func(context.Context, AutoRunSettings) context.Context
	// Now permite fixar o relógio nos testes.
	Now func() time.Time
}

func (uc *AutoRunPlanningUseCase) now() time.Time {
	if uc.Now != nil {
		return uc.Now()
	}
	return time.Now()
}

// RunDue executa a janela noturna de todas as empresas que venceram agora.
// Devolve quantas empresas rodaram.
func (uc *AutoRunPlanningUseCase) RunDue(ctx context.Context) (int, error) {
	pendentes, err := uc.Repo.DueSettings(ctx, uc.now())
	if err != nil {
		return 0, err
	}
	executadas := 0
	for _, s := range pendentes {
		ectx := ctx
		if uc.Session != nil {
			ectx = uc.Session(ctx, s)
		}
		if _, err := uc.Run(ectx, s, TriggerAutomatico, s.UpdatedBy); err != nil {
			// Uma empresa com problema não pode impedir as demais.
			continue
		}
		executadas++
	}
	return executadas, nil
}

// Run executa o planejamento de uma empresa sob cadeado e corte.
func (uc *AutoRunPlanningUseCase) Run(
	ctx context.Context,
	s AutoRunSettings,
	trigger Trigger,
	actor *uuid.UUID,
) (*RunRecord, error) {
	if s.PlanCode <= 0 {
		return nil, errorsuc.NewValidationError("nenhum plano de planejamento configurado para a execução automática")
	}
	if uc.Pipeline == nil {
		return nil, errorsuc.NewValidationError("o cálculo de planejamento não está disponível nesta instalação")
	}

	ok, err := uc.Repo.TryLock(ctx, s.EnterpriseID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errorsuc.NewConflictError(
			"já existe um cálculo de planejamento em andamento para esta empresa; aguarde o término")
	}
	defer func() { _ = uc.Repo.Unlock(context.WithoutCancel(ctx), s.EnterpriseID) }()

	// O corte é tomado antes de ler qualquer coisa: é ele que define o que este
	// ciclo enxerga e o que fica para o próximo.
	snapshot := uc.now()

	alterados := 0
	if s.LastSnapshotAt != nil {
		if alterados, err = uc.Repo.ChangedItemsSince(ctx, s.EnterpriseID, *s.LastSnapshotAt); err != nil {
			return nil, err
		}
	}

	rec := RunRecord{
		EnterpriseID: s.EnterpriseID,
		PlanCode:     s.PlanCode,
		Trigger:      trigger,
		TriggeredBy:  actor,
		SnapshotAt:   snapshot,
		StartedAt:    snapshot,
		ChangedItems: alterados,
		Status:       "RUNNING",
	}
	id, err := uc.Repo.StartRun(ctx, rec)
	if err != nil {
		return nil, err
	}
	rec.ID = id

	result, err := uc.Pipeline.Execute(ctx, request.RunPlanningPipelineDTO{
		PlanCode:           s.PlanCode,
		InitialOrderNumber: s.InitialOrderNumber,
		GenerateLLC:        s.GenerateLLC,
		StartFrom:          snapshot,
	})
	if err != nil {
		rec.Status = "FAILED"
		rec.Notes = err.Error()
		// A falha da execução é o erro principal; se nem o registro dela gravar,
		// isso vai junto na mensagem (o histórico ficaria "em andamento").
		msg := ""
		if errFim := uc.Repo.FinishRun(context.WithoutCancel(ctx), id, rec); errFim != nil {
			msg += "; o histórico da execução não pôde ser encerrado: " + errFim.Error()
		}
		if errFoto := uc.Repo.SaveSnapshot(context.WithoutCancel(ctx), s.EnterpriseID, snapshot, "FAILED"); errFoto != nil {
			msg += "; a fotografia do plano não pôde ser gravada: " + errFoto.Error()
		}
		return nil, fmt.Errorf("execução do planejamento: %w%s", err, msg)
	}

	viavel := result.Viable
	rec.Status = "SUCCESS"
	rec.MRPOrders = result.MRPOrders
	rec.CRPOverload = result.CRPOverload
	rec.Viable = &viavel
	if len(result.Notes) > 0 {
		rec.Notes = result.Notes[0]
	}
	if err := uc.Repo.FinishRun(ctx, id, rec); err != nil {
		return nil, err
	}
	fim := uc.now()
	rec.FinishedAt = &fim
	// O corte só avança quando o ciclo termina bem: se falhar, a próxima
	// execução reprocessa a mesma janela em vez de pular alterações.
	if err := uc.Repo.SaveSnapshot(ctx, s.EnterpriseID, snapshot, "SUCCESS"); err != nil {
		return nil, err
	}
	return &rec, nil
}

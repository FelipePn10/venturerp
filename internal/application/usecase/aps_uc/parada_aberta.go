package aps_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// paradaAbertaRepo é o cronômetro de parada do chão de fábrica.
type paradaAbertaRepo interface {
	AbrirParadaDeMaquina(context.Context, int64, string, string) (int64, bool, error)
	FecharParadaDeMaquina(context.Context, int64) (int64, float64, error)
	ParadaAbertaDaMaquina(context.Context, int64) (int64, string, string, float64, bool, error)
}

// ParadaEmCurso é o que a tela do operador precisa mostrar.
type ParadaEmCurso struct {
	ID           int64   `json:"id"`
	MachineID    int64   `json:"machine_id"`
	DowntimeType string  `json:"downtime_type"`
	Reason       string  `json:"reason"`
	Minutes      float64 `json:"minutes"`
	Open         bool    `json:"open"`
	// Reprogramacao conta o que a parada empurrou. Sem isto o operador fecha a
	// parada e nada indica que a fila do dia mudou — e a fila mudou.
	Reprogramacao *ResumoReprogramacao `json:"reprogramacao,omitempty"`
}

// ResumoReprogramacao é o efeito da parada sobre a fila daquela máquina.
type ResumoReprogramacao struct {
	OrdensReprogramadas int     `json:"ordens_reprogramadas"`
	OperacoesAgendadas  int     `json:"operacoes_agendadas"`
	OrdensForaDoPrazo   []int64 `json:"ordens_fora_do_prazo,omitempty"`
	Aviso               string  `json:"aviso,omitempty"`
}

func (uc *APSUseCase) paradaRepo() (paradaAbertaRepo, error) {
	repo, ok := uc.repo.(paradaAbertaRepo)
	if !ok {
		return nil, errorsuc.NewValidationError("o registro de parada não está disponível")
	}
	return repo, nil
}

// AbrirParada marca que a máquina parou AGORA. O operador não digita horário:
// parada de chão de fábrica dura minutos e acontece várias vezes por turno —
// exigir data e hora nas duas pontas garante que ninguém registre, e o que não
// é registrado deixa o planejamento achando que a máquina produziu o turno todo.
func (uc *APSUseCase) AbrirParada(ctx context.Context, machineID int64, motivo, descricao string) (ParadaEmCurso, error) {
	repo, err := uc.paradaRepo()
	if err != nil {
		return ParadaEmCurso{}, err
	}
	if machineID <= 0 {
		return ParadaEmCurso{}, errorsuc.NewValidationError("informe a máquina")
	}
	kind := strings.ToUpper(strings.TrimSpace(motivo))
	if kind == "" {
		kind = "UNPLANNED"
	}
	if kind != "PLANNED" && kind != "UNPLANNED" && kind != "MAINTENANCE" {
		return ParadaEmCurso{}, errorsuc.NewValidationError(
			"motivo da parada inválido; use PLANNED (programada), UNPLANNED (quebra) ou MAINTENANCE (manutenção)")
	}
	id, jaExistia, err := repo.AbrirParadaDeMaquina(ctx, machineID, kind, strings.TrimSpace(descricao))
	if err != nil {
		return ParadaEmCurso{}, err
	}
	if id == 0 {
		return ParadaEmCurso{}, errorsuc.NewNotFoundError("máquina não encontrada nesta empresa")
	}
	if jaExistia {
		// Tocar duas vezes no botão é normal num terminal de chão de fábrica.
		// Devolver a parada que já está aberta é mais útil que recusar.
		return uc.ParadaEmAberto(ctx, machineID)
	}
	return ParadaEmCurso{ID: id, MachineID: machineID, DowntimeType: kind, Reason: strings.TrimSpace(descricao), Open: true}, nil
}

// FecharParada encerra a parada em curso e devolve quanto tempo durou.
func (uc *APSUseCase) FecharParada(ctx context.Context, machineID int64) (ParadaEmCurso, error) {
	repo, err := uc.paradaRepo()
	if err != nil {
		return ParadaEmCurso{}, err
	}
	if machineID <= 0 {
		return ParadaEmCurso{}, errorsuc.NewValidationError("informe a máquina")
	}
	id, minutos, err := repo.FecharParadaDeMaquina(ctx, machineID)
	if err != nil || id == 0 {
		return ParadaEmCurso{}, errorsuc.NewNotFoundError("esta máquina não tem parada em aberto")
	}
	out := ParadaEmCurso{ID: id, MachineID: machineID, Minutes: minutos, Open: false}
	out.Reprogramacao = uc.reprogramarAposParada(ctx, machineID)
	return out, nil
}

// reprogramarAposParada reorganiza a fila daquela máquina a partir de agora.
//
// O tempo perdido não volta: as ordens que ainda cabem no que sobrou do dia
// ficam, e as que não cabem empurram para os próximos dias, entrando entre o que
// já estava programado lá conforme a data de entrega. Sem isto, fechar a parada
// não mexeria em nada e a fila do dia seguiria prometendo o que a máquina já não
// consegue fazer — o planejamento só enxergaria a perda no próximo MRP.
//
// Só a máquina afetada é reprogramada. Resequenciar a fábrica inteira a cada
// parada de cinco minutos moveria ordens que nada têm a ver com o problema.
//
// A falha aqui NÃO derruba o encerramento da parada: o registro do que
// aconteceu no chão de fábrica vale por si, e é o dado que não pode se perder.
func (uc *APSUseCase) reprogramarAposParada(ctx context.Context, machineID int64) *ResumoReprogramacao {
	resumo, err := uc.SequenceOrders(ctx, request.SequenceOrdersDTO{
		MachineIDs: []int64{machineID},
		StartFrom:  time.Now(),
		Direction:  "FORWARD",
	})
	if err != nil {
		return &ResumoReprogramacao{Aviso: "a parada foi registrada, mas a fila desta máquina não pôde ser reprogramada agora: " + err.Error()}
	}
	return &ResumoReprogramacao{
		OrdensReprogramadas: resumo.OrdersProcessed,
		OperacoesAgendadas:  resumo.ScheduledOperations,
		OrdensForaDoPrazo:   resumo.LateOrders,
	}
}

// ParadaEmAberto responde se a máquina está parada neste instante e há quanto tempo.
func (uc *APSUseCase) ParadaEmAberto(ctx context.Context, machineID int64) (ParadaEmCurso, error) {
	repo, err := uc.paradaRepo()
	if err != nil {
		return ParadaEmCurso{}, err
	}
	id, motivo, descricao, minutos, aberta, err := repo.ParadaAbertaDaMaquina(ctx, machineID)
	if err != nil {
		return ParadaEmCurso{}, err
	}
	return ParadaEmCurso{ID: id, MachineID: machineID, DowntimeType: motivo, Reason: descricao, Minutes: minutos, Open: aberta}, nil
}

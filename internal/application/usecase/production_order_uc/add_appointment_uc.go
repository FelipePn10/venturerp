package production_order_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/employeeresolution"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type AddAppointmentUseCase struct {
	// CustoAutomatico recalcula o custo real da OF depois do movimento. Opcional.
	CustoAutomatico *SettleProductionCostUseCase
	Repo            repository.ProductionOrderRepository
	Auth            ports.AuthService
	// StockRepo: com ele, o apontamento baixa os materiais automáticos da OF
	// (OUT) na proporção do produzido. Sem ele (testes), só grava o apontamento.
	StockRepo stockrepo.StockRepository
	// Employees traduz o CÓDIGO do funcionário que a tela manda para a chave
	// interna que `production_appointments` exige. Opcional: sem ele o valor
	// segue como está (comportamento anterior).
	Employees employeeresolution.Finder
	// Transacao grava o apontamento e as baixas dos componentes juntos: ou os
	// dois, ou nenhum. Sem ela (testes), roda direto nos repositórios.
	Transacao EmTransacaoApontamento
}

// ApontamentoRepos são os repositórios ligados à transação do apontamento.
type ApontamentoRepos struct {
	Ordens  repository.ProductionOrderRepository
	Estoque stockrepo.StockRepository
}

// EmTransacaoApontamento roda fn numa transação; erro devolvido desfaz tudo.
type EmTransacaoApontamento func(ctx context.Context, fn func(ApontamentoRepos) error) error

// Execute grava o movimento e, dando certo, atualiza o custo real da OF
// (CustoAutomatico; ver SettleProductionCostUseCase.Atualizar).
func (uc *AddAppointmentUseCase) Execute(ctx context.Context, dto request.AddAppointmentDTO) (*entity.ProductionAppointment, error) {
	out, err := uc.executar(ctx, dto)
	if err == nil {
		uc.CustoAutomatico.Atualizar(ctx, dto.ProductionOrderID)
	}
	return out, err
}

func (uc *AddAppointmentUseCase) executar(
	ctx context.Context,
	dto request.AddAppointmentDTO,
) (*entity.ProductionAppointment, error) {
	if !uc.Auth.CanReportProduction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	dto.CreatedBy = actor
	if dto.ProductionOrderID <= 0 {
		return nil, errorsuc.NewValidationError("ordem de fabricacao obrigatoria")
	}
	if dto.EmployeeID == nil || *dto.EmployeeID <= 0 {
		return nil, errorsuc.NewValidationError("informe o operador")
	}
	operador, err := employeeresolution.ResolveID(ctx, uc.Employees, *dto.EmployeeID)
	if err != nil {
		return nil, err
	}
	dto.EmployeeID = &operador
	if err := conferirHorasDoApontamento(dto.StartTime, dto.EndTime); err != nil {
		return nil, err
	}
	if dto.ProducedQty.IsNegative() || dto.ScrappedQty.IsNegative() || !dto.ProducedQty.Add(dto.ScrappedQty).IsPositive() {
		return nil, errorsuc.NewValidationError("quantidades boas/refugadas devem ser nao negativas e possuir total maior que zero")
	}
	if dto.ScrappedQty.IsPositive() && (dto.ScrapReason == nil || strings.TrimSpace(*dto.ScrapReason) == "") {
		return nil, errorsuc.NewValidationError("informe o motivo do refugo")
	}

	// Data vazia é hoje; data mal digitada é erro (antes virava "hoje" em silêncio).
	if msg := datetime.PrimeiraInvalida(datetime.Campo("data do apontamento", &dto.AppointmentDate)); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	appointmentDate := datetime.ParseDateOrDefault(dto.AppointmentDate, time.Now())

	appointment := &entity.ProductionAppointment{
		ProductionOrderID: dto.ProductionOrderID,
		MachineID:         dto.MachineID,
		EmployeeID:        dto.EmployeeID,
		AppointmentDate:   appointmentDate,
		StartTime:         dto.StartTime,
		EndTime:           dto.EndTime,
		ProducedQty:       dto.ProducedQty,
		ScrappedQty:       dto.ScrappedQty,
		ScrapReason:       dto.ScrapReason,
		Notes:             dto.Notes,
		CreatedBy:         dto.CreatedBy,
	}

	// Os materiais são lidos antes de gravar qualquer coisa: se a leitura
	// falhar, nada é gravado.
	var baixas []*stockentity.StockMovement
	if uc.shouldBackflush(dto) {
		if baixas, err = uc.planejarBaixas(ctx, dto); err != nil {
			return nil, err
		}
	}

	emTransacao := uc.Transacao
	if emTransacao == nil {
		emTransacao = func(ctx context.Context, fn func(ApontamentoRepos) error) error {
			return fn(ApontamentoRepos{Ordens: uc.Repo, Estoque: uc.StockRepo})
		}
	}
	var saved *entity.ProductionAppointment
	err = emTransacao(ctx, func(r ApontamentoRepos) error {
		var err error
		if saved, err = r.Ordens.AddAppointment(ctx, appointment); err != nil {
			return err
		}
		for _, mov := range baixas {
			if _, err := r.Estoque.CreateMovement(ctx, mov); err != nil {
				return fmt.Errorf("baixando o material %d da OF: %w", mov.ItemCode, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (uc *AddAppointmentUseCase) shouldBackflush(dto request.AddAppointmentDTO) bool {
	return uc.StockRepo != nil && dto.ProducedQty.IsPositive()
}

// planejarBaixas calcula as saídas automáticas dos materiais da OF para o
// bom acumulado depois deste apontamento (ver baixa_automatica_da_of.go).
// Falha de leitura é erro: nada é gravado.
func (uc *AddAppointmentUseCase) planejarBaixas(ctx context.Context, dto request.AddAppointmentDTO) ([]*stockentity.StockMovement, error) {
	order, err := uc.Repo.GetByCode(ctx, dto.ProductionOrderID)
	if err != nil {
		return nil, fmt.Errorf("lendo a OF para a baixa dos materiais: %w", err)
	}
	materiais, err := lerMateriaisAutomaticos(ctx, uc.Repo, order.ID)
	if err != nil || len(materiais) == 0 {
		return nil, err
	}
	baixado := map[int64]decimal.Decimal{}
	apontadoAntes := decimal.Zero
	if b, ok := uc.Repo.(baixasDaOrdemRepository); ok {
		if baixado, _, apontadoAntes, err = b.BaixasDaOrdem(ctx, order.ID); err != nil {
			return nil, err
		}
	}
	fracao := fracaoProduzida(apontadoAntes.Add(dto.ProducedQty), order.PlannedQty)
	return baixasAutomaticas(materiais, baixado, fracao, dto.BackflushWarehouseID, stockentity.MovementTypeOut, nil, order.ID, dto.CreatedBy), nil
}

// conferirHorasDoApontamento valida a hora de início e de fim (HH:MM ou
// HH:MM:SS). Hora em outro formato chegava ao banco e voltava como "erro
// interno do servidor".
func conferirHorasDoApontamento(inicio, fim *string) error {
	ler := func(rotulo string, v *string) (*time.Time, error) {
		if v == nil || strings.TrimSpace(*v) == "" {
			return nil, nil
		}
		texto := strings.TrimSpace(*v)
		for _, formato := range []string{"15:04", "15:04:05"} {
			if t, err := time.Parse(formato, texto); err == nil {
				return &t, nil
			}
		}
		return nil, errorsuc.NewValidationError("a hora de " + rotulo + " deve estar no formato HH:MM (ex.: 08:30)")
	}
	if _, err := ler("início", inicio); err != nil {
		return err
	}
	// Fim menor que o início é válido: turno que vira a noite (22:00 → 06:00).
	_, err := ler("fim", fim)
	return err
}

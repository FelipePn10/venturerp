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
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structurerepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type AddAppointmentUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// StructureRepo and StockRepo are optional. When both are set and the
	// appointment carries a backflush warehouse, the BOM components are consumed
	// automatically (OUT) in proportion to the produced quantity.
	StructureRepo structurerepo.ItemStructureRepository
	StockRepo     stockrepo.StockRepository
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

func (uc *AddAppointmentUseCase) Execute(
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
	if dto.ProducedQty < 0 || dto.ScrappedQty < 0 || dto.ProducedQty+dto.ScrappedQty <= 0 {
		return nil, errorsuc.NewValidationError("quantidades boas/refugadas devem ser nao negativas e possuir total maior que zero")
	}
	if dto.ScrappedQty > 0 && (dto.ScrapReason == nil || strings.TrimSpace(*dto.ScrapReason) == "") {
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

	// A estrutura é lida antes de gravar qualquer coisa: se a leitura falhar,
	// nada é gravado.
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
				return fmt.Errorf("baixando o componente %d (backflush): %w", mov.ItemCode, err)
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
	return uc.StructureRepo != nil && uc.StockRepo != nil &&
		dto.BackflushWarehouseID != nil && dto.ProducedQty > 0
}

// planejarBaixas calcula as saídas dos componentes (backflush) para a
// quantidade produzida. Falha de leitura é erro — antes o apontamento era
// gravado e os componentes não saíam do estoque, sem aviso.
func (uc *AddAppointmentUseCase) planejarBaixas(ctx context.Context, dto request.AddAppointmentDTO) ([]*stockentity.StockMovement, error) {
	order, err := uc.Repo.GetByCode(ctx, dto.ProductionOrderID)
	if err != nil {
		return nil, fmt.Errorf("lendo a OF para a baixa dos componentes: %w", err)
	}

	var children []*structureChild
	var rawErr error
	if order.Mask != "" {
		raw, e := uc.StructureRepo.GetDirectChildrenForMask(ctx, order.ItemCode, order.Mask)
		rawErr = e
		children = structureChildrenForBackflush(raw)
	} else {
		raw, e := uc.StructureRepo.GetAllDirectChildren(ctx, order.ItemCode)
		rawErr = e
		children = structureChildrenForBackflush(raw)
	}
	if rawErr != nil {
		return nil, fmt.Errorf("lendo a estrutura do item para a baixa dos componentes: %w", rawErr)
	}
	var baixas []*stockentity.StockMovement

	refType := stockentity.ReferenceTypeProductionOrder
	refCode := dto.ProductionOrderID
	for _, c := range children {
		// A fórmula de perdas é a mesma do MRP (structentity.QuantidadeComPerda);
		// aqui usávamos (1 + perda/100) e o consumo divergia do planejado.
		base := dto.ProducedQty
		if c.fixed {
			base = 1
		}
		consumed := structentity.QuantidadeComPerda(base*c.qty, c.loss, structentity.FormulaPerdaPadrao)
		if consumed <= 0 {
			continue
		}
		mov := &stockentity.StockMovement{
			ItemCode:      c.code,
			WarehouseID:   *dto.BackflushWarehouseID,
			MovementType:  stockentity.MovementTypeOut,
			Quantity:      decimal.NewFromFloat(consumed),
			ReferenceType: &refType,
			ReferenceCode: &refCode,
			CreatedBy:     dto.CreatedBy,
		}
		baixas = append(baixas, mov)
	}
	return baixas, nil
}

func structureChildrenForBackflush(raw []*structentity.ItemStructure) []*structureChild {
	selected := structentity.SelectPrimarySubstituteComponents(raw)
	children := make([]*structureChild, 0, len(selected))
	for _, c := range selected {
		if c.IsCoproduct {
			continue
		}
		children = append(children, &structureChild{
			code: c.ChildCode,
			// Na unidade de ESTOQUE: é o que vai baixar do saldo.
			qty:   c.QuantidadeNaUnidadeDeEstoque(),
			loss:  c.LossPercentage,
			fixed: c.IsFixedQty,
		})
	}
	return children
}

type structureChild struct {
	code  int64
	qty   float64
	loss  float64
	fixed bool
}

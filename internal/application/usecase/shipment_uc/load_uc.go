package shipment_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/google/uuid"
)

type CreateLoadInput = repository.CreateLoadInput

func (uc *ShipmentUseCase) CreateLoad(ctx context.Context, in CreateLoadInput) (*response.ShipmentLoadResponse, error) {
	if in.CreatedBy == uuid.Nil {
		return nil, fmt.Errorf("usuário responsável pela carga é obrigatório")
	}
	load, err := uc.Repo.CreateLoad(ctx, in)
	if err != nil {
		return nil, err
	}
	return toShipmentLoadResponse(load), nil
}

func (uc *ShipmentUseCase) GetLoad(ctx context.Context, code int64) (*response.ShipmentLoadResponse, error) {
	load, err := uc.Repo.GetLoadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentLoadResponse(load), nil
}

func (uc *ShipmentUseCase) ListLoads(ctx context.Context, f repository.LoadFilter) ([]*response.ShipmentLoadResponse, error) {
	loads, err := uc.Repo.ListLoads(ctx, f)
	if err != nil {
		return nil, err
	}
	return toShipmentLoadResponses(loads), nil
}

func (uc *ShipmentUseCase) AddShipmentToLoad(ctx context.Context, loadCode, shipmentCode int64, sequence int) (*response.ShipmentLoadShipmentResponse, error) {
	load, err := uc.Repo.GetLoadByCode(ctx, loadCode)
	if err != nil {
		return nil, err
	}
	if load.Status == entity.LoadStatusShipped || load.Status == entity.LoadStatusCancelled {
		return nil, fmt.Errorf("carga %d não aceita romaneios no status %s", loadCode, load.Status)
	}
	ship, err := uc.Repo.GetByCode(ctx, shipmentCode)
	if err != nil {
		return nil, err
	}
	if ship.Status == entity.ShipmentStatusShipped || ship.Status == entity.ShipmentStatusCancelled {
		return nil, fmt.Errorf("romaneio %d não pode ser incluído na carga no status %s", shipmentCode, ship.Status)
	}
	if sequence <= 0 {
		sequence = load.TotalShipments + 1
	}
	linked, err := uc.Repo.AddShipmentToLoad(ctx, loadCode, shipmentCode, sequence)
	if err != nil {
		return nil, err
	}
	_ = uc.Repo.RecalcLoadTotals(ctx, loadCode)
	return toShipmentLoadShipmentResponse(linked), nil
}

func (uc *ShipmentUseCase) RemoveShipmentFromLoad(ctx context.Context, loadCode, shipmentCode int64) error {
	load, err := uc.Repo.GetLoadByCode(ctx, loadCode)
	if err != nil {
		return err
	}
	if load.Status != entity.LoadStatusPlanned {
		return fmt.Errorf("só é possível remover romaneio de carga planejada")
	}
	if err := uc.Repo.RemoveShipmentFromLoad(ctx, loadCode, shipmentCode); err != nil {
		return err
	}
	return uc.Repo.RecalcLoadTotals(ctx, loadCode)
}

func (uc *ShipmentUseCase) AddFiscalNoteToLoad(ctx context.Context, in repository.AddFiscalNoteToLoadInput) (*response.ShipmentLoadFiscalNoteResponse, error) {
	load, err := uc.Repo.GetLoadByCode(ctx, in.LoadCode)
	if err != nil {
		return nil, err
	}
	if load.Status == entity.LoadStatusShipped || load.Status == entity.LoadStatusCancelled {
		return nil, fmt.Errorf("carga %d não aceita notas no status %s", in.LoadCode, load.Status)
	}
	if in.FiscalExitID <= 0 {
		return nil, fmt.Errorf("fiscal_exit_id é obrigatório")
	}
	if in.Sequence <= 0 {
		in.Sequence = load.TotalFiscalNotes + 1
	}
	note, err := uc.Repo.AddFiscalNoteToLoad(ctx, in)
	if err != nil {
		return nil, err
	}
	_ = uc.Repo.RecalcLoadTotals(ctx, in.LoadCode)
	return toShipmentLoadFiscalNoteResponse(note), nil
}

func (uc *ShipmentUseCase) TransitionLoad(ctx context.Context, code int64, next entity.LoadStatus, actor uuid.UUID, note string) error {
	load, err := uc.Repo.GetLoadByCode(ctx, code)
	if err != nil {
		return err
	}
	if !load.Status.CanTransitionTo(next) {
		return fmt.Errorf("transição inválida: carga %d está %s e não pode ir para %s", code, load.Status, next)
	}
	if next == entity.LoadStatusReleased && load.TotalShipments == 0 && len(load.Shipments) == 0 {
		return fmt.Errorf("carga %d não possui romaneios para liberar", code)
	}
	return uc.Repo.UpdateLoadStatus(ctx, code, next, &actor, note)
}

func (uc *ShipmentUseCase) CreateDeliveryInstruction(ctx context.Context, d *entity.DeliveryInstruction) (*response.DeliveryInstructionResponse, error) {
	if d.Title == "" || d.Instruction == "" {
		return nil, fmt.Errorf("título e orientação são obrigatórios")
	}
	if d.Priority == 0 {
		d.Priority = 5
	}
	d.Active = true
	created, err := uc.Repo.CreateDeliveryInstruction(ctx, d)
	if err != nil {
		return nil, err
	}
	return toDeliveryInstructionResponse(created), nil
}

func (uc *ShipmentUseCase) ListDeliveryInstructions(ctx context.Context, loadCode *int64, activeOnly bool) ([]*response.DeliveryInstructionResponse, error) {
	list, err := uc.Repo.ListDeliveryInstructions(ctx, loadCode, activeOnly)
	if err != nil {
		return nil, err
	}
	return toDeliveryInstructionResponses(list), nil
}

func (uc *ShipmentUseCase) CreateDispatchBox(ctx context.Context, b *entity.DispatchBox) (*response.DispatchBoxResponse, error) {
	if b.Code == "" {
		return nil, fmt.Errorf("código do box é obrigatório")
	}
	b.Active = true
	created, err := uc.Repo.CreateDispatchBox(ctx, b)
	if err != nil {
		return nil, err
	}
	return toDispatchBoxResponse(created), nil
}

func (uc *ShipmentUseCase) ListDispatchBoxes(ctx context.Context, activeOnly bool) ([]*response.DispatchBoxResponse, error) {
	list, err := uc.Repo.ListDispatchBoxes(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	return toDispatchBoxResponses(list), nil
}

func (uc *ShipmentUseCase) AssignBoxToLoad(ctx context.Context, loadCode int64, boxCode string, actor uuid.UUID) error {
	if boxCode == "" {
		return fmt.Errorf("box de expedição é obrigatório")
	}
	return uc.Repo.AssignBoxToLoad(ctx, loadCode, boxCode, &actor)
}

func (uc *ShipmentUseCase) LoadMonitor(ctx context.Context, f repository.LoadFilter) ([]*response.LoadMonitorResponse, error) {
	rows, err := uc.Repo.LoadMonitor(ctx, f)
	if err != nil {
		return nil, err
	}
	return toLoadMonitorResponses(rows), nil
}

func (uc *ShipmentUseCase) SeparationMonitor(ctx context.Context, f repository.LoadFilter) ([]*response.SeparationMonitorResponse, error) {
	rows, err := uc.Repo.SeparationMonitor(ctx, f)
	if err != nil {
		return nil, err
	}
	return toSeparationMonitorResponses(rows), nil
}

func (uc *ShipmentUseCase) LogisticPanel(ctx context.Context) (*response.LogisticPanelResponse, error) {
	summary, err := uc.Repo.LogisticPanel(ctx)
	if err != nil {
		return nil, err
	}
	return toLogisticPanelResponse(summary), nil
}

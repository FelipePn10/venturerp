// Package tool_sheet_uc implements the "Ficha de Produção da Ferramenta" (tool
// production sheet): it binds physical tool serials to the operations of a
// production order. Read-heavy joins across order → operations → tools → serials
// are done directly against sqlc.Queries (mirroring OrderOperationsUseCase).
package tool_sheet_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	toolentity "github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// ToolSheetUseCase manages the tool production sheet.
type ToolSheetUseCase struct {
	Q *sqlc.Queries
}

func New(q *sqlc.Queries) *ToolSheetUseCase {
	return &ToolSheetUseCase{Q: q}
}

// mapOrderType converts a planned-order type into the shop-floor order label.
// PRODUCTION and manual orders are "OF"; OUTSOURCING is "OFC" (excluded from the
// order LOV); anything else is passed through.
func mapOrderType(raw string) string {
	switch raw {
	case "", "PRODUCTION":
		return "OF"
	case "OUTSOURCING":
		return "OFC"
	default:
		return raw
	}
}

func mapSheetOrder(o sqlc.DBToolSheetOrder) response.ToolProductionSheetOrderResponse {
	raw := pgutil.FromPgText(o.OrderType)
	return response.ToolProductionSheetOrderResponse{
		OrderID:     o.ID,
		OrderNumber: o.OrderNumber,
		Type:        mapOrderType(raw),
		TypeRaw:     raw,
		StartDate:   pgutil.FromPgDateToPtr(o.StartDate),
		EndDate:     pgutil.FromPgDateToPtr(o.EndDate),
		Quantity:    pgutil.FromPgNumericToFloat64(o.PlannedQty),
		ItemCode:    o.ItemCode,
		ItemName:    pgutil.FromPgText(o.ItemName),
		Configured:  o.Mask,
		Status:      o.Status,
	}
}

// enterprise resolves the "Empresa" being accessed. The deployment is
// single-tenant, so the first registered enterprise is used.
func (uc *ToolSheetUseCase) enterprise(ctx context.Context) (int64, string) {
	ents, err := uc.Q.ListEnterprises(ctx)
	if err != nil || len(ents) == 0 {
		return 0, ""
	}
	return ents[0].ID, ents[0].Name
}

// ListOrders returns the production orders eligible for the sheet (OFC excluded).
func (uc *ToolSheetUseCase) ListOrders(ctx context.Context, search string) ([]response.ToolProductionSheetOrderResponse, error) {
	rows, err := uc.Q.ListEligibleSheetOrders(ctx, search)
	if err != nil {
		return nil, fmt.Errorf("listing eligible orders: %w", err)
	}
	entID, entName := uc.enterprise(ctx)
	out := make([]response.ToolProductionSheetOrderResponse, 0, len(rows))
	for _, r := range rows {
		h := mapSheetOrder(r)
		h.EnterpriseID, h.EnterpriseName = entID, entName
		out = append(out, h)
	}
	return out, nil
}

// GetSheet builds the full sheet for a production order: header + operations,
// each operation listing its tools with the assigned serial and the serials
// available for selection.
func (uc *ToolSheetUseCase) GetSheet(ctx context.Context, orderID int64) (*response.ToolProductionSheetResponse, error) {
	order, err := uc.Q.GetToolSheetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order %d not found: %w", orderID, err)
	}
	header := mapSheetOrder(order)
	header.EnterpriseID, header.EnterpriseName = uc.enterprise(ctx)

	rows, err := uc.Q.ListSheetOperationTools(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("listing operations: %w", err)
	}

	// available-serials cache keyed by tool id (each tool queried once)
	serialCache := map[int64][]response.ToolSerialResponse{}
	availableSerials := func(toolID int64) []response.ToolSerialResponse {
		if v, ok := serialCache[toolID]; ok {
			return v
		}
		list := uc.serialsForTool(ctx, toolID)
		serialCache[toolID] = list
		return list
	}

	ops := make([]response.SheetOperationResponse, 0)
	index := map[int64]int{} // operation_id → position in ops
	for _, r := range rows {
		pos, ok := index[r.OperationID]
		if !ok {
			op := response.SheetOperationResponse{
				OperationID:   r.OperationID,
				Sequence:      int(r.Sequence),
				OperationCode: pgutil.FromPgInt8Ptr(r.OperationCode),
				OperationName: r.OperationName,
				OperationDesc: pgutil.FromPgText(r.OperationDesc),
				ResourceCode:  pgutil.FromPgInt8Ptr(r.ResourceCode),
				ResourceName:  pgutil.FromPgText(r.ResourceName),
				Status:        r.OperationStatus,
				Tools:         []response.SheetOperationToolResponse{},
			}
			ops = append(ops, op)
			pos = len(ops) - 1
			index[r.OperationID] = pos
		}
		// A row with no tool (operation without route tools) contributes only the
		// operation header.
		if !r.ToolID.Valid {
			continue
		}
		tool := response.SheetOperationToolResponse{
			ToolID:           r.ToolID.Int64,
			ToolCode:         pgInt8OrZero(r.ToolCode),
			ToolName:         pgutil.FromPgText(r.ToolName),
			QtyRequired:      pgutil.FromPgNumericToFloat64(r.QtyRequired),
			AvailableSerials: availableSerials(r.ToolID.Int64),
		}
		if r.AssignedSerialID.Valid {
			id := r.AssignedSerialID.Int64
			tool.AssignedSerialID = &id
			tool.AssignedSerialNumber = pgutil.FromPgText(r.AssignedSerialNumber)
			tool.AssignedSerialStatus = pgutil.FromPgText(r.AssignedSerialStatus)
			tool.CanSubstitute = true
		}
		ops[pos].Tools = append(ops[pos].Tools, tool)
	}

	return &response.ToolProductionSheetResponse{Header: header, Operations: ops}, nil
}

func (uc *ToolSheetUseCase) serialsForTool(ctx context.Context, toolID int64) []response.ToolSerialResponse {
	rows, err := uc.Q.ListToolSerials(ctx, sqlc.ListToolSerialsParams{ToolID: toolID, OnlyActive: true})
	if err != nil {
		return []response.ToolSerialResponse{}
	}
	out := make([]response.ToolSerialResponse, 0, len(rows))
	for _, s := range rows {
		out = append(out, response.ToolSerialResponse{
			ID:           s.ID,
			ToolID:       s.ToolID,
			SerialNumber: s.SerialNumber,
			Status:       s.Status,
			LifeUsed:     pgutil.FromPgNumericToFloat64(s.LifeUsed),
			Location:     pgutil.FromPgText(s.Location),
			Notes:        pgutil.FromPgText(s.Notes),
			IsActive:     s.IsActive,
			Available:    s.IsActive && s.Status == toolentity.SerialActive,
		})
	}
	return out
}

// Assign binds (or re-binds) a serial to an operation/tool.
func (uc *ToolSheetUseCase) Assign(ctx context.Context, dto request.AssignToolSerialDTO) (*response.SheetOperationToolResponse, error) {
	if err := uc.validateAssignment(ctx, dto.OperationID, dto.ToolID, dto.SerialID); err != nil {
		return nil, err
	}
	if _, err := uc.Q.AssignToolSerial(ctx, sqlc.AssignToolSerialParams{
		OperationID:  dto.OperationID,
		ToolID:       dto.ToolID,
		ToolSerialID: dto.SerialID,
		AssignedBy:   pgutil.ToPgUUID(dto.AssignedBy),
	}); err != nil {
		return nil, fmt.Errorf("assigning serial: %w", err)
	}
	return uc.operationToolView(ctx, dto.OperationID, dto.ToolID)
}

// Substitute replaces the serial already bound to an operation/tool and records
// the change in the audit trail. It requires an existing binding — the
// "Substituir" action is only functional when a serial is already assigned.
func (uc *ToolSheetUseCase) Substitute(ctx context.Context, dto request.SubstituteToolSerialDTO) (*response.SheetOperationToolResponse, error) {
	current, err := uc.Q.GetOperationToolSerial(ctx, sqlc.GetOperationToolSerialParams{
		OperationID: dto.OperationID, ToolID: dto.ToolID,
	})
	if err != nil {
		return nil, fmt.Errorf("operação não possui série vinculada para substituir")
	}
	if err := uc.validateAssignment(ctx, dto.OperationID, dto.ToolID, dto.NewSerialID); err != nil {
		return nil, err
	}
	if current.ToolSerialID == dto.NewSerialID {
		return nil, fmt.Errorf("a nova série é igual à série atual")
	}
	if _, err := uc.Q.AssignToolSerial(ctx, sqlc.AssignToolSerialParams{
		OperationID:  dto.OperationID,
		ToolID:       dto.ToolID,
		ToolSerialID: dto.NewSerialID,
		AssignedBy:   pgutil.ToPgUUID(dto.SubstitutedBy),
	}); err != nil {
		return nil, fmt.Errorf("substituting serial: %w", err)
	}
	oldID := current.ToolSerialID
	if _, err := uc.Q.RecordToolSerialSubstitution(ctx, sqlc.RecordToolSerialSubstitutionParams{
		OperationID:   dto.OperationID,
		ToolID:        dto.ToolID,
		OldSerialID:   pgutil.ToPgInt8Ptr(&oldID),
		NewSerialID:   dto.NewSerialID,
		Reason:        pgutil.ToPgTextFromString(dto.Reason),
		SubstitutedBy: pgutil.ToPgUUID(dto.SubstitutedBy),
	}); err != nil {
		return nil, fmt.Errorf("recording substitution: %w", err)
	}
	return uc.operationToolView(ctx, dto.OperationID, dto.ToolID)
}

// ListSubstitutions returns the substitution history for an operation/tool.
func (uc *ToolSheetUseCase) ListSubstitutions(ctx context.Context, operationID, toolID int64) ([]response.ToolSerialSubstitutionResponse, error) {
	rows, err := uc.Q.ListToolSerialSubstitutions(ctx, sqlc.ListToolSerialSubstitutionsParams{
		OperationID: operationID, ToolID: toolID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing substitutions: %w", err)
	}
	out := make([]response.ToolSerialSubstitutionResponse, 0, len(rows))
	for _, s := range rows {
		out = append(out, response.ToolSerialSubstitutionResponse{
			ID:              s.ID,
			OperationID:     s.OperationID,
			ToolID:          s.ToolID,
			ToolCode:        s.ToolCode,
			ToolName:        s.ToolName,
			OldSerialID:     pgutil.FromPgInt8Ptr(s.OldSerialID),
			OldSerialNumber: pgutil.FromPgText(s.OldSerialNumber),
			NewSerialID:     s.NewSerialID,
			NewSerialNumber: s.NewSerialNumber,
			Reason:          pgutil.FromPgText(s.Reason),
			SubstitutedAt:   pgutil.FromPgTimestamptz(s.SubstitutedAt),
		})
	}
	return out, nil
}

// validateAssignment guards that the operation exists, the serial exists, belongs
// to the tool, and is available to run production.
func (uc *ToolSheetUseCase) validateAssignment(ctx context.Context, operationID, toolID, serialID int64) error {
	if operationID <= 0 || toolID <= 0 || serialID <= 0 {
		return fmt.Errorf("operation_id, tool_id and serial_id are required")
	}
	if _, err := uc.Q.GetProductionOrderOperation(ctx, operationID); err != nil {
		return fmt.Errorf("operação %d não encontrada", operationID)
	}
	serial, err := uc.Q.GetToolSerial(ctx, serialID)
	if err != nil {
		return fmt.Errorf("série %d não encontrada", serialID)
	}
	if serial.ToolID != toolID {
		return fmt.Errorf("a série %d não pertence à ferramenta %d", serialID, toolID)
	}
	if !serial.IsActive || serial.Status != toolentity.SerialActive {
		return fmt.Errorf("a série %s não está disponível (status %s)", serial.SerialNumber, serial.Status)
	}
	return nil
}

// operationToolView returns the single operation/tool row (with assigned serial
// and available serials) after an assign/substitute, so the caller can refresh
// the affected line without reloading the whole sheet.
func (uc *ToolSheetUseCase) operationToolView(ctx context.Context, operationID, toolID int64) (*response.SheetOperationToolResponse, error) {
	binding, err := uc.Q.GetOperationToolSerial(ctx, sqlc.GetOperationToolSerialParams{
		OperationID: operationID, ToolID: toolID,
	})
	if err != nil {
		return nil, fmt.Errorf("reading binding: %w", err)
	}
	serial, err := uc.Q.GetToolSerial(ctx, binding.ToolSerialID)
	if err != nil {
		return nil, fmt.Errorf("reading serial: %w", err)
	}
	view := &response.SheetOperationToolResponse{
		ToolID:               toolID,
		AssignedSerialID:     &binding.ToolSerialID,
		AssignedSerialNumber: serial.SerialNumber,
		AssignedSerialStatus: serial.Status,
		CanSubstitute:        true,
		AvailableSerials:     uc.serialsForTool(ctx, toolID),
	}
	// tool code/name for display
	if t, err := uc.Q.GetTool(ctx, toolID); err == nil {
		view.ToolCode = t.Code
		view.ToolName = t.Name
	}
	return view, nil
}

func pgInt8OrZero(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

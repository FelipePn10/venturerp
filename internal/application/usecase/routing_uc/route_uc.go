package routing_uc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"github.com/google/uuid"
)

type RouteUseCase struct {
	repo  repository.RoutingRepository
	items any
	auth  ports.AuthService
}

func NewRouteUseCase(repo repository.RoutingRepository, items ...any) *RouteUseCase {
	uc := &RouteUseCase{repo: repo}
	for _, dep := range items {
		if auth, ok := dep.(ports.AuthService); ok {
			uc.auth = auth
		} else {
			uc.items = dep
		}
	}
	return uc
}

func (uc *RouteUseCase) Create(ctx context.Context, dto request.CreateRouteDTO) (*response.ManufacturingRouteResponse, error) {
	itemCode, err := uc.resolveItemCode(ctx, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	var actor uuid.UUID
	if uc.auth != nil {
		actor, err = uc.auth.UserID(ctx)
		if err != nil {
			return nil, errorsuc.ErrUnauthorized
		}
	}
	alt := dto.Alternative
	if alt <= 0 {
		alt = 1
	}
	code, err := uc.repo.NextRouteCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating route code: %w", err)
	}

	rt, err := entity.NewManufacturingRoute(code, itemCode, dto.Mask, alt, dto.Description, dto.IsStandard, dto.ValidFrom, dto.ValidTo, actor)
	if err != nil {
		return nil, err
	}

	created, err := uc.repo.CreateRoute(ctx, rt)
	if err != nil {
		return nil, err
	}
	return toRouteResponse(created), nil
}

func (uc *RouteUseCase) Update(ctx context.Context, dto request.UpdateRouteDTO) (*response.ManufacturingRouteResponse, error) {
	rt, err := uc.repo.GetRouteByID(ctx, dto.ID)
	if err != nil {
		return nil, fmt.Errorf("roteiro não encontrado: %w", err)
	}
	rt.Description = dto.Description
	rt.Situation = entity.RouteSituation(dto.Situation)
	rt.IsStandard = dto.IsStandard
	rt.ValidFrom = dto.ValidFrom
	rt.ValidTo = dto.ValidTo

	updated, err := uc.repo.UpdateRoute(ctx, rt)
	if err != nil {
		return nil, err
	}
	return toRouteResponse(updated), nil
}

func (uc *RouteUseCase) GetDetail(ctx context.Context, id int64) (*response.RouteDetailResponse, error) {
	rt, err := uc.repo.GetRouteByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("roteiro não encontrado: %w", err)
	}

	ops, err := uc.repo.GetRouteOperations(ctx, id)
	if err != nil {
		return nil, err
	}

	edges, err := uc.repo.GetNetworkEdges(ctx, id)
	if err != nil {
		return nil, err
	}

	opResps := make([]response.RouteOperationResponse, 0, len(ops))
	for _, op := range ops {
		opResps = append(opResps, toRouteOpResponse(op))
	}

	edgeResps := make([]response.NetworkEdgeResponse, 0, len(edges))
	for _, e := range edges {
		edgeResps = append(edgeResps, response.NetworkEdgeResponse{
			ID:            e.ID,
			PredecessorID: e.PredecessorID,
			SuccessorID:   e.SuccessorID,
			OverlapPct:    e.OverlapPct,
		})
	}

	resources, err := uc.repo.ListResourcesByRoute(ctx, id)
	if err != nil {
		return nil, err
	}
	resResps := make([]response.RouteOpResourceResponse, 0, len(resources))
	for _, r := range resources {
		resResps = append(resResps, toResourceResponse(r))
	}

	return &response.RouteDetailResponse{
		Route:      *toRouteResponse(rt),
		Operations: opResps,
		Network:    edgeResps,
		Resources:  resResps,
	}, nil
}

// ─── alternative resources ─────────────────────────────────────────────────────

func (uc *RouteUseCase) AddResource(ctx context.Context, dto request.AddRouteOpResourceDTO) (*response.RouteOpResourceResponse, error) {
	if dto.RouteOperationID <= 0 || dto.WorkCenterID <= 0 {
		return nil, fmt.Errorf("informe a operação do roteiro e o centro de trabalho")
	}
	tf := dto.TimeFactor
	if tf <= 0 {
		tf = 1
	}
	prio := dto.Priority
	if prio <= 0 {
		prio = 1
	}
	res := &entity.RouteOpResource{
		RouteOperationID: dto.RouteOperationID,
		WorkCenterID:     dto.WorkCenterID,
		Priority:         prio,
		TimeFactor:       tf,
		IsPrimary:        false, // set via SetRouteOpResourcePrimary to avoid unique conflicts
	}
	created, err := uc.repo.AddRouteOpResource(ctx, res)
	if err != nil {
		return nil, err
	}
	if dto.IsPrimary {
		created, err = uc.repo.SetRouteOpResourcePrimary(ctx, created.ID, created.RouteOperationID, created.WorkCenterID)
		if err != nil {
			return nil, err
		}
	}
	r := toResourceResponse(created)
	return &r, nil
}

func (uc *RouteUseCase) UpdateResource(ctx context.Context, dto request.UpdateRouteOpResourceDTO) (*response.RouteOpResourceResponse, error) {
	tf := dto.TimeFactor
	if tf <= 0 {
		tf = 1
	}
	prio := dto.Priority
	if prio <= 0 {
		prio = 1
	}
	updated, err := uc.repo.UpdateRouteOpResource(ctx, &entity.RouteOpResource{ID: dto.ID, Priority: prio, TimeFactor: tf})
	if err != nil {
		return nil, err
	}
	r := toResourceResponse(updated)
	return &r, nil
}

func (uc *RouteUseCase) SetPrimaryResource(ctx context.Context, resourceID int64) (*response.RouteOpResourceResponse, error) {
	res, err := uc.repo.GetRouteOpResource(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("recurso não encontrado: %w", err)
	}
	updated, err := uc.repo.SetRouteOpResourcePrimary(ctx, res.ID, res.RouteOperationID, res.WorkCenterID)
	if err != nil {
		return nil, err
	}
	r := toResourceResponse(updated)
	return &r, nil
}

func (uc *RouteUseCase) RemoveResource(ctx context.Context, resourceID int64) error {
	return uc.repo.RemoveRouteOpResource(ctx, resourceID)
}

func (uc *RouteUseCase) ListResources(ctx context.Context, routeOperationID int64) ([]response.RouteOpResourceResponse, error) {
	resources, err := uc.repo.ListResourcesByRouteOp(ctx, routeOperationID)
	if err != nil {
		return nil, err
	}
	out := make([]response.RouteOpResourceResponse, 0, len(resources))
	for _, r := range resources {
		out = append(out, toResourceResponse(r))
	}
	return out, nil
}

func toResourceResponse(r *entity.RouteOpResource) response.RouteOpResourceResponse {
	return response.RouteOpResourceResponse{
		ID:               r.ID,
		RouteOperationID: r.RouteOperationID,
		WorkCenterID:     r.WorkCenterID,
		WorkCenterName:   r.WorkCenterName,
		Priority:         r.Priority,
		TimeFactor:       r.TimeFactor,
		IsPrimary:        r.IsPrimary,
	}
}

func (uc *RouteUseCase) ListByItem(ctx context.Context, publicCode request.TextCode) ([]*response.ManufacturingRouteResponse, error) {
	itemCode, err := uc.resolveItemCode(ctx, publicCode)
	if err != nil {
		return nil, err
	}
	routes, err := uc.repo.ListRoutesByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ManufacturingRouteResponse, 0, len(routes))
	for _, rt := range routes {
		out = append(out, toRouteResponse(rt))
	}
	return out, nil
}

func (uc *RouteUseCase) resolveItemCode(ctx context.Context, publicCode request.TextCode) (int64, error) {
	if uc.items != nil {
		item, err := itemresolution.Resolve(ctx, uc.items, publicCode)
		if err != nil {
			return 0, err
		}
		return int64(item.Code), nil
	}
	code, err := strconv.ParseInt(publicCode.String(), 10, 64)
	if err != nil || code <= 0 {
		return 0, fmt.Errorf("item_code deve ser um código de item válido")
	}
	return code, nil
}

func (uc *RouteUseCase) Deactivate(ctx context.Context, id int64) error {
	return uc.repo.DeactivateRoute(ctx, id)
}

func (uc *RouteUseCase) AddOperation(ctx context.Context, dto request.AddRouteOperationDTO) (*response.RouteOperationResponse, error) {
	if dto.TimeUnit != nil && !validTimeUnit(*dto.TimeUnit) {
		return nil, fmt.Errorf("unidade de tempo %q inválida: use minuto, hora ou dia", *dto.TimeUnit)
	}
	remittance, err := normalizeThirdPartyRemittancePtr(dto.ThirdPartyRemittance)
	if err != nil {
		return nil, err
	}
	serviceItemCode, err := resolveOptionalItemCode(ctx, uc.items, dto.ServiceItemCode)
	if err != nil {
		return nil, err
	}
	sit := entity.RouteOpSituation(dto.Situation)
	if sit == "" {
		sit = entity.RouteOpApproved
	}
	op, err := entity.NewRouteOperation(dto.RouteID, dto.Sequence, dto.OperationID,
		dto.WorkCenterID, dto.StandardTime, dto.SetupTime, dto.Notes)
	if err != nil {
		return nil, err
	}
	op.Situation = sit
	op.RunTime = dto.RunTime
	op.LaborTime = dto.LaborTime
	op.RunBaseQty = dto.RunBaseQty
	op.QueueTime = dto.QueueTime
	op.WaitTime = dto.WaitTime
	op.MoveTime = dto.MoveTime
	op.CrewSize = dto.CrewSize
	op.TimeUnit = dto.TimeUnit
	op.SupplierID = dto.SupplierID
	op.ServiceItemCode = serviceItemCode
	op.CostPerUnit = dto.CostPerUnit
	op.LeadTimeDays = dto.LeadTimeDays
	op.ThirdPartyRemittance = remittance

	created, err := uc.repo.AddRouteOperation(ctx, op)
	if err != nil {
		return nil, err
	}
	r := toRouteOpResponse(created)
	return &r, nil
}

func (uc *RouteUseCase) UpdateOperation(ctx context.Context, dto request.UpdateRouteOperationDTO) (*response.RouteOperationResponse, error) {
	if dto.TimeUnit != nil && !validTimeUnit(*dto.TimeUnit) {
		return nil, fmt.Errorf("unidade de tempo %q inválida: use minuto, hora ou dia", *dto.TimeUnit)
	}
	remittance, err := normalizeThirdPartyRemittancePtr(dto.ThirdPartyRemittance)
	if err != nil {
		return nil, err
	}
	serviceItemCode, err := resolveOptionalItemCode(ctx, uc.items, dto.ServiceItemCode)
	if err != nil {
		return nil, err
	}
	op := &entity.RouteOperation{
		ID:                   dto.ID,
		WorkCenterID:         dto.WorkCenterID,
		StandardTime:         dto.StandardTime,
		SetupTime:            dto.SetupTime,
		RunTime:              dto.RunTime,
		LaborTime:            dto.LaborTime,
		RunBaseQty:           dto.RunBaseQty,
		QueueTime:            dto.QueueTime,
		WaitTime:             dto.WaitTime,
		MoveTime:             dto.MoveTime,
		CrewSize:             dto.CrewSize,
		TimeUnit:             dto.TimeUnit,
		SupplierID:           dto.SupplierID,
		ServiceItemCode:      serviceItemCode,
		CostPerUnit:          dto.CostPerUnit,
		LeadTimeDays:         dto.LeadTimeDays,
		ThirdPartyRemittance: remittance,
		Situation:            entity.RouteOpSituation(dto.Situation),
		Notes:                dto.Notes,
	}
	updated, err := uc.repo.UpdateRouteOperation(ctx, op)
	if err != nil {
		return nil, err
	}
	r := toRouteOpResponse(updated)
	return &r, nil
}

func normalizeThirdPartyRemittancePtr(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized, err := normalizeThirdPartyRemittance(*value)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func (uc *RouteUseCase) RemoveOperation(ctx context.Context, id int64) error {
	return uc.repo.RemoveRouteOperation(ctx, id)
}

func (uc *RouteUseCase) SetEdge(ctx context.Context, dto request.SetNetworkEdgeDTO) (*response.NetworkEdgeResponse, error) {
	if dto.OverlapPct < 0 || dto.OverlapPct > 100 {
		return nil, fmt.Errorf("a sobreposição deve estar entre 0 e 100")
	}
	edge := &entity.NetworkEdge{
		PredecessorID: dto.PredecessorID,
		SuccessorID:   dto.SuccessorID,
		OverlapPct:    dto.OverlapPct,
	}
	saved, err := uc.repo.SetNetworkEdge(ctx, edge)
	if err != nil {
		return nil, err
	}
	return &response.NetworkEdgeResponse{
		ID:            saved.ID,
		PredecessorID: saved.PredecessorID,
		SuccessorID:   saved.SuccessorID,
		OverlapPct:    saved.OverlapPct,
	}, nil
}

func (uc *RouteUseCase) DeleteEdge(ctx context.Context, dto request.DeleteNetworkEdgeDTO) error {
	return uc.repo.DeleteNetworkEdge(ctx, dto.PredecessorID, dto.SuccessorID)
}

func (uc *RouteUseCase) GetEdges(ctx context.Context, routeID int64) ([]response.NetworkEdgeResponse, error) {
	edges, err := uc.repo.GetNetworkEdges(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar as precedências do roteiro %d: %w", routeID, err)
	}
	out := make([]response.NetworkEdgeResponse, 0, len(edges))
	for _, e := range edges {
		out = append(out, response.NetworkEdgeResponse{
			ID:            e.ID,
			PredecessorID: e.PredecessorID,
			SuccessorID:   e.SuccessorID,
			OverlapPct:    e.OverlapPct,
		})
	}
	return out, nil
}

func toRouteResponse(rt *entity.ManufacturingRoute) *response.ManufacturingRouteResponse {
	return &response.ManufacturingRouteResponse{
		ID:          rt.ID,
		Code:        rt.Code,
		ItemCode:    rt.ItemCode,
		Mask:        rt.Mask,
		Alternative: rt.Alternative,
		Description: rt.Description,
		Situation:   string(rt.Situation),
		IsStandard:  rt.IsStandard,
		ValidFrom:   rt.ValidFrom,
		ValidTo:     rt.ValidTo,
		IsActive:    rt.IsActive,
		CreatedAt:   rt.CreatedAt,
	}
}

func toRouteOpResponse(op *entity.RouteOperation) response.RouteOperationResponse {
	return response.RouteOperationResponse{
		ID:               op.ID,
		RouteID:          op.RouteID,
		Sequence:         op.Sequence,
		OperationID:      op.OperationID,
		OperationName:    op.OperationName,
		WorkCenterID:     op.WorkCenterID,
		WorkCenterName:   op.WorkCenterName,
		StandardTime:     op.StandardTime,
		SetupTime:        op.SetupTime,
		EffectiveStdTime: op.EffectiveStdTime,
		EffectiveSetup:   op.EffectiveSetup,
		EffTime: response.OperationTimeBreakdown{
			Setup:      op.EffTime.Setup,
			Run:        op.EffTime.Run,
			Labor:      op.EffTime.Labor,
			RunBaseQty: op.EffTime.RunBaseQty,
			Queue:      op.EffTime.Queue,
			Wait:       op.EffTime.Wait,
			Move:       op.EffTime.Move,
			CrewSize:   op.EffTime.CrewSize,
		},
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          op.CostPerUnit,
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: op.ThirdPartyRemittance,
		Situation:            string(op.Situation),
		Notes:                op.Notes,
	}
}

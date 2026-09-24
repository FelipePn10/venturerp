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

	rt, err := entity.NewManufacturingRoute(code, itemCode, dto.Mask, alt, dto.Description, dto.IsStandard, dto.ValidFrom.Ponteiro(), dto.ValidTo.Ponteiro(), actor)
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
	rt.ValidFrom = dto.ValidFrom.Ponteiro()
	rt.ValidTo = dto.ValidTo.Ponteiro()

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

	// Quantidade que entra em cada etapa para o roteiro entregar 1 peça boa.
	// Com refugo essa conta é o que separa "soltei 100 e entreguei 96" de uma
	// ordem que fecha. A referência é 1 para a tela poder escalar para qualquer
	// lote sem outra chamada; o cálculo é linear na quantidade boa.
	const qtdeDeReferencia = 1.0
	entraPorEtapa := entity.QuantidadePorOperacao(ops, edges, qtdeDeReferencia)

	opResps := make([]response.RouteOperationResponse, 0, len(ops))
	for _, op := range ops {
		r := toRouteOpResponse(op)
		r.InputQty = entraPorEtapa[op.ID]
		opResps = append(opResps, r)
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

	// Documento e inspeção falham em silêncio de propósito: são enriquecimento
	// da tela, e um roteiro sem nenhum dos dois é o caso normal. Derrubar a
	// leitura do roteiro inteiro porque a leitura de anexos falhou seria pior
	// que mostrá-lo sem eles.
	docResps := []response.OperationDocumentResponse{}
	if docs, errDoc := uc.repo.ListDocumentsByRoute(ctx, id); errDoc == nil {
		docResps = mapDocumentos(docs)
	}
	insResps := []response.RouteInspectionResponse{}
	if ins, errIns := uc.repo.ListInspectionsByRoute(ctx, id); errIns == nil {
		insResps = make([]response.RouteInspectionResponse, 0, len(ins))
		for _, i := range ins {
			insResps = append(insResps, toInspectionResponse(i))
		}
	}

	return &response.RouteDetailResponse{
		Route:        *toRouteResponse(rt),
		Operations:   opResps,
		Network:      edgeResps,
		Resources:    resResps,
		Documents:    docResps,
		Inspections:  insResps,
		ReferenceQty: qtdeDeReferencia,
		ReleaseQty:   entity.QuantidadeASoltar(ops, edges, qtdeDeReferencia),
	}, nil
}

// ─── alternative resources ─────────────────────────────────────────────────────

func (uc *RouteUseCase) AddResource(ctx context.Context, dto request.AddRouteOpResourceDTO) (*response.RouteOpResourceResponse, error) {
	if dto.RouteOperationID <= 0 || dto.WorkCenterID <= 0 {
		return nil, errorsuc.NewValidationError("informe a operação do roteiro e o centro de trabalho")
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
		return 0, errorsuc.NewValidationError("item_code deve ser um código de item válido")
	}
	return code, nil
}

func (uc *RouteUseCase) Deactivate(ctx context.Context, id int64) error {
	return uc.repo.DeactivateRoute(ctx, id)
}

func (uc *RouteUseCase) AddOperation(ctx context.Context, dto request.AddRouteOperationDTO) (*response.RouteOperationResponse, error) {
	if dto.TimeUnit != nil {
		unidade, ok := normalizaUnidadeDeTempo(*dto.TimeUnit)
		if !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("unidade de tempo %q inválida: use SEGUNDO, MIN, HORA ou DIA", *dto.TimeUnit))
		}
		dto.TimeUnit = &unidade
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
	if err := validaRefugo(dto.ScrapPct); err != nil {
		return nil, err
	}
	op.ScrapPct = dto.ScrapPct
	op.InspectionRequired = dto.InspectionRequired

	created, err := uc.repo.AddRouteOperation(ctx, op)
	if err != nil {
		return nil, err
	}
	r := uc.etapaEnriquecida(ctx, dto.RouteID, created)
	return &r, nil
}

// etapaEnriquecida devolve a etapa como ela aparece ao recarregar a tela: com
// nome da operação, nome do centro de trabalho, tempo efetivo, refugo efetivo e
// quantidade de entrada.
//
// A gravação devolve só a linha crua da tabela — sem os campos que vêm do JOIN
// com a operação de biblioteca. A tela que confiasse nessa resposta mostraria
// uma linha em branco até alguém recarregar, e o usuário concluiria que não
// gravou. Reler custa uma consulta e elimina a dúvida.
func (uc *RouteUseCase) etapaEnriquecida(ctx context.Context, routeID int64, crua *entity.RouteOperation) response.RouteOperationResponse {
	ops, err := uc.repo.GetRouteOperations(ctx, routeID)
	if err != nil {
		return toRouteOpResponse(crua)
	}
	edges, _ := uc.repo.GetNetworkEdges(ctx, routeID)
	entra := entity.QuantidadePorOperacao(ops, edges, 1)
	for _, op := range ops {
		if op.ID == crua.ID {
			r := toRouteOpResponse(op)
			r.InputQty = entra[op.ID]
			return r
		}
	}
	return toRouteOpResponse(crua)
}

func (uc *RouteUseCase) UpdateOperation(ctx context.Context, dto request.UpdateRouteOperationDTO) (*response.RouteOperationResponse, error) {
	if dto.TimeUnit != nil {
		unidade, ok := normalizaUnidadeDeTempo(*dto.TimeUnit)
		if !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("unidade de tempo %q inválida: use SEGUNDO, MIN, HORA ou DIA", *dto.TimeUnit))
		}
		dto.TimeUnit = &unidade
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
		ScrapPct:             dto.ScrapPct,
		InspectionRequired:   dto.InspectionRequired,
		Situation:            entity.RouteOpSituation(dto.Situation),
		Notes:                dto.Notes,
	}
	if err := validaRefugo(dto.ScrapPct); err != nil {
		return nil, err
	}
	updated, err := uc.repo.UpdateRouteOperation(ctx, op)
	if err != nil {
		return nil, err
	}
	routeID, errRota := uc.repo.RouteIDOfOperation(ctx, dto.ID)
	if errRota != nil {
		r := toRouteOpResponse(updated)
		return &r, nil
	}
	r := uc.etapaEnriquecida(ctx, routeID, updated)
	return &r, nil
}

// validaRefugo recusa o que o banco também recusaria, mas com a explicação
// que o CHECK não dá. 100% de refugo não é um caso extremo: é a operação
// inteira jogada fora, e a conta `sai / (1 − refugo)` estouraria.
func validaRefugo(pct *float64) error {
	if pct == nil {
		return nil
	}
	if *pct < 0 {
		return errorsuc.NewValidationError("o refugo da operação não pode ser negativo")
	}
	if *pct >= 100 {
		return errorsuc.NewValidationError(
			"o refugo da operação precisa ser menor que 100%: com 100% nada sai bom e não existe quantidade a soltar")
	}
	return nil
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
	if dto.RouteID <= 0 {
		return nil, errorsuc.NewValidationError("abra o roteiro antes de definir suas dependências")
	}
	if _, err := uc.repo.GetRouteByID(ctx, dto.RouteID); err != nil {
		return nil, errorsuc.NewValidationError("roteiro não encontrado na empresa autenticada; carregue novamente os roteiros do item")
	}
	ops, err := uc.repo.GetRouteOperations(ctx, dto.RouteID)
	if err != nil {
		return nil, err
	}
	found := map[int64]bool{}
	for _, op := range ops {
		found[op.ID] = true
	}
	if !found[dto.PredecessorID] || !found[dto.SuccessorID] {
		return nil, errorsuc.NewValidationError("selecione duas etapas já cadastradas neste roteiro; atualize a lista se alguma etapa foi removida")
	}
	if dto.PredecessorID == dto.SuccessorID {
		return nil, errorsuc.NewValidationError("uma etapa não pode depender de si mesma")
	}
	edges, err := uc.repo.GetNetworkEdges(ctx, dto.RouteID)
	if err != nil {
		return nil, err
	}
	edges = append(edges, &entity.NetworkEdge{PredecessorID: dto.PredecessorID, SuccessorID: dto.SuccessorID, OverlapPct: dto.OverlapPct})
	if entity.CriticalPath(ops, edges, 1).HasCycle() {
		return nil, errorsuc.NewValidationError("esta dependência cria um ciclo; a etapa não pode depender de uma etapa que já depende dela")
	}
	if dto.OverlapPct < 0 || dto.OverlapPct > 100 {
		return nil, errorsuc.NewValidationError("a sobreposição deve estar entre 0 e 100")
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
		ID:                    op.ID,
		RouteID:               op.RouteID,
		Sequence:              op.Sequence,
		OperationID:           op.OperationID,
		OperationName:         op.OperationName,
		WorkCenterID:          op.WorkCenterID,
		EffectiveWorkCenterID: op.EffectiveWorkCenterID,
		WorkCenterName:        op.WorkCenterName,
		StandardTime:          op.StandardTime,
		SetupTime:             op.SetupTime,
		EffectiveStdTime:      op.EffectiveStdTime,
		EffectiveSetup:        op.EffectiveSetup,
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
		ScrapPct:             op.ScrapPct,
		EffectiveScrap:       op.EffectiveScrap,
		InspectionRequired:   op.InspectionRequired,
		Situation:            string(op.Situation),
		Notes:                op.Notes,
	}
}

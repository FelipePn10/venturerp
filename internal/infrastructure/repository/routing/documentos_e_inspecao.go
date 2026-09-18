package routing

import (
	"context"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ─── documentos de processo ───────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) CreateOperationDocument(ctx context.Context, doc *entity.OperationDocument) (*entity.OperationDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateOperationDocument(ctx, sqlc.CreateOperationDocumentParams{
		OperationID:      doc.OperationID,
		RouteOperationID: doc.RouteOperationID,
		Kind:             doc.Kind,
		Title:            doc.Title,
		Reference:        pgutil.ToPgTextFromPtr(doc.Reference),
		Revision:         pgutil.ToPgTextFromPtr(doc.Revision),
		Instructions:     pgutil.ToPgTextFromPtr(doc.Instructions),
		CreatedBy:        pgutil.ToPgUUID(doc.CreatedBy),
		EnterpriseID:     empresa,
	})
	if err != nil {
		return nil, fmt.Errorf("gravando o documento da operação: %w", err)
	}
	return documentoParaEntidade(row, row.RouteOperationID != nil), nil
}

func (r *RoutingRepositorySQLC) UpdateOperationDocument(ctx context.Context, doc *entity.OperationDocument) (*entity.OperationDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateOperationDocument(ctx, sqlc.UpdateOperationDocumentParams{
		ID:           doc.ID,
		Kind:         doc.Kind,
		Title:        doc.Title,
		Reference:    pgutil.ToPgTextFromPtr(doc.Reference),
		Revision:     pgutil.ToPgTextFromPtr(doc.Revision),
		Instructions: pgutil.ToPgTextFromPtr(doc.Instructions),
		EnterpriseID: empresa,
	})
	if err != nil {
		return nil, errorsuc.NewNotFoundError("documento não encontrado")
	}
	return documentoParaEntidade(row, row.RouteOperationID != nil), nil
}

func (r *RoutingRepositorySQLC) DeactivateOperationDocument(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.DeactivateOperationDocument(ctx, sqlc.DeactivateOperationDocumentParams{ID: id, EnterpriseID: empresa})
}

func (r *RoutingRepositorySQLC) ListDocumentsByOperation(ctx context.Context, operationID int64) ([]*entity.OperationDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListDocumentsByOperation(ctx, sqlc.ListDocumentsByOperationParams{
		OperationID: &operationID, EnterpriseID: empresa,
	})
	if err != nil {
		return nil, fmt.Errorf("lendo os documentos da operação: %w", err)
	}
	out := make([]*entity.OperationDocument, 0, len(rows))
	for _, row := range rows {
		out = append(out, documentoParaEntidade(row, false))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) ListDocumentsForRouteOperation(ctx context.Context, routeOperationID int64) ([]*entity.OperationDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListDocumentsForRouteOperation(ctx, sqlc.ListDocumentsForRouteOperationParams{
		RouteOperationID: &routeOperationID, EnterpriseID: empresa,
	})
	if err != nil {
		return nil, fmt.Errorf("lendo os documentos da etapa: %w", err)
	}
	out := make([]*entity.OperationDocument, 0, len(rows))
	for _, row := range rows {
		out = append(out, documentoParaEntidade(sqlc.OperationDocument{
			ID: row.ID, OperationID: row.OperationID, RouteOperationID: row.RouteOperationID,
			Kind: row.Kind, Title: row.Title, Reference: row.Reference, Revision: row.Revision,
			Instructions: row.Instructions, IsActive: row.IsActive,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CreatedBy: row.CreatedBy,
		}, row.RouteOperationID != nil))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) ListDocumentsByRoute(ctx context.Context, routeID int64) ([]*entity.OperationDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListDocumentsByRoute(ctx, sqlc.ListDocumentsByRouteParams{
		RouteID: routeID, EnterpriseID: empresa,
	})
	if err != nil {
		return nil, fmt.Errorf("lendo os documentos do roteiro: %w", err)
	}
	out := make([]*entity.OperationDocument, 0, len(rows))
	for _, row := range rows {
		doc := documentoParaEntidade(sqlc.OperationDocument{
			ID: row.ID, OperationID: row.OperationID, RouteOperationID: row.RouteOperationID,
			Kind: row.Kind, Title: row.Title, Reference: row.Reference, Revision: row.Revision,
			Instructions: row.Instructions, IsActive: row.IsActive,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CreatedBy: row.CreatedBy,
		}, row.RouteOperationID != nil)
		// A etapa a que o documento se aplica NESTE roteiro. Um documento de
		// biblioteca aparece uma vez por etapa que usa a operação — é isso que o
		// operador vê no posto, e é assim que a tela consegue agrupar por etapa.
		passo := row.StepID
		doc.RouteOperationID = &passo
		out = append(out, doc)
	}
	return out, nil
}

func documentoParaEntidade(row sqlc.OperationDocument, daEtapa bool) *entity.OperationDocument {
	return &entity.OperationDocument{
		ID:               row.ID,
		OperationID:      row.OperationID,
		RouteOperationID: row.RouteOperationID,
		Kind:             row.Kind,
		Title:            row.Title,
		Reference:        pgutil.FromPgTextPtr(row.Reference),
		Revision:         pgutil.FromPgTextPtr(row.Revision),
		Instructions:     pgutil.FromPgTextPtr(row.Instructions),
		IsActive:         row.IsActive,
		CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:        pgutil.FromPgUUID(row.CreatedBy),
		IsStepLevel:      daEtapa,
	}
}

// ─── pontos de inspeção do roteiro ────────────────────────────────────────────

func (r *RoutingRepositorySQLC) CreateRouteInspection(ctx context.Context, ins *entity.RouteInspection) (*entity.RouteInspection, error) {
	row, err := r.q.CreateInspectionPlan(ctx, sqlc.CreateInspectionPlanParams{
		ItemCode:         ins.ItemCode,
		RouteOperationID: pgutil.ToPgInt8Ptr(&ins.RouteOperationID),
		PointType:        sqlc.InspectionPointType(ins.PointType),
		Description:      ins.Description,
		SampleSize:       ins.SampleSize,
		AcceptanceLevel:  ins.AcceptanceLevel,
		Instructions:     pgutil.ToPgTextFromPtr(ins.Instructions),
		CreatedBy:        pgutil.ToPgUUID(ins.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("gravando o ponto de inspeção: %w", err)
	}
	ins.ID = row.ID
	ins.IsActive = row.IsActive
	ins.CreatedAt = pgutil.FromPgTimestamptz(row.CreatedAt)
	return ins, nil
}

func (r *RoutingRepositorySQLC) ListInspectionsByRoute(ctx context.Context, routeID int64) ([]*entity.RouteInspection, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListInspectionPlansByRoute(ctx, sqlc.ListInspectionPlansByRouteParams{
		RouteID: routeID, EnterpriseID: empresa,
	})
	if err != nil {
		return nil, fmt.Errorf("lendo os pontos de inspeção do roteiro: %w", err)
	}
	out := make([]*entity.RouteInspection, 0, len(rows))
	for _, row := range rows {
		ins := &entity.RouteInspection{
			ID:                  row.ID,
			ItemCode:            row.ItemCode,
			StepSequence:        row.StepSequence,
			PointType:           string(row.PointType),
			Description:         row.Description,
			SampleSize:          pgutil.FromPgNumericToFloat64(row.SampleSize),
			AcceptanceLevel:     pgutil.FromPgNumericToFloat64(row.AcceptanceLevel),
			Instructions:        pgutil.FromPgTextPtr(row.Instructions),
			CharacteristicCount: row.CharacteristicCount,
			IsActive:            row.IsActive,
			CreatedAt:           pgutil.FromPgTimestamptz(row.CreatedAt),
		}
		if row.RouteOperationID != nil {
			ins.RouteOperationID = *row.RouteOperationID
		}
		out = append(out, ins)
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) DeactivateRouteInspection(ctx context.Context, id int64) error {
	return r.q.DeactivateInspectionPlan(ctx, id)
}

func (r *RoutingRepositorySQLC) RouteIDOfOperation(ctx context.Context, routeOperationID int64) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	return r.q.RouteIDOfOperation(ctx, sqlc.RouteIDOfOperationParams{ID: routeOperationID, EnterpriseID: empresa})
}

package technical_assistance

import (
	"context"
	"encoding/json"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const rmaColumns = `code,enterprise_id,call_code,status,reason_code,reason_description,eligibility_status,
eligibility_reason,authorization_number,authorized_at,reverse_carrier_code,reverse_tracking_code,received_at,
inspection_notes,inspected_at,destination,sla_due_at,product_cost,freight_cost,service_cost,fiscal_document_key,
stock_movement_code,idempotency_key,created_at,updated_at,created_by`

func (r *RepositoryPGX) CreateRMA(ctx context.Context, enterpriseID int64, value *entity.RMA) (*entity.RMA, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var existing int64
	err = tx.QueryRow(ctx, `SELECT code FROM technical_assistance_rmas WHERE enterprise_id=$1 AND idempotency_key=$2`, enterpriseID, value.IdempotencyKey).Scan(&existing)
	if err == nil {
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
		return r.GetRMA(ctx, enterpriseID, existing)
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}
	var callExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM technical_assistance_calls c JOIN enterprise e ON e.code=c.enterprise_code WHERE c.code=$1 AND e.id=$2)`, value.CallCode, enterpriseID).Scan(&callExists); err != nil {
		return nil, err
	}
	if !callExists {
		return nil, errorsuc.NewNotFoundError("chamado não encontrado na empresa autenticada")
	}
	value.EnterpriseID = enterpriseID
	err = tx.QueryRow(ctx, `INSERT INTO technical_assistance_rmas(enterprise_id,call_code,status,reason_code,reason_description,eligibility_status,eligibility_reason,sla_due_at,product_cost,freight_cost,service_cost,idempotency_key,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING `+rmaColumns,
		enterpriseID, value.CallCode, value.Status, value.ReasonCode, value.ReasonDescription, value.EligibilityStatus, value.EligibilityReason, value.SLADueAt, value.ProductCost, value.FreightCost, value.ServiceCost, value.IdempotencyKey, pgutil.ToPgUUID(value.CreatedBy)).Scan(rmaScanTargets(value)...)
	if err != nil {
		return nil, err
	}
	for _, item := range value.Items {
		item.RMACode = value.Code
		err = tx.QueryRow(ctx, `INSERT INTO technical_assistance_rma_items(rma_code,call_item_code,item_code,quantity,serial_number,lot_number,requested_destination)
			SELECT $1,$2,$3,$4,$5,$6,$7 WHERE EXISTS(SELECT 1 FROM technical_assistance_call_items i WHERE i.code=$2 AND i.call_code=$8)
			RETURNING code`, item.RMACode, item.CallItemCode, item.ItemCode, item.Quantity, item.SerialNumber, item.LotNumber, item.RequestedDestination, value.CallCode).Scan(&item.Code)
		if err != nil {
			return nil, fmt.Errorf("item do RMA não pertence ao chamado: %w", err)
		}
	}
	after, _ := json.Marshal(value)
	_, err = tx.Exec(ctx, `INSERT INTO technical_assistance_rma_events(enterprise_id,rma_code,event_type,after_state,actor_id) VALUES($1,$2,'CRIADO',$3,$4)`, enterpriseID, value.Code, after, pgutil.ToPgUUID(value.CreatedBy))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetRMA(ctx, enterpriseID, value.Code)
}

func (r *RepositoryPGX) GetRMA(ctx context.Context, enterpriseID, code int64) (*entity.RMA, error) {
	v := &entity.RMA{}
	if err := r.pool.QueryRow(ctx, `SELECT `+rmaColumns+` FROM technical_assistance_rmas WHERE code=$1 AND enterprise_id=$2`, code, enterpriseID).Scan(rmaScanTargets(v)...); err != nil {
		return nil, err
	}
	items, err := r.pool.Query(ctx, `SELECT code,rma_code,call_item_code,item_code,quantity,serial_number,lot_number,requested_destination,inspection_result FROM technical_assistance_rma_items WHERE rma_code=$1 ORDER BY code`, code)
	if err != nil {
		return nil, err
	}
	for items.Next() {
		item := &entity.RMAItem{}
		if err = items.Scan(&item.Code, &item.RMACode, &item.CallItemCode, &item.ItemCode, &item.Quantity, &item.SerialNumber, &item.LotNumber, &item.RequestedDestination, &item.InspectionResult); err != nil {
			items.Close()
			return nil, err
		}
		v.Items = append(v.Items, item)
	}
	items.Close()
	events, err := r.pool.Query(ctx, `SELECT code,rma_code,event_type,before_state,after_state,reason,correlation_id,occurred_at,actor_id FROM technical_assistance_rma_events WHERE enterprise_id=$1 AND rma_code=$2 ORDER BY occurred_at,code`, enterpriseID, code)
	if err != nil {
		return nil, err
	}
	defer events.Close()
	for events.Next() {
		event := &entity.RMAEvent{}
		if err = events.Scan(&event.Code, &event.RMACode, &event.EventType, &event.BeforeState, &event.AfterState, &event.Reason, &event.CorrelationID, &event.OccurredAt, &event.ActorID); err != nil {
			return nil, err
		}
		v.Events = append(v.Events, event)
	}
	if err = events.Err(); err != nil {
		return nil, err
	}
	v.Evidences, err = r.ListRMAEvidences(ctx, enterpriseID, code)
	return v, err
}

func (r *RepositoryPGX) ListRMAsByCall(ctx context.Context, enterpriseID, callCode int64) ([]*entity.RMA, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+rmaColumns+` FROM technical_assistance_rmas WHERE enterprise_id=$1 AND call_code=$2 ORDER BY created_at DESC,code DESC`, enterpriseID, callCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*entity.RMA, 0)
	for rows.Next() {
		v := &entity.RMA{}
		if err = rows.Scan(rmaScanTargets(v)...); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) TransitionRMA(ctx context.Context, enterpriseID, code int64, nextStatus string, reason *string, correlationID *string, actorID uuid.UUID, changes *entity.RMA) (*entity.RMA, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	current := &entity.RMA{}
	if err = tx.QueryRow(ctx, `SELECT `+rmaColumns+` FROM technical_assistance_rmas WHERE code=$1 AND enterprise_id=$2 FOR UPDATE`, code, enterpriseID).Scan(rmaScanTargets(current)...); err != nil {
		return nil, err
	}
	before, _ := json.Marshal(current)
	err = tx.QueryRow(ctx, `UPDATE technical_assistance_rmas SET status=$3,authorization_number=$4,authorized_at=$5,reverse_carrier_code=$6,reverse_tracking_code=$7,received_at=$8,inspection_notes=$9,inspected_at=$10,destination=$11,product_cost=$12,freight_cost=$13,service_cost=$14,fiscal_document_key=$15,stock_movement_code=$16,updated_at=NOW() WHERE code=$1 AND enterprise_id=$2 RETURNING `+rmaColumns,
		code, enterpriseID, nextStatus, changes.AuthorizationNumber, changes.AuthorizedAt, changes.ReverseCarrierCode, changes.ReverseTrackingCode, changes.ReceivedAt, changes.InspectionNotes, changes.InspectedAt, changes.Destination, changes.ProductCost, changes.FreightCost, changes.ServiceCost, changes.FiscalDocumentKey, changes.StockMovementCode).Scan(rmaScanTargets(changes)...)
	if err != nil {
		return nil, err
	}
	after, _ := json.Marshal(changes)
	_, err = tx.Exec(ctx, `INSERT INTO technical_assistance_rma_events(enterprise_id,rma_code,event_type,before_state,after_state,reason,correlation_id,actor_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, enterpriseID, code, "STATUS_"+nextStatus, before, after, reason, correlationID, pgutil.ToPgUUID(actorID))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetRMA(ctx, enterpriseID, code)
}

func (r *RepositoryPGX) CreateRMAEvidence(ctx context.Context, enterpriseID int64, value *entity.RMAEvidence) (*entity.RMAEvidence, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO technical_assistance_rma_evidences
		(enterprise_id,rma_code,file_name,content_type,content,size_bytes,sha256,uploaded_by)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8
		WHERE EXISTS(SELECT 1 FROM technical_assistance_rmas WHERE code=$2 AND enterprise_id=$1)
		ON CONFLICT(enterprise_id,rma_code,sha256) DO UPDATE SET file_name=EXCLUDED.file_name
		RETURNING id,rma_code,file_name,content_type,content,size_bytes,sha256,uploaded_by,created_at`,
		enterpriseID, value.RMACode, value.FileName, value.ContentType, value.Content, value.SizeBytes,
		value.SHA256, pgutil.ToPgUUID(value.UploadedBy))
	if err := row.Scan(&value.ID, &value.RMACode, &value.FileName, &value.ContentType, &value.Content,
		&value.SizeBytes, &value.SHA256, &value.UploadedBy, &value.CreatedAt); err != nil {
		return nil, err
	}
	return value, nil
}

func (r *RepositoryPGX) ListRMAEvidences(ctx context.Context, enterpriseID, rmaCode int64) ([]*entity.RMAEvidence, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,rma_code,file_name,content_type,size_bytes,sha256,uploaded_by,created_at
		FROM technical_assistance_rma_evidences WHERE enterprise_id=$1 AND rma_code=$2 ORDER BY created_at DESC,id`, enterpriseID, rmaCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*entity.RMAEvidence, 0)
	for rows.Next() {
		value := &entity.RMAEvidence{}
		if err = rows.Scan(&value.ID, &value.RMACode, &value.FileName, &value.ContentType, &value.SizeBytes,
			&value.SHA256, &value.UploadedBy, &value.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) GetRMAEvidence(ctx context.Context, enterpriseID, rmaCode int64, evidenceID uuid.UUID) (*entity.RMAEvidence, error) {
	value := &entity.RMAEvidence{}
	err := r.pool.QueryRow(ctx, `SELECT id,rma_code,file_name,content_type,content,size_bytes,sha256,uploaded_by,created_at
		FROM technical_assistance_rma_evidences WHERE enterprise_id=$1 AND rma_code=$2 AND id=$3`, enterpriseID, rmaCode, evidenceID).
		Scan(&value.ID, &value.RMACode, &value.FileName, &value.ContentType, &value.Content, &value.SizeBytes,
			&value.SHA256, &value.UploadedBy, &value.CreatedAt)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func rmaScanTargets(v *entity.RMA) []any {
	return []any{&v.Code, &v.EnterpriseID, &v.CallCode, &v.Status, &v.ReasonCode, &v.ReasonDescription, &v.EligibilityStatus, &v.EligibilityReason, &v.AuthorizationNumber, &v.AuthorizedAt, &v.ReverseCarrierCode, &v.ReverseTrackingCode, &v.ReceivedAt, &v.InspectionNotes, &v.InspectedAt, &v.Destination, &v.SLADueAt, &v.ProductCost, &v.FreightCost, &v.ServiceCost, &v.FiscalDocumentKey, &v.StockMovementCode, &v.IdempotencyKey, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy}
}

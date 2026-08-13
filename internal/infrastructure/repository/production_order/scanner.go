package production_order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/jackc/pgx/v5"
)

func (r *ProductionOrderRepositoryPGX) CreateScanToken(ctx context.Context, token *entity.ScanToken) error {
	command, err := r.pool.Exec(ctx, `INSERT INTO production_scan_tokens
 (enterprise_id,production_order_id,operation_id,token_hash,valid_until,created_by)
 SELECT $1,o.id,$3,$4,$5,$6 FROM production_orders o
 WHERE o.id=$2 AND o.enterprise_id=$1 AND ($3::bigint IS NULL OR EXISTS(
  SELECT 1 FROM production_order_operations op WHERE op.id=$3 AND op.production_order_id=o.id AND op.enterprise_id=$1))`,
		token.EnterpriseID, token.ProductionOrderID, token.OperationID, token.TokenHash, token.ValidUntil, token.CreatedBy)
	if err != nil {
		return fmt.Errorf("criar token de apontamento: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("ordem/operacao nao encontrada na empresa autenticada")
	}
	return nil
}

func (r *ProductionOrderRepositoryPGX) RevokeScanToken(ctx context.Context, enterpriseID int64, hash []byte, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE production_scan_tokens SET active=FALSE,revoked_at=$3 WHERE enterprise_id=$1 AND token_hash=$2`, enterpriseID, hash, at)
	return err
}

func (r *ProductionOrderRepositoryPGX) ExecuteScan(ctx context.Context, command entity.ScanCommand) (result *entity.ScanResult, retErr error) {
	defer func() {
		if retErr == nil {
			return
		}
		_, _ = r.pool.Exec(ctx, `INSERT INTO production_scan_events(enterprise_id,user_id,device_id,action,result,idempotency_key,request_fingerprint,message)
		 VALUES($1,$2,$3,$4,'REJEITADO',$5,$6,$7) ON CONFLICT(enterprise_id,user_id,action,idempotency_key) DO NOTHING`,
			command.EnterpriseID, command.UserID, command.DeviceID, string(command.Action), command.IdempotencyKey, command.Fingerprint, retErr.Error())
	}()
	var priorFingerprint, priorResponse []byte
	var priorMessage *string
	var priorResult string
	err := r.pool.QueryRow(ctx, `SELECT request_fingerprint,response,message,result FROM production_scan_events WHERE enterprise_id=$1 AND user_id=$2 AND action=$3 AND idempotency_key=$4`, command.EnterpriseID, command.UserID, string(command.Action), command.IdempotencyKey).Scan(&priorFingerprint, &priorResponse, &priorMessage, &priorResult)
	if err == nil {
		if !bytes.Equal(priorFingerprint, command.Fingerprint) {
			return nil, fmt.Errorf("chave de idempotencia reutilizada com conteudo diferente")
		}
		if priorResult != "SUCESSO" {
			if priorMessage != nil {
				return nil, fmt.Errorf("%s", *priorMessage)
			}
			return nil, fmt.Errorf("leitura rejeitada anteriormente")
		}
		var replay entity.ScanResult
		if e := json.Unmarshal(priorResponse, &replay); e != nil {
			return nil, e
		}
		replay.Replayed = true
		return &replay, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var tokenID, orderID, orderNumber int64
	var operationID *int64
	var status string
	err = tx.QueryRow(ctx, `SELECT t.id,t.production_order_id,t.operation_id,o.order_number,o.status FROM production_scan_tokens t JOIN production_orders o ON o.id=t.production_order_id AND o.enterprise_id=t.enterprise_id WHERE t.enterprise_id=$1 AND t.token_hash=$2 AND t.active AND t.valid_from<=NOW() AND (t.valid_until IS NULL OR t.valid_until>=NOW()) FOR UPDATE OF t,o`, command.EnterpriseID, command.TokenHash).Scan(&tokenID, &orderID, &operationID, &orderNumber, &status)
	if err != nil {
		return nil, fmt.Errorf("token invalido, expirado ou de outra empresa")
	}
	operationStatus := (*string)(nil)
	switch command.Action {
	case entity.ScanResolve:
	case entity.ScanStart:
		if status != "OPEN" {
			return nil, fmt.Errorf("OF deve estar ABERTA para iniciar")
		}
		if _, err = tx.Exec(ctx, `UPDATE production_orders SET status='IN_PROGRESS',start_date=COALESCE(start_date,CURRENT_DATE),updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, orderID, command.EnterpriseID); err != nil {
			return nil, err
		}
		status = "EM_ANDAMENTO"
	case entity.ScanAppoint:
		if status != "IN_PROGRESS" {
			return nil, fmt.Errorf("OF deve estar EM_ANDAMENTO para apontar")
		}
		if operationID != nil {
			var blocked bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_operations current JOIN production_order_operations prior ON prior.production_order_id=current.production_order_id AND prior.enterprise_id=current.enterprise_id AND prior.sequence<current.sequence WHERE current.id=$1 AND current.enterprise_id=$2 AND prior.status NOT IN ('DONE','SKIPPED'))`, *operationID, command.EnterpriseID).Scan(&blocked)
			if err != nil {
				return nil, err
			}
			if blocked {
				return nil, fmt.Errorf("operacao anterior ainda nao concluida")
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO production_appointments(production_order_id,operation_id,employee_id,appointment_date,produced_qty,scrapped_qty,scrap_reason,created_by) VALUES($1,$2,$3,CURRENT_DATE,$4::numeric,$5::numeric,$6,$7)`, orderID, operationID, command.EmployeeID, command.GoodQuantity.String(), command.ScrapQuantity.String(), command.ScrapReason, command.UserID)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE production_orders SET produced_qty=produced_qty+$2::numeric,scrapped_qty=scrapped_qty+$3::numeric,updated_at=NOW() WHERE id=$1 AND enterprise_id=$4`, orderID, command.GoodQuantity.String(), command.ScrapQuantity.String(), command.EnterpriseID)
		if err != nil {
			return nil, err
		}
		if operationID != nil {
			dbStatus := "IN_PROGRESS"
			publicStatus := "EM_ANDAMENTO"
			if command.CompleteOperation {
				dbStatus = "DONE"
				publicStatus = "CONCLUIDA"
			}
			_, err = tx.Exec(ctx, `UPDATE production_order_operations SET status=$3,actual_hours=COALESCE(actual_hours,0)+$4::numeric,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, *operationID, command.EnterpriseID, dbStatus, command.Hours.String())
			if err != nil {
				return nil, err
			}
			operationStatus = &publicStatus
		}
		status = "EM_ANDAMENTO"
	case entity.ScanComplete:
		if status != "IN_PROGRESS" {
			return nil, fmt.Errorf("OF deve estar EM_ANDAMENTO para concluir")
		}
		var pending bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_operations WHERE production_order_id=$1 AND enterprise_id=$2 AND status NOT IN ('DONE','SKIPPED'))`, orderID, command.EnterpriseID).Scan(&pending)
		if err != nil {
			return nil, err
		}
		if pending {
			return nil, fmt.Errorf("existem operacoes pendentes")
		}
		_, err = tx.Exec(ctx, `UPDATE production_orders SET status='COMPLETED',end_date=CURRENT_DATE,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, orderID, command.EnterpriseID)
		if err != nil {
			return nil, err
		}
		status = "CONCLUIDA"
	}
	if status == "OPEN" {
		status = "ABERTA"
	}
	if status == "IN_PROGRESS" {
		status = "EM_ANDAMENTO"
	}
	if status == "COMPLETED" {
		status = "CONCLUIDA"
	}
	result = &entity.ScanResult{ProductionOrderID: orderID, OperationID: operationID, OrderNumber: orderNumber, Status: status, OperationStatus: operationStatus}
	responseJSON, _ := json.Marshal(result)
	_, err = tx.Exec(ctx, `INSERT INTO production_scan_events(enterprise_id,token_id,production_order_id,operation_id,user_id,device_id,action,result,idempotency_key,request_fingerprint,response) VALUES($1,$2,$3,$4,$5,$6,$7,'SUCESSO',$8,$9,$10)`, command.EnterpriseID, tokenID, orderID, operationID, command.UserID, command.DeviceID, string(command.Action), command.IdempotencyKey, command.Fingerprint, responseJSON)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

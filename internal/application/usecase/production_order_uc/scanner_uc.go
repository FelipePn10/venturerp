package production_order_uc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	"github.com/shopspring/decimal"
)

type ProductionScannerUseCase struct {
	Repo repository.ProductionScannerRepository
	Auth ports.AuthService
}

func (uc *ProductionScannerUseCase) CreateToken(ctx context.Context, dto request.CreateProductionScanTokenDTO) (*response.ProductionScanTokenResponse, error) {
	if !uc.Auth.CanReleaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ProductionOrderID <= 0 {
		return nil, errorsuc.NewValidationError("ordem de fabricacao obrigatoria")
	}
	var validUntil *time.Time
	if dto.ValidUntil != nil {
		parsed, e := time.Parse(time.RFC3339, *dto.ValidUntil)
		if e != nil {
			return nil, errorsuc.NewValidationError("valid_until deve estar em RFC3339")
		}
		validUntil = &parsed
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return nil, fmt.Errorf("gerar token: %w", err)
	}
	token := "OF1_" + base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	record := &entity.ScanToken{EnterpriseID: enterpriseID, ProductionOrderID: dto.ProductionOrderID, OperationID: dto.OperationID, TokenHash: hash[:], ValidUntil: validUntil, CreatedBy: userID}
	if err = uc.Repo.CreateScanToken(ctx, record); err != nil {
		return nil, err
	}
	return &response.ProductionScanTokenResponse{Token: token, BarcodeValue: token, ProductionOrderID: dto.ProductionOrderID, OperationID: dto.OperationID, ValidUntil: validUntil}, nil
}

func (uc *ProductionScannerUseCase) Scan(ctx context.Context, dto request.ProductionScanDTO) (*entity.ScanResult, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	action := entity.ScanAction(strings.ToUpper(strings.TrimSpace(dto.Action)))
	if action != entity.ScanResolve && action != entity.ScanStart && action != entity.ScanAppoint && action != entity.ScanComplete {
		return nil, errorsuc.NewValidationError("acao deve ser RESOLVER, INICIAR, APONTAR ou CONCLUIR")
	}
	if strings.TrimSpace(dto.Token) == "" || strings.TrimSpace(dto.IdempotencyKey) == "" || strings.TrimSpace(dto.DeviceID) == "" {
		return nil, errorsuc.NewValidationError("token, idempotency_key e device_id sao obrigatorios")
	}
	good, e := decimal.NewFromString(defaultZero(dto.GoodQuantity))
	if e != nil {
		return nil, errorsuc.NewValidationError("good_quantity invalida")
	}
	scrap, e := decimal.NewFromString(defaultZero(dto.ScrapQuantity))
	if e != nil {
		return nil, errorsuc.NewValidationError("scrap_quantity invalida")
	}
	hours, e := decimal.NewFromString(defaultZero(dto.Hours))
	if e != nil {
		return nil, errorsuc.NewValidationError("hours invalida")
	}
	if good.IsNegative() || scrap.IsNegative() || hours.IsNegative() {
		return nil, errorsuc.NewValidationError("quantidades e horas nao podem ser negativas")
	}
	if action == entity.ScanAppoint && (dto.EmployeeID == nil || *dto.EmployeeID <= 0) {
		return nil, errorsuc.NewValidationError("operador obrigatorio")
	}
	if scrap.IsPositive() && (dto.ScrapReason == nil || strings.TrimSpace(*dto.ScrapReason) == "") {
		return nil, errorsuc.NewValidationError("motivo do refugo obrigatorio")
	}
	payload, _ := json.Marshal(dto)
	fingerprint := sha256.Sum256(payload)
	tokenHash := sha256.Sum256([]byte(dto.Token))
	return uc.Repo.ExecuteScan(ctx, entity.ScanCommand{EnterpriseID: enterpriseID, UserID: userID, TokenHash: tokenHash[:], Action: action, IdempotencyKey: dto.IdempotencyKey, DeviceID: dto.DeviceID, Fingerprint: fingerprint[:], EmployeeID: dto.EmployeeID, GoodQuantity: good, ScrapQuantity: scrap, Hours: hours, ScrapReason: dto.ScrapReason, CompleteOperation: dto.CompleteOperation})
}
func defaultZero(v string) string {
	if strings.TrimSpace(v) == "" {
		return "0"
	}
	return v
}

package technical_assistance_uc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	tarepo "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository"
	"github.com/google/uuid"
)

const maxRMAEvidenceSize = 10 * 1024 * 1024

var rmaTransitions = map[string][]string{
	"SOLICITADO":        {"AUTORIZADO", "CANCELADO"},
	"AUTORIZADO":        {"LOGISTICA_REVERSA", "RECEBIDO", "CANCELADO"},
	"LOGISTICA_REVERSA": {"RECEBIDO", "CANCELADO"},
	"RECEBIDO":          {"INSPECIONADO"},
	"INSPECIONADO":      {"RESOLVIDO"},
}

func (uc *UseCase) rmaRepository() (tarepo.RMARepository, error) {
	if uc.RMAs != nil {
		return uc.RMAs, nil
	}
	if repo, ok := uc.Repo.(tarepo.RMARepository); ok {
		return repo, nil
	}
	return nil, errorsuc.NewValidationError("o fluxo de RMA não está configurado")
}

func (uc *UseCase) rmaEvidenceRepository() (tarepo.RMAEvidenceRepository, error) {
	if repo, ok := uc.RMAs.(tarepo.RMAEvidenceRepository); ok {
		return repo, nil
	}
	if repo, ok := uc.Repo.(tarepo.RMAEvidenceRepository); ok {
		return repo, nil
	}
	return nil, errorsuc.NewValidationError("o armazenamento de evidências do RMA não está configurado")
}

func (uc *UseCase) AddRMAEvidence(ctx context.Context, rmaCode int64, fileName, contentType string, content []byte) (*response.TechnicalAssistanceRMAEvidenceResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := uc.actorID(ctx)
	if err != nil {
		return nil, err
	}
	if rmaCode <= 0 || len(content) == 0 || len(content) > maxRMAEvidenceSize {
		return nil, errorsuc.NewValidationError("a evidência deve possuir conteúdo e no máximo 10 MiB")
	}
	cleanName := filepath.Base(strings.TrimSpace(fileName))
	if cleanName == "" || cleanName == "." || cleanName != fileName || strings.ContainsAny(fileName, `/\`) || strings.ContainsRune(fileName, '\x00') {
		return nil, errorsuc.NewValidationError("o nome do arquivo da evidência é inválido")
	}
	if !allowedRMAEvidenceType(contentType) {
		return nil, errorsuc.NewValidationError("tipo de evidência não permitido; use PDF, PNG ou JPEG")
	}
	repo, err := uc.rmaEvidenceRepository()
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(content)
	created, err := repo.CreateRMAEvidence(ctx, tenantID, &entity.RMAEvidence{RMACode: rmaCode, FileName: cleanName,
		ContentType: contentType, Content: content, SizeBytes: int64(len(content)), SHA256: fmt.Sprintf("%x", digest), UploadedBy: actor})
	if err != nil {
		return nil, err
	}
	return rmaEvidenceResponse(created), nil
}

func (uc *UseCase) ListRMAEvidences(ctx context.Context, rmaCode int64) ([]*response.TechnicalAssistanceRMAEvidenceResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := uc.rmaEvidenceRepository()
	if err != nil {
		return nil, err
	}
	rows, err := repo.ListRMAEvidences(ctx, tenantID, rmaCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.TechnicalAssistanceRMAEvidenceResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, rmaEvidenceResponse(row))
	}
	return out, nil
}

func (uc *UseCase) GetRMAEvidence(ctx context.Context, rmaCode int64, evidenceID uuid.UUID) (*entity.RMAEvidence, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := uc.rmaEvidenceRepository()
	if err != nil {
		return nil, err
	}
	return repo.GetRMAEvidence(ctx, tenantID, rmaCode, evidenceID)
}

func allowedRMAEvidenceType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0])) {
	case "application/pdf", "image/png", "image/jpeg":
		return true
	default:
		return false
	}
}

func rmaEvidenceResponse(value *entity.RMAEvidence) *response.TechnicalAssistanceRMAEvidenceResponse {
	return &response.TechnicalAssistanceRMAEvidenceResponse{ID: value.ID, RMACode: value.RMACode, FileName: value.FileName,
		ContentType: value.ContentType, SizeBytes: value.SizeBytes, SHA256: value.SHA256,
		DownloadURL: fmt.Sprintf("/api/technical-assistance/rmas/%d/evidences/%s", value.RMACode, value.ID),
		UploadedBy:  value.UploadedBy, CreatedAt: value.CreatedAt}
}

func (uc *UseCase) CreateRMA(ctx context.Context, dto request.CreateTechnicalAssistanceRMADTO) (*response.TechnicalAssistanceRMAResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := uc.actorID(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := uc.rmaRepository()
	if err != nil {
		return nil, err
	}
	dto.ReasonCode = strings.TrimSpace(dto.ReasonCode)
	dto.EligibilityStatus = strings.ToUpper(strings.TrimSpace(dto.EligibilityStatus))
	dto.EligibilityReason = strings.TrimSpace(dto.EligibilityReason)
	dto.IdempotencyKey = strings.TrimSpace(dto.IdempotencyKey)
	if dto.CallCode <= 0 {
		return nil, errorsuc.NewValidationError("o chamado é obrigatório")
	}
	if dto.IdempotencyKey == "" || len(dto.IdempotencyKey) > 100 {
		return nil, errorsuc.NewValidationError("a chave de idempotência é obrigatória e deve possuir até 100 caracteres")
	}
	if dto.ReasonCode == "" || dto.EligibilityReason == "" {
		return nil, errorsuc.NewValidationError("motivo e justificativa de elegibilidade são obrigatórios")
	}
	if dto.EligibilityStatus != "ELEGIVEL" && dto.EligibilityStatus != "NAO_ELEGIVEL" && dto.EligibilityStatus != "REVISAO" {
		return nil, errorsuc.NewValidationError("situação de elegibilidade inválida")
	}
	sla, err := time.Parse(time.RFC3339, dto.SLADueAt)
	if err != nil {
		return nil, errorsuc.NewValidationError("sla_due_at deve usar o formato ISO 8601")
	}
	if len(dto.Items) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos um item para o RMA")
	}
	rma := &entity.RMA{CallCode: dto.CallCode, Status: "SOLICITADO", ReasonCode: dto.ReasonCode, ReasonDescription: dto.ReasonDescription, EligibilityStatus: dto.EligibilityStatus, EligibilityReason: dto.EligibilityReason, SLADueAt: sla, ProductCost: dto.ProductCost, FreightCost: dto.FreightCost, ServiceCost: dto.ServiceCost, IdempotencyKey: dto.IdempotencyKey, CreatedBy: actor, Items: make([]*entity.RMAItem, 0, len(dto.Items))}
	if rma.ProductCost < 0 || rma.FreightCost < 0 || rma.ServiceCost < 0 {
		return nil, errorsuc.NewValidationError("os custos do RMA não podem ser negativos")
	}
	seen := map[int64]bool{}
	for _, item := range dto.Items {
		if item.CallItemCode <= 0 || item.ItemCode <= 0 || item.Quantity <= 0 {
			return nil, errorsuc.NewValidationError("item, linha do chamado e quantidade positiva são obrigatórios")
		}
		if seen[item.CallItemCode] {
			return nil, errorsuc.NewValidationError("uma linha do chamado não pode ser repetida no RMA")
		}
		seen[item.CallItemCode] = true
		dest := normalizeRMADestination(item.RequestedDestination)
		if item.RequestedDestination != nil && dest == nil {
			return nil, errorsuc.NewValidationError("destino solicitado inválido")
		}
		rma.Items = append(rma.Items, &entity.RMAItem{CallItemCode: item.CallItemCode, ItemCode: item.ItemCode, Quantity: item.Quantity, SerialNumber: item.SerialNumber, LotNumber: item.LotNumber, RequestedDestination: dest})
	}
	created, err := repo.CreateRMA(ctx, tenantID, rma)
	if err != nil {
		return nil, err
	}
	return rmaResponse(created), nil
}

func (uc *UseCase) GetRMA(ctx context.Context, code int64) (*response.TechnicalAssistanceRMAResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := uc.rmaRepository()
	if err != nil {
		return nil, err
	}
	row, err := repo.GetRMA(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}
	return rmaResponse(row), nil
}
func (uc *UseCase) ListRMAsByCall(ctx context.Context, callCode int64) ([]*response.TechnicalAssistanceRMAResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := uc.rmaRepository()
	if err != nil {
		return nil, err
	}
	rows, err := repo.ListRMAsByCall(ctx, tenantID, callCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.TechnicalAssistanceRMAResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, rmaResponse(row))
	}
	return out, nil
}

func (uc *UseCase) TransitionRMA(ctx context.Context, code int64, dto request.TransitionTechnicalAssistanceRMADTO) (*response.TechnicalAssistanceRMAResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := uc.actorID(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := uc.rmaRepository()
	if err != nil {
		return nil, err
	}
	current, err := repo.GetRMA(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}
	next := strings.ToUpper(strings.TrimSpace(dto.Status))
	if !containsRMAAction(rmaTransitions[current.Status], next) {
		return nil, errorsuc.NewConflictError("a transição informada não é permitida para a situação atual do RMA")
	}
	if (next == "CANCELADO" || next == "RESOLVIDO") && (dto.Reason == nil || strings.TrimSpace(*dto.Reason) == "") {
		return nil, errorsuc.NewValidationError("o motivo é obrigatório para cancelar ou resolver o RMA")
	}
	if next == "AUTORIZADO" {
		if current.EligibilityStatus != "ELEGIVEL" {
			return nil, errorsuc.NewConflictError("somente RMA elegível pode ser autorizado")
		}
		if dto.AuthorizationNumber == nil || strings.TrimSpace(*dto.AuthorizationNumber) == "" {
			return nil, errorsuc.NewValidationError("o número de autorização é obrigatório")
		}
		now := time.Now().UTC()
		current.AuthorizedAt = &now
	}
	if next == "LOGISTICA_REVERSA" && (dto.ReverseTrackingCode == nil || strings.TrimSpace(*dto.ReverseTrackingCode) == "") {
		return nil, errorsuc.NewValidationError("o código de rastreio da logística reversa é obrigatório")
	}
	if next == "RECEBIDO" {
		now := time.Now().UTC()
		current.ReceivedAt = &now
	}
	if next == "INSPECIONADO" {
		if dto.InspectionNotes == nil || strings.TrimSpace(*dto.InspectionNotes) == "" {
			return nil, errorsuc.NewValidationError("o resultado da inspeção é obrigatório")
		}
		now := time.Now().UTC()
		current.InspectedAt = &now
	}
	dest := normalizeRMADestination(dto.Destination)
	if dto.Destination != nil && dest == nil {
		return nil, errorsuc.NewValidationError("destino inválido; use REPARO, TROCA, CREDITO ou SUCATA")
	}
	if next == "RESOLVIDO" && dest == nil && current.Destination == nil {
		return nil, errorsuc.NewValidationError("o destino é obrigatório para resolver o RMA")
	}
	current.AuthorizationNumber = coalesceString(dto.AuthorizationNumber, current.AuthorizationNumber)
	current.ReverseCarrierCode = coalesceInt64(dto.ReverseCarrierCode, current.ReverseCarrierCode)
	current.ReverseTrackingCode = coalesceString(dto.ReverseTrackingCode, current.ReverseTrackingCode)
	current.InspectionNotes = coalesceString(dto.InspectionNotes, current.InspectionNotes)
	if dest != nil {
		current.Destination = dest
	}
	current.FiscalDocumentKey = coalesceString(dto.FiscalDocumentKey, current.FiscalDocumentKey)
	current.StockMovementCode = coalesceInt64(dto.StockMovementCode, current.StockMovementCode)
	if dto.ProductCost != nil {
		current.ProductCost = *dto.ProductCost
	}
	if dto.FreightCost != nil {
		current.FreightCost = *dto.FreightCost
	}
	if dto.ServiceCost != nil {
		current.ServiceCost = *dto.ServiceCost
	}
	if current.ProductCost < 0 || current.FreightCost < 0 || current.ServiceCost < 0 {
		return nil, errorsuc.NewValidationError("os custos do RMA não podem ser negativos")
	}
	updated, err := repo.TransitionRMA(ctx, tenantID, code, next, dto.Reason, dto.CorrelationID, actor, current)
	if err != nil {
		return nil, err
	}
	return rmaResponse(updated), nil
}

func rmaResponse(v *entity.RMA) *response.TechnicalAssistanceRMAResponse {
	out := &response.TechnicalAssistanceRMAResponse{Code: v.Code, CallCode: v.CallCode, Status: v.Status, ReasonCode: v.ReasonCode, ReasonDescription: v.ReasonDescription, EligibilityStatus: v.EligibilityStatus, EligibilityReason: v.EligibilityReason, AuthorizationNumber: v.AuthorizationNumber, AuthorizedAt: v.AuthorizedAt, ReverseCarrierCode: v.ReverseCarrierCode, ReverseTrackingCode: v.ReverseTrackingCode, ReceivedAt: v.ReceivedAt, InspectionNotes: v.InspectionNotes, InspectedAt: v.InspectedAt, Destination: v.Destination, SLADueAt: v.SLADueAt, ProductCost: v.ProductCost, FreightCost: v.FreightCost, ServiceCost: v.ServiceCost, TotalCost: v.ProductCost + v.FreightCost + v.ServiceCost, FiscalDocumentKey: v.FiscalDocumentKey, StockMovementCode: v.StockMovementCode, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, AllowedActions: append([]string(nil), rmaTransitions[v.Status]...), Items: make([]response.TechnicalAssistanceRMAItemResponse, 0, len(v.Items)), Events: make([]response.TechnicalAssistanceRMAEventResponse, 0, len(v.Events)), Evidences: make([]response.TechnicalAssistanceRMAEvidenceResponse, 0, len(v.Evidences))}
	for _, i := range v.Items {
		out.Items = append(out.Items, response.TechnicalAssistanceRMAItemResponse{Code: i.Code, CallItemCode: i.CallItemCode, ItemCode: i.ItemCode, Quantity: i.Quantity, SerialNumber: i.SerialNumber, LotNumber: i.LotNumber, RequestedDestination: i.RequestedDestination, InspectionResult: i.InspectionResult})
	}
	for _, e := range v.Events {
		out.Events = append(out.Events, response.TechnicalAssistanceRMAEventResponse{Code: e.Code, EventType: e.EventType, BeforeState: json.RawMessage(e.BeforeState), AfterState: json.RawMessage(e.AfterState), Reason: e.Reason, CorrelationID: e.CorrelationID, OccurredAt: e.OccurredAt, ActorID: e.ActorID})
	}
	for _, evidence := range v.Evidences {
		out.Evidences = append(out.Evidences, *rmaEvidenceResponse(evidence))
	}
	return out
}
func normalizeRMADestination(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.ToUpper(strings.TrimSpace(*v))
	switch s {
	case "REPARO", "TROCA", "CREDITO", "SUCATA":
		return &s
	}
	return nil
}
func containsRMAAction(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func coalesceString(v, current *string) *string {
	if v != nil {
		return v
	}
	return current
}
func coalesceInt64(v, current *int64) *int64 {
	if v != nil {
		return v
	}
	return current
}

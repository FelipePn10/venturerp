package routing_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/google/uuid"
)

// Domínios fechados. Oferecer na tela uma opção que o CHECK recusa é pior que
// não oferecer: o usuário escolhe, preenche o resto e só então leva erro.
var (
	tiposDeDocumento = map[string]string{
		"DESENHO":   "Desenho",
		"INSTRUCAO": "Instrução de trabalho",
		"FICHA":     "Ficha de processo",
		"NORMA":     "Norma",
		"FOTO":      "Foto",
		"OUTRO":     "Outro",
	}
	pontosDeInspecao = map[string]string{
		"RECEBIMENTO": "Recebimento",
		"PROCESSO":    "Processo",
		"EXPEDICAO":   "Expedição",
	}
)

func listaLegivel(m map[string]string) string {
	partes := make([]string, 0, len(m))
	for k := range m {
		partes = append(partes, k)
	}
	return strings.Join(partes, ", ")
}

// ─── documentos ───────────────────────────────────────────────────────────────

func (uc *RouteUseCase) AddDocument(ctx context.Context, dto request.CreateOperationDocumentDTO) (*response.OperationDocumentResponse, error) {
	titulo := strings.TrimSpace(dto.Title)
	if titulo == "" {
		return nil, errorsuc.NewValidationError("informe o título do documento")
	}
	tipo := strings.ToUpper(strings.TrimSpace(dto.Kind))
	if tipo == "" {
		tipo = "INSTRUCAO"
	}
	if _, ok := tiposDeDocumento[tipo]; !ok {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"tipo de documento %q inválido: use %s", dto.Kind, listaLegivel(tiposDeDocumento)))
	}
	// Um vínculo e apenas um: o documento ou vale para toda etapa que usa a
	// operação, ou é desta etapa. Os dois ao mesmo tempo apareceria duplicado no
	// posto e ninguém saberia qual revisão vale.
	temOperacao := dto.OperationID != nil && *dto.OperationID > 0
	temEtapa := dto.RouteOperationID != nil && *dto.RouteOperationID > 0
	if temOperacao == temEtapa {
		return nil, errorsuc.NewValidationError(
			"informe OU a operação de biblioteca (o documento vale em todo roteiro) OU a etapa do roteiro (o documento é deste item), nunca as duas")
	}

	var actor uuid.UUID
	if uc.auth != nil {
		id, err := uc.auth.UserID(ctx)
		if err != nil {
			return nil, errorsuc.ErrUnauthorized
		}
		actor = id
	}

	doc := &entity.OperationDocument{
		Kind: tipo, Title: titulo,
		Reference: textoOuNil(dto.Reference), Revision: textoOuNil(dto.Revision),
		Instructions: textoOuNil(dto.Instructions),
		CreatedBy:    actor,
	}
	if temOperacao {
		doc.OperationID = dto.OperationID
	} else {
		doc.RouteOperationID = dto.RouteOperationID
	}

	criado, err := uc.repo.CreateOperationDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	r := toDocumentResponse(criado)
	return &r, nil
}

func (uc *RouteUseCase) UpdateDocument(ctx context.Context, dto request.UpdateOperationDocumentDTO) (*response.OperationDocumentResponse, error) {
	titulo := strings.TrimSpace(dto.Title)
	if titulo == "" {
		return nil, errorsuc.NewValidationError("informe o título do documento")
	}
	tipo := strings.ToUpper(strings.TrimSpace(dto.Kind))
	if _, ok := tiposDeDocumento[tipo]; !ok {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"tipo de documento %q inválido: use %s", dto.Kind, listaLegivel(tiposDeDocumento)))
	}
	atualizado, err := uc.repo.UpdateOperationDocument(ctx, &entity.OperationDocument{
		ID: dto.ID, Kind: tipo, Title: titulo,
		Reference: textoOuNil(dto.Reference), Revision: textoOuNil(dto.Revision),
		Instructions: textoOuNil(dto.Instructions),
	})
	if err != nil {
		return nil, err
	}
	r := toDocumentResponse(atualizado)
	return &r, nil
}

func (uc *RouteUseCase) RemoveDocument(ctx context.Context, id int64) error {
	return uc.repo.DeactivateOperationDocument(ctx, id)
}

func (uc *RouteUseCase) ListDocumentsByOperation(ctx context.Context, operationID int64) ([]response.OperationDocumentResponse, error) {
	docs, err := uc.repo.ListDocumentsByOperation(ctx, operationID)
	if err != nil {
		return nil, err
	}
	return mapDocumentos(docs), nil
}

func (uc *RouteUseCase) ListDocumentsForStep(ctx context.Context, routeOperationID int64) ([]response.OperationDocumentResponse, error) {
	docs, err := uc.repo.ListDocumentsForRouteOperation(ctx, routeOperationID)
	if err != nil {
		return nil, err
	}
	return mapDocumentos(docs), nil
}

// ─── pontos de inspeção ───────────────────────────────────────────────────────

func (uc *RouteUseCase) AddInspection(ctx context.Context, dto request.CreateRouteInspectionDTO) (*response.RouteInspectionResponse, error) {
	if dto.RouteOperationID <= 0 {
		return nil, errorsuc.NewValidationError("informe a etapa do roteiro onde a inspeção acontece")
	}
	descricao := strings.TrimSpace(dto.Description)
	if descricao == "" {
		return nil, errorsuc.NewValidationError("descreva o que será inspecionado")
	}
	ponto := strings.ToUpper(strings.TrimSpace(dto.PointType))
	if ponto == "" {
		ponto = "PROCESSO" // inspeção no meio do roteiro é o caso comum
	}
	if _, ok := pontosDeInspecao[ponto]; !ok {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"ponto de inspeção %q inválido: use %s", dto.PointType, listaLegivel(pontosDeInspecao)))
	}
	if dto.SampleSize <= 0 {
		return nil, errorsuc.NewValidationError("o tamanho da amostra precisa ser maior que zero")
	}
	if dto.AcceptanceLevel < 0 || dto.AcceptanceLevel > 100 {
		return nil, errorsuc.NewValidationError("o nível de aceitação vai de 0 a 100%")
	}

	// O item vem do roteiro a que a etapa pertence — o plano de inspeção é do
	// item, e pedir isso na tela seria pedir uma informação que o sistema já tem
	// (e que o usuário poderia errar).
	rota, seq, err := uc.rotaDaEtapa(ctx, dto.RouteOperationID)
	if err != nil {
		return nil, err
	}

	var actor uuid.UUID
	if uc.auth != nil {
		id, errAuth := uc.auth.UserID(ctx)
		if errAuth != nil {
			return nil, errorsuc.ErrUnauthorized
		}
		actor = id
	}

	criado, err := uc.repo.CreateRouteInspection(ctx, &entity.RouteInspection{
		RouteOperationID: dto.RouteOperationID,
		ItemCode:         rota.ItemCode,
		StepSequence:     seq,
		PointType:        ponto,
		Description:      descricao,
		SampleSize:       dto.SampleSize,
		AcceptanceLevel:  dto.AcceptanceLevel,
		Instructions:     textoOuNil(dto.Instructions),
		CreatedBy:        actor,
	})
	if err != nil {
		return nil, err
	}
	r := toInspectionResponse(criado)
	return &r, nil
}

func (uc *RouteUseCase) RemoveInspection(ctx context.Context, id int64) error {
	return uc.repo.DeactivateRouteInspection(ctx, id)
}

// rotaDaEtapa localiza o roteiro (e a sequência) de uma etapa pelo seu id.
func (uc *RouteUseCase) rotaDaEtapa(ctx context.Context, routeOperationID int64) (*entity.ManufacturingRoute, int16, error) {
	// O repositório lê etapas por roteiro, não por id de etapa. Em vez de
	// acrescentar uma query só para isto, a busca passa pelo roteiro — e o
	// caminho já valida que a etapa pertence à empresa do usuário.
	rotaID, err := uc.repo.RouteIDOfOperation(ctx, routeOperationID)
	if err != nil {
		return nil, 0, errorsuc.NewNotFoundError("etapa de roteiro não encontrada")
	}
	rota, err := uc.repo.GetRouteByID(ctx, rotaID)
	if err != nil {
		return nil, 0, errorsuc.NewNotFoundError("roteiro não encontrado")
	}
	ops, err := uc.repo.GetRouteOperations(ctx, rotaID)
	if err != nil {
		return nil, 0, err
	}
	for _, op := range ops {
		if op.ID == routeOperationID {
			return rota, op.Sequence, nil
		}
	}
	return rota, 0, nil
}

// ─── mapeadores ───────────────────────────────────────────────────────────────

func textoOuNil(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil
	}
	return &t
}

func toDocumentResponse(d *entity.OperationDocument) response.OperationDocumentResponse {
	return response.OperationDocumentResponse{
		ID:               d.ID,
		OperationID:      d.OperationID,
		RouteOperationID: d.RouteOperationID,
		Kind:             d.Kind,
		Title:            d.Title,
		Reference:        d.Reference,
		Revision:         d.Revision,
		Instructions:     d.Instructions,
		IsStepLevel:      d.IsStepLevel,
	}
}

func mapDocumentos(docs []*entity.OperationDocument) []response.OperationDocumentResponse {
	out := make([]response.OperationDocumentResponse, 0, len(docs))
	for _, d := range docs {
		out = append(out, toDocumentResponse(d))
	}
	return out
}

func toInspectionResponse(i *entity.RouteInspection) response.RouteInspectionResponse {
	return response.RouteInspectionResponse{
		ID:                  i.ID,
		RouteOperationID:    i.RouteOperationID,
		StepSequence:        i.StepSequence,
		PointType:           i.PointType,
		Description:         i.Description,
		SampleSize:          i.SampleSize,
		AcceptanceLevel:     i.AcceptanceLevel,
		Instructions:        i.Instructions,
		CharacteristicCount: i.CharacteristicCount,
	}
}

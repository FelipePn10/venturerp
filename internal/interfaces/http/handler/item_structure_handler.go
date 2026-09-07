package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/go-chi/chi/v5"
)

func (h *ItemStructureHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateStructureComponentDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}

	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, result)
}

func (h *ItemStructureHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateStructureComponentDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}

	if dto.ParentCode.String() == "" || dto.ChildCode.String() == "" {
		jsonError(w, http.StatusBadRequest, "parent_code e child_code são obrigatórios")
		return
	}

	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// Delete removes a structure component identified by its public item codes.
func (h *ItemStructureHandler) Delete(w http.ResponseWriter, r *http.Request) {
	parent := request.TextCode(chi.URLParam(r, "parentCode"))
	child := request.TextCode(chi.URLParam(r, "childCode"))
	if parent.String() == "" || child.String() == "" {
		jsonError(w, http.StatusBadRequest, "parentCode e childCode são obrigatórios")
		return
	}
	var mask *string
	if value := r.URL.Query().Get("mask"); value != "" {
		mask = &value
	}
	if err := h.deleteUC.Execute(r.Context(), parent, child, mask); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetTree returns the BOM tree for a root item.
func (h *ItemStructureHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	rootItemCode := request.TextCode(chi.URLParam(r, "rootItemCode"))
	if rootItemCode.String() == "" {
		jsonError(w, http.StatusBadRequest, "rootItemCode é obrigatório")
		return
	}

	dto := request.GetStructureTreeDTO{
		RootItemCode: rootItemCode,
	}

	result, err := h.treeUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// GetAllDirectChildren returns direct children of a structure component.
func (h *ItemStructureHandler) GetAllDirectChildren(w http.ResponseWriter, r *http.Request) {
	parentItemCode := request.TextCode(chi.URLParam(r, "parentItemCode"))
	if parentItemCode.String() == "" {
		jsonError(w, http.StatusBadRequest, "parentItemCode é obrigatório")
		return
	}

	dto := request.GetAllDirectChildrenDTO{
		ParentItemCode: parentItemCode,
	}

	result, err := h.getAllStructure.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

//// ResolveForMask resolves BOM tree using a mask configuration.
//func (h *ItemStructureHandler) ResolveForMask(w http.ResponseWriter, r *http.Request) {
//	rootItemCode, err := parseCode(r, "rootItemCode")
//	if err != nil {
//		jsonError(w, http.StatusBadRequest, err.Error())
//		return
//	}
//
//	maskValue := r.URL.Query().Get("mask")
//	if maskValue == "" {
//		jsonError(w, http.StatusBadRequest, "query param 'mask' is required (example: ?mask=100%23100%2350)")
//		return
//	}
//
//	dto := request.ResolveStructureForMaskDTO{
//		RootItemCode:  rootItemCode,
//		RootMaskValue: maskValue,
//	}
//
//	result, err := h.resolveUC.Execute(r.Context(), dto)
//	if err != nil {
//		jsonError(w, http.StatusUnprocessableEntity, err.Error())
//		return
//	}
//
//	jsonResponse(w, http.StatusOK, result)
//}

// parseCode extracts and validates a numeric code from URL params.
func parseCode(r *http.Request, param string) (int64, error) {
	raw := chi.URLParam(r, param)

	code, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || code <= 0 {
		return 0, fmt.Errorf("o parâmetro '%s' deve ser um número maior que zero", param)
	}

	return code, nil
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	msg = strings.NewReplacer(
		"conteúdo da requisição inválido", "corpo da requisição inválido",
		"invalid shipment code", "código do romaneio inválido",
		"invalid load code", "código da carga inválido",
		"invalid volume id", "identificador do volume inválido",
		"código inválido", "identificador inválido",
		"not found", "não encontrado",
		"is required", "é obrigatório",
		"not configured", "não configurado",
	).Replace(msg)
	if status >= http.StatusInternalServerError {
		msg = "erro interno do servidor"
	}
	code := "ERRO_INTERNO"
	switch status {
	case http.StatusBadRequest:
		code = "REQUISICAO_INVALIDA"
	case http.StatusNotFound:
		code = "REGISTRO_NAO_ENCONTRADO"
	case http.StatusConflict:
		code = "CONFLITO_DE_DOMINIO"
	case http.StatusUnprocessableEntity:
		code = "VALIDACAO_DE_DOMINIO"
	case http.StatusForbidden:
		code = "ACESSO_NEGADO"
	}
	jsonResponse(w, status, map[string]string{"error": msg, "code": code})
}

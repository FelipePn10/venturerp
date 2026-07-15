package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/system_update_uc"
)

type SystemUpdateHandler struct {
	manager *system_update_uc.Manager
}

func NewSystemUpdateHandler(manager *system_update_uc.Manager) *SystemUpdateHandler {
	return &SystemUpdateHandler{manager: manager}
}

func (h *SystemUpdateHandler) Status(w http.ResponseWriter, r *http.Request) {
	status, err := h.manager.Status(r.Context())
	if err != nil {
		http.Error(w, `{"error":"não foi possível consultar a atualização"}`, http.StatusInternalServerError)
		return
	}
	writeSystemUpdateJSON(w, http.StatusOK, status)
}

func (h *SystemUpdateHandler) Request(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Version string `json:"version"`
	}
	if r.Body != http.NoBody {
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
			return
		}
	}
	status, err := h.manager.Request(r.Context(), input.Version)
	if errors.Is(err, system_update_uc.ErrUpdateInProgress) {
		http.Error(w, `{"error":"uma atualização já está em andamento"}`, http.StatusConflict)
		return
	}
	if errors.Is(err, system_update_uc.ErrInvalidVersion) || errors.Is(err, system_update_uc.ErrNoRelease) {
		http.Error(w, `{"error":"versão de atualização inválida ou indisponível"}`, http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"não foi possível solicitar a atualização"}`, http.StatusInternalServerError)
		return
	}
	writeSystemUpdateJSON(w, http.StatusAccepted, status)
}

func writeSystemUpdateJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

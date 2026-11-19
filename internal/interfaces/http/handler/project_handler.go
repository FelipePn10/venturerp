package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/project_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// ProjectHandler: cadastro de Projetos (VPLA0120).
type ProjectHandler struct{ uc *project_uc.UseCase }

func NewProjectHandler(uc *project_uc.UseCase) *ProjectHandler { return &ProjectHandler{uc: uc} }

func (h *ProjectHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{code}", h.Get)
	r.Put("/{code}", h.Update)
	return r
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.List(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("status"))
	if err != nil {
		respondErr(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Get(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		respondErr(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.ProjectDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.ProjectDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Update(r.Context(), chi.URLParam(r, "code"), dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

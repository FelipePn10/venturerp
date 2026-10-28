package handler

import (
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise"
	"github.com/go-chi/chi/v5"
)

func (h *EnterpriseHandler) CreateEnterprise(w http.ResponseWriter, r *http.Request) {
	var req request.CreateEnterpriseDTO

	if !security.DecodeBody(w, r, &req) {
		return
	}
	// O criador é o usuário da sessão — o corpo não escolhe quem fica ADMIN.
	ator, err := h.createEnterpriseUC.Auth.UserID(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	req.CreatedBy = ator

	enterprise, err := mapper.ToEnterpriseEntity(req)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	created, err := h.createEnterpriseUC.Execute(r.Context(), enterprise)
	if err != nil {
		// A duplicate code is a client conflict (409), not a server error.
		if c, ok := errorsuc.AsConflict(err); ok {
			h.Conflict(w, c.Error())
			return
		}
		security.RespondUseCaseError(w, err)
		return
	}

	h.Created(w, created, "enterprise created succesfully")
}

func (h *EnterpriseHandler) GetEnterprise(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(chi.URLParam(r, "code"))
	if err != nil {
		h.BadRequest(w, "código inválido")
		return
	}
	result, err := h.getEnterpriseUC.Execute(r.Context(), code)
	if err != nil {
		h.NotFound(w, "empresa não encontrada")
		return
	}
	h.OK(w, result)
}

func (h *EnterpriseHandler) ListEnterprises(w http.ResponseWriter, r *http.Request) {
	results, err := h.listEnterprisesUC.Execute(r.Context())
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, results)
}

func (h *EnterpriseHandler) Identificacao(w http.ResponseWriter, r *http.Request) {
	out, err := h.identificacaoUC.Obter(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	h.OK(w, out)
}

func (h *EnterpriseHandler) AtualizarIdentificacao(w http.ResponseWriter, r *http.Request) {
	var req request.IdentificacaoEmpresaDTO
	if !security.DecodeBody(w, r, &req) {
		return
	}
	out, err := h.identificacaoUC.Atualizar(r.Context(), req)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	h.OK(w, out)
}

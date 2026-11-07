package handler

import (
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/shopspring/decimal"
)

// SimularTributacaoHandler expõe o simulador de tributação: o mesmo cálculo da
// nota, sem gravar nada.
type SimularTributacaoHandler struct {
	uc *fiscal_uc.SimularTributacaoUseCase
}

func NewSimularTributacaoHandler(uc *fiscal_uc.SimularTributacaoUseCase) *SimularTributacaoHandler {
	return &SimularTributacaoHandler{uc: uc}
}

func (h *SimularTributacaoHandler) Simular(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.SimularTributacaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.uc.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// ImpostosDoItem: a mesma simulação, para a prévia fiscal do cadastro do item
// (venda a empresa contribuinte, para industrialização, na UF pedida). Fica sob
// /api/items para quem cadastra item ver os impostos sem acesso ao módulo fiscal.
func (h *SimularTributacaoHandler) ImpostosDoItem(w http.ResponseWriter, r *http.Request) {
	codigo := request.TextCode(itemCodePathParam(r, "item"))
	uf := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("uf")))
	if len(uf) != 2 {
		security.RespondError(w, http.StatusBadRequest, "informe a UF de destino em \"uf\"")
		return
	}
	// Contribuinte é quem tem inscrição estadual: sem uma, o motor calcularia
	// como consumidor final (IPI na base do ICMS, DIFAL).
	pessoa, finalidade, ie := "J", "INDUSTRIALIZACAO", "CONTRIBUINTE"
	out, err := h.uc.Execute(r.Context(), fiscal_uc.SimularTributacaoDTO{
		UFDestino: &uf, TipoPessoa: &pessoa, FinalidadeCompra: &finalidade, IEDestinatario: &ie,
		Itens: []fiscal_uc.SimularTributacaoItemDTO{{ItemCode: &codigo, Quantidade: decimal.NewFromInt(1), ValorUnitario: decimal.NewFromInt(100)}},
	})
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

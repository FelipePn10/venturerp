package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/margin_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/margin/entity"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// MarginHandler expõe a margem de contribuição: parâmetros do mês, geração e
// apuração (FoccoERP FCST0108 / FCST0254 / FCST0320).
type MarginHandler struct{ uc *margin_uc.UseCase }

func NewMarginHandler(uc *margin_uc.UseCase) *MarginHandler { return &MarginHandler{uc: uc} }

type parametrosDTO struct {
	Ano                  int     `json:"ano"`
	Mes                  int     `json:"mes"`
	IRPct                float64 `json:"ir_pct"`
	AdminPct             float64 `json:"admin_pct"`
	FreightPct           float64 `json:"freight_pct"`
	FinancialRateMonthly float64 `json:"financial_rate_monthly"`
	AvgSalesTermDays     int     `json:"avg_sales_term_days"`
	AvgPurchaseTermDays  int     `json:"avg_purchase_term_days"`
	ProductionCycleDays  int     `json:"production_cycle_days"`
	MaterialPaymentDays  int     `json:"material_payment_days"`
	LaborPaymentDays     int     `json:"labor_payment_days"`
	IPIPaymentDays       int     `json:"ipi_payment_days"`
	ICMSPaymentDays      int     `json:"icms_payment_days"`
	PISPaymentDays       int     `json:"pis_payment_days"`
	COFINSPaymentDays    int     `json:"cofins_payment_days"`
}

func (d parametrosDTO) paraEntidade() entity.Parametros {
	return entity.Parametros{
		Ano: d.Ano, Mes: d.Mes,
		IRPct: d.IRPct, AdminPct: d.AdminPct, FreightPct: d.FreightPct,
		FinancialRateMonthly: d.FinancialRateMonthly,
		AvgSalesTermDays:     d.AvgSalesTermDays,
		AvgPurchaseTermDays:  d.AvgPurchaseTermDays,
		ProductionCycleDays:  d.ProductionCycleDays,
		MaterialPaymentDays:  d.MaterialPaymentDays,
		LaborPaymentDays:     d.LaborPaymentDays,
		IPIPaymentDays:       d.IPIPaymentDays,
		ICMSPaymentDays:      d.ICMSPaymentDays,
		PISPaymentDays:       d.PISPaymentDays,
		COFINSPaymentDays:    d.COFINSPaymentDays,
	}
}

// comCalculados devolve os parâmetros junto com o ciclo de caixa e a taxa real,
// que são derivados — mostrá-los evita que o usuário refaça a conta de cabeça.
func comCalculados(p entity.Parametros) map[string]any {
	return map[string]any{
		"ano": p.Ano, "mes": p.Mes,
		"ir_pct": p.IRPct, "admin_pct": p.AdminPct, "freight_pct": p.FreightPct,
		"financial_rate_monthly":  p.FinancialRateMonthly,
		"avg_sales_term_days":     p.AvgSalesTermDays,
		"avg_purchase_term_days":  p.AvgPurchaseTermDays,
		"production_cycle_days":   p.ProductionCycleDays,
		"material_payment_days":   p.MaterialPaymentDays,
		"labor_payment_days":      p.LaborPaymentDays,
		"ipi_payment_days":        p.IPIPaymentDays,
		"icms_payment_days":       p.ICMSPaymentDays,
		"pis_payment_days":        p.PISPaymentDays,
		"cofins_payment_days":     p.COFINSPaymentDays,
		"cash_cycle_days":         p.CicloDeCaixaDias(),
		"real_financial_rate_pct": p.TaxaFinanceiraRealPct(),
	}
}

func (h *MarginHandler) SaveParameters(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var dto parametrosDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	p := dto.paraEntidade()
	if err := h.uc.SalvarParametros(r.Context(), p); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, comCalculados(p))
}

func (h *MarginHandler) GetParameters(w http.ResponseWriter, r *http.Request) {
	ano, _ := strconv.Atoi(r.URL.Query().Get("ano"))
	mes, _ := strconv.Atoi(r.URL.Query().Get("mes"))
	if ano == 0 || mes == 0 {
		agora := time.Now()
		if ano == 0 {
			ano = agora.Year()
		}
		if mes == 0 {
			mes = int(agora.Month())
		}
	}
	p, err := h.uc.ParametrosDoMes(r.Context(), ano, mes)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	if p == nil {
		security.RespondJSON(w, http.StatusOK, nil)
		return
	}
	security.RespondJSON(w, http.StatusOK, comCalculados(*p))
}

func (h *MarginHandler) ListParameters(w http.ResponseWriter, r *http.Request) {
	ano, _ := strconv.Atoi(r.URL.Query().Get("ano"))
	if ano == 0 {
		ano = time.Now().Year()
	}
	lista, err := h.uc.ListarParametros(r.Context(), ano)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(lista))
	for _, p := range lista {
		out = append(out, comCalculados(p))
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func datasDaQuery(r *http.Request) (time.Time, time.Time, error) {
	de, err := time.Parse("2006-01-02", strings.TrimSpace(r.URL.Query().Get("de")))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	ate, err := time.Parse("2006-01-02", strings.TrimSpace(r.URL.Query().Get("ate")))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return de, ate, nil
}

func (h *MarginHandler) Generate(w http.ResponseWriter, r *http.Request) {
	de, ate, err := datasDaQuery(r)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "informe o período no formato ano-mês-dia (de e ate)")
		return
	}
	base := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("base")))
	if base == "" {
		base = "PADRAO"
	}
	resumo, err := h.uc.Gerar(r.Context(), de, ate, base)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, resumo)
}

func (h *MarginHandler) Report(w http.ResponseWriter, r *http.Request) {
	de, ate, err := datasDaQuery(r)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "informe o período no formato ano-mês-dia (de e ate)")
		return
	}
	ordem := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("ordem")))
	var item, cliente *int64
	if v, err := strconv.ParseInt(r.URL.Query().Get("item_code"), 10, 64); err == nil && v > 0 {
		item = &v
	}
	if v, err := strconv.ParseInt(r.URL.Query().Get("customer_code"), 10, 64); err == nil && v > 0 {
		cliente = &v
	}
	linhas, err := h.uc.Apuracao(r.Context(), de, ate, ordem, item, cliente)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, linhas)
}

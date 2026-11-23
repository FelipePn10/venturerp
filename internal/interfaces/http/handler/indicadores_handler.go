package handler

import (
	"net/http"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/indicadores_uc"
)

// IndicadoresHandler expõe os indicadores prontos.
type IndicadoresHandler struct{ UC *indicadores_uc.UseCase }

// Calcular: GET /api/indicadores?de=AAAA-MM-DD&ate=AAAA-MM-DD (padrão: últimos 90 dias).
func (h *IndicadoresHandler) Calcular(w http.ResponseWriter, r *http.Request) {
	ate := time.Now()
	de := ate.AddDate(0, 0, -90)
	for _, p := range []struct {
		nome string
		dst  *time.Time
	}{{"de", &de}, {"ate", &ate}} {
		if v := r.URL.Query().Get(p.nome); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				jsonError(w, http.StatusBadRequest, "data inválida em "+p.nome+" (use AAAA-MM-DD)")
				return
			}
			*p.dst = t
		}
	}
	out, err := h.UC.Calcular(r.Context(), de, ate)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

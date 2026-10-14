package handler

import (
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/forecast_uc"
)

func StatisticalForecastHandler(w http.ResponseWriter, r *http.Request) {
	var dto forecast_uc.StatisticalForecastDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := forecast_uc.Execute(dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

package ibpt_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity"
)

func toIBPTRateResponse(r *entity.IBPTRate) *response.IBPTRateResponse {
	if r == nil {
		return nil
	}
	return &response.IBPTRateResponse{
		ID:               r.ID,
		NCM:              r.NCM,
		Ex:               r.Ex,
		UF:               r.UF,
		Tipo:             r.Tipo,
		Descricao:        r.Descricao,
		NacionalFederal:  r.NacionalFederal,
		ImportadoFederal: r.ImportadoFederal,
		Estadual:         r.Estadual,
		Municipal:        r.Municipal,
		VigenciaInicio:   r.VigenciaInicio,
		VigenciaFim:      r.VigenciaFim,
		Chave:            r.Chave,
		Versao:           r.Versao,
		Fonte:            r.Fonte,
		CreatedAt:        r.CreatedAt,
	}
}

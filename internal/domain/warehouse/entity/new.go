package entity

import (
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

func NewWarehouse(
	code string,
	description string,
	location types.TypeLocation,
	types types.TypeWarehouse,
	disposition bool,
	reservationsAllowed bool,
	created_by uuid.UUID,
) (*Warehouse, error) {
	switch {
	case code == "":
		return nil, errors.New("código do almoxarifado é obrigatório")
	case description == "":
		return nil, errors.New("descrição do almoxarifado é obrigatória")
	case !location.IsValid():
		return nil, errors.New("localização deve ser INTERNO, EXTERNO, INSPECAO, REJEICAO, RESERVA, TRANSITO, ESPECIAL, EXPEDICAO ou ASSISTENCIA_TECNICA")
	case !types.IsValid():
		return nil, errors.New("tipo deve ser NORMAL ou LINHA DE PRODUÇÃO")

	case created_by == uuid.Nil:
		return nil, errors.New("usuário responsável é obrigatório")
	}
	return &Warehouse{
		Code:                code,
		Description:         description,
		Location:            location,
		Type:                types,
		Disposition:         disposition,
		ReservationsAllowed: reservationsAllowed,
		CreatedBy:           created_by,
	}, nil
}

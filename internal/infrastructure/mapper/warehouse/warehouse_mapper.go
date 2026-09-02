package mapper

import "github.com/FelipePn10/panossoerp/internal/domain/enums/types"

func WarehouseLocationToDomain(value string) types.TypeLocation {
	values := map[string]types.TypeLocation{
		"INTERNO": types.INTERNO, "EXTERNO": types.EXTERNO,
		"ASSISTENCIA": types.ASSISTENCIA, "ASSISTENCIA_TECNICA": types.ASSISTENCIA_TECNICA,
		"REJEICAO": types.REJEICAO, "INSPECAO": types.INSPECAO,
		"RESERVA": types.RESERVA, "TRANSITO": types.TRANSITO,
		"ESPECIAL": types.ESPECIAL, "EXPEDICAO": types.EXPEDICAO,
	}
	if location, ok := values[value]; ok {
		return location
	}
	return types.INTERNO
}

func WarehouseLocationToDB(value types.TypeLocation) string {
	switch value {
	case types.INTERNO:
		return "INTERNO"
	case types.EXTERNO:
		return "EXTERNO"
	case types.ASSISTENCIA:
		return "ASSISTENCIA"
	case types.REJEICAO:
		return "REJEICAO"
	case types.INSPECAO:
		return "INSPECAO"
	case types.RESERVA:
		return "RESERVA"
	case types.TRANSITO:
		return "TRANSITO"
	case types.ESPECIAL:
		return "ESPECIAL"
	case types.EXPEDICAO:
		return "EXPEDICAO"
	case types.ASSISTENCIA_TECNICA:
		return "ASSISTENCIA_TECNICA"
	default:
		return ""
	}
}

func WarehouseTypeToDomain(value string) types.TypeWarehouse {
	if value == "LINHA_DE_PRODUCAO" {
		return types.LINHA_DE_PRODUCAO
	}
	return types.NORMAL
}

func WarehouseTypeToDB(value types.TypeWarehouse) string {
	if value == types.LINHA_DE_PRODUCAO {
		return "LINHA_DE_PRODUCAO"
	}
	return "NORMAL"
}

package mapper

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

func WarehouseTypeToDomain(t string) types.TypeWarehouse {
	switch t {
	case "INTERNO":
		return types.TypeWarehouse(types.INTERNO)
	case "EXTERNO":
		return types.TypeWarehouse(types.EXTERNO)
	case "ASSISTENCIA":
		return types.TypeWarehouse(types.ASSISTENCIA)
	case "REJEICAO":
		return types.TypeWarehouse(types.REJEICAO)
	case "INSPECAO":
		return types.TypeWarehouse(types.INSPECAO)
	case "RESERVA":
		return types.TypeWarehouse(types.RESERVA)
	case "TRANSITO":
		return types.TypeWarehouse(types.TRANSITO)
	case "ESPECIAL":
		return types.TypeWarehouse(types.ESPECIAL)
	default:
		panic("invalid warehouse type: " + t)
	}
}

func WarehouseTypeToDB(t types.TypeWarehouse) string {
	switch t {
	case types.TypeWarehouse(types.INTERNO):
		return "INTERNO"
	case types.TypeWarehouse(types.EXTERNO):
		return "EXTERNO"
	case types.TypeWarehouse(types.ASSISTENCIA):
		return "ASSISTENCIA"
	case types.TypeWarehouse(types.REJEICAO):
		return "REJEICAO"
	case types.TypeWarehouse(types.INSPECAO):
		return "INSPECAO"
	case types.TypeWarehouse(types.RESERVA):
		return "RESERVA"
	case types.TypeWarehouse(types.TRANSITO):
		return "TRANSITO"
	case types.TypeWarehouse(types.ESPECIAL):
		return "ESPECIAL"
	default:
		panic("invalid warehouse type")
	}
}

func WarehouseLocationToDomain(l string) types.TypeLocation {
	switch l {
	case "LINHA_DE_PRODUCAO":
		return types.TypeLocation(types.LINHA_DE_PRODUCAO)
	case "NORMAL":
		return types.TypeLocation(types.NORMAL)
	default:
		panic("invalid warehouse location: " + l)
	}
}

func WarehouseLocationToDB(l types.TypeLocation) string {
	switch l {
	case types.TypeLocation(types.LINHA_DE_PRODUCAO):
		return "LINHA_DE_PRODUCAO"
	case types.TypeLocation(types.NORMAL):
		return "NORMAL"
	default:
		panic("invalid warehouse location")
	}
}

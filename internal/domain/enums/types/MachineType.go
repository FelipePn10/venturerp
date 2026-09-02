package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

type MachineTypeEnum string

const (
	MachineCut      MachineTypeEnum = "CUT"
	MachineBend     MachineTypeEnum = "BEND"
	MachineWeld     MachineTypeEnum = "WELD"
	MachineAssemble MachineTypeEnum = "ASSEMBLE"
	MachinePaint    MachineTypeEnum = "PAINT"
	MachineLathe    MachineTypeEnum = "LATHE"
	MachineMill     MachineTypeEnum = "MILL"
	MachineInject   MachineTypeEnum = "INJECTION"
	MachinePress    MachineTypeEnum = "PRESS"
)

func (t MachineTypeEnum) IsValid() bool {
	switch t {
	case MachineCut, MachineBend, MachineWeld, MachineAssemble, MachinePaint, MachineLathe, MachineMill, MachineInject, MachinePress:
		return true
	default:
		return false
	}
}

func (t *MachineTypeEnum) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed := MachineTypeEnum(strings.ToUpper(strings.TrimSpace(value)))
	if !parsed.IsValid() {
		return fmt.Errorf("classificação de máquina %q inválida: use uma destas — %s", value, MachineTypeValues())
	}
	*t = parsed
	return nil
}

type MachineCapacityUnit string

const (
	Pieces   MachineCapacityUnit = "PEÇAS"
	Kilogram MachineCapacityUnit = "KG"
	Units    MachineCapacityUnit = "UN"
	Ton      MachineCapacityUnit = "T"
	Sheets   MachineCapacityUnit = "CHAPAS"

	Meters       MachineCapacityUnit = "M"
	SquareMeters MachineCapacityUnit = "M2"
	CubicMeters  MachineCapacityUnit = "M3"
	Liters       MachineCapacityUnit = "LITROS"
)

type CapacityPeriod string

const (
	Minute CapacityPeriod = "MINUTO"
	Hour   CapacityPeriod = "HORA"
	Day    CapacityPeriod = "DIA"
)

// MachineTypeValues lista as classificações aceitas, para as mensagens de erro.
func MachineTypeValues() string {
	values := []MachineTypeEnum{MachineCut, MachineBend, MachineWeld, MachineAssemble,
		MachinePaint, MachineLathe, MachineMill, MachineInject, MachinePress}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, string(v))
	}
	return strings.Join(parts, ", ")
}

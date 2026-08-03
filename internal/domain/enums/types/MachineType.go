package types

import (
	"encoding/json"
	"fmt"
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
	parsed := MachineTypeEnum(value)
	if !parsed.IsValid() {
		return fmt.Errorf("invalid MachineTypeEnum: %s", value)
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

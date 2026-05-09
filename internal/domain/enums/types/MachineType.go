package types

type MachineTypeEnum string

const (
	MachineCut      MachineTypeEnum = "CORTE"
	MachineBend     MachineTypeEnum = "DOBRAR"
	MachineWeld     MachineTypeEnum = "SOLDAR"
	MachineAssemble MachineTypeEnum = "MONTAR"
	MachinePaint    MachineTypeEnum = "PINTAR"
	MachineLathe    MachineTypeEnum = "TORNO"
	MachineMill     MachineTypeEnum = "MOINHO"
	MachineInject   MachineTypeEnum = "INJEÇÃO"
	MachinePress    MachineTypeEnum = "IMPRENSA"
)

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

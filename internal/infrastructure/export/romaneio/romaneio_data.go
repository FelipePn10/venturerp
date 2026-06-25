package romaneio

import "time"

type RomaneioData struct {
	Title    string
	Subtitle string
	Code     int64
	Date     time.Time
	Status   string
	Notes    string

	ReferenceType string
	ReferenceCode int64

	Enterprise   CompanyInfo
	Destinatario CompanyInfo
	Carrier      CarrierInfo

	Items []RomaneioItem

	TotalVolumes int
	TotalWeight  float64
	TotalGross   float64
	TotalNet     float64

	TransportInfo TransportInfo
	GeneratedAt   time.Time

	// Packing detail (volumes / handling units).
	Volumes []RomaneioVolume

	// Identificação fiscal e segurança.
	Seals     string // lacres
	NFeNumber int64
	NFeKey    string

	// Branding for the professional letterhead (filled from the company's
	// fiscal config). Both are optional.
	Logo          []byte // company logo, raw PNG or JPEG
	BrandColorHex string // brand colour as #RRGGBB
}

// RomaneioVolume is one packed handling unit shown in the romaneio's volume table.
type RomaneioVolume struct {
	Number      int
	PackageType string
	NetWeight   float64
	GrossWeight float64
	LengthCm    float64
	WidthCm     float64
	HeightCm    float64
	CubageM3    float64
	Marking     string
	Contents    string
}

type CompanyInfo struct {
	Name     string
	CNPJCPF  string
	IE       string
	Street   string
	Number   string
	District string
	City     string
	UF       string
	CEP      string
	Phone    string
	Email    string
}

type CarrierInfo struct {
	Name        string
	CNPJCPF     string
	Plate       string
	Driver      string
	ANTT        string
	FreightType string
}

type RomaneioItem struct {
	Sequence    int
	ItemCode    int64
	Description string
	Mask        string
	NCM         string
	CFOP        string
	Quantity    float64
	Unit        string
	UnitPrice   float64
	TotalPrice  float64
	WeightNet   float64
	WeightGross float64

	ICMSPct   float64
	ICMSValue float64
	IPIPct    float64
	IPIValue  float64
	PISPct    float64
	COFINSPct float64
	STPct     float64
	STValue   float64

	PhotoURL string
}

type TransportInfo struct {
	FreightType       string
	FreightValue      float64
	InsuranceValue    float64
	VolumeQuantity    float64
	VolumeType        string
	NetWeight         float64
	GrossWeight       float64
	EstimatedDelivery string
}

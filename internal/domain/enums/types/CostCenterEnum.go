package types

// TypeCC represents cost center type
type TypeCC string

const (
	CCAuxiliary      TypeCC = "AUXILIARY"
	CCProductive     TypeCC = "PRODUCTIVE"
	CCAdministrative TypeCC = "ADMINISTRATIVE"
	CCCommercial     TypeCC = "COMMERCIAL"
)

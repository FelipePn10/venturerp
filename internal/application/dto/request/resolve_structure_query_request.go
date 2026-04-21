package request

// Mask vazio → item genérico (consulta sem filtro de máscara).
// Mask preenchido → item configurado (propagação de máscara).
type ResolveStructureQueryDTO struct {
	ItemCode int64
	Mask     string
}

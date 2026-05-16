package entity

import "time"

type PlanoContas struct {
	ID         int64
	Codigo     string
	Descricao  string
	Tipo       string
	Natureza   string
	ParentCode *string
	Nivel      int32
	IsActive   bool
	CreatedAt  time.Time
}

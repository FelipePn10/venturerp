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
	// Conta contábil em que o plano gerencial é contabilizado (contabilização
	// automática da nota de entrada).
	AccountingAccountID *int64
}

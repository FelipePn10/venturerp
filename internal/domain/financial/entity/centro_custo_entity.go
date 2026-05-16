package entity

import "time"

type CentroCusto struct {
	ID        int64
	Codigo    string
	Descricao string
	Tipo      string
	IsActive  bool
	CreatedAt time.Time
}

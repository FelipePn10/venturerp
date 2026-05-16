package entity

import "time"

type CondicaoPagamento struct {
	ID        int64
	Nome      string
	Parcelas  []byte
	Ativo     bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

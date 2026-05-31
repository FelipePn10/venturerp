package entity

import "time"

type Country struct {
	ID        int64
	Sigla     string
	Name      string
	DDI       *string
	BacenCode *string
	SisComex  *string
	IsActive  bool
	CreatedAt time.Time
}

type UF struct {
	ID        int64
	Sigla     string
	Name      string
	CountryID int64
	IBGECode  *string
	IsActive  bool
	CreatedAt time.Time
}

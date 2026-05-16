package entity

import (
	"time"

	"github.com/google/uuid"
)

type FiscalConfig struct {
	ID                       int64
	CnpjEmpresa              string
	RazaoSocial              string
	IEEmpresa                *string
	RegimeTributario         string
	UFEmpresa                string
	IcmsInternoAliquota      float64
	IcmsDiferimentoPercentual float64
	FocusNfeToken            *string
	FocusNfeAmbiente         string
	JurosMes                 float64
	MultaAtraso              float64
	VencimentoIcmsDia        int
	VencimentoIPIDia         int
	VencimentoPisCofinsDia   int
	CreatedAt                time.Time
	UpdatedAt                time.Time
	UpdatedBy                uuid.UUID
}

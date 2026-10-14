package entity

import (
	"time"

	"github.com/google/uuid"
)

// Etapas dos efeitos de NF-e que podem ficar pendentes.
const (
	PendenciaAutorizacao  = "AUTORIZACAO"
	PendenciaCancelamento = "CANCELAMENTO"
)

// PendenciaNFe: efeitos de uma nota já autorizada/cancelada na SEFAZ que ainda
// não entraram no ERP (estoque, reservas, títulos, situação do pedido).
type PendenciaNFe struct {
	FiscalExitID     int64      `json:"fiscal_exit_id"`
	Etapa            string     `json:"etapa"`
	Situacao         string     `json:"situacao"`
	Tentativas       int        `json:"tentativas"`
	UltimoErro       string     `json:"ultimo_erro,omitempty"`
	ProximaTentativa time.Time  `json:"proxima_tentativa"`
	CriadoEm         time.Time  `json:"criado_em"`
	ResolvidoEm      *time.Time `json:"resolvido_em,omitempty"`
}

// PendenciaVencida é um item da fila do processo em segundo plano, com o que
// é preciso para rodar na sessão da empresa dona.
type PendenciaVencida struct {
	FiscalExitID   int64
	Etapa          string
	EnterpriseID   int64
	EnterpriseCode int64
	CriadoPor      *uuid.UUID
}

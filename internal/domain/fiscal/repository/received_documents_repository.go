package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReceivedDocument é uma NF-e emitida contra o CNPJ da empresa, conhecida
// pela distribuição DF-e antes mesmo de o fornecedor mandar o XML.
type ReceivedDocument struct {
	ID            int64
	ChaveAcesso   string
	CNPJEmitente  string
	NomeEmitente  string
	NumeroNF      *int64
	Serie         *string
	DataEmissao   *time.Time
	ValorTotal    decimal.Decimal
	Situacao      string
	Manifestacao  *string
	XMLCompleto   bool
	Versao        int64
	FiscalEntryID *int64
	SyncedAt      time.Time
}

// PrazoManifestacaoDias: a manifestação conclusiva (confirmação,
// desconhecimento ou operação não realizada) vence 180 dias após a emissão.
const PrazoManifestacaoDias = 180

// AlertaPrazoDias: a partir de quantos dias do vencimento a nota entra em alerta.
const AlertaPrazoDias = 30

type ReceivedDocumentsFilter struct {
	SomentePendentes bool // sem nota de entrada lançada
	// SomentePrazo: só as notas sem manifestação conclusiva com o prazo
	// vencendo (até AlertaPrazoDias) ou vencido, na data Hoje.
	SomentePrazo bool
	Hoje         time.Time
	Busca        string
	Limite       int
}

// DFeStatus é a situação da sincronização da empresa.
type DFeStatus struct {
	Automatico      bool
	SincronizadoEm  *time.Time
	UltimaTentativa *time.Time
	UltimoErro      *string
	// Notas sem manifestação conclusiva com prazo vencendo (≤ 30 dias) e vencido.
	PrazoProximo int
	PrazoVencido int
}

// EmpresaDFe é uma empresa reservada para a sincronização agendada.
type EmpresaDFe struct {
	EnterpriseID   int64
	EnterpriseCode int64
	Ator           uuid.UUID
}

type ReceivedDocumentsRepository interface {
	DFeVersion(ctx context.Context) (int64, error)
	// ReservarSincronizacaoDFe marca (atomicamente, valendo para várias
	// instâncias da API) as empresas com sincronização automática ligada,
	// token da Focus e última tentativa há mais de `intervalo`. Não é por
	// empresa: é o agendador que chama, antes de haver sessão.
	ReservarSincronizacaoDFe(ctx context.Context, intervalo time.Duration) ([]EmpresaDFe, error)
	// RegistrarResultadoDFe grava a tentativa (e o erro, ou limpa o erro).
	RegistrarResultadoDFe(ctx context.Context, erro *string) error
	DFeStatus(ctx context.Context, hoje time.Time) (*DFeStatus, error)
	SetDFeAutomatico(ctx context.Context, ativo bool) error
	// UpsertReceivedDocuments grava os documentos e avança a versão da empresa.
	UpsertReceivedDocuments(ctx context.Context, docs []ReceivedDocument, maxVersao int64) (int, error)
	ListReceivedDocuments(ctx context.Context, f ReceivedDocumentsFilter) ([]ReceivedDocument, error)
	SetManifestacao(ctx context.Context, chave, tipo string) error
}

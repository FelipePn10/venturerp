package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
)

// ReformaTributariaRepository guarda a parametrização da reforma tributária:
// a tabela oficial de cClassTrib (nacional, só leitura), as alíquotas por ano e
// o cClassTrib padrão por CFOP (ambos da empresa da sessão), e o retrato do
// grupo IBS/CBS de cada nota transmitida.
type ReformaTributariaRepository interface {
	// AliquotasDoAno devolve o cadastro da empresa para o ano, ou nil.
	AliquotasDoAno(ctx context.Context, ano int) (*reforma.Aliquotas, error)
	ListarAliquotas(ctx context.Context) ([]reforma.AliquotasCadastradas, error)
	SalvarAliquotas(ctx context.Context, ano int, a reforma.Aliquotas, fundamento string, actor uuid.UUID) error
	ExcluirAliquotas(ctx context.Context, ano int) error

	// ListarClassificacoes filtra a tabela oficial por CST e por texto.
	ListarClassificacoes(ctx context.Context, cst, busca string, somenteNFe bool) ([]reforma.ClassificacaoOficial, error)
	BuscarClassificacoes(ctx context.Context, codigos []string) (map[string]reforma.ClassificacaoOficial, error)

	ListarCFOP(ctx context.Context) ([]reforma.CFOPClassificacao, error)
	SalvarCFOP(ctx context.Context, cfop, classTrib string, observacao *string, actor uuid.UUID) error
	ExcluirCFOP(ctx context.Context, cfop string) error

	// CandidatosDaNota devolve, por linha da nota da empresa, o cClassTrib do
	// item, do CFOP e da classificação fiscal vigente na emissão.
	CandidatosDaNota(ctx context.Context, exitID int64) (map[int64]reforma.Candidatos, error)
	// DefinirClassTribDoItem grava (ou limpa, com nil) o cClassTrib próprio de
	// uma linha de nota ainda não transmitida.
	DefinirClassTribDoItem(ctx context.Context, exitID, itemID int64, classTrib *string) error
	// Parametros devolve o modo da empresa (sem cadastro: automático).
	Parametros(ctx context.Context) (reforma.Parametros, error)
	SalvarParametros(ctx context.Context, p reforma.Parametros, actor uuid.UUID) error

	// GravarRetrato registra o que foi transmitido, linha a linha, e os totais.
	GravarRetrato(ctx context.Context, exitID int64, itens []reforma.RetratoItem, totalIBS, totalCBS decimal.Decimal) error
}

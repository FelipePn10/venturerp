package lote

import (
	"context"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
)

// Filtro da lista de lotes.
type Filtro struct {
	Status            string
	Origem            string
	ItemCode          int64
	SupplierCode      int64
	ProductionOrderID int64
	Limite            int
}

// AtualizacaoDaAmostragem é a troca de severidade aplicada junto com a
// decisão, na mesma transação.
type AtualizacaoDaAmostragem struct {
	ProcedimentoID int64
	ItemCode       int64
	Chave          string
	Referencia     string
	Fn             amostragem.Atualizacao
}

// NaoConformidade aberta pela decisão que não aproveita o lote inteiro.
type NaoConformidade struct {
	Descricao  string
	Quantidade string // decimal em texto
	Severidade string // CRITICA, MAIOR, MENOR
}

// Preparacao valida a decisão contra o lote travado e devolve os efeitos.
type Preparacao func(l *Lote, almox map[int64]Almoxarifado) (*AtualizacaoDaAmostragem, *NaoConformidade, error)

// Repository do lote de inspeção, sempre na empresa da sessão.
type Repository interface {
	// Abrir grava o lote (numerado por empresa) com as características.
	Abrir(ctx context.Context, l *Lote) error
	Obter(ctx context.Context, id int64) (*Lote, error)
	// ObterPorLegado resolve o id antigo (ordem de recebimento, registro de
	// qualidade, registro de procurement) das rotas de compatibilidade.
	ObterPorLegado(ctx context.Context, kind string, id int64) (*Lote, error)
	Listar(ctx context.Context, f Filtro) ([]Lote, error)
	// GravarResultados grava (ou corrige) resultados de um lote aberto.
	GravarResultados(ctx context.Context, loteID int64, rs []Resultado, ator uuid.UUID) error
	// Decidir trava o lote, chama preparar com ele e os almoxarifados da
	// empresa (validação e efeitos calculados sobre o estado travado) e grava a
	// decisão, o estoque, a amostragem e a NC numa transação só.
	Decidir(ctx context.Context, loteID int64, d *Decisao, ator uuid.UUID, preparar Preparacao) (*Lote, error)
	Cancelar(ctx context.Context, id int64, motivo string, ator uuid.UUID) error
	Almoxarifados(ctx context.Context, ids []int64) (map[int64]Almoxarifado, error)
	VincularRelatorio(ctx context.Context, loteID, relatorioID int64, ator uuid.UUID) error
	DesvincularRelatorio(ctx context.Context, loteID, relatorioID int64) error
	Relatorios(ctx context.Context, loteID int64) ([]int64, error)
}

// Fonte é o plano de onde o lote copia as características.
type Fonte struct {
	RouteID         *int64
	PlanID          *int64
	AlmoxInspecao   *int64
	ProcedimentoID  *int64
	RouteOperation  *int64
	Caracteristicas []Caracteristica
}

// Fontes encontra o plano do lote na empresa da sessão.
type Fontes interface {
	// RoteiroDeRecebimento: roteiro ativo e vigente do item (máscara exata
	// antes da genérica); nil quando o item não tem roteiro.
	RoteiroDeRecebimento(ctx context.Context, item int64, mask string) (*Fonte, error)
	PlanoDeInspecao(ctx context.Context, planoID int64) (*Fonte, error)
}

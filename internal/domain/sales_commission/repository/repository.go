package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_commission/entity"
	"github.com/shopspring/decimal"
)

// Totais são os dois números sobre os quais a comissão pode incidir. Vêm da
// capa do documento para o cálculo não depender da tela.
type Totais struct {
	TotalProdutos decimal.Decimal
	TotalLiquido  decimal.Decimal
	// Representante e percentual da CAPA. Servem para a tela abrir com o que o
	// documento já tem quando o rateio ainda não foi montado — documento sem
	// rateio nenhum é o que fazia o relatório de comissão ler duas fontes.
	RepresentativeCode *int64
	CommissionPct      decimal.Decimal
}

type Repository interface {
	// Listar devolve o rateio do documento com o nome do representante.
	Listar(ctx context.Context, doc entity.Documento, documentCode int64) ([]*entity.Rateio, error)
	// Substituir grava o rateio inteiro numa transação e espelha o principal na
	// capa do documento, para os relatórios antigos continuarem certos.
	Substituir(ctx context.Context, doc entity.Documento, documentCode int64, linhas []*entity.Rateio) ([]*entity.Rateio, error)
	// Totais lê a capa do documento (e valida que ele é da empresa da sessão).
	Totais(ctx context.Context, doc entity.Documento, documentCode int64) (Totais, error)
	// RepresentantesAtivos diz quais dos códigos existem e estão ativos.
	RepresentantesAtivos(ctx context.Context, codigos []int64) (map[int64]string, error)
}

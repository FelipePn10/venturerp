package mrp_calculation_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
)

// ChaveEstruturaSemConversao: no log do cálculo, as linhas de estrutura cuja
// unidade não converte para a de estoque atual do componente.
const ChaveEstruturaSemConversao = "estrutura_sem_conversao"

// MRPComEstruturaEmDia recalcula a quantidade das linhas de estrutura na
// unidade de estoque atual antes de cada cálculo — o cálculo do plano e o do
// cenário passam por aqui. É a rede de segurança: unidade ou conversão podem
// mudar por caminhos que não avisam a estrutura (vigência futura que começa a
// valer, carga direta no banco), e o MRP não pode comprar 4.160 m de tubo por
// um fator velho. A linha que não converte vai para o log do cálculo.
type MRPComEstruturaEmDia struct {
	mrpservice.MRPService
	Estruturas ports.RecalculoDeEstrutura
	// Log grava o aviso no log guardado do cálculo (o que o impresso do plano
	// de materiais e a consulta posterior leem). Sem ele, o aviso só vai na
	// resposta.
	Log AnotadorDoLog
}

// AnotadorDoLog junta uma chave aos erros de um cálculo já encerrado.
type AnotadorDoLog interface {
	AnotarNoLog(ctx context.Context, logCode int64, chave string, valor any) error
}

// ChaveLogNaoAnotado: o aviso não chegou ao log guardado (o cálculo valeu).
const ChaveLogNaoAnotado = "estrutura_sem_conversao_nao_gravado"

func (m *MRPComEstruturaEmDia) Calculate(ctx context.Context, planCode, initialOrderNumber int64, generateLLC bool) (*entity.MRPCalculationLog, error) {
	var avisos []string
	if m.Estruturas != nil {
		_, sem, err := m.Estruturas.RecalcularDaEmpresa(ctx, "conferência antes do cálculo do MRP")
		if err != nil {
			return nil, fmt.Errorf("conferindo as unidades das estruturas antes do MRP: %w", err)
		}
		avisos = sem
	}
	log, err := m.MRPService.Calculate(ctx, planCode, initialOrderNumber, generateLLC)
	if err != nil || log == nil || len(avisos) == 0 {
		return log, err
	}
	if log.Errors == nil {
		log.Errors = map[string]interface{}{}
	}
	log.Errors[ChaveEstruturaSemConversao] = avisos
	if m.Log != nil && log.Code > 0 {
		// O cálculo já valeu e está gravado: falhar aqui desfaria nada e
		// esconderia o resultado. A falha vai na resposta, ao lado do aviso.
		if err := m.Log.AnotarNoLog(ctx, log.Code, ChaveEstruturaSemConversao, avisos); err != nil {
			log.Errors[ChaveLogNaoAnotado] = err.Error()
		}
	}
	return log, nil
}

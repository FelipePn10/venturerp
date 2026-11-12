package impressao_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/google/uuid"
)

// Tenant é a empresa autenticada: as tabelas comerciais usam o código
// (enterprise_code) e as demais o id (enterprise_id).
type Tenant struct{ ID, Code int64 }

// Consultas lê os modelos de impressão. Implementada com SQL na
// infraestrutura; toda consulta filtra pela empresa.
type Consultas interface {
	Emitente(ctx context.Context, t Tenant) (Emitente, error)
	NomeUsuario(ctx context.Context, id uuid.UUID) string
	PedidoVenda(ctx context.Context, t Tenant, code int64) (*PedidoVenda, error)
	NotaEntrada(ctx context.Context, t Tenant, id int64) (*NotaEntrada, error)
	Titulos(ctx context.Context, t Tenant, receber bool, f FiltroTitulos) ([]Titulo, error)
	ContasBancarias(ctx context.Context, t Tenant) ([]ContaBancaria, error)
	PlanoMateriais(ctx context.Context, t Tenant, plano int64) (*PlanoMateriais, error)
	Capacidade(ctx context.Context, t Tenant, plano int64) (*Capacidade, error)
	Programacao(ctx context.Context, t Tenant, de, ate time.Time, grupo *int64) (*Programacao, error)
	Kardex(ctx context.Context, t Tenant, f FiltroKardex) (*Kardex, error)
	CustoProducao(ctx context.Context, t Tenant, ordens []int64, de, ate *time.Time) ([]CustoOrdem, error)
	Romaneio(ctx context.Context, t Tenant, code int64) (*Romaneio, error)
	Proposta(ctx context.Context, t Tenant, code int64) (*Proposta, error)
	Cotacao(ctx context.Context, t Tenant, code int64) (*Cotacao, error)
	FichaCliente(ctx context.Context, t Tenant, code int64) (*FichaCliente, error)
}

// Gerador desenha cada documento com o layout próprio dele.
type Gerador interface {
	PedidoVenda(d PedidoVenda) []byte
	NotaEntrada(d NotaEntrada) []byte
	Titulos(d Titulos) []byte
	PlanoMateriais(d PlanoMateriais) []byte
	Capacidade(d Capacidade) []byte
	Programacao(d Programacao) []byte
	Kardex(d Kardex) []byte
	CustoProducao(d CustoProducao) []byte
	Romaneio(d Romaneio) []byte
	Proposta(d Proposta) []byte
	SolicitacaoCotacao(d Cotacao) []byte
	MapaCotacao(d Cotacao) []byte
	FichaCliente(d FichaCliente) []byte
}

// Arquivo é o PDF pronto com o nome sugerido.
type Arquivo struct {
	Nome  string
	Bytes []byte
}

type UseCase struct {
	Consultas Consultas
	Gerador   Gerador
	Auth      ports.AuthService
	Agora     func() time.Time
}

func (uc *UseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

// preparar valida a permissão e monta o cabeçalho comum.
func (uc *UseCase) preparar(ctx context.Context, pode bool) (Tenant, Cabecalho, error) {
	if !pode {
		return Tenant{}, Cabecalho{}, errorsuc.ErrUnauthorized
	}
	id, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return Tenant{}, Cabecalho{}, errorsuc.ErrUnauthorized
	}
	code, err := uc.Auth.EnterpriseCode(ctx)
	if err != nil {
		return Tenant{}, Cabecalho{}, errorsuc.ErrUnauthorized
	}
	t := Tenant{ID: id, Code: code}
	em, err := uc.Consultas.Emitente(ctx, t)
	if err != nil {
		return t, Cabecalho{}, err
	}
	cab := Cabecalho{Emitente: em, Gerado: uc.agora()}
	if user, err := uc.Auth.UserID(ctx); err == nil {
		cab.EmitidoPor = uc.Consultas.NomeUsuario(ctx, user)
	}
	return t, cab, nil
}

func naoEncontrado(oQue string) error {
	return errorsuc.NewValidationError(oQue + " não encontrado")
}

func (uc *UseCase) PedidoVenda(ctx context.Context, code int64) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanGetSalesOrder(ctx))
	if err != nil {
		return nil, err
	}
	d, err := uc.Consultas.PedidoVenda(ctx, t, code)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, naoEncontrado("pedido de venda")
	}
	d.Cabecalho = cab
	d.Parcelas = parcelasDoPedido(d)
	return &Arquivo{fmt.Sprintf("pedido-venda-%d.pdf", d.Numero), uc.Gerador.PedidoVenda(*d)}, nil
}

func (uc *UseCase) NotaEntrada(ctx context.Context, id int64) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanGetFiscalEntry(ctx))
	if err != nil {
		return nil, err
	}
	d, err := uc.Consultas.NotaEntrada(ctx, t, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, naoEncontrado("nota de entrada")
	}
	d.Cabecalho = cab
	return &Arquivo{fmt.Sprintf("nf-entrada-%d.pdf", d.Numero), uc.Gerador.NotaEntrada(*d)}, nil
}

func (uc *UseCase) Titulos(ctx context.Context, receber bool, f FiltroTitulos) (*Arquivo, error) {
	pode := uc.Auth.CanListContasPagar(ctx)
	if receber {
		pode = uc.Auth.CanListContasReceber(ctx)
	}
	t, cab, err := uc.preparar(ctx, pode)
	if err != nil {
		return nil, err
	}
	lista, err := uc.Consultas.Titulos(ctx, t, receber, f)
	if err != nil {
		return nil, err
	}
	d := Titulos{Cabecalho: cab, Receber: receber, Titulos: lista, Referencia: cab.Gerado, Filtro: descreverFiltro(f)}
	if receber {
		if d.Contas, err = uc.Consultas.ContasBancarias(ctx, t); err != nil {
			return nil, err
		}
	}
	nome := "contas-a-pagar.pdf"
	if receber {
		nome = "contas-a-receber.pdf"
	}
	return &Arquivo{nome, uc.Gerador.Titulos(d)}, nil
}

func descreverFiltro(f FiltroTitulos) string {
	var p []string
	if f.De != nil || f.Ate != nil {
		de, ate := "início", "hoje em diante"
		if f.De != nil {
			de = f.De.Format("02/01/2006")
		}
		if f.Ate != nil {
			ate = f.Ate.Format("02/01/2006")
		}
		p = append(p, "vencimento de "+de+" a "+ate)
	}
	if f.SomenteAbertos {
		p = append(p, "somente em aberto")
	}
	if len(p) == 0 {
		return "todos os títulos"
	}
	return strings.Join(p, " · ")
}

func (uc *UseCase) PlanoMateriais(ctx context.Context, plano int64) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanRunMRPCalculation(ctx) || uc.Auth.CanListProductionPlans(ctx))
	if err != nil {
		return nil, err
	}
	d, err := uc.Consultas.PlanoMateriais(ctx, t, plano)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, naoEncontrado("plano de produção")
	}
	d.Cabecalho = cab
	return &Arquivo{fmt.Sprintf("plano-materiais-%d.pdf", plano), uc.Gerador.PlanoMateriais(*d)}, nil
}

func (uc *UseCase) Capacidade(ctx context.Context, plano int64) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanRunMRPCalculation(ctx) || uc.Auth.CanListProductionPlans(ctx))
	if err != nil {
		return nil, err
	}
	d, err := uc.Consultas.Capacidade(ctx, t, plano)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, naoEncontrado("plano de produção")
	}
	d.Cabecalho = cab
	return &Arquivo{fmt.Sprintf("capacidade-plano-%d.pdf", plano), uc.Gerador.Capacidade(*d)}, nil
}

func (uc *UseCase) Programacao(ctx context.Context, de, ate time.Time, grupo *int64) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanRunMRPCalculation(ctx) || uc.Auth.CanReportProduction(ctx))
	if err != nil {
		return nil, err
	}
	if ate.Before(de) {
		return nil, errorsuc.NewValidationError("a data final é anterior à inicial")
	}
	d, err := uc.Consultas.Programacao(ctx, t, de, ate, grupo)
	if err != nil {
		return nil, err
	}
	d.Cabecalho, d.De, d.Ate = cab, de, ate
	return &Arquivo{"programacao-producao.pdf", uc.Gerador.Programacao(*d)}, nil
}

func (uc *UseCase) Kardex(ctx context.Context, f FiltroKardex) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanListStockMovements(ctx))
	if err != nil {
		return nil, err
	}
	if f.Ate.Before(f.De) {
		return nil, errorsuc.NewValidationError("a data final é anterior à inicial")
	}
	d, err := uc.Consultas.Kardex(ctx, t, f)
	if err != nil {
		return nil, err
	}
	d.Cabecalho, d.De, d.Ate = cab, f.De, f.Ate
	return &Arquivo{"kardex.pdf", uc.Gerador.Kardex(*d)}, nil
}

func (uc *UseCase) CustoProducao(ctx context.Context, ordens []int64, de, ate *time.Time) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanReportProduction(ctx) || uc.Auth.CanRunMRPCalculation(ctx))
	if err != nil {
		return nil, err
	}
	lista, err := uc.Consultas.CustoProducao(ctx, t, ordens, de, ate)
	if err != nil {
		return nil, err
	}
	filtro := "ordens encerradas"
	switch {
	case len(ordens) > 0:
		filtro = fmt.Sprintf("%d ordem(ns) selecionada(s)", len(ordens))
	case de != nil && ate != nil:
		filtro = "encerradas de " + de.Format("02/01/2006") + " a " + ate.Format("02/01/2006")
	}
	return &Arquivo{"custo-producao.pdf", uc.Gerador.CustoProducao(CustoProducao{Cabecalho: cab, Filtro: filtro, Ordens: lista})}, nil
}

func (uc *UseCase) Romaneio(ctx context.Context, code int64) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanGetSalesOrder(ctx) || uc.Auth.CanListStockMovements(ctx))
	if err != nil {
		return nil, err
	}
	d, err := uc.Consultas.Romaneio(ctx, t, code)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, naoEncontrado("romaneio")
	}
	d.Cabecalho = cab
	return &Arquivo{fmt.Sprintf("romaneio-%d.pdf", d.Codigo), uc.Gerador.Romaneio(*d)}, nil
}

func (uc *UseCase) Proposta(ctx context.Context, code int64) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanGetSalesOrder(ctx))
	if err != nil {
		return nil, err
	}
	d, err := uc.Consultas.Proposta(ctx, t, code)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, naoEncontrado("orçamento")
	}
	d.Cabecalho = cab
	pv := d.comoPedido()
	d.Parcelas = parcelasDoPedido(&pv)
	return &Arquivo{fmt.Sprintf("proposta-%d.pdf", d.Numero), uc.Gerador.Proposta(*d)}, nil
}

// Cotacao imprime a solicitação para um fornecedor (fornecedor > 0) ou o mapa
// comparativo de todos os fornecedores convidados.
func (uc *UseCase) Cotacao(ctx context.Context, code, fornecedor int64) (*Arquivo, error) {
	t, cab, err := uc.preparar(ctx, uc.Auth.CanGetPurchaseOrder(ctx) || uc.Auth.CanListPurchaseOrders(ctx))
	if err != nil {
		return nil, err
	}
	d, err := uc.Consultas.Cotacao(ctx, t, code)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, naoEncontrado("cotação")
	}
	d.Cabecalho = cab
	d.Comprador = cab.EmitidoPor
	if fornecedor <= 0 {
		return &Arquivo{fmt.Sprintf("mapa-cotacao-%d.pdf", code), uc.Gerador.MapaCotacao(*d)}, nil
	}
	for i := range d.Fornecedores {
		if d.Fornecedores[i].Codigo == fornecedor {
			d.Fornecedor = &d.Fornecedores[i]
			return &Arquivo{fmt.Sprintf("cotacao-%d-fornecedor-%d.pdf", code, fornecedor), uc.Gerador.SolicitacaoCotacao(*d)}, nil
		}
	}
	return nil, errorsuc.NewValidationError(fmt.Sprintf("o fornecedor %d não foi convidado para a cotação %d", fornecedor, code))
}

func (uc *UseCase) FichaCliente(ctx context.Context, code int64) (*Arquivo, error) {
	// Como a ficha anterior: o cadastro do cliente é visível a quem a rota
	// admite (ADMIN/USER); não há permissão específica de cliente.
	t, cab, err := uc.preparar(ctx, true)
	if err != nil {
		return nil, err
	}
	d, err := uc.Consultas.FichaCliente(ctx, t, code)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, naoEncontrado("cliente")
	}
	d.Cabecalho = cab
	return &Arquivo{fmt.Sprintf("ficha-cliente-%d.pdf", code), uc.Gerador.FichaCliente(*d)}, nil
}

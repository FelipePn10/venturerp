package fiscal_uc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialRepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

type notaFalsa struct {
	fiscalrepo.FiscalRepository
	exit  *fiscalentity.FiscalExit
	itens []*fiscalentity.FiscalExitItem
}

func (n *notaFalsa) GetExitByID(context.Context, int64) (*fiscalentity.FiscalExit, error) {
	return n.exit, nil
}
func (n *notaFalsa) GetExitItems(context.Context, int64) ([]*fiscalentity.FiscalExitItem, error) {
	return n.itens, nil
}

type pendenciasFalsas struct {
	fiscalrepo.PendenciasNFeRepository
	resolvidas []string
	falhas     []string
}

func (p *pendenciasFalsas) Resolver(_ context.Context, _ int64, etapa string) error {
	p.resolvidas = append(p.resolvidas, etapa)
	return nil
}
func (p *pendenciasFalsas) RegistrarFalha(_ context.Context, _ int64, _ string, motivo string) error {
	p.falhas = append(p.falhas, motivo)
	return nil
}

// depósito padrão por item (0 = sem depósito no cadastro).
type itensFalsos map[int64]int

func (f itensFalsos) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	dep, ok := f[int64(code)]
	if !ok {
		return nil, itemrepo.ErrNotFound
	}
	it := &itementity.Item{}
	it.Warehouse.WarehouseCode = dep
	return it, nil
}

type clientesFalsos struct {
	customerrepo.CustomerRepository
}

func (clientesFalsos) GetCustomerByCode(_ context.Context, code int64) (*customerentity.Customer, error) {
	// id do cadastro diferente do código, como em produção depois da cópia por empresa
	return &customerentity.Customer{ID: 9001, Code: code}, nil
}

type titulosCapturados struct {
	financialRepo.FinancialRepository
	titulos []*financialEntity.ContaReceber
}

func (t *titulosCapturados) CreateContaReceber(_ context.Context, cr *financialEntity.ContaReceber) (*financialEntity.ContaReceber, error) {
	t.titulos = append(t.titulos, cr)
	return cr, nil
}

type estoqueQueFalha struct{ fakeStockRepo }

func (estoqueQueFalha) CreateMovement(context.Context, *stockentity.StockMovement) (*stockentity.StockMovement, error) {
	return nil, errors.New("saldo bloqueado")
}

func efeitosDeTeste(exit *fiscalentity.FiscalExit, itens []*fiscalentity.FiscalExitItem, repos EfeitosRepos, itensCad itensFalsos) (*EfeitosNFe, *pendenciasFalsas) {
	pend := &pendenciasFalsas{}
	repos.Pendencias = pend
	return &EfeitosNFe{
		Fiscal:     &notaFalsa{exit: exit, itens: itens},
		Orders:     repos.Pedidos,
		Itens:      itensCad,
		Pendencias: pend,
		Transacao:  func(_ context.Context, fn func(EfeitosRepos) error) error { return fn(repos) },
	}, pend
}

// Linha do pedido com depósito manda; sem ela vale o depósito padrão do item;
// só a reserva do pedido da nota é consumida e o pedido fica faturado.
func TestEfeitosAutorizacao_BaixaPeloDepositoDaLinhaOuPadrao(t *testing.T) {
	stock := &fakeStockRepo{reservations: []*stockentity.StockReservation{
		{ID: 42, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: 500},
		{ID: 99, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: 777},
	}}
	sales := &fakeSalesRepo{items: []*salesentity.SalesOrderItem{
		{ItemCode: 1001, WarehouseCode: i64p(7)},
		{ItemCode: 1002}, // sem depósito na linha → padrão do item
	}}
	exit := &fiscalentity.FiscalExit{ID: 99, SalesOrderCode: i64p(500), DataEmissao: time.Now()}
	itens := []*fiscalentity.FiscalExitItem{
		{ItemCode: i64p(1001), Quantity: 5},
		{ItemCode: i64p(1002), Quantity: 3},
		{ItemCode: nil, Quantity: 9}, // serviço/sem item: não movimenta
	}
	ef, pend := efeitosDeTeste(exit, itens, EfeitosRepos{Estoque: stock, Pedidos: sales}, itensFalsos{1002: 8})

	if err := ef.Executar(context.Background(), 99, fiscalentity.PendenciaAutorizacao, uuid.New()); err != nil {
		t.Fatal(err)
	}
	porItem := map[int64]int64{}
	for _, m := range stock.movements {
		if m.MovementType != stockentity.MovementTypeOut || m.ReferenceType == nil || *m.ReferenceType != stockentity.ReferenceTypeNFExit {
			t.Fatalf("movimento inesperado: %+v", m)
		}
		porItem[m.ItemCode] = m.WarehouseID
	}
	if len(stock.movements) != 2 || porItem[1001] != 7 || porItem[1002] != 8 {
		t.Fatalf("depósitos = %v, quer 1001→7 (linha) e 1002→8 (padrão)", porItem)
	}
	if len(stock.consumed) != 1 || stock.consumed[0] != 42 {
		t.Fatalf("reservas consumidas = %v, quer [42]", stock.consumed)
	}
	if sales.changed[500] != salesentity.SalesOrderStatusInvoiced {
		t.Fatalf("pedido 500 = %v, quer faturado", sales.changed[500])
	}
	if len(pend.resolvidas) != 1 || len(pend.falhas) != 0 {
		t.Fatalf("pendência: resolvidas=%v falhas=%v", pend.resolvidas, pend.falhas)
	}
}

// Sem depósito na linha nem no cadastro do item a autorização é recusada antes
// de transmitir: depois de autorizada, a nota não volta.
func TestEfeitosPreConferencia_ItemSemDepositoRecusa(t *testing.T) {
	exit := &fiscalentity.FiscalExit{ID: 1}
	itens := []*fiscalentity.FiscalExitItem{{ItemCode: i64p(1003), Description: strp("Chapa 3mm"), Quantity: 1}}
	ef, _ := efeitosDeTeste(exit, itens, EfeitosRepos{}, itensFalsos{1003: 0})

	err := ef.PreConferirDepositos(context.Background(), exit, itens)
	if _, ok := errorsuc.AsValidation(err); !ok || !strings.Contains(err.Error(), "Chapa 3mm") {
		t.Fatalf("erro = %v, quer validação citando o item", err)
	}
}

// Falha no meio (a baixa) devolve o erro, guarda o motivo na pendência e não a
// resolve — a transação real desfaz o que já entrou.
func TestEfeitosAutorizacao_FalhaFicaPendente(t *testing.T) {
	exit := &fiscalentity.FiscalExit{ID: 5, DataEmissao: time.Now()}
	itens := []*fiscalentity.FiscalExitItem{{ItemCode: i64p(1001), Quantity: 1}}
	ef, pend := efeitosDeTeste(exit, itens, EfeitosRepos{Estoque: &estoqueQueFalha{}}, itensFalsos{1001: 3})

	err := ef.Executar(context.Background(), 5, fiscalentity.PendenciaAutorizacao, uuid.New())
	if err == nil || !strings.Contains(err.Error(), "saldo bloqueado") {
		t.Fatalf("erro = %v, quer a falha da baixa", err)
	}
	if len(pend.resolvidas) != 0 || len(pend.falhas) != 1 {
		t.Fatalf("pendência: resolvidas=%v falhas=%v, quer só a falha registrada", pend.resolvidas, pend.falhas)
	}
}

// O título a receber guarda o CÓDIGO do cliente: é por ele que Contas a
// Receber filtra e a análise de crédito soma. Gravar o id fazia o título sumir.
func TestEfeitosAutorizacao_TituloGuardaOCodigoDoCliente(t *testing.T) {
	fin := &titulosCapturados{}
	cliente := int64(77)
	exit := &fiscalentity.FiscalExit{ID: 55, NumeroNF: 10, CustomerCode: &cliente, ValorTotal: 100, DataEmissao: time.Now()}
	ef, _ := efeitosDeTeste(exit, nil, EfeitosRepos{Financeiro: fin}, nil)
	ef.Customers = clientesFalsos{}
	ef.GeraTitulos = true

	if err := ef.Executar(context.Background(), 55, fiscalentity.PendenciaAutorizacao, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(fin.titulos) != 1 || fin.titulos[0].ClienteID == nil || *fin.titulos[0].ClienteID != 77 {
		t.Fatalf("cliente do título = %v, quer o código 77", fin.titulos)
	}
}

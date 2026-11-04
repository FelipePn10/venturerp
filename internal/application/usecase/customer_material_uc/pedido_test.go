package customer_material_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
)

// pedidosFalsos devolve os pedidos cadastrados no teste; o resto é "não
// encontrado", como o pedido de outra empresa.
type pedidosFalsos map[int64]*entity.PedidoDeBeneficiamento

func (p pedidosFalsos) VincularPedidoDaOF(context.Context, int64, *int64) error { return nil }

func (p pedidosFalsos) PedidoDeBeneficiamento(_ context.Context, code int64, _ int64) (*entity.PedidoDeBeneficiamento, error) {
	if pedido, ok := p[code]; ok {
		return pedido, nil
	}
	return nil, errorsuc.NewNotFoundError("pedido não encontrado")
}

// O pedido 5001 do cliente 100: uma linha de serviço, R$ 230,00, 1 PC.
func pedidoDoCliente100() pedidosFalsos {
	return pedidosFalsos{
		5001: {Code: 5001, Numero: 90, CustomerCode: 100, Status: "P", Ativo: true, Linhas: []entity.LinhaDoPedido{{
			Code: 71, Sequencia: 1, ItemCode: 9, ItemBusinessCode: "25027532", Descricao: "SERVICO INDUSTRIALIZACAO",
			Unidade: "PC", Quantidade: dec("1"), PrecoUnitario: dec("230"),
		}}},
		6001: {Code: 6001, Numero: 91, CustomerCode: 200, Status: "P", Ativo: true},
	}
}

func esperaValidacao(t *testing.T, err error, trecho string) {
	t.Helper()
	if err == nil {
		t.Fatalf("esperado erro contendo %q", trecho)
	}
	if _, ok := errorsuc.AsValidation(err); !ok || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("esperado erro de validação com %q, veio %v", trecho, err)
	}
}

func TestReceberRecusaPedidoInexistenteOuDeOutroCliente(t *testing.T) {
	uc, repo := novoUC()
	uc.Pedidos = pedidoDoCliente100()

	nova := remessaBase()
	inexistente := int64(9999)
	nova.SalesOrderCode = &inexistente
	_, err := uc.Receber(context.Background(), nova, autorValido)
	esperaValidacao(t, err, "não encontrado")

	deOutro := int64(6001)
	nova.SalesOrderCode = &deOutro
	_, err = uc.Receber(context.Background(), nova, autorValido)
	esperaValidacao(t, err, "outro cliente")

	if repo.registrada != nil {
		t.Fatal("remessa com pedido inválido não pode ser gravada")
	}
}

func TestReceberComPedidoValidoEntraLiberado(t *testing.T) {
	uc, repo := novoUC()
	uc.Pedidos = pedidoDoCliente100()
	if _, err := uc.Receber(context.Background(), remessaBase(), autorValido); err != nil {
		t.Fatal(err)
	}
	if repo.registrada.Blocked {
		t.Fatal("pedido conferido: o material entra liberado")
	}
}

func TestReceberRecusaPedidoCanceladoOuFaturado(t *testing.T) {
	for status, trecho := range map[string]string{entity.PedidoCancelado: "cancelado", entity.PedidoFaturado: "faturado"} {
		uc, _ := novoUC()
		pedidos := pedidoDoCliente100()
		pedidos[5001].Status = status
		uc.Pedidos = pedidos
		_, err := uc.Receber(context.Background(), remessaBase(), autorValido)
		esperaValidacao(t, err, trecho)
	}
}

func TestVincularPedidoLiberaSoOBloqueioPorFaltaDePedido(t *testing.T) {
	semPedido := MotivoSemPedido
	divergente := "quantidade conferida diferente da NF-e; regularize antes de produzir"
	for motivo, libera := range map[*string]bool{&semPedido: true, &divergente: false} {
		repo := &repoFalso{remessa: &entity.Remessa{ID: 1, CustomerCode: 100, Status: entity.StatusAberta, Blocked: true, BlockReason: motivo}}
		uc := &UseCase{Repo: repo, Pedidos: pedidoDoCliente100()}
		if _, err := uc.VincularPedido(context.Background(), 1, 5001, autorValido); err != nil {
			t.Fatal(err)
		}
		if repo.pedidoVinculado != 5001 || repo.desbloqueou != libera {
			t.Fatalf("motivo %q: vinculado=%d desbloqueou=%v (esperado %v)", *motivo, repo.pedidoVinculado, repo.desbloqueou, libera)
		}
	}
}

func TestVincularPedidoDeOutroClienteERecusado(t *testing.T) {
	repo := &repoFalso{remessa: &entity.Remessa{ID: 1, CustomerCode: 100, Status: entity.StatusAberta}}
	uc := &UseCase{Repo: repo, Pedidos: pedidoDoCliente100()}
	_, err := uc.VincularPedido(context.Background(), 1, 6001, autorValido)
	esperaValidacao(t, err, "outro cliente")
	if repo.pedidoVinculado != 0 {
		t.Fatal("nada pode ser gravado")
	}
}

// O faturamento traz o serviço do pedido e liga a linha da nota à linha do
// pedido: é a autorização que soma ao atendido e fecha o pedido.
func TestFaturarTrazOServicoDoPedido(t *testing.T) {
	uc, repo, notas := montarFaturamento()
	pedido := int64(5001)
	repo.remessa.SalesOrderCode = &pedido
	uc.Pedidos = pedidoDoCliente100()

	req := pedidoPadrao()
	req.Servico = ServicoFaturado{} // a tela não mandou nada: vem do pedido
	if _, err := uc.Faturar(context.Background(), req, autorValido); err != nil {
		t.Fatal(err)
	}
	servico := notas.itens[0]
	if servico.Cfop != "5124" || !servico.TotalPrice.Equal(decimal.NewFromFloat(230)) || servico.Description == nil || *servico.Description != "SERVICO INDUSTRIALIZACAO" {
		t.Fatalf("linha de serviço não veio do pedido: %+v", servico)
	}
	if servico.SalesOrderItemCode == nil || *servico.SalesOrderItemCode != 71 {
		t.Fatalf("a linha de serviço tem de apontar a linha 71 do pedido: %+v", servico.SalesOrderItemCode)
	}
	if servico.ItemCode != nil {
		t.Fatal("serviço não leva item_code: a autorização baixaria estoque do serviço")
	}
	for _, it := range notas.itens[1:] {
		if it.SalesOrderItemCode != nil {
			t.Fatalf("material do cliente não atende o pedido: %+v", it)
		}
	}
	if notas.saida.SalesOrderCode == nil || *notas.saida.SalesOrderCode != 5001 {
		t.Fatal("a nota leva o pedido")
	}
}

func TestFaturarNaoPassaDoSaldoDaLinha(t *testing.T) {
	uc, repo, notas := montarFaturamento()
	pedido := int64(5001)
	repo.remessa.SalesOrderCode = &pedido
	uc.Pedidos = pedidoDoCliente100()

	req := pedidoPadrao()
	req.Servico.Quantidade = "2" // o pedido tem 1
	_, err := uc.Faturar(context.Background(), req, autorValido)
	esperaValidacao(t, err, "passa do saldo")
	if notas.saida != nil {
		t.Fatal("nenhuma nota pode ser criada")
	}
}

func TestFaturarPedidoComVariasLinhasPedeAEscolha(t *testing.T) {
	uc, repo, _ := montarFaturamento()
	pedido := int64(5001)
	repo.remessa.SalesOrderCode = &pedido
	pedidos := pedidoDoCliente100()
	pedidos[5001].Linhas = append(pedidos[5001].Linhas, entity.LinhaDoPedido{
		Code: 72, Sequencia: 2, Descricao: "TRATAMENTO", Unidade: "PC", Quantidade: dec("1"), PrecoUnitario: dec("50"),
	})
	uc.Pedidos = pedidos

	req := pedidoPadrao()
	req.Servico = ServicoFaturado{}
	_, err := uc.Faturar(context.Background(), req, autorValido)
	esperaValidacao(t, err, "escolha")

	escolhida := int64(72)
	req.Servico.SalesOrderItemCode = &escolhida
	uc2, repo2, notas2 := montarFaturamento()
	repo2.remessa.SalesOrderCode = &pedido
	uc2.Pedidos = pedidos
	if _, err := uc2.Faturar(context.Background(), req, autorValido); err != nil {
		t.Fatal(err)
	}
	if *notas2.itens[0].SalesOrderItemCode != 72 || !notas2.itens[0].TotalPrice.Equal(decimal.NewFromFloat(50)) {
		t.Fatalf("faturou a linha errada: %+v", notas2.itens[0])
	}
}

func TestPropostaDeFaturamentoSugereServicoEDevolucoes(t *testing.T) {
	uc, repo, _ := montarFaturamento()
	pedido := int64(5001)
	repo.remessa.SalesOrderCode = &pedido
	uc.Pedidos = pedidoDoCliente100()

	p, err := uc.PropostaDeFaturamento(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if p.Aviso != "" || p.Servico.ValorUnit != "230" || p.Servico.SalesOrderItemCode == nil {
		t.Fatalf("proposta sem o serviço do pedido: %+v", p)
	}
	if len(p.Devolucoes) != 2 || p.Devolucoes[0].Quantidade != "10" {
		t.Fatalf("a proposta devolve o saldo inteiro de cada linha: %+v", p.Devolucoes)
	}
}

func TestPropostaSemPedidoAvisa(t *testing.T) {
	uc, _, _ := montarFaturamento()
	uc.Pedidos = pedidoDoCliente100()
	p, err := uc.PropostaDeFaturamento(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if p.Aviso == "" || p.Pedido != nil {
		t.Fatalf("sem pedido a proposta avisa: %+v", p)
	}
}

// A nota pendente aponta para a linha do pedido atual: trocar o pedido antes de
// descartá-la faria a autorização atender um pedido que a remessa não tem mais.
func TestTrocarPedidoComNotaPendenteERecusado(t *testing.T) {
	atual := int64(5002)
	repo := &repoFalso{remessa: &entity.Remessa{ID: 1, CustomerCode: 100, Status: entity.StatusParcial, SalesOrderCode: &atual,
		NotaPendente: &entity.NotaPendente{ID: 7, Numero: 763, Serie: "1", Status: "DRAFT"}}}
	uc := &UseCase{Repo: repo, Pedidos: pedidoDoCliente100()}
	_, err := uc.VincularPedido(context.Background(), 1, 5001, autorValido)
	if _, conflito := errorsuc.AsConflict(err); !conflito || !strings.Contains(err.Error(), "763/1") {
		t.Fatalf("erro = %v", err)
	}
	if repo.pedidoVinculado != 0 {
		t.Fatal("não deveria ter trocado o pedido")
	}
}

func TestOFNaoLigaAPedidoCancelado(t *testing.T) {
	pedidos := pedidoDoCliente100()
	pedidos[5001].Status = entity.PedidoCancelado
	uc := &UseCase{Repo: &repoFalso{}, Pedidos: pedidos}
	codigo := int64(5001)
	err := uc.VincularPedidoDaOF(context.Background(), 1, &codigo)
	if err == nil || !strings.Contains(err.Error(), "cancelado") {
		t.Fatalf("erro = %v", err)
	}
	// Faturado continua valendo: a hora pode ser apontada depois da nota.
	pedidos[5001].Status = entity.PedidoFaturado
	if err := uc.VincularPedidoDaOF(context.Background(), 1, &codigo); err != nil {
		t.Fatal(err)
	}
}

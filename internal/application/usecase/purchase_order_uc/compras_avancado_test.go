package purchase_order_uc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	procuremententity "github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

// ─── dublês ─────────────────────────────────────────────────────────────────

type authLivre struct {
	ports.AuthService
	uid uuid.UUID
}

func (authLivre) CanListPurchaseOrders(context.Context) bool    { return true }
func (authLivre) CanGetPurchaseOrder(context.Context) bool      { return true }
func (authLivre) CanUpdatePurchaseOrder(context.Context) bool   { return true }
func (authLivre) CanCreatePurchaseOrder(context.Context) bool   { return true }
func (a authLivre) UserID(context.Context) (uuid.UUID, error)   { return a.uid, nil }
func (authLivre) EnterpriseID(context.Context) (int64, error)   { return 1, nil }
func (authLivre) EnterpriseCode(context.Context) (int64, error) { return 1, nil }

type repoMemoria struct {
	porepo.PurchaseOrderRepository
	pedidos map[int64]*entity.PurchaseOrder
	itens   map[int64][]*entity.PurchaseOrderItem
}

func (r *repoMemoria) GetByCode(_ context.Context, c int64) (*entity.PurchaseOrder, error) {
	p, ok := r.pedidos[c]
	if !ok {
		return nil, errors.New("não encontrado")
	}
	cp := *p
	return &cp, nil
}
func (r *repoMemoria) ListItems(_ context.Context, c int64) ([]*entity.PurchaseOrderItem, error) {
	return r.itens[c], nil
}
func (r *repoMemoria) Update(_ context.Context, o *entity.PurchaseOrder) (*entity.PurchaseOrder, error) {
	cp := *o
	r.pedidos[o.Code] = &cp
	return &cp, nil
}
func (r *repoMemoria) ListByStatus(_ context.Context, s entity.PurchaseOrderStatus) ([]*entity.PurchaseOrder, error) {
	var out []*entity.PurchaseOrder
	for _, p := range r.pedidos {
		if p.Status == s {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

type politicaFixa struct{ teto float64 }

func (p politicaFixa) EvaluatePurchaseApproval(_ context.Context, _ int64, _ *int64, v float64) (*procuremententity.ApprovalDecision, error) {
	return &procuremententity.ApprovalDecision{AutoApprove: v <= p.teto, Ceiling: p.teto}, nil
}

type consultasFalsas struct {
	ConsultasCompras
	linhas []LinhaEmAberto
	envios []Envio
	dados  *DadosDocumento
}

func (c *consultasFalsas) LinhasEmAberto(context.Context, FiltroAcompanhamento) ([]LinhaEmAberto, error) {
	return c.linhas, nil
}
func (c *consultasFalsas) RegistrarEnvio(_ context.Context, e Envio) (*Envio, error) {
	e.ID = int64(len(c.envios) + 1)
	c.envios = append(c.envios, e)
	return &e, nil
}
func (c *consultasFalsas) DadosDocumento(context.Context, int64) (*DadosDocumento, error) {
	return c.dados, nil
}

type emailFalso struct {
	enviado *ports.EmailMessage
	falha   error
}

func (e *emailFalso) Send(_ context.Context, m ports.EmailMessage) error {
	e.enviado = &m
	return e.falha
}

type pdfFalso struct{}

func (pdfFalso) PedidoCompra(d DocumentoPedido) ([]byte, error) {
	return []byte("%PDF-falso " + d.Situacao), nil
}

type condicoesFalsas map[int64]*customerentity.PaymentCondition

func (c condicoesFalsas) GetPaymentConditionByCode(_ context.Context, code int64) (*customerentity.PaymentCondition, error) {
	if p, ok := c[code]; ok {
		return p, nil
	}
	return nil, errors.New("não encontrada")
}

func data(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

// ─── testes ─────────────────────────────────────────────────────────────────

func TestValidarEmails(t *testing.T) {
	got, err := validarEmails([]string{"Joana@Forn.com.br; compras@forn.com.br", "joana@forn.com.br", " "})
	if err != nil || len(got) != 2 || got[0] != "joana@forn.com.br" {
		t.Fatalf("got %v err %v", got, err)
	}
	for _, ruim := range [][]string{{}, {"sem-arroba"}, {"a@semponto"}, {"Nome <a@b.com>"}} {
		if _, err := validarEmails(ruim); err == nil {
			t.Errorf("%v deveria ser recusado", ruim)
		}
	}
	muitos := make([]string, 11)
	for i := range muitos {
		muitos[i] = strings.Repeat("a", i+1) + "@x.com"
	}
	if _, err := validarEmails(muitos); err == nil {
		t.Error("mais de 10 destinatários deveria ser recusado")
	}
}

func TestAcompanhamentoFiltraEOrdena(t *testing.T) {
	hoje := data("2026-10-10")
	d := func(s string) *time.Time { v := data(s); return &v }
	c := &consultasFalsas{linhas: []LinhaEmAberto{
		{LineCode: 1, DeliveryDate: d("2026-10-05")},                                // 5 dias de atraso
		{LineCode: 2, DeliveryDate: d("2026-10-01"), PromisedDate: d("2026-10-15")}, // prometido: em dia
		{LineCode: 3, DeliveryDate: d("2026-10-12")},                                // próximos 7
		{LineCode: 4}, // sem data
		{LineCode: 5, PromisedDate: d("2026-10-08")}, // 2 dias de atraso
	}}
	uc := &AcompanhamentoUseCase{Consultas: c, Auth: authLivre{}}
	casos := map[string][]int64{
		AcompanhamentoAtrasadas:   {1, 5},
		AcompanhamentoProximos7:   {3, 2},
		AcompanhamentoSemPromessa: {1, 3, 4},
		"":                        {1, 5, 3, 2, 4},
	}
	for sit, quer := range casos {
		got, err := uc.Linhas(context.Background(), sit, nil, hoje)
		if err != nil {
			t.Fatal(err)
		}
		var codes []int64
		for _, l := range got {
			codes = append(codes, l.LineCode)
		}
		if len(codes) != len(quer) {
			t.Fatalf("%q: %v, quer %v", sit, codes, quer)
		}
		for i := range quer {
			if codes[i] != quer[i] {
				t.Fatalf("%q: %v, quer %v", sit, codes, quer)
			}
		}
	}
	if _, err := uc.Linhas(context.Background(), "QUALQUER", nil, hoje); err == nil {
		t.Error("situação inválida deveria ser recusada")
	}
}

func pedidoAprovado(code int64, linhas ...*entity.PurchaseOrderItem) *repoMemoria {
	wh := int64(3)
	for _, l := range linhas {
		l.WarehouseID = &wh
		l.IsActive = true
		l.Status = entity.PurchaseOrderItemStatusOPEN
	}
	sup := int64(77)
	return &repoMemoria{
		pedidos: map[int64]*entity.PurchaseOrder{code: {Code: code, OrderNumber: 42, Status: entity.PurchaseOrderStatusDRAFT, IsActive: true,
			SupplierCode: &sup, EmissionDate: data("2026-10-01"), CurrencyCode: "BRL", FreightType: "CIF"}},
		itens: map[int64][]*entity.PurchaseOrderItem{code: linhas},
	}
}

func TestGeracaoPassaPelaAlcada(t *testing.T) {
	ctx := context.Background()
	repo := pedidoAprovado(10, &entity.PurchaseOrderItem{Code: 1, Sequence: 1, ItemCode: 5, RequestedQty: 10, UnitPrice: 50})
	g := &Geracao{Repo: repo, Aprovacao: &ApprovePurchaseOrderUseCase{Repo: repo, Auth: authLivre{}, Policy: politicaFixa{teto: 100}}}
	po, aviso, err := g.Aprovar(ctx, repo.pedidos[10])
	if err != nil {
		t.Fatal(err)
	}
	if po.Status != entity.PurchaseOrderStatusREQUESTED || po.AlcadaStatus != "B" || !strings.Contains(aviso, "alçada") {
		t.Fatalf("pedido de 500 com alçada de 100: %s/%s aviso=%q", po.Status, po.AlcadaStatus, aviso)
	}

	repo2 := pedidoAprovado(11, &entity.PurchaseOrderItem{Code: 2, Sequence: 1, ItemCode: 5, RequestedQty: 1, UnitPrice: 50})
	g2 := &Geracao{Repo: repo2, Aprovacao: &ApprovePurchaseOrderUseCase{Repo: repo2, Auth: authLivre{}, Policy: politicaFixa{teto: 100}}}
	if po, aviso, err := g2.Aprovar(ctx, repo2.pedidos[11]); err != nil || po.Status != entity.PurchaseOrderStatusAPPROVED || aviso != "" {
		t.Fatalf("pedido de 50: %v %q %v", po.Status, aviso, err)
	}

	// Linha sem almoxarifado: o pedido fica em rascunho e o motivo volta como aviso.
	repo3 := pedidoAprovado(12, &entity.PurchaseOrderItem{Code: 3, Sequence: 1, ItemCode: 5, RequestedQty: 1, UnitPrice: 1})
	repo3.itens[12][0].WarehouseID = nil
	g3 := &Geracao{Repo: repo3, Aprovacao: &ApprovePurchaseOrderUseCase{Repo: repo3, Auth: authLivre{}}}
	if po, aviso, err := g3.Aprovar(ctx, repo3.pedidos[12]); err != nil || po.Status != entity.PurchaseOrderStatusDRAFT || !strings.Contains(aviso, "almoxarifado") {
		t.Fatalf("sem almoxarifado: %v %q %v", po.Status, aviso, err)
	}

	// Sem aprovação configurada, a capa nasce rascunho e o aviso orienta.
	capa := &entity.PurchaseOrder{Code: 13, Status: entity.PurchaseOrderStatusAPPROVED, AlcadaStatus: "A"}
	var nada *Geracao
	nada.NovaCapa(capa)
	if capa.Status != entity.PurchaseOrderStatusDRAFT || capa.AlcadaStatus != "N" {
		t.Fatalf("capa gerada: %s/%s", capa.Status, capa.AlcadaStatus)
	}
	if _, aviso, _ := nada.Aprovar(ctx, capa); !strings.Contains(aviso, "rascunho") {
		t.Fatalf("aviso sem aprovação: %q", aviso)
	}
}

func TestEnviarPedido(t *testing.T) {
	ctx := context.Background()
	repo := pedidoAprovado(20, &entity.PurchaseOrderItem{Code: 1, Sequence: 1, ItemCode: 5, RequestedQty: 2, UnitPrice: 10})
	c := &consultasFalsas{dados: &DadosDocumento{Empresa: Parte{Nome: "TECNOFER"}}}
	mail := &emailFalso{}
	uc := &DocumentoPedidoUseCase{Repo: repo, Consultas: c, Gerador: pdfFalso{}, Email: mail, Auth: authLivre{uid: uuid.New()},
		Agora: func() time.Time { return data("2026-10-07") }}

	if _, err := uc.Enviar(ctx, 20, EnviarDTO{Para: []string{"a@b.com"}}); err == nil || !strings.Contains(err.Error(), "aprovado") {
		t.Fatalf("rascunho não vai ao fornecedor: %v", err)
	}
	repo.pedidos[20].Status = entity.PurchaseOrderStatusAPPROVED
	env, err := uc.Enviar(ctx, 20, EnviarDTO{Para: []string{"a@b.com"}})
	if err != nil || env.Situacao != "ENVIADO" {
		t.Fatalf("envio: %+v %v", env, err)
	}
	m := mail.enviado
	if m == nil || len(m.Attachments) != 1 || m.Attachments[0].FileName != "pedido-compra-42.pdf" || !strings.HasPrefix(string(m.Attachments[0].Content), "%PDF") {
		t.Fatalf("mensagem: %+v", m)
	}
	if !strings.Contains(m.Subject, "nº 42") || !strings.Contains(m.Subject, "TECNOFER") || !strings.Contains(m.Text, "pedido de compra nº 42") {
		t.Fatalf("assunto/texto: %q / %q", m.Subject, m.Text)
	}

	mail.falha = errors.New("SMTP fora do ar")
	env, err = uc.Enviar(ctx, 20, EnviarDTO{Para: []string{"a@b.com"}, Mensagem: "<b>oi</b>"})
	if err == nil || env == nil || env.Situacao != "FALHOU" || env.Erro != "SMTP fora do ar" {
		t.Fatalf("falha registrada: %+v %v", env, err)
	}
	if !strings.Contains(mail.enviado.HTML, "&lt;b&gt;") {
		t.Fatalf("mensagem do usuário escapada no HTML: %q", mail.enviado.HTML)
	}
	if len(c.envios) != 2 {
		t.Fatalf("envios registrados: %d", len(c.envios))
	}
}

func TestPrevisaoPelaCondicao(t *testing.T) {
	ctx := context.Background()
	entrega := data("2026-10-20")
	repo := pedidoAprovado(30, &entity.PurchaseOrderItem{Code: 1, Sequence: 1, ItemCode: 5, RequestedQty: 10, UnitPrice: 30, DeliveryDate: &entrega})
	repo.pedidos[30].Status = entity.PurchaseOrderStatusAPPROVED
	cond := int64(3060)
	repo.pedidos[30].PaymentTermCode = &cond
	condicoes := condicoesFalsas{3060: {Description: "30/60", Installments: []*customerentity.PaymentInstallment{
		{InstallmentNumber: 1, DueDays: 30, IsActive: true, BaseEvent: string(customerentity.BaseFaturamento)},
		{InstallmentNumber: 2, DueDays: 60, IsActive: true, BaseEvent: string(customerentity.BaseFaturamento)},
	}}}
	uc := &PrevisaoPagamentosUseCase{Repo: repo, Condicoes: condicoes, Auth: authLivre{}}
	p, err := uc.DoPedido(ctx, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Parcelas) != 2 || p.Total != 300 || !p.Parcelas[0].Vencimento.Equal(data("2026-11-19")) || !p.Parcelas[1].Vencimento.Equal(data("2026-12-19")) {
		t.Fatalf("previsão: %+v", p)
	}
	geral, err := uc.Geral(ctx, data("2026-11-01"), data("2026-11-30"))
	if err != nil || len(geral.Parcelas) != 1 || geral.Total != 150 {
		t.Fatalf("geral de novembro: %+v %v", geral, err)
	}
	// Rascunho não entra na previsão geral (ainda não é compromisso).
	repo.pedidos[30].Status = entity.PurchaseOrderStatusDRAFT
	if g, _ := uc.Geral(ctx, data("2026-01-01"), data("2027-12-31")); len(g.Parcelas) != 0 {
		t.Fatalf("rascunho na previsão: %+v", g)
	}
	// Condição inexistente: à vista na entrega, com aviso.
	outra := int64(9)
	repo.pedidos[30].PaymentTermCode = &outra
	p, _ = uc.DoPedido(ctx, 30)
	if len(p.Parcelas) != 1 || !p.Parcelas[0].Vencimento.Equal(entrega) || len(p.Avisos) != 1 {
		t.Fatalf("sem condição: %+v", p)
	}
}

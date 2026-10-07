package purchase_quotation_uc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	procuremententity "github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	qentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
	qrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository"
	reqentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	reqrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
)

type authGerar struct{ ports.AuthService }

func (authGerar) CanCreatePurchaseOrder(context.Context) bool   { return true }
func (authGerar) CanUpdatePurchaseOrder(context.Context) bool   { return true }
func (authGerar) EnterpriseCode(context.Context) (int64, error) { return 1, nil }
func (authGerar) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-0000000e2e01"), nil
}

type cotacaoFalsa struct {
	qrepo.PurchaseQuotationRepository
	itens   []*qentity.PurchaseQuotationItem
	precos  []*qentity.PurchaseQuotationPrice
	fechada bool
}

func (c *cotacaoFalsa) GetByCode(context.Context, int64) (*qentity.PurchaseQuotation, error) {
	return &qentity.PurchaseQuotation{Code: 5, EnterpriseCode: 1}, nil
}
func (c *cotacaoFalsa) ListItems(context.Context, int64) ([]*qentity.PurchaseQuotationItem, error) {
	return c.itens, nil
}
func (c *cotacaoFalsa) ListSelectedPrices(context.Context, int64) ([]*qentity.PurchaseQuotationPrice, error) {
	return c.precos, nil
}
func (c *cotacaoFalsa) UpdateStatus(context.Context, int64, string) error {
	c.fechada = true
	return nil
}

type reqFalsa struct {
	reqrepo.PurchaseRequisitionRepository
	atendido map[int64]float64
}

func (r *reqFalsa) RegisterAttendance(_ context.Context, id int64, q float64) (*reqentity.PurchaseRequisitionItem, error) {
	r.atendido[id] += q
	return &reqentity.PurchaseRequisitionItem{}, nil
}

type poFalso struct {
	porepo.PurchaseOrderRepository
	pedidos map[int64]*poentity.PurchaseOrder
	itens   map[int64][]*poentity.PurchaseOrderItem
}

func (p *poFalso) NextOrderNumber(context.Context, int64) (int64, error) {
	return int64(len(p.pedidos) + 1), nil
}
func (p *poFalso) CreateWithItems(_ context.Context, o *poentity.PurchaseOrder, it []*poentity.PurchaseOrderItem) (*poentity.PurchaseOrder, error) {
	o.Code = int64(100 + len(p.pedidos))
	o.IsActive = true
	o.AplicarTotais(poentity.CalcularTotais(o, it))
	cp := *o
	p.pedidos[o.Code] = &cp
	p.itens[o.Code] = it
	return o, nil
}
func (p *poFalso) GetByCode(_ context.Context, c int64) (*poentity.PurchaseOrder, error) {
	cp := *p.pedidos[c]
	return &cp, nil
}
func (p *poFalso) ListItems(_ context.Context, c int64) ([]*poentity.PurchaseOrderItem, error) {
	return p.itens[c], nil
}
func (p *poFalso) Update(_ context.Context, o *poentity.PurchaseOrder) (*poentity.PurchaseOrder, error) {
	cp := *o
	p.pedidos[o.Code] = &cp
	return &cp, nil
}

type almoxFixo struct{}

func (almoxFixo) AlmoxarifadoPadraoDoItem(context.Context, int64) (*int64, error) {
	v := int64(9)
	return &v, nil
}

type tetoDe100 struct{}

func (tetoDe100) EvaluatePurchaseApproval(_ context.Context, _ int64, _ *int64, v float64) (*procuremententity.ApprovalDecision, error) {
	return &procuremententity.ApprovalDecision{AutoApprove: v <= 100, Ceiling: 100}, nil
}

// O pedido da cotação passa pela alçada, leva a condição cotada, o vínculo com
// a cotação e com a requisição, e a entrega pelo prazo do fornecedor.
func TestGerarPedidoDaCotacaoPassaPelaAlcada(t *testing.T) {
	req := int64(77)
	termo := int64(3060)
	cot := &cotacaoFalsa{
		itens: []*qentity.PurchaseQuotationItem{
			{ID: 1, ItemCode: 10, Quantity: 2, SourceType: qentity.SourceRequisition, SourceItemID: &req},
			{ID: 2, ItemCode: 11, Quantity: 1},
		},
		precos: []*qentity.PurchaseQuotationPrice{
			{QuotationItemID: 1, SupplierCode: 500, UnitPrice: 30, LeadTimeDays: 10, PaymentTermCode: &termo}, // 60
			{QuotationItemID: 2, SupplierCode: 600, UnitPrice: 300},                                           // 300 > alçada
		},
	}
	pos := &poFalso{pedidos: map[int64]*poentity.PurchaseOrder{}, itens: map[int64][]*poentity.PurchaseOrderItem{}}
	reqs := &reqFalsa{atendido: map[int64]float64{}}
	aprov := &purchase_order_uc.ApprovePurchaseOrderUseCase{Repo: pos, Auth: authGerar{}, Policy: tetoDe100{}}
	uc := &GenerateOrdersFromQuotationUseCase{Quotations: cot, Reqs: reqs, POs: pos, Auth: authGerar{},
		Geracao: &purchase_order_uc.Geracao{Repo: pos, Almoxarifados: almoxFixo{}, Aprovacao: aprov}}
	res, err := uc.Execute(context.Background(), request.GenerateOrdersFromQuotationDTO{QuotationCode: 5, CreatedBy: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Orders) != 2 {
		t.Fatalf("pedidos: %+v", res)
	}
	a, b := res.Orders[0], res.Orders[1]
	if a.Status != poentity.PurchaseOrderStatusAPPROVED || a.AlcadaStatus != "A" || a.PaymentTermCode == nil || *a.PaymentTermCode != termo {
		t.Fatalf("pedido dentro da alçada: %+v", a)
	}
	if b.Status != poentity.PurchaseOrderStatusREQUESTED || b.AlcadaStatus != "B" || len(res.Skipped) != 1 {
		t.Fatalf("pedido acima da alçada: %s/%s avisos %v", b.Status, b.AlcadaStatus, res.Skipped)
	}
	la := pos.itens[a.Code][0]
	if la.WarehouseID == nil || *la.WarehouseID != 9 || la.QuotationCode == nil || *la.QuotationCode != 5 ||
		la.PurchaseRequisitionItemID == nil || *la.PurchaseRequisitionItemID != req || la.TotalPrice != 60 {
		t.Fatalf("linha da cotação: %+v", la)
	}
	if la.DeliveryDate == nil || la.DeliveryDate.Before(time.Now().AddDate(0, 0, 9)) {
		t.Fatalf("entrega pelo prazo de 10 dias: %v", la.DeliveryDate)
	}
	if a.CreatedBy.String() != "00000000-0000-0000-0000-0000000e2e01" {
		t.Fatalf("autor do pedido gerado = %s, quer o usuário da sessão", a.CreatedBy)
	}
	if reqs.atendido[req] != 2 || !cot.fechada {
		t.Fatalf("requisição atendida %v, cotação fechada %v", reqs.atendido, cot.fechada)
	}
}

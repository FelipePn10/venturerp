// Package contrato_venda_uc: contrato de venda (comercial, Fase 5.2).
package contrato_venda_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	dom "github.com/FelipePn10/panossoerp/internal/domain/contratovenda"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
)

type UseCase struct {
	Repo  dom.Repository
	Auth  ports.AuthService
	Items any
	Agora func() time.Time
}

func (uc *UseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

func (uc *UseCase) ator(ctx context.Context) (uuid.UUID, error) {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return uuid.Nil, errorsuc.ErrUnauthorized
	}
	return a, nil
}

func dia(s, campo string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), time.Local)
	if err != nil {
		return time.Time{}, errorsuc.NewValidationError(fmt.Sprintf("%s inválida (use AAAA-MM-DD)", campo))
	}
	return t, nil
}

type ItemDTO struct {
	ItemCode      request.TextCode `json:"item_code"`
	Quantidade    decimal.Decimal  `json:"quantidade"`
	PrecoUnitario decimal.Decimal  `json:"preco_unitario"`
}

type CriarDTO struct {
	CustomerCode    int64            `json:"customer_code"`
	Descricao       string           `json:"descricao"`
	VigenciaInicio  string           `json:"vigencia_inicio"`
	VigenciaFim     string           `json:"vigencia_fim"`
	ValorContratado *decimal.Decimal `json:"valor_contratado,omitempty"`
	AlertaSaldoPct  *int             `json:"alerta_saldo_pct,omitempty"`
	AlertaDias      *int             `json:"alerta_dias,omitempty"`
	PaymentTermCode *int64           `json:"payment_term_code,omitempty"`
	Observacao      *string          `json:"observacao,omitempty"`
	Itens           []ItemDTO        `json:"itens"`
}

func (uc *UseCase) Criar(ctx context.Context, dto CriarDTO) (*dom.Contrato, error) {
	por, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	ini, err := dia(dto.VigenciaInicio, "data de início")
	if err != nil {
		return nil, err
	}
	fim, err := dia(dto.VigenciaFim, "data de fim")
	if err != nil {
		return nil, err
	}
	c := &dom.Contrato{CustomerCode: dto.CustomerCode, Descricao: strings.TrimSpace(dto.Descricao), VigenciaInicio: ini, VigenciaFim: fim,
		ValorContratado: dto.ValorContratado, AlertaSaldoPct: 20, AlertaDias: 30, PaymentTermCode: dto.PaymentTermCode, Observacao: dto.Observacao}
	if dto.AlertaSaldoPct != nil {
		c.AlertaSaldoPct = *dto.AlertaSaldoPct
	}
	if dto.AlertaDias != nil {
		c.AlertaDias = *dto.AlertaDias
	}
	for _, it := range dto.Itens {
		item, err := itemresolution.Resolve(ctx, uc.Items, it.ItemCode)
		if err != nil {
			return nil, err
		}
		c.Itens = append(c.Itens, dom.Item{ItemCode: int64(item.Code), Quantidade: it.Quantidade, PrecoUnitario: it.PrecoUnitario})
	}
	if err := c.ConferirCadastro(); err != nil {
		return nil, err
	}
	if err := uc.Repo.Criar(ctx, c, &por); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, c.ID)
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*dom.Contrato, error) {
	return uc.Repo.Obter(ctx, id)
}

func (uc *UseCase) Listar(ctx context.Context, f dom.Filtro) ([]dom.Contrato, error) {
	f.Situacao = strings.ToUpper(strings.TrimSpace(f.Situacao))
	return uc.Repo.Listar(ctx, f)
}

// Mudar a situação: ATIVO, ENCERRADO ou CANCELADO.
func (uc *UseCase) Mudar(ctx context.Context, id int64, para, motivo string) (*dom.Contrato, error) {
	por, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	para = strings.ToUpper(strings.TrimSpace(para))
	c, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	tem, err := uc.Repo.TemChamada(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := dom.Transicao(c, para, tem); err != nil {
		return nil, err
	}
	var m *string
	if para == dom.Encerrado || para == dom.Cancelado {
		motivo = strings.TrimSpace(motivo)
		if motivo == "" {
			return nil, errorsuc.NewValidationError("informe o motivo")
		}
		m = &motivo
	}
	if err := uc.Repo.Mudar(ctx, id, c.Situacao, para, m, &por); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, id)
}

type ParcelaDTO struct {
	ContratoItemID int64           `json:"contrato_item_id"`
	Data           string          `json:"data"`
	Quantidade     decimal.Decimal `json:"quantidade"`
}

func (uc *UseCase) Programar(ctx context.Context, id int64, dto ParcelaDTO) (*dom.Contrato, error) {
	d, err := dia(dto.Data, "data prevista")
	if err != nil {
		return nil, err
	}
	if !dto.Quantidade.IsPositive() {
		return nil, errorsuc.NewValidationError("a quantidade da parcela é positiva")
	}
	c, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	pertence := false
	for _, it := range c.Itens {
		if it.ID == dto.ContratoItemID {
			pertence = true
		}
	}
	if !pertence {
		return nil, errorsuc.NewValidationError("o item não é deste contrato")
	}
	if d.Before(c.VigenciaInicio.AddDate(0, 0, -1)) || d.After(c.VigenciaFim.AddDate(0, 0, 1)) {
		return nil, errorsuc.NewValidationError("a parcela está fora da vigência do contrato")
	}
	if err := uc.Repo.Programar(ctx, dto.ContratoItemID, d, dto.Quantidade); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, id)
}

func (uc *UseCase) RemoverParcela(ctx context.Context, id, parcela int64) (*dom.Contrato, error) {
	if err := uc.Repo.RemoverParcela(ctx, parcela); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, id)
}

type ChamadaDTO struct {
	Linhas            []dom.LinhaChamada `json:"linhas"`
	DataEntrega       string             `json:"data_entrega"`
	SalesDivisionCode *int64             `json:"sales_division_code,omitempty"`
}

type ChamadaResponse struct {
	SalesOrderCode int64           `json:"sales_order_code"`
	OrderNumber    int64           `json:"order_number"`
	Total          decimal.Decimal `json:"total"`
	Contrato       *dom.Contrato   `json:"contrato"`
}

// Chamar gera o pedido de venda de chamada (rascunho) pelo preço do contrato e
// consome o saldo. A data de entrega precisa estar na vigência.
func (uc *UseCase) Chamar(ctx context.Context, id int64, dto ChamadaDTO) (*ChamadaResponse, error) {
	por, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	entrega, err := dia(dto.DataEntrega, "data de entrega")
	if err != nil {
		return nil, err
	}
	c, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	total, err := dom.ConferirChamada(c, dto.Linhas, entrega)
	if err != nil {
		return nil, err
	}
	empresa, err := uc.Auth.EnterpriseCode(ctx)
	if err != nil {
		return nil, err
	}
	porItem := map[int64]dom.Item{}
	for _, it := range c.Itens {
		porItem[it.ID] = it
	}
	agora := uc.agora()
	nota := fmt.Sprintf("Pedido de chamada do contrato de venda %d — %s", c.Numero, c.Descricao)
	cliente := c.CustomerCode
	pedido := &orderentity.SalesOrder{EnterpriseCode: empresa, Status: orderentity.SalesOrderStatusDraft, Origin: orderentity.SalesOrderOriginNormal,
		EmissionDate: agora, DigitDate: agora, CustomerCode: &cliente, SalesDivisionCode: dto.SalesDivisionCode, PaymentTermCode: c.PaymentTermCode,
		CurrencyCode: "BRL", Notes: &nota, CreatedBy: por}
	itens := make([]*orderentity.SalesOrderItem, 0, len(dto.Linhas))
	for i, l := range dto.Linhas {
		it := porItem[l.ContratoItemID]
		obs := fmt.Sprintf("Contrato %d", c.Numero)
		e := entrega
		itens = append(itens, &orderentity.SalesOrderItem{Sequence: i + 1, ItemCode: it.ItemCode, DigitDate: agora, RequestedQty: l.Quantidade,
			UnitPrice: it.PrecoUnitario, DeliveryDate: &e, Status: orderentity.SalesOrderItemStatusOpen, Notes: &obs})
	}
	criado, err := uc.Repo.Chamar(ctx, dom.Chamada{ContratoID: id, Linhas: dto.Linhas, Data: entrega, Pedido: pedido, Itens: itens, Por: por})
	if err != nil {
		return nil, err
	}
	atual, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ChamadaResponse{SalesOrderCode: criado.Code, OrderNumber: criado.OrderNumber, Total: total, Contrato: atual}, nil
}

// Package ordem_servico_uc: ordem de serviço da assistência técnica
// (pós-venda, Fase 3.3).
package ordem_servico_uc

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
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/ordem"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type UseCase struct {
	Repo        dom.Repository
	Auth        ports.AuthService
	SalesOrders orderrepo.SalesOrderRepository
	// Items traduz o código de negócio do item da peça.
	Items any
	// SLA: abrir a OS é a primeira resposta do chamado.
	SLA interface {
		Responder(ctx context.Context, callCode int64) error
	}
	Agora func() time.Time
}

func (uc *UseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

func (uc *UseCase) ator(ctx context.Context) *uuid.UUID {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return nil
	}
	return &a
}

func limpo(s *string) *string {
	if s == nil {
		return nil
	}
	if v := strings.TrimSpace(*s); v != "" {
		return &v
	}
	return nil
}

// ─── Parâmetros ──────────────────────────────────────────────────────────────

func (uc *UseCase) Parametros(ctx context.Context) (*dom.Parametros, error) {
	return uc.Repo.Parametros(ctx)
}

// ParametrosDTO: o item de serviço vem pelo código de negócio.
type ParametrosDTO struct {
	CustoHora   decimal.Decimal  `json:"custo_hora"`
	PrecoHora   decimal.Decimal  `json:"preco_hora"`
	CustoKM     decimal.Decimal  `json:"custo_km"`
	PrecoKM     decimal.Decimal  `json:"preco_km"`
	WarehouseID *int64           `json:"warehouse_id,omitempty"`
	ItemServico request.TextCode `json:"item_servico"`
}

func (uc *UseCase) SalvarParametros(ctx context.Context, dto ParametrosDTO) (*dom.Parametros, error) {
	p := dom.Parametros{CustoHora: dto.CustoHora, PrecoHora: dto.PrecoHora, CustoKM: dto.CustoKM, PrecoKM: dto.PrecoKM, WarehouseID: dto.WarehouseID}
	if strings.TrimSpace(string(dto.ItemServico)) != "" {
		item, err := itemresolution.Resolve(ctx, uc.Items, dto.ItemServico)
		if err != nil {
			return nil, err
		}
		c := int64(item.Code)
		p.ItemServicoCode = &c
	}
	for _, v := range []decimal.Decimal{p.CustoHora, p.PrecoHora, p.CustoKM, p.PrecoKM} {
		if v.IsNegative() {
			return nil, errorsuc.NewValidationError("custo e preço não podem ser negativos")
		}
	}
	if err := uc.Repo.SalvarParametros(ctx, &p, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Repo.Parametros(ctx)
}

// ─── OS ──────────────────────────────────────────────────────────────────────

type AberturaDTO struct {
	CallCode     int64   `json:"call_code"`
	CallItemCode *int64  `json:"call_item_code,omitempty"`
	Local        string  `json:"local"`
	TecnicoCode  *int64  `json:"tecnico_code,omitempty"`
	AgendadaPara *string `json:"agendada_para,omitempty"`
	Observacoes  *string `json:"observacoes,omitempty"`
}

func quando(s *string) (*time.Time, error) {
	v := limpo(s)
	if v == nil {
		return nil, nil
	}
	for _, f := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(f, *v, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, errorsuc.NewValidationError("data do agendamento inválida (use AAAA-MM-DD HH:MM)")
}

func (uc *UseCase) Abrir(ctx context.Context, dto AberturaDTO) (*OSResponse, error) {
	if dto.CallCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o chamado")
	}
	local := strings.ToUpper(strings.TrimSpace(dto.Local))
	if local != dom.LocalCliente && local != dom.LocalFabrica {
		return nil, errorsuc.NewValidationError("o atendimento é no cliente ou na fábrica")
	}
	ch, err := uc.Repo.DoChamado(ctx, dto.CallCode, dto.CallItemCode)
	if err != nil {
		return nil, err
	}
	agenda, err := quando(dto.AgendadaPara)
	if err != nil {
		return nil, err
	}
	situacao := dom.Aberta
	if agenda != nil {
		if err := dom.Agenda(*agenda, uc.agora()); err != nil {
			return nil, err
		}
		situacao = dom.Agendada
	}
	o := &dom.OS{CallCode: ch.CallCode, CallItemCode: ch.CallItemCode, SerieID: ch.SerieID, ItemCode: ch.ItemCode, CustomerCode: ch.CustomerCode,
		Local: local, Situacao: situacao, TecnicoCode: dto.TecnicoCode, AgendadaPara: agenda, Cobertura: ch.Cobertura, Observacoes: limpo(dto.Observacoes)}
	if err := uc.Repo.Abrir(ctx, o, uc.ator(ctx)); err != nil {
		return nil, err
	}
	if uc.SLA != nil {
		if err := uc.SLA.Responder(ctx, o.CallCode); err != nil {
			return nil, err
		}
	}
	return uc.Obter(ctx, o.ID)
}

// OSResponse: a OS com os lançamentos, o histórico e os totais.
type OSResponse struct {
	*dom.OS
	Horas    []dom.Hora    `json:"horas"`
	Pecas    []dom.Peca    `json:"pecas"`
	Despesas []dom.Despesa `json:"despesas"`
	Eventos  []dom.Evento  `json:"eventos"`
	Totais   dom.Totais    `json:"totais"`
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*OSResponse, error) {
	o, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	h, p, d, e, err := uc.Repo.Lancamentos(ctx, id)
	if err != nil {
		return nil, err
	}
	return &OSResponse{OS: o, Horas: h, Pecas: p, Despesas: d, Eventos: e, Totais: totais(h, p, d)}, nil
}

func totais(h []dom.Hora, p []dom.Peca, d []dom.Despesa) dom.Totais {
	lh := make([]dom.Linha, 0, len(h))
	var trab, desl decimal.Decimal
	for _, x := range h {
		lh = append(lh, dom.Linha{Custo: x.Horas.Mul(x.CustoHora), Preco: x.Horas.Mul(x.PrecoHora), Cobrado: x.Cobrado})
		if x.Tipo == dom.HoraDeslocamento {
			desl = desl.Add(x.Horas)
		} else {
			trab = trab.Add(x.Horas)
		}
	}
	lp := make([]dom.Linha, 0, len(p))
	for _, x := range p {
		lp = append(lp, dom.Linha{Custo: x.Quantidade.Mul(x.CustoUnitario), Preco: x.Quantidade.Mul(x.PrecoUnitario), Cobrado: x.Cobrado})
	}
	ld := make([]dom.Linha, 0, len(d))
	for _, x := range d {
		ld = append(ld, dom.Linha{Custo: x.Quantidade.Mul(x.Custo), Preco: x.Quantidade.Mul(x.Preco), Cobrado: x.Cobrado})
	}
	t := dom.Somar(lh, lp, ld)
	t.HorasTrabalho, t.HorasViagem = trab, desl
	for _, f := range []*decimal.Decimal{&t.CustoHoras, &t.CustoPecas, &t.CustoDespesas, &t.CustoTotal, &t.ACobrar, &t.CustoGarantia} {
		*f = f.Round(2)
	}
	return t
}

func (uc *UseCase) Listar(ctx context.Context, f dom.Filtro) ([]dom.OS, error) {
	f.Situacao = strings.ToUpper(strings.TrimSpace(f.Situacao))
	return uc.Repo.Listar(ctx, f)
}

type AgendaDTO struct {
	TecnicoCode *int64 `json:"tecnico_code,omitempty"`
	Quando      string `json:"quando"`
}

func (uc *UseCase) Agendar(ctx context.Context, id int64, dto AgendaDTO) (*OSResponse, error) {
	q, err := quando(&dto.Quando)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, errorsuc.NewValidationError("informe a data e a hora do agendamento")
	}
	if err := dom.Agenda(*q, uc.agora()); err != nil {
		return nil, err
	}
	if err := uc.Repo.Mudar(ctx, id, dom.Mudanca{Para: dom.Agendada, TecnicoCode: dto.TecnicoCode, AgendadaPara: q, Por: uc.ator(ctx)}); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

func verdade(b bool) *bool { return &b }

func (uc *UseCase) Iniciar(ctx context.Context, id int64, tecnico *int64) (*OSResponse, error) {
	o, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if tecnico == nil && o.TecnicoCode == nil {
		return nil, errorsuc.NewValidationError("informe o técnico que vai executar a OS")
	}
	m := dom.Mudanca{Para: dom.EmExecucao, TecnicoCode: tecnico, Por: uc.ator(ctx)}
	if o.Local == dom.LocalFabrica {
		m.SerieEmAssistencia = verdade(true)
	}
	if err := uc.Repo.Mudar(ctx, id, m); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

type ConclusaoDTO struct {
	DefeitoConstatado string `json:"defeito_constatado"`
	Solucao           string `json:"solucao"`
}

func (uc *UseCase) Concluir(ctx context.Context, id int64, dto ConclusaoDTO) (*OSResponse, error) {
	if err := dom.Conclusao(dto.DefeitoConstatado, dto.Solucao); err != nil {
		return nil, err
	}
	o, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	h, _, _, _, err := uc.Repo.Lancamentos(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(h) == 0 {
		return nil, errorsuc.NewValidationError("aponte as horas do atendimento antes de concluir a OS")
	}
	defeito, solucao := strings.TrimSpace(dto.DefeitoConstatado), strings.TrimSpace(dto.Solucao)
	m := dom.Mudanca{Para: dom.Concluida, DefeitoConstatado: &defeito, Solucao: &solucao, Por: uc.ator(ctx)}
	if o.Local == dom.LocalFabrica {
		m.SerieEmAssistencia = verdade(false)
	}
	if err := uc.Repo.Mudar(ctx, id, m); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

// Reabrir: OS concluída volta à execução (retrabalho antes de faturar).
func (uc *UseCase) Reabrir(ctx context.Context, id int64, motivo string) (*OSResponse, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe por que a OS volta à execução")
	}
	o, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	m := dom.Mudanca{Para: dom.EmExecucao, Detalhe: &motivo, Por: uc.ator(ctx)}
	if o.Local == dom.LocalFabrica {
		m.SerieEmAssistencia = verdade(true)
	}
	if err := uc.Repo.Mudar(ctx, id, m); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

func (uc *UseCase) Cancelar(ctx context.Context, id int64, motivo string) (*OSResponse, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do cancelamento")
	}
	o, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	_, p, _, _, err := uc.Repo.Lancamentos(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, x := range p {
		if x.StockMovementID != nil {
			return nil, errorsuc.NewValidationError("a OS tem peça que já saiu do estoque: remova a peça (ela volta ao estoque) antes de cancelar")
		}
	}
	m := dom.Mudanca{Para: dom.Cancelada, MotivoCancelamento: &motivo, Detalhe: &motivo, Por: uc.ator(ctx)}
	if o.Local == dom.LocalFabrica && o.Situacao == dom.EmExecucao {
		m.SerieEmAssistencia = verdade(false)
	}
	if err := uc.Repo.Mudar(ctx, id, m); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

// ─── Lançamentos ─────────────────────────────────────────────────────────────

type HoraDTO struct {
	TecnicoCode *int64           `json:"tecnico_code,omitempty"`
	Data        string           `json:"data"`
	Tipo        string           `json:"tipo"`
	Horas       decimal.Decimal  `json:"horas"`
	CustoHora   *decimal.Decimal `json:"custo_hora,omitempty"`
	PrecoHora   *decimal.Decimal `json:"preco_hora,omitempty"`
	Observacao  *string          `json:"observacao,omitempty"`
}

func (uc *UseCase) AddHora(ctx context.Context, id int64, dto HoraDTO) (*OSResponse, error) {
	o, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	data, err := time.Parse("2006-01-02", strings.TrimSpace(dto.Data))
	if err != nil {
		return nil, errorsuc.NewValidationError("data do apontamento inválida (use AAAA-MM-DD)")
	}
	tipo := strings.ToUpper(strings.TrimSpace(dto.Tipo))
	if tipo == "" {
		tipo = dom.HoraTrabalho
	}
	if tipo != dom.HoraTrabalho && tipo != dom.HoraDeslocamento {
		return nil, errorsuc.NewValidationError("o tipo da hora é trabalho ou deslocamento")
	}
	if !dto.Horas.IsPositive() {
		return nil, errorsuc.NewValidationError("informe as horas")
	}
	p, err := uc.Repo.Parametros(ctx)
	if err != nil {
		return nil, err
	}
	h := &dom.Hora{TecnicoCode: dto.TecnicoCode, Data: data, Tipo: tipo, Horas: dto.Horas, CustoHora: p.CustoHora, PrecoHora: p.PrecoHora,
		Cobrado: o.Cobertura.HoraCobrada(tipo), Observacao: limpo(dto.Observacao)}
	if h.TecnicoCode == nil {
		h.TecnicoCode = o.TecnicoCode
	}
	if dto.CustoHora != nil {
		h.CustoHora = *dto.CustoHora
	}
	if dto.PrecoHora != nil {
		h.PrecoHora = *dto.PrecoHora
	}
	if err := uc.Repo.AddHora(ctx, id, h, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

type PecaDTO struct {
	ItemCode      request.TextCode `json:"item_code"`
	Quantidade    decimal.Decimal  `json:"quantidade"`
	WarehouseID   *int64           `json:"warehouse_id,omitempty"`
	PrecoUnitario decimal.Decimal  `json:"preco_unitario"`
	Observacao    *string          `json:"observacao,omitempty"`
}

func (uc *UseCase) AddPeca(ctx context.Context, id int64, dto PecaDTO) (*OSResponse, error) {
	o, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if !dto.Quantidade.IsPositive() {
		return nil, errorsuc.NewValidationError("informe a quantidade da peça")
	}
	item, err := itemresolution.Resolve(ctx, uc.Items, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	almox := dto.WarehouseID
	if almox == nil {
		p, err := uc.Repo.Parametros(ctx)
		if err != nil {
			return nil, err
		}
		almox = p.WarehouseID
	}
	if almox == nil {
		return nil, errorsuc.NewValidationError("informe o almoxarifado da peça (ou o almoxarifado da assistência nos parâmetros da OS)")
	}
	cobrado := o.Cobertura.PecaCobrada()
	if cobrado && !dto.PrecoUnitario.IsPositive() {
		return nil, errorsuc.NewValidationError("a peça é cobrada do cliente (fora da garantia): informe o preço")
	}
	por := uuid.Nil
	if a := uc.ator(ctx); a != nil {
		por = *a
	}
	pc := &dom.Peca{ItemCode: int64(item.Code), Quantidade: dto.Quantidade, WarehouseID: *almox, PrecoUnitario: dto.PrecoUnitario,
		Cobrado: cobrado, Observacao: limpo(dto.Observacao)}
	if !cobrado {
		pc.PrecoUnitario = decimal.Zero
	}
	if err := uc.Repo.AddPeca(ctx, id, pc, por); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

type DespesaDTO struct {
	Tipo       string           `json:"tipo"`
	Quantidade decimal.Decimal  `json:"quantidade"`
	Custo      *decimal.Decimal `json:"custo,omitempty"`
	Preco      *decimal.Decimal `json:"preco,omitempty"`
	Observacao *string          `json:"observacao,omitempty"`
}

func (uc *UseCase) AddDespesa(ctx context.Context, id int64, dto DespesaDTO) (*OSResponse, error) {
	o, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	tipo := strings.ToUpper(strings.TrimSpace(dto.Tipo))
	switch tipo {
	case "KM", "HOSPEDAGEM", "ALIMENTACAO", "PEDAGIO", "OUTROS":
	default:
		return nil, errorsuc.NewValidationError("tipo de despesa inválido")
	}
	qtd := dto.Quantidade
	if !qtd.IsPositive() {
		qtd = decimal.NewFromInt(1)
	}
	d := &dom.Despesa{Tipo: tipo, Quantidade: qtd, Cobrado: o.Cobertura.DespesaCobrada(), Observacao: limpo(dto.Observacao)}
	if tipo == dom.DespesaKM {
		p, err := uc.Repo.Parametros(ctx)
		if err != nil {
			return nil, err
		}
		d.Custo, d.Preco = p.CustoKM, p.PrecoKM
	}
	if dto.Custo != nil {
		d.Custo = *dto.Custo
	}
	if dto.Preco != nil {
		d.Preco = *dto.Preco
	}
	if d.Custo.IsNegative() || d.Preco.IsNegative() {
		return nil, errorsuc.NewValidationError("custo e preço não podem ser negativos")
	}
	if tipo != dom.DespesaKM && d.Custo.IsZero() {
		return nil, errorsuc.NewValidationError("informe o valor da despesa")
	}
	if !d.Cobrado {
		d.Preco = decimal.Zero
	} else if d.Preco.IsZero() {
		d.Preco = d.Custo // despesa repassada pelo valor gasto
	}
	if err := uc.Repo.AddDespesa(ctx, id, d, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

func (uc *UseCase) Remover(ctx context.Context, id int64, tipo string, lancamento int64) (*OSResponse, error) {
	por := uuid.Nil
	if a := uc.ator(ctx); a != nil {
		por = *a
	}
	if err := uc.Repo.RemoverLancamento(ctx, id, tipo, lancamento, por); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

// ─── Faturamento ─────────────────────────────────────────────────────────────

type FaturamentoDTO struct {
	PaymentTermCode   *int64 `json:"payment_term_code,omitempty"`
	PriceTableCode    *int64 `json:"price_table_code,omitempty"`
	SalesDivisionCode *int64 `json:"sales_division_code,omitempty"`
}

// Faturar gera o pedido de venda do que a OS cobra: as peças cobradas (que saem
// do estoque pela NF do pedido) e uma linha do item de serviço com a mão de
// obra e as despesas cobradas. A OS fica FATURADA com o pedido.
func (uc *UseCase) Faturar(ctx context.Context, id int64, dto FaturamentoDTO) (*OSResponse, error) {
	if uc.SalesOrders == nil {
		return nil, errorsuc.NewValidationError("o faturamento da OS não está disponível")
	}
	r, err := uc.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.Situacao != dom.Concluida {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("só OS concluída é faturada (esta está %s)", dom.Rotulo(r.Situacao)))
	}
	if !r.Totais.ACobrar.IsPositive() {
		return nil, errorsuc.NewValidationError("nada a cobrar: o atendimento foi todo coberto pela garantia")
	}
	servico := decimal.Zero
	for _, h := range r.Horas {
		if h.Cobrado {
			servico = servico.Add(h.Horas.Mul(h.PrecoHora))
		}
	}
	for _, d := range r.Despesas {
		if d.Cobrado {
			servico = servico.Add(d.Quantidade.Mul(d.Preco))
		}
	}
	p, err := uc.Repo.Parametros(ctx)
	if err != nil {
		return nil, err
	}
	if servico.IsPositive() && p.ItemServicoCode == nil {
		return nil, errorsuc.NewValidationError("cadastre o item de serviço nos parâmetros da OS para faturar a mão de obra e as despesas")
	}
	empresa, err := uc.Auth.EnterpriseCode(ctx)
	if err != nil {
		return nil, err
	}
	actor := uuid.Nil
	if a := uc.ator(ctx); a != nil {
		actor = *a
	}
	nota := fmt.Sprintf("Faturamento da ordem de serviço %d (chamado %d)", r.Numero, deref(r.CallNumero))
	cliente := r.CustomerCode
	f := dom.Faturamento{
		Pedido: &orderentity.SalesOrder{EnterpriseCode: empresa, Status: orderentity.SalesOrderStatusDraft,
			Origin: orderentity.SalesOrderOriginAssistance, EmissionDate: uc.agora(), DigitDate: uc.agora(), CustomerCode: &cliente,
			SalesDivisionCode: dto.SalesDivisionCode, PriceTableCode: dto.PriceTableCode, PaymentTermCode: dto.PaymentTermCode, CurrencyCode: "BRL",
			Notes: &nota, CreatedBy: actor},
		Mudanca: dom.Mudanca{Por: uc.ator(ctx)},
	}
	linha := func(item int64, qtd, preco decimal.Decimal, almox *int64, obs string) {
		f.Itens = append(f.Itens, &orderentity.SalesOrderItem{Sequence: len(f.Itens) + 1, ItemCode: item, DigitDate: uc.agora(),
			WarehouseCode: almox, RequestedQty: qtd, UnitPrice: preco, Status: orderentity.SalesOrderItemStatusOpen, Notes: &obs})
	}
	for _, pc := range r.Pecas {
		if !pc.Cobrado {
			continue
		}
		almox := pc.WarehouseID
		linha(pc.ItemCode, pc.Quantidade, pc.PrecoUnitario, &almox, fmt.Sprintf("Peça da OS %d", r.Numero))
	}
	if servico.IsPositive() {
		linha(*p.ItemServicoCode, decimal.NewFromInt(1), servico.Round(2), nil, fmt.Sprintf("Mão de obra e despesas da OS %d", r.Numero))
	}
	// Pedido, linhas, totais e a OS FATURADA numa transação só.
	if _, err := uc.Repo.Faturar(ctx, id, f); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

func deref(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

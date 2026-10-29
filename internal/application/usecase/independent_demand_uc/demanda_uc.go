package independent_demand_uc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	itemtypes "github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemvo "github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
	prodentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	forecastentity "github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// ParametrosDoPlanejamento lê os parâmetros do MRP (janela de consumo).
type ParametrosDoPlanejamento interface {
	LoadTypedPlanningParams(ctx context.Context) (*mrpentity.TypedPlanningParams, error)
}

// PoliticasDePlanejamento lê a política de planejamento efetiva dos itens com
// regra (VPLA0310).
type PoliticasDePlanejamento interface {
	PoliticasDePlanejamento(ctx context.Context, empresa mrpentity.PoliticaEfetiva) (map[int64]mrpentity.PoliticaEfetiva, error)
}

// PrevisaoDeVendas lê a previsão do ano, para a posição mostrar o mesmo
// consumo do cálculo.
type PrevisaoDeVendas interface {
	ListForecasts(ctx context.Context, year int) ([]*forecastentity.SalesForecast, error)
}

// SaidaDeEstoque lança o movimento de estoque (o caso de uso da Central de
// Estoque, com o tipo de movimento e as regras dele).
type SaidaDeEstoque interface {
	Execute(ctx context.Context, dto request.CreateStockMovementDTO) (*response.StockMovementResponse, error)
}

// CriadorDeOF abre a ordem de produção (o caso de uso da VPRO0900).
type CriadorDeOF interface {
	Execute(ctx context.Context, dto request.CreateProductionOrderDTO) (*prodentity.ProductionOrder, error)
}

// DemandaUseCase é o cadastro e o acompanhamento da demanda independente
// (VPLA0102). Ver .ai/tasks/demanda-independente-enterprise.md.
type DemandaUseCase struct {
	Repo  repository.IndependentDemandRepository
	Acomp repository.AcompanhamentoRepository
	Auth  ports.AuthService
	// Items resolve o código de negócio do item.
	Items any
	// Opcionais: sem eles a posição sai sem o consumo, e "Atender" e "Gerar
	// OF agora" ficam indisponíveis.
	Params ParametrosDoPlanejamento
	// Politicas: a política de planejamento por item (janela de consumo e
	// estratégia). Sem ela, vale a dos parâmetros da empresa.
	Politicas PoliticasDePlanejamento
	Previsoes PrevisaoDeVendas
	Saida     SaidaDeEstoque
	OF        CriadorDeOF
	// Transacao grava as entregas de uma programação juntas (ou todas, ou
	// nenhuma) e faz o "Atender" com a demanda travada junto da saída. Sem ela
	// (testes), grava direto.
	Transacao EmTransacao
}

// Repos são os repositórios e a saída de estoque ligados a uma transação.
type Repos struct {
	Demandas repository.IndependentDemandRepository
	Acomp    repository.AcompanhamentoRepository
	// Saida pode faltar (instalação sem a Central de Estoque).
	Saida SaidaDeEstoque
}

// EmTransacao roda fn com os repositórios ligados a uma transação; erro
// devolvido por fn desfaz tudo.
type EmTransacao func(ctx context.Context, fn func(Repos) error) error

// emTransacao usa a transação configurada ou, sem ela, os repositórios do
// caso de uso direto.
func (uc *DemandaUseCase) emTransacao(ctx context.Context, fn func(Repos) error) error {
	if uc.Transacao != nil {
		return uc.Transacao(ctx, fn)
	}
	return fn(Repos{Demandas: uc.Repo, Acomp: uc.Acomp, Saida: uc.Saida})
}

func (uc *DemandaUseCase) Criar(ctx context.Context, dto request.CreateIndependentDemandDTO) ([]*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanCreateIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	autor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	base, err := uc.montar(ctx, dto.ItemCode, dto.Mask, dto.CostCenterCode, dto.Quantity, dto.DemandDate, dto.Motivo,
		dto.WarehouseID, dto.CustomerCode, dto.Projeto, dto.Prioridade, dto.Observacao)
	if err != nil {
		return nil, err
	}
	base.CreatedBy = autor

	entregas := []entity.Entrega{{Data: base.DemandDate, Quantidade: base.Quantity}}
	if dto.Entregas > 1 {
		var msg string
		entregas, msg = entity.Programar(base.Quantity, base.DemandDate, dto.Entregas, dto.IntervaloDias)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		if dto.CodeDemand > 0 {
			return nil, errorsuc.NewValidationError("a programação numera as entregas sozinha: deixe o número da demanda em branco")
		}
		prog := uuid.New()
		base.Programacao = &prog
	}

	var out []*response.IndependentDemandResponse
	err = uc.emTransacao(ctx, func(r Repos) error {
		out = make([]*response.IndependentDemandResponse, 0, len(entregas))
		if err := r.Acomp.TravarNumeracao(ctx); err != nil {
			return err
		}
		for _, e := range entregas {
			d := *base
			d.DemandDate, d.Quantity = e.Data, e.Quantidade
			criada, err := criarComNumero(ctx, r.Demandas, r.Acomp, &d, dto.CodeDemand)
			if err != nil {
				return err
			}
			criada.ItemBusinessCode, criada.ItemName = base.ItemBusinessCode, base.ItemName
			out = append(out, uc.resposta(criada, entity.Posicao{Situacao: entity.PosicaoAberta, APlanejar: criada.Quantity}, decimal.Zero))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// criarComNumero grava com o número informado ou com o primeiro livre. Na
// transação a numeração está travada; sem ela, duas criações ao mesmo tempo
// podem pegar o mesmo número livre, e a segunda tenta o próximo.
func criarComNumero(ctx context.Context, repo repository.IndependentDemandRepository, acomp repository.AcompanhamentoRepository,
	d *entity.IndependentDemand, informado int64) (*entity.IndependentDemand, error) {
	if informado > 0 {
		d.CodeDemand = informado
		criada, err := repo.Create(ctx, d)
		if errors.Is(err, repository.ErrCodigoEmUso) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("já existe a demanda %d nesta empresa", informado))
		}
		return criada, err
	}
	for tentativa := 0; ; tentativa++ {
		code, err := acomp.NextCode(ctx)
		if err != nil {
			return nil, err
		}
		d.CodeDemand = code
		criada, err := repo.Create(ctx, d)
		if errors.Is(err, repository.ErrCodigoEmUso) && tentativa < 4 {
			continue
		}
		return criada, err
	}
}

func (uc *DemandaUseCase) Alterar(ctx context.Context, dto request.UpdateIndependentDemandDTO) (*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanUpdateIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	autor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	atual, err := uc.Repo.GetByCode(ctx, dto.CodeDemand)
	if err != nil {
		return nil, err
	}
	if atual.Situacao != entity.SituacaoAberta {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a demanda %d está %s: não se altera", dto.CodeDemand, strings.ToLower(atual.Situacao)))
	}
	manterNaoInformados(&dto, atual)
	d, err := uc.montar(ctx, dto.ItemCode, dto.Mask, dto.CostCenterCode, dto.Quantity, dto.DemandDate, dto.Motivo,
		dto.WarehouseID, dto.CustomerCode, dto.Projeto, dto.Prioridade, dto.Observacao)
	if err != nil {
		return nil, err
	}
	if d.ItemCode != atual.ItemCode && atual.Atendida.IsPositive() {
		return nil, errorsuc.NewValidationError("a demanda já tem saída para a finalidade: o item não muda (cancele e cadastre outra)")
	}
	if d.Quantity.LessThan(atual.Atendida) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a quantidade não fica abaixo do que já saiu para a finalidade (%s)", atual.Atendida.String()))
	}
	d.CodeDemand = dto.CodeDemand
	d.UpdatedBy = &autor
	if _, err := uc.Repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return uc.Detalhe(ctx, dto.CodeDemand)
}

// manterNaoInformados: o campo que não veio na alteração fica como está. O
// desktop até a 1.8.0 não conhece motivo, destino, cliente, projeto,
// prioridade nem observação, e a alteração por ele apagava esses campos. Para
// limpar, a tela nova manda 0 (almoxarifado, cliente) ou texto vazio.
func manterNaoInformados(dto *request.UpdateIndependentDemandDTO, atual *entity.IndependentDemand) {
	if dto.Motivo == "" {
		dto.Motivo = atual.Motivo
	}
	if dto.WarehouseID == nil {
		dto.WarehouseID = atual.WarehouseID
	}
	if dto.CustomerCode == nil {
		dto.CustomerCode = atual.CustomerCode
	}
	if dto.Projeto == nil {
		dto.Projeto = atual.Projeto
	}
	if dto.Prioridade == 0 {
		dto.Prioridade = atual.Prioridade
	}
	if dto.Observacao == nil {
		dto.Observacao = atual.Observacao
	}
}

// Cancelar tira a demanda do planejamento sem nada atendido; Encerrar fecha o
// saldo da que já teve saída ou ordem (o histórico fica).
func (uc *DemandaUseCase) Cancelar(ctx context.Context, code int64, motivo string) error {
	return uc.encerrar(ctx, code, entity.SituacaoCancelada, motivo)
}

func (uc *DemandaUseCase) Encerrar(ctx context.Context, code int64, motivo string) error {
	return uc.encerrar(ctx, code, entity.SituacaoEncerrada, motivo)
}

func (uc *DemandaUseCase) encerrar(ctx context.Context, code int64, situacao, motivo string) error {
	if !uc.Auth.CanDeleteIndependentDemand(ctx) {
		return errorsuc.ErrUnauthorized
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return errorsuc.NewValidationError("informe o motivo")
	}
	autor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	atual, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if situacao == entity.SituacaoCancelada && atual.Atendida.IsPositive() {
		return errorsuc.NewValidationError("a demanda já teve saída para a finalidade: encerre o saldo em vez de cancelar")
	}
	ok, err := uc.Acomp.Encerrar(ctx, code, situacao, motivo, autor)
	if err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewValidationError(fmt.Sprintf("a demanda %d não está aberta", code))
	}
	return nil
}

// Atender lança a saída de estoque para a finalidade (amostra, feira, consumo
// interno, assistência, projeto). A demanda de expectativa de venda não se
// atende assim: quem a consome é o pedido de venda.
func (uc *DemandaUseCase) Atender(ctx context.Context, code int64, dto request.AtenderDemandaDTO) (*response.IndependentDemandResponse, error) {
	if uc.Saida == nil {
		return nil, errorsuc.NewValidationError("saída de estoque indisponível nesta instalação")
	}
	if !uc.Auth.CanUpdateIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	// A demanda fica travada da leitura do saldo até a saída gravar: dois
	// "Atender" ao mesmo tempo não passam do saldo.
	err := uc.emTransacao(ctx, func(r Repos) error {
		if r.Saida == nil {
			return errorsuc.NewValidationError("saída de estoque indisponível nesta instalação")
		}
		if err := r.Acomp.TravarDemanda(ctx, code); err != nil {
			return err
		}
		d, err := r.Demandas.GetByCode(ctx, code)
		if err != nil {
			return err
		}
		switch {
		case d.Situacao != entity.SituacaoAberta:
			return errorsuc.NewValidationError(fmt.Sprintf("a demanda %d não está aberta", code))
		case entity.ConsumidaPelaVenda(d.Motivo):
			return errorsuc.NewValidationError("demanda de " + strings.ToLower(entity.DescricaoDoMotivo(d.Motivo)) +
				" não se atende por saída: o pedido de venda a consome")
		case !dto.Quantidade.IsPositive():
			return errorsuc.NewValidationError("a quantidade precisa ser maior que zero")
		case dto.Quantidade.GreaterThan(d.Saldo()):
			return errorsuc.NewValidationError(fmt.Sprintf("o saldo a atender é %s", d.Saldo().String()))
		case dto.WarehouseID <= 0:
			return errorsuc.NewValidationError("informe o almoxarifado de onde o material sai")
		case strings.TrimSpace(dto.MovementTypeCode) == "":
			return errorsuc.NewValidationError("informe o tipo de movimento da saída")
		}
		mask := ""
		if d.Mask != nil {
			mask = *d.Mask
		}
		ref, refCode := entity.ReferenciaAtendimento, d.CodeDemand
		nota := fmt.Sprintf("Demanda %d — %s", d.CodeDemand, entity.DescricaoDoMotivo(d.Motivo))
		if dto.Observacao != nil && strings.TrimSpace(*dto.Observacao) != "" {
			nota += ": " + strings.TrimSpace(*dto.Observacao)
		}
		tipo := strings.TrimSpace(dto.MovementTypeCode)
		var centro *int32
		if d.CostCenterCode != nil {
			c := int32(*d.CostCenterCode)
			centro = &c
		}
		_, err = r.Saida.Execute(ctx, request.CreateStockMovementDTO{
			ItemCode: d.ItemCode, Mask: mask, WarehouseID: dto.WarehouseID, MovementType: "OUT", Quantity: dto.Quantidade,
			ReferenceType: &ref, ReferenceCode: &refCode, Notes: &nota, MovementTypeCode: &tipo, CostCenterCode: centro,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return uc.Detalhe(ctx, code)
}

// GerarOF abre a OF agora, vinculada à demanda (urgência: não espera o MRP).
// A OF entra como suprimento firme no próximo cálculo, que só complementa os
// componentes. A OF cobre o que nenhuma ordem ligada cobre ainda (sem ordem):
// sem isso, um segundo clique abria outra OF da demanda inteira. Pode passar
// do sem ordem (lote mínimo, múltiplo), mas não nasce quando tudo já tem
// ordem; e a OF e o vínculo gravam na mesma transação.
func (uc *DemandaUseCase) GerarOF(ctx context.Context, code int64, dto request.GerarOFDaDemandaDTO) (*response.IndependentDemandResponse, error) {
	if uc.OF == nil {
		return nil, errorsuc.NewValidationError("abertura de OF indisponível nesta instalação")
	}
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	d, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if d.Situacao != entity.SituacaoAberta {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a demanda %d não está aberta", code))
	}
	// Só item fabricado tem OF: o comprado é atendido pela compra que o MRP
	// sugere, e o de terceiro ou serviço não gera ordem. Antes a OF de item
	// comprado esbarrava na falta de roteiro, com uma mensagem que não dizia
	// o motivo.
	if msg, err := uc.itemSemOF(ctx, d.ItemCode); err != nil {
		return nil, err
	} else if msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	pos, err := uc.posicionar(ctx, []*entity.IndependentDemand{d})
	if err != nil {
		return nil, err
	}
	ordens, err := uc.Acomp.Ordens(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	semOrdem := entity.SemOrdem(pos[0].APlanejar, entity.Coberta(ordens))
	if !semOrdem.IsPositive() {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a demanda %d já está coberta pelas ordens ligadas (ou não tem nada a planejar): não há saldo sem ordem para outra OF", code))
	}
	qtd := dto.Quantidade
	if !qtd.IsPositive() {
		qtd = semOrdem
	}
	mask := ""
	if d.Mask != nil {
		mask = *d.Mask
	}
	nota := fmt.Sprintf("Demanda independente %d — %s", d.CodeDemand, entity.DescricaoDoMotivo(d.Motivo))
	if dto.Notes != nil && strings.TrimSpace(*dto.Notes) != "" {
		nota += ": " + strings.TrimSpace(*dto.Notes)
	}
	fim := dto.EndDate
	if fim == nil || strings.TrimSpace(*fim) == "" {
		v := d.DemandDate.Format("2006-01-02")
		fim = &v
	}
	// A demanda guarda o código do centro; a OF, o id.
	var centroID *int64
	if d.CostCenterCode != nil {
		if centroID, err = uc.Acomp.IDDoCentro(ctx, *d.CostCenterCode); err != nil {
			return nil, err
		}
	}
	// O item vai pela chave interna: a demanda já guarda o código resolvido.
	ctxItem := context.WithValue(ctx, contextkey.ItemCodeTranslatedKey, true)
	if _, err := uc.OF.Execute(ctxItem, request.CreateProductionOrderDTO{
		ItemCode: request.TextCode(strconv.FormatInt(d.ItemCode, 10)), Mask: mask, PlannedQty: qtd,
		StartDate: dto.StartDate, EndDate: fim, CostCenterID: centroID, WarehouseID: d.WarehouseID,
		Priority: dto.Priority, Notes: &nota,
		VinculoDemanda: &prodentity.VinculoDemanda{DemandID: d.ID, Quantidade: decimal.Min(qtd, semOrdem), OrdensVistas: len(ordens)},
	}); err != nil {
		return nil, err
	}
	return uc.Detalhe(ctx, code)
}

// itemSemOF diz por que o item não tem OF (vazio = fabricado). Sem o
// cadastro à mão (montagem sem repositório de itens), deixa a OF decidir.
func (uc *DemandaUseCase) itemSemOF(ctx context.Context, itemCode int64) (string, error) {
	repo, ok := uc.Items.(interface {
		FindItemByCode(ctx context.Context, code itemvo.ItemCode) (*itementity.Item, error)
	})
	if !ok {
		return "", nil
	}
	code, err := itemvo.NewItemCode(itemCode)
	if err != nil {
		return "", nil
	}
	it, err := repo.FindItemByCode(ctx, code)
	if err != nil {
		return "", err
	}
	switch it.Engineering.Type {
	case itemtypes.COMPRADO:
		return fmt.Sprintf("o item %s é comprado: a demanda é atendida pela compra que o MRP sugere (VMRP0100), não por OF", it.BusinessCode), nil
	case itemtypes.DE_TERCEIRO, itemtypes.SERVICO:
		return fmt.Sprintf("o item %s não é fabricado (de terceiro ou serviço): não gera OF", it.BusinessCode), nil
	}
	return "", nil
}

// Buscar é a lista de acompanhamento, com a posição de cada demanda. item é
// o código de negócio (opcional).
func (uc *DemandaUseCase) Buscar(ctx context.Context, f entity.Filtro, item string) ([]*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanListIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if item != "" {
		it, err := itemresolution.Resolve(ctx, uc.Items, request.TextCode(item))
		if err != nil {
			return nil, err
		}
		code := int64(it.Code)
		f.ItemCode = &code
	}
	ds, err := uc.Acomp.Buscar(ctx, f)
	if err != nil {
		return nil, err
	}
	pos, err := uc.posicionar(ctx, ds)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(ds))
	for _, d := range ds {
		ids = append(ids, d.ID)
	}
	cobertura, err := uc.Acomp.Cobertura(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]*response.IndependentDemandResponse, 0, len(ds))
	for i, d := range ds {
		out = append(out, uc.resposta(d, pos[i], cobertura[d.ID]))
	}
	return out, nil
}

func (uc *DemandaUseCase) Detalhe(ctx context.Context, code int64) (*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanViewIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	d, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	pos, err := uc.posicionar(ctx, []*entity.IndependentDemand{d})
	if err != nil {
		return nil, err
	}
	ordens, err := uc.Acomp.Ordens(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	r := uc.resposta(d, pos[0], entity.Coberta(ordens))
	for _, o := range ordens {
		r.Ordens = append(r.Ordens, response.OrdemDaDemandaResponse{PlannedOrderCode: o.PlannedOrderCode, ProductionOrderID: o.ProductionOrderID,
			OrderNumber: o.OrderNumber, Status: o.Status, Quantidade: o.Quantidade, Planejada: o.Planejada, Produzida: o.Produzida})
	}
	return r, nil
}

// parametros são os parâmetros do planejamento com a política de cada item
// (janela de consumo e estratégia), os mesmos que o MRP usa.
func (uc *DemandaUseCase) parametros(ctx context.Context) (*mrpentity.TypedPlanningParams, error) {
	params := mrpentity.DefaultTypedPlanningParams()
	if uc.Params != nil {
		p, err := uc.Params.LoadTypedPlanningParams(ctx)
		if err != nil {
			return nil, err
		}
		params = p
	}
	if uc.Politicas != nil {
		pols, err := uc.Politicas.PoliticasDePlanejamento(ctx, params.PoliticaDaEmpresa())
		if err != nil {
			return nil, err
		}
		params.Politicas = pols
	}
	return params, nil
}

// quadroDeConsumo é o consumo dos itens das demandas como o MRP o calcula:
// as demandas abertas dos mesmos itens, a previsão e as vendas.
type quadroDeConsumo struct {
	params   *mrpentity.TypedPlanningParams
	demandas []*entity.IndependentDemand
	previsao []entity.Consumivel
	vendas   []entity.Venda
	pos      []entity.Posicao
	prev     []entity.PosicaoPrevisao
	sobra    []decimal.Decimal
}

func (uc *DemandaUseCase) quadro(ctx context.Context, ds []*entity.IndependentDemand, itensExtras ...int64) (*quadroDeConsumo, error) {
	params, err := uc.parametros(ctx)
	if err != nil {
		return nil, err
	}
	hoje := mrpservice.Hoje()
	itens := map[int64]bool{}
	var codigos []int64
	marcar := func(item int64) {
		if !itens[item] {
			itens[item] = true
			codigos = append(codigos, item)
		}
	}
	for _, d := range ds {
		marcar(d.ItemCode)
	}
	for _, item := range itensExtras {
		marcar(item)
	}
	// Junta as demandas abertas dos mesmos itens que a lista não trouxe
	// (filtro por motivo ou período), para o consumo ser o mesmo do MRP.
	todas := append([]*entity.IndependentDemand(nil), ds...)
	if len(codigos) > 0 {
		abertas, err := uc.Repo.List(ctx)
		if err != nil {
			return nil, err
		}
		presentes := map[int64]bool{}
		for _, d := range ds {
			presentes[d.CodeDemand] = true
		}
		for _, d := range abertas {
			if itens[d.ItemCode] && !presentes[d.CodeDemand] {
				todas = append(todas, d)
			}
		}
	}
	var previsao []entity.Consumivel
	if uc.Previsoes != nil && len(codigos) > 0 {
		for _, ano := range []int{hoje.Year(), hoje.Year() + 1} {
			fs, err := uc.Previsoes.ListForecasts(ctx, ano)
			if err != nil {
				return nil, err
			}
			var dosItens []*forecastentity.SalesForecast
			for _, f := range fs {
				if itens[f.ItemCode] {
					dosItens = append(dosItens, f)
				}
			}
			previsao = append(previsao, mrpservice.ConsumiveisDaPrevisao(dosItens, params.DesconsideraSemanasPassadas, hoje)...)
		}
	}
	vendas, err := uc.Acomp.Vendas(ctx, codigos)
	if err != nil {
		return nil, err
	}
	pos, prev, sobra := entity.PosicionarComPolitica(todas, previsao, vendas, mrpservice.PoliticaDeConsumo(params), hoje)
	return &quadroDeConsumo{params: params, demandas: todas, previsao: previsao, vendas: vendas, pos: pos, prev: prev, sobra: sobra}, nil
}

// posicionar calcula a posição como o MRP: as demandas abertas dos mesmos
// itens e a previsão entram no consumo, para a venda ser repartida igual.
func (uc *DemandaUseCase) posicionar(ctx context.Context, ds []*entity.IndependentDemand) ([]entity.Posicao, error) {
	q, err := uc.quadro(ctx, ds)
	if err != nil {
		return nil, err
	}
	return q.pos[:len(ds)], nil
}

// montar valida e resolve o que vem da tela.
func (uc *DemandaUseCase) montar(ctx context.Context, item request.TextCode, mask *string, centro *int64, qtd decimal.Decimal,
	data, motivo string, almox, cliente *int64, projeto *string, prioridade int, obs *string) (*entity.IndependentDemand, error) {
	dataNec, msg := datetime.Obrigatoria("data da necessidade", data)
	if msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	it, err := itemresolution.ResolveActive(ctx, uc.Items, item)
	if err != nil {
		return nil, err
	}
	if mask != nil {
		m := strings.TrimSpace(*mask)
		mask = &m
		if m == "" {
			mask = nil
		} else if err := itemresolution.ValidateMask(ctx, uc.Items, int64(it.Code), m); err != nil {
			return nil, err
		}
	}
	if motivo == "" {
		motivo = entity.MotivoEstoque
	}
	// Item sob encomenda: só o pedido gera necessidade. A expectativa de
	// venda não entraria no MRP — melhor recusar do que cadastrar em vão.
	if entity.ConsumidaPelaVenda(motivo) {
		params, err := uc.parametros(ctx)
		if err != nil {
			return nil, err
		}
		if params.Politica(int64(it.Code)).SobEncomenda() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"o item %s é planejado sob encomenda (VPLA0310): só o pedido de venda gera necessidade. "+
					"Use um motivo de finalidade própria (amostra, feira, consumo interno, assistência, projeto) ou mude a política do item",
				it.BusinessCode))
		}
	}
	if prioridade == 0 {
		prioridade = 3
	}
	d := &entity.IndependentDemand{
		ItemCode: int64(it.Code), Mask: mask, CostCenterCode: centro, Quantity: qtd, DemandDate: dataNec,
		Motivo: motivo, WarehouseID: positivo(almox), CustomerCode: positivo(cliente), Projeto: textoOuNil(projeto),
		Prioridade: prioridade, Observacao: textoOuNil(obs), Situacao: entity.SituacaoAberta, IsActive: true,
		ItemBusinessCode: string(it.BusinessCode), ItemName: it.Name,
	}
	if msg := d.Validar(); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	if centro != nil && *centro > 0 {
		if _, err := uc.Acomp.IDDoCentro(ctx, *centro); err != nil {
			return nil, err
		}
	} else {
		d.CostCenterCode = nil
	}
	// O almoxarifado vai para a OF e para a saída: tem de ser desta empresa.
	if d.WarehouseID != nil {
		ok, err := uc.Acomp.AlmoxarifadoDaEmpresa(ctx, *d.WarehouseID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("almoxarifado %d não encontrado nesta empresa", *d.WarehouseID))
		}
	}
	if d.CustomerCode != nil {
		ok, err := uc.Acomp.ClienteDaEmpresa(ctx, *d.CustomerCode)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("cliente %d não encontrado nesta empresa", *d.CustomerCode))
		}
	}
	return d, nil
}

func (uc *DemandaUseCase) resposta(d *entity.IndependentDemand, p entity.Posicao, coberta decimal.Decimal) *response.IndependentDemandResponse {
	familia := "FINALIDADE"
	if entity.ConsumidaPelaVenda(d.Motivo) {
		familia = "VENDA"
	}
	var venceEm *time.Time
	if p.VenceEm != nil {
		v := *p.VenceEm
		venceEm = &v
	}
	return &response.IndependentDemandResponse{
		CodeDemand: d.CodeDemand, ItemCode: d.ItemCode, ItemBusinessCode: d.ItemBusinessCode, ItemName: d.ItemName,
		Mask: d.Mask, CostCenterCode: d.CostCenterCode, Quantity: d.Quantity, DemandDate: d.DemandDate, IsActive: d.IsActive,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, CreatedBy: d.CreatedBy,
		Motivo: d.Motivo, MotivoDescricao: entity.DescricaoDoMotivo(d.Motivo), Familia: familia,
		WarehouseID: d.WarehouseID, WarehouseNome: d.WarehouseNome, CustomerCode: d.CustomerCode, CustomerNome: d.CustomerNome,
		Projeto: d.Projeto, Prioridade: d.Prioridade,
		Observacao: d.Observacao, Programacao: d.Programacao, Situacao: d.Situacao, EncerradaEm: d.EncerradaEm,
		MotivoEncerramento: d.MotivoEncerramento,
		Posicao:            p.Situacao, Consumida: p.Consumida, Atendida: d.Atendida, APlanejar: p.APlanejar, VenceEm: venceEm,
		Coberta: coberta, SemOrdem: entity.SemOrdem(p.APlanejar, coberta),
		Consumo: consumoResp(p.Consumo),
	}
}

func consumoResp(cs []entity.ConsumoDaVenda) []response.ConsumoDaVendaResponse {
	if len(cs) == 0 {
		return nil
	}
	out := make([]response.ConsumoDaVendaResponse, 0, len(cs))
	for _, c := range cs {
		out = append(out, response.ConsumoDaVendaResponse{Pedido: c.Venda.Pedido, Sequencia: c.Venda.Sequencia, Cliente: c.Venda.Cliente,
			Data: c.Venda.Data, Quantidade: c.Quantidade})
	}
	return out
}

// mascaraDaChave tira a máscara da chave de consumo (item|máscara).
func mascaraDaChave(chave string) string {
	if i := strings.IndexByte(chave, '|'); i >= 0 {
		return chave[i+1:]
	}
	return ""
}

// ConsumoDoItem é a visão do consumo do item (SAP MD73): previsão por semana,
// demandas de expectativa de venda e vendas, com quem consumiu o quê.
func (uc *DemandaUseCase) ConsumoDoItem(ctx context.Context, item request.TextCode) (*response.ConsumoDoItemResponse, error) {
	if !uc.Auth.CanListIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	it, err := itemresolution.Resolve(ctx, uc.Items, item)
	if err != nil {
		return nil, err
	}
	code := int64(it.Code)
	q, err := uc.quadro(ctx, nil, code)
	if err != nil {
		return nil, err
	}
	pol := q.params.Politica(code)
	out := &response.ConsumoDoItemResponse{
		ItemBusinessCode: string(it.BusinessCode), ItemName: it.Name, Estrategia: pol.Estrategia,
		ConsumoDiasAntes: pol.ConsumoDiasAntes, ConsumoDiasDepois: pol.ConsumoDiasDepois, SobEncomenda: pol.SobEncomenda(),
		Previsoes: []response.PrevisaoConsumidaResponse{}, Demandas: []response.DemandaConsumidaResponse{}, Vendas: []response.VendaDoConsumoResponse{},
	}
	for i, p := range q.previsao {
		pp := q.prev[i]
		out.Previsoes = append(out.Previsoes, response.PrevisaoConsumidaResponse{Mask: mascaraDaChave(p.Chave), Semana: p.Data,
			Quantidade: p.Quantidade, Consumida: pp.Consumida, Resto: pp.Resto, Consumo: consumoResp(pp.Consumo)})
		out.TotalExpectativa = out.TotalExpectativa.Add(p.Quantidade)
		out.TotalConsumido = out.TotalConsumido.Add(pp.Consumida)
		out.TotalAPlanejar = out.TotalAPlanejar.Add(pp.Resto)
	}
	for i, d := range q.demandas {
		if !entity.ConsumidaPelaVenda(d.Motivo) {
			continue
		}
		p := q.pos[i]
		out.Demandas = append(out.Demandas, response.DemandaConsumidaResponse{CodeDemand: d.CodeDemand, Mask: d.Mask, Motivo: d.Motivo,
			Data: d.DemandDate, Quantidade: d.Quantity, Consumida: p.Consumida, APlanejar: p.APlanejar, Posicao: p.Situacao,
			Consumo: consumoResp(p.Consumo)})
		out.TotalExpectativa = out.TotalExpectativa.Add(d.Quantity)
		out.TotalConsumido = out.TotalConsumido.Add(p.Consumida)
		out.TotalAPlanejar = out.TotalAPlanejar.Add(p.APlanejar)
	}
	for i, v := range q.vendas {
		acima := q.sobra[i]
		out.Vendas = append(out.Vendas, response.VendaDoConsumoResponse{Pedido: v.Pedido, Sequencia: v.Sequencia, Cliente: v.Cliente,
			Mask: mascaraDaChave(v.Chave), Data: v.Data, Quantidade: v.Quantidade,
			Consumiu: decimal.Max(decimal.Zero, v.Quantidade.Sub(acima)), Acima: acima})
		out.TotalVendido = out.TotalVendido.Add(v.Quantidade)
		out.TotalAcima = out.TotalAcima.Add(acima)
	}
	return out, nil
}

func positivo(v *int64) *int64 {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}

func textoOuNil(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil
	}
	return &t
}

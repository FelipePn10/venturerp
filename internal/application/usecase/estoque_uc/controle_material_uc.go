package estoque_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// SituacoesLote lista o cadastro de situações de lote.
func (uc *UseCase) SituacoesLote(ctx context.Context) ([]entity.SituacaoLote, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ListarSituacoesLote(ctx)
}

// SalvarSituacaoLote cria ou altera uma situação.
func (uc *UseCase) SalvarSituacaoLote(ctx context.Context, s entity.SituacaoLote) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	s.Code = strings.ToUpper(strings.TrimSpace(s.Code))
	s.Description = strings.TrimSpace(s.Description)
	if s.Code == "" || len(s.Code) > 20 || s.Description == "" {
		return errorsuc.NewValidationError("informe o código (até 20 letras) e a descrição da situação")
	}
	return uc.Ops.SalvarSituacaoLote(ctx, s)
}

// Parametros lê as regras de estoque da empresa.
func (uc *UseCase) Parametros(ctx context.Context) (entity.ParametrosEstoque, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return entity.ParametrosEstoque{}, err
	}
	return uc.Ops.ParametrosDeEstoque(ctx)
}

// SalvarParametros grava as regras de estoque da empresa.
func (uc *UseCase) SalvarParametros(ctx context.Context, p entity.ParametrosEstoque) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	if p.ExpiredLotAction != entity.LoteVencidoBloqueia && p.ExpiredLotAction != entity.LoteVencidoAvisa {
		return errorsuc.NewValidationError("escolha o que fazer com o lote vencido")
	}
	if p.RetroactiveMonths != nil && *p.RetroactiveMonths < 0 {
		return errorsuc.NewValidationError("o limite de meses retroativos não pode ser negativo")
	}
	for _, t := range []*decimal.Decimal{p.ToleranceA, p.ToleranceB, p.ToleranceC} {
		if t != nil && (t.IsNegative() || t.GreaterThan(decimal.NewFromInt(100))) {
			return errorsuc.NewValidationError("a tolerância por classe deve ficar entre 0 e 100%")
		}
	}
	if p.ObsoleteDays != nil && *p.ObsoleteDays <= 0 {
		return errorsuc.NewValidationError("os dias sem movimento da provisão devem ser maiores que zero")
	}
	if p.ObsoleteProvisionPct != nil && (p.ObsoleteProvisionPct.IsNegative() || p.ObsoleteProvisionPct.GreaterThan(decimal.NewFromInt(100))) {
		return errorsuc.NewValidationError("o percentual da provisão deve ficar entre 0 e 100%")
	}
	if p.ConsignmentDays != nil && *p.ConsignmentDays <= 0 {
		return errorsuc.NewValidationError("o prazo da consignação deve ser maior que zero")
	}
	if p.ConsignmentWarehouseID != nil {
		if err := uc.Ops.ConferirAlmoxDeConsignacao(ctx, *p.ConsignmentWarehouseID); err != nil {
			return err
		}
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Ops.SalvarParametrosDeEstoque(ctx, p, ator)
}

// CartoesKanban lista os cartões da empresa.
func (uc *UseCase) CartoesKanban(ctx context.Context) ([]entity.CartaoKanban, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ListarCartoesKanban(ctx)
}

// SalvarCartaoKanban cria ou altera um cartão.
func (uc *UseCase) SalvarCartaoKanban(ctx context.Context, c entity.CartaoKanban) (*entity.CartaoKanban, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	if c.ItemCode <= 0 {
		return nil, errorsuc.NewValidationError("escolha o item do cartão")
	}
	if c.CardCount <= 0 || !c.QuantityPerCard.IsPositive() {
		return nil, errorsuc.NewValidationError("informe o número de cartões e a quantidade por cartão, maiores que zero")
	}
	if c.ReorderPoint.IsNegative() {
		return nil, errorsuc.NewValidationError("o ponto de acionamento não pode ser negativo")
	}
	if c.Status != "INACTIVE" {
		c.Status = "ACTIVE"
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Ops.SalvarCartaoKanban(ctx, c, ator)
}

// AcionarCartaoKanban transforma cartões vazios em reposição: requisição de compra
// para item comprado e ordem planejada para item fabricado.
func (uc *UseCase) AcionarCartaoKanban(ctx context.Context, id int64, cartoes int) (*entity.ResultadoGeracaoReposicao, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	c, err := uc.Ops.ObterCartaoKanban(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status != "ACTIVE" {
		return nil, errorsuc.NewValidationError("este cartão está inativo")
	}
	if cartoes <= 0 {
		cartoes = 1
	}
	if cartoes > c.CardCount {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o circuito tem %d cartões", c.CardCount))
	}
	origem := entity.OrigemCompra
	if c.Fabricado {
		origem = entity.OrigemProducao
	}
	nota := fmt.Sprintf("Gerada pelo cartão kanban %d", c.ID)
	return uc.GerarReposicao(ctx, request.GerarReposicaoDTO{Notes: &nota, Lines: []request.LinhaReposicaoDTO{{
		ItemCode: c.ItemCode, WarehouseID: c.WarehouseID, SourceType: origem,
		Quantity: c.QuantityPerCard.Mul(decimal.NewFromInt(int64(cartoes))).String(),
	}}})
}

// LinhasDeCargaParaOnda junta as linhas pendentes dos pedidos da carga.
func (uc *UseCase) LinhasDeCargaParaOnda(ctx context.Context, carga, almox int64) ([]entity.LinhaPedidoOnda, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	if carga <= 0 || almox <= 0 {
		return nil, errorsuc.NewValidationError("informe a carga e o almoxarifado")
	}
	return uc.Ops.LinhasDeCargaParaOnda(ctx, carga, almox)
}

// IdadeDosLotes lista o saldo em lote do mais antigo para o mais novo.
func (uc *UseCase) IdadeDosLotes(ctx context.Context, almox *int64) ([]entity.IdadeDoLote, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.IdadeDosLotes(ctx, almox)
}

// NiveisPeloConsumo sugere os níveis da política a partir do consumo real.
func (uc *UseCase) NiveisPeloConsumo(ctx context.Context, item int64, mask string, almox *int64, meses, prazo, diasSeguranca int) (*entity.NiveisSugeridos, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	if item <= 0 {
		return nil, errorsuc.NewValidationError("escolha o item")
	}
	if meses <= 0 {
		meses = 6
	}
	if diasSeguranca <= 0 {
		diasSeguranca = 7
	}
	return uc.Ops.NiveisPeloConsumo(ctx, item, strings.TrimSpace(mask), almox, min(meses, maxMesesDeAnalise), prazo, diasSeguranca)
}

// MotivosDeDivergencia lista os motivos da empresa.
func (uc *UseCase) MotivosDeDivergencia(ctx context.Context) ([]entity.MotivoDivergencia, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ListarMotivosDivergencia(ctx)
}

// SalvarMotivoDeDivergencia cria ou altera um motivo.
func (uc *UseCase) SalvarMotivoDeDivergencia(ctx context.Context, m entity.MotivoDivergencia) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	m.Code = strings.ToUpper(strings.TrimSpace(m.Code))
	m.Description = strings.TrimSpace(m.Description)
	if m.Code == "" || len(m.Code) > 20 {
		return errorsuc.NewValidationError("informe o código do motivo, com até 20 caracteres")
	}
	if m.Description == "" {
		return errorsuc.NewValidationError("informe a descrição do motivo")
	}
	return uc.Ops.SalvarMotivoDivergencia(ctx, m)
}

// ControleDoItem lê como o estoque do item é controlado.
func (uc *UseCase) ControleDoItem(ctx context.Context, item int64) (entity.ControleDoItem, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return entity.ControleDoItem{}, err
	}
	return uc.Ops.ControleDoItem(ctx, item)
}

// SalvarControleDoItem liga o controle de lote e de endereço do item.
func (uc *UseCase) SalvarControleDoItem(ctx context.Context, c entity.ControleDoItem) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	if c.LotIssueRule == "" {
		c.LotIssueRule = entity.BaixaFEFO
	}
	if c.LotIssueRule != entity.BaixaFEFO && c.LotIssueRule != entity.BaixaFIFO {
		return errorsuc.NewValidationError("a regra de baixa do lote é FEFO ou FIFO")
	}
	return uc.Ops.SalvarControleDoItem(ctx, c)
}

// MudarSituacaoDoLote troca a situação e o grau do lote, com motivo.
func (uc *UseCase) MudarSituacaoDoLote(ctx context.Context, m entity.MudancaSituacaoLote) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	m.Lot, m.Reason = strings.TrimSpace(m.Lot), strings.TrimSpace(m.Reason)
	if m.ItemCode <= 0 || m.Lot == "" {
		return errorsuc.NewValidationError("informe o item e o lote")
	}
	if len([]rune(m.Reason)) < 5 {
		return errorsuc.NewValidationError("informe o motivo da mudança")
	}
	m.Status, m.Grade = vazioNil(m.Status), vazioNil(m.Grade)
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Ops.MudarSituacaoDoLote(ctx, m, ator)
}

// ImplantarLote dá lote a um saldo que entrou sem lote.
func (uc *UseCase) ImplantarLote(ctx context.Context, i entity.ImplantacaoLote) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	i.Lot, i.Address = strings.TrimSpace(i.Lot), strings.TrimSpace(i.Address)
	if i.ItemCode <= 0 || i.WarehouseID <= 0 || i.Lot == "" {
		return errorsuc.NewValidationError("informe o item, o almoxarifado e o lote")
	}
	if !i.Quantity.GreaterThan(decimal.Zero) {
		return errorsuc.NewValidationError("a quantidade precisa ser maior que zero")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Ops.ImplantarLote(ctx, i, ator)
}

// ToleranciasPorPerfil lista os perfis da empresa com o limite de ajuste de cada um.
func (uc *UseCase) ToleranciasPorPerfil(ctx context.Context) ([]entity.ToleranciaDoPerfil, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ToleranciasPorPerfil(ctx)
}

// SalvarToleranciaDoPerfil grava até quanto o perfil fecha inventário sem alçada.
func (uc *UseCase) SalvarToleranciaDoPerfil(ctx context.Context, t entity.ToleranciaDoPerfil) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	if t.PerfilID <= 0 {
		return errorsuc.NewValidationError("escolha o perfil")
	}
	if t.ValorDocumento == nil && t.ValorLinha == nil {
		return errorsuc.NewValidationError("informe o limite por inventário ou por linha")
	}
	for _, v := range []*decimal.Decimal{t.ValorDocumento, t.ValorLinha} {
		if v != nil && v.IsNegative() {
			return errorsuc.NewValidationError("o limite de ajuste não pode ser negativo")
		}
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Ops.SalvarToleranciaDoPerfil(ctx, t, ator)
}

func (uc *UseCase) ExcluirToleranciaDoPerfil(ctx context.Context, perfil int64) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	return uc.Ops.ExcluirToleranciaDoPerfil(ctx, perfil)
}

// Cadernos lista os cadernos de contagem do inventário.
func (uc *UseCase) Cadernos(ctx context.Context, inventario int64) ([]entity.CadernoInventario, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.Cadernos(ctx, inventario)
}

// GerarCadernos divide o inventário em cadernos de até `linhas` linhas.
func (uc *UseCase) GerarCadernos(ctx context.Context, inventario int64, linhas int) ([]entity.CadernoInventario, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	if linhas <= 0 || linhas > 500 {
		return nil, errorsuc.NewValidationError("informe de 1 a 500 linhas por caderno")
	}
	return uc.Ops.GerarCadernos(ctx, inventario, linhas)
}

func (uc *UseCase) MarcarCadernoImpresso(ctx context.Context, inventario int64, numero int) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Ops.MarcarCadernoImpresso(ctx, inventario, numero, ator)
}

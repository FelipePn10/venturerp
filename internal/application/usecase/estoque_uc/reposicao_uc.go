package estoque_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

func (uc *UseCase) PoliticasDeReposicao(ctx context.Context) ([]entity.PoliticaReposicao, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ListarPoliticasDeReposicao(ctx)
}

func quantidadeOpcional(campo string, v *string) (*decimal.Decimal, error) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil, nil
	}
	q, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(*v), ",", "."))
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s: informe um número", campo))
	}
	return &q, nil
}

func (uc *UseCase) SalvarPoliticaDeReposicao(ctx context.Context, dto request.PoliticaReposicaoDTO) (*entity.PoliticaReposicao, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	p := entity.PoliticaReposicao{
		ItemCode: dto.ItemCode, WarehouseID: dto.WarehouseID, Origem: strings.ToUpper(strings.TrimSpace(dto.SourceType)),
		AlmoxOrigem: dto.SourceWarehouseID, Ativa: dto.IsActive == nil || *dto.IsActive, CreatedBy: ator,
	}
	if p.Minimo, err = quantidadeOpcional("mínimo", dto.MinimumQty); err != nil {
		return nil, err
	}
	if p.Maximo, err = quantidadeOpcional("máximo", dto.MaximumQty); err != nil {
		return nil, err
	}
	if p.Seguranca, err = quantidadeOpcional("segurança", dto.SafetyQty); err != nil {
		return nil, err
	}
	if err := p.Validar(); err != nil {
		return nil, err
	}
	return uc.Ops.SalvarPoliticaDeReposicao(ctx, p)
}

func (uc *UseCase) ExcluirPoliticaDeReposicao(ctx context.Context, id int64) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	return uc.Ops.ExcluirPoliticaDeReposicao(ctx, id)
}

// GerarReposicao transforma as sugestões aceitas nos documentos de cada origem:
// transferência vira ordem de reposição por par de almoxarifados, compra vira uma
// requisição de compra e produção vira ordem planejada. Cada documento é do seu
// módulo, então não há transação única: se um falhar, a mensagem diz o que já foi
// gerado, e o recálculo já conta esses documentos como a caminho.
func (uc *UseCase) GerarReposicao(ctx context.Context, dto request.GerarReposicaoDTO) (*entity.ResultadoGeracaoReposicao, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	linhas, err := linhasDeGeracao(dto)
	if err != nil {
		return nil, err
	}
	compra := false
	producao := false
	for _, l := range linhas {
		compra = compra || l.Origem == entity.OrigemCompra
		producao = producao || l.Origem == entity.OrigemProducao
	}
	if compra && (!uc.Auth.CanCreatePurchaseOrder(ctx) || uc.Compras == nil) {
		return nil, errorsuc.ErrUnauthorized
	}
	if producao && (!uc.Auth.CanCreatePlannedOrder(ctx) || uc.Planejadas == nil) {
		return nil, errorsuc.ErrUnauthorized
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	nota := "Gerada pela reposição de estoque"
	if v := vazioNil(dto.Notes); v != nil {
		nota = strings.TrimSpace(*v)
	}
	out := &entity.ResultadoGeracaoReposicao{Transferencias: []int64{}, OrdensPlanejadas: []int64{}}
	falhou := func(err error) (*entity.ResultadoGeracaoReposicao, error) {
		feito := descreverGerado(out)
		if feito == "" {
			return nil, err
		}
		motivo := "Calcule de novo para ver o que falta."
		if v, ok := errorsuc.AsValidation(err); ok {
			motivo = v.Error() + "."
		}
		return nil, errorsuc.NewConflictError(fmt.Sprintf("A reposição parou no meio e já gerou %s. %s", feito, motivo))
	}

	// Transferência primeiro: ela reserva na origem antes de a compra olhar o saldo.
	for _, par := range paresDeTransferencia(linhas) {
		req := &entity.RequisicaoMaterial{WarehouseID: par.origem, DestinoID: &par.destino, Notes: &nota, CreatedBy: ator}
		for _, l := range par.linhas {
			req.Itens = append(req.Itens, entity.RequisicaoMaterialItem{ItemCode: l.ItemCode, Quantity: l.Quantidade})
		}
		criada, err := uc.Ops.CriarRequisicaoMaterial(ctx, req)
		if err != nil {
			return falhou(err)
		}
		out.Transferencias = append(out.Transferencias, criada.Code)
	}
	if compra {
		itens := []request.RequisitionItemInput{}
		for _, l := range linhas {
			if l.Origem == entity.OrigemCompra {
				itens = append(itens, request.RequisitionItemInput{ItemCode: l.ItemCode, WarehouseID: l.WarehouseID, Quantity: l.Quantidade})
			}
		}
		criada, err := uc.Compras.Create(ctx, request.CreatePurchaseRequisitionDTO{Notes: &nota, Items: itens})
		if err != nil {
			return falhou(err)
		}
		out.RequisicaoCompra = &criada.Code
	}
	hoje, err := uc.Ops.HojeDaEmpresa(ctx)
	if err != nil {
		return falhou(err)
	}
	for _, l := range linhas {
		if l.Origem != entity.OrigemProducao {
			continue
		}
		ordem, err := uc.Planejadas.Execute(ctx, request.CreatePlannedOrderDTO{
			ItemCode: l.ItemCode, Quantity: l.Quantidade, OrderType: string(types.OrderProduction),
			DemandType: string(types.DemandReplenishment), NeedDate: hoje.Format("2006-01-02"), WarehouseCode: l.WarehouseID, Notes: &nota, CreatedBy: ator,
		})
		if err != nil {
			return falhou(err)
		}
		out.OrdensPlanejadas = append(out.OrdensPlanejadas, ordem.OrderNumber)
	}
	return out, nil
}

func linhasDeGeracao(dto request.GerarReposicaoDTO) ([]entity.LinhaGeracaoReposicao, error) {
	linhas := make([]entity.LinhaGeracaoReposicao, 0, len(dto.Lines))
	for i, l := range dto.Lines {
		q, err := quantidade(fmt.Sprintf("linha %d: quantidade", i+1), l.Quantity)
		if err != nil {
			return nil, err
		}
		if l.ItemCode <= 0 {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("linha %d: escolha o item", i+1))
		}
		origem := strings.ToUpper(strings.TrimSpace(l.SourceType))
		switch origem {
		case entity.OrigemCompra, entity.OrigemProducao:
		case entity.OrigemTransferencia:
			if l.WarehouseID == nil || l.SourceWarehouseID == nil {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("linha %d: a transferência precisa de origem e destino", i+1))
			}
		default:
			return nil, errorsuc.NewValidationError(fmt.Sprintf("linha %d: escolha a origem da reposição", i+1))
		}
		linhas = append(linhas, entity.LinhaGeracaoReposicao{
			ItemCode: l.ItemCode, WarehouseID: l.WarehouseID, Origem: origem, AlmoxOrigem: l.SourceWarehouseID, Quantidade: q,
		})
	}
	if len(linhas) == 0 {
		return nil, errorsuc.NewValidationError("escolha ao menos um item com quantidade")
	}
	return linhas, nil
}

type parTransferencia struct {
	origem, destino int64
	linhas          []entity.LinhaGeracaoReposicao
}

func paresDeTransferencia(linhas []entity.LinhaGeracaoReposicao) []*parTransferencia {
	pares := map[[2]int64]*parTransferencia{}
	var out []*parTransferencia
	for _, l := range linhas {
		if l.Origem != entity.OrigemTransferencia {
			continue
		}
		k := [2]int64{*l.AlmoxOrigem, *l.WarehouseID}
		p := pares[k]
		if p == nil {
			p = &parTransferencia{origem: k[0], destino: k[1]}
			pares[k] = p
			out = append(out, p)
		}
		p.linhas = append(p.linhas, l)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].origem != out[j].origem {
			return out[i].origem < out[j].origem
		}
		return out[i].destino < out[j].destino
	})
	return out
}

func descreverGerado(r *entity.ResultadoGeracaoReposicao) string {
	var partes []string
	if n := len(r.Transferencias); n > 0 {
		partes = append(partes, fmt.Sprintf("%d ordem(ns) de reposição", n))
	}
	if r.RequisicaoCompra != nil {
		partes = append(partes, fmt.Sprintf("requisição de compra %d", *r.RequisicaoCompra))
	}
	if n := len(r.OrdensPlanejadas); n > 0 {
		partes = append(partes, fmt.Sprintf("%d ordem(ns) planejada(s)", n))
	}
	return strings.Join(partes, ", ")
}

// AprovacaoDeRequisicao aplica na requisição a decisão da alçada.
type AprovacaoDeRequisicao struct {
	Ops interface {
		LiberarRequisicaoAprovada(ctx context.Context, id int64) error
		ReprovarRequisicaoMaterial(ctx context.Context, id int64) error
	}
}

var _ aprovacao.Efetivador = (*AprovacaoDeRequisicao)(nil)

// Efetivar abre a requisição e reserva o material.
func (a *AprovacaoDeRequisicao) Efetivar(ctx context.Context, documento, _ int64) error {
	return a.Ops.LiberarRequisicaoAprovada(ctx, documento)
}

// Reprovar encerra a requisição sem reserva.
func (a *AprovacaoDeRequisicao) Reprovar(ctx context.Context, documento, _ int64, _ string) error {
	return a.Ops.ReprovarRequisicaoMaterial(ctx, documento)
}

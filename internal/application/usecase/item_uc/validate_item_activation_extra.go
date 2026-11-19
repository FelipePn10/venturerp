package item_uc

import (
	"context"
	"fmt"
	"strings"

	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"github.com/shopspring/decimal"
)

// compradoComProcesso: item cadastrado como COMPRADO que tem estrutura e/ou
// roteiro ativos. O MRP compra item comprado — a estrutura e o roteiro ficam
// esquecidos e a fábrica nunca recebe a OF (o BU-120/50 da Tecnofer).
func (uc *ValidateItemActivationUseCase) compradoComProcesso(ctx context.Context, report *ItemActivationReport, code int64) error {
	var tem []string
	if uc.StructureRepo != nil {
		filhos, err := uc.StructureRepo.GetAllDirectChildren(ctx, code)
		if err != nil {
			return fmt.Errorf("lendo a estrutura do item: %w", err)
		}
		// GetAllDirectChildren já devolve só os componentes ativos.
		if len(filhos) > 0 {
			tem = append(tem, fmt.Sprintf("estrutura com %d componente(s)", len(filhos)))
		}
	}
	if uc.RoutingRepo != nil {
		roteiros, err := uc.RoutingRepo.ListRoutesByItem(ctx, code)
		if err != nil {
			return fmt.Errorf("lendo os roteiros do item: %w", err)
		}
		ativos := 0
		for _, r := range roteiros {
			if r.IsActive {
				ativos++
			}
		}
		if ativos > 0 {
			tem = append(tem, "roteiro de fabricação ativo")
		}
	}
	if len(tem) > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf(
			"item cadastrado como comprado, mas tem %s: o MRP vai comprar o item e não gerar ordem de fabricação — mude o tipo para fabricado ou inative a estrutura/roteiro",
			strings.Join(tem, " e ")))
	}
	return nil
}

// conferirFatores: linha da estrutura cuja quantidade não está na unidade de
// estoque atual do componente (fator velho) ou que não converte.
func (uc *ValidateItemActivationUseCase) conferirFatores(ctx context.Context, report *ItemActivationReport, code int64) error {
	if uc.Fatores == nil {
		return nil
	}
	defasadas, sem, err := uc.Fatores.ConferirDoPai(ctx, code)
	if err != nil {
		return fmt.Errorf("conferindo as unidades da estrutura: %w", err)
	}
	for _, d := range defasadas {
		report.Issues = append(report.Issues, "estrutura com quantidade em unidade velha: "+d+" — será recalculada no próximo MRP; confira o cadastro")
	}
	report.Issues = append(report.Issues, sem...)
	return nil
}

// conferirOperacoesExternas: a operação externa precisa de um item de serviço
// que NÃO seja o próprio produto (senão a requisição compra o produto pronto)
// e de um fornecedor (na operação ou como fonte do item de serviço); sem eles a
// requisição de serviço não sai ou não vira pedido.
func (uc *ValidateItemActivationUseCase) conferirOperacoesExternas(ctx context.Context, report *ItemActivationReport, code int64) error {
	if uc.RoutingRepo == nil {
		return nil
	}
	leitor, ok := uc.RoutingRepo.(externasDoRoteiro)
	if !ok {
		return nil
	}
	ops, err := routingrepo.OperacoesExternasDaOrdem(ctx, leitor, code, "", decimal.NewFromInt(1))
	if err != nil {
		return fmt.Errorf("lendo as operações externas do roteiro: %w", err)
	}
	for _, op := range ops {
		switch {
		case op.ServiceItemCode == nil:
			report.Warnings = append(report.Warnings, fmt.Sprintf("operação externa %s sem item de serviço: a OF não gera requisição de serviço para ela", op.OperationName))
		case *op.ServiceItemCode == code:
			report.Issues = append(report.Issues, fmt.Sprintf("operação externa %s usa o próprio produto como item de serviço: a requisição compraria o produto pronto — cadastre o item do serviço (ex.: zincagem)", op.OperationName))
		case op.SupplierID == nil:
			temFonte := false
			if uc.Suppliers != nil {
				_, achou, err := uc.Suppliers.GetPreferredSupplier(ctx, *op.ServiceItemCode)
				if err != nil {
					return fmt.Errorf("lendo o fornecedor do serviço da operação %s: %w", op.OperationName, err)
				}
				temFonte = achou
			}
			if !temFonte {
				report.Issues = append(report.Issues, fmt.Sprintf("operação externa %s sem fornecedor: informe o terceiro na operação ou cadastre a fonte de suprimento do item de serviço dela", op.OperationName))
			}
		}
	}
	return nil
}

// externasDoRoteiro: as operações externas do roteiro do item.
type externasDoRoteiro interface {
	GetExternalOpsByItem(ctx context.Context, itemCode int64) ([]*routingentity.ExternalOp, error)
}

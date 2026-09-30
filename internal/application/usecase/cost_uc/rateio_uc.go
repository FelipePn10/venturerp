// Cadastro do esquema de rateio de indiretos e leitura do histórico de custo.
//
// Fica no mesmo caso de uso do custo-padrão de propósito: as regras só existem
// para alimentar a apuração, e separá-las em outro pacote faria a validação da
// taxa viver longe de quem a aplica.
package cost_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	"github.com/google/uuid"
)

// rateioRepo é a fatia do repositório que governa indiretos e histórico. Acessada
// por asserção de tipo porque a interface principal do custo-padrão é usada por
// outros módulos que não precisam disso.
type rateioRepo interface {
	ListarRegrasDeRateio(context.Context) ([]*entity.RegraDeRateio, error)
	CriarRegraDeRateio(context.Context, *entity.RegraDeRateio) (*entity.RegraDeRateio, error)
	AtualizarRegraDeRateio(context.Context, *entity.RegraDeRateio) (*entity.RegraDeRateio, error)
	DesativarRegraDeRateio(context.Context, int64) error
	HistoricoDeCusto(context.Context, int64, string, int) ([]*entity.HistoricoDeCusto, error)
}

func (uc *StandardCostUseCase) rateio() (rateioRepo, error) {
	r, ok := uc.repo.(rateioRepo)
	if !ok {
		return nil, fmt.Errorf("esquema de rateio não configurado nesta instalação")
	}
	return r, nil
}

func (uc *StandardCostUseCase) ListarRegrasDeRateio(ctx context.Context) ([]response.CostOverheadRuleResponse, error) {
	repo, err := uc.rateio()
	if err != nil {
		return nil, err
	}
	regras, err := repo.ListarRegrasDeRateio(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]response.CostOverheadRuleResponse, 0, len(regras))
	for _, r := range regras {
		out = append(out, paraRegraResponse(r))
	}
	return out, nil
}

func (uc *StandardCostUseCase) CriarRegraDeRateio(ctx context.Context, dto request.CostOverheadRuleDTO) (*response.CostOverheadRuleResponse, error) {
	repo, err := uc.rateio()
	if err != nil {
		return nil, err
	}
	regra, err := montarRegra(dto)
	if err != nil {
		return nil, err
	}
	criada, err := repo.CriarRegraDeRateio(ctx, regra)
	if err != nil {
		return nil, err
	}
	resp := paraRegraResponse(criada)
	return &resp, nil
}

func (uc *StandardCostUseCase) AtualizarRegraDeRateio(ctx context.Context, id int64, dto request.CostOverheadRuleDTO) (*response.CostOverheadRuleResponse, error) {
	repo, err := uc.rateio()
	if err != nil {
		return nil, err
	}
	regra, err := montarRegra(dto)
	if err != nil {
		return nil, err
	}
	regra.ID = id
	atualizada, err := repo.AtualizarRegraDeRateio(ctx, regra)
	if err != nil {
		return nil, err
	}
	resp := paraRegraResponse(atualizada)
	return &resp, nil
}

func (uc *StandardCostUseCase) DesativarRegraDeRateio(ctx context.Context, id int64) error {
	repo, err := uc.rateio()
	if err != nil {
		return err
	}
	return repo.DesativarRegraDeRateio(ctx, id)
}

func (uc *StandardCostUseCase) HistoricoDeCusto(ctx context.Context, itemCode int64, mask string, limite int) ([]response.CostHistoryResponse, error) {
	repo, err := uc.rateio()
	if err != nil {
		return nil, err
	}
	linhas, err := repo.HistoricoDeCusto(ctx, itemCode, mask, limite)
	if err != nil {
		return nil, err
	}
	out := make([]response.CostHistoryResponse, 0, len(linhas))
	for _, h := range linhas {
		out = append(out, response.CostHistoryResponse{
			ID: h.ID, ItemCode: h.ItemCode, Mask: h.Mask, LotSize: h.LotSize,
			MaterialCost: h.Componentes.Material, SetupCost: h.Componentes.Setup,
			MachineCost: h.Componentes.Maquina, LaborCost: h.Componentes.MaoDeObra,
			SubcontractCost: h.Componentes.Subcontratacao, OverheadCost: h.Componentes.Overhead,
			OwnLevelCost: h.Componentes.NivelProprio, LowerLevelCost: h.Componentes.NivelInferior,
			TotalCost: h.TotalCost, Currency: h.Currency,
			Overheads: paraRateiosResponse(h.Rateios), CalculatedAt: h.CalculatedAt,
		})
	}
	return out, nil
}

// basesValidas e metodosValidos fecham o domínio no caso de uso, antes do banco.
// O CHECK do banco é a última linha de defesa; a mensagem boa vem daqui.
var basesValidas = map[entity.BaseDeRateio]string{
	entity.BaseMaterial:       "material",
	entity.BaseSetup:          "preparação",
	entity.BaseMaquina:        "hora-máquina",
	entity.BaseMaoDeObra:      "hora-homem",
	entity.BaseConversao:      "conversão (preparação + máquina + mão de obra)",
	entity.BaseSubcontratacao: "serviço de terceiro",
	entity.BaseTotal:          "custo total antes dos indiretos",
}

var metodosValidos = map[entity.MetodoDeRateio]bool{
	entity.MetodoPercentual:      true,
	entity.MetodoValorPorHora:    true,
	entity.MetodoValorPorUnidade: true,
}

func montarRegra(dto request.CostOverheadRuleDTO) (*entity.RegraDeRateio, error) {
	codigo := strings.ToUpper(strings.TrimSpace(dto.Code))
	if codigo == "" {
		return nil, errorsuc.NewValidationError("informe o código da regra de rateio")
	}
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("informe a descrição da regra: é o que explica o indireto no custo do produto")
	}
	base := entity.BaseDeRateio(strings.ToUpper(strings.TrimSpace(dto.Base)))
	if _, ok := basesValidas[base]; !ok {
		return nil, errorsuc.NewValidationError("base do rateio inválida: use MATERIAL, SETUP, MAQUINA, MAO_DE_OBRA, CONVERSAO, SUBCONTRATACAO ou TOTAL")
	}
	metodo := entity.MetodoDeRateio(strings.ToUpper(strings.TrimSpace(dto.Method)))
	if !metodosValidos[metodo] {
		return nil, errorsuc.NewValidationError("método do rateio inválido: use PERCENTUAL, VALOR_POR_HORA ou VALOR_POR_UNIDADE")
	}
	if dto.Rate <= 0 {
		return nil, errorsuc.NewValidationError("a taxa do rateio tem de ser maior que zero")
	}
	// O erro mais comum: digitar 12 querendo 12%. Avisar aqui, com o número que a
	// pessoa escreveu, é muito mais útil que a violação de CHECK do banco.
	if metodo == entity.MetodoPercentual && dto.Rate > 1 {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"a taxa percentual entra como fração: para %g%% informe %g, não %g",
			dto.Rate, dto.Rate/100, dto.Rate))
	}
	if metodo == entity.MetodoValorPorHora &&
		base != entity.BaseMaquina && base != entity.BaseMaoDeObra &&
		base != entity.BaseConversao && base != entity.BaseSetup {
		return nil, errorsuc.NewValidationError(
			"valor por hora só pode incidir sobre bases medidas em horas: máquina, mão de obra, preparação ou conversão")
	}

	inicio, err := dataDaRegra(dto.ValidFrom, "data inicial de vigência")
	if err != nil {
		return nil, err
	}
	if inicio == nil {
		return nil, errorsuc.NewValidationError("informe a data inicial de vigência da regra")
	}
	fim, err := dataDaRegra(valorOuVazioStr(dto.ValidTo), "data final de vigência")
	if err != nil {
		return nil, err
	}
	if fim != nil && fim.Before(*inicio) {
		return nil, errorsuc.NewValidationError("a data final da vigência é anterior à data inicial")
	}

	ativa := true
	if dto.IsActive != nil {
		ativa = *dto.IsActive
	}
	autor, err := uuid.Parse(dto.CreatedBy)
	if err != nil {
		return nil, errorsuc.NewValidationError("não foi possível identificar o usuário da sessão")
	}

	return &entity.RegraDeRateio{
		Code: codigo, Description: strings.TrimSpace(dto.Description),
		Base: base, Method: metodo, Rate: dto.Rate,
		WorkCenterID: dto.WorkCenterID, ItemCode: dto.ItemCode,
		PlanoContasID: dto.PlanoContasID, CentroCustoID: dto.CentroCustoID,
		ValidFrom: *inicio, ValidTo: fim, IsActive: ativa,
		Notes: dto.Notes, CreatedBy: autor,
	}, nil
}

func dataDaRegra(bruto, campo string) (*time.Time, error) {
	bruto = strings.TrimSpace(bruto)
	if bruto == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", bruto)
	if err != nil {
		return nil, errorsuc.NewValidationError(campo + " inválida: use o formato AAAA-MM-DD")
	}
	return &t, nil
}

func valorOuVazioStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func paraRegraResponse(r *entity.RegraDeRateio) response.CostOverheadRuleResponse {
	out := response.CostOverheadRuleResponse{
		ID: r.ID, Code: r.Code, Description: r.Description,
		Base: string(r.Base), Method: string(r.Method), Rate: r.Rate,
		WorkCenterID: r.WorkCenterID, ItemCode: r.ItemCode,
		PlanoContasID: r.PlanoContasID, CentroCustoID: r.CentroCustoID,
		ValidFrom: r.ValidFrom.Format("2006-01-02"), IsActive: r.IsActive, Notes: r.Notes,
	}
	if r.ValidTo != nil {
		fim := r.ValidTo.Format("2006-01-02")
		out.ValidTo = &fim
	}
	return out
}

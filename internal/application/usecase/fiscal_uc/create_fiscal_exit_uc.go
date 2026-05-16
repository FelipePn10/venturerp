package fiscal_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type CreateFiscalExitUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *CreateFiscalExitUseCase) Execute(ctx context.Context, dto request.CreateFiscalExitDTO) (*entity.FiscalExit, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	config, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	ncmTaxes, err := uc.Repo.ListNcmTaxes(ctx)
	if err != nil {
		return nil, err
	}

	interstateTable, err := uc.Repo.ListICMSInterstate(ctx)
	if err != nil {
		return nil, err
	}

	internalTable, err := uc.Repo.ListICMSInternal(ctx)
	if err != nil {
		return nil, err
	}

	ncmMap := make(map[string]*engine.NcmTaxConfig)
	for _, n := range ncmTaxes {
		ncmMap[n.Ncm] = &engine.NcmTaxConfig{
			AliqIPI:    n.AliqIPI,
			AliqPis:    n.AliqPis,
			AliqCofins: n.AliqCofins,
			CstPis:     n.CstPis,
			CstCofins:  n.CstCofins,
			CstIPI:     n.CstIPI,
		}
	}

	internalEngineMap := make(map[string]engine.ICMSInternalConfig)
	for uf, v := range internalTable {
		internalEngineMap[uf] = engine.ICMSInternalConfig{
			ICMS: v.ICMS,
			FCP:  v.FCP,
		}
	}

	destUF := ""
	if dto.UFDestinatario != nil {
		destUF = *dto.UFDestinatario
	}
	origemUF := config.UFEmpresa

	taxItems := make([]engine.TaxItem, 0, len(dto.Itens))
	for _, it := range dto.Itens {
		ncm := ""
		if it.Ncm != nil {
			ncm = *it.Ncm
		}
		itemFrete := dto.ValorFrete
		itemDesconto := dto.ValorDesconto
		taxItems = append(taxItems, engine.TaxItem{
			Ncm:              ncm,
			ValorUnitario:    it.UnitPrice,
			Quantidade:       it.Quantity,
			ValorFrete:       itemFrete,
			ValorDesconto:    itemDesconto,
			OrigemMercadoria: it.OrigemMercadoria,
		})
	}

	destTipo := "contribuinte"
	if dto.IEDestinatario == nil || *dto.IEDestinatario == "" || *dto.IEDestinatario == "ISENTO" {
		destTipo = "nao_contribuinte"
	} else if dto.TipoPessoa != nil && *dto.TipoPessoa == "F" {
		destTipo = "pessoa_fisica"
	}

	params := engine.TaxCalculationParams{
		Itens:       taxItems,
		EmitenteUF:  origemUF,
		DestinoUF:   destUF,
		DestinoTipo: destTipo,
		Cfop:        dto.Cfop,
	}

	fiscalCfg := engine.FiscalConfig{
		UFEmpresa:               config.UFEmpresa,
		IcmsInternoAliquota:     config.IcmsInternoAliquota,
		IcmsDiferimentoPercentual: config.IcmsDiferimentoPercentual,
	}

	scenario := engine.TaxScenarioConfig{
		AliqICMS:   config.IcmsInternoAliquota,
		DifICMSPct: config.IcmsDiferimentoPercentual,
		CstICMS:    "00",
		CalcDifal:  true,
	}

	taxResult, err := engine.CalcularImpostos(params, ncmMap, interstateTable, internalEngineMap, scenario, fiscalCfg)
	if err != nil {
		return nil, err
	}

	dataEmissao, _ := time.Parse("2006-01-02", dto.DataEmissao)
	var dataSaida *time.Time
	if dto.DataSaida != nil {
		t, _ := time.Parse("2006-01-02", *dto.DataSaida)
		dataSaida = &t
	}

	exit := &entity.FiscalExit{
		NumeroNF:               dto.NumeroNF,
		Serie:                  dto.Serie,
		DataEmissao:            dataEmissao,
		DataSaida:              dataSaida,
		CnpjDestinatario:       dto.CnpjDestinatario,
		RazaoSocialDestinatario: dto.RazaoSocialDestinatario,
		IEDestinatario:         dto.IEDestinatario,
		UFDestinatario:         dto.UFDestinatario,
		Cfop:                   dto.Cfop,
		NaturezaOperacao:       dto.NaturezaOperacao,
		ValorProdutos:          dto.ValorProdutos,
		ValorFrete:             dto.ValorFrete,
		ValorSeguro:            dto.ValorSeguro,
		ValorDesconto:          dto.ValorDesconto,
		ValorIPI:               taxResult.Totais.ValorIPI,
		ValorICMS:              taxResult.Totais.ValorICMS,
		ValorPIS:               taxResult.Totais.ValorPIS,
		ValorCOFINS:            taxResult.Totais.ValorCOFINS,
		ValorTotal:             dto.ValorProdutos + taxResult.Totais.ValorIPI + taxResult.Totais.ValorICMS + dto.ValorFrete + dto.ValorSeguro - dto.ValorDesconto,
		SalesOrderCode:         dto.SalesOrderCode,
		Status:                 entity.ExitStatusDraft,
		CreatedBy:              userID,
	}

	created, err := uc.Repo.CreateExit(ctx, exit)
	if err != nil {
		return nil, err
	}

	for i, it := range dto.Itens {
		var itemTax engine.TaxItemResult
		if i < len(taxResult.Itens) {
			itemTax = taxResult.Itens[i]
		}

		item := &entity.FiscalExitItem{
			FiscalExitID:      created.ID,
			Sequence:          it.Sequence,
			ItemCode:          it.ItemCode,
			Ncm:               it.Ncm,
			Cfop:              it.Cfop,
			Quantity:          it.Quantity,
			UnitPrice:         it.UnitPrice,
			TotalPrice:        it.TotalPrice,
			BaseICMS:          itemTax.BaseICMS,
			AliqICMS:          itemTax.AliquotaICMS,
			ValorICMS:         itemTax.ValorICMS,
			ValorICMSDiferido: itemTax.ValorICMSDiferido,
			BaseIPI:           itemTax.BaseIPI,
			AliqIPI:           itemTax.AliquotaIPI,
			ValorIPI:          itemTax.ValorIPI,
			ValorPIS:          itemTax.ValorPIS,
			ValorCOFINS:       itemTax.ValorCOFINS,
			CstICMS:           &itemTax.CSTICMS,
			CstIPI:            &itemTax.CSTIPI,
			CstPIS:            &itemTax.CSTPIS,
			CstCOFINS:         &itemTax.CSTCOFINS,
			OrigemMercadoria:  it.OrigemMercadoria,
			Description:       it.Description,
		}
		if _, err := uc.Repo.CreateExitItem(ctx, item); err != nil {
			return nil, err
		}
	}

	items, _ := uc.Repo.GetExitItems(ctx, created.ID)
	created.Itens = items

	return created, nil
}

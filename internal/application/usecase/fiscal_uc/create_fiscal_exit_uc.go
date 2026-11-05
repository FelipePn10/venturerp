package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"

	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type CreateFiscalExitUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
	// Customers e SalesOrders sao opcionais e servem para completar o
	// destinatario: a NF-e exige o endereco completo e ninguem o digita a cada
	// nota. Sem eles a nota continua sendo criada com o que foi informado.
	Customers   customerrepo.CustomerRepository
	SalesOrders salesrepo.SalesOrderRepository
	// Beneficios resolve o cBenef da linha pelos parâmetros de ICMS/IPI.
	// Opcional: sem ele, só vale o código informado na linha.
	Beneficios ResolvedorBeneficioFiscal
	// Parametros acha a regra do cadastro de parâmetros de ICMS/IPI (VFIS0320)
	// de cada linha: alíquota, redução, diferimento, CST, CSOSN e ST por
	// item/NCM, UF, cliente, finalidade e tipo de nota. Opcional: sem ele vale a
	// regra geral.
	Parametros ResolvedorParametroICMSIPI
	// Cadastro lê a finalidade da compra (pedido, cliente) e os indicadores do
	// tipo de nota. Opcional: sem ele vale só o informado na nota.
	Cadastro CadastroTributario
}

// ResolvedorBeneficioFiscal acha o código de benefício fiscal (cBenef) de um
// item numa operação de saída para a UF de destino.
type ResolvedorBeneficioFiscal interface {
	BeneficioFiscal(ctx context.Context, itemCode *int64, ncm *string, ufDestino string, cliente *int64) (*string, error)
}

// conferirLinhasDaNota recusa a linha sem quantidade: o imposto é calculado sobre
// quantidade × preço, então ela criava a nota com total preenchido e ICMS zerado,
// e o erro só aparecia na rejeição da SEFAZ.
func conferirLinhasDaNota(itens []request.CreateFiscalExitItemDTO) error {
	if len(itens) == 0 {
		return errorsuc.NewValidationError("inclua ao menos um item na nota")
	}
	for i := range itens {
		if itens[i].Quantity.IsZero() {
			itens[i].Quantity = itens[i].Quantidade
		}
		if !itens[i].Quantity.IsPositive() {
			return errorsuc.NewValidationError(fmt.Sprintf("informe a quantidade do item da linha %d", i+1))
		}
		if itens[i].UnitPrice.IsNegative() {
			return errorsuc.NewValidationError(fmt.Sprintf("o preço do item da linha %d não pode ser negativo", i+1))
		}
	}
	return nil
}

// cstComBeneficio: CSTs de ICMS que levam cBenef nas UFs que o exigem. Mandar o
// código num CST sem benefício (00, 10…) também é rejeitado pela SEFAZ.
var cstComBeneficio = map[string]bool{"20": true, "30": true, "40": true, "41": true, "50": true, "51": true, "53": true, "70": true, "90": true}

func (uc *CreateFiscalExitUseCase) Execute(ctx context.Context, dto request.CreateFiscalExitDTO) (*response.FiscalExitResponse, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if err := conferirLinhasDaNota(dto.Itens); err != nil {
		return nil, err
	}

	if err := resolverDestinatario(ctx, &dto, uc.Customers, uc.SalesOrders); err != nil {
		return nil, err
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	destUF := ""
	if dto.UFDestinatario != nil {
		destUF = *dto.UFDestinatario
	}
	op := OperacaoTributavel{
		UFDestino: destUF, IE: dto.IEDestinatario, TipoPessoa: dto.TipoPessoa,
		Cliente: dto.CustomerCode, Pedido: dto.SalesOrderCode, TipoNotaID: dto.InvoiceTypeID, Cfop: dto.Cfop,
		ValorFrete: dto.ValorFrete, ValorDesconto: dto.ValorDesconto,
	}
	if dto.FinalidadeCompra != nil {
		op.Finalidade = *dto.FinalidadeCompra
	}
	for _, it := range dto.Itens {
		op.Itens = append(op.Itens, LinhaTributavel{
			ItemCode: it.ItemCode, Ncm: it.Ncm, Quantidade: it.Quantity, ValorUnitario: it.UnitPrice,
			Origem: it.OrigemMercadoria, MvaPct: it.MvaPct, AliqInternaDestinoST: it.AliqInternaDestinoST, RedBaseSTPct: it.RedBaseSTPct,
		})
	}
	trib, err := (&Tributador{Repo: uc.Repo, Parametros: uc.Parametros, Cadastro: uc.Cadastro}).Calcular(ctx, op)
	if err != nil {
		return nil, err
	}
	destTipo, regras, taxResult := trib.DestinoTipo, trib.Regras, trib.Resultado

	if dto.InvoiceTypeID != nil && dto.SalesOrderCode == nil {
		if leitor, ok := uc.Cadastro.(interface {
			ExigePedidoNoTipoNota(context.Context, int64) (bool, error)
		}); ok {
			exige, err := leitor.ExigePedidoNoTipoNota(ctx, *dto.InvoiceTypeID)
			if err != nil {
				return nil, err
			}
			if exige {
				return nil, errorsuc.NewValidationError("este tipo de nota só pode ser usado com pedido de venda: fature pelo pedido ou escolha outro tipo")
			}
		}
	}

	// Número de controle do rascunho. O número fiscal é da Focus: ela controla a
	// sequência pelo painel, não recebe número nem série no envio e devolve os dois
	// na autorização (UpdateExitNumbering). Este aqui não sai no XML nem na tela.
	serie := strings.TrimSpace(dto.Serie)
	if serie == "" {
		serie = "1"
	}
	numeroNF := dto.NumeroNF
	if numeroNF <= 0 {
		proximo, err := uc.Repo.GetNextNFNumber(ctx)
		if err != nil {
			return nil, err
		}
		numeroNF = proximo
	} else {
		existe, err := uc.Repo.NFNumberExists(ctx, numeroNF, serie)
		if err != nil {
			return nil, err
		}
		if existe {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("já existe uma NF-e com o número %d na série %s: deixe o número em branco para usar o próximo", numeroNF, serie))
		}
	}

	dataEmissao, msg := datetime.Obrigatoria("data de emissão", dto.DataEmissao)
	if msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	var dataSaida *time.Time
	if dto.DataSaida != nil {
		t, msg := datetime.Obrigatoria("data de saída", *dto.DataSaida)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		dataSaida = &t
	}
	var fiscalCouponDate *time.Time
	if dto.FiscalCouponDate != nil {
		t, msg := datetime.Obrigatoria("data do cupom fiscal", *dto.FiscalCouponDate)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		fiscalCouponDate = &t
	}

	exit := &entity.FiscalExit{
		NumeroNF:                numeroNF,
		Serie:                   serie,
		DataEmissao:             dataEmissao,
		DataSaida:               dataSaida,
		CnpjDestinatario:        dto.CnpjDestinatario,
		RazaoSocialDestinatario: dto.RazaoSocialDestinatario,
		IEDestinatario:          dto.IEDestinatario,
		UFDestinatario:          dto.UFDestinatario,
		DestLogradouro:          dto.DestLogradouro,
		DestNumero:              dto.DestNumero,
		DestComplemento:         dto.DestComplemento,
		DestBairro:              dto.DestBairro,
		DestMunicipio:           dto.DestMunicipio,
		DestCodigoMunicipio:     dto.DestCodigoMunicipio,
		DestCEP:                 dto.DestCEP,
		DestEmail:               dto.DestEmail,
		DestTelefone:            dto.DestTelefone,
		CustomerCode:            dto.CustomerCode,
		Cfop:                    dto.Cfop,
		NaturezaOperacao:        dto.NaturezaOperacao,
		ValorProdutos:           dto.ValorProdutos,
		ValorFrete:              dto.ValorFrete,
		ValorSeguro:             dto.ValorSeguro,
		ValorDesconto:           dto.ValorDesconto,
		ValorIPI:                taxResult.Totais.ValorIPI,
		// ICMS destacado (vICMS do XML): o da operação menos o diferido. É o que
		// vai para o livro, a apuração e o "ICMS a recolher" da contabilização.
		ValorICMS:   taxResult.Totais.ValorICMS.Sub(taxResult.Totais.ValorDiferido).Round(2),
		ValorPIS:    taxResult.Totais.ValorPIS,
		ValorCOFINS: taxResult.Totais.ValorCOFINS,
		BaseICMSST:  taxResult.Totais.BaseICMSST,
		ValorICMSST: taxResult.Totais.ValorICMSST,
		// ValorTotal = produtos + IPI + ICMS-ST + frete + seguro - desconto
		// (ICMS próprio não soma no total da NF-e; o ICMS-ST soma, pois é cobrado do destinatário)
		ValorTotal:            dto.ValorProdutos.Add(taxResult.Totais.ValorIPI).Add(taxResult.Totais.ValorICMSST).Add(dto.ValorFrete).Add(dto.ValorSeguro).Sub(dto.ValorDesconto),
		SalesOrderCode:        dto.SalesOrderCode,
		SourceType:            dto.SourceType,
		ShipmentLoadCode:      dto.ShipmentLoadCode,
		ShipmentCode:          dto.ShipmentCode,
		FiscalCouponNumber:    dto.FiscalCouponNumber,
		FiscalCouponDate:      fiscalCouponDate,
		FiscalCouponECFSerial: dto.FiscalCouponECFSerial,
		InvoiceTypeID:         dto.InvoiceTypeID,
		Status:                entity.ExitStatusDraft,
		CreatedBy:             userID,
	}

	if trib.Finalidade != "" {
		f := trib.Finalidade
		exit.FinalidadeCompra = &f
	}
	var legais []string
	credito := decimal.Zero
	for i, r := range taxResult.Itens {
		legais = append(legais, regras[i].DispositivosAplicados(destTipo, r)...)
		credito = credito.Add(r.ValorCreditoSimples)
	}
	exit.InformacoesComplementares = informacoesComplementares(dto.InformacoesComplementares, trib.Regime == engine.RegimeSimples,
		credito, trib.Config.AliquotaCreditoSimples, semRepetir(legais))

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
			FiscalExitID:       created.ID,
			Mask:               it.Mask,
			Lot:                it.Lot,
			Address:            it.Address,
			Sequence:           it.Sequence,
			ItemCode:           it.ItemCode,
			Ncm:                it.Ncm,
			Cfop:               it.Cfop,
			Quantity:           it.Quantity,
			UnitPrice:          it.UnitPrice,
			TotalPrice:         it.TotalPrice,
			BaseICMS:           itemTax.BaseICMS,
			AliqICMS:           itemTax.AliquotaICMS,
			ValorICMS:          itemTax.ValorICMS,
			ValorICMSDiferido:  itemTax.ValorICMSDiferido,
			BaseIPI:            itemTax.BaseIPI,
			AliqIPI:            itemTax.AliquotaIPI,
			ValorIPI:           itemTax.ValorIPI,
			AliqPIS:            itemTax.AliquotaPIS,
			ValorPIS:           itemTax.ValorPIS,
			AliqCOFINS:         itemTax.AliquotaCOFINS,
			ValorCOFINS:        itemTax.ValorCOFINS,
			BaseICMSST:         itemTax.BaseICMSST,
			AliqICMSST:         itemTax.AliquotaICMSST,
			ValorICMSST:        itemTax.ValorICMSST,
			MVA:                itemTax.MVA,
			CstICMS:            &itemTax.CSTICMS,
			CstIPI:             &itemTax.CSTIPI,
			CstPIS:             &itemTax.CSTPIS,
			CstCOFINS:          &itemTax.CSTCOFINS,
			OrigemMercadoria:   it.OrigemMercadoria,
			Description:        it.Description,
			SalesOrderItemCode: it.SalesOrderItemCode,
			UnidadeComercial:   it.UnidadeComercial,
			CodigoProduto:      it.CodigoProduto,
			// Redução de base, diferimento, crédito do Simples e grupo do DIFAL:
			// o que o XML leva da linha.
			ICMSReducaoBasePct:    itemTax.ReducaoBaseICMSPct,
			ICMSDiferimentoPct:    itemTax.DiferimentoICMSPct,
			AliqCreditoSimples:    itemTax.AliqCreditoSimplesPct,
			ValorCreditoSimples:   itemTax.ValorCreditoSimples,
			ICMSUFDestBase:        itemTax.BaseUFDest,
			ICMSUFDestAliqInterna: itemTax.AliqInternaUFDestPct,
			ICMSAliqInterestadual: itemTax.AliqInterestadualPct,
			ICMSUFDestValor:       itemTax.ValorDIFAL,
			FCPUFDestPct:          itemTax.FCPUFDestPct,
			FCPUFDestValor:        itemTax.ValorFCP,
			ICMSSTReducaoBasePct:  itemTax.ReducaoBaseSTPct,
		}
		if regras[i] != nil {
			item.RegraICMSIPIID = &regras[i].ID
		}
		if cstComBeneficio[itemTax.CSTICMS] {
			item.CodigoBeneficioFiscal = it.CodigoBeneficioFiscal
			if cb := regras[i].BeneficioFiscal(destTipo); (item.CodigoBeneficioFiscal == nil || strings.TrimSpace(*item.CodigoBeneficioFiscal) == "") && cb != "" {
				item.CodigoBeneficioFiscal = &cb
			}
			if (item.CodigoBeneficioFiscal == nil || strings.TrimSpace(*item.CodigoBeneficioFiscal) == "") && uc.Beneficios != nil {
				cb, err := uc.Beneficios.BeneficioFiscal(ctx, it.ItemCode, it.Ncm, destUF, dto.CustomerCode)
				if err != nil {
					return nil, fmt.Errorf("lendo o código de benefício fiscal da linha %d: %w", it.Sequence, err)
				}
				item.CodigoBeneficioFiscal = cb
			}
			// Diferimento da configuração: o cBenef dela (PR830001 no PR).
			if (item.CodigoBeneficioFiscal == nil || strings.TrimSpace(*item.CodigoBeneficioFiscal) == "") && itemTax.CSTICMS == "51" &&
				trib.Config.CBenefDiferimento != nil && strings.TrimSpace(*trib.Config.CBenefDiferimento) != "" {
				cb := strings.TrimSpace(*trib.Config.CBenefDiferimento)
				item.CodigoBeneficioFiscal = &cb
			}
		}
		if _, err := uc.Repo.CreateExitItem(ctx, item); err != nil {
			return nil, err
		}
	}

	items, errItens := uc.Repo.GetExitItems(ctx, created.ID)
	if errItens != nil {
		return nil, fmt.Errorf("relendo os itens da nota criada: %w", errItens)
	}
	created.Itens = items

	return toFiscalExitResponse(created), nil
}

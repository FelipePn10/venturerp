package fiscal_uc

import (
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/shopspring/decimal"
)

func toFiscalExitResponse(e *entity.FiscalExit) *response.FiscalExitResponse {
	if e == nil {
		return nil
	}
	finalidade := e.Finalidade
	if finalidade == 0 {
		finalidade = 1
	}
	return &response.FiscalExitResponse{
		Finalidade:                finalidade,
		NFeReferenciada:           e.NFeReferenciada,
		FiscalEntryID:             e.FiscalEntryID,
		SupplierCode:              e.SupplierCode,
		ID:                        e.ID,
		ChaveAcesso:               e.ChaveAcesso,
		NumeroNF:                  numeroDefinitivo(e),
		Serie:                     e.Serie,
		DataEmissao:               e.DataEmissao,
		DataSaida:                 e.DataSaida,
		CnpjDestinatario:          e.CnpjDestinatario,
		RazaoSocialDestinatario:   e.RazaoSocialDestinatario,
		IEDestinatario:            e.IEDestinatario,
		UFDestinatario:            e.UFDestinatario,
		DestLogradouro:            e.DestLogradouro,
		DestNumero:                e.DestNumero,
		DestComplemento:           e.DestComplemento,
		DestBairro:                e.DestBairro,
		DestMunicipio:             e.DestMunicipio,
		DestCodigoMunicipio:       e.DestCodigoMunicipio,
		DestCEP:                   e.DestCEP,
		DestEmail:                 e.DestEmail,
		DestTelefone:              e.DestTelefone,
		CustomerCode:              e.CustomerCode,
		Cfop:                      e.Cfop,
		NaturezaOperacao:          e.NaturezaOperacao,
		ValorProdutos:             e.ValorProdutos.InexactFloat64(),
		ValorFrete:                e.ValorFrete.InexactFloat64(),
		ValorSeguro:               e.ValorSeguro.InexactFloat64(),
		ValorDesconto:             e.ValorDesconto.InexactFloat64(),
		ValorIPI:                  e.ValorIPI.InexactFloat64(),
		ValorICMS:                 e.ValorICMS.InexactFloat64(),
		ValorPIS:                  e.ValorPIS.InexactFloat64(),
		ValorCOFINS:               e.ValorCOFINS.InexactFloat64(),
		BaseICMSST:                e.BaseICMSST.InexactFloat64(),
		ValorICMSST:               e.ValorICMSST.InexactFloat64(),
		ValorTotal:                e.ValorTotal.InexactFloat64(),
		SalesOrderCode:            e.SalesOrderCode,
		SourceType:                e.SourceType,
		FinalidadeCompra:          e.FinalidadeCompra,
		InvoiceTypeID:             e.InvoiceTypeID,
		InformacoesComplementares: e.InformacoesComplementares,
		ShipmentLoadCode:          e.ShipmentLoadCode,
		ShipmentCode:              e.ShipmentCode,
		FiscalCouponNumber:        e.FiscalCouponNumber,
		FiscalCouponDate:          e.FiscalCouponDate,
		FiscalCouponECFSerial:     e.FiscalCouponECFSerial,
		Status:                    string(e.Status),
		Protocolo:                 e.Protocolo,
		XmlPath:                   e.XmlPath,
		DanfePath:                 e.DanfePath,
		FocusRef:                  e.FocusRef,
		IsActive:                  e.IsActive,
		CreatedAt:                 e.CreatedAt,
		UpdatedAt:                 e.UpdatedAt,
		CreatedBy:                 e.CreatedBy,
		Itens:                     toFiscalExitItemValues(e.Itens),
	}
}

func toFiscalExitResponses(list []*entity.FiscalExit) []*response.FiscalExitResponse {
	out := make([]*response.FiscalExitResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toFiscalExitResponse(e))
	}
	return out
}

func toFiscalExitItemValues(items []*entity.FiscalExitItem) []response.FiscalExitItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.FiscalExitItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, response.FiscalExitItemResponse{
			ID:                    it.ID,
			FiscalExitID:          it.FiscalExitID,
			Sequence:              it.Sequence,
			ItemCode:              it.ItemCode,
			Ncm:                   it.Ncm,
			Cfop:                  it.Cfop,
			Quantity:              it.Quantity.InexactFloat64(),
			UnitPrice:             it.UnitPrice.InexactFloat64(),
			TotalPrice:            it.TotalPrice.InexactFloat64(),
			BaseICMS:              it.BaseICMS.InexactFloat64(),
			AliqICMS:              it.AliqICMS.InexactFloat64(),
			ValorICMS:             it.ValorICMS.InexactFloat64(),
			ValorICMSDiferido:     it.ValorICMSDiferido.InexactFloat64(),
			BaseIPI:               it.BaseIPI.InexactFloat64(),
			AliqIPI:               it.AliqIPI.InexactFloat64(),
			ValorIPI:              it.ValorIPI.InexactFloat64(),
			AliqPIS:               it.AliqPIS.InexactFloat64(),
			ValorPIS:              it.ValorPIS.InexactFloat64(),
			AliqCOFINS:            it.AliqCOFINS.InexactFloat64(),
			ValorCOFINS:           it.ValorCOFINS.InexactFloat64(),
			BaseICMSST:            it.BaseICMSST.InexactFloat64(),
			AliqICMSST:            it.AliqICMSST.InexactFloat64(),
			ValorICMSST:           it.ValorICMSST.InexactFloat64(),
			MVA:                   it.MVA.InexactFloat64(),
			CstICMS:               it.CstICMS,
			CstIPI:                it.CstIPI,
			CstPIS:                it.CstPIS,
			CstCOFINS:             it.CstCOFINS,
			OrigemMercadoria:      it.OrigemMercadoria,
			Description:           it.Description,
			ClassTribIBSCBS:       it.ClassTribIBSCBS,
			IBSCBSEnviado:         ibscbsEnviadoResponse(it.ReformaEnviada),
			RegraICMSIPIID:        it.RegraICMSIPIID,
			ICMSReducaoBasePct:    it.ICMSReducaoBasePct.InexactFloat64(),
			ICMSDiferimentoPct:    it.ICMSDiferimentoPct.InexactFloat64(),
			AliqCreditoSimples:    it.AliqCreditoSimples.InexactFloat64(),
			ValorCreditoSimples:   it.ValorCreditoSimples.InexactFloat64(),
			ICMSUFDestBase:        it.ICMSUFDestBase.InexactFloat64(),
			ICMSUFDestAliqInterna: it.ICMSUFDestAliqInterna.InexactFloat64(),
			ICMSAliqInterestadual: it.ICMSAliqInterestadual.InexactFloat64(),
			ICMSUFDestValor:       it.ICMSUFDestValor.InexactFloat64(),
			FCPUFDestPct:          it.FCPUFDestPct.InexactFloat64(),
			FCPUFDestValor:        it.FCPUFDestValor.InexactFloat64(),
			CreatedAt:             it.CreatedAt,
		})
	}
	return out
}

func ibscbsEnviadoResponse(e *entity.ReformaEnviada) *response.FiscalExitIBSCBSEnviado {
	if e == nil {
		return nil
	}
	return &response.FiscalExitIBSCBSEnviado{CST: e.CST, ClassTrib: e.ClassTrib, Origem: e.Origem, Base: fptrD(e.Base),
		IBSUF: fptrD(e.IBSUF), IBSMun: fptrD(e.IBSMun), CBS: fptrD(e.CBS), AliqEfetivaCBS: fptrD(e.AliqEfetivaCBS)}
}

// fptrD leva o valor gravado (decimal, opcional) à resposta, que continua numérica.
func fptrD(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	f := d.InexactFloat64()
	return &f
}

func toFiscalEntryResponse(e *entity.FiscalEntry) *response.FiscalEntryResponse {
	if e == nil {
		return nil
	}
	return &response.FiscalEntryResponse{
		ID:                  e.ID,
		ChaveAcesso:         e.ChaveAcesso,
		NumeroNF:            e.NumeroNF,
		Serie:               e.Serie,
		Modelo:              e.Modelo,
		DataEmissao:         e.DataEmissao,
		DataEntrada:         e.DataEntrada,
		CnpjEmitente:        e.CnpjEmitente,
		RazaoSocialEmitente: e.RazaoSocialEmitente,
		IEEmitente:          e.IEEmitente,
		UFEmitente:          e.UFEmitente,
		ValorProdutos:       e.ValorProdutos.InexactFloat64(),
		ValorFrete:          e.ValorFrete.InexactFloat64(),
		ValorSeguro:         e.ValorSeguro.InexactFloat64(),
		ValorDesconto:       e.ValorDesconto.InexactFloat64(),
		ValorIPI:            e.ValorIPI.InexactFloat64(),
		ValorICMS:           e.ValorICMS.InexactFloat64(),
		ValorPIS:            e.ValorPIS.InexactFloat64(),
		ValorCOFINS:         e.ValorCOFINS.InexactFloat64(),
		ValorTotal:          e.ValorTotal.InexactFloat64(),
		TipoDocumento:       e.TipoDocumento,
		PurchaseOrderCode:   e.PurchaseOrderCode,
		SupplierCode:        e.SupplierCode,
		CteCode:             e.CteCode,
		Status:              string(e.Status),
		XmlPath:             e.XmlPath,
		Notes:               e.Notes,
		IsActive:            e.IsActive,
		CreatedAt:           e.CreatedAt,
		UpdatedAt:           e.UpdatedAt,
		CreatedBy:           e.CreatedBy,
		Itens:               toFiscalEntryItemValues(e.Itens),
		Warnings:            e.Warnings,
	}
}

func toFiscalEntryResponses(list []*entity.FiscalEntry) []*response.FiscalEntryResponse {
	out := make([]*response.FiscalEntryResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toFiscalEntryResponse(e))
	}
	return out
}

func toFiscalEntryItemValues(items []*entity.FiscalEntryItem) []response.FiscalEntryItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.FiscalEntryItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, response.FiscalEntryItemResponse{
			ID:                 it.ID,
			FiscalEntryID:      it.FiscalEntryID,
			Sequence:           it.Sequence,
			ItemCode:           it.ItemCode,
			ItemSupplierID:     it.ItemSupplierID,
			SupplierItemCode:   it.SupplierItemCode,
			ResolutionStrategy: it.ResolutionStrategy,
			ResolvedAt:         it.ResolvedAt,
			Ncm:                it.Ncm,
			Cfop:               it.Cfop,
			Quantity:           it.Quantity.InexactFloat64(),
			UnitPrice:          it.UnitPrice.InexactFloat64(),
			TotalPrice:         it.TotalPrice.InexactFloat64(),
			BaseICMS:           it.BaseICMS.InexactFloat64(),
			AliqICMS:           it.AliqICMS.InexactFloat64(),
			ValorICMS:          it.ValorICMS.InexactFloat64(),
			BaseIPI:            it.BaseIPI.InexactFloat64(),
			AliqIPI:            it.AliqIPI.InexactFloat64(),
			ValorIPI:           it.ValorIPI.InexactFloat64(),
			ValorPIS:           it.ValorPIS.InexactFloat64(),
			ValorCOFINS:        it.ValorCOFINS.InexactFloat64(),
			CstICMS:            it.CstICMS,
			CstIPI:             it.CstIPI,
			CstPIS:             it.CstPIS,
			CstCOFINS:          it.CstCOFINS,
			GeraCreditoICMS:    it.GeraCreditoICMS,
			GeraCreditoIPI:     it.GeraCreditoIPI,
			GeraCreditoPIS:     it.GeraCreditoPIS,
			GeraCreditoCOFINS:  it.GeraCreditoCOFINS,
			Description:        it.Description,
			Notes:              it.Notes,
			CreatedAt:          it.CreatedAt,
		})
	}
	return out
}

func toFiscalConfigResponse(c *entity.FiscalConfig) *response.FiscalConfigResponse {
	if c == nil {
		return nil
	}
	return &response.FiscalConfigResponse{
		ID:                         c.ID,
		CnpjEmpresa:                c.CnpjEmpresa,
		RazaoSocial:                c.RazaoSocial,
		IEEmpresa:                  c.IEEmpresa,
		RegimeTributario:           c.RegimeTributario,
		UFEmpresa:                  c.UFEmpresa,
		IcmsInternoAliquota:        c.IcmsInternoAliquota.InexactFloat64(),
		IcmsDiferimentoPercentual:  c.IcmsDiferimentoPercentual.InexactFloat64(),
		ExigirRegraICMSIPI:         c.ExigirRegraICMSIPI,
		AliquotaCreditoSimples:     c.AliquotaCreditoSimples.InexactFloat64(),
		CBenefDiferimento:          c.CBenefDiferimento,
		ObsRetornoIndustrializacao: c.ObsRetornoIndustrializacao,
		CBenefSuspensao:            c.CBenefSuspensao,
		FocusNfeConfigured:         c.FocusNfeToken != nil && strings.TrimSpace(*c.FocusNfeToken) != "",
		FocusNfeAmbiente:           c.FocusNfeAmbiente,
		FocusProducaoTravado:       c.EmProducao(),
		NomeFantasia:               c.TradeName,
		JurosMes:                   c.JurosMes.InexactFloat64(),
		MultaAtraso:                c.MultaAtraso.InexactFloat64(),
		VencimentoIcmsDia:          c.VencimentoIcmsDia,
		VencimentoIPIDia:           c.VencimentoIPIDia,
		VencimentoPisCofinsDia:     c.VencimentoPisCofinsDia,
		Logradouro:                 c.Logradouro,
		Numero:                     c.Numero,
		Complemento:                c.Complemento,
		Bairro:                     c.Bairro,
		Municipio:                  c.Municipio,
		CodigoMunicipio:            c.CodigoMunicipio,
		CEP:                        c.CEP,
		Telefone:                   c.Telefone,
		BrandColor:                 c.BrandColor,
		CreatedAt:                  c.CreatedAt,
		UpdatedAt:                  c.UpdatedAt,
		UpdatedBy:                  c.UpdatedBy,
	}
}

func toFiscalCTeResponse(c *entity.FiscalCTe) *response.FiscalCTeResponse {
	if c == nil {
		return nil
	}
	return &response.FiscalCTeResponse{
		ID:                  c.ID,
		ChaveAcesso:         c.ChaveAcesso,
		NumeroCTe:           c.NumeroCTe,
		Serie:               c.Serie,
		DataEmissao:         c.DataEmissao,
		DataEntrada:         c.DataEntrada,
		CnpjEmitente:        c.CnpjEmitente,
		RazaoSocialEmitente: c.RazaoSocialEmitente,
		IEEmitente:          c.IEEmitente,
		UFEmitente:          c.UFEmitente,
		Cfop:                c.Cfop,
		ValorFrete:          c.ValorFrete.InexactFloat64(),
		ValorSeguro:         c.ValorSeguro.InexactFloat64(),
		ValorOutros:         c.ValorOutros.InexactFloat64(),
		ValorTotal:          c.ValorTotal.InexactFloat64(),
		ValorICMS:           c.ValorICMS.InexactFloat64(),
		BaseICMS:            c.BaseICMS.InexactFloat64(),
		AliqICMS:            c.AliqICMS.InexactFloat64(),
		CstICMS:             c.CstICMS,
		TipoRateio:          c.TipoRateio,
		FiscalEntryID:       c.FiscalEntryID,
		Status:              c.Status,
		FocusRef:            c.FocusRef,
		Protocolo:           c.Protocolo,
		EmissionData:        c.EmissionData,
		XmlPath:             c.XmlPath,
		Notes:               c.Notes,
		IsActive:            c.IsActive,
		CreatedAt:           c.CreatedAt,
		UpdatedAt:           c.UpdatedAt,
	}
}

func toFiscalCTeResponses(list []*entity.FiscalCTe) []*response.FiscalCTeResponse {
	out := make([]*response.FiscalCTeResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toFiscalCTeResponse(c))
	}
	return out
}

func toCartaCorrecaoResponses(list []*entity.CartaCorrecao) []*response.CartaCorrecaoResponse {
	out := make([]*response.CartaCorrecaoResponse, 0, len(list))
	for _, c := range list {
		if c == nil {
			continue
		}
		out = append(out, &response.CartaCorrecaoResponse{
			ID:            c.ID,
			FiscalExitID:  c.FiscalExitID,
			NumeroSeq:     c.NumeroSeq,
			TextoCorrecao: c.TextoCorrecao,
			FocusRef:      c.FocusRef,
			Status:        c.Status,
			Protocolo:     c.Protocolo,
			ChaveEvento:   c.ChaveEvento,
			CreatedAt:     c.CreatedAt,
		})
	}
	return out
}

func toNcmTaxTableResponse(n *entity.NcmTaxTable) *response.NcmTaxTableResponse {
	if n == nil {
		return nil
	}
	return &response.NcmTaxTableResponse{
		ID:          n.ID,
		Ncm:         n.Ncm,
		AliqIPI:     n.AliqIPI.InexactFloat64(),
		AliqPis:     n.AliqPis.InexactFloat64(),
		AliqCofins:  n.AliqCofins.InexactFloat64(),
		CstPis:      n.CstPis,
		CstCofins:   n.CstCofins,
		CstIPI:      n.CstIPI,
		Description: n.Description,
		IsActive:    n.IsActive,
		CreatedAt:   n.CreatedAt,
	}
}

func toNcmTaxTableResponses(list []*entity.NcmTaxTable) []*response.NcmTaxTableResponse {
	out := make([]*response.NcmTaxTableResponse, 0, len(list))
	for _, n := range list {
		out = append(out, toNcmTaxTableResponse(n))
	}
	return out
}

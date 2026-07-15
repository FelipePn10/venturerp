package fiscal_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

func toFiscalExitResponse(e *entity.FiscalExit) *response.FiscalExitResponse {
	if e == nil {
		return nil
	}
	return &response.FiscalExitResponse{
		ID:                      e.ID,
		ChaveAcesso:             e.ChaveAcesso,
		NumeroNF:                e.NumeroNF,
		Serie:                   e.Serie,
		DataEmissao:             e.DataEmissao,
		DataSaida:               e.DataSaida,
		CnpjDestinatario:        e.CnpjDestinatario,
		RazaoSocialDestinatario: e.RazaoSocialDestinatario,
		IEDestinatario:          e.IEDestinatario,
		UFDestinatario:          e.UFDestinatario,
		Cfop:                    e.Cfop,
		NaturezaOperacao:        e.NaturezaOperacao,
		ValorProdutos:           e.ValorProdutos,
		ValorFrete:              e.ValorFrete,
		ValorSeguro:             e.ValorSeguro,
		ValorDesconto:           e.ValorDesconto,
		ValorIPI:                e.ValorIPI,
		ValorICMS:               e.ValorICMS,
		ValorPIS:                e.ValorPIS,
		ValorCOFINS:             e.ValorCOFINS,
		BaseICMSST:              e.BaseICMSST,
		ValorICMSST:             e.ValorICMSST,
		ValorTotal:              e.ValorTotal,
		SalesOrderCode:          e.SalesOrderCode,
		SourceType:              e.SourceType,
		ShipmentLoadCode:        e.ShipmentLoadCode,
		ShipmentCode:            e.ShipmentCode,
		FiscalCouponNumber:      e.FiscalCouponNumber,
		FiscalCouponDate:        e.FiscalCouponDate,
		FiscalCouponECFSerial:   e.FiscalCouponECFSerial,
		Status:                  string(e.Status),
		Protocolo:               e.Protocolo,
		XmlPath:                 e.XmlPath,
		DanfePath:               e.DanfePath,
		FocusRef:                e.FocusRef,
		IsActive:                e.IsActive,
		CreatedAt:               e.CreatedAt,
		UpdatedAt:               e.UpdatedAt,
		CreatedBy:               e.CreatedBy,
		Itens:                   toFiscalExitItemValues(e.Itens),
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
			ID:                it.ID,
			FiscalExitID:      it.FiscalExitID,
			Sequence:          it.Sequence,
			ItemCode:          it.ItemCode,
			Ncm:               it.Ncm,
			Cfop:              it.Cfop,
			Quantity:          it.Quantity,
			UnitPrice:         it.UnitPrice,
			TotalPrice:        it.TotalPrice,
			BaseICMS:          it.BaseICMS,
			AliqICMS:          it.AliqICMS,
			ValorICMS:         it.ValorICMS,
			ValorICMSDiferido: it.ValorICMSDiferido,
			BaseIPI:           it.BaseIPI,
			AliqIPI:           it.AliqIPI,
			ValorIPI:          it.ValorIPI,
			AliqPIS:           it.AliqPIS,
			ValorPIS:          it.ValorPIS,
			AliqCOFINS:        it.AliqCOFINS,
			ValorCOFINS:       it.ValorCOFINS,
			BaseICMSST:        it.BaseICMSST,
			AliqICMSST:        it.AliqICMSST,
			ValorICMSST:       it.ValorICMSST,
			MVA:               it.MVA,
			CstICMS:           it.CstICMS,
			CstIPI:            it.CstIPI,
			CstPIS:            it.CstPIS,
			CstCOFINS:         it.CstCOFINS,
			OrigemMercadoria:  it.OrigemMercadoria,
			Description:       it.Description,
			CreatedAt:         it.CreatedAt,
		})
	}
	return out
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
		ValorProdutos:       e.ValorProdutos,
		ValorFrete:          e.ValorFrete,
		ValorSeguro:         e.ValorSeguro,
		ValorDesconto:       e.ValorDesconto,
		ValorIPI:            e.ValorIPI,
		ValorICMS:           e.ValorICMS,
		ValorPIS:            e.ValorPIS,
		ValorCOFINS:         e.ValorCOFINS,
		ValorTotal:          e.ValorTotal,
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
			ID:                it.ID,
			FiscalEntryID:     it.FiscalEntryID,
			Sequence:          it.Sequence,
			ItemCode:          it.ItemCode,
			Ncm:               it.Ncm,
			Cfop:              it.Cfop,
			Quantity:          it.Quantity,
			UnitPrice:         it.UnitPrice,
			TotalPrice:        it.TotalPrice,
			BaseICMS:          it.BaseICMS,
			AliqICMS:          it.AliqICMS,
			ValorICMS:         it.ValorICMS,
			BaseIPI:           it.BaseIPI,
			AliqIPI:           it.AliqIPI,
			ValorIPI:          it.ValorIPI,
			ValorPIS:          it.ValorPIS,
			ValorCOFINS:       it.ValorCOFINS,
			CstICMS:           it.CstICMS,
			CstIPI:            it.CstIPI,
			CstPIS:            it.CstPIS,
			CstCOFINS:         it.CstCOFINS,
			GeraCreditoICMS:   it.GeraCreditoICMS,
			GeraCreditoIPI:    it.GeraCreditoIPI,
			GeraCreditoPIS:    it.GeraCreditoPIS,
			GeraCreditoCOFINS: it.GeraCreditoCOFINS,
			Description:       it.Description,
			Notes:             it.Notes,
			CreatedAt:         it.CreatedAt,
		})
	}
	return out
}

func toFiscalConfigResponse(c *entity.FiscalConfig) *response.FiscalConfigResponse {
	if c == nil {
		return nil
	}
	return &response.FiscalConfigResponse{
		ID:                        c.ID,
		CnpjEmpresa:               c.CnpjEmpresa,
		RazaoSocial:               c.RazaoSocial,
		IEEmpresa:                 c.IEEmpresa,
		RegimeTributario:          c.RegimeTributario,
		UFEmpresa:                 c.UFEmpresa,
		IcmsInternoAliquota:       c.IcmsInternoAliquota,
		IcmsDiferimentoPercentual: c.IcmsDiferimentoPercentual,
		FocusNfeToken:             c.FocusNfeToken,
		FocusNfeAmbiente:          c.FocusNfeAmbiente,
		JurosMes:                  c.JurosMes,
		MultaAtraso:               c.MultaAtraso,
		VencimentoIcmsDia:         c.VencimentoIcmsDia,
		VencimentoIPIDia:          c.VencimentoIPIDia,
		VencimentoPisCofinsDia:    c.VencimentoPisCofinsDia,
		Logradouro:                c.Logradouro,
		Numero:                    c.Numero,
		Complemento:               c.Complemento,
		Bairro:                    c.Bairro,
		Municipio:                 c.Municipio,
		CodigoMunicipio:           c.CodigoMunicipio,
		CEP:                       c.CEP,
		Telefone:                  c.Telefone,
		BrandColor:                c.BrandColor,
		CreatedAt:                 c.CreatedAt,
		UpdatedAt:                 c.UpdatedAt,
		UpdatedBy:                 c.UpdatedBy,
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
		ValorFrete:          c.ValorFrete,
		ValorSeguro:         c.ValorSeguro,
		ValorOutros:         c.ValorOutros,
		ValorTotal:          c.ValorTotal,
		ValorICMS:           c.ValorICMS,
		BaseICMS:            c.BaseICMS,
		AliqICMS:            c.AliqICMS,
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
		AliqIPI:     n.AliqIPI,
		AliqPis:     n.AliqPis,
		AliqCofins:  n.AliqCofins,
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

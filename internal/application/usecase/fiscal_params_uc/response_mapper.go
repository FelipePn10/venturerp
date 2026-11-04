package fiscal_params_uc

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/shopspring/decimal"
)

// ─── enum-pointer helpers ──────────────────────────────────────────────────────

func redTargetPtr(t *entity.IcmsReductionTarget) *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}

func acresTypePtr(t *entity.IcmsAcresType) *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}

func difalTypePtr(t *entity.IcmsDifalType) *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}

// ─── CFOP ──────────────────────────────────────────────────────────────────────

func toCFOPResponse(c *entity.CFOP) *response.CFOPResponse {
	if c == nil {
		return nil
	}
	return &response.CFOPResponse{
		ID:              c.ID,
		Code:            c.Code,
		Description:     c.Description,
		DescriptionFull: c.DescriptionFull,
		Utilization:     string(c.Utilization),
		OrigemClasIPI:   c.OrigemClasIPI,
		IndOperacao:     string(c.IndOperacao),
		TipoUtilizacao:  string(c.TipoUtilizacao),
		CodigoAnexoSN:   c.CodigoAnexoSN,
		DIFAL:           c.DIFAL,
		Doacao:          c.Doacao,
		IsActive:        c.IsActive,
		CreatedAt:       c.CreatedAt,
		Oficial:         c.Oficial.Oficial,
		InicioVigencia:  diaISO(c.Oficial.InicioVigencia),
		FimVigencia:     diaISO(c.Oficial.FimVigencia),
		IndNFe:          c.Oficial.NFe,
		IndComunica:     c.Oficial.Comunicacao,
		IndTransp:       c.Oficial.Transporte,
		IndDevol:        c.Oficial.Devolucao,
		IndRetor:        c.Oficial.Retorno,
		IndAnula:        c.Oficial.Anulacao,
		IndRemes:        c.Oficial.Remessa,
		IndComb:         c.Oficial.Combustivel,
		IndExcIBSCBS:    c.Oficial.ExclusivoIBSCBS,
	}
}

func diaISO(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func toCFOPResponses(list []*entity.CFOP) []*response.CFOPResponse {
	out := make([]*response.CFOPResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toCFOPResponse(c))
	}
	return out
}

// ─── Legal Device ──────────────────────────────────────────────────────────────

func toLegalDeviceResponse(d *entity.LegalDevice) *response.LegalDeviceResponse {
	if d == nil {
		return nil
	}
	return &response.LegalDeviceResponse{
		ID:          d.ID,
		Code:        d.Code,
		Type:        string(d.Type),
		Description: d.Description,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
	}
}

func toLegalDeviceResponses(list []*entity.LegalDevice) []*response.LegalDeviceResponse {
	out := make([]*response.LegalDeviceResponse, 0, len(list))
	for _, d := range list {
		out = append(out, toLegalDeviceResponse(d))
	}
	return out
}

// ─── Tax Param (ICMS/IPI) ──────────────────────────────────────────────────────

func toTaxParamResponse(t *entity.ICMSIPITaxParam) *response.TaxParamResponse {
	if t == nil {
		return nil
	}
	return &response.TaxParamResponse{
		ID:                               t.ID,
		NCMCode:                          t.NCMCode,
		ItemCode:                         t.ItemCode,
		ItemConfigMask:                   t.ItemConfigMask,
		UF:                               t.UF,
		OperationType:                    string(t.OperationType),
		CustomerCode:                     t.CustomerCode,
		CustomerEstablishmentCode:        t.CustomerEstablishmentCode,
		MarketSegmentID:                  t.MarketSegmentID,
		InvoiceTypeExitID:                t.InvoiceTypeExitID,
		InvoiceTypeEntryID:               t.InvoiceTypeEntryID,
		TaxTypeID:                        t.TaxTypeID,
		IsPreferred:                      t.IsPreferred,
		IsSimpleOptante:                  t.IsSimpleOptante,
		IsActive:                         t.IsActive,
		ICMSPctContrib:                   t.ICMSPctContrib.InexactFloat64(),
		LegalDeviceICMSContribID:         t.LegalDeviceICMSContribID,
		ICMSPctNonContrib:                t.ICMSPctNonContrib.InexactFloat64(),
		LegalDeviceICMSNonContribID:      t.LegalDeviceICMSNonContribID,
		ICMSRedPctContrib:                t.ICMSRedPctContrib.InexactFloat64(),
		ICMSRedTargetContrib:             redTargetPtr(t.ICMSRedTargetContrib),
		LegalDeviceICMSRedContribID:      t.LegalDeviceICMSRedContribID,
		ICMSRedPctNonContrib:             t.ICMSRedPctNonContrib.InexactFloat64(),
		ICMSRedTargetNonContrib:          redTargetPtr(t.ICMSRedTargetNonContrib),
		LegalDeviceICMSRedNonContribID:   t.LegalDeviceICMSRedNonContribID,
		ICMSDeferralPct:                  t.ICMSDeferralPct.InexactFloat64(),
		ICMSDeferralTarget:               redTargetPtr(t.ICMSDeferralTarget),
		LegalDeviceICMSDeferralID:        t.LegalDeviceICMSDeferralID,
		CodBenefRBC:                      t.CodBenefRBC,
		ICMSSubstPctContrib:              t.ICMSSubstPctContrib.InexactFloat64(),
		LegalDeviceICMSSubstContribID:    t.LegalDeviceICMSSubstContribID,
		ICMSSubstPctNonContrib:           t.ICMSSubstPctNonContrib.InexactFloat64(),
		LegalDeviceICMSSubstNonContribID: t.LegalDeviceICMSSubstNonContribID,
		ICMSSubstPctContribUC:            t.ICMSSubstPctContribUC.InexactFloat64(),
		ICMSSubstRedPct:                  t.ICMSSubstRedPct.InexactFloat64(),
		LegalDeviceICMSSubstRedID:        t.LegalDeviceICMSSubstRedID,
		ICMSInternalPct:                  t.ICMSInternalPct.InexactFloat64(),
		BCICMSSTModality:                 t.BCICMSSTModality,
		ICMSPctForSTContrib:              t.ICMSPctForSTContrib.InexactFloat64(),
		ICMSPctForSTNonContrib:           t.ICMSPctForSTNonContrib.InexactFloat64(),
		CSTSituationB:                    t.CSTSituationB,
		CSOSNIGMS:                        t.CSOSNIGMS,
		CSTICMSContrib:                   t.CSTICMSContrib,
		CSTICMSNonContrib:                t.CSTICMSNonContrib,
		CodBeneficioFiscal:               t.CodBeneficioFiscal,
		CSTICMSContribDev:                t.CSTICMSContribDev,
		CSTICMSNonContribDev:             t.CSTICMSNonContribDev,
		IPIRedPctContrib:                 t.IPIRedPctContrib.InexactFloat64(),
		IPIRedTargetContrib:              redTargetPtr(t.IPIRedTargetContrib),
		LegalDeviceIPIContribID:          t.LegalDeviceIPIContribID,
		IPIRedPctNonContrib:              t.IPIRedPctNonContrib.InexactFloat64(),
		IPIRedTargetNonContrib:           redTargetPtr(t.IPIRedTargetNonContrib),
		LegalDeviceIPINonContribID:       t.LegalDeviceIPINonContribID,
		CSTIPIExit:                       t.CSTIPIExit,
		CSTIPIEntry:                      t.CSTIPIEntry,
		ICMSPctOrigins1238:               t.ICMSPctOrigins1238.InexactFloat64(),
		CalcBaseRedFCI:                   t.CalcBaseRedFCI,
		ICMSSubstPctOrigins1238:          t.ICMSSubstPctOrigins1238.InexactFloat64(),
		CSTICMSFci:                       t.CSTICMSFci,
		UsesICMSZonaFranca:               t.UsesICMSZonaFranca,
		DifAliqSTContribUC:               t.DifAliqSTContribUC.InexactFloat64(),
		CodBenefContrib:                  t.CodBenefContrib,
		CodBenefNonContrib:               t.CodBenefNonContrib,
		ICMSAcresPctContrib:              t.ICMSAcresPctContrib.InexactFloat64(),
		ICMSAcresTypeContrib:             acresTypePtr(t.ICMSAcresTypeContrib),
		ICMSAcresSumContrib:              t.ICMSAcresSumContrib,
		ICMSAcresPctNonContrib:           t.ICMSAcresPctNonContrib.InexactFloat64(),
		ICMSAcresTypeNonContrib:          acresTypePtr(t.ICMSAcresTypeNonContrib),
		ICMSAcresSumNonContrib:           t.ICMSAcresSumNonContrib,
		ICMSSTAcresPctContrib:            t.ICMSSTAcresPctContrib.InexactFloat64(),
		ICMSSTAcresTypeContrib:           acresTypePtr(t.ICMSSTAcresTypeContrib),
		ICMSSTAcresSumContrib:            t.ICMSSTAcresSumContrib,
		ICMSSTAcresPctNonContrib:         t.ICMSSTAcresPctNonContrib.InexactFloat64(),
		ICMSSTAcresTypeNonContrib:        acresTypePtr(t.ICMSSTAcresTypeNonContrib),
		ICMSSTAcresSumNonContrib:         t.ICMSSTAcresSumNonContrib,
		FCPSTPartilhaPct:                 t.FCPSTPartilhaPct.InexactFloat64(),
		ICMSDifalRedPct:                  t.ICMSDifalRedPct.InexactFloat64(),
		ICMSDifalType:                    difalTypePtr(t.ICMSDifalType),
		DifalPurchaseRedPct:              t.DifalPurchaseRedPct.InexactFloat64(),
		DifalPurchaseRedTarget:           redTargetPtr(t.DifalPurchaseRedTarget),
		FinalidadeCompra:                 t.FinalidadeCompra,
		MvaPct:                           t.MvaPct.InexactFloat64(),
		MvaAjustarInterestadual:          t.MvaAjustarInterestadual,
		SupplierCode:                     t.SupplierCode,
		OrigemReducaoID:                  t.OrigemReducaoID,
		CreatedAt:                        t.CreatedAt,
	}
}

func toTaxParamResponses(list []*entity.ICMSIPITaxParam) []*response.TaxParamResponse {
	out := make([]*response.TaxParamResponse, 0, len(list))
	for _, t := range list {
		out = append(out, toTaxParamResponse(t))
	}
	return out
}

// ─── DAPI Transfer Reason ──────────────────────────────────────────────────────

func toDAPITransferReasonResponse(d *entity.DAPITransferReason) *response.DAPITransferReasonResponse {
	if d == nil {
		return nil
	}
	return &response.DAPITransferReasonResponse{
		ID:          d.ID,
		Code:        d.Code,
		Reason:      d.Reason,
		Destination: d.Destination,
		ValidFrom:   d.ValidFrom,
		ValidTo:     d.ValidTo,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
	}
}

func toDAPITransferReasonResponses(list []*entity.DAPITransferReason) []*response.DAPITransferReasonResponse {
	out := make([]*response.DAPITransferReasonResponse, 0, len(list))
	for _, d := range list {
		out = append(out, toDAPITransferReasonResponse(d))
	}
	return out
}

// ─── ICMS Apuração Adjustment Code ─────────────────────────────────────────────

func toICMSApuracaoAdjCodeResponse(c *entity.ICMSApuracaoAdjustmentCode) *response.ICMSApuracaoAdjCodeResponse {
	if c == nil {
		return nil
	}
	return &response.ICMSApuracaoAdjCodeResponse{
		ID:          c.ID,
		Code:        c.Code,
		UF:          c.UF,
		Description: c.Description,
		ValidFrom:   c.ValidFrom,
		ValidTo:     c.ValidTo,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
	}
}

func toICMSApuracaoAdjCodeResponses(list []*entity.ICMSApuracaoAdjustmentCode) []*response.ICMSApuracaoAdjCodeResponse {
	out := make([]*response.ICMSApuracaoAdjCodeResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toICMSApuracaoAdjCodeResponse(c))
	}
	return out
}

// ─── ICMS Adjustment Code ──────────────────────────────────────────────────────

func toICMSAdjustmentCodeResponse(c *entity.ICMSAdjustmentCode) *response.ICMSAdjustmentCodeResponse {
	if c == nil {
		return nil
	}
	return &response.ICMSAdjustmentCodeResponse{
		ID:          c.ID,
		UF:          c.UF,
		Code:        c.Code,
		Description: c.Description,
		TableRef:    string(c.TableRef),
		ValidFrom:   c.ValidFrom,
		ValidTo:     c.ValidTo,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
	}
}

func toICMSAdjustmentCodeResponses(list []*entity.ICMSAdjustmentCode) []*response.ICMSAdjustmentCodeResponse {
	out := make([]*response.ICMSAdjustmentCodeResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toICMSAdjustmentCodeResponse(c))
	}
	return out
}

// ─── ICMS Apuração Line ────────────────────────────────────────────────────────

func toICMSApuracaoLineResponse(l *entity.ICMSApuracaoLine) *response.ICMSApuracaoLineResponse {
	if l == nil {
		return nil
	}
	return &response.ICMSApuracaoLineResponse{
		ID:                l.ID,
		Code:              l.Code,
		Description:       l.Description,
		LineType:          string(l.LineType),
		AcceptsEntries:    l.AcceptsEntries,
		Nature:            l.Nature,
		ApuracaoAdjCodeID: l.ApuracaoAdjCodeID,
		IsActive:          l.IsActive,
		CreatedAt:         l.CreatedAt,
	}
}

func toICMSApuracaoLineResponses(list []*entity.ICMSApuracaoLine) []*response.ICMSApuracaoLineResponse {
	out := make([]*response.ICMSApuracaoLineResponse, 0, len(list))
	for _, l := range list {
		out = append(out, toICMSApuracaoLineResponse(l))
	}
	return out
}

// ─── ICMS Summary Entry / Note / Additional ────────────────────────────────────

func toICMSSummaryEntryResponse(e *entity.ICMSSummaryEntry) *response.ICMSSummaryEntryResponse {
	if e == nil {
		return nil
	}
	return &response.ICMSSummaryEntryResponse{
		ID:             e.ID,
		Period:         e.Period,
		UF:             e.UF,
		CFOPID:         e.CFOPID,
		ICMSBase:       e.ICMSBase.InexactFloat64(),
		ICMSValue:      e.ICMSValue.InexactFloat64(),
		ICMSBaseOther:  e.ICMSBaseOther.InexactFloat64(),
		ICMSValueOther: e.ICMSValueOther.InexactFloat64(),
		IsActive:       e.IsActive,
		CreatedAt:      e.CreatedAt,
	}
}

func toICMSSummaryEntryResponses(list []*entity.ICMSSummaryEntry) []*response.ICMSSummaryEntryResponse {
	out := make([]*response.ICMSSummaryEntryResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toICMSSummaryEntryResponse(e))
	}
	return out
}

func toICMSSummaryEntryNoteResponse(n *entity.ICMSSummaryEntryNote) *response.ICMSSummaryEntryNoteResponse {
	if n == nil {
		return nil
	}
	return &response.ICMSSummaryEntryNoteResponse{
		ID:             n.ID,
		SummaryEntryID: n.SummaryEntryID,
		NoteNumber:     n.NoteNumber,
		NoteSeries:     n.NoteSeries,
		EmitterCNPJ:    n.EmitterCNPJ,
		IssueDate:      n.IssueDate,
		ItemValue:      n.ItemValue.InexactFloat64(),
		ICMSBase:       n.ICMSBase.InexactFloat64(),
		ICMSValue:      n.ICMSValue.InexactFloat64(),
		Observation:    n.Observation,
		CreatedAt:      n.CreatedAt,
	}
}

func toICMSSummaryEntryAdditionalResponse(a *entity.ICMSSummaryEntryAdditional) *response.ICMSSummaryEntryAdditionalResponse {
	if a == nil {
		return nil
	}
	return &response.ICMSSummaryEntryAdditionalResponse{
		ID:                   a.ID,
		SummaryEntryID:       a.SummaryEntryID,
		Sequence:             a.Sequence,
		ArrecadacaoIndicator: string(a.ArrecadacaoIndicator),
		Processo:             a.Processo,
		Arrecadacao:          a.Arrecadacao,
		Description:          a.Description,
		DIEFTable:            a.DIEFTable,
		DIEFCode:             a.DIEFCode,
		CreatedAt:            a.CreatedAt,
	}
}

func toICMSSummaryEntryAdditionalResponses(list []*entity.ICMSSummaryEntryAdditional) []*response.ICMSSummaryEntryAdditionalResponse {
	out := make([]*response.ICMSSummaryEntryAdditionalResponse, 0, len(list))
	for _, a := range list {
		out = append(out, toICMSSummaryEntryAdditionalResponse(a))
	}
	return out
}

// ─── Simples Nacional Apuração ─────────────────────────────────────────────────

func toSimplesNacionalApuracaoResponse(s *entity.SimplesNacionalApuracao) *response.SimplesNacionalApuracaoResponse {
	if s == nil {
		return nil
	}
	return &response.SimplesNacionalApuracaoResponse{
		ID:                  s.ID,
		Period:              s.Period,
		Annex:               string(s.Annex),
		ReceitaInterna:      s.ReceitaInterna.InexactFloat64(),
		ReceitaExterna:      s.ReceitaExterna.InexactFloat64(),
		FolhaPagamento:      s.FolhaPagamento.InexactFloat64(),
		ReceitaBruta12M:     s.ReceitaBruta12M.InexactFloat64(),
		SimplesRecolhido:    s.SimplesRecolhido.InexactFloat64(),
		AliquotaNominal:     s.AliquotaNominal.InexactFloat64(),
		AliquotaEfetiva:     s.AliquotaEfetiva.InexactFloat64(),
		AliquotaEfetivaICMS: s.AliquotaEfetivaICMS.InexactFloat64(),
		ParcelaDeduzir:      s.ParcelaDeduzir.InexactFloat64(),
		Observation:         s.Observation,
		IsActive:            s.IsActive,
		CreatedAt:           s.CreatedAt,
	}
}

func toSimplesNacionalApuracaoResponses(list []*entity.SimplesNacionalApuracao) []*response.SimplesNacionalApuracaoResponse {
	out := make([]*response.SimplesNacionalApuracaoResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toSimplesNacionalApuracaoResponse(s))
	}
	return out
}

// ─── ICMS Reduction/Substitution ───────────────────────────────────────────────

func toICMSReductionSubstitutionResponse(r *entity.ICMSReductionSubstitution) *response.ICMSReductionSubstitutionResponse {
	if r == nil {
		return nil
	}
	return &response.ICMSReductionSubstitutionResponse{
		ID:                               r.ID,
		ItemID:                           r.ItemID,
		ItemMask:                         r.ItemMask,
		NCMCode:                          r.NCMCode,
		UF:                               r.UF,
		OperationType:                    string(r.OperationType),
		CustomerID:                       r.CustomerID,
		EstablishmentID:                  r.EstablishmentID,
		SupplierID:                       r.SupplierID,
		InvoiceTypeOutID:                 r.InvoiceTypeOutID,
		InvoiceTypeInID:                  r.InvoiceTypeInID,
		MarketSegmentID:                  r.MarketSegmentID,
		IsPreferential:                   r.IsPreferential,
		ICMSPctContrib:                   fptrD(r.ICMSPctContrib),
		ICMSPctNonContrib:                fptrD(r.ICMSPctNonContrib),
		LegalDeviceICMSContribID:         r.LegalDeviceICMSContribID,
		LegalDeviceICMSNonContribID:      r.LegalDeviceICMSNonContribID,
		ICMSRedPctContrib:                fptrD(r.ICMSRedPctContrib),
		ICMSRedTargetContrib:             string(r.ICMSRedTargetContrib),
		ICMSRedPctNonContrib:             fptrD(r.ICMSRedPctNonContrib),
		ICMSRedTargetNonContrib:          string(r.ICMSRedTargetNonContrib),
		LegalDeviceICMSRedContribID:      r.LegalDeviceICMSRedContribID,
		LegalDeviceICMSRedNonContribID:   r.LegalDeviceICMSRedNonContribID,
		ICMSDeferralPct:                  fptrD(r.ICMSDeferralPct),
		ICMSDeferralTarget:               string(r.ICMSDeferralTarget),
		LegalDeviceICMSDeferralID:        r.LegalDeviceICMSDeferralID,
		ICMSDeferralBenefitCode:          r.ICMSDeferralBenefitCode,
		ICMSSubstPctContrib:              fptrD(r.ICMSSubstPctContrib),
		ICMSSubstPctNonContrib:           fptrD(r.ICMSSubstPctNonContrib),
		ICMSSubstPctContribUC:            fptrD(r.ICMSSubstPctContribUC),
		ICMSSubstRedPct:                  fptrD(r.ICMSSubstRedPct),
		ICMSInternalPct:                  fptrD(r.ICMSInternalPct),
		LegalDeviceICMSSubstContribID:    r.LegalDeviceICMSSubstContribID,
		LegalDeviceICMSSubstNonContribID: r.LegalDeviceICMSSubstNonContribID,
		LegalDeviceICMSSubstRedID:        r.LegalDeviceICMSSubstRedID,
		ModBCICMSST:                      r.ModBCICMSST,
		ICMSPctForSTContrib:              fptrD(r.ICMSPctForSTContrib),
		ICMSPctForSTNonContrib:           fptrD(r.ICMSPctForSTNonContrib),
		CSTICMSContrib:                   r.CSTICMSContrib,
		CSTICMSNonContrib:                r.CSTICMSNonContrib,
		CSOSNTICMS:                       r.CSOSNTICMS,
		CSTICMSContribDev:                r.CSTICMSContribDev,
		CSTICMSNonContribDev:             r.CSTICMSNonContribDev,
		CSTSitTribB:                      r.CSTSitTribB,
		FiscalBenefitCodeContrib:         r.FiscalBenefitCodeContrib,
		FiscalBenefitCodeNonContrib:      r.FiscalBenefitCodeNonContrib,
		FiscalBenefitCode:                r.FiscalBenefitCode,
		IPIRedPctContrib:                 fptrD(r.IPIRedPctContrib),
		IPIRedTargetContrib:              string(r.IPIRedTargetContrib),
		IPIRedPctNonContrib:              fptrD(r.IPIRedPctNonContrib),
		IPIRedTargetNonContrib:           string(r.IPIRedTargetNonContrib),
		LegalDeviceIPIContribID:          r.LegalDeviceIPIContribID,
		LegalDeviceIPINonContribID:       r.LegalDeviceIPINonContribID,
		CSTIPIOut:                        r.CSTIPIOut,
		CSTIPIIn:                         r.CSTIPIIn,
		FCIICMSPct:                       fptrD(r.FCIICMSPct),
		FCIReduceBase:                    r.FCIReduceBase,
		FCIICMSSubstPct:                  fptrD(r.FCIICMSSubstPct),
		FCICSTICMs:                       r.FCICSTICMs,
		FCIUseICMSZF:                     r.FCIUseICMSZF,
		FCIDIFALSTContribUCPct:           fptrD(r.FCIDIFALSTContribUCPct),
		ICMSAddPctContrib:                fptrD(r.ICMSAddPctContrib),
		ICMSAddTypeContrib:               r.ICMSAddTypeContrib,
		ICMSAddSumAliqContrib:            r.ICMSAddSumAliqContrib,
		ICMSAddPctNonContrib:             fptrD(r.ICMSAddPctNonContrib),
		ICMSAddTypeNonContrib:            r.ICMSAddTypeNonContrib,
		ICMSAddSumAliqNonContrib:         r.ICMSAddSumAliqNonContrib,
		ICMSSTAddPctContrib:              fptrD(r.ICMSSTAddPctContrib),
		ICMSSTAddTypeContrib:             r.ICMSSTAddTypeContrib,
		ICMSSTAddPctNonContrib:           fptrD(r.ICMSSTAddPctNonContrib),
		ICMSSTAddTypeNonContrib:          r.ICMSSTAddTypeNonContrib,
		FCPPartitionPct:                  fptrD(r.FCPPartitionPct),
		DIFALICMSRedPct:                  fptrD(r.DIFALICMSRedPct),
		DIFALICMSType:                    r.DIFALICMSType,
		DIFALPurchaseRedPct:              fptrD(r.DIFALPurchaseRedPct),
		IsSimplesOptante:                 r.IsSimplesOptante,
		IsActive:                         r.IsActive,
		CreatedAt:                        r.CreatedAt,
	}
}

func toICMSReductionSubstitutionResponses(list []*entity.ICMSReductionSubstitution) []*response.ICMSReductionSubstitutionResponse {
	out := make([]*response.ICMSReductionSubstitutionResponse, 0, len(list))
	for _, r := range list {
		out = append(out, toICMSReductionSubstitutionResponse(r))
	}
	return out
}

// ─── ICMS-ST Restitution ───────────────────────────────────────────────────────

func toICMSSTRestitutionResponse(r *entity.ICMSSTRestitution) *response.ICMSSTRestitutionResponse {
	if r == nil {
		return nil
	}
	return &response.ICMSSTRestitutionResponse{
		ID:                      r.ID,
		EmpresaID:               r.EmpresaID,
		Period:                  r.Period,
		RestitutionType:         string(r.RestitutionType),
		UF:                      r.UF,
		OrigDocModel:            r.OrigDocModel,
		OrigDocSeries:           r.OrigDocSeries,
		OrigDocNumber:           r.OrigDocNumber,
		OrigDocDate:             r.OrigDocDate,
		OrigEmitterCNPJ:         r.OrigEmitterCNPJ,
		OrigEmitterIE:           r.OrigEmitterIE,
		ItemID:                  r.ItemID,
		ItemCode:                r.ItemCode,
		CFOP:                    r.CFOP,
		MotivoCode:              r.MotivoCode,
		CSTICMS:                 r.CSTICMS,
		ICMSSTBase:              r.ICMSSTBase.InexactFloat64(),
		ICMSSTAliq:              r.ICMSSTAliq.InexactFloat64(),
		ICMSSTValue:             r.ICMSSTValue.InexactFloat64(),
		ICMSSTBaseRestitution:   r.ICMSSTBaseRestitution.InexactFloat64(),
		ICMSSTValueRestitution:  r.ICMSSTValueRestitution.InexactFloat64(),
		ICMSSTConsolidatedBase:  r.ICMSSTConsolidatedBase.InexactFloat64(),
		ICMSSTConsolidatedValue: r.ICMSSTConsolidatedValue.InexactFloat64(),
		H030IndEstoque:          r.H030IndEstoque,
		SpedBlock:               r.SpedBlock,
		IsActive:                r.IsActive,
		CreatedAt:               r.CreatedAt,
	}
}

func toICMSSTRestitutionResponses(list []*entity.ICMSSTRestitution) []*response.ICMSSTRestitutionResponse {
	out := make([]*response.ICMSSTRestitutionResponse, 0, len(list))
	for _, r := range list {
		out = append(out, toICMSSTRestitutionResponse(r))
	}
	return out
}

// ─── Special Adjustment Note ───────────────────────────────────────────────────

func toSpecialAdjustmentNoteResponse(n *entity.SpecialAdjustmentNote) *response.SpecialAdjustmentNoteResponse {
	if n == nil {
		return nil
	}
	return &response.SpecialAdjustmentNoteResponse{
		ID:                      n.ID,
		EmpresaID:               n.EmpresaID,
		Purpose:                 string(n.Purpose),
		Status:                  string(n.Status),
		Number:                  n.Number,
		Series:                  n.Series,
		IssueDate:               n.IssueDate,
		Period:                  n.Period,
		InvoiceTypeID:           n.InvoiceTypeID,
		CFOPID:                  n.CFOPID,
		ICMSApuracaoLineID:      n.ICMSApuracaoLineID,
		AdjustmentCodeID:        n.AdjustmentCodeID,
		AdjustmentDocCodeID:     n.AdjustmentDocCodeID,
		History:                 n.History,
		AutoGenerateSummary:     n.AutoGenerateSummary,
		GeneratedSummaryEntryID: n.GeneratedSummaryEntryID,
		TotalValue:              n.TotalValue.InexactFloat64(),
		TotalICMS:               n.TotalICMS.InexactFloat64(),
		TotalIPI:                n.TotalIPI.InexactFloat64(),
		Observation:             n.Observation,
		Items:                   toSpecialAdjustmentNoteItemValues(n.Items),
		CreatedAt:               n.CreatedAt,
	}
}

func toSpecialAdjustmentNoteResponses(list []*entity.SpecialAdjustmentNote) []*response.SpecialAdjustmentNoteResponse {
	out := make([]*response.SpecialAdjustmentNoteResponse, 0, len(list))
	for _, n := range list {
		out = append(out, toSpecialAdjustmentNoteResponse(n))
	}
	return out
}

func toSpecialAdjustmentNoteItemResponse(it *entity.SpecialAdjustmentNoteItem) *response.SpecialAdjustmentNoteItemResponse {
	if it == nil {
		return nil
	}
	return &response.SpecialAdjustmentNoteItemResponse{
		ID:                it.ID,
		NoteID:            it.NoteID,
		Sequence:          it.Sequence,
		ItemID:            it.ItemID,
		ItemCode:          it.ItemCode,
		Description:       it.Description,
		Quantity:          it.Quantity.InexactFloat64(),
		Unit:              it.Unit,
		UnitValue:         it.UnitValue.InexactFloat64(),
		TotalValue:        it.TotalValue.InexactFloat64(),
		ICMSBase:          it.ICMSBase.InexactFloat64(),
		ICMSPct:           it.ICMSPct.InexactFloat64(),
		ICMSDeferralPct:   it.ICMSDeferralPct.InexactFloat64(),
		ICMSValue:         it.ICMSValue.InexactFloat64(),
		ICMSDeferredValue: it.ICMSDeferredValue.InexactFloat64(),
		IPIBase:           it.IPIBase.InexactFloat64(),
		IPIPct:            it.IPIPct.InexactFloat64(),
		IPIValue:          it.IPIValue.InexactFloat64(),
		CSTICMS:           it.CSTICMS,
		CSTIPI:            it.CSTIPI,
		CFOPID:            it.CFOPID,
		CreatedAt:         it.CreatedAt,
	}
}

func toSpecialAdjustmentNoteItemValues(items []entity.SpecialAdjustmentNoteItem) []response.SpecialAdjustmentNoteItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.SpecialAdjustmentNoteItemResponse, 0, len(items))
	for i := range items {
		out = append(out, *toSpecialAdjustmentNoteItemResponse(&items[i]))
	}
	return out
}

func toSpecialAdjustmentNoteItemResponses(items []*entity.SpecialAdjustmentNoteItem) []*response.SpecialAdjustmentNoteItemResponse {
	out := make([]*response.SpecialAdjustmentNoteItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toSpecialAdjustmentNoteItemResponse(it))
	}
	return out
}

// fptrD leva o percentual (decimal) à resposta, que continua numérica.
func fptrD(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	f := d.InexactFloat64()
	return &f
}

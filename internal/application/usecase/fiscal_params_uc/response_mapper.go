package fiscal_params_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
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
	}
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
		ICMSPctContrib:                   t.ICMSPctContrib,
		LegalDeviceICMSContribID:         t.LegalDeviceICMSContribID,
		ICMSPctNonContrib:                t.ICMSPctNonContrib,
		LegalDeviceICMSNonContribID:      t.LegalDeviceICMSNonContribID,
		ICMSRedPctContrib:                t.ICMSRedPctContrib,
		ICMSRedTargetContrib:             redTargetPtr(t.ICMSRedTargetContrib),
		LegalDeviceICMSRedContribID:      t.LegalDeviceICMSRedContribID,
		ICMSRedPctNonContrib:             t.ICMSRedPctNonContrib,
		ICMSRedTargetNonContrib:          redTargetPtr(t.ICMSRedTargetNonContrib),
		LegalDeviceICMSRedNonContribID:   t.LegalDeviceICMSRedNonContribID,
		ICMSDeferralPct:                  t.ICMSDeferralPct,
		ICMSDeferralTarget:               redTargetPtr(t.ICMSDeferralTarget),
		LegalDeviceICMSDeferralID:        t.LegalDeviceICMSDeferralID,
		CodBenefRBC:                      t.CodBenefRBC,
		ICMSSubstPctContrib:              t.ICMSSubstPctContrib,
		LegalDeviceICMSSubstContribID:    t.LegalDeviceICMSSubstContribID,
		ICMSSubstPctNonContrib:           t.ICMSSubstPctNonContrib,
		LegalDeviceICMSSubstNonContribID: t.LegalDeviceICMSSubstNonContribID,
		ICMSSubstPctContribUC:            t.ICMSSubstPctContribUC,
		ICMSSubstRedPct:                  t.ICMSSubstRedPct,
		LegalDeviceICMSSubstRedID:        t.LegalDeviceICMSSubstRedID,
		ICMSInternalPct:                  t.ICMSInternalPct,
		BCICMSSTModality:                 t.BCICMSSTModality,
		ICMSPctForSTContrib:              t.ICMSPctForSTContrib,
		ICMSPctForSTNonContrib:           t.ICMSPctForSTNonContrib,
		CSTSituationB:                    t.CSTSituationB,
		CSOSNIGMS:                        t.CSOSNIGMS,
		CSTICMSContrib:                   t.CSTICMSContrib,
		CSTICMSNonContrib:                t.CSTICMSNonContrib,
		CodBeneficioFiscal:               t.CodBeneficioFiscal,
		CSTICMSContribDev:                t.CSTICMSContribDev,
		CSTICMSNonContribDev:             t.CSTICMSNonContribDev,
		IPIRedPctContrib:                 t.IPIRedPctContrib,
		IPIRedTargetContrib:              redTargetPtr(t.IPIRedTargetContrib),
		LegalDeviceIPIContribID:          t.LegalDeviceIPIContribID,
		IPIRedPctNonContrib:              t.IPIRedPctNonContrib,
		IPIRedTargetNonContrib:           redTargetPtr(t.IPIRedTargetNonContrib),
		LegalDeviceIPINonContribID:       t.LegalDeviceIPINonContribID,
		CSTIPIExit:                       t.CSTIPIExit,
		CSTIPIEntry:                      t.CSTIPIEntry,
		ICMSPctOrigins1238:               t.ICMSPctOrigins1238,
		CalcBaseRedFCI:                   t.CalcBaseRedFCI,
		ICMSSubstPctOrigins1238:          t.ICMSSubstPctOrigins1238,
		CSTICMSFci:                       t.CSTICMSFci,
		UsesICMSZonaFranca:               t.UsesICMSZonaFranca,
		DifAliqSTContribUC:               t.DifAliqSTContribUC,
		CodBenefContrib:                  t.CodBenefContrib,
		CodBenefNonContrib:               t.CodBenefNonContrib,
		ICMSAcresPctContrib:              t.ICMSAcresPctContrib,
		ICMSAcresTypeContrib:             acresTypePtr(t.ICMSAcresTypeContrib),
		ICMSAcresSumContrib:              t.ICMSAcresSumContrib,
		ICMSAcresPctNonContrib:           t.ICMSAcresPctNonContrib,
		ICMSAcresTypeNonContrib:          acresTypePtr(t.ICMSAcresTypeNonContrib),
		ICMSAcresSumNonContrib:           t.ICMSAcresSumNonContrib,
		ICMSSTAcresPctContrib:            t.ICMSSTAcresPctContrib,
		ICMSSTAcresTypeContrib:           acresTypePtr(t.ICMSSTAcresTypeContrib),
		ICMSSTAcresSumContrib:            t.ICMSSTAcresSumContrib,
		ICMSSTAcresPctNonContrib:         t.ICMSSTAcresPctNonContrib,
		ICMSSTAcresTypeNonContrib:        acresTypePtr(t.ICMSSTAcresTypeNonContrib),
		ICMSSTAcresSumNonContrib:         t.ICMSSTAcresSumNonContrib,
		FCPSTPartilhaPct:                 t.FCPSTPartilhaPct,
		ICMSDifalRedPct:                  t.ICMSDifalRedPct,
		ICMSDifalType:                    difalTypePtr(t.ICMSDifalType),
		DifalPurchaseRedPct:              t.DifalPurchaseRedPct,
		DifalPurchaseRedTarget:           redTargetPtr(t.DifalPurchaseRedTarget),
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
		ICMSBase:       e.ICMSBase,
		ICMSValue:      e.ICMSValue,
		ICMSBaseOther:  e.ICMSBaseOther,
		ICMSValueOther: e.ICMSValueOther,
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
		ItemValue:      n.ItemValue,
		ICMSBase:       n.ICMSBase,
		ICMSValue:      n.ICMSValue,
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
		ReceitaInterna:      s.ReceitaInterna,
		ReceitaExterna:      s.ReceitaExterna,
		FolhaPagamento:      s.FolhaPagamento,
		ReceitaBruta12M:     s.ReceitaBruta12M,
		SimplesRecolhido:    s.SimplesRecolhido,
		AliquotaNominal:     s.AliquotaNominal,
		AliquotaEfetiva:     s.AliquotaEfetiva,
		AliquotaEfetivaICMS: s.AliquotaEfetivaICMS,
		ParcelaDeduzir:      s.ParcelaDeduzir,
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
		ICMSPctContrib:                   r.ICMSPctContrib,
		ICMSPctNonContrib:                r.ICMSPctNonContrib,
		LegalDeviceICMSContribID:         r.LegalDeviceICMSContribID,
		LegalDeviceICMSNonContribID:      r.LegalDeviceICMSNonContribID,
		ICMSRedPctContrib:                r.ICMSRedPctContrib,
		ICMSRedTargetContrib:             string(r.ICMSRedTargetContrib),
		ICMSRedPctNonContrib:             r.ICMSRedPctNonContrib,
		ICMSRedTargetNonContrib:          string(r.ICMSRedTargetNonContrib),
		LegalDeviceICMSRedContribID:      r.LegalDeviceICMSRedContribID,
		LegalDeviceICMSRedNonContribID:   r.LegalDeviceICMSRedNonContribID,
		ICMSDeferralPct:                  r.ICMSDeferralPct,
		ICMSDeferralTarget:               string(r.ICMSDeferralTarget),
		LegalDeviceICMSDeferralID:        r.LegalDeviceICMSDeferralID,
		ICMSDeferralBenefitCode:          r.ICMSDeferralBenefitCode,
		ICMSSubstPctContrib:              r.ICMSSubstPctContrib,
		ICMSSubstPctNonContrib:           r.ICMSSubstPctNonContrib,
		ICMSSubstPctContribUC:            r.ICMSSubstPctContribUC,
		ICMSSubstRedPct:                  r.ICMSSubstRedPct,
		ICMSInternalPct:                  r.ICMSInternalPct,
		LegalDeviceICMSSubstContribID:    r.LegalDeviceICMSSubstContribID,
		LegalDeviceICMSSubstNonContribID: r.LegalDeviceICMSSubstNonContribID,
		LegalDeviceICMSSubstRedID:        r.LegalDeviceICMSSubstRedID,
		ModBCICMSST:                      r.ModBCICMSST,
		ICMSPctForSTContrib:              r.ICMSPctForSTContrib,
		ICMSPctForSTNonContrib:           r.ICMSPctForSTNonContrib,
		CSTICMSContrib:                   r.CSTICMSContrib,
		CSTICMSNonContrib:                r.CSTICMSNonContrib,
		CSOSNTICMS:                       r.CSOSNTICMS,
		CSTICMSContribDev:                r.CSTICMSContribDev,
		CSTICMSNonContribDev:             r.CSTICMSNonContribDev,
		CSTSitTribB:                      r.CSTSitTribB,
		FiscalBenefitCodeContrib:         r.FiscalBenefitCodeContrib,
		FiscalBenefitCodeNonContrib:      r.FiscalBenefitCodeNonContrib,
		FiscalBenefitCode:                r.FiscalBenefitCode,
		IPIRedPctContrib:                 r.IPIRedPctContrib,
		IPIRedTargetContrib:              string(r.IPIRedTargetContrib),
		IPIRedPctNonContrib:              r.IPIRedPctNonContrib,
		IPIRedTargetNonContrib:           string(r.IPIRedTargetNonContrib),
		LegalDeviceIPIContribID:          r.LegalDeviceIPIContribID,
		LegalDeviceIPINonContribID:       r.LegalDeviceIPINonContribID,
		CSTIPIOut:                        r.CSTIPIOut,
		CSTIPIIn:                         r.CSTIPIIn,
		FCIICMSPct:                       r.FCIICMSPct,
		FCIReduceBase:                    r.FCIReduceBase,
		FCIICMSSubstPct:                  r.FCIICMSSubstPct,
		FCICSTICMs:                       r.FCICSTICMs,
		FCIUseICMSZF:                     r.FCIUseICMSZF,
		FCIDIFALSTContribUCPct:           r.FCIDIFALSTContribUCPct,
		ICMSAddPctContrib:                r.ICMSAddPctContrib,
		ICMSAddTypeContrib:               r.ICMSAddTypeContrib,
		ICMSAddSumAliqContrib:            r.ICMSAddSumAliqContrib,
		ICMSAddPctNonContrib:             r.ICMSAddPctNonContrib,
		ICMSAddTypeNonContrib:            r.ICMSAddTypeNonContrib,
		ICMSAddSumAliqNonContrib:         r.ICMSAddSumAliqNonContrib,
		ICMSSTAddPctContrib:              r.ICMSSTAddPctContrib,
		ICMSSTAddTypeContrib:             r.ICMSSTAddTypeContrib,
		ICMSSTAddPctNonContrib:           r.ICMSSTAddPctNonContrib,
		ICMSSTAddTypeNonContrib:          r.ICMSSTAddTypeNonContrib,
		FCPPartitionPct:                  r.FCPPartitionPct,
		DIFALICMSRedPct:                  r.DIFALICMSRedPct,
		DIFALICMSType:                    r.DIFALICMSType,
		DIFALPurchaseRedPct:              r.DIFALPurchaseRedPct,
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
		ICMSSTBase:              r.ICMSSTBase,
		ICMSSTAliq:              r.ICMSSTAliq,
		ICMSSTValue:             r.ICMSSTValue,
		ICMSSTBaseRestitution:   r.ICMSSTBaseRestitution,
		ICMSSTValueRestitution:  r.ICMSSTValueRestitution,
		ICMSSTConsolidatedBase:  r.ICMSSTConsolidatedBase,
		ICMSSTConsolidatedValue: r.ICMSSTConsolidatedValue,
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
		TotalValue:              n.TotalValue,
		TotalICMS:               n.TotalICMS,
		TotalIPI:                n.TotalIPI,
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
		Quantity:          it.Quantity,
		Unit:              it.Unit,
		UnitValue:         it.UnitValue,
		TotalValue:        it.TotalValue,
		ICMSBase:          it.ICMSBase,
		ICMSPct:           it.ICMSPct,
		ICMSDeferralPct:   it.ICMSDeferralPct,
		ICMSValue:         it.ICMSValue,
		ICMSDeferredValue: it.ICMSDeferredValue,
		IPIBase:           it.IPIBase,
		IPIPct:            it.IPIPct,
		IPIValue:          it.IPIValue,
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

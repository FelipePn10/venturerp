package quality_uc

// Registro de qualidade (/api/quality/records) sobre o lote de inspeção, desde
// a migração 000418. O registro antigo (quality_records) foi copiado para
// lotes e é resolvido pelo vínculo legado; o registro novo é um lote de
// processo (ou final) aberto com o plano informado. As quantidades e o
// resultado viram a decisão de uso.

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/lote_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
)

const legadoRegistro = "QUALITY_RECORD"

func (uc *QualityUseCase) lotes() (*lote_uc.UseCase, error) {
	if uc.Lotes == nil {
		return nil, errorsuc.NewValidationError("o lote de inspeção não está configurado")
	}
	return uc.Lotes, nil
}

func registroDoLote(l *dom.Lote) *response.QualityRecordResponse {
	out := &response.QualityRecordResponse{
		ID: l.ID, ProductionOrderID: l.ProductionOrderID, Lot: l.Lote, ItemCode: l.ItemCode,
		InspectedQty: l.Quantidade.InexactFloat64(), Result: "PENDENTE", Notes: l.Observacoes,
		InspectedAt: l.UpdatedAt, CreatedAt: l.CreatedAt,
	}
	if l.LegacyKind != nil && *l.LegacyKind == legadoRegistro && l.LegacyID != nil {
		out.ID = *l.LegacyID
	}
	if l.InspectionPlanID != nil {
		out.PlanID = *l.InspectionPlanID
	}
	if l.CreatedBy != nil {
		out.CreatedBy = *l.CreatedBy
	}
	if d := l.Decisao; d != nil {
		out.InspectedAt = d.DecididoEm
		out.ApprovedQty = d.Aproveitada().InexactFloat64()
		out.RejectedQty = d.Retrabalho.Add(d.Rejeitada).Add(d.Sucata).InexactFloat64()
		switch d.Tipo {
		case dom.DecisaoAceito:
			out.Result = "APROVADO"
		case dom.DecisaoRejeitado:
			out.Result = "REJEITADO"
		default:
			out.Result = "CONDICIONAL"
		}
	} else if l.Status == dom.Cancelado {
		out.Result = "CANCELADO"
	}
	return out
}

func (uc *QualityUseCase) CreateRecord(ctx context.Context, dto request.CreateQualityRecordDTO) (*response.QualityRecordResponse, error) {
	if !dto.InspectedQty.IsPositive() {
		return nil, errorsuc.NewValidationError("a quantidade inspecionada deve ser maior que zero")
	}
	if dto.PlanID <= 0 {
		return nil, errorsuc.NewValidationError("informe o plano de inspeção")
	}
	resultado := strings.ToUpper(strings.TrimSpace(dto.Result))
	if resultado == "" {
		resultado = "PENDENTE"
	}
	switch resultado {
	case "PENDENTE", "APROVADO", "REJEITADO", "CONDICIONAL":
	default:
		return nil, errorsuc.NewValidationError("resultado inválido (PENDENTE, APROVADO, REJEITADO ou CONDICIONAL)")
	}
	if resultado != "PENDENTE" && !dto.ApprovedQty.Add(dto.RejectedQty).Equal(dto.InspectedQty) {
		return nil, errorsuc.NewValidationError("a quantidade aprovada mais a rejeitada deve somar a inspecionada")
	}
	lu, err := uc.lotes()
	if err != nil {
		return nil, err
	}
	plano := dto.PlanID
	r, err := lu.Abrir(ctx, lote_uc.AberturaDTO{
		Origem: string(dom.OrigemProcesso), ItemCode: dto.ItemCode, Lote: dto.Lot, Quantidade: dto.InspectedQty,
		ProductionOrderID: dto.ProductionOrderID, InspectionPlanID: &plano, Observacoes: dto.Notes,
		SemAmostragem: true, SemCaracteristicas: len(dto.Measurements) == 0,
	})
	if err != nil {
		return nil, err
	}
	if len(dto.Measurements) > 0 {
		indice := map[int64]int{}
		porOrigem := map[int64]int64{}
		for _, c := range r.Caracteristicas {
			if c.OrigemCarac != nil {
				porOrigem[*c.OrigemCarac] = c.ID
			}
		}
		rs := make([]lote_uc.ResultadoDTO, 0, len(dto.Measurements))
		for _, m := range dto.Measurements {
			id, ok := porOrigem[m.CharacteristicID]
			if !ok {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("a característica %d não é do plano %d", m.CharacteristicID, dto.PlanID))
			}
			indice[id]++
			v, conforme := m.MeasuredValue, m.IsConformant
			rs = append(rs, lote_uc.ResultadoDTO{CaracteristicaID: id, AmostraIndice: indice[id], Valor: &v, Conforme: &conforme})
		}
		if r, err = lu.GravarResultados(ctx, r.ID, rs); err != nil {
			return nil, err
		}
	}
	if resultado != "PENDENTE" {
		dec := lote_uc.DecisaoDTO{Aceita: dto.ApprovedQty, Rejeitada: dto.RejectedQty, Motivo: dto.Notes}
		if resultado == "CONDICIONAL" {
			dec.Aceita, dec.Restrita = decimal.Zero, dto.ApprovedQty
		}
		if (resultado != "APROVADO" || dto.RejectedQty.IsPositive()) && dec.Motivo == nil {
			m := "registro de qualidade " + strings.ToLower(resultado)
			dec.Motivo = &m
		}
		if resultado == "APROVADO" && !r.Avaliacao.AceitoPelaAmostragem && len(r.Resultados) > 0 {
			c, m := "CONCESSAO", "aprovado no registro de qualidade com medição fora"
			dec.Tratamento = &c
			if dec.Motivo == nil {
				dec.Motivo = &m
			}
		}
		if r, err = lu.Decidir(ctx, r.ID, dec); err != nil {
			return nil, err
		}
	}
	return registroDoLote(r.Lote), nil
}

func (uc *QualityUseCase) GetRecord(ctx context.Context, id int64) (*response.QualityRecordResponse, error) {
	lu, err := uc.lotes()
	if err != nil {
		return nil, err
	}
	l, err := lu.ObterPorLegado(ctx, legadoRegistro, id)
	if err != nil {
		if !errorsuc.EhNaoEncontrado(err) {
			return nil, err
		}
		r, errNovo := lu.Obter(ctx, id)
		if errNovo != nil || r.InspectionPlanID == nil || r.Origem == dom.OrigemRecebimento {
			return nil, errorsuc.NewNotFoundError("registro de qualidade não encontrado")
		}
		l = r.Lote
	}
	return registroDoLote(l), nil
}

func (uc *QualityUseCase) listarRegistros(ctx context.Context, f dom.Filtro) ([]*response.QualityRecordResponse, error) {
	lu, err := uc.lotes()
	if err != nil {
		return nil, err
	}
	ls, err := lu.Listar(ctx, f)
	if err != nil {
		return nil, err
	}
	out := []*response.QualityRecordResponse{}
	for i := range ls {
		if ls[i].InspectionPlanID == nil || ls[i].Origem == dom.OrigemRecebimento {
			continue
		}
		out = append(out, registroDoLote(&ls[i]))
	}
	return out, nil
}

func (uc *QualityUseCase) ListRecordsByOrder(ctx context.Context, orderID int64) ([]*response.QualityRecordResponse, error) {
	return uc.listarRegistros(ctx, dom.Filtro{ProductionOrderID: orderID, Limite: 500})
}

func (uc *QualityUseCase) ListRecordsByItem(ctx context.Context, itemCode int64) ([]*response.QualityRecordResponse, error) {
	return uc.listarRegistros(ctx, dom.Filtro{ItemCode: itemCode, Limite: 100})
}

// loteDaNC resolve o quality_record_id que a tela manda: registro antigo
// continua apontado; registro novo é o lote.
func (uc *QualityUseCase) loteDaNC(ctx context.Context, registro *int64) (legado *int64, lote *int64) {
	if registro == nil || uc.Lotes == nil {
		return registro, nil
	}
	if l, err := uc.Lotes.ObterPorLegado(ctx, legadoRegistro, *registro); err == nil {
		return registro, &l.ID
	}
	if r, err := uc.Lotes.Obter(ctx, *registro); err == nil {
		return nil, &r.ID
	}
	return registro, nil
}

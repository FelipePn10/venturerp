package structure

import (
	"context"
	"encoding/json"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

// Ações registradas no histórico da estrutura.
const (
	histInclusao  = "INCLUSAO"
	histAlteracao = "ALTERACAO"
	histExclusao  = "EXCLUSAO"
)

// atorDoContexto devolve o usuário autenticado, quando há um.
//
// O histórico é gravado aqui, e não por gatilho no banco, porque o gatilho não
// tem como saber quem está operando — e um histórico sem autor não responde à
// pergunta que motiva existir ("quem trocou a quantidade?").
func atorDoContexto(ctx context.Context) *uuid.UUID {
	user, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok || user == nil {
		return nil
	}
	id, err := uuid.Parse(user.ID)
	if err != nil {
		return nil
	}
	return &id
}

// instantaneo serializa o estado do componente para o histórico.
func instantaneo(s *entity.ItemStructure) []byte {
	if s == nil {
		return nil
	}
	dados := map[string]any{
		"quantidade":             s.Quantity,
		"formula_quantidade":     s.QuantityFormula,
		"arredondamento":         s.QuantityRounding,
		"casas_decimais":         s.QuantityScale,
		"unidade":                string(s.UnitOfMeasurement),
		"perda_percentual":       s.LossPercentage,
		"formula_perda":          s.LossFormula,
		"perda_setup":            s.SetupLoss,
		"tipo_perda_custo":       s.CostLossType,
		"perda_custo":            s.CostLoss,
		"situacao":               string(s.Health),
		"sequencia":              s.Sequence,
		"vigencia_inicio":        s.StartDate,
		"vigencia_fim":           s.EndDate,
		"coproduto":              s.IsCoproduct,
		"quantidade_por_ordem":   s.IsFixedQty,
		"grupo_alternativo":      s.SubstituteGroup,
		"prioridade_alternativo": s.SubstitutePriority,
		"almoxarifado":           s.WarehouseCode,
		"almoxarifado_linha":     s.LineWarehouseCode,
		"centro_custo":           s.CostCenterCode,
		"critico_mps":            s.IsCriticalMPS,
		"gera_inspecao":          s.GeneratesInspection,
		"ativo":                  s.IsActive,
	}
	raw, err := json.Marshal(dados)
	if err != nil {
		return nil
	}
	return raw
}

// registrarHistorico grava uma linha do histórico. Uma falha aqui não pode
// derrubar a operação de negócio: o cadastro já foi gravado, e perder a trilha
// de auditoria é menos grave do que recusar um trabalho já concluído.
func (r *ItemStructureRepositorySQLC) registrarHistorico(
	ctx context.Context,
	acao string,
	antes, depois *entity.ItemStructure,
) {
	if r.pool == nil {
		return
	}
	ref := depois
	if ref == nil {
		ref = antes
	}
	if ref == nil {
		return
	}
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO item_structure_history
		    (structure_id, parent_code, child_code, action, changed_by, before_state, after_state)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		ref.ID, ref.ParentCode, ref.ChildCode, acao, atorDoContexto(ctx),
		instantaneo(antes), instantaneo(depois))
}

// componenteAtual lê o componente como ele está agora, para o histórico
// registrar o "antes". Devolve nil quando não encontra — o histórico então
// grava só o "depois", que ainda é melhor do que não registrar nada.
func (r *ItemStructureRepositorySQLC) componenteAtual(
	ctx context.Context,
	parentCode, childCode int64,
	parentMask *string,
) *entity.ItemStructure {
	if r.pool == nil {
		return nil
	}
	filhos, err := r.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil
	}
	for _, f := range filhos {
		if f.ChildCode != childCode {
			continue
		}
		if parentMask == nil && f.ParentMask == nil {
			return f
		}
		if parentMask != nil && f.ParentMask != nil && *parentMask == *f.ParentMask {
			return f
		}
	}
	return nil
}

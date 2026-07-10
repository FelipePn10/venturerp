package configurator_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/configurator/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/formula"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

func validRuleOp(op string) string {
	switch op {
	case entity.OpEqual, entity.OpDifferent, entity.OpGreater, entity.OpLess, entity.OpBelongs, entity.OpNotBelongs:
		return op
	}
	return entity.OpEqual
}

func int4Ptr(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

func int4ToPtr(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

// resolveAnswerCodes maps each answer to the chosen variable's CODE (the canonical
// value the rule engines compare against). Free/numeric/option answers use the
// literal value.
func (uc *ConfiguratorUseCase) resolveAnswerCodes(ctx context.Context, answers []request.CfgMaskAnswerInput) map[int64]string {
	out := map[int64]string{}
	for _, a := range answers {
		if a.VariableID != nil {
			if v, err := uc.Q.GetCfgVariable(ctx, *a.VariableID); err == nil {
				out[a.CharacteristicID] = v.Code
				continue
			}
		}
		out[a.CharacteristicID] = a.Value
	}
	return out
}

// ─── Regras de Variáveis Equivalentes ─────────────────────────────────────────

func (uc *ConfiguratorUseCase) CreateEquivalentRule(ctx context.Context, dto request.CfgEquivalentRuleDTO) (*response.CfgEquivalentRuleResponse, error) {
	if dto.ParentItemCode <= 0 || dto.ChildItemCode <= 0 {
		return nil, fmt.Errorf("item pai e item filho são obrigatórios")
	}
	if dto.ParentCharacteristicID <= 0 || dto.ChildCharacteristicID <= 0 {
		return nil, fmt.Errorf("característica do pai e do filho são obrigatórias")
	}
	row, err := uc.Q.CreateCfgEquivalentRule(ctx, equivParams(dto, 0))
	if err != nil {
		return nil, fmt.Errorf("criando regra de variáveis equivalentes: %w", err)
	}
	return equivToResponse(row), nil
}

func (uc *ConfiguratorUseCase) UpdateEquivalentRule(ctx context.Context, dto request.CfgEquivalentRuleDTO) (*response.CfgEquivalentRuleResponse, error) {
	row, err := uc.Q.UpdateCfgEquivalentRule(ctx, equivParams(dto, dto.ID))
	if err != nil {
		return nil, fmt.Errorf("atualizando regra de variáveis equivalentes: %w", err)
	}
	return equivToResponse(row), nil
}

func (uc *ConfiguratorUseCase) GetEquivalentRule(ctx context.Context, id int64) (*response.CfgEquivalentRuleResponse, error) {
	row, err := uc.Q.GetCfgEquivalentRule(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("regra não encontrada: %w", err)
	}
	return equivToResponse(row), nil
}

func (uc *ConfiguratorUseCase) ListEquivalentRulesByParent(ctx context.Context, parentItemCode int64, onlyActive bool) ([]*response.CfgEquivalentRuleResponse, error) {
	rows, err := uc.Q.ListCfgEquivalentRulesByParent(ctx, parentItemCode, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgEquivalentRuleResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, equivToResponse(r))
	}
	return out, nil
}

func (uc *ConfiguratorUseCase) DeactivateEquivalentRule(ctx context.Context, id int64) error {
	return uc.Q.DeactivateCfgEquivalentRule(ctx, id)
}

// ApplyEquivalent maps the parent item's configuration to the equivalent child
// answers: for each active rule whose parent condition matches the answers, emit
// the child characteristic → child variable assignment.
func (uc *ConfiguratorUseCase) ApplyEquivalent(ctx context.Context, dto request.CfgApplyEquivalentDTO) (*response.CfgAppliedEquivalentResponse, error) {
	if dto.ParentItemCode <= 0 {
		return nil, fmt.Errorf("parent_item_code é obrigatório")
	}
	answers := uc.resolveAnswerCodes(ctx, dto.Answers)
	rules, err := uc.Q.ListCfgEquivalentRulesByParent(ctx, dto.ParentItemCode, true)
	if err != nil {
		return nil, fmt.Errorf("carregando regras: %w", err)
	}
	out := &response.CfgAppliedEquivalentResponse{ParentItemCode: dto.ParentItemCode}
	for _, r := range rules {
		actual, ok := answers[r.ParentCharacteristicID]
		if !ok {
			continue
		}
		if !entity.MatchOperator(r.ParentOperator, actual, pgutil.FromPgText(r.ParentVariableCode)) {
			continue
		}
		out.ChildAnswers = append(out.ChildAnswers, response.CfgChildAnswer{
			RuleID:           r.ID,
			ChildItemCode:    r.ChildItemCode,
			ChildSeq:         int4ToPtr(r.ChildSeq),
			CharacteristicID: r.ChildCharacteristicID,
			Operator:         r.ChildOperator,
			VariableID:       pgutil.FromPgInt8Ptr(r.ChildVariableID),
			VariableCode:     pgutil.FromPgText(r.ChildVariableCode),
			Formula:          pgutil.FromPgText(r.Formula),
		})
	}
	return out, nil
}

// ─── Regras de Itens Configurados ─────────────────────────────────────────────

func (uc *ConfiguratorUseCase) CreateItemRule(ctx context.Context, dto request.CfgItemRuleDTO) (*response.CfgItemRuleResponse, error) {
	if dto.ItemCode <= 0 || dto.TargetTable == "" || dto.TargetField == "" {
		return nil, fmt.Errorf("item, tabela e campo são obrigatórios")
	}
	row, err := uc.Q.CreateCfgItemRule(ctx, itemRuleParams(dto, 0))
	if err != nil {
		return nil, fmt.Errorf("criando regra de item configurado: %w", err)
	}
	if err := uc.replaceConditions(ctx, row.ID, dto.Conditions); err != nil {
		return nil, err
	}
	return uc.itemRuleView(ctx, row)
}

func (uc *ConfiguratorUseCase) UpdateItemRule(ctx context.Context, dto request.CfgItemRuleDTO) (*response.CfgItemRuleResponse, error) {
	row, err := uc.Q.UpdateCfgItemRule(ctx, itemRuleParams(dto, dto.ID))
	if err != nil {
		return nil, fmt.Errorf("atualizando regra de item configurado: %w", err)
	}
	if err := uc.replaceConditions(ctx, dto.ID, dto.Conditions); err != nil {
		return nil, err
	}
	return uc.itemRuleView(ctx, row)
}

func (uc *ConfiguratorUseCase) GetItemRule(ctx context.Context, id int64) (*response.CfgItemRuleResponse, error) {
	row, err := uc.Q.GetCfgItemRule(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("regra não encontrada: %w", err)
	}
	return uc.itemRuleView(ctx, row)
}

func (uc *ConfiguratorUseCase) ListItemRulesByItem(ctx context.Context, itemCode int64, onlyActive bool) ([]*response.CfgItemRuleResponse, error) {
	rows, err := uc.Q.ListCfgItemRulesByItem(ctx, itemCode, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgItemRuleResponse, 0, len(rows))
	for _, r := range rows {
		v, err := uc.itemRuleView(ctx, r)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (uc *ConfiguratorUseCase) DeleteItemRule(ctx context.Context, id int64) error {
	return uc.Q.DeleteCfgItemRule(ctx, id)
}

// EvaluateItemRules returns the field assignments for the active rules whose
// conditions all match the item's configuration.
func (uc *ConfiguratorUseCase) EvaluateItemRules(ctx context.Context, dto request.CfgEvaluateItemRulesDTO) (*response.CfgEvaluatedRulesResponse, error) {
	if dto.ItemCode <= 0 {
		return nil, fmt.Errorf("item_code é obrigatório")
	}
	answers := uc.resolveAnswerCodes(ctx, dto.Answers)
	rules, err := uc.Q.ListCfgItemRulesByItem(ctx, dto.ItemCode, true)
	if err != nil {
		return nil, fmt.Errorf("carregando regras: %w", err)
	}
	// vars for Botão F (formula) — numeric answers keyed by normalized code.
	var vars map[string]float64
	out := &response.CfgEvaluatedRulesResponse{ItemCode: dto.ItemCode}
	for _, r := range rules {
		conds, err := uc.Q.ListCfgItemRuleConditions(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		if !allConditionsMatch(conds, answers) {
			continue
		}
		content := pgutil.FromPgText(r.Content)
		// Botão F: when a formula is set, the content is the computed result.
		if expr := pgutil.FromPgText(r.Formula); expr != "" {
			if vars == nil {
				vars = uc.formulaVars(ctx, dto.Answers)
			}
			if v, ok := formula.EvaluateSafe(expr, vars); ok {
				content = formatNum(v)
			}
		}
		out.Assignments = append(out.Assignments, response.CfgFieldAssignment{
			RuleID:      r.ID,
			TargetTable: r.TargetTable,
			TargetField: r.TargetField,
			Content:     content,
			Formula:     pgutil.FromPgText(r.Formula),
			Description: pgutil.FromPgText(r.Description),
		})
	}
	return out, nil
}

// allConditionsMatch reports whether every condition holds (AND). An empty
// condition set never fires (a rule must have at least one condition).
func allConditionsMatch(conds []sqlc.DBCfgItemRuleCondition, answers map[int64]string) bool {
	if len(conds) == 0 {
		return false
	}
	for _, c := range conds {
		actual, ok := answers[c.CharacteristicID]
		if !ok {
			return false
		}
		if !entity.MatchOperator(c.Operator, actual, pgutil.FromPgText(c.VariableCode)) {
			return false
		}
	}
	return true
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func (uc *ConfiguratorUseCase) replaceConditions(ctx context.Context, ruleID int64, conds []request.CfgItemRuleConditionDTO) error {
	if err := uc.Q.DeleteCfgItemRuleConditionsByRule(ctx, ruleID); err != nil {
		return fmt.Errorf("limpando condições: %w", err)
	}
	for i, c := range conds {
		if _, err := uc.Q.AddCfgItemRuleCondition(ctx, ruleID, c.CharacteristicID,
			validRuleOp(c.Operator), pgutil.ToPgInt8Ptr(c.VariableID), int32(i+1)); err != nil {
			return fmt.Errorf("gravando condição: %w", err)
		}
	}
	return nil
}

func (uc *ConfiguratorUseCase) itemRuleView(ctx context.Context, r sqlc.DBCfgItemRule) (*response.CfgItemRuleResponse, error) {
	conds, err := uc.Q.ListCfgItemRuleConditions(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	out := &response.CfgItemRuleResponse{
		ID: r.ID, ItemCode: r.ItemCode, TargetTable: r.TargetTable, TargetField: r.TargetField,
		Content: pgutil.FromPgText(r.Content), Formula: pgutil.FromPgText(r.Formula),
		Description: pgutil.FromPgText(r.Description), Situation: r.Situation,
	}
	for _, c := range conds {
		out.Conditions = append(out.Conditions, response.CfgItemRuleConditionResponse{
			ID: c.ID, CharacteristicID: c.CharacteristicID, Operator: c.Operator,
			VariableID: pgutil.FromPgInt8Ptr(c.VariableID), VariableCode: pgutil.FromPgText(c.VariableCode),
		})
	}
	return out, nil
}

func equivParams(dto request.CfgEquivalentRuleDTO, id int64) sqlc.CfgEquivalentRuleParams {
	return sqlc.CfgEquivalentRuleParams{
		ID:                     id,
		ParentItemCode:         dto.ParentItemCode,
		ParentUom:              textOrNull(dto.ParentUOM),
		ChildItemCode:          dto.ChildItemCode,
		ChildSeq:               int4Ptr(dto.ChildSeq),
		ParentCharacteristicID: dto.ParentCharacteristicID,
		ParentOperator:         validRuleOp(dto.ParentOperator),
		ParentVariableID:       pgutil.ToPgInt8Ptr(dto.ParentVariableID),
		ChildCharacteristicID:  dto.ChildCharacteristicID,
		ChildOperator:          validRuleOp(dto.ChildOperator),
		ChildVariableID:        pgutil.ToPgInt8Ptr(dto.ChildVariableID),
		Formula:                textOrNull(dto.Formula),
		CreatedBy:              pgutil.ToPgUUID(dto.CreatedBy),
	}
}

func itemRuleParams(dto request.CfgItemRuleDTO, id int64) sqlc.CfgItemRuleParams {
	sit := dto.Situation
	if sit != "INACTIVE" {
		sit = "ACTIVE"
	}
	return sqlc.CfgItemRuleParams{
		ID:          id,
		ItemCode:    dto.ItemCode,
		TargetTable: dto.TargetTable,
		TargetField: dto.TargetField,
		Content:     textOrNull(dto.Content),
		Formula:     textOrNull(dto.Formula),
		Description: textOrNull(dto.Description),
		Situation:   sit,
		CreatedBy:   pgutil.ToPgUUID(dto.CreatedBy),
	}
}

func equivToResponse(r sqlc.DBCfgEquivalentRule) *response.CfgEquivalentRuleResponse {
	return &response.CfgEquivalentRuleResponse{
		ID:                     r.ID,
		ParentItemCode:         r.ParentItemCode,
		ParentUOM:              pgutil.FromPgText(r.ParentUom),
		ChildItemCode:          r.ChildItemCode,
		ChildSeq:               int4ToPtr(r.ChildSeq),
		ParentCharacteristicID: r.ParentCharacteristicID,
		ParentOperator:         r.ParentOperator,
		ParentVariableID:       pgutil.FromPgInt8Ptr(r.ParentVariableID),
		ParentVariableCode:     pgutil.FromPgText(r.ParentVariableCode),
		ChildCharacteristicID:  r.ChildCharacteristicID,
		ChildOperator:          r.ChildOperator,
		ChildVariableID:        pgutil.FromPgInt8Ptr(r.ChildVariableID),
		ChildVariableCode:      pgutil.FromPgText(r.ChildVariableCode),
		Formula:                pgutil.FromPgText(r.Formula),
		IsActive:               r.IsActive,
	}
}

// ─── Botão Itens do Tipo Recebimento ──────────────────────────────────────────

func (uc *ConfiguratorUseCase) AddReceivingItem(ctx context.Context, charID int64, dto request.CfgReceivingItemDTO) (*response.CfgReceivingItemResponse, error) {
	char, err := uc.Q.GetCfgCharacteristic(ctx, charID)
	if err != nil {
		return nil, fmt.Errorf("característica %d não encontrada", charID)
	}
	if char.ReceivingType == entity.RecebNenhum {
		return nil, fmt.Errorf("a característica não possui tipo de recebimento")
	}
	rt := dto.ReceivingType
	if rt != entity.RecebRecebimento && rt != entity.RecebVinculo {
		return nil, fmt.Errorf("tipo de recebimento deve ser RECEBIMENTO ou VINCULO")
	}
	row, err := uc.Q.AddCfgCharReceivingItem(ctx, charID, pgutil.ToPgInt8Ptr(dto.VariableID), rt,
		pgutil.ToPgInt8Ptr(dto.ItemCode), pgutil.ToPgInt8Ptr(dto.ClassificationCode))
	if err != nil {
		return nil, fmt.Errorf("gravando item de recebimento: %w", err)
	}
	return receivingItemToResponse(row), nil
}

func (uc *ConfiguratorUseCase) ListReceivingItems(ctx context.Context, charID int64) ([]*response.CfgReceivingItemResponse, error) {
	rows, err := uc.Q.ListCfgCharReceivingItems(ctx, charID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgReceivingItemResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, receivingItemToResponse(r))
	}
	return out, nil
}

func (uc *ConfiguratorUseCase) DeleteReceivingItem(ctx context.Context, id int64) error {
	return uc.Q.DeleteCfgCharReceivingItem(ctx, id)
}

func receivingItemToResponse(r sqlc.DBCfgCharReceivingItem) *response.CfgReceivingItemResponse {
	return &response.CfgReceivingItemResponse{
		ID: r.ID, CharacteristicID: r.CharacteristicID, VariableID: pgutil.FromPgInt8Ptr(r.VariableID),
		VariableCode: pgutil.FromPgText(r.VariableCode), ReceivingType: r.ReceivingType,
		ItemCode: pgutil.FromPgInt8Ptr(r.ItemCode), ClassificationCode: pgutil.FromPgInt8Ptr(r.ClassificationCode),
	}
}

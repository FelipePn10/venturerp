//go:build integration

package configurator_uc_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/configurator_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc"
	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	restrictionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/restriction"
	structurequery "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// End-to-end configurator flow against a real Postgres: conjunto → variáveis →
// características (ESCOLHA + INF_NUMERICA) → características do item → geração de
// máscara (com default e validação numérica) → guarda de edição pós-máscara.
func TestIntegration_Configurator_Flow(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := configurator_uc.New(q)
	ctx := context.Background()
	uid := uuid.New()

	code := testutil.UniqueCode()
	itemCode := testutil.UniqueCode()

	// item_masks.created_by references users — seed one for the persist step.
	uid = uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users (id, name, email, password, created_at, updated_at, role)
		VALUES ($1,'cfg-test',$2,'x',NOW(),NOW(),'ADMIN')`, uid, fmt.Sprintf("cfg-%d@test.local", code))

	// cleanup
	defer func() {
		testutil.Exec(t, pool, "DELETE FROM item_masks WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_item_characteristics WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("CFG%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_sets WHERE description = $1", fmt.Sprintf("COR-%d", code))
		testutil.Exec(t, pool, "DELETE FROM users WHERE id = $1", uid)
	}()

	// Conjunto + variáveis
	set, err := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: fmt.Sprintf("COR-%d", code), CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateSet: %v", err)
	}
	azul, err := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{
		SetID: set.ID, Code: "AZ", Description: "Azul", MaskComposition: "AZUL", CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateVariable azul: %v", err)
	}
	verde, err := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{
		SetID: set.ID, Code: "VE", Description: "Verde", MaskComposition: "VERDE", CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateVariable verde: %v", err)
	}

	// Característica ESCOLHA (default = azul)
	cor, err := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
		Code: fmt.Sprintf("CFG%d_COR", code), Description: "Cor da tampa", Type: "ESCOLHA",
		SetID: &set.ID, DefaultVariableID: &azul.ID, CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateCharacteristic cor: %v", err)
	}
	// Característica INF_NUMERICA (1..100 múltiplo de 2)
	min, max, mult := 1.0, 100.0, 2.0
	larg, err := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
		Code: fmt.Sprintf("CFG%d_LARG", code), Description: "Largura", Type: "INF_NUMERICA",
		NumMin: &min, NumMax: &max, NumMultiple: &mult, CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateCharacteristic larg: %v", err)
	}

	// Características do item (seq 10 cor, 20 largura)
	icCor, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: cor.ID, Sequence: 10, DefaultVariableID: &azul.ID})
	if err != nil {
		t.Fatalf("AddItemCharacteristic cor: %v", err)
	}
	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: larg.ID, Sequence: 20}); err != nil {
		t.Fatalf("AddItemCharacteristic larg: %v", err)
	}

	// Máscara com respostas explícitas: verde + 50 → "VERDE#50"
	m, err := uc.GenerateMask(ctx, request.CfgGenerateMaskDTO{
		ItemCode: itemCode,
		Answers: []request.CfgMaskAnswerInput{
			{CharacteristicID: cor.ID, VariableID: &verde.ID},
			{CharacteristicID: larg.ID, Value: "50"},
		},
	})
	if err != nil {
		t.Fatalf("GenerateMask: %v", err)
	}
	if m.Mask != "VERDE#50" {
		t.Fatalf("mask = %q, want VERDE#50", m.Mask)
	}

	// Sem responder a cor → usa o default do item (azul): "AZUL#40"
	m2, err := uc.GenerateMask(ctx, request.CfgGenerateMaskDTO{
		ItemCode: itemCode,
		Answers:  []request.CfgMaskAnswerInput{{CharacteristicID: larg.ID, Value: "40"}},
	})
	if err != nil {
		t.Fatalf("GenerateMask default: %v", err)
	}
	if m2.Mask != "AZUL#40" {
		t.Fatalf("default mask = %q, want AZUL#40", m2.Mask)
	}

	// Validação numérica: 51 não é múltiplo de 2 → erro
	if _, err := uc.GenerateMask(ctx, request.CfgGenerateMaskDTO{
		ItemCode: itemCode,
		Answers:  []request.CfgMaskAnswerInput{{CharacteristicID: larg.ID, Value: "51"}},
	}); err == nil {
		t.Fatal("esperado erro para largura=51 (não múltiplo de 2)")
	}

	// Persistir a máscara e então verificar a guarda de edição
	if _, err := uc.GenerateMask(ctx, request.CfgGenerateMaskDTO{
		ItemCode: itemCode, Persist: true, CreatedBy: uid,
		Answers: []request.CfgMaskAnswerInput{
			{CharacteristicID: cor.ID, VariableID: &azul.ID},
			{CharacteristicID: larg.ID, Value: "20"},
		},
	}); err != nil {
		t.Fatalf("GenerateMask persist: %v", err)
	}
	// alterar a sequência agora deve falhar
	if _, err := uc.UpdateItemCharacteristic(ctx, request.UpdateCfgItemCharacteristicDTO{
		ID: icCor.ID, Sequence: 15}); err == nil {
		t.Fatal("esperado erro ao alterar sequência com máscara gerada")
	}
	// remover agora deve falhar
	if err := uc.RemoveItemCharacteristic(ctx, icCor.ID); err == nil {
		t.Fatal("esperado erro ao remover característica com máscara gerada")
	}
}

// Cartesian mask generation with a restriction/dependency filtering combinations:
// COR{AZ,VE} × TAMPA{RED,QUAD} = 4; a dependency "COR=AZ ⇒ TAMPA=RED" drops
// (AZ,QUAD), leaving 3 valid masks.
func TestIntegration_Configurator_CartesianWithRestriction(t *testing.T) {
	q, pool := testutil.Queries(t)
	oracle := &restriction_uc.EvaluateRestrictionsUseCase{Repo: restrictionrepo.NewRestrictionRepositorySQLC(q)}
	uc := configurator_uc.New(q).WithRestrictions(oracle)
	ctx := context.Background()
	uid := uuid.New()

	code := testutil.UniqueCode()
	itemCode := testutil.UniqueCode()

	testutil.Exec(t, pool, `INSERT INTO users (id, name, email, password, created_at, updated_at, role)
		VALUES ($1,'cfg-test',$2,'x',NOW(),NOW(),'ADMIN')`, uid, fmt.Sprintf("cart-%d@test.local", code))

	defer func() {
		testutil.Exec(t, pool, "DELETE FROM restrictions WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM item_masks WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_item_characteristics WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("CART%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_sets WHERE description LIKE $1", fmt.Sprintf("CART%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM users WHERE id = $1", uid)
	}()

	mkSet := func(desc string) int64 {
		s, err := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: desc, CreatedBy: uid})
		if err != nil {
			t.Fatalf("CreateSet %s: %v", desc, err)
		}
		return s.ID
	}
	mkVar := func(setID int64, vcode, mask string) int64 {
		v, err := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{
			SetID: setID, Code: vcode, Description: vcode, MaskComposition: mask, CreatedBy: uid})
		if err != nil {
			t.Fatalf("CreateVariable %s: %v", vcode, err)
		}
		return v.ID
	}
	mkChar := func(ccode string, setID int64) int64 {
		c, err := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
			Code: ccode, Description: ccode, Type: "ESCOLHA", SetID: &setID, CreatedBy: uid})
		if err != nil {
			t.Fatalf("CreateCharacteristic %s: %v", ccode, err)
		}
		return c.ID
	}

	corSet := mkSet(fmt.Sprintf("CART%d-COR", code))
	azID := mkVar(corSet, "AZ", "AZUL")
	veID := mkVar(corSet, "VE", "VERDE")
	formaSet := mkSet(fmt.Sprintf("CART%d-FORMA", code))
	mkVar(formaSet, "RED", "REDONDA")
	mkVar(formaSet, "QUAD", "QUADRADA")

	corChar := mkChar(fmt.Sprintf("CART%d_COR", code), corSet)
	tampaChar := mkChar(fmt.Sprintf("CART%d_TAMPA", code), formaSet)

	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: corChar, Sequence: 10}); err != nil {
		t.Fatalf("AddItemCharacteristic cor: %v", err)
	}
	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: tampaChar, Sequence: 20}); err != nil {
		t.Fatalf("AddItemCharacteristic tampa: %v", err)
	}

	// Dependency: COR=AZ ⇒ TAMPA=RED (answer_value = variable code).
	var rid int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO restrictions (situation, item_code, weight, created_by) VALUES ('ACTIVE',$1,16,$2) RETURNING id`,
		itemCode, uid).Scan(&rid); err != nil {
		t.Fatalf("insert restriction: %v", err)
	}
	testutil.Exec(t, pool, `INSERT INTO restriction_dominants (restriction_id, question_id, operator, condition_type, answer_value, sequence)
		VALUES ($1,$2,'EQUAL','AND','AZ',1)`, rid, corChar)
	testutil.Exec(t, pool, `INSERT INTO restriction_determinants (restriction_id, question_id, operator, answer_value)
		VALUES ($1,$2,'EQUAL','RED')`, rid, tampaChar)

	res, err := uc.GenerateMasks(ctx, request.CfgGenerateMasksDTO{
		ItemCode: itemCode,
		Restrict: []request.CfgMaskRestrictInput{{CharacteristicID: corChar, VariableIDs: []int64{azID, veID}}},
	})
	if err != nil {
		t.Fatalf("GenerateMasks: %v", err)
	}
	if res.TotalCombinations != 4 {
		t.Fatalf("total = %d, want 4", res.TotalCombinations)
	}
	if res.ValidCount != 3 {
		t.Fatalf("valid = %d, want 3", res.ValidCount)
	}
	got := make([]string, 0, len(res.Masks))
	for _, m := range res.Masks {
		got = append(got, m.Mask)
	}
	sort.Strings(got)
	want := []string{"AZUL#REDONDA", "VERDE#QUADRADA", "VERDE#REDONDA"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("masks = %v, want %v", got, want)
		}
	}
}

// Description flow: create description type → item description loads a grid line
// per item characteristic → render (Botão V) produces "Cor da tampa: AZUL".
func TestIntegration_Configurator_ItemDescription(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := configurator_uc.New(q)
	ctx := context.Background()
	uid := uuid.New()

	code := testutil.UniqueCode()
	itemCode := testutil.UniqueCode()

	defer func() {
		testutil.Exec(t, pool, "DELETE FROM cfg_item_descriptions WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_item_characteristics WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("DSC%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_sets WHERE description = $1", fmt.Sprintf("DSC%d-COR", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_description_types WHERE code = $1", fmt.Sprintf("DSC%d", code))
	}()

	dt, err := uc.CreateDescriptionType(ctx, request.CfgDescriptionTypeDTO{
		Code: fmt.Sprintf("DSC%d", code), Description: "Rótulo NF", Kind: "RELATORIO", CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateDescriptionType: %v", err)
	}

	set, _ := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: fmt.Sprintf("DSC%d-COR", code), CreatedBy: uid})
	azul, _ := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{
		SetID: set.ID, Code: "AZ", Description: "Azul", MaskComposition: "AZUL", CreatedBy: uid})
	cor, err := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
		Code: fmt.Sprintf("DSC%d_COR", code), Description: "Cor da tampa", Type: "ESCOLHA",
		SetID: &set.ID, Mask: "#COR#", CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateCharacteristic: %v", err)
	}
	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: cor.ID, Sequence: 10}); err != nil {
		t.Fatalf("AddItemCharacteristic: %v", err)
	}

	desc, err := uc.CreateItemDescription(ctx, request.CreateCfgItemDescriptionDTO{
		ItemCode: itemCode, DescriptionTypeID: dt.ID, CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateItemDescription: %v", err)
	}
	if len(desc.Lines) != 1 {
		t.Fatalf("lines = %d, want 1 (uma por característica)", len(desc.Lines))
	}

	// configure the line: mask description (DESCRICAO) + ": " + answer
	if _, err := uc.UpdateItemDescriptionLines(ctx, desc.ID, request.UpdateCfgItemDescriptionLinesDTO{
		Lines: []request.CfgItemDescriptionLineDTO{{
			ID: desc.Lines[0].ID, OrderIndex: 1, ShowCharacteristic: true, ShowMask: true,
			DescType: "DESCRICAO", Text: ": ", LineBreak: false,
		}},
	}); err != nil {
		t.Fatalf("UpdateItemDescriptionLines: %v", err)
	}

	rendered, err := uc.RenderItemDescription(ctx, desc.ID, request.CfgRenderDescriptionDTO{
		Answers: []request.CfgMaskAnswerInput{{CharacteristicID: cor.ID, VariableID: &azul.ID}},
	})
	if err != nil {
		t.Fatalf("RenderItemDescription: %v", err)
	}
	if rendered.Text != "Cor da tampa: AZUL" {
		t.Fatalf("render = %q, want %q", rendered.Text, "Cor da tampa: AZUL")
	}
}

// Regras de Variáveis Equivalentes: pai COR=AZ ⇒ filho TAMPA=RED. Aplicar a
// configuração do pai retorna a resposta equivalente do filho.
func TestIntegration_Configurator_EquivalentRule(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := configurator_uc.New(q)
	ctx := context.Background()
	uid := uuid.New()

	code := testutil.UniqueCode()
	parentItem := testutil.UniqueCode()
	childItem := testutil.UniqueCode()

	defer func() {
		testutil.Exec(t, pool, "DELETE FROM cfg_equivalent_rules WHERE parent_item_code = $1", parentItem)
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("EQV%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_sets WHERE description LIKE $1", fmt.Sprintf("EQV%d%%", code))
	}()

	corSet, _ := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: fmt.Sprintf("EQV%d-COR", code), CreatedBy: uid})
	az, _ := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{SetID: corSet.ID, Code: "AZ", Description: "Azul", MaskComposition: "AZUL", CreatedBy: uid})
	formaSet, _ := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: fmt.Sprintf("EQV%d-FORMA", code), CreatedBy: uid})
	red, _ := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{SetID: formaSet.ID, Code: "RED", Description: "Redonda", MaskComposition: "REDONDA", CreatedBy: uid})

	cor, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{Code: fmt.Sprintf("EQV%d_COR", code), Description: "Cor", Type: "ESCOLHA", SetID: &corSet.ID, CreatedBy: uid})
	tampa, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{Code: fmt.Sprintf("EQV%d_TAMPA", code), Description: "Tampa", Type: "ESCOLHA", SetID: &formaSet.ID, CreatedBy: uid})

	if _, err := uc.CreateEquivalentRule(ctx, request.CfgEquivalentRuleDTO{
		ParentItemCode: parentItem, ChildItemCode: childItem,
		ParentCharacteristicID: cor.ID, ParentOperator: "EQUAL", ParentVariableID: &az.ID,
		ChildCharacteristicID: tampa.ID, ChildOperator: "EQUAL", ChildVariableID: &red.ID, CreatedBy: uid,
	}); err != nil {
		t.Fatalf("CreateEquivalentRule: %v", err)
	}

	applied, err := uc.ApplyEquivalent(ctx, request.CfgApplyEquivalentDTO{
		ParentItemCode: parentItem,
		Answers:        []request.CfgMaskAnswerInput{{CharacteristicID: cor.ID, VariableID: &az.ID}},
	})
	if err != nil {
		t.Fatalf("ApplyEquivalent: %v", err)
	}
	if len(applied.ChildAnswers) != 1 {
		t.Fatalf("child answers = %d, want 1", len(applied.ChildAnswers))
	}
	ca := applied.ChildAnswers[0]
	if ca.ChildItemCode != childItem || ca.CharacteristicID != tampa.ID || ca.VariableCode != "RED" {
		t.Fatalf("child answer = %+v, want item %d / tampa %d / RED", ca, childItem, tampa.ID)
	}
}

// Regras de Itens Configurados: quando Opção=SIM, definir engineering.loss_pct=66.
func TestIntegration_Configurator_ItemRule(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := configurator_uc.New(q)
	ctx := context.Background()
	uid := uuid.New()

	code := testutil.UniqueCode()
	item := testutil.UniqueCode()

	defer func() {
		testutil.Exec(t, pool, "DELETE FROM cfg_item_rules WHERE item_code = $1", item)
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("IRL%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_sets WHERE description LIKE $1", fmt.Sprintf("IRL%d%%", code))
	}()

	set, _ := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: fmt.Sprintf("IRL%d-OPT", code), CreatedBy: uid})
	sim, _ := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{SetID: set.ID, Code: "SIM", Description: "Sim", MaskComposition: "SIM", CreatedBy: uid})
	nao, _ := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{SetID: set.ID, Code: "NAO", Description: "Nao", MaskComposition: "NAO", CreatedBy: uid})
	opt, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{Code: fmt.Sprintf("IRL%d_OPT", code), Description: "Opção", Type: "ESCOLHA", SetID: &set.ID, CreatedBy: uid})

	rule, err := uc.CreateItemRule(ctx, request.CfgItemRuleDTO{
		ItemCode: item, TargetTable: "ENGENHARIA", TargetField: "loss_percentage", Content: "66",
		Description: "Perda 66% quando opção sim", Situation: "ACTIVE",
		Conditions: []request.CfgItemRuleConditionDTO{{CharacteristicID: opt.ID, Operator: "EQUAL", VariableID: &sim.ID}},
	})
	if err != nil {
		t.Fatalf("CreateItemRule: %v", err)
	}
	if len(rule.Conditions) != 1 {
		t.Fatalf("conditions = %d, want 1", len(rule.Conditions))
	}

	// answer SIM → rule fires
	ev, err := uc.EvaluateItemRules(ctx, request.CfgEvaluateItemRulesDTO{
		ItemCode: item, Answers: []request.CfgMaskAnswerInput{{CharacteristicID: opt.ID, VariableID: &sim.ID}},
	})
	if err != nil {
		t.Fatalf("EvaluateItemRules: %v", err)
	}
	if len(ev.Assignments) != 1 || ev.Assignments[0].TargetField != "loss_percentage" || ev.Assignments[0].Content != "66" {
		t.Fatalf("assignments = %+v, want engineering.loss_percentage=66", ev.Assignments)
	}

	// answer NAO → rule does not fire
	ev2, _ := uc.EvaluateItemRules(ctx, request.CfgEvaluateItemRulesDTO{
		ItemCode: item, Answers: []request.CfgMaskAnswerInput{{CharacteristicID: opt.ID, VariableID: &nao.ID}},
	})
	if len(ev2.Assignments) != 0 {
		t.Fatalf("assignments com NAO = %d, want 0", len(ev2.Assignments))
	}
}

// Guard: a fórmula é obrigatória para característica do tipo FORMULA no vínculo
// com o item. Também exercita o Botão Itens Vinculados. (O guard por fórmula na
// estrutura compartilha o mesmo caminho `itemLocked` e a query é validada à parte;
// não é exercitado aqui por exigir um item real, dado o FK de item_structures.)
func TestIntegration_Configurator_FormulaGuardAndLinkedItems(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := configurator_uc.New(q)
	ctx := context.Background()
	uid := uuid.New()

	code := testutil.UniqueCode()
	itemCode := testutil.UniqueCode()

	defer func() {
		testutil.Exec(t, pool, "DELETE FROM cfg_item_characteristics WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("GRD%d%%", code))
	}()

	fchar, err := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
		Code: fmt.Sprintf("GRD%d_F", code), Description: "Área", Type: "FORMULA", CreatedBy: uid})
	if err != nil {
		t.Fatalf("CreateCharacteristic FORMULA: %v", err)
	}
	// sem fórmula → erro
	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: fchar.ID, Sequence: 5}); err == nil {
		t.Fatal("esperado erro: fórmula obrigatória para característica FORMULA")
	}
	// com fórmula → ok
	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: fchar.ID, Sequence: 5, Formula: "L*A"}); err != nil {
		t.Fatalf("AddItemCharacteristic com fórmula: %v", err)
	}

	// Botão Itens Vinculados → o item aparece para a característica.
	items, err := uc.ListItemsByCharacteristic(ctx, fchar.ID)
	if err != nil {
		t.Fatalf("ListItemsByCharacteristic: %v", err)
	}
	if len(items) != 1 || items[0] != itemCode {
		t.Fatalf("itens vinculados = %v, want [%d]", items, itemCode)
	}
}

// Avaliação de fórmula (Botão F): característica FORMULA calcula a resposta a
// partir de outra característica (LARGURA*2), e regra de item com fórmula calcula
// o conteúdo (QTD*10).
func TestIntegration_Configurator_Formula(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := configurator_uc.New(q)
	ctx := context.Background()
	uid := uuid.New()

	code := testutil.UniqueCode()
	itemCode := testutil.UniqueCode()

	defer func() {
		testutil.Exec(t, pool, "DELETE FROM cfg_item_rules WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_item_characteristics WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("FRM%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_sets WHERE description = $1", fmt.Sprintf("FRM%d-OPT", code))
	}()

	// LARGURA (numérica) e AREA (fórmula = LARGURA*2). Códigos únicos por execução;
	// a fórmula referencia a característica pelo seu código normalizado.
	largCode := fmt.Sprintf("FRM%d_LARG", code)
	larg, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
		Code: largCode, Description: "Largura", Type: "INF_NUMERICA", CreatedBy: uid})
	area, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
		Code: fmt.Sprintf("FRM%d_AREA", code), Description: "Área", Type: "FORMULA", CreatedBy: uid})
	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: larg.ID, Sequence: 10}); err != nil {
		t.Fatalf("add larg: %v", err)
	}
	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: area.ID, Sequence: 20, Formula: largCode + "*2"}); err != nil {
		t.Fatalf("add area: %v", err)
	}

	m, err := uc.GenerateMask(ctx, request.CfgGenerateMaskDTO{
		ItemCode: itemCode, Answers: []request.CfgMaskAnswerInput{{CharacteristicID: larg.ID, Value: "5"}},
	})
	if err != nil {
		t.Fatalf("GenerateMask: %v", err)
	}
	if m.Mask != "5#10" {
		t.Fatalf("mask = %q, want 5#10 (AREA=LARGURA*2)", m.Mask)
	}

	// Regra de item com fórmula (Botão F): conteúdo = QTD*10.
	set, _ := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: fmt.Sprintf("FRM%d-OPT", code), CreatedBy: uid})
	sim, _ := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{SetID: set.ID, Code: "SIM", Description: "Sim", MaskComposition: "SIM", CreatedBy: uid})
	opt, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{Code: fmt.Sprintf("FRM%d_OPT", code), Description: "Opção", Type: "ESCOLHA", SetID: &set.ID, CreatedBy: uid})
	qtdCode := fmt.Sprintf("FRM%d_QTD", code)
	qtd, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{Code: qtdCode, Description: "Qtd", Type: "INF_NUMERICA", CreatedBy: uid})

	if _, err := uc.CreateItemRule(ctx, request.CfgItemRuleDTO{
		ItemCode: itemCode, TargetTable: "ENGENHARIA", TargetField: "peso", Formula: qtdCode + "*10", Situation: "ACTIVE",
		Conditions: []request.CfgItemRuleConditionDTO{{CharacteristicID: opt.ID, Operator: "EQUAL", VariableID: &sim.ID}},
	}); err != nil {
		t.Fatalf("CreateItemRule: %v", err)
	}
	ev, err := uc.EvaluateItemRules(ctx, request.CfgEvaluateItemRulesDTO{
		ItemCode: itemCode,
		Answers: []request.CfgMaskAnswerInput{
			{CharacteristicID: opt.ID, VariableID: &sim.ID},
			{CharacteristicID: qtd.ID, Value: "5"},
		},
	})
	if err != nil {
		t.Fatalf("EvaluateItemRules: %v", err)
	}
	if len(ev.Assignments) != 1 || ev.Assignments[0].Content != "50" {
		t.Fatalf("assignment = %+v, want peso=50 (QTD*10)", ev.Assignments)
	}
}

// Botão Itens do Tipo Recebimento: uma característica com tipo de recebimento
// recebe respostas/itens vinculados por tipo.
func TestIntegration_Configurator_ReceivingItems(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := configurator_uc.New(q)
	ctx := context.Background()
	uid := uuid.New()
	code := testutil.UniqueCode()

	defer func() {
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("RCV%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_sets WHERE description = $1", fmt.Sprintf("RCV%d-OPT", code))
	}()

	set, _ := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: fmt.Sprintf("RCV%d-OPT", code), CreatedBy: uid})
	v, _ := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{SetID: set.ID, Code: "CLI", Description: "Cliente", MaskComposition: "CLI", CreatedBy: uid})
	// característica sem tipo de recebimento → erro
	plain, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{Code: fmt.Sprintf("RCV%d_P", code), Description: "Plain", Type: "ESCOLHA", SetID: &set.ID, CreatedBy: uid})
	if _, err := uc.AddReceivingItem(ctx, plain.ID, request.CfgReceivingItemDTO{ReceivingType: "RECEBIMENTO"}); err == nil {
		t.Fatal("esperado erro: característica sem tipo de recebimento")
	}
	// característica com tipo de recebimento
	ch, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
		Code: fmt.Sprintf("RCV%d_C", code), Description: "Material", Type: "ESCOLHA", SetID: &set.ID,
		ReceivingType: "RECEBIMENTO_VINCULO", CreatedBy: uid})
	if _, err := uc.AddReceivingItem(ctx, ch.ID, request.CfgReceivingItemDTO{
		VariableID: &v.ID, ReceivingType: "RECEBIMENTO", ItemCode: ptrInt64(4444)}); err != nil {
		t.Fatalf("AddReceivingItem: %v", err)
	}
	items, err := uc.ListReceivingItems(ctx, ch.ID)
	if err != nil {
		t.Fatalf("ListReceivingItems: %v", err)
	}
	if len(items) != 1 || items[0].ReceivingType != "RECEBIMENTO" || items[0].VariableCode != "CLI" {
		t.Fatalf("receiving items = %+v, want 1 (RECEBIMENTO/CLI)", items)
	}
}

func ptrInt64(v int64) *int64 { return &v }

// Etapa 2 (religação): valida que o repositório de resolução de estrutura roteia
// para o cfg_* quando o item é configurado no novo modelo — perguntas, respostas
// de máscara e criação de máscara propagada.
func TestIntegration_Structure_CfgRouting(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := configurator_uc.New(q)
	repo := structurequery.NewStructureQueryRepository(q)
	ctx := context.Background()
	uid := uuid.New()
	code := testutil.UniqueCode()
	itemCode := testutil.UniqueCode()

	testutil.Exec(t, pool, `INSERT INTO users (id, name, email, password, created_at, updated_at, role)
		VALUES ($1,'str-test',$2,'x',NOW(),NOW(),'ADMIN')`, uid, fmt.Sprintf("str-%d@test.local", code))
	defer func() {
		testutil.Exec(t, pool, "DELETE FROM item_masks WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_item_characteristics WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cfg_characteristics WHERE code LIKE $1", fmt.Sprintf("STR%d%%", code))
		testutil.Exec(t, pool, "DELETE FROM cfg_sets WHERE description = $1", fmt.Sprintf("STR%d-COR", code))
		testutil.Exec(t, pool, "DELETE FROM users WHERE id = $1", uid)
	}()

	set, _ := uc.CreateSet(ctx, request.CreateCfgSetDTO{Description: fmt.Sprintf("STR%d-COR", code), CreatedBy: uid})
	az, _ := uc.CreateVariable(ctx, request.CreateCfgVariableDTO{SetID: set.ID, Code: "AZUL", Description: "Azul", MaskComposition: "AZUL", CreatedBy: uid})
	cor, _ := uc.CreateCharacteristic(ctx, request.CreateCfgCharacteristicDTO{
		Code: fmt.Sprintf("STR%d_COR", code), Description: "Cor", Type: "ESCOLHA", SetID: &set.ID, CreatedBy: uid})
	if _, err := uc.AddItemCharacteristic(ctx, request.AddCfgItemCharacteristicDTO{
		ItemCode: itemCode, CharacteristicID: cor.ID, Sequence: 10}); err != nil {
		t.Fatalf("AddItemCharacteristic: %v", err)
	}
	if _, err := uc.GenerateMask(ctx, request.CfgGenerateMaskDTO{
		ItemCode: itemCode, Persist: true, CreatedBy: uid,
		Answers: []request.CfgMaskAnswerInput{{CharacteristicID: cor.ID, VariableID: &az.ID}},
	}); err != nil {
		t.Fatalf("GenerateMask: %v", err)
	}

	// perguntas do item via cfg
	qs, err := repo.GetItemQuestions(ctx, itemCode)
	if err != nil {
		t.Fatalf("GetItemQuestions: %v", err)
	}
	if len(qs) != 1 || qs[0].QuestionID != cor.ID || qs[0].Position != 10 {
		t.Fatalf("questions = %+v, want char %d / pos 10", qs, cor.ID)
	}
	// respostas de máscara via cfg
	ans, err := repo.GetMaskAnswersByItemAndValue(ctx, itemCode, "AZUL")
	if err != nil {
		t.Fatalf("GetMaskAnswersByItemAndValue: %v", err)
	}
	if len(ans) != 1 || ans[0].QuestionID() != cor.ID || ans[0].OptionID() != az.ID || ans[0].OptionValue() != "AZUL" {
		t.Fatalf("answers = %+v, want char %d / var %d / AZUL", ans, cor.ID, az.ID)
	}
	// criação de máscara propagada grava em cfg_item_mask_answers
	if err := repo.CreateMaskForItem(ctx, itemCode, "AZUL2",
		[]maskservice.ChildMaskAnswerInput{{QuestionID: cor.ID, OptionID: az.ID, Position: 10}}, uid); err != nil {
		t.Fatalf("CreateMaskForItem: %v", err)
	}
	ans2, err := repo.GetMaskAnswersByItemAndValue(ctx, itemCode, "AZUL2")
	if err != nil || len(ans2) != 1 || ans2[0].QuestionID() != cor.ID {
		t.Fatalf("máscara propagada não persistiu em cfg: %+v err=%v", ans2, err)
	}
}

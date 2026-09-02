package service

import (
	"context"

	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	str "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/google/uuid"
)

// Node representa um item na árvore BOM resolvida.
type Node struct {
	Component     *str.ItemStructure
	EffectiveMask *string // nil para nós genéricos
	RequiresMask  bool    // comp.Inherit=false + tem perguntas: subárvore precisa de máscara explícita
	Level         int
	Children      []*Node

	// Quantity é a quantidade por unidade do pai já com a fórmula avaliada
	// (quando houver) com as variáveis da configuração do pai; FormulaApplied
	// diz se a fórmula foi de fato usada ou se valeu a quantidade fixa.
	Quantity       float64
	FormulaApplied bool
}

// Resolve constrói a árvore BOM para um item configurado (com máscara conhecida).
// parentAnswers são as respostas do item itemCode para a máscara fornecida.
// createdBy é usado quando uma máscara propagada precisa ser auto-criada.
func (r *Resolver) Resolve(
	ctx context.Context,
	itemCode int64,
	mask string,
	parentAnswers []maskvo.MaskAnswer,
	level int,
	visited map[int64]bool,
	createdBy uuid.UUID,
) ([]*Node, error) {
	if visited[itemCode] {
		return nil, nil // guarda contra ciclos no caminho atual
	}
	visited[itemCode] = true
	defer delete(visited, itemCode)

	children, err := r.repo.GetDirectChildrenForMask(ctx, itemCode, mask)
	if err != nil {
		return nil, err
	}

	// As variáveis da configuração do pai (COMPRIMENTO, PROFUNDIDADE, …) valem
	// para todas as fórmulas de quantidade dos seus componentes; buscamos uma
	// única vez por nível.
	vars := r.formulaVars(ctx, children, itemCode, mask)

	nodes := make([]*Node, 0, len(children))
	for _, comp := range children {
		node, err := r.resolveChild(ctx, comp, parentAnswers, level, visited, createdBy)
		if err != nil {
			return nil, err
		}
		node.Quantity, node.FormulaApplied = comp.ResolvedQuantity(vars)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// formulaVars carrega as respostas nomeadas da configuração do pai apenas
// quando algum componente do nível realmente usa fórmula de quantidade.
func (r *Resolver) formulaVars(ctx context.Context, children []*str.ItemStructure, itemCode int64, mask string) map[string]float64 {
	if mask == "" {
		return nil
	}
	needed := false
	for _, comp := range children {
		if comp.HasQuantityFormula() {
			needed = true
			break
		}
	}
	if !needed {
		return nil
	}
	vars, err := r.repo.GetMaskAnswersWithNames(ctx, itemCode, mask)
	if err != nil {
		return nil
	}
	return vars
}

// ResolveGeneric constrói a árvore BOM para um item genérico (sem máscara).
// Usa GetDirectChildrenForMask com mask="" → retorna apenas filhos universais.
func (r *Resolver) ResolveGeneric(
	ctx context.Context,
	parentCode int64,
	level int,
	visited map[int64]bool,
) ([]*Node, error) {
	if visited[parentCode] {
		return nil, nil
	}
	visited[parentCode] = true
	defer delete(visited, parentCode)

	children, err := r.repo.GetDirectChildrenForMask(ctx, parentCode, "")
	if err != nil {
		return nil, err
	}

	nodes := make([]*Node, 0, len(children))
	for _, comp := range children {
		// Sem máscara não há variáveis: vale a quantidade fixa cadastrada.
		node := &Node{Component: comp, Level: level, Quantity: comp.Quantity}
		sub, err := r.ResolveGeneric(ctx, comp.ChildCode, level+1, visited)
		if err != nil {
			return nil, err
		}
		node.Children = sub
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// resolveChild determina o que um componente filho contribui para a árvore.
func (r *Resolver) resolveChild(
	ctx context.Context,
	comp *str.ItemStructure,
	parentAnswers []maskvo.MaskAnswer,
	level int,
	visited map[int64]bool,
	createdBy uuid.UUID,
) (*Node, error) {
	node := &Node{Component: comp, Level: level}

	questions, err := r.repo.GetItemQuestions(ctx, comp.ChildCode)
	if err != nil {
		return nil, err
	}

	switch {
	case comp.Inherit:
		// Herda máscara do pai via propagação.
		childMask := maskservice.PropagateMask(parentAnswers, questions)
		node.EffectiveMask = childMask

		if childMask == nil {
			// Propagação incompleta: fallback genérico.
			sub, err := r.ResolveGeneric(ctx, comp.ChildCode, level+1, visited)
			if err != nil {
				return nil, err
			}
			node.Children = sub
			return node, nil
		}

		childAnswers, err := r.ensureChildMask(ctx, comp.ChildCode, *childMask, parentAnswers, questions, createdBy)
		if err != nil {
			return nil, err
		}

		sub, err := r.Resolve(ctx, comp.ChildCode, *childMask, childAnswers, level+1, visited, createdBy)
		if err != nil {
			return nil, err
		}
		node.Children = sub

	case len(questions) == 0:
		// Genérico (Inherit=false, sem perguntas): recursa sem máscara.
		sub, err := r.ResolveGeneric(ctx, comp.ChildCode, level+1, visited)
		if err != nil {
			return nil, err
		}
		node.Children = sub

	default:
		// Configurado com Inherit=false: máscara definida manualmente na estrutura.
		// Exibido como folha — o caller pode expandir com uma consulta dedicada.
		node.RequiresMask = true
	}

	return node, nil
}

// ensureChildMask busca (ou cria) o registro de máscara do filho,
// retornando suas respostas para que a propagação continue.
func (r *Resolver) ensureChildMask(
	ctx context.Context,
	childCode int64,
	mask string,
	parentAnswers []maskvo.MaskAnswer,
	childQuestions []maskservice.ItemQuestion,
	createdBy uuid.UUID,
) ([]maskvo.MaskAnswer, error) {
	answers, err := r.repo.GetMaskAnswersByItemAndValue(ctx, childCode, mask)
	if err != nil {
		return nil, err
	}
	if len(answers) > 0 {
		return answers, nil
	}

	// Máscara ainda não existe: cria automaticamente com as respostas derivadas do pai.
	derived := maskservice.DeriveChildAnswers(parentAnswers, childQuestions)
	if err := r.repo.CreateMaskForItem(ctx, childCode, mask, derived, createdBy); err != nil {
		return nil, err
	}

	return r.repo.GetMaskAnswersByItemAndValue(ctx, childCode, mask)
}

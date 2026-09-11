package service

import (
	"context"
	"fmt"
	"strings"

	cfgentity "github.com/FelipePn10/panossoerp/internal/domain/configurator/entity"

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

	// ConfiguracaoIncompleta marca o nó cujo filho tem característica sem
	// origem: o pai não a responde, nenhuma regra de equivalência a deriva e
	// ela não tem resposta padrão. Antes isso virava silenciosamente a
	// estrutura genérica do filho — o usuário via uma lista de componentes
	// plausível que não correspondia à configuração pedida. É o "incomplete
	// configuration" do Oracle e a configuração inconsistente do SAP.
	ConfiguracaoIncompleta   bool
	CaracteristicasFaltantes []string
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
		// A máscara do filho vem de três fontes, nesta ordem: a mesma
		// característica respondida no pai, uma regra de equivalência, ou a
		// resposta padrão da característica no item.
		derivadas, faltantes, err := r.derivarRespostasDoFilho(ctx, comp, parentAnswers, questions)
		if err != nil {
			return nil, err
		}
		if len(faltantes) > 0 {
			// Sem origem para alguma característica: a subestrutura específica
			// da configuração não pode ser determinada. Mostra os componentes
			// genéricos (esses valem para qualquer máscara) e sinaliza, em vez
			// de entregar a estrutura genérica como se fosse a configurada.
			node.ConfiguracaoIncompleta = true
			node.CaracteristicasFaltantes = faltantes
			sub, err := r.ResolveGeneric(ctx, comp.ChildCode, level+1, visited)
			if err != nil {
				return nil, err
			}
			node.Children = sub
			return node, nil
		}

		partes := make([]string, 0, len(derivadas))
		for _, d := range derivadas {
			partes = append(partes, d.Valor)
		}
		mascaraFilho := strings.Join(partes, "#")
		node.EffectiveMask = &mascaraFilho

		childAnswers, err := r.ensureChildMask(ctx, comp.ChildCode, mascaraFilho, derivadas, createdBy)
		if err != nil {
			return nil, err
		}
		childMask := &mascaraFilho

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
	derivadas []respostaDerivada,
	createdBy uuid.UUID,
) ([]maskvo.MaskAnswer, error) {
	answers, err := r.repo.GetMaskAnswersByItemAndValue(ctx, childCode, mask)
	if err != nil {
		return nil, err
	}
	if len(answers) > 0 {
		return answers, nil
	}

	// Máscara ainda não existe: cria com as respostas derivadas.
	derived := make([]maskservice.ChildMaskAnswerInput, 0, len(derivadas))
	for _, d := range derivadas {
		derived = append(derived, maskservice.ChildMaskAnswerInput{QuestionID: d.QuestionID, OptionID: d.OptionID, Position: d.Position})
	}
	if err := r.repo.CreateMaskForItem(ctx, childCode, mask, derived, createdBy); err != nil {
		return nil, err
	}

	return r.repo.GetMaskAnswersByItemAndValue(ctx, childCode, mask)
}

// respostaDerivada é uma característica do filho já resolvida, com a origem do
// valor — útil para explicar ao usuário de onde veio cada pedaço da máscara.
type respostaDerivada struct {
	QuestionID int64
	OptionID   int64
	Position   int32
	Valor      string
	Origem     string // "pai" | "equivalencia" | "padrao"
}

// derivarRespostasDoFilho resolve cada característica do filho na ordem em que
// os ERPs de configuração fazem: herança direta (mesma característica
// respondida no pai), regra de equivalência (o procedure do SAP, a
// equivalência do Focco) e resposta padrão (o opcional padrão do Protheus).
// O que sobrar volta como faltante — configuração incompleta, nunca silêncio.
func (r *Resolver) derivarRespostasDoFilho(
	ctx context.Context,
	comp *str.ItemStructure,
	parentAnswers []maskvo.MaskAnswer,
	childQuestions []maskservice.ItemQuestion,
) ([]respostaDerivada, []string, error) {
	if len(childQuestions) == 0 {
		return nil, nil, nil
	}
	respostaDoPai := make(map[int64]maskvo.MaskAnswer, len(parentAnswers))
	for _, a := range parentAnswers {
		respostaDoPai[a.QuestionID()] = a
	}

	caracteristicas, err := r.repo.ListItemCharacteristics(ctx, comp.ChildCode)
	if err != nil {
		return nil, nil, err
	}
	padrao := make(map[int64]*int64, len(caracteristicas))
	codigo := make(map[int64]string, len(caracteristicas))
	for _, c := range caracteristicas {
		padrao[c.CharacteristicID] = c.DefaultVariableID
		codigo[c.CharacteristicID] = c.Code
	}

	equivalentes, err := r.equivalentesAplicaveis(ctx, comp, respostaDoPai)
	if err != nil {
		return nil, nil, err
	}

	derivadas := make([]respostaDerivada, 0, len(childQuestions))
	faltantes := make([]string, 0)
	for _, q := range childQuestions {
		if a, ok := respostaDoPai[q.QuestionID]; ok {
			derivadas = append(derivadas, respostaDerivada{QuestionID: q.QuestionID, OptionID: a.OptionID(), Position: q.Position, Valor: a.OptionValue(), Origem: "pai"})
			continue
		}
		variavel := equivalentes[q.QuestionID]
		origem := "equivalencia"
		if variavel == nil {
			variavel = padrao[q.QuestionID]
			origem = "padrao"
		}
		if variavel == nil {
			nome := codigo[q.QuestionID]
			if nome == "" {
				nome = fmt.Sprintf("característica %d", q.QuestionID)
			}
			faltantes = append(faltantes, nome)
			continue
		}
		valor, err := r.repo.GetVariableMaskComposition(ctx, *variavel)
		if err != nil {
			return nil, nil, fmt.Errorf("lendo a composição de máscara da variável %d: %w", *variavel, err)
		}
		derivadas = append(derivadas, respostaDerivada{QuestionID: q.QuestionID, OptionID: *variavel, Position: q.Position, Valor: valor, Origem: origem})
	}
	return derivadas, faltantes, nil
}

// equivalentesAplicaveis devolve característica do filho → variável, para as
// regras cuja condição no pai é satisfeita pela configuração atual.
func (r *Resolver) equivalentesAplicaveis(
	ctx context.Context,
	comp *str.ItemStructure,
	respostaDoPai map[int64]maskvo.MaskAnswer,
) (map[int64]*int64, error) {
	regras, err := r.repo.ListEquivalentRules(ctx, comp.ParentCode)
	if err != nil {
		return nil, err
	}
	fora := map[int64]*int64{}
	for _, regra := range regras {
		if regra.ChildItemCode != comp.ChildCode || regra.ChildVariableID == nil || regra.ParentVariableID == nil {
			continue
		}
		resposta, ok := respostaDoPai[regra.ParentCharacteristicID]
		if !ok {
			continue
		}
		// A comparação é por variável (id), não por texto: o código da variável
		// pode repetir entre conjuntos diferentes.
		igual := resposta.OptionID() == *regra.ParentVariableID
		switch regra.ParentOperator {
		case cfgentity.OpEqual:
			if !igual {
				continue
			}
		case cfgentity.OpDifferent:
			if igual {
				continue
			}
		default:
			// Operador que depende de ordem/faixa não é avaliável por id.
			continue
		}
		fora[regra.ChildCharacteristicID] = regra.ChildVariableID
	}
	return fora, nil
}

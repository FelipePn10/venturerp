package structure

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc"
)

// rotulos traduz a chave do instantâneo para o nome que o usuário vê.
var rotulos = map[string]string{
	"quantidade":             "Quantidade",
	"formula_quantidade":     "Fórmula da quantidade",
	"arredondamento":         "Arredondamento",
	"casas_decimais":         "Casas decimais",
	"unidade":                "Unidade de medida",
	"perda_percentual":       "Perda (%)",
	"formula_perda":          "Fórmula da perda",
	"perda_setup":            "Perda de preparação",
	"tipo_perda_custo":       "Tipo da perda de custo",
	"perda_custo":            "Perda de custo",
	"situacao":               "Situação na estrutura",
	"sequencia":              "Posição",
	"vigencia_inicio":        "Início da vigência",
	"vigencia_fim":           "Fim da vigência",
	"coproduto":              "Co-produto",
	"quantidade_por_ordem":   "Quantidade por ordem",
	"grupo_alternativo":      "Grupo de alternativos",
	"prioridade_alternativo": "Prioridade do alternativo",
	"almoxarifado":           "Almoxarifado",
	"almoxarifado_linha":     "Almoxarifado de linha",
	"centro_custo":           "Centro de custo",
	"critico_mps":            "Crítico para o plano mestre",
	"gera_inspecao":          "Gera inspeção",
	"ativo":                  "Ativo",
}

// legivel formata um valor do instantâneo para leitura.
func legivel(v any) string {
	switch t := v.(type) {
	case nil:
		return "—"
	case bool:
		if t {
			return "Sim"
		}
		return "Não"
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", t), "0"), ".")
	case string:
		if t == "" {
			return "—"
		}
		// Datas chegam em ISO; mostra só o dia.
		if len(t) >= 10 && t[4] == '-' && t[7] == '-' {
			return t[8:10] + "/" + t[5:7] + "/" + t[0:4]
		}
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

// comparar devolve só os campos que mudaram, já com rótulo em português.
//
// Guardar o instantâneo inteiro e mostrar a diferença é o que faz o histórico
// ser útil: quem lê quer ver "Quantidade: 2 → 3", não dois blocos de JSON.
func comparar(antes, depois map[string]any) []structure_uc.FieldChange {
	chaves := map[string]struct{}{}
	for k := range antes {
		chaves[k] = struct{}{}
	}
	for k := range depois {
		chaves[k] = struct{}{}
	}
	out := []structure_uc.FieldChange{}
	for k := range chaves {
		a, d := legivel(antes[k]), legivel(depois[k])
		if a == d {
			continue
		}
		rotulo, ok := rotulos[k]
		if !ok {
			rotulo = k
		}
		out = append(out, structure_uc.FieldChange{Field: rotulo, Before: a, After: d})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Field < out[j].Field })
	return out
}

// ListByParent devolve o histórico do item pai, do mais recente ao mais antigo.
func (r *ItemStructureRepositorySQLC) ListByParent(
	ctx context.Context,
	parentCode int64,
	limit int,
) ([]structure_uc.StructureHistoryEntry, error) {
	if r.pool == nil {
		return []structure_uc.StructureHistoryEntry{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT h.id, p.business_code, c.business_code, h.action, h.changed_by,
		       COALESCE(u.name, ''), h.changed_at, h.before_state, h.after_state
		  FROM item_structure_history h
		  LEFT JOIN items p ON p.code = h.parent_code
		  LEFT JOIN items c ON c.code = h.child_code
		  LEFT JOIN users u ON u.id = h.changed_by
		 WHERE h.parent_code = $1
		 ORDER BY h.changed_at DESC, h.id DESC
		 LIMIT $2`, parentCode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []structure_uc.StructureHistoryEntry{}
	for rows.Next() {
		var e structure_uc.StructureHistoryEntry
		var pai, filho *string
		var antesRaw, depoisRaw []byte
		if err := rows.Scan(&e.ID, &pai, &filho, &e.Action, &e.ChangedBy,
			&e.ChangedByName, &e.ChangedAt, &antesRaw, &depoisRaw); err != nil {
			return nil, err
		}
		if pai != nil {
			e.ParentCode = *pai
		}
		if filho != nil {
			e.ChildCode = *filho
		}
		var antes, depois map[string]any
		_ = json.Unmarshal(antesRaw, &antes)
		_ = json.Unmarshal(depoisRaw, &depois)
		e.Changes = comparar(antes, depois)
		out = append(out, e)
	}
	return out, rows.Err()
}

// Esquema de rateio de indiretos, histórico de custo e a conferência de posse do
// item (migração 000373).
//
// Escrito à mão em pgx, como os módulos recentes, e não em sqlc: as consultas
// aqui têm filtro condicional e leitura de JSONB, que o sqlc não expressa bem.
//
// ⚠️ Posse do item: `item_standard_costs`, `item_purchase_costs` e
// `cost_rollup_log` NÃO têm coluna de empresa. A empresa do custo é a empresa do
// item — `items.code` é único global e cada item pertence a uma empresa. Por isso
// a conferência é feita contra `items.enterprise_id`, e não por coluna
// denormalizada que poderia discordar da verdade.
package standard_cost

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ConferirItemDaEmpresa recusa operar o custo de um item de outra empresa.
//
// Sem esta conferência, o código do item vinha do cliente e qualquer empresa podia
// LER o custo-padrão e o custo de compra de um item alheio — ou SOBRESCREVÊ-LOS,
// porque o upsert é por `item_code`. Não era mistura acidental (os códigos não
// colidem), era falta de controle de acesso.
func (r *StandardCostRepositorySQLC) ConferirItemDaEmpresa(ctx context.Context, itemCode int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if r.pool == nil {
		// Repositório montado sem pool: melhor recusar que deixar passar sem
		// conferir a posse.
		return fmt.Errorf("conferência de posse do item não configurada")
	}
	var dono int64
	err = r.pool.QueryRow(ctx, `SELECT enterprise_id FROM public.items WHERE code = $1`, itemCode).Scan(&dono)
	if err == pgx.ErrNoRows {
		return errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado", itemCode))
	}
	if err != nil {
		return fmt.Errorf("conferindo a empresa do item %d: %w", itemCode, err)
	}
	if dono != empresa {
		// Mensagem de "não encontrado", não de "sem permissão": dizer que o item
		// existe em outra empresa já é informação que não deve sair daqui.
		return errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado", itemCode))
	}
	return nil
}

const colunasRegra = `id, enterprise_id, code, description, base::text, method::text, rate,
work_center_id, item_code, plano_contas_id, centro_custo_id, valid_from, valid_to, is_active,
notes, created_by::text, created_at, updated_at`

func scanRegra(row pgx.Row) (*entity.RegraDeRateio, error) {
	var r entity.RegraDeRateio
	var base, metodo, criadoPor string
	if err := row.Scan(&r.ID, &r.EnterpriseID, &r.Code, &r.Description, &base, &metodo, &r.Rate,
		&r.WorkCenterID, &r.ItemCode, &r.PlanoContasID, &r.CentroCustoID, &r.ValidFrom, &r.ValidTo,
		&r.IsActive, &r.Notes, &criadoPor, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	r.Base = entity.BaseDeRateio(base)
	r.Method = entity.MetodoDeRateio(metodo)
	if id, err := uuid.Parse(criadoPor); err == nil {
		r.CreatedBy = id
	}
	return &r, nil
}

func (r *StandardCostRepositorySQLC) ListarRegrasDeRateio(ctx context.Context) ([]*entity.RegraDeRateio, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT %s FROM public.cost_overhead_rules
WHERE enterprise_id = $1
ORDER BY is_active DESC, code`, colunasRegra), empresa)
	if err != nil {
		return nil, fmt.Errorf("listando regras de rateio: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.RegraDeRateio, 0)
	for rows.Next() {
		regra, err := scanRegra(rows)
		if err != nil {
			return nil, fmt.Errorf("lendo regra de rateio: %w", err)
		}
		out = append(out, regra)
	}
	return out, rows.Err()
}

func (r *StandardCostRepositorySQLC) CriarRegraDeRateio(ctx context.Context, regra *entity.RegraDeRateio) (*entity.RegraDeRateio, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Regra com escopo de item exige que o item seja da empresa: uma regra apontando
	// item alheio nunca aplicaria, e ficaria no cadastro parecendo ativa.
	if regra.ItemCode != nil {
		if err := r.ConferirItemDaEmpresa(ctx, *regra.ItemCode); err != nil {
			return nil, err
		}
	}
	// Enums entram como $n::text::<enum>: com o cast direto o pgx não codifica a
	// string Go num OID de enum desconhecido e a gravação falha com erro de domínio.
	criada, err := scanRegra(r.pool.QueryRow(ctx, fmt.Sprintf(`
INSERT INTO public.cost_overhead_rules
  (enterprise_id, code, description, base, method, rate, work_center_id, item_code,
   plano_contas_id, centro_custo_id, valid_from, valid_to, is_active, notes, created_by)
VALUES ($1,$2,$3,$4::text::cost_overhead_base_enum,$5::text::cost_overhead_method_enum,
        $6,$7,$8,$9,$10,$11,$12,$13,$14,$15::uuid)
RETURNING %s`, colunasRegra),
		empresa, regra.Code, regra.Description, string(regra.Base), string(regra.Method),
		regra.Rate, regra.WorkCenterID, regra.ItemCode, regra.PlanoContasID, regra.CentroCustoID,
		regra.ValidFrom, regra.ValidTo, regra.IsActive, regra.Notes, regra.CreatedBy.String()))
	if err != nil {
		return nil, traduzirErroDeRateio(err)
	}
	return criada, nil
}

func (r *StandardCostRepositorySQLC) AtualizarRegraDeRateio(ctx context.Context, regra *entity.RegraDeRateio) (*entity.RegraDeRateio, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if regra.ItemCode != nil {
		if err := r.ConferirItemDaEmpresa(ctx, *regra.ItemCode); err != nil {
			return nil, err
		}
	}
	atualizada, err := scanRegra(r.pool.QueryRow(ctx, fmt.Sprintf(`
UPDATE public.cost_overhead_rules SET
  description = $3, base = $4::text::cost_overhead_base_enum,
  method = $5::text::cost_overhead_method_enum, rate = $6,
  work_center_id = $7, item_code = $8, plano_contas_id = $9, centro_custo_id = $10,
  valid_from = $11, valid_to = $12, is_active = $13, notes = $14, updated_at = NOW()
WHERE id = $1 AND enterprise_id = $2
RETURNING %s`, colunasRegra),
		regra.ID, empresa, regra.Description, string(regra.Base), string(regra.Method),
		regra.Rate, regra.WorkCenterID, regra.ItemCode, regra.PlanoContasID, regra.CentroCustoID,
		regra.ValidFrom, regra.ValidTo, regra.IsActive, regra.Notes))
	if err == pgx.ErrNoRows {
		return nil, errorsuc.NewNotFoundError("regra de rateio não encontrada")
	}
	if err != nil {
		return nil, traduzirErroDeRateio(err)
	}
	return atualizada, nil
}

// DesativarRegraDeRateio não apaga: uma apuração antiga foi feita com aquela regra,
// e apagá-la deixaria o histórico apontando para nada.
func (r *StandardCostRepositorySQLC) DesativarRegraDeRateio(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
UPDATE public.cost_overhead_rules SET is_active = FALSE, updated_at = NOW()
WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("desativando regra de rateio: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("regra de rateio não encontrada")
	}
	return nil
}

func (r *StandardCostRepositorySQLC) GravarHistoricoDeCusto(ctx context.Context, h *entity.HistoricoDeCusto) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var rateios []byte
	if len(h.Rateios) > 0 {
		if rateios, err = json.Marshal(h.Rateios); err != nil {
			return fmt.Errorf("serializando o rastro dos rateios: %w", err)
		}
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO public.item_standard_cost_history
  (enterprise_id, item_code, mask, lot_size, material_cost, setup_cost, machine_cost,
   labor_cost, subcontract_cost, overhead_cost, own_level_cost, lower_level_cost,
   total_cost, currency, overhead_detail, calculated_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::uuid)`,
		empresa, h.ItemCode, h.Mask, h.LotSize,
		h.Componentes.Material, h.Componentes.Setup, h.Componentes.Maquina,
		h.Componentes.MaoDeObra, h.Componentes.Subcontratacao, h.Componentes.Overhead,
		h.Componentes.NivelProprio, h.Componentes.NivelInferior,
		h.TotalCost, orDefault(h.Currency, "BRL"), rateios, h.CalculatedBy.String())
	if err != nil {
		return fmt.Errorf("gravando histórico de custo: %w", err)
	}
	return nil
}

func (r *StandardCostRepositorySQLC) HistoricoDeCusto(ctx context.Context, itemCode int64, mask string, limite int) ([]*entity.HistoricoDeCusto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.ConferirItemDaEmpresa(ctx, itemCode); err != nil {
		return nil, err
	}
	if limite <= 0 || limite > 200 {
		limite = 24 // dois anos de apuração mensal
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, item_code, mask, lot_size, material_cost, setup_cost, machine_cost,
       labor_cost, subcontract_cost, overhead_cost, own_level_cost, lower_level_cost,
       total_cost, currency, overhead_detail, calculated_at
FROM public.item_standard_cost_history
WHERE enterprise_id = $1 AND item_code = $2 AND mask = $3
ORDER BY calculated_at DESC, id DESC
LIMIT $4`, empresa, itemCode, mask, limite)
	if err != nil {
		return nil, fmt.Errorf("lendo histórico de custo: %w", err)
	}
	defer rows.Close()

	out := make([]*entity.HistoricoDeCusto, 0, limite)
	for rows.Next() {
		var h entity.HistoricoDeCusto
		var rateios []byte
		if err := rows.Scan(&h.ID, &h.ItemCode, &h.Mask, &h.LotSize,
			&h.Componentes.Material, &h.Componentes.Setup, &h.Componentes.Maquina,
			&h.Componentes.MaoDeObra, &h.Componentes.Subcontratacao, &h.Componentes.Overhead,
			&h.Componentes.NivelProprio, &h.Componentes.NivelInferior,
			&h.TotalCost, &h.Currency, &rateios, &h.CalculatedAt); err != nil {
			return nil, fmt.Errorf("lendo apuração do histórico: %w", err)
		}
		h.EnterpriseID = empresa
		if len(rateios) > 0 {
			_ = json.Unmarshal(rateios, &h.Rateios)
		}
		out = append(out, &h)
	}
	return out, rows.Err()
}

// CentrosDoRoteiro devolve os centros de trabalho que o roteiro ativo do item usa.
// É o que decide quais regras com escopo de centro alcançam este item.
func (r *StandardCostRepositorySQLC) CentrosDoRoteiro(ctx context.Context, itemCode int64, mask string) (map[int64]bool, error) {
	rows, err := r.pool.Query(ctx, `
SELECT DISTINCT COALESCE(ro.work_center_id, o.work_center_id)
FROM manufacturing_routes mr
JOIN route_operations ro ON ro.route_id = mr.id
LEFT JOIN operations o ON o.id = ro.operation_id
WHERE mr.item_code = $1 AND COALESCE(mr.mask,'') = COALESCE($2,'') AND mr.is_active
  AND COALESCE(ro.work_center_id, o.work_center_id) IS NOT NULL`, itemCode, mask)
	if err != nil {
		// Roteiro ausente ou schema diferente não impede a apuração: sem centro
		// conhecido, só as regras de escopo geral aplicam. Parar a apuração de custo
		// porque o item não tem roteiro seria pior.
		return map[int64]bool{}, nil
	}
	defer rows.Close()
	centros := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			centros[id] = true
		}
	}
	return centros, nil
}

// traduzirErroDeRateio transforma violação de restrição em mensagem que a pessoa
// entenda: sem isso o CHECK do banco chega à tela em inglês do PostgreSQL.
func traduzirErroDeRateio(err error) error {
	if err == nil {
		return nil
	}
	texto := err.Error()
	switch {
	case strings.Contains(texto, "cost_overhead_rules_codigo_unico"):
		return errorsuc.NewConflictError("já existe uma regra de rateio com este código")
	case strings.Contains(texto, "cost_overhead_rules_taxa_positiva"):
		return errorsuc.NewValidationError("a taxa do rateio tem de ser maior que zero")
	case strings.Contains(texto, "cost_overhead_rules_percentual_plausivel"):
		return errorsuc.NewValidationError(
			"o percentual está acima de 1000%: informe a fração (0,12 para 12%), não o número inteiro")
	case strings.Contains(texto, "cost_overhead_rules_hora_com_base_de_hora"):
		return errorsuc.NewValidationError(
			"valor por hora só pode incidir sobre máquina, mão de obra, preparação ou conversão — bases medidas em horas")
	case strings.Contains(texto, "cost_overhead_rules_vigencia"):
		return errorsuc.NewValidationError("a data final da vigência é anterior à data inicial")
	}
	return err
}

// ─── custo-padrão com todos os componentes ───────────────────────────────────

const colunasCustoPadrao = `id, item_code, mask, material_cost, setup_cost, machine_cost,
labor_cost, subcontract_cost, overhead_cost, own_level_cost, lower_level_cost,
lot_size, total_cost, currency, calculated_at, calculated_by::text`

func scanCustoPadrao(row pgx.Row) (*entity.ItemStandardCost, error) {
	var c entity.ItemStandardCost
	var autor *string
	if err := row.Scan(&c.ID, &c.ItemCode, &c.Mask, &c.MaterialCost, &c.SetupCost,
		&c.MachineCost, &c.LaborCost, &c.SubcontractCost, &c.OverheadCost,
		&c.OwnLevelCost, &c.LowerLevelCost, &c.LotSize, &c.TotalCost, &c.Currency,
		&c.CalculatedAt, &autor); err != nil {
		return nil, err
	}
	if autor != nil {
		if id, err := uuid.Parse(*autor); err == nil {
			c.CalculatedBy = id
		}
	}
	return &c, nil
}

// UpsertItemStandardCost grava o custo-padrão com TODOS os componentes.
//
// Escrito à mão, substituindo a consulta do sqlc: a gerada cobre apenas material,
// mão de obra e overhead. Com a conversão separada em componentes (migração
// 000373), gravar só esses três deixava preparação, hora-máquina e serviço de
// terceiro FORA do total gravado — e é o total gravado que a precificação lê. A
// resposta da apuração estava certa; o valor guardado, não.
func (r *StandardCostRepositorySQLC) UpsertItemStandardCost(ctx context.Context, cost *entity.ItemStandardCost) (*entity.ItemStandardCost, error) {
	if err := r.ConferirItemDaEmpresa(ctx, cost.ItemCode); err != nil {
		return nil, err
	}
	lote := cost.LotSize
	if lote <= 0 {
		lote = 1
	}
	moeda := cost.Currency
	if moeda == "" {
		moeda = "BRL"
	}
	saved, err := scanCustoPadrao(r.pool.QueryRow(ctx, fmt.Sprintf(`
INSERT INTO public.item_standard_costs
  (item_code, mask, material_cost, setup_cost, machine_cost, labor_cost,
   subcontract_cost, overhead_cost, own_level_cost, lower_level_cost, lot_size,
   currency, calculated_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::uuid)
ON CONFLICT (item_code, mask) DO UPDATE SET
  material_cost    = EXCLUDED.material_cost,
  setup_cost       = EXCLUDED.setup_cost,
  machine_cost     = EXCLUDED.machine_cost,
  labor_cost       = EXCLUDED.labor_cost,
  subcontract_cost = EXCLUDED.subcontract_cost,
  overhead_cost    = EXCLUDED.overhead_cost,
  own_level_cost   = EXCLUDED.own_level_cost,
  lower_level_cost = EXCLUDED.lower_level_cost,
  lot_size         = EXCLUDED.lot_size,
  currency         = EXCLUDED.currency,
  calculated_at    = NOW(),
  calculated_by    = EXCLUDED.calculated_by
RETURNING %s`, colunasCustoPadrao),
		cost.ItemCode, cost.Mask, cost.MaterialCost, cost.SetupCost, cost.MachineCost,
		cost.LaborCost, cost.SubcontractCost, cost.OverheadCost, cost.OwnLevelCost,
		cost.LowerLevelCost, lote, moeda, cost.CalculatedBy.String()))
	if err != nil {
		return nil, fmt.Errorf("gravando o custo-padrão do item %d: %w", cost.ItemCode, err)
	}
	return saved, nil
}

// GetItemStandardCost lê o custo-padrão com os componentes.
func (r *StandardCostRepositorySQLC) GetItemStandardCost(ctx context.Context, itemCode int64, mask string) (*entity.ItemStandardCost, error) {
	if err := r.ConferirItemDaEmpresa(ctx, itemCode); err != nil {
		return nil, err
	}
	c, err := scanCustoPadrao(r.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT %s FROM public.item_standard_costs WHERE item_code = $1 AND mask = $2`, colunasCustoPadrao),
		itemCode, mask))
	if err == pgx.ErrNoRows {
		return nil, errorsuc.NewNotFoundError("custo-padrão não apurado para o item")
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o custo-padrão do item %d: %w", itemCode, err)
	}
	return c, nil
}

// ListItemStandardCosts devolve o custo-padrão do item em todas as máscaras.
func (r *StandardCostRepositorySQLC) ListItemStandardCosts(ctx context.Context, itemCode int64) ([]*entity.ItemStandardCost, error) {
	if err := r.ConferirItemDaEmpresa(ctx, itemCode); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT %s FROM public.item_standard_costs WHERE item_code = $1 ORDER BY mask`, colunasCustoPadrao), itemCode)
	if err != nil {
		return nil, fmt.Errorf("listando o custo-padrão do item %d: %w", itemCode, err)
	}
	defer rows.Close()
	out := make([]*entity.ItemStandardCost, 0)
	for rows.Next() {
		c, err := scanCustoPadrao(rows)
		if err != nil {
			return nil, fmt.Errorf("lendo custo-padrão: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

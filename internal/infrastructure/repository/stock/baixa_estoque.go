package stock

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// Regras de baixa: de QUAL linha de saldo (endereço, lote) sai o que o movimento
// tira. A NF-e de saída, o apontamento e o ajuste de inventário não dizem o
// endereço nem o lote; antes a saída caía numa linha sem endereço que ficava
// negativa, enquanto o endereço seguia mostrando material que já tinha saído —
// e, como a linha negativa não tem valor, o custo médio do item subia a cada
// saída (valor total ÷ quantidade menor).

// TravarItensTx serializa, até o fim da transação, quem mexe no saldo destes
// itens. As rotinas de estoque tocam saldo, lote e reserva em ordens diferentes:
// sem uma trava comum na frente, duas operações do mesmo item se esperavam até o
// banco derrubar uma delas, e quem lançava via "erro interno". Em ordem crescente
// para dois documentos com os mesmos itens não se cruzarem. A trava é reentrante.
func TravarItensTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, itens ...int64) error {
	// Antes do item: o fechamento do mês trava esta mesma chave por inteiro, e um
	// lançamento pela metade não pode entrar nem sair da foto do mês.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared(hashtextextended($1, 0))`, chaveDoFechamento(enterpriseID)); err != nil {
		return fmt.Errorf("aguardando o fechamento de estoque: %w", err)
	}
	ordem := append([]int64(nil), itens...)
	sort.Slice(ordem, func(i, j int) bool { return ordem[i] < ordem[j] })
	for i, item := range ordem {
		if i > 0 && item == ordem[i-1] {
			continue
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('stock_item:' || $1::bigint::text || ':' || $2::bigint::text, 0))`, enterpriseID, item); err != nil {
			return fmt.Errorf("travando o saldo do item: %w", err)
		}
	}
	return nil
}

func chaveDoFechamento(enterpriseID int64) string {
	return fmt.Sprintf("fechamento-estoque:%d", enterpriseID)
}

// linhaDeSaldo é o saldo de um endereço do item no almoxarifado.
type linhaDeSaldo struct {
	endereco   string
	quantidade decimal.Decimal
	reservado  decimal.Decimal
	bloqueado  bool
}

func (l linhaDeSaldo) livre() decimal.Decimal {
	return decimal.Max(l.quantidade.Sub(l.reservado), decimal.Zero)
}

// repartirSaida decide de quais endereços sai a quantidade de uma saída sem
// endereço: primeiro a linha sem endereço, depois os endereços em ordem, usando
// o livre e só então o reservado. Endereço bloqueado fica de fora. Devolve o
// quanto sai de cada endereço e o que não coube em lugar nenhum.
func repartirSaida(linhas []linhaDeSaldo, quantidade decimal.Decimal) (map[string]decimal.Decimal, decimal.Decimal) {
	ordem := make([]linhaDeSaldo, 0, len(linhas))
	for _, l := range linhas {
		if !l.bloqueado {
			ordem = append(ordem, l)
		}
	}
	sort.SliceStable(ordem, func(i, j int) bool { return ordem[i].endereco < ordem[j].endereco })

	partes := map[string]decimal.Decimal{}
	resta := quantidade
	tirar := func(endereco string, ate decimal.Decimal) {
		q := decimal.Min(ate, resta)
		if q.IsPositive() {
			partes[endereco] = partes[endereco].Add(q)
			resta = resta.Sub(q)
		}
	}
	for _, l := range ordem {
		tirar(l.endereco, l.livre())
	}
	for _, l := range ordem {
		tirar(l.endereco, decimal.Max(l.quantidade, decimal.Zero).Sub(partes[l.endereco]))
	}
	return partes, resta
}

// repartirExcedenteReservado diz quanto do reservado muda em cada linha: a reserva
// é do item, repartida pelo livre de cada endereço, e a saída de um endereço pode
// deixar nele mais reservado que saldo. O excedente vai para o livre dos outros;
// o que não couber fica onde está.
func repartirExcedenteReservado(linhas []linhaDeSaldo) map[string]decimal.Decimal {
	excedente := decimal.Zero
	for _, l := range linhas {
		excedente = excedente.Add(decimal.Max(l.reservado.Sub(decimal.Max(l.quantidade, decimal.Zero)), decimal.Zero))
	}
	mudancas := map[string]decimal.Decimal{}
	movido := decimal.Zero
	for _, l := range linhas {
		q := decimal.Min(l.livre(), excedente.Sub(movido))
		if q.IsPositive() {
			mudancas[l.endereco] = q
			movido = movido.Add(q)
		}
	}
	for _, l := range linhas {
		q := decimal.Min(decimal.Max(l.reservado.Sub(decimal.Max(l.quantidade, decimal.Zero)), decimal.Zero), movido)
		if q.IsPositive() {
			mudancas[l.endereco] = mudancas[l.endereco].Sub(q)
			movido = movido.Sub(q)
		}
	}
	return mudancas
}

func reequilibrarReservadoTx(ctx context.Context, tx pgx.Tx, enterpriseID, item int64, mask string, almox int64) error {
	rows, err := tx.Query(ctx,
		`SELECT address, quantity, reserved_qty FROM public.stock_balances
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 ORDER BY address FOR UPDATE`,
		enterpriseID, item, mask, almox)
	if err != nil {
		return fmt.Errorf("lendo a reserva por endereço: %w", err)
	}
	var linhas []linhaDeSaldo
	for rows.Next() {
		var l linhaDeSaldo
		if err := rows.Scan(&l.endereco, &l.quantidade, &l.reservado); err != nil {
			rows.Close()
			return fmt.Errorf("lendo a reserva por endereço: %w", err)
		}
		linhas = append(linhas, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("lendo a reserva por endereço: %w", err)
	}
	for endereco, q := range repartirExcedenteReservado(linhas) {
		if q.IsZero() {
			continue
		}
		if _, err := tx.Exec(ctx,
			`UPDATE public.stock_balances SET reserved_qty = reserved_qty + $6, updated_at = NOW()
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
			enterpriseID, item, mask, almox, endereco, q); err != nil {
			return fmt.Errorf("acertando a reserva por endereço: %w", err)
		}
	}
	return nil
}

// linhaDeLote é o saldo de um lote num endereço.
type linhaDeLote struct {
	id         int64
	lote       string
	endereco   string
	quantidade decimal.Decimal
	reservado  decimal.Decimal
}

// alocarLoteInformado distribui a saída de um lote informado pelos endereços em
// que ele está (só no endereço informado, quando há). Devolve o quanto sai de
// cada endereço, o que faltou e se o almoxarifado tem saldo em lote do item —
// sem nenhum, o item não é controlado por lote ali e o lote digitado é só
// referência do documento.
func alocarLoteInformado(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement, endereco string, quantidade decimal.Decimal) (map[string]decimal.Decimal, decimal.Decimal, bool, error) {
	rows, err := tx.Query(ctx,
		`SELECT lot, address, quantity, reserved_qty FROM public.stock_lot_balances
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND quantity > 0
		  ORDER BY address FOR UPDATE`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID)
	if err != nil {
		return nil, decimal.Zero, false, fmt.Errorf("lendo o saldo do lote: %w", err)
	}
	defer rows.Close()
	lote := strings.TrimSpace(*m.Lot)
	partes := map[string]decimal.Decimal{}
	resta := quantidade
	controlado := false
	for rows.Next() {
		var l linhaDeLote
		if err := rows.Scan(&l.lote, &l.endereco, &l.quantidade, &l.reservado); err != nil {
			return nil, decimal.Zero, false, fmt.Errorf("lendo o saldo do lote: %w", err)
		}
		controlado = true
		if l.lote != lote || (endereco != "" && l.endereco != endereco) {
			continue
		}
		q := decimal.Min(l.quantidade, resta)
		if q.IsPositive() {
			partes[l.endereco] = partes[l.endereco].Add(q)
			resta = resta.Sub(q)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, decimal.Zero, false, fmt.Errorf("lendo o saldo do lote: %w", err)
	}
	return partes, resta, controlado, nil
}

// baixarLotesSemInformar acerta os lotes depois de uma saída que não disse o
// lote: o que os lotes do endereço somam acima do saldo do endereço já saiu.
// Sai primeiro o que vence antes. Devolve o resumo para a observação do
// movimento, que é o que sobra de rastreabilidade dessa baixa.
func baixarLotesSemInformar(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement, endereco string) ([]string, error) {
	var saldo decimal.Decimal
	err := tx.QueryRow(ctx,
		`SELECT quantity FROM public.stock_balances
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, endereco).Scan(&saldo)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lendo o saldo do endereço: %w", err)
	}
	// Sai primeiro o que pode sair, na regra do item (FEFO ou FIFO).
	ordem := `l.expires_at IS NULL, l.expires_at, l.received_at NULLS LAST, b.lot`
	var regra string
	err = tx.QueryRow(ctx, `SELECT lot_issue_rule FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, m.ItemCode).Scan(&regra)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lendo a regra de baixa do item: %w", err)
	}
	if regra == entity.BaixaFIFO {
		ordem = `l.received_at NULLS LAST, l.expires_at IS NULL, l.expires_at, b.lot`
	}
	rows, err := tx.Query(ctx,
		`SELECT b.id, b.lot, b.address, b.quantity, b.reserved_qty
		   FROM public.stock_lot_balances b
		   LEFT JOIN public.stock_lots l ON l.enterprise_id=b.enterprise_id AND l.item_code=b.item_code AND l.mask=b.mask AND l.lot=b.lot
		  WHERE b.enterprise_id=$1 AND b.item_code=$2 AND b.mask=$3 AND b.warehouse_id=$4 AND b.address=$5 AND b.quantity > 0
		  ORDER BY `+loteSemSaidaSQL+`, (b.quantity = b.reserved_qty), `+ordem+`
		  FOR UPDATE OF b`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, endereco)
	if err != nil {
		return nil, fmt.Errorf("lendo os lotes do endereço: %w", err)
	}
	var lotes []linhaDeLote
	soma := decimal.Zero
	for rows.Next() {
		var l linhaDeLote
		if err := rows.Scan(&l.id, &l.lote, &l.endereco, &l.quantidade, &l.reservado); err != nil {
			rows.Close()
			return nil, fmt.Errorf("lendo os lotes do endereço: %w", err)
		}
		lotes = append(lotes, l)
		soma = soma.Add(l.quantidade)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lendo os lotes do endereço: %w", err)
	}
	excesso := soma.Sub(decimal.Max(saldo, decimal.Zero))
	var resumo []string
	for _, l := range lotes {
		if !excesso.IsPositive() {
			break
		}
		q := decimal.Min(l.quantidade, excesso)
		if _, err := tx.Exec(ctx,
			`UPDATE public.stock_lot_balances
			    SET quantity = quantity - $2, reserved_qty = LEAST(reserved_qty, quantity - $2),
			        last_movement_at = $3, updated_at = NOW()
			  WHERE id = $1`, l.id, q, m.CreatedAt); err != nil {
			return nil, fmt.Errorf("baixando o lote %s: %w", l.lote, err)
		}
		excesso = excesso.Sub(q)
		resumo = append(resumo, fmt.Sprintf("%s lote %s", entity.QuantidadeBR(q), l.lote))
	}
	return resumo, nil
}

// conferirItemEAlmoxarifado recusa o movimento de item ou almoxarifado que não
// é da empresa. As tabelas de estoque não têm chave estrangeira para eles: sem
// isto o lançamento criava saldo de um código inexistente, ou apontava para o
// almoxarifado de outra empresa.
func conferirItemEAlmoxarifado(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	var item, almox bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM items WHERE code=$2 AND enterprise_id=$1),
		        EXISTS (SELECT 1 FROM warehouse WHERE id=$3 AND enterprise_id=$1)`,
		enterpriseID, m.ItemCode, m.WarehouseID).Scan(&item, &almox); err != nil {
		return fmt.Errorf("conferindo o item e o almoxarifado do movimento: %w", err)
	}
	if !item {
		return errorsuc.NewValidationError("o item informado não existe nesta empresa")
	}
	if !almox {
		return errorsuc.NewValidationError("o almoxarifado informado não existe nesta empresa")
	}
	return nil
}

// conferirEndereco valida o endereço de um movimento quando o almoxarifado usa
// endereços cadastrados. Entrada só em endereço ativo; endereço bloqueado
// (inventário, avaria, quarentena) não recebe nem entrega. O ajuste de contagem
// e o estorno passam: é justamente o que acerta um endereço bloqueado.
func conferirEndereco(ctx context.Context, tx pgx.Tx, enterpriseID, warehouseID int64, endereco string, entrada bool, referencia *string) error {
	if endereco == "" {
		return nil
	}
	if referencia != nil {
		switch *referencia {
		case entity.ReferenceTypeInventory, entity.ReferenceTypeCycleCount, entity.ReferenceTypeReversal:
			return nil
		}
	}
	var usaCadastro, existe, ativo, bloqueado bool
	var motivo *string
	err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1 AND warehouse_id=$2),
		        a.address IS NOT NULL, COALESCE(a.is_active, FALSE), COALESCE(a.is_blocked, FALSE), a.block_reason
		   FROM (SELECT 1) x
		   LEFT JOIN manufacturing_warehouse_addresses a ON a.enterprise_id=$1 AND a.warehouse_id=$2 AND a.address=$3`,
		enterpriseID, warehouseID, endereco).Scan(&usaCadastro, &existe, &ativo, &bloqueado, &motivo)
	if err != nil {
		return fmt.Errorf("conferindo o endereço %s: %w", endereco, err)
	}
	if !usaCadastro {
		return nil
	}
	if bloqueado {
		detalhe := ""
		if motivo != nil && strings.TrimSpace(*motivo) != "" {
			detalhe = " (" + strings.TrimSpace(*motivo) + ")"
		}
		return errorsuc.NewValidationError(fmt.Sprintf("o endereço %s está bloqueado%s: desbloqueie-o ou use outro endereço", endereco, detalhe))
	}
	if entrada && (!existe || !ativo) {
		return errorsuc.NewValidationError(fmt.Sprintf("o endereço %s não está cadastrado ou está inativo neste almoxarifado", endereco))
	}
	return nil
}

// enderecosBloqueados: endereços do almoxarifado que não entregam material.
func enderecosBloqueados(ctx context.Context, tx pgx.Tx, enterpriseID, warehouseID int64) (map[string]bool, error) {
	rows, err := tx.Query(ctx,
		`SELECT address FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1 AND warehouse_id=$2 AND is_blocked`,
		enterpriseID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("lendo os endereços bloqueados: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, fmt.Errorf("lendo os endereços bloqueados: %w", err)
		}
		out[a] = true
	}
	return out, rows.Err()
}

// conferirBloqueioDeInventario recusa o movimento do item que está num
// inventário com bloqueio ligado, até ele terminar. O próprio ajuste passa.
func conferirBloqueioDeInventario(ctx context.Context, tx pgx.Tx, enterpriseID, warehouseID, itemCode int64, referencia *string) error {
	if referencia != nil && *referencia == entity.ReferenceTypeInventory {
		return nil
	}
	var codigo int64
	err := tx.QueryRow(ctx,
		`SELECT i.code FROM public.physical_inventories i
		  WHERE i.enterprise_id=$1 AND i.warehouse_id=$2 AND i.blocks_movements
		    AND i.status IN ('OPEN','EM_CONFERENCIA','AGUARDANDO_APROVACAO')
		    AND (NOT i.partial OR EXISTS (SELECT 1 FROM public.physical_inventory_items x WHERE x.inventory_id=i.id AND x.item_code=$3))
		  LIMIT 1`, enterpriseID, warehouseID, itemCode).Scan(&codigo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("conferindo o bloqueio de inventário: %w", err)
	}
	return errorsuc.NewValidationError(fmt.Sprintf(
		"o item está no inventário %d, que bloqueia a movimentação deste almoxarifado até o fechamento", codigo))
}

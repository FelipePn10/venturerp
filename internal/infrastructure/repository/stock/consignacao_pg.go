package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var _ repository.ConsignacaoEstoque = (*StockRepositorySQLC)(nil)

const consignacaoCols = `c.id, c.direction, c.partner_code, c.item_code, c.mask, c.lot, c.origin_warehouse_id, c.warehouse_id,
	c.quantity, c.settled_qty, c.returned_qty, c.unit_value, c.fiscal_exit_id, c.document, c.issue_date, c.due_date, c.status, c.notes,
	(SELECT COALESCE(SUM(b.quantity),0) FROM stock_consignment_billings b JOIN stock_consignment_settlements s ON s.id=b.settlement_id WHERE s.consignment_id=c.id)`

func scanConsignacao(row pgx.Row, extra ...any) (*entity.Consignacao, error) {
	var c entity.Consignacao
	dst := append([]any{&c.ID, &c.Direcao, &c.Parceiro, &c.ItemCode, &c.Mask, &c.Lote, &c.AlmoxOrigem, &c.Almox,
		&c.Quantidade, &c.Baixada, &c.Devolvida, &c.Valor, &c.NotaSaida, &c.Documento, &c.Emissao, &c.Prazo, &c.Status, &c.Notas, &c.Faturada}, extra...)
	if err := row.Scan(dst...); err != nil {
		return nil, err
	}
	return &c, nil
}

// AlmoxDeConsignacao lê o almoxarifado que guarda a mercadoria consignada.
func (r *StockRepositorySQLC) AlmoxDeConsignacao(ctx context.Context) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	p, err := lerParametrosDeEstoque(ctx, r.pool, empresa)
	if err != nil {
		return 0, err
	}
	if p.ConsignmentWarehouseID == nil {
		return 0, errorsuc.NewValidationError("informe o almoxarifado de consignação em Estoque › Parâmetros")
	}
	return *p.ConsignmentWarehouseID, nil
}

// ConferirAlmoxDeConsignacao recusa almoxarifado que conta para promessa: a
// mercadoria com o cliente não pode parecer disponível para venda.
func (r *StockRepositorySQLC) ConferirAlmoxDeConsignacao(ctx context.Context, almox int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var disponivel bool
	var local string
	err = r.pool.QueryRow(ctx, `SELECT w.disposition, w.location::text FROM warehouse w WHERE w.id=$1 AND w.enterprise_id=$2`, almox, empresa).Scan(&disponivel, &local)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewValidationError("o almoxarifado de consignação não existe nesta empresa")
	}
	if err != nil {
		return fmt.Errorf("lendo o almoxarifado de consignação: %w", err)
	}
	if disponivel {
		return errorsuc.NewValidationError("o almoxarifado de consignação precisa estar sem disponibilidade: a mercadoria com o cliente não pode ser prometida")
	}
	// Inspeção e rejeição travam a saída, e o trânsito é da transferência: a venda
	// consignada ficaria presa, com a nota já autorizada.
	switch local {
	case "INSPECAO", "REJEICAO", "TRANSITO":
		return errorsuc.NewValidationError("escolha um almoxarifado próprio para a consignação: inspeção, rejeição e trânsito têm outro uso")
	}
	return nil
}

// RegistrarRemessa leva a linha da nota ao almoxarifado de consignação e abre o saldo
// do cliente. A mercadoria segue nossa: não há custo de venda aqui.
func (r *StockRepositorySQLC) RegistrarRemessa(ctx context.Context, m entity.RemessaConsignada) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrindo a remessa em consignação: %w", err)
	}
	defer tx.Rollback(ctx)
	param, err := lerParametrosDeEstoque(ctx, tx, empresa)
	if err != nil {
		return err
	}
	if param.ConsignmentWarehouseID == nil {
		return errorsuc.NewValidationError("informe o almoxarifado de consignação em Estoque › Parâmetros")
	}
	var prazo *time.Time
	if param.ConsignmentDays != nil {
		p := m.Emissao.AddDate(0, 0, *param.ConsignmentDays)
		prazo = &p
	}
	ref := entity.ReferenceTypeConsignment
	if _, err := transferirTx(ctx, tx, empresa, entity.TransferenciaEstoque{
		ItemCode: m.ItemCode, Mask: m.Mask, DeAlmox: m.AlmoxOrigem, ParaAlmox: *param.ConsignmentWarehouseID, Quantidade: m.Quantidade,
		Lote: m.Lote, EnderecoDe: m.Endereco, LinhaDaOnda: m.LinhaDaOnda, Ator: m.Ator, ReferenciaTipo: &ref, ReferenciaCodigo: &m.NotaSaida,
	}, false); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO stock_consignments(enterprise_id,direction,partner_code,item_code,mask,lot,origin_warehouse_id,warehouse_id,quantity,unit_value,fiscal_exit_id,document,issue_date,due_date,created_by)
		 VALUES($1,'SAIDA',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		empresa, m.Cliente, m.ItemCode, m.Mask, m.Lote, m.AlmoxOrigem, *param.ConsignmentWarehouseID, m.Quantidade, m.ValorUnitario,
		m.NotaSaida, m.Documento, m.Emissao.Format("2006-01-02"), prazo, m.Ator); err != nil {
		return fmt.Errorf("gravando a remessa em consignação: %w", err)
	}
	return tx.Commit(ctx)
}

// SaldoConsignado soma o que o cliente tem do item. Com lote, só o que pode ser dele:
// a remessa daquele lote ou a que saiu sem lote.
func (r *StockRepositorySQLC) SaldoConsignado(ctx context.Context, cliente, item int64, mask string, lote *string) (decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	var q decimal.Decimal
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity-settled_qty-returned_qty),0) FROM stock_consignments
		  WHERE enterprise_id=$1 AND direction='SAIDA' AND status='ABERTA' AND partner_code=$2 AND item_code=$3 AND mask=$4
		    AND ($5::text IS NULL OR lot IS NULL OR lot=$5)`,
		empresa, cliente, item, mask, lote).Scan(&q)
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo o saldo consignado: %w", err)
	}
	return q, nil
}

// LotesConsignados reparte a quantidade pelos lotes das remessas do cliente, da mais
// antiga à mais nova. A nota sem lote não pode levar o lote de outro cliente, que
// está no mesmo almoxarifado.
func (r *StockRepositorySQLC) LotesConsignados(ctx context.Context, cliente, item int64, mask string, quantidade decimal.Decimal) ([]entity.LoteConsignado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT lot, quantity-settled_qty-returned_qty FROM stock_consignments
		  WHERE enterprise_id=$1 AND direction='SAIDA' AND status='ABERTA' AND partner_code=$2 AND item_code=$3 AND mask=$4
		  ORDER BY issue_date, id`, empresa, cliente, item, mask)
	if err != nil {
		return nil, fmt.Errorf("lendo os lotes do cliente: %w", err)
	}
	defer rows.Close()
	out := []entity.LoteConsignado{}
	resta := quantidade
	for rows.Next() {
		var l entity.LoteConsignado
		if err := rows.Scan(&l.Lote, &l.Quantidade); err != nil {
			return nil, fmt.Errorf("lendo o lote do cliente: %w", err)
		}
		if !resta.IsPositive() {
			continue
		}
		l.Quantidade = decimal.Min(l.Quantidade, resta)
		resta = resta.Sub(l.Quantidade)
		out = append(out, l)
	}
	return out, rows.Err()
}

// RegistrarVendaConsignada baixa as remessas do cliente, da mais antiga à mais nova.
func (r *StockRepositorySQLC) RegistrarVendaConsignada(ctx context.Context, v entity.VendaConsignada) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrindo a baixa da consignação: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx,
		`SELECT id, quantity-settled_qty-returned_qty FROM stock_consignments
		  WHERE enterprise_id=$1 AND direction='SAIDA' AND status='ABERTA' AND partner_code=$2 AND item_code=$3 AND mask=$4
		    AND ($5::text IS NULL OR lot IS NULL OR lot=$5)
		  ORDER BY lot IS NOT DISTINCT FROM $5 DESC, issue_date, id FOR UPDATE`, empresa, v.Cliente, v.ItemCode, v.Mask, v.Lote)
	if err != nil {
		return fmt.Errorf("lendo as remessas do cliente: %w", err)
	}
	type remessa struct {
		id    int64
		saldo decimal.Decimal
	}
	var remessas []remessa
	for rows.Next() {
		var m remessa
		if err := rows.Scan(&m.id, &m.saldo); err != nil {
			rows.Close()
			return fmt.Errorf("lendo a remessa: %w", err)
		}
		remessas = append(remessas, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	falta := v.Quantidade
	for _, m := range remessas {
		if !falta.IsPositive() {
			break
		}
		q := decimal.Min(falta, m.saldo)
		if err := baixarConsignacaoTx(ctx, tx, m.id, entity.BaixaConsignacaoVenda, q, v.Emissao, &v.NotaSaida, &v.Documento, v.Ator); err != nil {
			return err
		}
		falta = falta.Sub(q)
	}
	if falta.IsPositive() {
		var codigo string
		if err := tx.QueryRow(ctx, `SELECT COALESCE(NULLIF(business_code,''), code::text) FROM items WHERE enterprise_id=$1 AND code=$2`, empresa, v.ItemCode).Scan(&codigo); err != nil {
			return fmt.Errorf("lendo o item da consignação: %w", err)
		}
		return errorsuc.NewValidationError(fmt.Sprintf("o cliente não tem saldo consignado do item %s: faltam %s", codigo, entity.QuantidadeBR(falta)))
	}
	return tx.Commit(ctx)
}

// baixarConsignacaoTx grava a baixa e encerra a consignação que zerou.
func baixarConsignacaoTx(ctx context.Context, tx pgx.Tx, id int64, tipo string, q decimal.Decimal, data time.Time, nota *int64, documento *string, ator uuid.UUID) error {
	coluna := "settled_qty"
	if tipo == entity.BaixaConsignacaoDevolucao {
		coluna = "returned_qty"
	}
	if _, err := tx.Exec(ctx,
		`UPDATE stock_consignments SET `+coluna+`=`+coluna+`+$2, updated_at=NOW(),
		        status=CASE WHEN settled_qty+returned_qty+$2 >= quantity THEN 'ENCERRADA' ELSE status END
		  WHERE id=$1`, id, q); err != nil {
		return fmt.Errorf("baixando a consignação: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO stock_consignment_settlements(consignment_id,kind,quantity,settled_on,fiscal_exit_id,document,created_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7)`, id, tipo, q, data.Format("2006-01-02"), nota, documento, ator); err != nil {
		return fmt.Errorf("gravando a baixa da consignação: %w", err)
	}
	return nil
}

func lerConsignacaoTx(ctx context.Context, tx pgx.Tx, empresa, id int64) (*entity.Consignacao, error) {
	c, err := scanConsignacao(tx.QueryRow(ctx, `SELECT `+consignacaoCols+` FROM stock_consignments c WHERE c.id=$1 AND c.enterprise_id=$2 FOR UPDATE`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("consignação não encontrada")
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a consignação: %w", err)
	}
	return c, nil
}

// DevolverConsignacao: a remessa volta ao almoxarifado; o material do fornecedor sai
// só do controle, porque nunca entrou no estoque.
func (r *StockRepositorySQLC) DevolverConsignacao(ctx context.Context, b entity.BaixaConsignacao) (*entity.Consignacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	hoje, err := r.HojeDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo a devolução: %w", err)
	}
	defer tx.Rollback(ctx)
	c, err := lerConsignacaoTx(ctx, tx, empresa, b.ID)
	if err != nil {
		return nil, err
	}
	if err := c.Conferir(b.Quantidade); err != nil {
		return nil, err
	}
	if c.Direcao == entity.ConsignacaoSaida {
		destino := b.Almox
		if destino == 0 {
			destino = *c.AlmoxOrigem
		}
		if destino == *c.Almox {
			return nil, errorsuc.NewValidationError("a devolução precisa voltar para outro almoxarifado: este é o de consignação")
		}
		ref := entity.ReferenceTypeConsignment
		if _, err := transferirTx(ctx, tx, empresa, entity.TransferenciaEstoque{
			ItemCode: c.ItemCode, Mask: c.Mask, DeAlmox: *c.Almox, ParaAlmox: destino, Quantidade: b.Quantidade,
			Lote: c.Lote, Ator: b.Ator, ReferenciaTipo: &ref, ReferenciaCodigo: &c.ID,
		}, false); err != nil {
			return nil, err
		}
	}
	if err := baixarConsignacaoTx(ctx, tx, c.ID, entity.BaixaConsignacaoDevolucao, b.Quantidade, hoje, nil, b.Documento, b.Ator); err != nil {
		return nil, err
	}
	out, err := lerConsignacaoTx(ctx, tx, empresa, b.ID)
	if err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

// ConsumirConsignacao: o material do fornecedor passa a ser nosso, pelo valor combinado.
func (r *StockRepositorySQLC) ConsumirConsignacao(ctx context.Context, b entity.BaixaConsignacao) (*entity.Consignacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	hoje, err := r.HojeDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo o consumo: %w", err)
	}
	defer tx.Rollback(ctx)
	c, err := lerConsignacaoTx(ctx, tx, empresa, b.ID)
	if err != nil {
		return nil, err
	}
	if c.Direcao != entity.ConsignacaoEntrada {
		return nil, errorsuc.NewValidationError("só o material recebido do fornecedor é consumido; a remessa ao cliente baixa pela nota de venda")
	}
	if err := c.Conferir(b.Quantidade); err != nil {
		return nil, err
	}
	var almoxDaEmpresa bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM warehouse WHERE id=$1 AND enterprise_id=$2)`, b.Almox, empresa).Scan(&almoxDaEmpresa); err != nil {
		return nil, fmt.Errorf("conferindo o almoxarifado: %w", err)
	}
	if !almoxDaEmpresa {
		return nil, errorsuc.NewValidationError("o almoxarifado que recebe o material não existe nesta empresa")
	}
	ref := entity.ReferenceTypeConsignment
	if err := CreateMovementTx(ctx, tx, empresa, &entity.StockMovement{ItemCode: c.ItemCode, Mask: c.Mask, WarehouseID: b.Almox,
		MovementType: entity.MovementTypeIn, Quantity: b.Quantidade, UnitPrice: c.Valor, Lot: c.Lote,
		ReferenceType: &ref, ReferenceCode: &c.ID, CreatedBy: b.Ator}); err != nil {
		return nil, err
	}
	if err := baixarConsignacaoTx(ctx, tx, c.ID, entity.BaixaConsignacaoConsumo, b.Quantidade, hoje, nil, b.Documento, b.Ator); err != nil {
		return nil, err
	}
	out, err := lerConsignacaoTx(ctx, tx, empresa, b.ID)
	if err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

// FaturarConsumoTx roda na aprovação da nota de entrada: o que o fornecedor fatura do
// material já consumido não entra de novo no estoque. A nota segue gerando a conta a
// pagar. Devolve quanto da quantidade já estava no estoque por consumo.
func FaturarConsumoTx(ctx context.Context, tx pgx.Tx, empresa, fornecedor, item int64, quantidade decimal.Decimal, entrada int64) (decimal.Decimal, error) {
	rows, err := tx.Query(ctx,
		`SELECT s.id, s.quantity - COALESCE((SELECT SUM(b.quantity) FROM stock_consignment_billings b WHERE b.settlement_id=s.id),0)
		   FROM stock_consignment_settlements s
		   JOIN stock_consignments c ON c.id=s.consignment_id
		  WHERE c.enterprise_id=$1 AND c.direction='ENTRADA' AND c.partner_code=$2 AND c.item_code=$3 AND s.kind=$4 AND NOT s.reversed
		  ORDER BY s.settled_on, s.id FOR UPDATE OF s`, empresa, fornecedor, item, entity.BaixaConsignacaoConsumo)
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo o consumo da consignação: %w", err)
	}
	type pendente struct {
		id  int64
		qtd decimal.Decimal
	}
	var pendentes []pendente
	for rows.Next() {
		var p pendente
		if err := rows.Scan(&p.id, &p.qtd); err != nil {
			rows.Close()
			return decimal.Zero, fmt.Errorf("lendo o consumo da consignação: %w", err)
		}
		pendentes = append(pendentes, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return decimal.Zero, err
	}
	coberta := decimal.Zero
	for _, p := range pendentes {
		q := decimal.Min(p.qtd, quantidade.Sub(coberta))
		if !q.IsPositive() {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO stock_consignment_billings(enterprise_id,settlement_id,fiscal_entry_id,quantity) VALUES($1,$2,$3,$4)
			 ON CONFLICT (settlement_id,fiscal_entry_id) DO UPDATE SET quantity=stock_consignment_billings.quantity+EXCLUDED.quantity`,
			empresa, p.id, entrada, q); err != nil {
			return decimal.Zero, fmt.Errorf("gravando o faturamento da consignação: %w", err)
		}
		coberta = coberta.Add(q)
	}
	return coberta, nil
}

// DesfazerFaturamentoTx: cancelada a nota de entrada, o consumo volta a aguardar nota.
func DesfazerFaturamentoTx(ctx context.Context, tx pgx.Tx, empresa, entrada int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM stock_consignment_billings WHERE enterprise_id=$1 AND fiscal_entry_id=$2`, empresa, entrada); err != nil {
		return fmt.Errorf("desfazendo o faturamento da consignação: %w", err)
	}
	return nil
}

// ReceberConsignacao registra o material do fornecedor. Não mexe no estoque.
func (r *StockRepositorySQLC) ReceberConsignacao(ctx context.Context, c entity.Consignacao) (*entity.Consignacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var item, fornecedor bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM items WHERE enterprise_id=$1 AND code=$2), EXISTS (SELECT 1 FROM suppliers WHERE enterprise_id=$1 AND code=$3)`,
		empresa, c.ItemCode, c.Parceiro).Scan(&item, &fornecedor); err != nil {
		return nil, fmt.Errorf("conferindo a consignação: %w", err)
	}
	if !item {
		return nil, errorsuc.NewValidationError("item não existe nesta empresa")
	}
	if !fornecedor {
		return nil, errorsuc.NewValidationError("fornecedor não existe nesta empresa")
	}
	out, err := scanConsignacao(r.pool.QueryRow(ctx,
		`INSERT INTO stock_consignments AS c (enterprise_id,direction,partner_code,item_code,mask,lot,quantity,unit_value,document,issue_date,due_date,notes,created_by)
		 VALUES($1,'ENTRADA',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING `+consignacaoCols,
		empresa, c.Parceiro, c.ItemCode, c.Mask, c.Lote, c.Quantidade, c.Valor, c.Documento, c.Emissao.Format("2006-01-02"), c.Prazo, c.Notas, c.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("gravando a consignação recebida: %w", err)
	}
	return out, nil
}

// ListarConsignacoes lê as consignações com o nome do parceiro e do item.
func (r *StockRepositorySQLC) ListarConsignacoes(ctx context.Context, direcao string, soAbertas bool) ([]entity.Consignacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+consignacaoCols+`, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''),
		        COALESCE(CASE WHEN c.direction='SAIDA'
		          THEN (SELECT x.name FROM customers x WHERE x.code=c.partner_code AND x.enterprise_id=c.enterprise_id)
		          ELSE (SELECT s.name FROM suppliers s WHERE s.code=c.partner_code AND s.enterprise_id=c.enterprise_id) END, '')
		   FROM stock_consignments c
		   LEFT JOIN items i ON i.code=c.item_code AND i.enterprise_id=c.enterprise_id
		  WHERE c.enterprise_id=$1 AND ($2='' OR c.direction=$2) AND (NOT $3 OR c.status='ABERTA')
		  ORDER BY c.issue_date DESC, c.id DESC`, empresa, direcao, soAbertas)
	if err != nil {
		return nil, fmt.Errorf("listando as consignações: %w", err)
	}
	defer rows.Close()
	out := []entity.Consignacao{}
	for rows.Next() {
		var codigo, nome, parte string
		c, err := scanConsignacao(rows, &codigo, &nome, &parte)
		if err != nil {
			return nil, fmt.Errorf("lendo a consignação: %w", err)
		}
		c.Codigo, c.Nome, c.NomeParte = codigo, nome, parte
		out = append(out, *c)
	}
	return out, rows.Err()
}

// PodeCancelarNotaDeEstoque recusa, antes de ir à SEFAZ, a nota cujo estoque já foi
// adiante: remessa com venda ou devolução, transferência recebida no destino.
func (r *StockRepositorySQLC) PodeCancelarNotaDeEstoque(ctx context.Context, nota int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var baixada, recebida bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM stock_consignments WHERE enterprise_id=$1 AND fiscal_exit_id=$2 AND status<>'CANCELADA' AND settled_qty+returned_qty>0),
		        EXISTS (SELECT 1 FROM stock_cnpj_transfers WHERE origin_id=$1 AND fiscal_exit_id=$2 AND status='RECEBIDA')`,
		empresa, nota).Scan(&baixada, &recebida); err != nil {
		return fmt.Errorf("conferindo o estoque da nota: %w", err)
	}
	if baixada {
		return errorsuc.NewValidationError("esta remessa em consignação já tem venda ou devolução: cancele antes as notas de venda. Com devolução registrada, a remessa não pode mais ser cancelada")
	}
	if recebida {
		return errorsuc.NewValidationError("o destino já deu entrada nesta transferência: cancele a entrada antes de cancelar a nota")
	}
	return nil
}

// enderecoDaSaidaTx acha de que endereço a nota tirou o material: cancelada, ele volta
// para lá, e não para o saldo sem endereço do almoxarifado.
func enderecoDaSaidaTx(ctx context.Context, tx pgx.Tx, empresa int64, ref string, codigo, item int64, mask string, almox int64, lote *string) (*string, error) {
	var endereco *string
	err := tx.QueryRow(ctx,
		`SELECT NULLIF(address,'') FROM stock_movements
		  WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3 AND movement_type=$4 AND item_code=$5 AND mask=$6 AND warehouse_id=$7
		    AND lot IS NOT DISTINCT FROM $8 ORDER BY id LIMIT 1`,
		empresa, ref, codigo, entity.MovementTypeTransferOut, item, mask, almox, lote).Scan(&endereco)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lendo o endereço da saída: %w", err)
	}
	return endereco, nil
}

// devolverSeparacaoTx deixa a onda pronta para a nota que substituir a cancelada.
func devolverSeparacaoTx(ctx context.Context, tx pgx.Tx, empresa int64, ref string, codigo int64) error {
	if _, err := tx.Exec(ctx,
		`UPDATE stock_picking_wave_lines l SET invoiced_qty = GREATEST(l.invoiced_qty - x.q, 0)
		   FROM (SELECT wave_line_id, SUM(quantity) q FROM stock_movements
		          WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3 AND movement_type=$4 AND wave_line_id IS NOT NULL
		          GROUP BY wave_line_id) x
		  WHERE l.id = x.wave_line_id AND l.enterprise_id = $1`,
		empresa, ref, codigo, entity.MovementTypeTransferOut); err != nil {
		return fmt.Errorf("devolvendo a separação da nota: %w", err)
	}
	return nil
}

// DesfazerConsignacaoDaNotaTx acompanha o cancelamento da NF-e: a remessa volta à
// origem e a venda devolve o saldo às remessas. Repetir não desfaz de novo.
func DesfazerConsignacaoDaNotaTx(ctx context.Context, tx pgx.Tx, empresa, nota int64, ator uuid.UUID) error {
	rows, err := tx.Query(ctx, `SELECT `+consignacaoCols+` FROM stock_consignments c
		WHERE c.enterprise_id=$1 AND c.fiscal_exit_id=$2 AND c.status='ABERTA' FOR UPDATE`, empresa, nota)
	if err != nil {
		return fmt.Errorf("lendo as remessas da nota: %w", err)
	}
	var remessas []*entity.Consignacao
	for rows.Next() {
		c, err := scanConsignacao(rows)
		if err != nil {
			rows.Close()
			return fmt.Errorf("lendo a remessa: %w", err)
		}
		remessas = append(remessas, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	ref := entity.ReferenceTypeConsignment
	for _, c := range remessas {
		endereco, err := enderecoDaSaidaTx(ctx, tx, empresa, ref, nota, c.ItemCode, c.Mask, *c.AlmoxOrigem, c.Lote)
		if err != nil {
			return err
		}
		if _, err := transferirTx(ctx, tx, empresa, entity.TransferenciaEstoque{
			ItemCode: c.ItemCode, Mask: c.Mask, DeAlmox: *c.Almox, ParaAlmox: *c.AlmoxOrigem, Quantidade: c.Saldo(),
			Lote: c.Lote, EnderecoPara: endereco, Ator: ator, ReferenciaTipo: &ref, ReferenciaCodigo: &nota,
		}, false); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_consignments SET status='CANCELADA', updated_at=NOW() WHERE id=$1`, c.ID); err != nil {
			return fmt.Errorf("cancelando a remessa: %w", err)
		}
	}
	if len(remessas) > 0 {
		if err := devolverSeparacaoTx(ctx, tx, empresa, ref, nota); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx,
		`WITH desfeitas AS (
		     UPDATE stock_consignment_settlements s SET reversed=TRUE
		       FROM stock_consignments c
		      WHERE c.id=s.consignment_id AND c.enterprise_id=$1 AND s.fiscal_exit_id=$2 AND NOT s.reversed AND s.kind='VENDA'
		     RETURNING s.consignment_id, s.quantity)
		 UPDATE stock_consignments c SET settled_qty=c.settled_qty-d.q, status='ABERTA', updated_at=NOW()
		   FROM (SELECT consignment_id, SUM(quantity) q FROM desfeitas GROUP BY 1) d
		  WHERE c.id=d.consignment_id`, empresa, nota); err != nil {
		return fmt.Errorf("devolvendo o saldo da venda consignada: %w", err)
	}
	return nil
}

// ---------- Transferência entre CNPJs ----------

// EmpresaIrma acha o estabelecimento do mesmo titular pelo CNPJ da nota. Zero: o
// destinatário não é um estabelecimento nosso.
func (r *StockRepositorySQLC) EmpresaIrma(ctx context.Context, cnpj string) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	cnpj = soDigitos(cnpj)
	var proprio string
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(cnpj_empresa,'') FROM fiscal_configs WHERE enterprise_id=$1`, empresa).Scan(&proprio)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lendo o CNPJ da empresa: %w", err)
	}
	if !entity.MesmaRaizDeCNPJ(soDigitos(proprio), cnpj) {
		return 0, nil
	}
	// Duas empresas com o mesmo CNPJ é cadastro errado: escolher uma delas gravaria o
	// estoque de um cliente no outro.
	rows, err := r.pool.Query(ctx, `SELECT enterprise_id FROM fiscal_configs WHERE regexp_replace(cnpj_empresa,'\D','','g')=$1 AND enterprise_id<>$2 LIMIT 2`, cnpj, empresa)
	if err != nil {
		return 0, fmt.Errorf("procurando o estabelecimento de destino: %w", err)
	}
	defer rows.Close()
	var irmas []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("procurando o estabelecimento de destino: %w", err)
		}
		irmas = append(irmas, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("procurando o estabelecimento de destino: %w", err)
	}
	switch len(irmas) {
	case 0:
		return 0, nil
	case 1:
		return irmas[0], nil
	}
	return 0, errorsuc.NewValidationError("mais de uma empresa do sistema usa o CNPJ de destino desta nota: corrija o cadastro fiscal antes de emitir")
}

// ConferirTransitoEntreCNPJs barra antes da SEFAZ o que o envio recusaria depois.
func (r *StockRepositorySQLC) ConferirTransitoEntreCNPJs(ctx context.Context) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("conferindo o almoxarifado de trânsito: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = almoxDeTransitoTx(ctx, tx, empresa, 0)
	return err
}

// EnviarEntreCNPJs leva as linhas da nota ao trânsito da origem. O material continua
// no estoque da origem até o destino dar entrada na nota.
func (r *StockRepositorySQLC) EnviarEntreCNPJs(ctx context.Context, e entity.EnvioEntreCNPJs) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrindo a transferência: %w", err)
	}
	defer tx.Rollback(ctx)
	transito, err := almoxDeTransitoTx(ctx, tx, empresa, 0)
	if err != nil {
		return err
	}
	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO stock_cnpj_transfers(origin_id,destination_id,fiscal_exit_id,nfe_key,transit_warehouse,created_by)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, empresa, e.Destino, e.NotaSaida, e.Chave, transito, e.Ator).Scan(&id); err != nil {
		return fmt.Errorf("gravando a transferência: %w", err)
	}
	ref := entity.ReferenceTypeCNPJTransfer
	for _, l := range e.Linhas {
		if _, err := transferirTx(ctx, tx, empresa, entity.TransferenciaEstoque{
			ItemCode: l.ItemCode, Mask: l.Mask, DeAlmox: l.AlmoxDe, ParaAlmox: transito, Quantidade: l.Quantidade,
			Lote: l.Lote, EnderecoDe: l.Endereco, LinhaDaOnda: l.LinhaDaOnda, Ator: e.Ator, ReferenciaTipo: &ref, ReferenciaCodigo: &id,
		}, false); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO stock_cnpj_transfer_lines(transfer_id,origin_item,mask,lot,from_warehouse,quantity) VALUES($1,$2,$3,$4,$5,$6)`,
			id, l.ItemCode, l.Mask, l.Lote, l.AlmoxDe, l.Quantidade); err != nil {
			return fmt.Errorf("gravando a linha da transferência: %w", err)
		}
	}
	return tx.Commit(ctx)
}

type linhaTransferida struct {
	item int64
	mask string
	lote *string
	de   int64
	qtd  decimal.Decimal
}

func linhasTransferidasTx(ctx context.Context, tx pgx.Tx, id int64) ([]linhaTransferida, error) {
	rows, err := tx.Query(ctx, `SELECT origin_item, mask, lot, from_warehouse, quantity FROM stock_cnpj_transfer_lines WHERE transfer_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("lendo as linhas da transferência: %w", err)
	}
	defer rows.Close()
	var out []linhaTransferida
	for rows.Next() {
		var l linhaTransferida
		if err := rows.Scan(&l.item, &l.mask, &l.lote, &l.de, &l.qtd); err != nil {
			return nil, fmt.Errorf("lendo a linha da transferência: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ReceberEntreCNPJsTx roda na aprovação da nota de entrada do destino: a entrada já
// pôs o material no estoque dele, e aqui ele sai do trânsito da origem. É o único
// ponto em que uma empresa grava na outra, e só vale para a transferência registrada
// com esta empresa como destino e esta chave.
func ReceberEntreCNPJsTx(ctx context.Context, tx pgx.Tx, empresa int64, chave string, entrada int64, ator uuid.UUID) error {
	if chave == "" {
		return nil
	}
	var id, origem, transito int64
	err := tx.QueryRow(ctx,
		`SELECT id, origin_id, transit_warehouse FROM stock_cnpj_transfers
		  WHERE destination_id=$1 AND nfe_key=$2 AND status='EM_TRANSITO' FOR UPDATE`, empresa, chave).Scan(&id, &origem, &transito)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("procurando a transferência da nota: %w", err)
	}
	linhas, err := linhasTransferidasTx(ctx, tx, id)
	if err != nil {
		return err
	}
	ref := entity.ReferenceTypeCNPJTransfer
	for _, l := range linhas {
		if err := CreateMovementTx(ctx, tx, origem, &entity.StockMovement{ItemCode: l.item, Mask: l.mask, WarehouseID: transito,
			MovementType: entity.MovementTypeOut, Quantity: l.qtd, Lot: l.lote, ReferenceType: &ref, ReferenceCode: &id,
			DataAjustavel: true, CreatedBy: ator}); err != nil {
			return fmt.Errorf("baixando o trânsito da origem: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_cnpj_transfers SET status='RECEBIDA', fiscal_entry_id=$2, received_at=NOW() WHERE id=$1`, id, entrada); err != nil {
		return fmt.Errorf("encerrando a transferência: %w", err)
	}
	return nil
}

// DesfazerRecebimentoEntreCNPJsTx: cancelada a entrada, o material volta ao trânsito
// da origem pelo custo em que saiu. Só conta a baixa depois da última volta: a entrada
// refeita baixa de novo, e somar as duas criaria saldo no trânsito.
func DesfazerRecebimentoEntreCNPJsTx(ctx context.Context, tx pgx.Tx, empresa, entrada int64, ator uuid.UUID) error {
	var id, origem, transito int64
	err := tx.QueryRow(ctx,
		`SELECT id, origin_id, transit_warehouse FROM stock_cnpj_transfers
		  WHERE destination_id=$1 AND fiscal_entry_id=$2 AND status='RECEBIDA' FOR UPDATE`, empresa, entrada).Scan(&id, &origem, &transito)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("procurando a transferência da entrada: %w", err)
	}
	rows, err := tx.Query(ctx,
		`SELECT item_code, mask, lot, ABS(quantity), unit_price FROM stock_movements
		  WHERE enterprise_id=$1 AND warehouse_id=$2 AND reference_type=$3 AND reference_code=$4 AND movement_type=$5
		    AND id > COALESCE((SELECT MAX(d.id) FROM stock_movements d
		                        WHERE d.enterprise_id=$1 AND d.warehouse_id=$2 AND d.reference_type=$3 AND d.reference_code=$4 AND d.movement_type=$6),0)
		  ORDER BY id`,
		origem, transito, entity.ReferenceTypeCNPJTransfer, id, entity.MovementTypeOut, entity.MovementTypeIn)
	if err != nil {
		return fmt.Errorf("lendo a baixa do trânsito: %w", err)
	}
	var movs []*entity.StockMovement
	for rows.Next() {
		m := &entity.StockMovement{WarehouseID: transito, MovementType: entity.MovementTypeIn, DataAjustavel: true, CreatedBy: ator}
		if err := rows.Scan(&m.ItemCode, &m.Mask, &m.Lot, &m.Quantity, &m.UnitPrice); err != nil {
			rows.Close()
			return fmt.Errorf("lendo a baixa do trânsito: %w", err)
		}
		movs = append(movs, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	ref := entity.ReferenceTypeCNPJTransfer
	for _, m := range movs {
		m.ReferenceType, m.ReferenceCode = &ref, &id
		if err := CreateMovementTx(ctx, tx, origem, m); err != nil {
			return fmt.Errorf("devolvendo ao trânsito da origem: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_cnpj_transfers SET status='EM_TRANSITO', fiscal_entry_id=NULL, received_at=NULL WHERE id=$1`, id); err != nil {
		return fmt.Errorf("reabrindo a transferência: %w", err)
	}
	return nil
}

// DesfazerEnvioEntreCNPJsTx: cancelada a nota antes da entrada, o material volta do
// trânsito ao almoxarifado de origem.
func DesfazerEnvioEntreCNPJsTx(ctx context.Context, tx pgx.Tx, empresa, nota int64, ator uuid.UUID) error {
	var id, transito int64
	err := tx.QueryRow(ctx,
		`SELECT id, transit_warehouse FROM stock_cnpj_transfers WHERE origin_id=$1 AND fiscal_exit_id=$2 AND status='EM_TRANSITO' FOR UPDATE`,
		empresa, nota).Scan(&id, &transito)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("procurando a transferência da nota: %w", err)
	}
	linhas, err := linhasTransferidasTx(ctx, tx, id)
	if err != nil {
		return err
	}
	ref := entity.ReferenceTypeCNPJTransfer
	for _, l := range linhas {
		endereco, err := enderecoDaSaidaTx(ctx, tx, empresa, ref, id, l.item, l.mask, l.de, l.lote)
		if err != nil {
			return err
		}
		if _, err := transferirTx(ctx, tx, empresa, entity.TransferenciaEstoque{
			ItemCode: l.item, Mask: l.mask, DeAlmox: transito, ParaAlmox: l.de, Quantidade: l.qtd, Lote: l.lote, EnderecoPara: endereco, Ator: ator,
			ReferenciaTipo: &ref, ReferenciaCodigo: &id,
		}, false); err != nil {
			return err
		}
	}
	if err := devolverSeparacaoTx(ctx, tx, empresa, ref, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_cnpj_transfers SET status='CANCELADA' WHERE id=$1`, id); err != nil {
		return fmt.Errorf("cancelando a transferência: %w", err)
	}
	return nil
}

// TransferenciasEntreCNPJs lista o que a empresa enviou ou tem a receber.
func (r *StockRepositorySQLC) TransferenciasEntreCNPJs(ctx context.Context) ([]entity.TransferenciaEntreCNPJs, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.origin_id, COALESCE(o.razao_social,''), t.destination_id, COALESCE(d.razao_social,''), t.fiscal_exit_id, t.nfe_key, t.status,
		        CASE WHEN t.destination_id=$1 THEN t.fiscal_entry_id END, t.received_at, t.created_at
		   FROM stock_cnpj_transfers t
		   LEFT JOIN fiscal_configs o ON o.enterprise_id=t.origin_id
		   LEFT JOIN fiscal_configs d ON d.enterprise_id=t.destination_id
		  WHERE t.origin_id=$1 OR t.destination_id=$1 ORDER BY t.id DESC`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando as transferências: %w", err)
	}
	defer rows.Close()
	out := []entity.TransferenciaEntreCNPJs{}
	for rows.Next() {
		var t entity.TransferenciaEntreCNPJs
		if err := rows.Scan(&t.ID, &t.Origem, &t.NomeOrigem, &t.Destino, &t.NomeDest, &t.NotaSaida, &t.Chave, &t.Status, &t.Entrada, &t.RecebidaEm, &t.CriadaEm); err != nil {
			return nil, fmt.Errorf("lendo a transferência: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func soDigitos(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

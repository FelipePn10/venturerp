package impressao

import (
	"context"
	"errors"
	"fmt"
	"time"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/jackc/pgx/v5"
)

// ─── Proposta comercial ─────────────────────────────────────────────────────

func (c *Consultas) Proposta(ctx context.Context, t uc.Tenant, code int64) (*uc.Proposta, error) {
	p := &uc.Proposta{Codigo: code}
	var frete, seguro, desconto, condID string
	var clienteID int64
	var convertido *int64
	err := c.pool.QueryRow(ctx, `
		SELECT sq.quotation_number, sq.emission_date, sq.valid_until, sq.delivery_date, sq.status, COALESCE(sq.purchase_order_number,''),
		       COALESCE(r.name,''), COALESCE(st.code||' — '||st.description,''), COALESCE(pc.description,''), COALESCE(pc.id,0)::text,
		       COALESCE(sq.currency_code,'BRL'), COALESCE(sq.freight_type,''), COALESCE(sq.freight_value,0)::text,
		       COALESCE(sq.insurance_value,0)::text, COALESCE(sq.discount_value,0)::text, COALESCE(sq.notes,''), COALESCE(sq.obs_customer,''),
		       sq.converted_sales_order_code,
		       COALESCE(cu.id,0), COALESCE(cu.code,0), COALESCE(cu.name,''), COALESCE(cu.trade_name,''), COALESCE(cu.document_number,''), COALESCE(cu.state_registration,'')
		  FROM sales_quotations sq
		  LEFT JOIN customers cu ON cu.code = sq.customer_code AND cu.enterprise_id = $3
		  LEFT JOIN representatives r ON r.code = sq.representative_code AND r.enterprise_code = $2
		  LEFT JOIN sales_tables st ON st.code = sq.price_table_code AND st.enterprise_id = $3
		  LEFT JOIN payment_conditions pc ON pc.code = sq.payment_term_code AND pc.enterprise_id = $3
		 WHERE sq.code = $1 AND sq.enterprise_code = $2`, code, t.Code, t.ID).
		Scan(&p.Numero, &p.Emissao, &p.Validade, &p.Entrega, &p.Situacao, &p.PedidoCliente, &p.Representante, &p.Tabela, &p.Condicao, &condID,
			&p.Moeda, &p.TipoFrete, &frete, &seguro, &desconto, &p.Observacao, &p.ObservacaoCliente, &convertido,
			&clienteID, &p.Cliente.Codigo, &p.Cliente.Nome, &p.Cliente.Fantasia, &p.Cliente.CNPJCPF, &p.Cliente.IE)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("orçamento: %w", err)
	}
	p.Frete, p.Seguro, p.Desconto = d(frete), d(seguro), d(desconto)
	if convertido != nil {
		_ = c.pool.QueryRow(ctx, `SELECT order_number FROM sales_orders WHERE code=$1 AND enterprise_code=$2`, *convertido, t.Code).Scan(&p.Convertida)
	}
	var entrega string
	c.enderecosCliente(ctx, clienteID, &p.Cliente, &entrega)

	rows, err := c.pool.Query(ctx, `
		SELECT sqi.sequence, COALESCE(i.business_code, sqi.item_code::text), `+nomeComercial("i")+`,
		       COALESCE(fc.ncm,''), COALESCE(sqi.sales_uom,'UN'), sqi.requested_qty::text, COALESCE(sqi.attended_qty,0)::text, COALESCE(sqi.cancelled_qty,0)::text,
		       sqi.unit_price::text, COALESCE(sqi.discount_pct,0)::text, sqi.total_net::text, COALESCE(sqi.ipi_pct,0)::text,
		       COALESCE(sqi.total_ipi, sqi.total_net_with_ipi - sqi.total_net, 0)::text, COALESCE(sqi.total_st,0)::text,
		       sqi.delivery_date, COALESCE((i.engineering_weight->>'net')::numeric,0)::text, COALESCE((i.engineering_weight->>'gross')::numeric,0)::text,
		       COALESCE(sqi.notes,'')
		  FROM sales_quotation_items sqi
		  LEFT JOIN items i ON i.code = sqi.item_code AND i.enterprise_id = $2
		  LEFT JOIN fiscal_classifications fc ON fc.code::text = i.accounting_sale_fiscal_classification_code AND fc.enterprise_id = $2
		 WHERE sqi.sales_quotation_code = $1 AND sqi.is_active
		 ORDER BY sqi.sequence, sqi.code`, code, t.ID)
	if err != nil {
		return nil, fmt.Errorf("itens do orçamento: %w", err)
	}
	for rows.Next() {
		var it uc.ItemPedidoVenda
		var q, a, cc, pr, ds, tot, ipiP, ipi, st, pl, pb string
		if err := rows.Scan(&it.Seq, &it.Codigo, &it.Descricao, &it.NCM, &it.UM, &q, &a, &cc, &pr, &ds, &tot, &ipiP, &ipi, &st, &it.Entrega, &pl, &pb, &it.Situacao); err != nil {
			rows.Close()
			return nil, err
		}
		it.Quantidade, it.Atendida, it.Cancelada, it.Preco, it.DescontoPct = d(q), d(a), d(cc), d(pr), d(ds)
		it.Total, it.IPIPct, it.IPI, it.ST, it.PesoLiquido, it.PesoBruto = d(tot), d(ipiP), d(ipi), d(st), d(pl), d(pb)
		p.Itens = append(p.Itens, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lendo os dados do documento: %w", err)
	}
	if p.Parcelas, err = c.parcelasCondicao(ctx, condID); err != nil {
		return nil, err
	}
	if entrega != "" {
		p.ObservacaoCliente = joinNonEmpty("\n", p.ObservacaoCliente, "Entrega: "+entrega)
	}
	return p, nil
}

func joinNonEmpty(sep string, partes ...string) string {
	out := ""
	for _, s := range partes {
		if s == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += s
	}
	return out
}

func (c *Consultas) parcelasCondicao(ctx context.Context, condID string) ([]uc.Parcela, error) {
	if condID == "" || condID == "0" {
		return nil, nil
	}
	rows, err := c.pool.Query(ctx, `SELECT installment_number, due_days, COALESCE(percentage,0)::text, COALESCE(description,'')
	                                  FROM payment_condition_installments WHERE payment_condition_id = $1 AND is_active ORDER BY installment_number`, condID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uc.Parcela
	for rows.Next() {
		var p uc.Parcela
		var pct string
		if err := rows.Scan(&p.Numero, &p.Dias, &pct, &p.Descricao); err != nil {
			return nil, err
		}
		p.Percentual = d(pct)
		out = append(out, p)
	}
	return out, rows.Err()
}

// enderecosCliente preenche o endereço principal e devolve o de entrega.
func (c *Consultas) enderecosCliente(ctx context.Context, clienteID int64, p *uc.Parte, entrega *string) {
	if clienteID == 0 {
		return
	}
	rows, err := c.pool.Query(ctx, `SELECT address_type::text, COALESCE(street,''), COALESCE(number,''), COALESCE(complement,''), COALESCE(neighborhood,''),
	                                       COALESCE(city,''), COALESCE(uf,''), COALESCE(zip_code,'')
	                                  FROM customer_addresses WHERE customer_id=$1 ORDER BY is_default DESC, id`, clienteID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var tipo, rua, num, compl, bairro, cidade, uf, cep string
		if rows.Scan(&tipo, &rua, &num, &compl, &bairro, &cidade, &uf, &cep) != nil {
			return
		}
		end := endereco(rua, num, compl)
		if p.Endereco == "" && tipo != "ENTREGA" {
			p.Endereco, p.Bairro, p.Cidade, p.UF, p.CEP = end, bairro, cidade, uf, cep
		}
		if tipo == "ENTREGA" && *entrega == "" {
			*entrega = fmt.Sprintf("%s · %s · %s/%s · CEP %s", end, bairro, cidade, uf, cep)
		}
	}
}

// ─── Cotação de compra ──────────────────────────────────────────────────────

func (c *Consultas) Cotacao(ctx context.Context, t uc.Tenant, code int64) (*uc.Cotacao, error) {
	q := &uc.Cotacao{Codigo: code}
	err := c.pool.QueryRow(ctx, `SELECT emission_date, status, COALESCE(notes,'') FROM purchase_quotations
	                              WHERE code = $1 AND enterprise_code = $2 AND is_active`, code, t.Code).Scan(&q.Emissao, &q.Situacao, &q.Observacao)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cotação: %w", err)
	}
	fr, err := c.pool.Query(ctx, `
		SELECT s.id, s.code, s.name, COALESCE(s.trade_name,''), COALESCE(s.document_number,''), COALESCE(s.state_registration,''),
		       COALESCE((SELECT e.email FROM supplier_emails e WHERE e.supplier_id = s.id ORDER BY e.id LIMIT 1),''),
		       COALESCE((SELECT ph.number FROM supplier_phones ph WHERE ph.supplier_id = s.id ORDER BY ph.id LIMIT 1),'')
		  FROM purchase_quotation_suppliers qs
		  JOIN suppliers s ON s.code = qs.supplier_code AND s.enterprise_id = $2
		 WHERE qs.quotation_code = $1 ORDER BY s.name`, code, t.ID)
	if err != nil {
		return nil, fmt.Errorf("fornecedores da cotação: %w", err)
	}
	var ids []int64
	for fr.Next() {
		var p uc.Parte
		var id int64
		if err := fr.Scan(&id, &p.Codigo, &p.Nome, &p.Fantasia, &p.CNPJCPF, &p.IE, &p.Email, &p.Telefone); err != nil {
			fr.Close()
			return nil, err
		}
		ids = append(ids, id)
		q.Fornecedores = append(q.Fornecedores, p)
	}
	fr.Close()
	if err := fr.Err(); err != nil {
		return nil, fmt.Errorf("lendo os dados do documento: %w", err)
	}
	for i, id := range ids {
		c.enderecoFornecedor(ctx, id, &q.Fornecedores[i])
	}

	ir, err := c.pool.Query(ctx, `
		SELECT qi.id, qi.sequence, COALESCE(i.business_code, qi.item_code::text), `+nomeInterno("i")+`, COALESCE(qi.uom, i.supplies_purchase_uom, i.warehouse_unit_of_measurement, 'UN'),
		       qi.quantity::text, qi.delivery_date
		  FROM purchase_quotation_items qi
		  LEFT JOIN items i ON i.code = qi.item_code AND i.enterprise_id = $2
		 WHERE qi.quotation_code = $1 ORDER BY qi.sequence, qi.id`, code, t.ID)
	if err != nil {
		return nil, fmt.Errorf("itens da cotação: %w", err)
	}
	idx := map[int64]int{}
	for ir.Next() {
		var it uc.ItemCotacao
		var qt string
		var ent *time.Time
		if err := ir.Scan(&it.ID, &it.Seq, &it.Codigo, &it.Descricao, &it.UM, &qt, &ent); err != nil {
			ir.Close()
			return nil, err
		}
		it.Quantidade, it.Entrega = d(qt), ent
		idx[it.ID] = len(q.Itens)
		q.Itens = append(q.Itens, it)
	}
	ir.Close()
	if err := ir.Err(); err != nil {
		return nil, fmt.Errorf("lendo os dados do documento: %w", err)
	}

	pr, err := c.pool.Query(ctx, `
		SELECT qp.quotation_item_id, qp.supplier_code, qp.unit_price::text, COALESCE(qp.lead_time_days,0), COALESCE(pc.description,''),
		       COALESCE(qp.notes,''), qp.is_selected
		  FROM purchase_quotation_prices qp
		  JOIN purchase_quotation_items qi ON qi.id = qp.quotation_item_id AND qi.quotation_code = $1
		  LEFT JOIN payment_conditions pc ON pc.code = qp.payment_term_code AND pc.enterprise_id = $2
		 ORDER BY qp.quotation_item_id, qp.unit_price`, code, t.ID)
	if err != nil {
		return nil, fmt.Errorf("preços da cotação: %w", err)
	}
	defer pr.Close()
	for pr.Next() {
		var item int64
		var p uc.PrecoCotacao
		var v string
		if err := pr.Scan(&item, &p.Fornecedor, &v, &p.PrazoDias, &p.Condicao, &p.Obs, &p.Escolhido); err != nil {
			return nil, err
		}
		p.Unitario = d(v)
		if i, ok := idx[item]; ok {
			q.Itens[i].Precos = append(q.Itens[i].Precos, p)
		}
	}
	return q, pr.Err()
}

// ─── Ficha cadastral do cliente ─────────────────────────────────────────────

func (c *Consultas) FichaCliente(ctx context.Context, t uc.Tenant, code int64) (*uc.FichaCliente, error) {
	f := &uc.FichaCliente{}
	var id int64
	var limite, matriz string
	err := c.pool.QueryRow(ctx, `
		SELECT cu.id, cu.code, cu.name, COALESCE(cu.trade_name,''), COALESCE(cu.document_number,''), COALESCE(cu.state_registration,''),
		       COALESCE(cu.municipal_registration,''), COALESCE(cu.suframa_code,''), COALESCE(cu.website,''), cu.is_active, cu.blocked, COALESCE(cu.block_reason,''),
		       cu.created_at, COALESCE(cu.credit_limit,0)::text, COALESCE(cu.finalidade_compra,''),
		       COALESCE(ct.description,''), COALESCE(rg.description,''), COALESCE(ms.description,''),
		       COALESCE(pc.description,''), COALESCE(st.code||' — '||st.description,''), COALESCE(tr.name,''),
		       COALESCE((SELECT m.code||' — '||m.name FROM customers m WHERE m.code = cu.corporate_code AND m.enterprise_id = cu.enterprise_id AND m.code <> cu.code),'')
		  FROM customers cu
		  LEFT JOIN customer_types ct ON ct.id = cu.customer_type_id
		  LEFT JOIN regions rg ON rg.id = cu.region_id
		  LEFT JOIN market_segments ms ON ms.id = cu.market_segment_id
		  LEFT JOIN payment_conditions pc ON pc.id = cu.payment_condition_id
		  LEFT JOIN sales_tables st ON st.id = cu.sales_table_id
		  LEFT JOIN suppliers tr ON tr.id = cu.carrier_id
		 WHERE cu.code = $1 AND cu.enterprise_id = $2`, code, t.ID).
		Scan(&id, &f.Cliente.Codigo, &f.Cliente.Nome, &f.Cliente.Fantasia, &f.Cliente.CNPJCPF, &f.Cliente.IE, &f.InscricaoMunicipal, &f.Suframa, &f.Site,
			&f.Ativo, &f.Bloqueado, &f.MotivoBloqueio, &f.Cadastro, &limite, &f.Finalidade, &f.Tipo, &f.Regiao, &f.Segmento,
			&f.Condicao, &f.Tabela, &f.Transportadora, &matriz)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cliente: %w", err)
	}
	f.LimiteCredito, f.Matriz = d(limite), matriz

	ar, err := c.pool.Query(ctx, `SELECT address_type::text, COALESCE(street,''), COALESCE(number,''), COALESCE(complement,''), COALESCE(neighborhood,''),
	                                     COALESCE(city,''), COALESCE(uf,''), COALESCE(zip_code,''), is_default
	                                FROM customer_addresses WHERE customer_id = $1 ORDER BY is_default DESC, id`, id)
	if err != nil {
		return nil, err
	}
	for ar.Next() {
		var e uc.EnderecoCliente
		var rua, num, compl string
		if err := ar.Scan(&e.Tipo, &rua, &num, &compl, &e.Bairro, &e.Cidade, &e.UF, &e.CEP, &e.Padrao); err != nil {
			ar.Close()
			return nil, err
		}
		e.Endereco = endereco(rua, num, compl)
		if f.Cliente.Endereco == "" {
			f.Cliente.Endereco, f.Cliente.Bairro, f.Cliente.Cidade, f.Cliente.UF, f.Cliente.CEP = e.Endereco, e.Bairro, e.Cidade, e.UF, e.CEP
		}
		f.Enderecos = append(f.Enderecos, e)
	}
	ar.Close()
	if err := ar.Err(); err != nil {
		return nil, fmt.Errorf("lendo os dados do documento: %w", err)
	}

	cr, err := c.pool.Query(ctx, `SELECT COALESCE(name,''), COALESCE(position,''), COALESCE(email,''), COALESCE(phone,''), COALESCE(mobile,''), COALESCE(is_primary,false)
	                                FROM customer_contacts WHERE customer_id = $1 AND COALESCE(is_active,true) ORDER BY is_primary DESC, id`, id)
	if err != nil {
		return nil, err
	}
	for cr.Next() {
		var x uc.ContatoCliente
		if err := cr.Scan(&x.Nome, &x.Cargo, &x.Email, &x.Telefone, &x.Celular, &x.Principal); err != nil {
			cr.Close()
			return nil, err
		}
		if f.Cliente.Email == "" {
			f.Cliente.Email = x.Email
		}
		if f.Cliente.Telefone == "" {
			f.Cliente.Telefone = x.Telefone
		}
		f.Contatos = append(f.Contatos, x)
	}
	cr.Close()
	if err := cr.Err(); err != nil {
		return nil, fmt.Errorf("lendo os dados do documento: %w", err)
	}

	fr, err := c.pool.Query(ctx, `SELECT code, name, COALESCE(document_number,''), COALESCE(state_registration,'')
	                                FROM customers WHERE corporate_code = $1 AND code <> $1 AND enterprise_id = $2 ORDER BY code`, code, t.ID)
	if err != nil {
		return nil, err
	}
	for fr.Next() {
		var p uc.Parte
		if err := fr.Scan(&p.Codigo, &p.Nome, &p.CNPJCPF, &p.IE); err != nil {
			fr.Close()
			return nil, err
		}
		f.Filiais = append(f.Filiais, p)
	}
	fr.Close()
	if err := fr.Err(); err != nil {
		return nil, fmt.Errorf("lendo os dados do documento: %w", err)
	}

	// Posição comercial: títulos em aberto, pedidos em carteira e faturamento.
	var aberto, vencido, pedidos, fat string
	err = c.pool.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM contas_receber r WHERE r.cliente_id = $1 AND r.enterprise_id = $2 AND r.is_active AND r.status NOT IN ('RECEBIDO','CANCELADO','BAIXADO')),
		  (SELECT COALESCE(SUM(r.valor_bruto - COALESCE(r.valor_recebido,0)),0) FROM contas_receber r WHERE r.cliente_id = $1 AND r.enterprise_id = $2 AND r.is_active AND r.status NOT IN ('RECEBIDO','CANCELADO','BAIXADO'))::text,
		  (SELECT COALESCE(SUM(r.valor_bruto - COALESCE(r.valor_recebido,0)),0) FROM contas_receber r WHERE r.cliente_id = $1 AND r.enterprise_id = $2 AND r.is_active AND r.status NOT IN ('RECEBIDO','CANCELADO','BAIXADO') AND r.data_vencimento < CURRENT_DATE)::text,
		  (SELECT COUNT(*) FROM sales_orders so WHERE so.customer_code = $1 AND so.enterprise_code = $3 AND so.is_active AND so.status IN ('P','A')),
		  (SELECT COALESCE(SUM(so.total_net),0) FROM sales_orders so WHERE so.customer_code = $1 AND so.enterprise_code = $3 AND so.is_active AND so.status IN ('P','A'))::text,
		  (SELECT COALESCE(SUM(fx.valor_total),0) FROM fiscal_exits fx WHERE fx.customer_code = $1 AND fx.enterprise_id = $2 AND fx.status = 'AUTHORIZED' AND fx.data_emissao >= CURRENT_DATE - INTERVAL '12 months')::text`,
		code, t.ID, t.Code).Scan(&f.TitulosAbertos, &aberto, &vencido, &f.PedidosAbertos, &pedidos, &fat)
	if err != nil {
		return nil, fmt.Errorf("posição do cliente: %w", err)
	}
	f.EmAberto, f.Vencido, f.ValorPedidos, f.Faturado12m = d(aberto), d(vencido), d(pedidos), d(fat)
	var ult time.Time
	if err := c.pool.QueryRow(ctx, `SELECT data_emissao, numero_nf FROM fiscal_exits WHERE customer_code = $1 AND enterprise_id = $2 AND status = 'AUTHORIZED' ORDER BY data_emissao DESC, id DESC LIMIT 1`, code, t.ID).
		Scan(&ult, &f.UltimaNF); err == nil {
		f.UltimaCompra = &ult
	}
	return f, nil
}

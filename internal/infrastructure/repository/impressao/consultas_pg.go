// Package impressao lê os modelos dos documentos impressos com SQL direto.
// Leitura pura: nada aqui grava. Toda consulta filtra pela empresa — as
// tabelas comerciais (pedido de venda, representante) pela coluna
// enterprise_code (tenant.Code), as demais por enterprise_id (tenant.ID).
package impressao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Consultas struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Consultas { return &Consultas{pool: pool} }

var _ uc.Consultas = (*Consultas)(nil)

// d converte o texto de um numeric; vazio vira zero.
func d(s string) decimal.Decimal {
	v, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero
	}
	return v
}

func endereco(rua, numero, compl string) string {
	s := strings.TrimSpace(rua)
	if n := strings.TrimSpace(numero); n != "" {
		s += ", " + n
	}
	if c := strings.TrimSpace(compl); c != "" {
		s += " — " + c
	}
	return s
}

func (c *Consultas) Emitente(ctx context.Context, t uc.Tenant) (uc.Emitente, error) {
	var e uc.Emitente
	var rua, num, compl string
	err := c.pool.QueryRow(ctx, `
		SELECT COALESCE(razao_social,''), COALESCE(trade_name,''), COALESCE(cnpj_empresa,''), COALESCE(ie_empresa,''),
		       COALESCE(logradouro,''), COALESCE(numero,''), COALESCE(complemento,''), COALESCE(bairro,''),
		       COALESCE(municipio,''), COALESCE(uf_empresa,''), COALESCE(cep,''), COALESCE(telefone,''), COALESCE(email,''),
		       COALESCE(logo, ''::bytea), COALESCE(brand_color,'')
		  FROM fiscal_configs WHERE enterprise_id = $1 ORDER BY id LIMIT 1`, t.ID).
		Scan(&e.Nome, &e.Fantasia, &e.CNPJCPF, &e.IE, &rua, &num, &compl, &e.Bairro, &e.Cidade, &e.UF, &e.CEP, &e.Telefone, &e.Email, &e.Logo, &e.CorMarca)
	if errors.Is(err, pgx.ErrNoRows) {
		err = c.pool.QueryRow(ctx, `SELECT name FROM enterprise WHERE id=$1`, t.ID).Scan(&e.Nome)
		return e, err
	}
	e.Endereco = endereco(rua, num, compl)
	return e, err
}

func (c *Consultas) NomeUsuario(ctx context.Context, id uuid.UUID) string {
	var nome string
	_ = c.pool.QueryRow(ctx, `SELECT name FROM users WHERE id=$1`, id).Scan(&nome)
	return nome
}

// ─── Pedido de venda ────────────────────────────────────────────────────────

func (c *Consultas) PedidoVenda(ctx context.Context, t uc.Tenant, code int64) (*uc.PedidoVenda, error) {
	p := &uc.PedidoVenda{Codigo: code}
	var entrega *time.Time
	var comissao, frete, seguro, desconto, condID string
	var clienteID int64
	err := c.pool.QueryRow(ctx, `
		SELECT so.order_number, so.emission_date, so.delivery_date, so.delivery_date_firm, so.status, so.is_blocked, COALESCE(so.block_reason,''),
		       COALESCE(so.origin,''), COALESCE(so.sales_channel,''), COALESCE(sd.description,''),
		       COALESCE(r.name,''), COALESCE(so.commission_pct,0)::text, COALESCE(st.code||' — '||st.description,''),
		       COALESCE(pc.description,''), COALESCE(pc.id,0)::text, COALESCE(so.currency_code,'BRL'),
		       COALESCE(so.freight_type,''), COALESCE(so.freight_value,0)::text, COALESCE(so.insurance_value,0)::text, COALESCE(so.discount_value,0)::text,
		       COALESCE(so.notes,''), COALESCE(so.obs_customer,''), COALESCE(pp.code||' — '||pp.name,''),
		       COALESCE(cu.id,0), COALESCE(cu.code,0), COALESCE(cu.name,''), COALESCE(cu.trade_name,''), COALESCE(cu.document_number,''), COALESCE(cu.state_registration,''),
		       COALESCE(tr.name,'')
		  FROM sales_orders so
		  LEFT JOIN customers cu ON cu.code = so.customer_code AND cu.enterprise_id = $3
		  LEFT JOIN sales_divisions sd ON sd.id = so.sales_division_code AND sd.enterprise_id = $3
		  LEFT JOIN representatives r ON r.code = so.representative_code AND r.enterprise_code = $2
		  LEFT JOIN sales_tables st ON st.code = so.price_table_code AND st.enterprise_id = $3
		  LEFT JOIN payment_conditions pc ON pc.code = so.payment_term_code AND pc.enterprise_id = $3
		  LEFT JOIN production_plans pp ON pp.code = so.plan_code AND pp.enterprise_id = $3
		  LEFT JOIN suppliers tr ON tr.code = so.carrier_code AND tr.enterprise_id = $3
		 WHERE so.code = $1 AND so.enterprise_code = $2`, code, t.Code, t.ID).
		Scan(&p.Numero, &p.Emissao, &entrega, &p.EntregaFirme, &p.Situacao, &p.Bloqueado, &p.MotivoBloqueio,
			&p.Origem, &p.Canal, &p.Divisao, &p.Representante, &comissao, &p.Tabela,
			&p.Condicao, &condID, &p.Moeda, &p.TipoFrete, &frete, &seguro, &desconto,
			&p.Observacao, &p.ObservacaoCliente, &p.Plano,
			&clienteID, &p.Cliente.Codigo, &p.Cliente.Nome, &p.Cliente.Fantasia, &p.Cliente.CNPJCPF, &p.Cliente.IE, &p.Transportadora)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pedido de venda: %w", err)
	}
	p.Entrega = entrega
	p.ComissaoPct, p.Frete, p.Seguro, p.Desconto = d(comissao), d(frete), d(seguro), d(desconto)

	// Endereços: cobrança (ou o padrão) para o cliente; entrega quando houver.
	rows, err := c.pool.Query(ctx, `
		SELECT address_type::text, COALESCE(street,''), COALESCE(number,''), COALESCE(complement,''), COALESCE(neighborhood,''),
		       COALESCE(city,''), COALESCE(uf,''), COALESCE(zip_code,''), is_default
		  FROM customer_addresses WHERE customer_id = $1 ORDER BY is_default DESC, id`, clienteID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tipo, rua, num, compl, bairro, cidade, uf, cep string
		var padrao bool
		if err := rows.Scan(&tipo, &rua, &num, &compl, &bairro, &cidade, &uf, &cep, &padrao); err != nil {
			rows.Close()
			return nil, err
		}
		end := endereco(rua, num, compl)
		if p.Cliente.Endereco == "" && tipo != "ENTREGA" {
			p.Cliente.Endereco, p.Cliente.Bairro, p.Cliente.Cidade, p.Cliente.UF, p.Cliente.CEP = end, bairro, cidade, uf, cep
		}
		if tipo == "ENTREGA" && p.EnderecoEntrega == "" {
			p.EnderecoEntrega = fmt.Sprintf("%s · %s · %s/%s · CEP %s", end, bairro, cidade, uf, cep)
		}
	}
	rows.Close()

	rows, err = c.pool.Query(ctx, `
		SELECT soi.sequence, COALESCE(i.business_code, soi.item_code::text), COALESCE(NULLIF(i.commercial_description,''), i.name, ''),
		       COALESCE(fc.ncm,''), COALESCE(soi.sales_uom,'UN'),
		       soi.requested_qty::text, soi.attended_qty::text, soi.cancelled_qty::text, soi.unit_price::text, soi.discount_pct::text,
		       soi.total_net::text, soi.ipi_pct::text, COALESCE(soi.total_net_with_ipi - soi.total_net, 0)::text, soi.icms_pct::text,
		       COALESCE(soi.st_pct,0)::text, soi.pis_pct::text, soi.cofins_pct::text, soi.delivery_date, COALESCE(w.code,''),
		       COALESCE((i.engineering_weight->>'net')::numeric,0)::text, COALESCE((i.engineering_weight->>'gross')::numeric,0)::text,
		       COALESCE(soi.status,'')
		  FROM sales_order_items soi
		  LEFT JOIN items i ON i.code = soi.item_code AND i.enterprise_id = $2
		  LEFT JOIN fiscal_classifications fc ON fc.code::text = i.accounting_sale_fiscal_classification_code AND fc.enterprise_id = $2
		  LEFT JOIN warehouse w ON w.id = soi.warehouse_code AND w.enterprise_id = $2
		 WHERE soi.sales_order_code = $1 AND soi.is_active
		 ORDER BY soi.sequence, soi.code`, code, t.ID)
	if err != nil {
		return nil, fmt.Errorf("itens do pedido: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it uc.ItemPedidoVenda
		var q, a, cc, pr, ds, tot, ipiP, ipi, icms, st, pis, cof, pl, pb string
		if err := rows.Scan(&it.Seq, &it.Codigo, &it.Descricao, &it.NCM, &it.UM, &q, &a, &cc, &pr, &ds, &tot, &ipiP, &ipi, &icms, &st, &pis, &cof, &it.Entrega, &it.Deposito, &pl, &pb, &it.Situacao); err != nil {
			return nil, err
		}
		it.Quantidade, it.Atendida, it.Cancelada, it.Preco, it.DescontoPct = d(q), d(a), d(cc), d(pr), d(ds)
		it.Total, it.IPIPct, it.IPI, it.ICMSPct, it.PISPct, it.COFINSPct = d(tot), d(ipiP), d(ipi), d(icms), d(pis), d(cof)
		it.ST = it.Total.Mul(d(st)).Div(decimal.NewFromInt(100)).Round(2)
		it.PesoLiquido, it.PesoBruto = d(pl), d(pb)
		p.Itens = append(p.Itens, it)
	}
	rows.Close()

	if condID != "0" {
		pr, err := c.pool.Query(ctx, `
			SELECT installment_number, due_days, COALESCE(percentage,0)::text, COALESCE(description,'')
			  FROM payment_condition_installments WHERE payment_condition_id = $1 AND is_active ORDER BY installment_number`, condID)
		if err != nil {
			return nil, err
		}
		for pr.Next() {
			var par uc.Parcela
			var pct string
			if err := pr.Scan(&par.Numero, &par.Dias, &pct, &par.Descricao); err != nil {
				pr.Close()
				return nil, err
			}
			par.Percentual = d(pct)
			p.Parcelas = append(p.Parcelas, par)
		}
		pr.Close()
	}

	nr, err := c.pool.Query(ctx, `
		SELECT numero_nf, COALESCE(serie,''), data_emissao, valor_total::text, status, COALESCE(chave_acesso,'')
		  FROM fiscal_exits WHERE sales_order_code = $1 AND enterprise_id = $2 AND is_active ORDER BY id`, code, t.ID)
	if err != nil {
		return nil, err
	}
	defer nr.Close()
	for nr.Next() {
		var n uc.NotaVinculada
		var v string
		if err := nr.Scan(&n.Numero, &n.Serie, &n.Emissao, &v, &n.Situacao, &n.Chave); err != nil {
			return nil, err
		}
		n.Valor = d(v)
		p.Notas = append(p.Notas, n)
	}
	return p, nr.Err()
}

// ─── Nota de entrada ────────────────────────────────────────────────────────

func (c *Consultas) NotaEntrada(ctx context.Context, t uc.Tenant, id int64) (*uc.NotaEntrada, error) {
	n := &uc.NotaEntrada{ID: id}
	var prod, frete, seg, desc, outras, icms, st, ipi, pis, cof, total string
	var pedido *int64
	var aprovadoPor *uuid.UUID
	var supplierID int64
	err := c.pool.QueryRow(ctx, `
		SELECT fe.numero_nf, COALESCE(fe.serie,''), COALESCE(fe.modelo,'55'), COALESCE(fe.chave_acesso,''), COALESCE(fe.protocolo,''),
		       COALESCE(fe.natureza_operacao,''), fe.data_emissao, fe.data_entrada, fe.status,
		       COALESCE(s.id,0), COALESCE(s.code,0), COALESCE(s.name, fe.razao_social_emitente,''), COALESCE(s.trade_name,''),
		       COALESCE(fe.cnpj_emitente,''), COALESCE(fe.ie_emitente, s.state_registration,''), COALESCE(fe.uf_emitente,''),
		       fe.purchase_order_code, fe.valor_produtos::text, fe.valor_frete::text, fe.valor_seguro::text, fe.valor_desconto::text,
		       COALESCE(fe.valor_outras,0)::text, fe.valor_icms::text, COALESCE(fe.valor_icms_st,0)::text, fe.valor_ipi::text,
		       fe.valor_pis::text, fe.valor_cofins::text, fe.valor_total::text, COALESCE(fe.notes,''), fe.approved_at, fe.approved_by
		  FROM fiscal_entries fe
		  LEFT JOIN suppliers s ON s.code = fe.supplier_code AND s.enterprise_id = fe.enterprise_id
		 WHERE fe.id = $1 AND fe.enterprise_id = $2`, id, t.ID).
		Scan(&n.Numero, &n.Serie, &n.Modelo, &n.Chave, &n.Protocolo, &n.Natureza, &n.Emissao, &n.Entrada, &n.Situacao,
			&supplierID, &n.Fornecedor.Codigo, &n.Fornecedor.Nome, &n.Fornecedor.Fantasia, &n.Fornecedor.CNPJCPF, &n.Fornecedor.IE, &n.Fornecedor.UF,
			&pedido, &prod, &frete, &seg, &desc, &outras, &icms, &st, &ipi, &pis, &cof, &total, &n.Observacao, &n.AprovadaEm, &aprovadoPor)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("nota de entrada: %w", err)
	}
	if pedido != nil {
		n.PedidoCompra = *pedido
	}
	n.Produtos, n.Frete, n.Seguro, n.Desconto, n.Outras = d(prod), d(frete), d(seg), d(desc), d(outras)
	n.ICMS, n.ICMSST, n.IPI, n.PIS, n.COFINS, n.Total = d(icms), d(st), d(ipi), d(pis), d(cof), d(total)
	if aprovadoPor != nil {
		n.AprovadaPor = c.NomeUsuario(ctx, *aprovadoPor)
	}
	c.enderecoFornecedor(ctx, supplierID, &n.Fornecedor)

	rows, err := c.pool.Query(ctx, `
		SELECT fi.sequence, COALESCE(i.business_code, fi.item_code::text, ''), COALESCE(fi.description, i.name, ''), COALESCE(fi.ncm,''),
		       COALESCE(fi.cfop_entrada, fi.cfop, ''), COALESCE(fi.uom, i.warehouse_unit_of_measurement, ''),
		       COALESCE(fi.cst_icms,''), fi.quantity::text, fi.unit_price::text, fi.total_price::text,
		       fi.base_icms::text, fi.aliq_icms::text, fi.valor_icms::text, fi.aliq_ipi::text, fi.valor_ipi::text, fi.valor_pis::text, fi.valor_cofins::text,
		       fi.gera_credito_icms, fi.gera_credito_ipi, fi.gera_credito_pis, fi.gera_credito_cofins,
		       COALESCE(pc.codigo||' '||pc.descricao,''), COALESCE(w.code,''),
		       COALESCE(fi.purchase_order_code, poi.purchase_order_code, 0), COALESCE(poi.sequence,0),
		       COALESCE(poi.requested_qty,0)::text, COALESCE(poi.unit_price,0)::text,
		       COALESCE(fi.quantidade_estoque, fi.quantity)::text, COALESCE(fi.custo_aquisicao, sm.unit_price*sm.quantity, 0)::text
		  FROM fiscal_entry_items fi
		  LEFT JOIN items i ON i.code = fi.item_code AND i.enterprise_id = $2
		  LEFT JOIN plano_contas pc ON pc.id = fi.plano_contas_id AND pc.enterprise_id = $2
		  LEFT JOIN warehouse w ON w.id = fi.warehouse_id AND w.enterprise_id = $2
		  LEFT JOIN purchase_order_items poi ON poi.code = fi.purchase_order_item_code
		  LEFT JOIN stock_movements sm ON sm.id = fi.stock_movement_id
		 WHERE fi.fiscal_entry_id = $1 ORDER BY fi.sequence, fi.id`, id, t.ID)
	if err != nil {
		return nil, fmt.Errorf("itens da nota: %w", err)
	}
	for rows.Next() {
		var it uc.ItemNotaEntrada
		var q, u, tot, b, aI, vI, aP, vP, pis, cof, qp, pp, qe, custo string
		var linha int
		if err := rows.Scan(&it.Seq, &it.Codigo, &it.Descricao, &it.NCM, &it.CFOP, &it.UM, &it.CST, &q, &u, &tot, &b, &aI, &vI, &aP, &vP, &pis, &cof,
			&it.CreditaICMS, &it.CreditaIPI, &it.CreditaPIS, &it.CreditaCOFINS, &it.PlanoContas, &it.Deposito, &it.Pedido, &linha, &qp, &pp, &qe, &custo); err != nil {
			rows.Close()
			return nil, err
		}
		it.LinhaPedido = int64(linha)
		it.Quantidade, it.Unitario, it.Total, it.BaseICMS = d(q), d(u), d(tot), d(b)
		it.AliqICMS, it.ICMS, it.AliqIPI, it.IPI, it.PIS, it.COFINS = aliquota(d(aI)), d(vI), aliquota(d(aP)), d(vP), d(pis), d(cof)
		it.QtdPedido, it.PrecoPedido, it.QtdEstoque, it.CustoAquisicao = d(qp), d(pp), d(qe), d(custo)
		n.BaseICMS = n.BaseICMS.Add(it.BaseICMS)
		n.Itens = append(n.Itens, it)
	}
	rows.Close()

	pr, err := c.pool.Query(ctx, `
		SELECT fi.numero, COALESCE(NULLIF(fi.documento,''), cp.numero_documento, ''), fi.data_vencimento, fi.valor::text,
		       COALESCE(cp.status,''), COALESCE(cp.status_aprovacao,'')
		  FROM fiscal_entry_installments fi
		  LEFT JOIN contas_pagar cp ON cp.id = fi.conta_pagar_id
		 WHERE fi.fiscal_entry_id = $1 AND fi.enterprise_id = $2 ORDER BY fi.numero`, id, t.ID)
	if err != nil {
		return nil, err
	}
	for pr.Next() {
		var p uc.ParcelaNota
		var v, st, ap string
		if err := pr.Scan(&p.Numero, &p.Documento, &p.Vencimento, &v, &st, &ap); err != nil {
			pr.Close()
			return nil, err
		}
		p.Valor = d(v)
		p.Titulo = strings.TrimSpace(st + " " + rotuloAprovacao(ap))
		n.Parcelas = append(n.Parcelas, p)
	}
	pr.Close()
	if len(n.Parcelas) == 0 {
		// Notas antigas: as parcelas estão só no contas a pagar.
		cr, err := c.pool.Query(ctx, `SELECT COALESCE(parcela_numero,1), COALESCE(numero_documento,''), data_vencimento, valor_bruto::text, status, COALESCE(status_aprovacao,'')
		                                FROM contas_pagar WHERE fiscal_entry_id=$1 AND enterprise_id=$2 AND is_active ORDER BY parcela_numero, id`, id, t.ID)
		if err != nil {
			return nil, err
		}
		defer cr.Close()
		for cr.Next() {
			var p uc.ParcelaNota
			var v, st, ap string
			if err := cr.Scan(&p.Numero, &p.Documento, &p.Vencimento, &v, &st, &ap); err != nil {
				return nil, err
			}
			p.Valor, p.Titulo = d(v), strings.TrimSpace(st+" "+rotuloAprovacao(ap))
			n.Parcelas = append(n.Parcelas, p)
		}
	}
	return n, nil
}

// aliquota normaliza a alíquota para percentual: a nota digitada grava 19,5 e
// a importada do domínio fiscal grava 0,195.
func aliquota(v decimal.Decimal) decimal.Decimal {
	if v.IsPositive() && v.LessThan(decimal.NewFromInt(1)) {
		return v.Mul(decimal.NewFromInt(100))
	}
	return v
}

func rotuloAprovacao(s string) string {
	switch s {
	case "PENDENTE":
		return "· aprovação pendente"
	case "APROVADO":
		return "· aprovado"
	case "REJEITADO":
		return "· rejeitado"
	}
	return ""
}

func (c *Consultas) enderecoFornecedor(ctx context.Context, supplierID int64, p *uc.Parte) {
	if supplierID == 0 {
		return
	}
	var rua, num, compl string
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(street,''), COALESCE(number,''), COALESCE(complement,''), COALESCE(neighborhood,''),
	                                 COALESCE(city,''), COALESCE(uf,''), COALESCE(zip_code,'')
	                            FROM supplier_addresses WHERE supplier_id=$1 ORDER BY is_default DESC, id LIMIT 1`, supplierID).
		Scan(&rua, &num, &compl, &p.Bairro, &p.Cidade, &p.UF, &p.CEP)
	p.Endereco = endereco(rua, num, compl)
}

// ─── Títulos ────────────────────────────────────────────────────────────────

func (c *Consultas) Titulos(ctx context.Context, t uc.Tenant, receber bool, f uc.FiltroTitulos) ([]uc.Titulo, error) {
	var q string
	if receber {
		q = `SELECT cr.id, COALESCE(cr.numero_documento,''), COALESCE(cu.code,0), COALESCE(cu.name,''), COALESCE(cu.document_number,''),
		            cr.data_emissao, cr.data_vencimento, cr.data_recebimento, COALESCE(cr.parcela_numero,1), COALESCE(cr.parcela_total,1),
		            cr.valor_bruto::text, COALESCE(cr.desconto,0)::text, COALESCE(cr.juros,0)::text, COALESCE(cr.multa,0)::text, COALESCE(cr.valor_recebido,0)::text,
		            cr.status, '', COALESCE(pc.codigo||' '||pc.descricao,''), '',
		            CASE WHEN fx.id IS NOT NULL THEN 'NF-e '||fx.numero_nf||COALESCE(' · pedido '||so.order_number,'')
		                 WHEN so.code IS NOT NULL THEN 'Pedido '||so.order_number ELSE '' END,
		            COALESCE(cr.forma_pagamento,'')
		       FROM contas_receber cr
		       LEFT JOIN customers cu ON cu.code = cr.cliente_id AND cu.enterprise_id = cr.enterprise_id
		       LEFT JOIN plano_contas pc ON pc.id = cr.plano_contas_id
		       LEFT JOIN fiscal_exits fx ON fx.id = cr.fiscal_exit_id
		       LEFT JOIN sales_orders so ON so.code = COALESCE(cr.sales_order_id, fx.sales_order_code)
		      WHERE cr.enterprise_id = $1 AND cr.is_active
		        AND ($2::bigint IS NULL OR cr.cliente_id = $2)
		        AND ($3::bigint IS NULL OR cr.fiscal_exit_id = $3)`
	} else {
		q = `SELECT cp.id, COALESCE(cp.numero_documento,''), COALESCE(s.code,0), COALESCE(s.name,''), COALESCE(s.document_number, cp.fornecedor_cnpj,''),
		            cp.data_emissao, cp.data_vencimento, cp.data_pagamento, COALESCE(cp.parcela_numero,1), COALESCE(cp.parcela_total,1),
		            cp.valor_bruto::text, COALESCE(cp.desconto,0)::text, COALESCE(cp.juros,0)::text, COALESCE(cp.multa,0)::text, COALESCE(cp.valor_pago,0)::text,
		            cp.status, COALESCE(cp.status_aprovacao,''), COALESCE(pc.codigo||' '||pc.descricao,''), '',
		            CASE WHEN fe.id IS NOT NULL THEN 'NF '||fe.numero_nf||COALESCE(' · pedido '||fe.purchase_order_code,'')
		                 WHEN cp.purchase_order_id IS NOT NULL THEN 'Pedido '||cp.purchase_order_id ELSE COALESCE(cp.tipo_documento,'') END,
		            COALESCE(cp.forma_pagamento,'')
		       FROM contas_pagar cp
		       LEFT JOIN suppliers s ON s.code = cp.fornecedor_id AND s.enterprise_id = cp.enterprise_id
		       LEFT JOIN plano_contas pc ON pc.id = cp.plano_contas_id
		       LEFT JOIN fiscal_entries fe ON fe.id = cp.fiscal_entry_id
		      WHERE cp.enterprise_id = $1 AND cp.is_active
		        AND ($2::bigint IS NULL OR cp.fornecedor_id = $2)
		        AND ($3::bigint IS NULL OR cp.fiscal_entry_id = $3)`
	}
	tab := "cp"
	if receber {
		tab = "cr"
	}
	q += fmt.Sprintf(` AND ($4::date IS NULL OR %[1]s.data_vencimento >= $4) AND ($5::date IS NULL OR %[1]s.data_vencimento <= $5)
	                   AND (NOT $6 OR %[1]s.status NOT IN ('PAGO','RECEBIDO','CANCELADO','BAIXADO'))
	                 ORDER BY %[1]s.data_vencimento, 4, %[1]s.id`, tab)
	rows, err := c.pool.Query(ctx, q, t.ID, f.ParteID, f.NotaFiscalID, f.De, f.Ate, f.SomenteAbertos)
	if err != nil {
		return nil, fmt.Errorf("títulos: %w", err)
	}
	defer rows.Close()
	var out []uc.Titulo
	for rows.Next() {
		var x uc.Titulo
		var v, ds, j, m, pg string
		if err := rows.Scan(&x.ID, &x.Documento, &x.Parte.Codigo, &x.Parte.Nome, &x.Parte.CNPJCPF, &x.Emissao, &x.Vencimento, &x.Pagamento,
			&x.Parcela, &x.Parcelas, &v, &ds, &j, &m, &pg, &x.Situacao, &x.Aprovacao, &x.PlanoContas, &x.CentroCusto, &x.Origem, &x.Forma); err != nil {
			return nil, err
		}
		x.Valor, x.Desconto, x.Juros, x.Multa, x.Pago = d(v), d(ds), d(j), d(m), d(pg)
		out = append(out, x)
	}
	return out, rows.Err()
}

func (c *Consultas) ContasBancarias(ctx context.Context, t uc.Tenant) ([]uc.ContaBancaria, error) {
	rows, err := c.pool.Query(ctx, `SELECT COALESCE(banco,''), COALESCE(agencia,''), COALESCE(conta,'')||COALESCE('-'||NULLIF(digito,''),''),
	                                       COALESCE(titular,''), COALESCE(chave_pix,''), COALESCE(descricao,'')
	                                  FROM contas_bancarias WHERE enterprise_id=$1 AND is_active ORDER BY id`, t.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uc.ContaBancaria
	for rows.Next() {
		var b uc.ContaBancaria
		if err := rows.Scan(&b.Banco, &b.Agencia, &b.Conta, &b.Titular, &b.Pix, &b.Descricao); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ─── Plano de materiais ─────────────────────────────────────────────────────

func (c *Consultas) PlanoMateriais(ctx context.Context, t uc.Tenant, plano int64) (*uc.PlanoMateriais, error) {
	p := &uc.PlanoMateriais{Plano: plano}
	err := c.pool.QueryRow(ctx, `
		SELECT pp.name, l.finished_at, COALESCE(l.status,'NÃO CALCULADO'), COALESCE(l.errors::text,'')
		  FROM production_plans pp
		  LEFT JOIN LATERAL (SELECT finished_at, status, errors FROM mrp_calculation_logs
		                      WHERE plan_code = pp.code AND enterprise_id = pp.enterprise_id ORDER BY id DESC LIMIT 1) l ON TRUE
		 WHERE pp.code = $1 AND pp.enterprise_id = $2`, plano, t.ID).Scan(&p.Nome, &p.CalculadoEm, &p.Status, &p.Erros)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("plano: %w", err)
	}
	if p.Erros == "{}" {
		p.Erros = ""
	}

	// Demanda de origem: pedidos de venda em aberto e demandas independentes
	// ativas — o que o MRP leu.
	dr, err := c.pool.Query(ctx, `
		SELECT 'Pedido de venda', 'Pedido '||so.order_number, COALESCE(cu.name,''), COALESCE(i.business_code,''), COALESCE(i.name,''),
		       COALESCE(soi.sales_uom,'UN'), GREATEST(soi.requested_qty - soi.attended_qty - soi.cancelled_qty, soi.requested_qty)::text,
		       COALESCE(soi.delivery_date, so.delivery_date)
		  FROM sales_order_items soi
		  JOIN sales_orders so ON so.code = soi.sales_order_code
		  LEFT JOIN customers cu ON cu.code = so.customer_code AND cu.enterprise_id = $2
		  LEFT JOIN items i ON i.code = soi.item_code AND i.enterprise_id = $2
		 WHERE so.enterprise_code = $3 AND so.is_active AND soi.is_active
		   AND EXISTS (SELECT 1 FROM mrp_planned_suggestions s WHERE s.plan_code = $1 AND s.enterprise_id = $2 AND s.item_code = soi.item_code AND s.demand_type = 'SALES_ORDER')
		UNION ALL
		SELECT 'Demanda independente', 'Demanda '||idm.code, COALESCE(idm.motivo,''), COALESCE(i.business_code,''), COALESCE(i.name,''),
		       COALESCE(i.warehouse_unit_of_measurement,'UN'), idm.quantity::text, idm.demand_date
		  FROM independent_demands idm
		  LEFT JOIN items i ON i.code = idm.item_code AND i.enterprise_id = $2
		 WHERE idm.enterprise_id = $2 AND idm.is_active
		   AND EXISTS (SELECT 1 FROM mrp_planned_suggestions s WHERE s.plan_code = $1 AND s.enterprise_id = $2 AND s.item_code = idm.item_code AND s.demand_type = 'INDEPENDENT_DEMAND')
		 ORDER BY 8`, plano, t.ID, t.Code)
	if err != nil {
		return nil, fmt.Errorf("demandas: %w", err)
	}
	for dr.Next() {
		var x uc.DemandaPlano
		var q string
		if err := dr.Scan(&x.Origem, &x.Documento, &x.Cliente, &x.Codigo, &x.Descricao, &x.UM, &q, &x.Data); err != nil {
			dr.Close()
			return nil, err
		}
		x.Quantidade = d(q)
		p.Demandas = append(p.Demandas, x)
	}
	dr.Close()

	sr, err := c.pool.Query(ctx, `
		SELECT s.code, COALESCE(s.order_number,0), COALESCE(i.business_code, s.item_code::text), COALESCE(i.name,''), COALESCE(i.warehouse_unit_of_measurement,'UN'),
		       s.order_type, COALESCE(s.llc,0), COALESCE(pi.business_code,''), s.quantity::text, s.start_date, s.need_date, COALESCE(s.demand_type,''),
		       po.id IS NOT NULL OR poi.code IS NOT NULL OR COALESCE(pl.is_firm,false),
		       COALESCE('OF '||po.id, 'PC '||poi.purchase_order_code, ''),
		       COALESCE(sup.name,''), COALESCE(pr.price,0)::text,
		       COALESCE(rt.code||' alt. '||rt.alternative,''), COALESCE(rh.horas,0)::text
		  FROM mrp_planned_suggestions s
		  LEFT JOIN items i ON i.code = s.item_code AND i.enterprise_id = s.enterprise_id
		  LEFT JOIN items pi ON pi.code = s.parent_item_code AND pi.enterprise_id = s.enterprise_id
		  LEFT JOIN planned_orders pl ON pl.mrp_suggestion_code = s.code AND pl.enterprise_id = s.enterprise_id
		  LEFT JOIN production_orders po ON po.planned_order_id = pl.id AND po.enterprise_id = s.enterprise_id
		  LEFT JOIN LATERAL (SELECT x.code, x.purchase_order_code FROM purchase_order_items x WHERE x.planned_order_code = pl.code LIMIT 1) poi ON TRUE
		  LEFT JOIN LATERAL (SELECT ps.supplier_code FROM item_preferred_suppliers ps
		                      WHERE ps.item_code = s.item_code AND ps.enterprise_id = s.enterprise_id AND ps.is_active
		                      ORDER BY ps.is_preferred DESC, ps.ranking LIMIT 1) ips ON TRUE
		  LEFT JOIN suppliers sup ON sup.code = ips.supplier_code AND sup.enterprise_id = s.enterprise_id
		  LEFT JOIN LATERAL (SELECT ti.price FROM purchase_price_table_items ti JOIN purchase_price_tables tb ON tb.id = ti.table_id
		                      WHERE ti.item_code = s.item_code AND ti.is_active AND tb.enterprise_id = s.enterprise_id AND tb.is_active
		                        AND (tb.supplier_code IS NULL OR tb.supplier_code = ips.supplier_code)
		                      ORDER BY tb.validity_start DESC NULLS LAST LIMIT 1) pr ON TRUE
		  LEFT JOIN LATERAL (SELECT r.id, r.code, r.alternative FROM manufacturing_routes r
		                      WHERE r.item_code = s.item_code AND r.enterprise_id = s.enterprise_id AND r.is_active
		                      ORDER BY r.is_standard DESC, r.alternative LIMIT 1) rt ON s.order_type = 'FABRICACAO'
		  LEFT JOIN LATERAL (SELECT SUM(COALESCE(o.planned_hours,0) + COALESCE(o.setup_hours,0)) horas
		                       FROM production_order_operations o WHERE o.production_order_id = po.id) rh ON TRUE
		 WHERE s.plan_code = $1 AND s.enterprise_id = $2
		 ORDER BY s.llc, s.need_date, 3`, plano, t.ID)
	if err != nil {
		return nil, fmt.Errorf("sugestões: %w", err)
	}
	for sr.Next() {
		var x uc.SugestaoPlano
		var q, preco, horas string
		if err := sr.Scan(&x.Codigo, &x.Numero, &x.Item, &x.Descricao, &x.UM, &x.Tipo, &x.Nivel, &x.Pai, &q, &x.Inicio, &x.Necessidade, &x.Origem,
			&x.Firmada, &x.Gerou, &x.Fornecedor, &preco, &x.Roteiro, &horas); err != nil {
			sr.Close()
			return nil, err
		}
		x.Quantidade, x.PrecoReferencia, x.HorasPrevistas = d(q), d(preco), d(horas)
		p.Sugestoes = append(p.Sugestoes, x)
	}
	sr.Close()

	er, err := c.pool.Query(ctx, `
		SELECT COALESCE(i.business_code, e.item_code::text), e.message_type, COALESCE(e.description,''), e.suggested_date, e.handled_at IS NOT NULL
		  FROM mrp_exception_messages e LEFT JOIN items i ON i.code = e.item_code AND i.enterprise_id = e.enterprise_id
		 WHERE e.plan_code = $1 AND e.enterprise_id = $2 ORDER BY e.code`, plano, t.ID)
	if err != nil {
		return nil, err
	}
	defer er.Close()
	for er.Next() {
		var x uc.ExcecaoPlano
		if err := er.Scan(&x.Item, &x.Tipo, &x.Descricao, &x.Data, &x.Tratada); err != nil {
			return nil, err
		}
		p.Excecoes = append(p.Excecoes, x)
	}
	return p, er.Err()
}

// ─── Capacidade ─────────────────────────────────────────────────────────────

func (c *Consultas) Capacidade(ctx context.Context, t uc.Tenant, plano int64) (*uc.Capacidade, error) {
	p := &uc.Capacidade{Plano: plano, Maquinas: map[string][]string{}}
	if err := c.pool.QueryRow(ctx, `SELECT name FROM production_plans WHERE code=$1 AND enterprise_id=$2`, plano, t.ID).Scan(&p.Nome); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	rows, err := c.pool.Query(ctx, `
		SELECT COALESCE(mt.name, 'Centro '||cr.work_center_id), cr.req_date, cr.required_hours::text, cr.available_hours::text, cr.load_pct::text
		  FROM capacity_requirements cr
		  JOIN machine_types mt ON mt.id = cr.work_center_id
		 WHERE cr.plan_code = $1 AND (mt.enterprise_id = $2 OR mt.enterprise_id IS NULL)
		 ORDER BY 1, 2`, plano, t.ID)
	if err != nil {
		return nil, fmt.Errorf("carga: %w", err)
	}
	for rows.Next() {
		var x uc.CargaCentro
		var r, a, l string
		if err := rows.Scan(&x.Centro, &x.Data, &r, &a, &l); err != nil {
			rows.Close()
			return nil, err
		}
		x.Requerido, x.Disponivel, x.CargaPct = d(r), d(a), d(l)
		p.Cargas = append(p.Cargas, x)
	}
	rows.Close()

	// Ordens que formam a carga: operações das OFs firmadas pelo plano.
	or, err := c.pool.Query(ctx, `
		SELECT COALESCE(mt.name,''), 'OF '||po.id, COALESCE(i.business_code,''), o.operation_name, COALESCE(pl.start_date, pl.need_date, po.start_date, CURRENT_DATE),
		       (COALESCE(o.planned_hours,0) + COALESCE(o.setup_hours,0))::text
		  FROM production_orders po
		  JOIN planned_orders pl ON pl.id = po.planned_order_id AND pl.plan_code = $1
		  JOIN production_order_operations o ON o.production_order_id = po.id
		  LEFT JOIN machine_types mt ON mt.id = o.work_center_id
		  LEFT JOIN items i ON i.code = po.item_code AND i.enterprise_id = po.enterprise_id
		 WHERE po.enterprise_id = $2 AND po.is_active
		 ORDER BY 1, 5, 2, o.sequence`, plano, t.ID)
	if err != nil {
		return nil, err
	}
	for or.Next() {
		var x uc.OrdemNaCarga
		var h string
		if err := or.Scan(&x.Centro, &x.Ordem, &x.Item, &x.Operacao, &x.Data, &h); err != nil {
			or.Close()
			return nil, err
		}
		x.Horas = d(h)
		p.Ordens = append(p.Ordens, x)
	}
	or.Close()

	mr, err := c.pool.Query(ctx, `SELECT mt.name, m.name FROM machines m JOIN machine_types mt ON mt.code = m.machine_type_code
	                               WHERE m.enterprise_id = $1 AND m.is_active ORDER BY 1, 2`, t.ID)
	if err != nil {
		return nil, err
	}
	defer mr.Close()
	for mr.Next() {
		var centro, maq string
		if err := mr.Scan(&centro, &maq); err != nil {
			return nil, err
		}
		p.Maquinas[centro] = append(p.Maquinas[centro], maq)
	}
	return p, mr.Err()
}

// ─── Programação ────────────────────────────────────────────────────────────

func (c *Consultas) Programacao(ctx context.Context, t uc.Tenant, de, ate time.Time, grupo *int64) (*uc.Programacao, error) {
	p := &uc.Programacao{}
	if grupo != nil {
		_ = c.pool.QueryRow(ctx, `SELECT code||' — '||description FROM production_resource_groups WHERE id=$1 AND enterprise_id=$2`, *grupo, t.ID).Scan(&p.Grupo)
	}
	rows, err := c.pool.Query(ctx, `
		SELECT COALESCE(m.name, mt.name, 'Sem máquina'), COALESCE(mt.name,''), po.id, COALESCE(i.business_code,''), COALESCE(i.name,''),
		       COALESCE(o.sequence,0), COALESCE(o.operation_name,''), po.planned_qty::text, ps.scheduled_start, ps.scheduled_end,
		       COALESCE(o.status, ps.status), COALESCE(pl.need_date, po.end_date)
		  FROM production_sequences ps
		  JOIN production_orders po ON po.id = ps.production_order_id
		  LEFT JOIN planned_orders pl ON pl.id = po.planned_order_id
		  LEFT JOIN production_order_operations o ON o.id = ps.operation_id
		  LEFT JOIN machine_types mt ON mt.id = ps.work_center_id
		  LEFT JOIN machines m ON m.id = ps.machine_id
		  LEFT JOIN items i ON i.code = po.item_code AND i.enterprise_id = po.enterprise_id
		 WHERE ps.enterprise_id = $1 AND ps.scheduled_start < $3 AND ps.scheduled_end >= $2
		   AND ($4::bigint IS NULL OR m.resource_group_id = $4)
		 ORDER BY 1, ps.scheduled_start`, t.ID, de, ate.AddDate(0, 0, 1), grupo)
	if err != nil {
		return nil, fmt.Errorf("programação: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var x uc.OperacaoProgramada
		var q string
		if err := rows.Scan(&x.Maquina, &x.Centro, &x.Ordem, &x.Item, &x.Descricao, &x.Etapa, &x.Operacao, &q, &x.Inicio, &x.Fim, &x.Situacao, &x.Necessidade); err != nil {
			return nil, err
		}
		x.Quantidade = d(q)
		p.Operacoes = append(p.Operacoes, x)
	}
	return p, rows.Err()
}

// ─── Kardex ─────────────────────────────────────────────────────────────────

// sinal: tipos que somam no saldo. Os demais subtraem.
const entradasKardex = `('IN','ENTRADA','EP','DEV','AJUSTE_ENTRADA','TRANSFER_IN','RETORNO')`

func (c *Consultas) Kardex(ctx context.Context, t uc.Tenant, f uc.FiltroKardex) (*uc.Kardex, error) {
	k := &uc.Kardex{}
	if f.Deposito != nil {
		_ = c.pool.QueryRow(ctx, `SELECT code||' — '||description FROM warehouse WHERE id=$1 AND enterprise_id=$2`, *f.Deposito, t.ID).Scan(&k.Deposito)
	}
	fim := f.Ate.AddDate(0, 0, 1)
	rows, err := c.pool.Query(ctx, `
		WITH mov AS (
		  SELECT sm.*, (sm.movement_type IN `+entradasKardex+`) AS entrada
		    FROM stock_movements sm
		   WHERE sm.enterprise_id = $1 AND ($2::bigint IS NULL OR sm.warehouse_id = $2)
		     AND sm.quantity <> 0 AND sm.created_at < $4
		), itens AS (
		  SELECT DISTINCT m.item_code FROM mov m WHERE m.created_at >= $3
		)
		SELECT i.code, COALESCE(i.business_code, i.code::text), COALESCE(i.name,''), COALESCE(i.warehouse_unit_of_measurement,'UN'),
		       COALESCE((SELECT SUM(CASE WHEN m.entrada THEN m.quantity ELSE -m.quantity END) FROM mov m WHERE m.item_code = i.code AND m.created_at < $3),0)::text,
		       COALESCE((SELECT SUM(CASE WHEN m.entrada THEN m.quantity*m.unit_price ELSE -m.quantity*m.unit_price END) FROM mov m WHERE m.item_code = i.code AND m.created_at < $3),0)::text
		  FROM itens x JOIN items i ON i.code = x.item_code AND i.enterprise_id = $1
		 WHERE ($5 = '' OR i.business_code = $5 OR i.code::text = $5)
		 ORDER BY 2`, t.ID, f.Deposito, f.De, fim, strings.TrimSpace(f.Item))
	if err != nil {
		return nil, fmt.Errorf("kardex: %w", err)
	}
	type chave struct {
		code int64
		item uc.ItemKardex
	}
	var lista []chave
	for rows.Next() {
		var x chave
		var s, v string
		if err := rows.Scan(&x.code, &x.item.Codigo, &x.item.Descricao, &x.item.UM, &s, &v); err != nil {
			rows.Close()
			return nil, err
		}
		x.item.SaldoInicial, x.item.CustoInicial = d(s), d(v)
		lista = append(lista, x)
	}
	rows.Close()
	for _, x := range lista {
		mr, err := c.pool.Query(ctx, `
			SELECT sm.created_at, COALESCE(w.code,''), sm.movement_type, sm.movement_type IN `+entradasKardex+`, sm.quantity::text, sm.unit_price::text,
			       CASE sm.reference_type
			         WHEN 'NF_ENTRADA' THEN COALESCE('NF '||fe.numero_nf||' · '||LEFT(COALESCE(s.name,fe.razao_social_emitente,''),28), 'NF de entrada')
			         WHEN 'NF_SAIDA' THEN COALESCE('NF-e '||fx.numero_nf||' · '||LEFT(COALESCE(fx.razao_social_destinatario,''),28), 'NF-e de saída')
			         WHEN 'PRODUCTION_ORDER' THEN 'OF '||sm.reference_code
			         WHEN 'ADJUSTMENT' THEN 'Ajuste manual'
			         ELSE COALESCE(sm.reference_type,'')||COALESCE(' '||sm.reference_code,'') END,
			       COALESCE(sm.lot,''), COALESCE(sm.notes,'')
			  FROM stock_movements sm
			  LEFT JOIN warehouse w ON w.id = sm.warehouse_id
			  LEFT JOIN fiscal_entries fe ON sm.reference_type = 'NF_ENTRADA' AND fe.id = sm.reference_code AND fe.enterprise_id = sm.enterprise_id
			  LEFT JOIN suppliers s ON s.code = fe.supplier_code AND s.enterprise_id = fe.enterprise_id
			  LEFT JOIN fiscal_exits fx ON sm.reference_type = 'NF_SAIDA' AND fx.id = sm.reference_code AND fx.enterprise_id = sm.enterprise_id
			 WHERE sm.enterprise_id = $1 AND sm.item_code = $2 AND ($3::bigint IS NULL OR sm.warehouse_id = $3)
			   AND sm.quantity <> 0 AND sm.created_at >= $4 AND sm.created_at < $5
			 ORDER BY sm.created_at, sm.id`, t.ID, x.code, f.Deposito, f.De, fim)
		if err != nil {
			return nil, err
		}
		it := x.item
		saldo, valor := it.SaldoInicial, it.CustoInicial
		for mr.Next() {
			var m uc.MovimentoKardex
			var q, cst string
			if err := mr.Scan(&m.Data, &m.Deposito, &m.Tipo, &m.Entrada, &q, &cst, &m.Origem, &m.Lote, &m.Obs); err != nil {
				mr.Close()
				return nil, err
			}
			m.Quantidade, m.Custo = d(q), d(cst)
			if m.Entrada {
				saldo, valor = saldo.Add(m.Quantidade), valor.Add(m.Quantidade.Mul(m.Custo))
			} else {
				saldo, valor = saldo.Sub(m.Quantidade), valor.Sub(m.Quantidade.Mul(m.Custo))
			}
			it.Movimentos = append(it.Movimentos, m)
		}
		mr.Close()
		it.SaldoFinal = saldo
		if saldo.IsPositive() {
			it.CustoMedio = valor.Div(saldo).Round(4)
		}
		k.Itens = append(k.Itens, it)
	}
	return k, nil
}

// ─── Custo de produção ──────────────────────────────────────────────────────

func (c *Consultas) CustoProducao(ctx context.Context, t uc.Tenant, ordens []int64, de, ate *time.Time) ([]uc.CustoOrdem, error) {
	var fim *time.Time
	if ate != nil {
		x := ate.AddDate(0, 0, 1)
		fim = &x
	}
	rows, err := c.pool.Query(ctx, `
		SELECT po.id, COALESCE(i.business_code,''), COALESCE(i.name,''), COALESCE(i.warehouse_unit_of_measurement,'UN'),
		       COALESCE(pc.produced_qty, po.produced_qty)::text, po.start_date, po.end_date, po.status,
		       COALESCE(pc.material_cost_real,0)::text, COALESCE(pc.machine_cost_real,0)::text, COALESCE(pc.labor_cost_real,0)::text,
		       COALESCE(pc.overhead_cost_real,0)::text, COALESCE(pc.subcontract_cost_real,0)::text, COALESCE(pc.total_cost_real,0)::text,
		       COALESCE(pc.unit_cost_real,0)::text,
		       COALESCE(pc.material_cost_std,0)::text, COALESCE(pc.machine_cost_std,0)::text, COALESCE(pc.labor_cost_std,0)::text,
		       COALESCE(pc.overhead_cost_std,0)::text, COALESCE(pc.subcontract_cost_std,0)::text, COALESCE(pc.total_cost_std,0)::text,
		       pc.settled_at
		  FROM production_orders po
		  JOIN production_order_costs pc ON pc.production_order_id = po.id
		  LEFT JOIN items i ON i.code = po.item_code AND i.enterprise_id = po.enterprise_id
		 WHERE po.enterprise_id = $1
		   AND (cardinality($2::bigint[]) = 0 OR po.id = ANY($2))
		   AND ($3::timestamptz IS NULL OR pc.settled_at >= $3) AND ($4::timestamptz IS NULL OR pc.settled_at < $4)
		 ORDER BY pc.settled_at, po.id`, t.ID, ordens, de, fim)
	if err != nil {
		return nil, fmt.Errorf("custo de produção: %w", err)
	}
	var out []uc.CustoOrdem
	for rows.Next() {
		var o uc.CustoOrdem
		var v [14]string
		var q string
		if err := rows.Scan(&o.Ordem, &o.Item, &o.Descricao, &o.UM, &q, &o.Inicio, &o.Fim, &o.Situacao,
			&v[0], &v[1], &v[2], &v[3], &v[4], &v[5], &v[6], &v[7], &v[8], &v[9], &v[10], &v[11], &v[12], &o.Apurado); err != nil {
			rows.Close()
			return nil, err
		}
		o.Quantidade = d(q)
		o.Material, o.Maquina, o.MaoDeObra, o.Overhead, o.Terceiros, o.Total, o.Unitario = d(v[0]), d(v[1]), d(v[2]), d(v[3]), d(v[4]), d(v[5]), d(v[6])
		o.PadraoMaterial, o.PadraoMaquina, o.PadraoMaoDeObra, o.PadraoOverhead, o.PadraoTerceiros, o.PadraoTotal = d(v[7]), d(v[8]), d(v[9]), d(v[10]), d(v[11]), d(v[12])
		out = append(out, o)
	}
	rows.Close()
	for i := range out {
		o := &out[i]
		cr, err := c.pool.Query(ctx, `
			SELECT COALESCE(it.business_code,''), COALESCE(it.name,''), COALESCE(it.warehouse_unit_of_measurement,'UN'),
			       SUM(sm.quantity)::text, (SUM(sm.quantity*sm.unit_price))::text
			  FROM stock_movements sm LEFT JOIN items it ON it.code = sm.item_code AND it.enterprise_id = sm.enterprise_id
			 WHERE sm.enterprise_id = $1 AND sm.reference_type = 'PRODUCTION_ORDER' AND sm.reference_code = $2
			   AND sm.movement_type IN ('OUT','REP','REQ') AND sm.quantity <> 0
			 GROUP BY 1,2,3 ORDER BY 1`, t.ID, o.Ordem)
		if err != nil {
			return nil, err
		}
		for cr.Next() {
			var x uc.ConsumoOrdem
			var q, v string
			if err := cr.Scan(&x.Codigo, &x.Descricao, &x.UM, &q, &v); err != nil {
				cr.Close()
				return nil, err
			}
			x.Quantidade, x.Custo = d(q), d(v)
			o.Consumos = append(o.Consumos, x)
		}
		cr.Close()
		er, err := c.pool.Query(ctx, `
			SELECT o.sequence, o.operation_name, COALESCE(mt.name,''), (COALESCE(o.planned_hours,0)+COALESCE(o.setup_hours,0))::text,
			       COALESCE(o.actual_hours,0)::text, o.status
			  FROM production_order_operations o LEFT JOIN machine_types mt ON mt.id = o.work_center_id
			 WHERE o.production_order_id = $1 AND o.enterprise_id = $2 ORDER BY o.sequence`, o.Ordem, t.ID)
		if err != nil {
			return nil, err
		}
		for er.Next() {
			var x uc.EtapaOrdem
			var p, r string
			if err := er.Scan(&x.Seq, &x.Operacao, &x.Centro, &p, &r, &x.Situacao); err != nil {
				er.Close()
				return nil, err
			}
			x.Prevista, x.Real = d(p), d(r)
			o.Etapas = append(o.Etapas, x)
		}
		er.Close()
	}
	return out, nil
}

// ─── Romaneio ───────────────────────────────────────────────────────────────

func (c *Consultas) Romaneio(ctx context.Context, t uc.Tenant, code int64) (*uc.Romaneio, error) {
	r := &uc.Romaneio{Codigo: code}
	var id, clienteID, transpID int64
	var pedido *int64
	var frete, seguro, nfValor, pl, pb, cub string
	var nfNum *int64
	var vol *int
	var dRua, dNum, dCompl string
	err := c.pool.QueryRow(ctx, `
		SELECT s.id, s.status, s.created_at, s.separated_at, s.conferred_at, s.shipped_at, s.estimated_delivery, s.sales_order_code,
		       COALESCE(cu.id,0), COALESCE(cu.code,0), COALESCE(cu.name, fx.razao_social_destinatario, ''), COALESCE(cu.trade_name,''),
		       COALESCE(cu.document_number, fx.cnpj_destinatario, ''), COALESCE(cu.state_registration, fx.ie_destinatario, ''),
		       COALESCE(tr.id,0), COALESCE(tr.code,0), COALESCE(tr.name,''), COALESCE(tr.document_number,''), COALESCE(tr.state_registration,''),
		       COALESCE(s.vehicle_plate,''), COALESCE(s.driver_name,''), COALESCE(s.driver_document,''), COALESCE(s.antt_code,''), COALESCE(s.seals,''),
		       COALESCE(s.freight_modality, so.freight_type, ''), COALESCE(s.freight_value,0)::text, COALESCE(s.insurance_value,0)::text,
		       COALESCE(s.nfe_number, fx.numero_nf), COALESCE(fx.serie,''), COALESCE(s.nfe_key, fx.chave_acesso, ''), COALESCE(fx.valor_total,0)::text,
		       s.total_volumes, COALESCE(s.total_net_weight,0)::text, COALESCE(s.total_gross_weight, s.total_weight, 0)::text, COALESCE(s.total_cubage_m3,0)::text,
		       COALESCE(s.notes,''),
		       COALESCE(fx.dest_logradouro,''), COALESCE(fx.dest_numero,''), COALESCE(fx.dest_complemento,''), COALESCE(fx.dest_bairro,''),
		       COALESCE(fx.dest_municipio,''), COALESCE(fx.uf_destinatario,''), COALESCE(fx.dest_cep,'')
		  FROM shipments s
		  LEFT JOIN sales_orders so ON so.code = s.sales_order_code AND so.enterprise_code = $3
		  LEFT JOIN fiscal_exits fx ON fx.id = s.fiscal_exit_id AND fx.enterprise_id = s.enterprise_id
		  LEFT JOIN customers cu ON cu.code = COALESCE(so.customer_code, fx.customer_code) AND cu.enterprise_id = s.enterprise_id
		  LEFT JOIN suppliers tr ON tr.code = COALESCE(s.carrier_code, so.carrier_code) AND tr.enterprise_id = s.enterprise_id
		 WHERE s.code = $1 AND s.enterprise_id = $2`, code, t.ID, t.Code).
		Scan(&id, &r.Situacao, &r.Criado, &r.Separado, &r.Conferido, &r.Despachado, &r.PrevisaoEntrega, &pedido,
			&clienteID, &r.Destinatario.Codigo, &r.Destinatario.Nome, &r.Destinatario.Fantasia, &r.Destinatario.CNPJCPF, &r.Destinatario.IE,
			&transpID, &r.Transportadora.Codigo, &r.Transportadora.Nome, &r.Transportadora.CNPJCPF, &r.Transportadora.IE,
			&r.Placa, &r.Motorista, &r.DocMotorista, &r.ANTT, &r.Lacres, &r.ModalidadeFrete, &frete, &seguro,
			&nfNum, &r.NFSerie, &r.NFChave, &nfValor, &vol, &pl, &pb, &cub, &r.Observacao,
			&dRua, &dNum, &dCompl, &r.Destinatario.Bairro, &r.Destinatario.Cidade, &r.Destinatario.UF, &r.Destinatario.CEP)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("romaneio: %w", err)
	}
	r.Destinatario.Endereco = endereco(dRua, dNum, dCompl)
	if pedido != nil {
		r.PedidoVenda = *pedido
		_ = c.pool.QueryRow(ctx, `SELECT order_number FROM sales_orders WHERE code=$1 AND enterprise_code=$2`, *pedido, t.Code).Scan(&r.PedidoVenda)
	}
	if nfNum != nil {
		r.NFNumero = *nfNum
	}
	if vol != nil {
		r.Volumes = *vol
	}
	r.Frete, r.Seguro, r.NFValor, r.PesoLiquido, r.PesoBruto, r.Cubagem = d(frete), d(seguro), d(nfValor), d(pl), d(pb), d(cub)
	// Endereço completo: cadastro do cliente (cobrança/padrão) e entrega.
	if clienteID != 0 {
		ar, err := c.pool.Query(ctx, `SELECT address_type::text, COALESCE(street,''), COALESCE(number,''), COALESCE(complement,''), COALESCE(neighborhood,''),
		                                     COALESCE(city,''), COALESCE(uf,''), COALESCE(zip_code,'')
		                                FROM customer_addresses WHERE customer_id=$1 ORDER BY is_default DESC, id`, clienteID)
		if err == nil {
			for ar.Next() {
				var tipo, rua, num, compl, bairro, cidade, uf, cep string
				if ar.Scan(&tipo, &rua, &num, &compl, &bairro, &cidade, &uf, &cep) != nil {
					break
				}
				end := endereco(rua, num, compl)
				if r.Destinatario.Cidade == "" || r.Destinatario.Endereco == "" {
					r.Destinatario.Endereco, r.Destinatario.Bairro, r.Destinatario.Cidade, r.Destinatario.UF, r.Destinatario.CEP = end, bairro, cidade, uf, cep
				}
				if tipo == "ENTREGA" && r.EnderecoEntrega == "" {
					r.EnderecoEntrega = fmt.Sprintf("%s · %s · %s/%s · CEP %s", end, bairro, cidade, uf, cep)
				}
			}
			ar.Close()
		}
	}
	c.enderecoFornecedor(ctx, transpID, &r.Transportadora)

	rows, err := c.pool.Query(ctx, `
		SELECT si.sequence, COALESCE(i.business_code, si.item_code::text), COALESCE(NULLIF(i.commercial_description,''), i.name, ''),
		       COALESCE(i.warehouse_unit_of_measurement,'UN'), COALESCE(w.code,''), si.quantity::text, COALESCE(si.conferred_qty,0)::text, si.is_conferred,
		       COALESCE(NULLIF(si.unit_net_weight,0), (i.engineering_weight->>'net')::numeric, 0)::text,
		       COALESCE(NULLIF(si.unit_gross_weight,0), (i.engineering_weight->>'gross')::numeric, 0)::text
		  FROM shipment_items si
		  LEFT JOIN items i ON i.code = si.item_code AND i.enterprise_id = $2
		  LEFT JOIN warehouse w ON w.id = si.warehouse_id
		 WHERE si.shipment_id = $1 ORDER BY si.sequence, si.id`, id, t.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x uc.ItemRomaneio
		var q, cf, pl, pb string
		if err := rows.Scan(&x.Seq, &x.Codigo, &x.Descricao, &x.UM, &x.Deposito, &q, &cf, &x.Conferido, &pl, &pb); err != nil {
			rows.Close()
			return nil, err
		}
		x.Quantidade, x.Conferida, x.PesoLiquido, x.PesoBruto = d(q), d(cf), d(pl), d(pb)
		r.Itens = append(r.Itens, x)
	}
	rows.Close()

	vr, err := c.pool.Query(ctx, `
		SELECT volume_number, COALESCE(package_type,''), COALESCE(marking,''), COALESCE(contents,''), COALESCE(net_weight,0)::text, COALESCE(gross_weight,0)::text,
		       COALESCE(length_cm,0)::text, COALESCE(width_cm,0)::text, COALESCE(height_cm,0)::text, COALESCE(cubage_m3,0)::text
		  FROM shipment_volumes WHERE shipment_id = $1 ORDER BY volume_number`, id)
	if err != nil {
		return nil, err
	}
	defer vr.Close()
	for vr.Next() {
		var v uc.VolumeRomaneio
		var a, b, cc, dd, e, f string
		if err := vr.Scan(&v.Numero, &v.Embalagem, &v.Marcacao, &v.Conteudo, &a, &b, &cc, &dd, &e, &f); err != nil {
			return nil, err
		}
		v.PesoLiquido, v.PesoBruto, v.Comprimento, v.Largura, v.Altura, v.Cubagem = d(a), d(b), d(cc), d(dd), d(e), d(f)
		r.Embalagens = append(r.Embalagens, v)
	}
	return r, vr.Err()
}

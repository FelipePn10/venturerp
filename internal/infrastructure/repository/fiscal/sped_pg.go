package fiscal

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var _ repository.SpedRepository = (*FiscalRepositoryPG)(nil)

// NewSpedRepositoryPG expõe o repositório fiscal pelo contrato da EFD.
func NewSpedRepositoryPG(r repository.FiscalRepository) repository.SpedRepository {
	return r.(*FiscalRepositoryPG)
}

func (r *FiscalRepositoryPG) CarregarPeriodoEFD(ctx context.Context, inicio, fim time.Time) (*sped.DadosEFD, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Datas como texto: um time.Time vai como timestamptz e, convertido para
	// date no fuso do banco, cai no dia anterior.
	ini, fin := inicio.Format("2006-01-02"), fim.Format("2006-01-02")
	d := &sped.DadosEFD{}
	if err := r.empresaEFD(ctx, empresa, &d.Empresa); err != nil {
		return nil, err
	}
	if d.Entradas, err = r.entradasEFD(ctx, empresa, ini, fin); err != nil {
		return nil, err
	}
	if d.Saidas, d.Itens, err = r.saidasEFD(ctx, empresa, ini, fin); err != nil {
		return nil, err
	}
	if d.Fretes, err = r.fretesEFD(ctx, empresa, ini, fin); err != nil {
		return nil, err
	}
	if d.AjustesApuracao, err = r.ajustesEFD(ctx, empresa, inicio.Format("2006-01")); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *FiscalRepositoryPG) empresaEFD(ctx context.Context, empresa int64, e *sped.EFDEmpresa) error {
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(cnpj_empresa,''), COALESCE(razao_social,''), COALESCE(uf_empresa,''), COALESCE(ie_empresa,''),
		        COALESCE(codigo_municipio,''), COALESCE(trade_name,''), COALESCE(cep,''), COALESCE(logradouro,''),
		        COALESCE(numero,''), COALESCE(complemento,''), COALESCE(bairro,''), COALESCE(telefone,''), COALESCE(email,'')
		   FROM fiscal_configs WHERE enterprise_id=$1`, empresa).
		Scan(&e.CNPJ, &e.Nome, &e.UF, &e.IE, &e.CodigoMunicipio, &e.Fantasia, &e.CEP, &e.Endereco, &e.Numero,
			&e.Complemento, &e.Bairro, &e.Fone, &e.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewValidationError("a empresa não tem configuração fiscal: preencha CNPJ, IE, UF e município em VFIS0100")
	}
	if err != nil {
		return fmt.Errorf("configuração fiscal da empresa: %w", err)
	}
	e.CNPJ, e.IE, e.CEP, e.Fone = soDigitosRepo(e.CNPJ), soDigitosRepo(e.IE), soDigitosRepo(e.CEP), soDigitosRepo(e.Fone)
	return nil
}

// enderecoXML é o pedaço do XML (NF-e ou CT-e) que a EFD precisa do emitente:
// o código do município (0150) e, no CT-e, os municípios de origem e destino.
type enderecoXML struct {
	CNPJ, IE, CMun, Lgr, Nro, Cpl, Bairro string
	CMunIni, CMunFim                      string
}

func lerEnderecoEmitente(conteudo string) enderecoXML {
	if conteudo == "" {
		return enderecoXML{}
	}
	// O emitente e o ide ficam em profundidades diferentes (nfeProc/NFe/infNFe,
	// cteProc/CTe/infCte); procurar por elemento evita depender do envelope.
	dec := xml.NewDecoder(strings.NewReader(conteudo))
	var e enderecoXML
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "emit":
			var emit struct {
				CNPJ  string `xml:"CNPJ"`
				IE    string `xml:"IE"`
				Ender struct {
					Lgr    string `xml:"xLgr"`
					Nro    string `xml:"nro"`
					Cpl    string `xml:"xCpl"`
					Bairro string `xml:"xBairro"`
					CMun   string `xml:"cMun"`
				} `xml:"enderEmit"`
			}
			if dec.DecodeElement(&emit, &se) == nil && e.CMun == "" {
				e.CNPJ, e.IE, e.CMun = emit.CNPJ, emit.IE, emit.Ender.CMun
				e.Lgr, e.Nro, e.Cpl, e.Bairro = emit.Ender.Lgr, emit.Ender.Nro, emit.Ender.Cpl, emit.Ender.Bairro
			}
		case "cMunIni":
			_ = dec.DecodeElement(&e.CMunIni, &se)
		case "cMunFim":
			_ = dec.DecodeElement(&e.CMunFim, &se)
		}
	}
	return e
}

func participanteEFD(cnpj, nome, ie string, end enderecoXML) sped.Participante {
	doc := soDigitosRepo(cnpj)
	p := sped.Participante{Cod: doc, Nome: strings.TrimSpace(nome), IE: soDigitosRepo(firstNonEmptyStr(ie, end.IE)), CodMun: soDigitosRepo(end.CMun),
		End: strings.TrimSpace(end.Lgr), Num: strings.TrimSpace(end.Nro), Compl: strings.TrimSpace(end.Cpl), Bairro: strings.TrimSpace(end.Bairro)}
	if len(doc) == 11 {
		p.CPF = doc
	} else {
		p.CNPJ = doc
	}
	if p.Cod == "" {
		p.Cod = "SEMDOC"
	}
	return p
}

// As alíquotas das notas ficam como fração (0,12); a EFD as quer em % (12,00).

func (r *FiscalRepositoryPG) entradasEFD(ctx context.Context, empresa int64, ini, fin string) ([]sped.NotaEntrada, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, COALESCE(modelo,'55'), serie, numero_nf::text, COALESCE(chave_acesso,''), data_emissao, data_entrada,
		        cnpj_emitente, razao_social_emitente, COALESCE(ie_emitente,''), COALESCE(modalidade_frete,''), sem_pagamento,
		        valor_total, valor_produtos, valor_frete, valor_seguro, valor_desconto, valor_outras, valor_ipi,
		        valor_icms_st, valor_pis, valor_cofins, COALESCE(xml_content,'')
		   FROM fiscal_entries
		  WHERE enterprise_id=$1 AND is_active AND status IN ('APPROVED','WRITTEN_OFF')
		    AND modelo IN ('01','1B','04','55') AND data_entrada BETWEEN $2::date AND $3::date
		  ORDER BY data_entrada, id`, empresa, ini, fin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notas []sped.NotaEntrada
	var ids []int64
	for rows.Next() {
		var n sped.NotaEntrada
		var id int64
		var cnpj, nome, ie, modFrete, xmlc string
		var semPagamento bool
		if err := rows.Scan(&id, &n.Modelo, &n.Serie, &n.Numero, &n.Chave, &n.Emissao, &n.Entrada, &cnpj, &nome, &ie, &modFrete,
			&semPagamento, &n.Total, &n.Produtos, &n.Frete, &n.Seguro, &n.Desconto, &n.Outras, &n.IPI, &n.ST, &n.PIS, &n.COFINS, &xmlc); err != nil {
			return nil, err
		}
		n.Part = participanteEFD(cnpj, nome, ie, lerEnderecoEmitente(xmlc))
		n.IndFrt = modFrete
		if semPagamento {
			n.IndPgto = "9"
		}
		notas = append(notas, n)
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(ids) == 0 {
		return nil, nil
	}
	pos := make(map[int64]int, len(ids))
	for i, id := range ids {
		pos[id] = i
	}
	irows, err := r.pool.Query(ctx,
		`SELECT fi.fiscal_entry_id, fi.id, fi.sequence, fi.item_code, COALESCE(i.name, fi.description, ''), COALESCE(i.warehouse_unit_of_measurement::text,''),
		        i.engineering_type, i.supplies_type_of_use, COALESCE(i.commercial_sale_type,'') = 'REVENDA', COALESCE(i.commercial_is_packaging,false),
		        COALESCE(fi.ncm,''), COALESCE(fi.cest,''), COALESCE(fi.ean,''),
		        COALESCE(fi.description,''), COALESCE(fi.uom,''), fi.quantity, fi.fator_conversao,
		        fi.total_price, fi.valor_desconto, fi.valor_frete, fi.valor_seguro, fi.valor_outras, fi.valor_contabil,
		        COALESCE(fi.origem_mercadoria,''), COALESCE(fi.cst_icms,''), fi.cfop, COALESCE(fi.cfop_entrada,''),
		        fi.base_icms, fi.aliq_icms * 100, fi.valor_icms, fi.gera_credito_icms,
		        fi.base_icms_st, fi.valor_icms_st, fi.base_ipi, fi.aliq_ipi * 100, fi.valor_ipi, fi.gera_credito_ipi,
		        fi.valor_pis, fi.valor_cofins, fi.gera_credito_pis AND fi.gera_credito_cofins, fi.movimenta_estoque
		   FROM fiscal_entry_items fi
		   JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id AND fe.enterprise_id=$1
		   LEFT JOIN items i ON i.code = fi.item_code
		  WHERE fi.fiscal_entry_id = ANY($2)
		  ORDER BY fi.fiscal_entry_id, fi.sequence, fi.id`, empresa, ids)
	if err != nil {
		return nil, err
	}
	defer irows.Close()
	for irows.Next() {
		var entryID, itemID int64
		var it sped.ItemEntrada
		var itemCode *int64
		var engType, tipoUso *int16
		var revenda, embalagem bool
		var fator *decimal.Decimal
		var nomeItem, unidInv, ncm, cest, ean, origem, cst, cfop, cfopEntrada string
		var frete, seguro, outras decimal.Decimal
		if err := irows.Scan(&entryID, &itemID, &it.Seq, &itemCode, &nomeItem, &unidInv, &engType, &tipoUso, &revenda, &embalagem, &ncm, &cest, &ean,
			&it.Desc, &it.Unid, &it.Qtd, &fator, &it.Total, &it.Desconto, &frete, &seguro, &outras, &it.ValorContabil,
			&origem, &cst, &cfop, &cfopEntrada, &it.BaseICMS, &it.AliqICMS, &it.ICMS, &it.CreditaICMS,
			&it.BaseST, &it.ST, &it.BaseIPI, &it.AliqIPI, &it.IPI, &it.CreditaIPI,
			&it.PIS, &it.COFINS, &it.CreditaPISCOFINS, &it.MovEstoque); err != nil {
			return nil, err
		}
		if fator != nil {
			it.Fator = *fator
		}
		it.CFOP = firstNonEmptyStr(cfopEntrada, sped.CFOPEntrada(cfop))
		it.CST = sped.CSTICMS(origem, cst)
		if it.ValorContabil.IsZero() {
			it.ValorContabil = it.Total.Sub(it.Desconto).Add(frete).Add(seguro).Add(outras).Add(it.IPI).Add(it.ST)
		}
		// PIS/COFINS: a nota traz o valor; a base é o valor do item líquido do
		// desconto e a alíquota sai dela.
		base := it.Total.Sub(it.Desconto)
		if it.PIS.IsPositive() && base.IsPositive() {
			it.BasePIS, it.AliqPIS = base, it.PIS.Mul(decimal.NewFromInt(100)).Div(base).Round(4)
		}
		if it.COFINS.IsPositive() && base.IsPositive() {
			it.BaseCOFINS, it.AliqCOFINS = base, it.COFINS.Mul(decimal.NewFromInt(100)).Div(base).Round(4)
		}
		cod := fmt.Sprintf("NFE%d", itemID) // item sem cadastro: o código é a linha da nota
		if itemCode != nil {
			cod = fmt.Sprint(*itemCode)
		}
		it.Item = sped.ItemCadastro{Cod: cod, Desc: firstNonEmptyStr(nomeItem, it.Desc, cod), UnidInv: unidInv,
			Tipo: sped.TipoItem(cadastroDoItem(engType, tipoUso, revenda, embalagem), it.CFOP), NCM: soDigitosRepo(ncm), CEST: soDigitosRepo(cest), CodBarra: ean}
		i := pos[entryID]
		notas[i].Itens = append(notas[i].Itens, it)
	}
	return notas, irows.Err()
}

func (r *FiscalRepositoryPG) saidasEFD(ctx context.Context, empresa int64, ini, fin string) ([]sped.NotaSaida, []sped.ItemCadastro, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, serie, numero_nf::text, COALESCE(chave_acesso,''), data_emissao, COALESCE(data_saida, data_emissao), status,
		        COALESCE(cnpj_destinatario,''), COALESCE(razao_social_destinatario,''), COALESCE(ie_destinatario,''),
		        COALESCE(dest_codigo_municipio,''), COALESCE(dest_logradouro,''), COALESCE(dest_numero,''),
		        COALESCE(dest_complemento,''), COALESCE(dest_bairro,''), finalidade, condicao_pagamento_id IS NOT NULL,
		        valor_total, valor_produtos, valor_frete, valor_seguro, valor_desconto, valor_ipi, valor_icms_st,
		        base_icms_st, valor_pis, valor_cofins
		   FROM fiscal_exits
		  WHERE enterprise_id=$1 AND status IN ('AUTHORIZED','CANCELLED') AND COALESCE(chave_acesso,'') <> ''
		    AND data_emissao BETWEEN $2::date AND $3::date
		  ORDER BY serie, numero_nf`, empresa, ini, fin)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var notas []sped.NotaSaida
	var ids []int64
	for rows.Next() {
		var n sped.NotaSaida
		var id int64
		var status, cnpj, nome, ie string
		var end enderecoXML
		var finalidade int16
		var aPrazo bool
		if err := rows.Scan(&id, &n.Serie, &n.Numero, &n.Chave, &n.Emissao, &n.Saida, &status, &cnpj, &nome, &ie,
			&end.CMun, &end.Lgr, &end.Nro, &end.Cpl, &end.Bairro, &finalidade, &aPrazo,
			&n.Total, &n.Produtos, &n.Frete, &n.Seguro, &n.Desconto, &n.IPI, &n.ST, &n.BaseST, &n.PIS, &n.COFINS); err != nil {
			return nil, nil, err
		}
		n.Modelo = "55"
		n.Cancelada = status == "CANCELLED"
		n.Part = participanteEFD(cnpj, nome, ie, end)
		switch {
		case finalidade == 4:
			n.IndPgto = "9" // devolução: sem pagamento
		case aPrazo:
			n.IndPgto = "1"
		default:
			n.IndPgto = "0"
		}
		n.IndFrt = "9"
		if n.Frete.IsPositive() {
			n.IndFrt = "0"
		}
		notas = append(notas, n)
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	rows.Close()
	if len(ids) == 0 {
		return nil, nil, nil
	}
	pos := make(map[int64]int, len(ids))
	for i, id := range ids {
		pos[id] = i
	}
	irows, err := r.pool.Query(ctx,
		`SELECT xi.fiscal_exit_id, xi.item_code, COALESCE(i.name, xi.description, ''), COALESCE(i.warehouse_unit_of_measurement::text, xi.unidade_comercial, ''),
		        i.engineering_type, i.supplies_type_of_use, COALESCE(i.commercial_sale_type,'') = 'REVENDA', COALESCE(i.commercial_is_packaging,false),
		        COALESCE(xi.ncm,''), COALESCE(i.accounting_cest,''),
		        xi.origem_mercadoria, COALESCE(xi.cst_icms,''), xi.cfop, COALESCE(xi.cst_ipi,''),
		        xi.aliq_icms * 100, xi.base_icms, xi.valor_icms, xi.base_icms_st, xi.valor_icms_st, xi.base_ipi, xi.valor_ipi, xi.total_price
		   FROM fiscal_exit_items xi
		   JOIN fiscal_exits x ON x.id = xi.fiscal_exit_id AND x.enterprise_id=$1
		   LEFT JOIN items i ON i.code = xi.item_code
		  WHERE xi.fiscal_exit_id = ANY($2)
		  ORDER BY xi.fiscal_exit_id, xi.sequence, xi.id`, empresa, ids)
	if err != nil {
		return nil, nil, err
	}
	defer irows.Close()
	vistos := map[string]bool{}
	var itens []sped.ItemCadastro
	for irows.Next() {
		var exitID int64
		var itemCode *int64
		var engType, tipoUso *int16
		var revenda, embalagem bool
		var nomeItem, unid, ncm, cest, origem, cst string
		var it sped.ItemSaida
		if err := irows.Scan(&exitID, &itemCode, &nomeItem, &unid, &engType, &tipoUso, &revenda, &embalagem, &ncm, &cest, &origem, &cst, &it.CFOP, &it.CSTIPI,
			&it.AliqICMS, &it.BaseICMS, &it.ICMS, &it.BaseST, &it.ST, &it.BaseIPI, &it.IPI, &it.Total); err != nil {
			return nil, nil, err
		}
		it.CST = sped.CSTICMS(origem, cst)
		i := pos[exitID]
		notas[i].Itens = append(notas[i].Itens, it)
		if itemCode != nil && !notas[i].Cancelada {
			cod := fmt.Sprint(*itemCode)
			if !vistos[cod] {
				vistos[cod] = true
				itens = append(itens, sped.ItemCadastro{Cod: cod, Desc: firstNonEmptyStr(nomeItem, cod), UnidInv: unid,
					Tipo: sped.TipoItem(cadastroDoItem(engType, tipoUso, revenda, embalagem), ""), NCM: soDigitosRepo(ncm), CEST: soDigitosRepo(cest)})
			}
		}
	}
	return notas, itens, irows.Err()
}

func (r *FiscalRepositoryPG) fretesEFD(ctx context.Context, empresa int64, ini, fin string) ([]sped.Frete, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT serie, numero::text, COALESCE(chave_cte,''), data_emissao, lancado_em::date, cnpj_transportadora,
		        nome_transportadora, COALESCE(cfop,''), valor_frete, base_icms, aliq_icms, valor_icms, credita_icms,
		        COALESCE(xml_content,'')
		   FROM fiscal_freight_documents
		  WHERE enterprise_id=$1 AND status='LANCADO' AND lancado_em::date BETWEEN $2::date AND $3::date
		  ORDER BY lancado_em, id`, empresa, ini, fin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sped.Frete
	for rows.Next() {
		var f sped.Frete
		var cnpj, nome, cfop, xmlc string
		if err := rows.Scan(&f.Serie, &f.Numero, &f.Chave, &f.Emissao, &f.Lancamento, &cnpj, &nome, &cfop, &f.Valor,
			&f.Base, &f.Aliq, &f.ICMS, &f.CreditaICMS, &xmlc); err != nil {
			return nil, err
		}
		end := lerEnderecoEmitente(xmlc)
		f.Part = participanteEFD(cnpj, nome, "", end)
		f.CFOP = sped.CFOPEntrada(firstNonEmptyStr(cfop, "5353"))
		f.CST = "090"
		if f.ICMS.IsPositive() {
			f.CST = "000"
		}
		f.MunOrig, f.MunDest = soDigitosRepo(end.CMunIni), soDigitosRepo(end.CMunFim)
		out = append(out, f)
	}
	return out, rows.Err()
}

func firstNonEmptyStr(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func cadastroDoItem(engType, tipoUso *int16, revenda, embalagem bool) sped.CadastroDoItem {
	c := sped.CadastroDoItem{Revenda: revenda, Embalagem: embalagem}
	if engType != nil {
		v := int(*engType)
		c.TipoEngenharia = &v
	}
	if tipoUso != nil {
		v := int(*tipoUso)
		c.TipoUso = &v
	}
	return c
}

// ajustesEFD: as notas especiais de ajuste (VFIS0560) emitidas para o mês,
// com o código de ajuste da própria nota ou o da linha de apuração.
func (r *FiscalRepositoryPG) ajustesEFD(ctx context.Context, empresa int64, periodo string) ([]sped.AjusteApuracao, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(ac.code, lc.code, ''), COALESCE(n.history, n.observation, ''), n.total_icms
		   FROM special_adjustment_notes n
		   LEFT JOIN icms_apuracao_adjustment_codes ac ON ac.id = n.adjustment_code_id
		   LEFT JOIN icms_apuracao_lines l ON l.id = n.icms_apuracao_line_id
		   LEFT JOIN icms_apuracao_adjustment_codes lc ON lc.id = l.apuracao_adjustment_code_id
		  WHERE n.empresa_id = $1 AND n.purpose = 'AJUSTE' AND n.status = 'EMITIDA' AND n.period = $2
		  ORDER BY n.issue_date, n.id`, empresa, periodo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sped.AjusteApuracao
	for rows.Next() {
		var a sped.AjusteApuracao
		if err := rows.Scan(&a.Codigo, &a.Descricao, &a.Valor); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// InventarioEFD soma, por item, os movimentos até o fim do dia: quantidade com
// o sinal do tipo (o mesmo de SignedQuantity) e o valor que cada movimento
// registrou — entrada pelo custo de aquisição, saída pelo custo médio do
// momento, AJUSTE_CUSTO só valor. É o custo médio da data, sem refazer o
// cálculo. Item de terceiro em poder da empresa fica fora (não é estoque próprio).
func (r *FiscalRepositoryPG) InventarioEFD(ctx context.Context, data time.Time) ([]sped.ItemInventario, string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, "", err
	}
	rows, err := r.pool.Query(ctx,
		`WITH mov AS (
		    SELECT m.item_code,
		           CASE WHEN m.movement_type IN ('IN','TRANSFER_IN','EP','EPP','EPE','ENTRADA') THEN 1
		                WHEN m.movement_type IN ('OUT','TRANSFER_OUT','REP','SAIDA') THEN -1
		                WHEN m.movement_type = 'ADJUSTMENT' THEN sign(m.quantity)
		                ELSE 0 END AS sinal,
		           abs(m.quantity) AS qtd, abs(m.total_price) AS valor, m.movement_type, m.total_price
		      FROM stock_movements m
		     WHERE m.enterprise_id = $1 AND m.created_at < ($2::date + 1))
		 SELECT x.item_code, x.qtd, x.valor, COALESCE(i.name,''), COALESCE(i.warehouse_unit_of_measurement::text,''),
		        i.engineering_type, i.supplies_type_of_use, COALESCE(i.commercial_sale_type,'') = 'REVENDA', COALESCE(i.commercial_is_packaging,false),
		        COALESCE((SELECT fi.ncm FROM fiscal_entry_items fi JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id
		                   WHERE fe.enterprise_id = $1 AND fi.item_code = x.item_code AND fi.ncm IS NOT NULL ORDER BY fi.id DESC LIMIT 1),
		                 (SELECT xi.ncm FROM fiscal_exit_items xi JOIN fiscal_exits fx ON fx.id = xi.fiscal_exit_id
		                   WHERE fx.enterprise_id = $1 AND xi.item_code = x.item_code AND xi.ncm IS NOT NULL ORDER BY xi.id DESC LIMIT 1), ''),
		        COALESCE((SELECT b.avg_cost FROM stock_balances b WHERE b.enterprise_id = $1 AND b.item_code = x.item_code AND b.quantity > 0
		                   ORDER BY b.quantity DESC LIMIT 1), 0)
		   FROM (SELECT item_code,
		                SUM(sinal * qtd) AS qtd,
		                SUM(CASE WHEN movement_type = 'AJUSTE_CUSTO' THEN total_price ELSE sinal * valor END) AS valor
		           FROM mov GROUP BY item_code) x
		   LEFT JOIN items i ON i.code = x.item_code
		  WHERE x.qtd > 0 AND COALESCE(i.engineering_type, 1) <> 2
		  ORDER BY x.item_code`, empresa, data.Format("2006-01-02"))
	if err != nil {
		return nil, "", fmt.Errorf("inventário para o bloco H: %w", err)
	}
	defer rows.Close()
	var out []sped.ItemInventario
	for rows.Next() {
		var it sped.ItemInventario
		var code int64
		var nome, unid, ncm string
		var engType, tipoUso *int16
		var revenda, embalagem bool
		var custoAtual decimal.Decimal
		if err := rows.Scan(&code, &it.Quantidade, &it.Valor, &nome, &unid, &engType, &tipoUso, &revenda, &embalagem, &ncm, &custoAtual); err != nil {
			return nil, "", err
		}
		// Movimentos sem valor (entrada de produção sem custo, por exemplo):
		// vale o custo médio atual, para o item não ir zerado.
		if !it.Valor.IsPositive() && custoAtual.IsPositive() {
			it.Valor = it.Quantidade.Mul(custoAtual)
		}
		cod := fmt.Sprint(code)
		it.Item = sped.ItemCadastro{Cod: cod, Desc: firstNonEmptyStr(nome, cod), UnidInv: unid,
			Tipo: sped.TipoItem(cadastroDoItem(engType, tipoUso, revenda, embalagem), ""), NCM: soDigitosRepo(ncm)}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var conta string
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(a.account_number,'') FROM accounting_posting_params p
		   JOIN accounting_accounts a ON a.id = p.estoque_account_id WHERE p.enterprise_id = $1`, empresa).Scan(&conta)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, "", err
	}
	return out, conta, nil
}

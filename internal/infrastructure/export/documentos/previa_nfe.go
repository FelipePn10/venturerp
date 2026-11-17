package documentos

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// PreviaNFe desenha a prévia da NF-e no layout do DANFE, com os MESMOS campos
// da tela de prévia (DanfePrevia.tsx): quem confere imprime ou guarda em PDF o
// que vai ser transmitido. Chave e protocolo só existem depois da autorização,
// e cada página diz que a folha não tem valor fiscal.
func (Gerador) PreviaNFe(d response.PreviaNFeResponse) []byte {
	doc := lerPayloadNFe(d.PayloadEnviado)
	itens := listaJ(doc["items"])
	referenciadas := listaJ(doc["notas_referenciadas"])
	numero := "a definir"
	if d.NumeroNF > 0 {
		numero = milhar(fmt.Sprint(d.NumeroNF))
	}

	pdf := pdfkit.New()
	const mg = 22.0
	f := novaFolha(pdf, mg, 22)
	q := quadroNF{f}
	rodapeSimples(pdf, mg, verm, fmt.Sprintf("PRÉVIA — SEM VALOR FISCAL · NF-e %s série %s · conferência antes da transmissão", numero, ou(d.Serie, "—")))
	tarja := func() {
		f.p.FillRect(mg, f.y, f.cw(), 16, mistura(verm, branco, 0.88))
		f.p.TextCenter(mg+f.cw()/2, f.y+11, neg, 8.6, verm, "PRÉVIA — SEM VALOR FISCAL · chave e protocolo são gerados na autorização")
		f.y += 20
	}
	f.continua = func() float64 {
		f.y = mg
		tarja()
		f.p.StrokeRect(mg, f.y, f.cw(), 18, 0.8, grafite)
		f.texto(mg+6, f.y+12, neg, 7.6, grafite, "DANFE — PRÉVIA")
		f.p.TextRight(f.w-mg-6, f.y+12, neg, 7.6, grafite, fmt.Sprintf("NF-e Nº %s · série %s (continuação)", numero, ou(d.Serie, "—")))
		return f.y + 24
	}
	p := f.p
	cw := f.cw()
	tarja()

	// Canhoto.
	// O canhoto quebra em até duas linhas: razão social longa não pode sumir.
	p.StrokeRect(mg, f.y, cw-120, 24, 0.6, grafite)
	p.Text(mg+3.5, f.y+7, neg, 5.4, cinza, "RECEBEMOS DE")
	for i, l := range quebrar(fmt.Sprintf("%s os produtos e/ou serviços constantes da nota fiscal eletrônica indicada ao lado", ou(d.Emitente.Nome, "—")), reg, 6.8, cw-127) {
		if i == 2 {
			break
		}
		p.Text(mg+3.5, f.y+14.5+float64(i)*7.5, reg, 6.8, tinta, l)
	}
	q.campo(mg+cw-120, f.y, 120, 48, "NF-e", fmt.Sprintf("Nº %s · Série %s", numero, ou(d.Serie, "—")), true)
	f.y += 24
	q.campo(mg, f.y, (cw-120)/4, 24, "Data de recebimento", "", false)
	q.campo(mg+(cw-120)/4, f.y, (cw-120)*3/4, 24, "Identificação e assinatura do recebedor", "", false)
	f.y += 30
	for x := mg; x < mg+cw; x += 6 {
		p.StrokeLine(x, f.y, x+3, f.y, 0.5, cinza)
	}
	f.y += 6

	// Emitente · DANFE · chave.
	top := f.y
	we, wt := cw*0.42, cw*0.2
	wc := cw - we - wt
	p.StrokeRect(mg, top, we, 76, 0.8, grafite)
	// Razão social em até duas linhas, como no DANFE.
	ly := top + 14.0
	for i, l := range quebrar(ou(d.Emitente.Nome, "emitente sem nome"), neg, 8.8, we-12) {
		if i == 2 {
			break
		}
		p.Text(mg+6, ly, neg, 8.8, tinta, l)
		ly += 10
	}
	ly += 2
	for _, l := range []string{
		enderecoNFe(d.Emitente),
		juntar(" — ", d.Emitente.Bairro, cep(soDigitos(d.Emitente.CEP))),
		juntar(" / ", d.Emitente.Municipio, d.Emitente.UF),
		juntar(" · ", rotulo("Fone: ", telefone(d.Emitente.Telefone)), d.Emitente.Email),
	} {
		if l != "" {
			p.Text(mg+6, ly, reg, 7.2, tinta, fit(l, reg, 7.2, we-12))
			ly += 10
		}
	}
	tx := mg + we
	p.StrokeRect(tx, top, wt, 76, 0.8, grafite)
	p.TextCenter(tx+wt/2, top+15, neg, 13, tinta, "DANFE")
	p.TextCenter(tx+wt/2, top+25, reg, 6.4, tinta, "Documento Auxiliar da")
	p.TextCenter(tx+wt/2, top+32, reg, 6.4, tinta, "Nota Fiscal Eletrônica")
	es := "1"
	if txtJ(doc["tipo_documento"]) == "0" {
		es = "0"
	}
	p.Text(tx+8, top+44, reg, 6.4, tinta, "0 — Entrada")
	p.Text(tx+8, top+52, reg, 6.4, tinta, "1 — Saída")
	p.StrokeRect(tx+wt-26, top+38, 16, 16, 0.8, grafite)
	p.TextCenter(tx+wt-18, top+50, neg, 10, tinta, es)
	p.TextCenter(tx+wt/2, top+65, neg, 8.4, tinta, "Nº "+numero)
	p.TextCenter(tx+wt/2, top+73, reg, 6.6, tinta, "Série "+ou(d.Serie, "—"))
	cx := tx + wt
	p.StrokeRect(cx, top, wc, 76, 0.8, grafite)
	p.FillRect(cx+6, top+6, wc-12, 26, fundo)
	p.TextCenter(cx+wc/2, top+22, ita, 7, cinza, "código de barras gerado na autorização")
	q.campo(cx, top+36, wc, 22, "Chave de acesso", "Gerada na autorização", false)
	p.Text(cx+4, top+66, reg, 5.6, cinza, fit("Consulta de autenticidade no portal nacional da NF-e ou no site da SEFAZ autorizadora.", reg, 5.6, wc-8))
	f.y = top + 76

	campos := func(cs ...campoNFe) {
		soma := 0.0
		for _, c := range cs {
			soma += c.peso
		}
		x := mg
		for _, c := range cs {
			w := cw * c.peso / soma
			v := c.valor
			if v == "" && c.exigido {
				q.campoFalta(x, f.y, w, 22, c.rotulo)
			} else {
				q.campo(x, f.y, w, 22, c.rotulo, v, false)
			}
			x += w
		}
		f.y += 22
	}
	campos(campoNFe{"Natureza da operação", ou(txtJ(doc["natureza_operacao"]), d.NaturezaOperacao), 3, true},
		campoNFe{"Protocolo de autorização de uso", "Gerado na autorização", 2, false})
	campos(campoNFe{"Inscrição estadual", d.Emitente.IE, 1, true},
		campoNFe{"Insc. estadual do subst. tributário", txtJ(doc["inscricao_estadual_st"]), 1, false},
		campoNFe{"CNPJ", cnpj(d.Emitente.Documento), 1, true})

	secao := func(s string) {
		f.cabe(40)
		f.texto(mg, f.y+10, neg, 7, tinta, strings.ToUpper(s))
		f.y += 13
	}
	secao("Destinatário / remetente")
	campos(campoNFe{"Nome / razão social", d.Destinatario.Nome, 4, true},
		campoNFe{"CNPJ / CPF", cnpj(d.Destinatario.Documento), 2, true},
		campoNFe{"Data de emissão", dataISO(ou(txtJ(doc["data_emissao"]), d.DataEmissao)), 1, true})
	campos(campoNFe{"Endereço", enderecoNFe(d.Destinatario), 4, true},
		campoNFe{"Bairro / distrito", d.Destinatario.Bairro, 2, true},
		campoNFe{"CEP", cep(soDigitos(d.Destinatario.CEP)), 1, true},
		campoNFe{"Data de saída", dataISO(d.DataSaida), 1, false})
	campos(campoNFe{"Município", d.Destinatario.Municipio, 3, true},
		campoNFe{"Fone / fax", telefone(d.Destinatario.Telefone), 2, false},
		campoNFe{"UF", d.Destinatario.UF, 1, true},
		campoNFe{"Inscrição estadual", d.Destinatario.IE, 2, false})

	if len(d.Pagamento.Parcelas) > 0 {
		secao("Fatura / duplicatas")
		w := cw / 6
		for i, pa := range d.Pagamento.Parcelas {
			if i > 0 && i%6 == 0 {
				f.y += 24
				f.cabe(24)
			}
			x := mg + float64(i%6)*w
			q.campo(x, f.y, w, 24, fmt.Sprintf("%03d · %s", pa.Numero, dataISO(pa.Vencimento)), num(dec(pa.Valor), 2), true)
		}
		f.y += 24
	}

	secao("Cálculo do imposto")
	baseICMS := decimal.Zero
	for _, it := range itens {
		baseICMS = baseICMS.Add(numJ(it["icms_base_calculo"]))
	}
	t := d.Totais
	campos(campoNFe{"Base de cálc. do ICMS", num(baseICMS, 2), 1, false}, campoNFe{"Valor do ICMS", num(dec(t.ValorICMS), 2), 1, false},
		campoNFe{"Base de cálc. do ICMS ST", num(dec(t.BaseICMSST), 2), 1, false}, campoNFe{"Valor do ICMS ST", num(dec(t.ValorICMSST), 2), 1, false},
		campoNFe{"Valor total dos produtos", num(dec(t.ValorProdutos), 2), 1, false})
	campos(campoNFe{"Valor do frete", num(dec(t.ValorFrete), 2), 1, false}, campoNFe{"Valor do seguro", num(dec(t.ValorSeguro), 2), 1, false},
		campoNFe{"Desconto", num(dec(t.ValorDesconto), 2), 1, false}, campoNFe{"Outras despesas", num(numJ(doc["valor_outras_despesas"]), 2), 1, false},
		campoNFe{"Valor do IPI", num(dec(t.ValorIPI), 2), 1, false}, campoNFe{"Valor total da nota", num(dec(t.ValorTotalNF), 2), 1, false})

	secao("Transportador / volumes transportados")
	campos(campoNFe{"Razão social", txtJ(doc["nome_transportador"]), 3, false},
		campoNFe{"Frete por conta", modalidadeFrete[txtJ(doc["modalidade_frete"])], 2, false},
		campoNFe{"Placa do veículo", txtJ(doc["veiculo_placa"]), 1, false},
		campoNFe{"UF", txtJ(doc["veiculo_uf"]), 0.5, false},
		campoNFe{"CNPJ / CPF", cnpj(txtJ(doc["cnpj_transportador"])), 2, false})

	secao("Dados dos produtos / serviços")
	cols := []col{{"Código", 1.25, pdfkit.AlignLeft}, {"Descrição do produto / serviço", 2.6, pdfkit.AlignLeft},
		{"NCM/SH", 1.0, pdfkit.AlignCenter}, {"O/CST", 0.5, pdfkit.AlignCenter}, {"CFOP", 0.5, pdfkit.AlignCenter},
		{"UN", 0.4, pdfkit.AlignCenter}, {"Quant.", 0.75, pdfkit.AlignRight}, {"V. unit.", 0.8, pdfkit.AlignRight},
		{"V. total", 0.85, pdfkit.AlignRight}, {"BC ICMS", 0.8, pdfkit.AlignRight}, {"V. ICMS", 0.7, pdfkit.AlignRight},
		{"V. IPI", 0.65, pdfkit.AlignRight}, {"% ICMS", 0.55, pdfkit.AlignRight}, {"% IPI", 0.5, pdfkit.AlignRight}}
	const tamItem = 6.6
	tb := f.tabela(mg, cw, cols, estiloTabela{cabFundo: grafite, cabTexto: branco, grade: linha, vertical: true, tam: tamItem, tamCab: 5.6, altLinha: 14})
	tb.cabecalho()
	if len(itens) == 0 {
		tb.linha([]string{"", "Esta nota ainda não tem itens."}, ita, cinza, nil)
	}
	for _, it := range itens {
		// A descrição quebra em linhas em vez de cortar: é o que se confere.
		desc := quebrar(txtJ(it["descricao"]), reg, tamItem, tb.ws[1]-7)
		if len(desc) == 0 {
			desc = []string{""}
		}
		h := 14.0
		if extra := float64(len(desc)-1) * 8; extra > 0 {
			h += extra
		}
		topo := tb.linhaAlta([]string{txtJ(it["codigo_produto"]), "", txtJ(it["codigo_ncm"]),
			txtJ(it["icms_origem"]) + txtJ(it["icms_situacao_tributaria"]), txtJ(it["cfop"]), txtJ(it["unidade_comercial"]),
			qtd(numJ(it["quantidade_comercial"])), num(numJ(it["valor_unitario_comercial"]), 4), num(numJ(it["valor_bruto"]), 2),
			num(numJ(it["icms_base_calculo"]), 2), num(numJ(it["icms_valor"]), 2), num(numJ(it["ipi_valor"]), 2),
			qtd(numJ(it["icms_aliquota"])), qtd(numJ(it["ipi_aliquota"]))}, reg, tinta, nil, h)
		for i, l := range desc {
			f.p.Text(tb.xCol(1)+3.5, topo+9.6+float64(i)*8, reg, tamItem, tinta, l)
		}
	}
	f.y += 8

	secao("Dados adicionais")
	infCpl := txtJ(doc["informacoes_adicionais_contribuinte"])
	var chaves []string
	for _, n := range referenciadas {
		if c := txtJ(n["chave_nfe"]); c != "" {
			chaves = append(chaves, chaveNFe(c))
		} else if nn := txtJ(n["numero"]); nn != "" {
			chaves = append(chaves, nn)
		}
	}
	if len(chaves) > 0 {
		infCpl = juntar("\n", infCpl, "Notas referenciadas: "+strings.Join(chaves, "; "))
	}
	wl := cw * 0.72
	var ls []string
	for _, par := range strings.Split(infCpl, "\n") {
		ls = append(ls, quebrar(par, reg, 7, wl-12)...)
	}
	lf := quebrar(txtJ(doc["informacoes_adicionais_fisco"]), reg, 7, cw-wl-12)
	n := len(ls)
	if len(lf) > n {
		n = len(lf)
	}
	h := 20 + float64(n)*9
	if h < 48 {
		h = 48
	}
	f.cabe(h)
	p = f.p
	p.StrokeRect(mg, f.y, wl, h, 0.6, grafite)
	p.Text(mg+4, f.y+7, neg, 5.4, cinza, "INFORMAÇÕES COMPLEMENTARES")
	for i, l := range ls {
		p.Text(mg+6, f.y+18+float64(i)*9, reg, 7, tinta, l)
	}
	p.StrokeRect(mg+wl, f.y, cw-wl, h, 0.6, grafite)
	p.Text(mg+wl+4, f.y+7, neg, 5.4, cinza, "RESERVADO AO FISCO")
	for i, l := range lf {
		p.Text(mg+wl+6, f.y+18+float64(i)*9, reg, 7, tinta, l)
	}
	f.y += h
	return pdf.Render()
}

type campoNFe struct {
	rotulo, valor string
	peso          float64
	exigido       bool
}

// campoFalta é o campo obrigatório vazio: em vermelho, como na tela.
func (q quadroNF) campoFalta(x, y, w, h float64, rot string) {
	p := q.f.p
	p.StrokeRect(x, y, w, h, 0.6, grafite)
	p.Text(x+3.5, y+7, neg, 5.4, cinza, fit(strings.ToUpper(rot), neg, 5.4, w-7))
	p.Text(x+3.5, y+h-4.5, ita, 8, verm, "não informado")
}

var modalidadeFrete = map[string]string{
	"0": "0 — Por conta do emitente",
	"1": "1 — Por conta do destinatário",
	"2": "2 — Por conta de terceiros",
	"3": "3 — Transporte próprio do emitente",
	"4": "4 — Transporte próprio do destinatário",
	"9": "9 — Sem frete",
}

func enderecoNFe(p response.PreviaParteNFe) string {
	return juntar(", ", p.Logradouro, p.Numero, p.Complemento)
}

func lerPayloadNFe(payload string) map[string]any {
	var m map[string]any
	if json.Unmarshal([]byte(payload), &m) != nil || m == nil {
		return map[string]any{}
	}
	return m
}

func listaJ(v any) []map[string]any {
	l, _ := v.([]any)
	out := make([]map[string]any, 0, len(l))
	for _, e := range l {
		if m, ok := e.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func txtJ(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case float64:
		return decimal.NewFromFloat(x).String()
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

// numJ lê número do payload (vem como número ou como texto).
func numJ(v any) decimal.Decimal {
	d, err := decimal.NewFromString(txtJ(v))
	if err != nil {
		return decimal.Zero
	}
	return d
}

func soDigitos(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// dataISO mostra "2026-11-17..." como 17/11/2026.
func dataISO(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 10 {
		return s
	}
	return s[8:10] + "/" + s[5:7] + "/" + s[0:4]
}

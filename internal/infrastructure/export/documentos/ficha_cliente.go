package documentos

import (
	"fmt"
	"strings"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Ficha cadastral do cliente — o cadastro em formulário (identificação,
// classificação, condições comerciais), a posição comercial na data (títulos
// em aberto, vencidos, carteira, faturamento e uso do limite de crédito),
// endereços, contatos e filiais.

var azulCadastro = cor{R: 51, G: 78, B: 104}

func rotuloFinalidade(s string) string {
	switch strings.ToUpper(s) {
	case "REVENDA":
		return "Revenda"
	case "INDUSTRIALIZACAO":
		return "Industrialização"
	case "USO_CONSUMO":
		return "Uso e consumo"
	}
	return ou(s, "não informada")
}

func rotuloEndereco(s string) string {
	switch strings.ToUpper(s) {
	case "COBRANCA":
		return "Cobrança"
	case "ENTREGA":
		return "Entrega"
	case "FATURAMENTO":
		return "Faturamento"
	case "COMERCIAL":
		return "Comercial"
	}
	return ou(s, "—")
}

func (Gerador) FichaCliente(d uc.FichaCliente) []byte {
	doc := pdfkit.New()
	const mg = 30.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := azulCadastro
	claro := mistura(ac, branco, 0.9)
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Ficha cadastral do cliente %d · %s", d.Cliente.Codigo, emitidoPor(d.Cabecalho)))
	f.continua = func() float64 {
		f.texto(mg, mg+10, neg, 8.5, ac, fmt.Sprintf("FICHA CADASTRAL · CLIENTE %d (continuação)", d.Cliente.Codigo))
		f.p.StrokeLine(mg, mg+15, f.w-mg, mg+15, 0.8, ac)
		return mg + 26
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho: emitente à esquerda, cartão do cliente à direita.
	x := mg
	if lw := desenharLogo(p, f.logo, x, mg, 80, 44); lw > 0 {
		x += lw + 12
	}
	larguraEmp := cw - 210 - (x - mg)
	yy := mg + 11.0
	for _, l := range quebrar(d.Emitente.Nome, neg, 8.6, larguraEmp) {
		f.texto(x, yy, neg, 8.6, tinta, l)
		yy += 10
	}
	for _, l := range linhasDaParte(d.Emitente.Parte) {
		f.texto(x, yy, reg, 6.6, cinza, fit(l, reg, 6.6, larguraEmp))
		yy += 8
	}
	bx := f.w - mg - 196
	p.FillRect(bx, mg, 196, 60, ac)
	f.texto(bx+12, mg+15, neg, 7, mistura(ac, branco, 0.65), "FICHA CADASTRAL DO CLIENTE")
	f.texto(bx+12, mg+39, neg, 20, branco, fmt.Sprintf("Cód. %d", d.Cliente.Codigo))
	sit, cs := "ATIVO", verde
	switch {
	case d.Bloqueado:
		sit, cs = "BLOQUEADO", verm
	case !d.Ativo:
		sit, cs = "INATIVO", cinza
	}
	sw := larg(neg, 7, sit) + 12
	p.FillRect(bx+196-sw-10, mg+44, sw, 12, branco)
	p.Text(bx+196-sw-4, mg+52.5, neg, 7, cs, sit)
	f.texto(bx+12, mg+54, reg, 6.8, mistura(ac, branco, 0.75), "Cliente desde "+data(d.Cadastro))
	f.y = maxF(yy, mg+60) + 14

	// Grade de formulário.
	celula := func(x, y, w float64, rot, val string, destaque bool) {
		p := f.p
		p.StrokeRect(x, y, w, 26, 0.5, linha)
		p.Text(x+5, y+8, neg, 5.6, cinza, fit(strings.ToUpper(rot), neg, 5.6, w-10))
		ft, t := reg, 8.2
		if destaque {
			ft, t = neg, 8.8
		}
		p.Text(x+5, y+20, ft, t, tinta, fit(ou(val, "—"), ft, t, w-10))
	}
	grade := func(titulo string, linhas [][]struct {
		r, v string
		w    float64
		d    bool
	}) {
		f.cabe(18 + 26*float64(len(linhas)))
		f.p.FillRect(mg, f.y, cw, 15, claro)
		f.texto(mg+8, f.y+10.5, neg, 7.4, ac, titulo)
		f.y += 15
		for _, ln := range linhas {
			soma := 0.0
			for _, c := range ln {
				soma += c.w
			}
			xx := mg
			for _, c := range ln {
				w := cw * c.w / soma
				celula(xx, f.y, w, c.r, c.v, c.d)
				xx += w
			}
			f.y += 26
		}
		f.y += 10
	}
	type c = struct {
		r, v string
		w    float64
		d    bool
	}
	grade("IDENTIFICAÇÃO", [][]c{
		{{"Razão social", d.Cliente.Nome, 3, true}, {"Nome fantasia", d.Cliente.Fantasia, 1.6, false}},
		{{"CNPJ / CPF", cnpj(d.Cliente.CNPJCPF), 1.2, true}, {"Inscrição estadual", d.Cliente.IE, 1, false}, {"Inscrição municipal", d.InscricaoMunicipal, 1, false}, {"SUFRAMA", d.Suframa, 0.8, false}},
		{{"Matriz", ou(d.Matriz, "é a própria (matriz)"), 1.8, false}, {"Site", d.Site, 1.4, false}, {"Telefone", telefone(d.Cliente.Telefone), 0.9, false}, {"E-mail", d.Cliente.Email, 1.5, false}},
	})
	grade("CLASSIFICAÇÃO E CONDIÇÕES COMERCIAIS", [][]c{
		{{"Tipo de cliente", d.Tipo, 1, false}, {"Região", d.Regiao, 1, false}, {"Segmento de mercado", d.Segmento, 1.2, false}, {"Finalidade da compra", rotuloFinalidade(d.Finalidade), 1, true}},
		{{"Condição de pagamento", ou(d.Condicao, "não definida"), 1.1, false}, {"Tabela de venda", ou(d.Tabela, "não definida"), 1.2, false}, {"Transportadora", ou(d.Transportadora, "não definida"), 1.3, false}, {"Limite de crédito", limiteTexto(d.LimiteCredito), 0.9, true}},
	})

	// Posição comercial.
	f.cabe(110)
	f.p.FillRect(mg, f.y, cw, 15, claro)
	f.texto(mg+8, f.y+10.5, neg, 7.4, ac, "POSIÇÃO COMERCIAL EM "+data(d.Gerado))
	f.y += 21
	ultima := "sem compras"
	if d.UltimaCompra != nil {
		ultima = data(*d.UltimaCompra)
	}
	cartoes(f, mg, cw, [][3]string{
		{"Em aberto", moeda(d.EmAberto), fmt.Sprintf("%d título(s)", d.TitulosAbertos)},
		{"Vencido", moeda(d.Vencido), "em atraso hoje"},
		{"Pedidos em carteira", moeda(d.ValorPedidos), fmt.Sprintf("%d pedido(s) confirmado(s)", d.PedidosAbertos)},
		{"Faturado em 12 meses", moeda(d.Faturado12m), "notas autorizadas"},
		{"Última compra", ultima, rotulo("NF-e ", numSe(decimal.NewFromInt(d.UltimaNF), 0))},
	}, []cor{ac, verm, azulVenda, verde, cinza})
	// Uso do limite de crédito: exposição = em aberto + carteira.
	exp := d.EmAberto.Add(d.ValorPedidos)
	p = f.p
	if d.LimiteCredito.IsPositive() {
		uso := exp.Div(d.LimiteCredito).InexactFloat64()
		bw := cw - 200
		p.FillRect(mg+200, f.y, bw, 10, fundo)
		cb := verde
		if uso > 0.8 {
			cb = ambar
		}
		if uso > 1 {
			cb = verm
		}
		p.FillRect(mg+200, f.y, bw*minF(uso, 1), 10, cb)
		f.texto(mg, f.y+8, neg, 7.2, tinta, fmt.Sprintf("Uso do limite: %s de %s", moeda(exp), moeda(d.LimiteCredito)))
		p.TextRight(f.w-mg, f.y+21, neg, 7, cb, num(decimal.NewFromFloat(uso*100), 1)+"% do limite")
		f.y += 30
	} else {
		f.texto(mg, f.y+8, ita, 7.2, cinza, fmt.Sprintf("Sem limite de crédito definido · exposição atual %s (títulos em aberto + pedidos em carteira).", moeda(exp)))
		f.y += 20
	}
	if d.Bloqueado {
		f.cabe(24)
		f.p.FillRect(mg, f.y, cw, 18, mistura(verm, branco, 0.88))
		f.texto(mg+8, f.y+12, neg, 7.6, verm, fit("Cliente bloqueado: "+ou(d.MotivoBloqueio, "motivo não informado"), neg, 7.6, cw-16))
		f.y += 26
	}
	f.y += 6

	// Endereços.
	f.texto(mg, f.y+6, neg, 8, ac, "ENDEREÇOS")
	f.y += 10
	te := f.tabela(mg, cw, []col{{"Tipo", 0.8, pdfkit.AlignLeft}, {"Endereço", 3, pdfkit.AlignLeft}, {"Bairro", 1.3, pdfkit.AlignLeft},
		{"Cidade / UF", 1.4, pdfkit.AlignLeft}, {"CEP", 0.75, pdfkit.AlignCenter}, {"Padrão", 0.5, pdfkit.AlignCenter}},
		estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha})
	te.cabecalho()
	if len(d.Enderecos) == 0 {
		te.linha([]string{"", "Nenhum endereço cadastrado."}, ita, verm, nil)
	}
	tipos := map[string]bool{}
	for _, e := range d.Enderecos {
		tipos[strings.ToUpper(e.Tipo)] = true
		topo := te.linha([]string{rotuloEndereco(e.Tipo), e.Endereco, e.Bairro, juntar(" / ", e.Cidade, e.UF), cep(e.CEP), ""}, reg, tinta, nil)
		if e.Padrao {
			marcaOK(f.p, te.xCol(5)+te.ws[5]/2-4, topo+4, verde)
		}
	}
	if len(d.Enderecos) > 0 && !tipos["ENTREGA"] {
		f.texto(mg, f.y+9, ita, 6.8, cinza, "Sem endereço de entrega: romaneio e nota usam o endereço principal.")
		f.y += 12
	}
	f.y += 12

	// Contatos.
	f.texto(mg, f.y+6, neg, 8, ac, "CONTATOS")
	f.y += 10
	tc := f.tabela(mg, cw, []col{{"Nome", 1.6, pdfkit.AlignLeft}, {"Cargo", 1.1, pdfkit.AlignLeft}, {"E-mail", 1.9, pdfkit.AlignLeft},
		{"Telefone", 0.95, pdfkit.AlignLeft}, {"Celular", 0.95, pdfkit.AlignLeft}, {"Principal", 0.55, pdfkit.AlignCenter}},
		estiloTabela{cabFundo: claro, cabTexto: ac, grade: linha})
	tc.cabecalho()
	if len(d.Contatos) == 0 {
		tc.linha([]string{"Nenhum contato cadastrado."}, ita, cinza, nil)
	}
	for _, x := range d.Contatos {
		topo := tc.linha([]string{x.Nome, x.Cargo, x.Email, telefone(x.Telefone), telefone(x.Celular), ""}, reg, tinta, nil)
		if x.Principal {
			marcaOK(f.p, tc.xCol(5)+tc.ws[5]/2-4, topo+4, verde)
		}
	}
	f.y += 12

	if len(d.Filiais) > 0 {
		f.texto(mg, f.y+6, neg, 8, ac, "FILIAIS / ESTABELECIMENTOS")
		f.y += 10
		tf := f.tabela(mg, cw, []col{{"Código", 0.6, pdfkit.AlignCenter}, {"Nome", 3, pdfkit.AlignLeft}, {"CNPJ", 1.3, pdfkit.AlignLeft}, {"IE", 1, pdfkit.AlignLeft}},
			estiloTabela{cabFundo: claro, cabTexto: ac, grade: linha})
		tf.cabecalho()
		for _, x := range d.Filiais {
			tf.linha([]string{fmt.Sprint(x.Codigo), x.Nome, cnpj(x.CNPJCPF), x.IE}, reg, tinta, nil)
		}
		f.y += 12
	}

	// Conferência do cadastro.
	f.cabe(56)
	bw := (cw - 20) / 3
	for i, r := range []string{"Cadastrado / atualizado por", "Conferido (financeiro / crédito)", "Data"} {
		ax := mg + float64(i)*(bw+10)
		f.p.StrokeLine(ax, f.y+30, ax+bw, f.y+30, 0.6, tinta)
		f.p.TextCenter(ax+bw/2, f.y+40, reg, 6.6, cinza, r)
		if i == 0 && d.EmitidoPor != "" {
			f.p.TextCenter(ax+bw/2, f.y+26, reg, 7.4, tinta, d.EmitidoPor)
		}
	}
	return doc.Render()
}

func limiteTexto(v decimal.Decimal) string {
	if !v.IsPositive() {
		return "sem limite"
	}
	return moeda(v)
}

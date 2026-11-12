package documentos

import (
	"fmt"
	"strings"
	"time"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Romaneio de expedição — o documento que acompanha a carga: remetente,
// destinatário com o local de entrega, transporte (motorista, placa, lacres),
// a NF-e com o código de barras da chave, itens com peso e conferência, as
// etapas do romaneio, assinaturas de separação, conferência e motorista, e o
// canhoto de recebimento destacável.

var laranja = cor{R: 194, G: 65, B: 12}

func situacaoRomaneio(s string) (string, cor) {
	switch s {
	case "OPEN":
		return "ABERTO", azulVenda
	case "SEPARATED":
		return "SEPARADO", ambar
	case "CONFERRED":
		return "CONFERIDO", cor{R: 14, G: 116, B: 144}
	case "SHIPPED":
		return "DESPACHADO", verde
	case "CANCELLED":
		return "CANCELADO", verm
	}
	return ou(s, "—"), cinza
}

func (Gerador) Romaneio(d uc.Romaneio) []byte {
	doc := pdfkit.New()
	const mg = 26.0
	f := novaFolha(doc, mg, 20)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := laranja
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Romaneio nº %d · %s · a situação válida é a do sistema (VEXP0100)", d.Codigo, emitidoPor(d.Cabecalho)))
	f.continua = func() float64 {
		f.p.FillRect(mg, mg, f.cw(), 20, preto)
		f.texto(mg+8, mg+13.5, neg, 8.5, branco, "ROMANEIO DE EXPEDIÇÃO")
		f.p.TextRight(f.w-mg-8, mg+13.5, neg, 8.5, ac, fmt.Sprintf("Nº %d (continuação)", d.Codigo))
		return mg + 28
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho preto com faixa laranja; número e código de barras à direita.
	p.FillRect(mg, mg, cw, 70, preto)
	p.FillRect(mg, mg+70, cw, 4, ac)
	x := mg + 10
	if f.logo != nil {
		p.FillRect(mg+8, mg+10, 78, 50, branco)
		desenharLogo(p, f.logo, mg+12, mg+14, 70, 42)
		x = mg + 98
	}
	f.texto(x, mg+20, neg, 7, ac, "LOGÍSTICA · EXPEDIÇÃO")
	f.texto(x, mg+40, neg, 17, branco, "Romaneio de carga")
	sit, cs := situacaoRomaneio(d.Situacao)
	selo(p, x, mg+48, sit, cs, branco, 7)
	if d.PedidoVenda != 0 {
		f.texto(x+larg(neg, 7, sit)+22, mg+57, reg, 7.4, mistura(preto, branco, 0.7), fmt.Sprintf("Pedido de venda nº %d", d.PedidoVenda))
	}
	rx := f.w - mg - 176
	p.FillRect(rx, mg+8, 168, 56, branco)
	p.Text(rx+8, mg+19, neg, 6.4, cinza, "ROMANEIO Nº")
	p.Text(rx+8, mg+38, neg, 18, preto, milhar(fmt.Sprint(d.Codigo)))
	bc := fmt.Sprintf("ROM%d", d.Codigo)
	_ = p.DrawCode128B(bc, rx+62, mg+12, 102, 26, preto)
	p.TextCenter(rx+113, mg+46, reg, 5.8, cinza, bc)
	p.Text(rx+8, mg+57, reg, 6.8, cinza, "Criado em "+dataHora(d.Criado))
	f.y = mg + 84

	// Remetente, destinatário, transporte.
	colW := (cw - 16) / 3
	h := 104.0
	bloco := func(i int, titulo string) float64 {
		bx := mg + float64(i)*(colW+8)
		p.StrokeRect(bx, f.y, colW, h, 0.6, linha)
		p.FillRect(bx, f.y, colW, 16, mistura(ac, branco, 0.88))
		p.Text(bx+8, f.y+11, neg, 7, ac, titulo)
		return bx
	}
	escreve := func(bx float64, y float64, ls []string) {
		for _, l := range ls {
			for _, q := range quebrar(l, reg, 7.2, colW-16) {
				if y > f.y+h-4 {
					return
				}
				p.Text(bx+8, y, reg, 7.2, tinta, q)
				y += 9.2
			}
		}
	}
	bx := bloco(0, "REMETENTE")
	ny := f.y + 28.0
	for _, l := range quebrar(d.Emitente.Nome, neg, 8, colW-16) {
		p.Text(bx+8, ny, neg, 8, tinta, l)
		ny += 9.5
	}
	escreve(bx, ny+2, linhasDaParte(d.Emitente.Parte))

	bx = bloco(1, "DESTINATÁRIO")
	ny = f.y + 28
	for _, l := range quebrar(d.Destinatario.Nome, neg, 8, colW-16) {
		p.Text(bx+8, ny, neg, 8, tinta, l)
		ny += 9.5
	}
	ls := linhasDaParte(d.Destinatario)
	if d.EnderecoEntrega != "" {
		ls = append(ls, "Entregar em: "+d.EnderecoEntrega)
	}
	escreve(bx, ny+2, ls)

	bx = bloco(2, "TRANSPORTE")
	tr := [][2]string{
		{"Transportadora", ou(d.Transportadora.Nome, "a definir")},
		{"CNPJ", ou(cnpj(d.Transportadora.CNPJCPF), "—")},
		{"Modalidade", ou(rotuloFrete(d.ModalidadeFrete), "a definir")},
		{"Placa · ANTT", juntar(" · ", ou(d.Placa, "____-____"), d.ANTT)},
		{"Motorista", ou(d.Motorista, "_______________________")},
		{"Documento", ou(d.DocMotorista, "_______________________")},
		{"Lacres", ou(d.Lacres, "_______________________")},
		{"Previsão entrega", dataP(d.PrevisaoEntrega)},
	}
	ty := f.y + 27
	for _, t := range tr {
		p.Text(bx+8, ty, neg, 5.9, cinza, strings.ToUpper(t[0]))
		p.Text(bx+72, ty, reg, 7.2, tinta, fit(t[1], reg, 7.2, colW-80))
		ty += 9.6
	}
	f.y += h + 10

	// Nota fiscal.
	p.StrokeRect(mg, f.y, cw, 54, 0.8, preto)
	p.FillRect(mg, f.y, 90, 54, preto)
	p.Text(mg+8, f.y+14, neg, 6.4, ac, "NOTA FISCAL")
	if d.NFNumero != 0 {
		p.Text(mg+8, f.y+34, neg, 15, branco, fmt.Sprintf("%d", d.NFNumero))
		p.Text(mg+8, f.y+46, reg, 6.6, mistura(preto, branco, 0.7), "série "+ou(d.NFSerie, "1"))
		p.Text(mg+100, f.y+11, neg, 6, cinza, "NF-E · VALOR "+moeda(d.NFValor)+" · CHAVE DE ACESSO")
		if len(d.NFChave) == 44 {
			_ = p.DrawCode128B(d.NFChave, mg+96, f.y+14, cw-104, 24, preto)
			p.TextCenter(mg+96+(cw-104)/2, f.y+48, neg, 7.4, tinta, chaveNFe(d.NFChave))
		}
	} else {
		p.Text(mg+8, f.y+34, neg, 10, branco, "—")
		p.Text(mg+100, f.y+30, ita, 7.6, ambar, "Romaneio ainda sem NF-e vinculada: a carga não sai sem nota fiscal.")
	}
	f.y += 64

	// Itens.
	f.texto(mg, f.y+6, neg, 8.5, preto, "ITENS DA CARGA")
	f.y += 10
	cols := []col{{"#", 0.3, pdfkit.AlignCenter}, {"Código", 0.95, pdfkit.AlignLeft}, {"Descrição", 2.7, pdfkit.AlignLeft}, {"Dep.", 0.6, pdfkit.AlignCenter},
		{"Qtde", 0.6, pdfkit.AlignRight}, {"Un", 0.35, pdfkit.AlignCenter}, {"Peso un.", 0.65, pdfkit.AlignRight}, {"Peso líq.", 0.75, pdfkit.AlignRight},
		{"Peso bruto", 0.75, pdfkit.AlignRight}, {"Conferido", 0.7, pdfkit.AlignRight}, {"Conf.", 0.45, pdfkit.AlignCenter}}
	t := f.tabela(mg, cw, cols, estiloTabela{cabFundo: preto, cabTexto: branco, grade: linha, altLinha: 18})
	t.cabecalho()
	var pl, pb, q decimal.Decimal
	for _, it := range d.Itens {
		l, b := it.PesoLiquido.Mul(it.Quantidade), it.PesoBruto.Mul(it.Quantidade)
		pl, pb, q = pl.Add(l), pb.Add(b), q.Add(it.Quantidade)
		conf := ""
		if it.Conferida.IsPositive() {
			conf = qtd(it.Conferida)
		}
		topo := t.linha([]string{fmt.Sprint(it.Seq), it.Codigo, it.Descricao, it.Deposito, qtd(it.Quantidade), it.UM, num(it.PesoLiquido, 2),
			num(l, 2), num(b, 2), conf, ""}, reg, tinta, nil)
		cx := t.xCol(10) + t.ws[10]/2 - 4
		caixa(f.p, cx, topo+5, tinta)
		if it.Conferido {
			marcaOK(f.p, cx+1, topo+5, verde)
		}
	}
	t.linha([]string{"", "", "Totais", "", qtd(q), "", "", num(pl, 2), num(pb, 2), "", ""}, neg, tinta, &fundo)
	f.y += 10

	// Volumes.
	if len(d.Embalagens) > 0 {
		f.texto(mg, f.y+6, neg, 8.5, preto, "VOLUMES")
		f.y += 10
		tv := f.tabela(mg, cw, []col{{"Vol.", 0.4, pdfkit.AlignCenter}, {"Embalagem", 1, pdfkit.AlignLeft}, {"Marcação", 1, pdfkit.AlignLeft},
			{"Conteúdo", 2.2, pdfkit.AlignLeft}, {"C × L × A (cm)", 1.1, pdfkit.AlignCenter}, {"Cubagem", 0.7, pdfkit.AlignRight},
			{"Peso líq.", 0.7, pdfkit.AlignRight}, {"Peso bruto", 0.7, pdfkit.AlignRight}}, estiloTabela{cabFundo: mistura(ac, branco, 0.86), cabTexto: ac, grade: linha})
		tv.cabecalho()
		for _, v := range d.Embalagens {
			tv.linha([]string{fmt.Sprint(v.Numero), v.Embalagem, v.Marcacao, v.Conteudo,
				fmt.Sprintf("%s × %s × %s", num(v.Comprimento, 0), num(v.Largura, 0), num(v.Altura, 0)), num(v.Cubagem, 3), num(v.PesoLiquido, 2), num(v.PesoBruto, 2)}, reg, tinta, nil)
		}
		f.y += 10
	}

	// Totais da carga.
	if !d.PesoLiquido.IsPositive() {
		d.PesoLiquido = pl
	}
	if !d.PesoBruto.IsPositive() {
		d.PesoBruto = pb
	}
	f.cabe(40)
	tot := [][2]string{{"Volumes", fmt.Sprint(d.Volumes)}, {"Peso líquido", num(d.PesoLiquido, 2) + " kg"}, {"Peso bruto", num(d.PesoBruto, 2) + " kg"},
		{"Cubagem", num(d.Cubagem, 3) + " m³"}, {"Frete", moeda(d.Frete)}, {"Seguro", moeda(d.Seguro)}}
	tw := cw / float64(len(tot))
	for i, c := range tot {
		cx := mg + float64(i)*tw
		f.p.FillRect(cx+1, f.y, tw-2, 32, mistura(ac, branco, 0.9))
		f.texto(cx+8, f.y+11, neg, 5.9, ac, strings.ToUpper(c[0]))
		f.texto(cx+8, f.y+25, neg, 9.5, tinta, c[1])
	}
	f.y += 42

	// Etapas do romaneio.
	f.cabe(44)
	etapas := []struct {
		n string
		t *time.Time
	}{{"Criado", &d.Criado}, {"Separado", d.Separado}, {"Conferido", d.Conferido}, {"Despachado", d.Despachado}}
	ew := cw / float64(len(etapas))
	p = f.p
	p.FillRect(mg+ew/2, f.y+9, ew*float64(len(etapas)-1), 2, linha)
	for i, e := range etapas {
		cx := mg + ew*float64(i) + ew/2
		c := linha
		if e.t != nil {
			c = ac
		}
		p.FillRect(cx-6, f.y+4, 12, 12, c)
		if e.t != nil {
			marcaOK(p, cx-3.5, f.y+6, branco)
		}
		p.TextCenter(cx, f.y+27, neg, 7.2, tinta, e.n)
		p.TextCenter(cx, f.y+36, reg, 6.6, cinza, dataHoraP(e.t))
	}
	f.y += 46

	if obs := strings.TrimSpace(d.Observacao); obs != "" {
		ls := quebrar(obs, reg, 7.4, cw-20)
		hh := 16 + float64(len(ls))*9.5
		f.cabe(hh + 8)
		f.p.StrokeRect(mg, f.y, cw, hh, 0.6, linha)
		f.texto(mg+8, f.y+10, neg, 6.4, ac, "INSTRUÇÕES DE ENTREGA / OBSERVAÇÕES")
		for i, l := range ls {
			f.texto(mg+8, f.y+21+float64(i)*9.5, reg, 7.4, tinta, l)
		}
		f.y += hh + 10
	}

	// Assinaturas.
	f.cabe(150)
	p = f.p
	sw := (cw - 20) / 3
	for i, r := range []string{"Separado por (almoxarifado)", "Conferido por (expedição)", "Motorista — nome, RG e placa"} {
		sx := mg + float64(i)*(sw+10)
		p.StrokeLine(sx, f.y+30, sx+sw, f.y+30, 0.6, tinta)
		p.TextCenter(sx+sw/2, f.y+39, reg, 6.8, cinza, r)
		p.TextCenter(sx+sw/2, f.y+48, reg, 6.2, cinza, "Data ____/____/________  Hora ____:____")
	}
	f.y += 62

	// Canhoto destacável.
	p.StrokeDashedLine(mg, f.y, f.w-mg, f.y, 0.7, 4, 3, cinza)
	p.Text(mg, f.y-3, ita, 6, cinza, "destaque aqui")
	f.y += 8
	p.StrokeRect(mg, f.y, cw, 62, 0.8, preto)
	p.FillRect(f.w-mg-110, f.y, 110, 62, preto)
	p.Text(f.w-mg-100, f.y+14, neg, 6.4, ac, "ROMANEIO")
	p.Text(f.w-mg-100, f.y+36, neg, 16, branco, fmt.Sprintf("Nº %d", d.Codigo))
	if d.NFNumero != 0 {
		p.Text(f.w-mg-100, f.y+52, reg, 7.2, mistura(preto, branco, 0.7), fmt.Sprintf("NF-e %d", d.NFNumero))
	}
	for i, l := range quebrar(fmt.Sprintf("RECEBEMOS DE %s AS MERCADORIAS E VOLUMES CONSTANTES DESTE ROMANEIO.", strings.ToUpper(d.Emitente.Nome)), neg, 6.6, cw-130) {
		if i < 2 {
			p.Text(mg+8, f.y+10+float64(i)*8, neg, 6.6, tinta, l)
		}
	}
	p.StrokeLine(mg, f.y+22, f.w-mg-110, f.y+22, 0.5, linha)
	cwid := (cw - 110) / 3
	for i, r := range []string{"Data do recebimento", "Identificação do recebedor (nome e RG)", "Assinatura"} {
		cx := mg + float64(i)*cwid
		if i > 0 {
			p.StrokeLine(cx, f.y+22, cx, f.y+62, 0.5, linha)
		}
		p.Text(cx+6, f.y+31, neg, 5.8, cinza, strings.ToUpper(r))
	}
	f.y += 70
	return doc.Render()
}

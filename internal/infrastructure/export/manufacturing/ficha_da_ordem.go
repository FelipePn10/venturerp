package manufacturing

import (
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Ordem de produção impressa — a ficha que acompanha a peça no chão de
// fábrica, no formato das fichas de OP dos ERPs de manufatura (SAP "folha de
// controle da ordem", TOTVS "ordem de produção"): cabeçalho com o número da
// ordem e código de barras, dados da ordem em grade, roteiro com a ficha de
// apontamento de cada operação (externa: NF de remessa e retorno), lista de
// separação com conferência, ferramentas, serviços de terceiros, liberação da
// qualidade e assinaturas. Retrato A4.
//
// OrdemImpressa é o conteúdo já formatado (o que os testes conferem);
// RenderFichaDaOrdem só desenha.

type CampoImpresso struct{ Rotulo, Valor string }

type OperacaoImpressa struct {
	Seq, Operacao, Executa               string
	Entra, Prep, Exec, Total             string
	Situacao                             string
	Inspecao, Externa, Pulada, Concluida bool
	Inicio, Fim, Real                    string
	Fornecedor, OS, Prazo                string
}

type MaterialImpresso struct {
	Etapa, Codigo, Descricao, Necessario, Unidade, Deposito, Consumido string
}

type FerramentaImpressa struct{ Etapa, Codigo, Nome, Qtd string }

type TerceiroImpresso struct {
	OS, Etapa, Fornecedor, NaOrdem, Recebido, AReceber, Prazo, Situacao string
}

type OrdemImpressa struct {
	Numero, CodigoDeBarras, Situacao string
	Emissao                          string
	EmitidaPor                       string
	Grade                            [][]CampoImpresso // linhas da grade "dados da ordem"
	Observacoes                      []string
	Operacoes                        []OperacaoImpressa
	Materiais                        []MaterialImpresso
	Ferramentas                      []FerramentaImpressa
	Terceiros                        []TerceiroImpresso
	Rodape                           string
}

// MontarOrdemImpressa formata a ficha da ordem para impressão.
func MontarOrdemImpressa(f *entity.FichaDaOrdem, gerado time.Time) OrdemImpressa {
	un := strings.TrimSpace(f.Unidade)
	if un == "" {
		un = "UN"
	}
	item := codigoDoItem(f.ItemCode, f.ItemCodigo)
	if f.Mascara != "" {
		item += " · " + f.Mascara
	}
	roteiro := "Sem roteiro"
	if f.Roteiro != nil {
		roteiro = fmt.Sprintf("%d · alternativa %d", f.Roteiro.Codigo, f.Roteiro.Alternativa)
	}
	origem := rotuloOrigem(f.Origem)
	if f.Numero != 0 && f.Numero != f.ID {
		origem += fmt.Sprintf(" · nº %d", f.Numero)
	}
	if f.PedidoVenda != nil {
		origem += fmt.Sprintf(" · pedido %d", *f.PedidoVenda)
	}
	rotuloInicio, rotuloFim := "Início previsto", "Entrega prevista"
	switch f.Status {
	case "IN_PROGRESS":
		rotuloInicio = "Iniciada em"
	case "COMPLETED", "CLOSED":
		rotuloInicio, rotuloFim = "Iniciada em", "Concluída em"
	}
	deposito := "—"
	if f.Deposito != nil {
		deposito = fmt.Sprintf("%d", *f.Deposito)
	}
	revisao := vazioVira(f.Revisao, "—")

	o := OrdemImpressa{
		Numero:         fmt.Sprintf("%d", f.ID),
		CodigoDeBarras: fmt.Sprintf("OF%06d", f.ID),
		Situacao:       rotuloStatusOrdem(f.Status),
		Emissao:        gerado.Format("02/01/2006 15:04"),
		Grade: [][]CampoImpresso{
			{{"Produto", item}, {"Descrição", vazioVira(f.ItemNome, "—")}, {"Revisão", revisao}},
			{{"Quantidade a produzir", qtd(f.QtdPlanejada) + " " + un}, {"Produzido / refugo", qtd(f.QtdProduzida) + " / " + qtd(f.QtdRefugada)}, {"Depósito destino", deposito}, {"Prioridade", vazioVira(f.Prioridade, "Normal")}},
			{{rotuloInicio, data(f.Inicio)}, {rotuloFim, data(f.Fim)}, {"Roteiro", roteiro}, {"Origem", origem}},
		},
		Rodape: fmt.Sprintf("OF %d · %s", f.ID, item),
	}
	if obs := notasLegiveis(f.Observacoes); obs != "" {
		for _, l := range quebrarTexto(obs, 120) {
			o.Observacoes = append(o.Observacoes, l[0])
		}
	}

	osDaEtapa := map[int16]entity.OrdemTerceiroDaFicha{}
	for _, t := range f.OrdensTerceiro {
		if t.Etapa != nil {
			osDaEtapa[*t.Etapa] = t
		}
	}
	for _, e := range f.Etapas {
		op := OperacaoImpressa{
			Seq: fmt.Sprintf("%d", e.Sequencia), Operacao: e.Operacao, Executa: vazioVira(e.Centro, "—"),
			Entra: "—", Prep: horas(e.HorasPrep), Exec: horas(e.HorasPrevista), Total: horas(e.HorasPrep.Add(e.HorasPrevista)),
			Situacao: rotuloStatusEtapa(e.Status), Inspecao: e.Inspecao, Externa: e.Externa,
			Pulada: e.Status == "SKIPPED", Concluida: e.Status == "DONE",
			Inicio: dataHora(e.Inicio), Fim: dataHora(e.Fim), Real: horasReais(e),
		}
		if e.QtdEntra != nil {
			op.Entra = pecas(*e.QtdEntra, un)
		}
		if e.Externa {
			op.Executa = "Serviço externo"
			op.Fornecedor = vazioVira(e.Terceiro, "fornecedor não informado")
			op.Prep, op.Exec, op.Total = "—", "—", "—"
			if t, ok := osDaEtapa[e.Sequencia]; ok {
				op.OS = fmt.Sprintf("%d", t.Codigo)
				op.Prazo = data(t.Prazo)
				if t.Fornecedor != "" {
					op.Fornecedor = t.Fornecedor
				}
			}
		}
		if op.Pulada {
			op.Situacao = "Fantasma (pula)"
			op.Entra, op.Prep, op.Exec, op.Total = "—", "—", "—", "—"
		}
		if op.Concluida && op.Externa {
			op.Situacao = "Concluída (terceiro)"
		} else if op.Concluida && op.Real != "" {
			op.Situacao = "Concluída · " + op.Real + " h"
		}
		o.Operacoes = append(o.Operacoes, op)
	}

	for _, m := range f.Materiais {
		etapa := "Início"
		if m.Etapa != nil {
			etapa = fmt.Sprintf("%d", *m.Etapa)
		}
		codigo := codigoDoItem(m.ItemCode, m.ItemCodigo)
		if m.Mascara != "" {
			codigo += " · " + m.Mascara
		}
		consumido := ""
		if m.Atendida.IsPositive() {
			consumido = qtd(m.Atendida)
		}
		o.Materiais = append(o.Materiais, MaterialImpresso{
			Etapa: etapa, Codigo: codigo, Descricao: m.ItemNome, Necessario: qtd(m.Quantidade),
			Unidade: m.Unidade, Deposito: fmt.Sprintf("%d", m.Deposito), Consumido: consumido,
		})
	}
	for _, t := range f.Ferramentas {
		o.Ferramentas = append(o.Ferramentas, FerramentaImpressa{fmt.Sprintf("%d", t.Etapa), fmt.Sprintf("%d", t.Codigo), t.Nome, qtd(t.Quantidade)})
	}
	for _, t := range f.OrdensTerceiro {
		etapa := "—"
		if t.Etapa != nil {
			etapa = fmt.Sprintf("%d", *t.Etapa)
		}
		falta := decimal.Max(decimal.Zero, t.Quantidade.Sub(t.Recebida))
		o.Terceiros = append(o.Terceiros, TerceiroImpresso{
			OS: fmt.Sprintf("%d", t.Codigo), Etapa: etapa, Fornecedor: t.Fornecedor, NaOrdem: qtd(t.Quantidade),
			Recebido: qtd(t.Recebida), AReceber: qtd(falta), Prazo: data(t.Prazo), Situacao: rotuloStatusOS(t.Status),
		})
	}
	return o
}

func dataHora(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("02/01 15:04")
}

// RenderFichaDaOrdem desenha a ordem de produção.
func RenderFichaDaOrdem(f *entity.FichaDaOrdem, b Branding, gerado time.Time) []byte {
	return RenderFichaDaOrdemPor(f, b, gerado, "")
}

// RenderFichaDaOrdemPor desenha a ordem com quem a emitiu.
func RenderFichaDaOrdemPor(f *entity.FichaDaOrdem, b Branding, gerado time.Time, emitidaPor string) []byte {
	o := MontarOrdemImpressa(f, gerado)
	o.EmitidaPor = strings.TrimSpace(emitidaPor)
	return desenharOrdem(o, b)
}

// ─── desenho ────────────────────────────────────────────────────────────────

const (
	mg      = 28.0
	borda   = 0.6
	rodapeH = 26.0
)

var (
	corGrade  = pdfkit.Color{R: 150, G: 150, B: 150}
	corFundo  = pdfkit.Color{R: 238, G: 238, B: 238}
	corRotulo = pdfkit.Color{R: 70, G: 70, B: 70}
	corApagar = pdfkit.Color{R: 140, G: 140, B: 140}
)

type folha struct {
	doc      *pdfkit.Doc
	p        *pdfkit.Page
	th       pdfkit.Theme
	b        Branding
	logo     *pdfkit.Image
	o        OrdemImpressa
	w, h, cw float64
	y        float64
	secoes   int
}

func desenharOrdem(o OrdemImpressa, b Branding) []byte {
	doc := pdfkit.New()
	f := &folha{doc: doc, th: temaPretoEBranco(), b: b, o: o}
	f.logo, _ = doc.AddImage(logoEmCinza(b.Logo))
	f.w, f.h = doc.Size()
	f.cw = f.w - 2*mg
	doc.SetFooter(func(p *pdfkit.Page, n, total int) {
		top := f.h - mg - 10
		p.StrokeLine(mg, top-8, f.w-mg, top-8, borda, corGrade)
		p.Text(mg, top+2, pdfkit.FontRegular, 7, corRotulo, fit("VentureERP · "+o.Rodape, pdfkit.FontRegular, 7, f.cw*0.4))
		p.TextCenter(f.w/2, top+2, pdfkit.FontOblique, 6.8, corRotulo, "Emitida em "+o.Emissao+rotular(" por ", o.EmitidaPor)+" · a situação válida é a do sistema (VPRO0900)")
		p.TextRight(f.w-mg, top+2, pdfkit.FontBold, 7, corRotulo, fmt.Sprintf("Página %d de %d", n, total))
	})

	f.p = doc.AddPage()
	f.y = f.cabecalho()
	f.grade()
	if len(o.Observacoes) > 0 {
		f.observacoes()
	}
	f.operacoes()
	if len(o.Materiais) > 0 {
		f.materiais()
	}
	if len(o.Ferramentas) > 0 {
		f.ferramentas()
	}
	if len(o.Terceiros) > 0 {
		f.terceiros()
	}
	f.liberacao()
	return doc.Render()
}

func (f *folha) limite() float64 { return f.h - mg - rodapeH }

// cabe garante altura na página; quando não cabe abre outra, com o cabeçalho
// curto, e devolve true (a tabela redesenha o cabeçalho dela).
func (f *folha) cabe(altura float64) bool {
	if f.y+altura <= f.limite() {
		return false
	}
	f.p = f.doc.AddPage()
	f.y = f.cabecalhoCurto()
	return true
}

func (f *folha) cabecalho() float64 {
	p, th, top := f.p, f.th, mg
	const altura = 78.0
	esqW, dirW := 210.0, 156.0
	meioW := f.cw - esqW - dirW
	p.StrokeRect(mg, top, f.cw, altura, 1, f.th.Brand)
	p.StrokeLine(mg+esqW, top, mg+esqW, top+altura, borda, corGrade)
	p.StrokeLine(mg+esqW+meioW, top, mg+esqW+meioW, top+altura, borda, corGrade)

	// Empresa.
	x := mg + 8
	if f.logo != nil {
		lw, lh := f.logo.Fit(54, 40)
		p.DrawImage(f.logo, x+(54-lw)/2, top+8+(40-lh)/2, lw, lh)
		x += 62
	}
	larg := mg + esqW - x - 6
	co := f.b.Company
	p.Text(x, top+16, pdfkit.FontBold, 9.5, th.Title, fit(vazioVira(co.Name, "Empresa"), pdfkit.FontBold, 9.5, larg))
	yy := top + 27.0
	for _, l := range []string{juntar(" · ", rotular("CNPJ ", co.CNPJ), rotular("IE ", co.IE)), co.Address, juntar(" · ", co.Phone, co.Email)} {
		if strings.TrimSpace(l) == "" {
			continue
		}
		p.Text(x, yy, pdfkit.FontRegular, 6.6, corRotulo, fit(l, pdfkit.FontRegular, 6.6, larg))
		yy += 8.5
	}

	// Título.
	cx := mg + esqW + meioW/2
	p.TextCenter(cx, top+24, pdfkit.FontBold, 14, th.Title, "ORDEM DE PRODUÇÃO")
	p.TextCenter(cx, top+35, pdfkit.FontRegular, 7, corRotulo, "Ficha de fabricação, separação e apontamento")
	pill := pdfkit.TextWidth(pdfkit.FontBold, 7.5, strings.ToUpper(f.o.Situacao)) + 18
	p.FillRect(cx-pill/2, top+46, pill, 15, th.Brand)
	p.TextCenter(cx, top+56.5, pdfkit.FontBold, 7.5, th.BrandText, strings.ToUpper(f.o.Situacao))
	p.TextCenter(cx, top+71, pdfkit.FontRegular, 6.5, corRotulo, "Emissão "+f.o.Emissao)

	// Número e código de barras.
	dx := mg + esqW + meioW
	p.FillRect(dx, top, dirW, 15, corFundo)
	p.StrokeLine(dx, top+15, dx+dirW, top+15, borda, corGrade)
	p.Text(dx+8, top+10.5, pdfkit.FontBold, 6.5, corRotulo, "Nº DA ORDEM DE PRODUÇÃO")
	p.TextCenter(dx+dirW/2, top+38, pdfkit.FontBold, 21, th.Title, "OF "+f.o.Numero)
	if err := p.DrawCode128B(f.o.CodigoDeBarras, dx+12, top+45, dirW-24, 22, pdfkit.Black); err == nil {
		p.TextCenter(dx+dirW/2, top+74.5, pdfkit.FontRegular, 6, corRotulo, f.o.CodigoDeBarras)
	}
	return top + altura + 10
}

func (f *folha) cabecalhoCurto() float64 {
	p, top := f.p, mg
	p.FillRect(mg, top, f.cw, 22, f.th.Brand)
	p.Text(mg+8, top+14.5, pdfkit.FontBold, 8.5, f.th.BrandText, fit(vazioVira(f.b.Company.Name, "Empresa"), pdfkit.FontBold, 8.5, f.cw/2))
	p.TextRight(f.w-mg-8, top+14.5, pdfkit.FontBold, 9, f.th.BrandText, "ORDEM DE PRODUÇÃO · OF "+f.o.Numero+" (continuação)")
	return top + 32
}

func (f *folha) secao(titulo, nota string) {
	f.cabe(40)
	f.secoes++
	p := f.p
	p.FillRect(mg, f.y, 3, 11, f.th.Brand)
	t := fmt.Sprintf("%d. %s", f.secoes, strings.ToUpper(titulo))
	p.Text(mg+8, f.y+9, pdfkit.FontBold, 8.5, f.th.Title, t)
	if nota != "" {
		x := mg + 8 + pdfkit.TextWidth(pdfkit.FontBold, 8.5, t) + 8
		p.Text(x, f.y+9, pdfkit.FontOblique, 6.8, corRotulo, fit(nota, pdfkit.FontOblique, 6.8, f.w-mg-x))
	}
	f.y += 15
}

// celula desenha um campo de formulário: rótulo pequeno em cima, valor embaixo.
func (f *folha) celula(x, top, w, h float64, rotulo, valor string, negrito bool) {
	p := f.p
	p.StrokeRect(x, top, w, h, borda, corGrade)
	if valor != "" {
		// Campo já preenchido pelo sistema: o gabarito de preenchimento sai.
		if i := strings.Index(rotulo, "  _"); i > 0 {
			rotulo = rotulo[:i]
		}
	}
	p.Text(x+4, top+7.5, pdfkit.FontBold, 5.8, corRotulo, fit(strings.ToUpper(rotulo), pdfkit.FontBold, 5.8, w-8))
	fonte, tam := pdfkit.FontRegular, 8.5
	if negrito {
		fonte = pdfkit.FontBold
	}
	if valor != "" {
		p.Text(x+4, top+h-5, fonte, tam, f.th.Text, fit(valor, fonte, tam, w-8))
	}
}

func (f *folha) grade() {
	f.secao("Dados da ordem", "")
	pesos := [][]float64{{1.3, 2.3, 0.6}, {1, 1, 0.8, 0.8}, {1, 1, 1.1, 1.3}}
	for i, linha := range f.o.Grade {
		ws := distribuir(pesos[i%len(pesos)], f.cw)
		x := mg
		for j, c := range linha {
			f.celula(x, f.y, ws[j], 24, c.Rotulo, c.Valor, i == 0 || j < 2)
			x += ws[j]
		}
		f.y += 24
	}
	f.y += 10
}

func (f *folha) observacoes() {
	f.secao("Observações e instruções", "")
	h := float64(len(f.o.Observacoes))*10 + 8
	f.cabe(h)
	f.p.FillRect(mg, f.y, f.cw, h, corFundo)
	f.p.StrokeRect(mg, f.y, f.cw, h, borda, corGrade)
	yy := f.y + 11
	for _, l := range f.o.Observacoes {
		f.p.Text(mg+6, yy, pdfkit.FontRegular, 8, f.th.Text, fit(l, pdfkit.FontRegular, 8, f.cw-12))
		yy += 10
	}
	f.y += h + 10
}

type coluna struct {
	titulo string
	peso   float64
	alinha pdfkit.Align
}

func (f *folha) cabecalhoTabela(cols []coluna) []float64 {
	pesos := make([]float64, len(cols))
	for i, c := range cols {
		pesos[i] = c.peso
	}
	ws := distribuir(pesos, f.cw)
	p := f.p
	p.FillRect(mg, f.y, f.cw, 14, corFundo)
	p.StrokeRect(mg, f.y, f.cw, 14, borda, corGrade)
	x := mg
	for i, c := range cols {
		cell(p, strings.ToUpper(c.titulo), pdfkit.FontBold, 6.2, corRotulo, c.alinha, x, f.y+9.5, ws[i])
		x += ws[i]
	}
	f.y += 14
	return ws
}

func (f *folha) linhaTabela(cols []coluna, ws []float64, valores []string, h float64, fonte pdfkit.Font, cor pdfkit.Color, fundo *pdfkit.Color) {
	p := f.p
	if fundo != nil {
		p.FillRect(mg, f.y, f.cw, h, *fundo)
	}
	x := mg
	for i, c := range cols {
		p.StrokeRect(x, f.y, ws[i], h, borda, corGrade)
		if i < len(valores) {
			cell(p, valores[i], fonte, 7.8, cor, c.alinha, x, f.y+h/2+2.8, ws[i])
		}
		x += ws[i]
	}
	f.y += h
}

func (f *folha) operacoes() {
	f.secao("Roteiro de operações e apontamento", "preencha a ficha de cada operação ao executá-la")
	cols := []coluna{
		{"Seq.", 0.45, pdfkit.AlignCenter}, {"Operação", 2.1, pdfkit.AlignLeft}, {"Centro / executante", 1.9, pdfkit.AlignLeft},
		{"Qtd.", 0.6, pdfkit.AlignRight}, {"Prep. h", 0.55, pdfkit.AlignRight}, {"Exec. h", 0.55, pdfkit.AlignRight},
		{"Total h", 0.55, pdfkit.AlignRight}, {"Situação", 1.35, pdfkit.AlignLeft},
	}
	ws := f.cabecalhoTabela(cols)
	if len(f.o.Operacoes) == 0 {
		f.linhaTabela(cols, ws, []string{"", "Ordem sem roteiro: produção apontada só na ordem.", "", "", "", "", "", ""}, 16, pdfkit.FontOblique, corRotulo, nil)
	}
	for _, op := range f.o.Operacoes {
		altura := 17.0
		if !op.Pulada {
			altura += 22
		}
		if f.cabe(altura + 2) {
			ws = f.cabecalhoTabela(cols)
		}
		nome := op.Operacao
		if op.Inspecao {
			nome += "  [INSPEÇÃO]"
		}
		cor := f.th.Text
		if op.Pulada {
			cor = corApagar
		}
		fundo := corFundo
		f.linhaTabela(cols, ws, []string{op.Seq, nome, op.Executa, op.Entra, op.Prep, op.Exec, op.Total, op.Situacao}, 17, pdfkit.FontBold, cor, &fundo)
		if op.Pulada {
			continue
		}
		// Ficha de apontamento da operação.
		if op.Externa {
			campos := []CampoImpresso{
				{"Fornecedor" + rotular(" · OS ", op.OS) + rotular(" · prazo ", op.Prazo), op.Fornecedor},
				{"NF de remessa", ""}, {"Saída  _ _ / _ _", ""}, {"NF de retorno", ""}, {"Retorno  _ _ / _ _", ""}, {"Qtd. recebida", ""}, {"Visto", ""},
			}
			f.fichaApontamento(campos, []float64{2.6, 0.9, 0.9, 0.9, 0.9, 0.8, 0.7})
			continue
		}
		campos := []CampoImpresso{
			{"Início  _ _ / _ _  _ _ : _ _", op.Inicio}, {"Término  _ _ / _ _  _ _ : _ _", op.Fim}, {"Qtd. boa", ""}, {"Refugo / motivo", ""},
			{"Operador (matrícula)", ""}, {"Horas", op.Real}, {"Visto", ""},
		}
		f.fichaApontamento(campos, []float64{1.25, 1.25, 0.7, 1.1, 1.3, 0.6, 0.6})
	}
	f.y += 10
}

func (f *folha) fichaApontamento(campos []CampoImpresso, pesos []float64) {
	ws := distribuir(pesos, f.cw-14)
	f.p.FillRect(mg, f.y, 14, 22, corFundo)
	f.p.StrokeRect(mg, f.y, 14, 22, borda, corGrade)
	x := mg + 14
	for i, c := range campos {
		f.celula(x, f.y, ws[i], 22, c.Rotulo, c.Valor, false)
		x += ws[i]
	}
	f.y += 22
}

func (f *folha) materiais() {
	f.secao("Lista de separação de materiais", "o almoxarifado separa por etapa; marque a conferência")
	cols := []coluna{
		{"Etapa", 0.5, pdfkit.AlignCenter}, {"Código", 1.35, pdfkit.AlignLeft}, {"Descrição", 2.4, pdfkit.AlignLeft},
		{"Necessário", 0.8, pdfkit.AlignRight}, {"Un.", 0.4, pdfkit.AlignCenter}, {"Dep.", 0.45, pdfkit.AlignCenter},
		{"Lote", 0.8, pdfkit.AlignLeft}, {"Separado", 0.75, pdfkit.AlignRight}, {"Conf.", 0.45, pdfkit.AlignCenter},
	}
	ws := f.cabecalhoTabela(cols)
	for _, m := range f.o.Materiais {
		if f.cabe(18) {
			ws = f.cabecalhoTabela(cols)
		}
		topo := f.y
		f.linhaTabela(cols, ws, []string{m.Etapa, m.Codigo, m.Descricao, m.Necessario, m.Unidade, m.Deposito, "", m.Consumido, ""}, 18, pdfkit.FontRegular, f.th.Text, nil)
		f.caixinha(mg+soma(ws[:8])+ws[8]/2-4, topo+5)
	}
	f.y += 10
}

func (f *folha) caixinha(x, top float64) {
	f.p.StrokeRect(x, top, 8, 8, 0.8, f.th.Text)
}

func (f *folha) ferramentas() {
	f.secao("Ferramentas e dispositivos", "")
	cols := []coluna{{"Etapa", 0.5, pdfkit.AlignCenter}, {"Código", 1, pdfkit.AlignLeft}, {"Ferramenta / dispositivo", 4, pdfkit.AlignLeft}, {"Qtd.", 0.6, pdfkit.AlignRight}, {"Retirada", 0.9, pdfkit.AlignCenter}, {"Devolvida", 0.9, pdfkit.AlignCenter}}
	ws := f.cabecalhoTabela(cols)
	for _, t := range f.o.Ferramentas {
		if f.cabe(18) {
			ws = f.cabecalhoTabela(cols)
		}
		topo := f.y
		f.linhaTabela(cols, ws, []string{t.Etapa, t.Codigo, t.Nome, t.Qtd, "", ""}, 18, pdfkit.FontRegular, f.th.Text, nil)
		f.caixinha(mg+soma(ws[:4])+ws[4]/2-4, topo+5)
		f.caixinha(mg+soma(ws[:5])+ws[5]/2-4, topo+5)
	}
	f.y += 10
}

func (f *folha) terceiros() {
	f.secao("Serviços de terceiros", "remessa e retorno no VTPS0100: o recebimento total conclui a etapa")
	cols := []coluna{
		{"OS", 0.45, pdfkit.AlignCenter}, {"Etapa", 0.45, pdfkit.AlignCenter}, {"Fornecedor", 2.4, pdfkit.AlignLeft},
		{"Na ordem", 0.7, pdfkit.AlignRight}, {"Recebido", 0.7, pdfkit.AlignRight}, {"A receber", 0.7, pdfkit.AlignRight},
		{"Prazo", 0.75, pdfkit.AlignCenter}, {"Situação", 1.1, pdfkit.AlignLeft},
	}
	ws := f.cabecalhoTabela(cols)
	for _, t := range f.o.Terceiros {
		if f.cabe(17) {
			ws = f.cabecalhoTabela(cols)
		}
		f.linhaTabela(cols, ws, []string{t.OS, t.Etapa, t.Fornecedor, t.NaOrdem, t.Recebido, t.AReceber, t.Prazo, t.Situacao}, 17, pdfkit.FontRegular, f.th.Text, nil)
	}
	f.y += 10
}

func (f *folha) liberacao() {
	f.cabe(15 + 26 + 8 + 46)
	f.secao("Liberação da qualidade e responsáveis", "")
	p := f.p
	// Inspeção final.
	ws := distribuir([]float64{2.6, 0.9, 1.3, 0.8}, f.cw)
	p.StrokeRect(mg, f.y, ws[0], 26, borda, corGrade)
	p.Text(mg+4, f.y+7.5, pdfkit.FontBold, 5.8, corRotulo, "INSPEÇÃO FINAL")
	x := mg + 6
	for _, op := range []string{"Aprovado", "Aprovado com restrição", "Reprovado"} {
		f.caixinha(x, f.y+12)
		p.Text(x+11, f.y+19, pdfkit.FontRegular, 7.5, f.th.Text, op)
		x += 22 + pdfkit.TextWidth(pdfkit.FontRegular, 7.5, op)
	}
	xx := mg + ws[0]
	for i, r := range []string{"Qtd. aprovada", "Inspetor", "Data"} {
		f.celula(xx, f.y, ws[i+1], 26, r, "", false)
		xx += ws[i+1]
	}
	f.y += 34
	// Assinaturas.
	rotulos := []string{"Emitida por — PCP", "Separação — Almoxarifado", "Execução — Produção", "Liberação — Qualidade"}
	larg := (f.cw - 3*8) / 4
	for i, r := range rotulos {
		bx := mg + float64(i)*(larg+8)
		p.StrokeRect(bx, f.y, larg, 44, borda, corGrade)
		p.StrokeLine(bx+8, f.y+28, bx+larg-8, f.y+28, borda, corRotulo)
		p.TextCenter(bx+larg/2, f.y+36, pdfkit.FontBold, 6.5, corRotulo, strings.ToUpper(r))
		p.Text(bx+8, f.y+10, pdfkit.FontRegular, 6.2, corRotulo, "Data  _ _ / _ _ / _ _ _ _")
		if i == 0 && f.o.EmitidaPor != "" {
			p.TextCenter(bx+larg/2, f.y+24, pdfkit.FontRegular, 7.5, f.th.Text, fit(f.o.EmitidaPor, pdfkit.FontRegular, 7.5, larg-16))
		}
	}
	f.y += 52
}

func distribuir(pesos []float64, total float64) []float64 {
	soma := 0.0
	for _, p := range pesos {
		soma += p
	}
	out := make([]float64, len(pesos))
	for i, p := range pesos {
		out[i] = total * p / soma
	}
	return out
}

func soma(v []float64) float64 {
	t := 0.0
	for _, x := range v {
		t += x
	}
	return t
}

func juntar(sep string, partes ...string) string {
	var out []string
	for _, p := range partes {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return strings.Join(out, sep)
}

func rotular(prefixo, valor string) string {
	if strings.TrimSpace(valor) == "" {
		return ""
	}
	return prefixo + valor
}

package manufacturing

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/shopspring/decimal"
)

func fichaDeTeste() *entity.FichaDaOrdem {
	entra := decimal.NewFromFloat(9.2)
	montagem, zincagem := int16(30), int16(20)
	prazo := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	inicio := time.Date(2026, 11, 16, 8, 5, 0, 0, time.UTC)
	return &entity.FichaDaOrdem{
		ID: 77, Numero: 900077, ItemCode: 30016, ItemCodigo: "PORTA-01", ItemNome: "PORTA", Status: "IN_PROGRESS",
		QtdPlanejada: decimal.NewFromInt(10), Unidade: "UN", Observacoes: "parent_item=30016",
		Etapas: []entity.EtapaDaFicha{
			{Sequencia: 10, Operacao: "Corte", Centro: "1 — Serra", Status: "DONE", HorasPrep: decimal.NewFromFloat(0.25), HorasPrevista: decimal.NewFromFloat(1.5), HorasReais: decimal.NewFromFloat(1.25), QtdEntra: &entra, Inicio: &inicio},
			{Sequencia: 20, Operacao: "Zincagem", Externa: true, Terceiro: "900 — Galvano", Status: "IN_PROGRESS"},
			{Sequencia: 25, Operacao: "Rebarba", Status: "SKIPPED"},
			{Sequencia: 30, Operacao: "Pintura", Centro: "4 — Cabine", Status: "PENDING", Inspecao: true},
		},
		Materiais:      []entity.MaterialDaFicha{{ItemCode: 5, ItemCodigo: "DOB-01", ItemNome: "Dobradiça", Quantidade: decimal.NewFromInt(20), Atendida: decimal.NewFromInt(8), Unidade: "UN", Etapa: &montagem}},
		OrdensTerceiro: []entity.OrdemTerceiroDaFicha{{Codigo: 3, Etapa: &zincagem, Fornecedor: "900 — Galvano Ltda", Quantidade: decimal.NewFromInt(10), Recebida: decimal.NewFromInt(4), Prazo: &prazo, Status: "RELEASED_WITHOUT_PO"}},
	}
}

func TestOrdemImpressaDizQuemFazOQueEQuando(t *testing.T) {
	o := MontarOrdemImpressa(fichaDeTeste(), time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC))
	if o.Numero != "77" || o.CodigoDeBarras != "OF000077" || o.Situacao != "Em produção" {
		t.Fatalf("cabeçalho: %+v", o)
	}
	if o.Grade[0][0].Valor != "PORTA-01 (30016)" {
		t.Fatalf("o produto sai pelo código de negócio: %q", o.Grade[0][0].Valor)
	}
	if o.Grade[2][0].Rotulo != "Iniciada em" || !strings.Contains(o.Grade[2][3].Valor, "900077") {
		t.Fatalf("data real em produção e nº do planejamento na origem: %+v", o.Grade[2])
	}
	if len(o.Observacoes) != 1 || strings.Contains(o.Observacoes[0], "parent_item") || !strings.Contains(o.Observacoes[0], "item 30016") {
		t.Fatalf("nota técnica do MRP não pode vazar: %v", o.Observacoes)
	}
	corte, zinc, rebarba, pintura := o.Operacoes[0], o.Operacoes[1], o.Operacoes[2], o.Operacoes[3]
	if corte.Entra != "10" || corte.Total != "1,75" || corte.Situacao != "Concluída · 1,25 h" || corte.Inicio != "16/11 08:05" {
		t.Fatalf("operação interna concluída: %+v", corte)
	}
	if !zinc.Externa || zinc.Executa != "Serviço externo" || zinc.OS != "3" || zinc.Prazo != "20/11/2026" || zinc.Fornecedor != "900 — Galvano Ltda" {
		t.Fatalf("operação externa leva fornecedor, OS e prazo: %+v", zinc)
	}
	if !rebarba.Pulada || rebarba.Situacao != "Fantasma (pula)" {
		t.Fatalf("fantasma: %+v", rebarba)
	}
	if !pintura.Inspecao || pintura.Real != "" {
		t.Fatalf("pintura pendente com inspeção, sem horas reais: %+v", pintura)
	}
	if m := o.Materiais[0]; m.Etapa != "30" || m.Codigo != "DOB-01 (5)" || m.Consumido != "8" {
		t.Fatalf("material da etapa: %+v", m)
	}
	if tp := o.Terceiros[0]; tp.AReceber != "6" || tp.Situacao != "Liberada sem pedido" {
		t.Fatalf("serviço de terceiro: %+v", tp)
	}
}

func TestOrdemImpressaRendeAPaginacao(t *testing.T) {
	f := fichaDeTeste()
	for i := 0; i < 60; i++ {
		f.Materiais = append(f.Materiais, entity.MaterialDaFicha{ItemCode: int64(100 + i), ItemNome: "Componente", Quantidade: decimal.NewFromInt(1), Unidade: "UN"})
	}
	pdf := RenderFichaDaOrdem(f, Branding{}, time.Now())
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("não gerou PDF")
	}
	if n := bytes.Count(pdf, []byte("/Type /Page /Parent")); n < 2 {
		t.Fatalf("61 materiais tinham de ocupar mais de uma página; vieram %d", n)
	}
}

func TestOrdemImpressaEmPretoEBranco(t *testing.T) {
	// Logo vermelha 4×4: sai em cinza.
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 220, G: 30, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	cinza, err := png.Decode(bytes.NewReader(logoEmCinza(buf.Bytes())))
	if err != nil {
		t.Fatal(err)
	}
	if r, g, b, _ := cinza.At(1, 1).RGBA(); r != g || g != b {
		t.Fatalf("logo continua colorida: %d %d %d", r, g, b)
	}
	// Toda cor usada no desenho é cinza (R = G = B), mesmo com cor de marca.
	pdf := RenderFichaDaOrdem(fichaDeTeste(), Branding{BrandColorHex: "#C0392B", Logo: buf.Bytes()}, time.Now())
	cor := regexp.MustCompile(`([0-9.]+) ([0-9.]+) ([0-9.]+) (rg|RG)`)
	achou := 0
	for _, m := range cor.FindAllSubmatch(pdf, -1) {
		achou++
		if string(m[1]) != string(m[2]) || string(m[2]) != string(m[3]) {
			t.Fatalf("cor fora do preto e branco: %s", m[0])
		}
	}
	if achou == 0 {
		t.Skip("conteúdo do PDF comprimido: a checagem de cor não se aplica")
	}
}

// "Iniciada em" mostra o primeiro apontamento real; sem apontamento fica o previsto.
func TestOrdemImpressaUsaADataReal(t *testing.T) {
	previsto := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	f := fichaDeTeste()
	f.Inicio = &previsto
	if o := MontarOrdemImpressa(f, time.Now()); o.Grade[2][0].Valor != "16/11/2026" {
		t.Fatalf("em produção vale o início apontado: %+v", o.Grade[2][0])
	}
	for i := range f.Etapas {
		f.Etapas[i].Inicio = nil
	}
	if o := MontarOrdemImpressa(f, time.Now()); o.Grade[2][0].Rotulo != "Início previsto" || o.Grade[2][0].Valor != "10/11/2026" {
		t.Fatalf("sem apontamento fica o previsto: %+v", o.Grade[2][0])
	}
	fim := time.Date(2026, 11, 18, 17, 0, 0, 0, time.UTC)
	f.Status, f.Etapas[0].Inicio, f.Etapas[0].Fim = "COMPLETED", &previsto, &fim
	if o := MontarOrdemImpressa(f, time.Now()); o.Grade[2][1].Rotulo != "Concluída em" || o.Grade[2][1].Valor != "18/11/2026" {
		t.Fatalf("concluída mostra o último término apontado: %+v", o.Grade[2][1])
	}
}

// A prioridade sai em português; a instrução e as pausas da etapa, o operador e
// as assinaturas com data saem do que o sistema registrou.
func TestOrdemImpressaPrioridadeHistoricoEAssinaturas(t *testing.T) {
	f := fichaDeTeste()
	f.Prioridade, f.Status = "HIGH", "COMPLETED"
	f.Observacoes = "Rebarbar as bordas antes da pintura"
	h := func(hh, mm int) time.Time { return time.Date(2026, 11, 16, hh, mm, 0, 0, time.UTC) }
	f.Etapas[0].Instrucao = "Conferir   o esquadro"
	f.Etapas[0].Eventos = []entity.EventoDaEtapa{
		{Status: "IN_PROGRESS", Quando: h(8, 5), Quem: "Ana"},
		{Status: "PAUSED", Quando: h(10, 0), Quem: "Ana", Motivo: "almoço"},
		{Status: "IN_PROGRESS", Quando: h(11, 0), Quem: "Bruno"},
		{Status: "DONE", Quando: h(12, 0), Quem: "Bruno"},
	}
	emitida, separada, executada := h(7, 0), h(7, 30), h(12, 0)
	f.Emissao = entity.AssinaturaDaFicha{Nome: "Carla PCP", Quando: &emitida}
	f.Separacao = entity.AssinaturaDaFicha{Nome: "Davi", Quando: &separada}
	f.Execucao = entity.AssinaturaDaFicha{Nome: "Bruno", Quando: &executada}
	f.InspecaoFinal = &entity.InspecaoFinalDaFicha{Decisao: "ACEITO", QtdAceita: decimal.NewFromInt(10), Inspetor: "Eva", Quando: h(13, 0)}
	f.Qualidade = entity.AssinaturaDaFicha{Nome: "Eva", Quando: &executada}

	o := MontarOrdemImpressa(f, h(14, 0))
	if got := o.Grade[1][3].Valor; got != "Alta" {
		t.Fatalf("prioridade = %q, quer Alta", got)
	}
	if len(o.Observacoes) != 1 || o.Observacoes[0] != "Rebarbar as bordas antes da pintura" {
		t.Fatalf("observação da ordem: %v", o.Observacoes)
	}
	corte := o.Operacoes[0]
	if corte.Operador != "Bruno" {
		t.Fatalf("operador = %q, quer quem concluiu", corte.Operador)
	}
	if len(corte.Notas) != 2 || corte.Notas[0] != "Instrução: Conferir o esquadro" ||
		corte.Notas[1] != "Histórico: Pausada 16/11 10:00 (almoço) · Ana; Retomada 16/11 11:00 · Bruno" {
		t.Fatalf("notas da etapa: %q", corte.Notas)
	}
	quer := [4]AssinaturaImpressa{{"Carla PCP", "16/11/2026"}, {"Davi", "16/11/2026"}, {"Bruno", "16/11/2026"}, {"Eva", "16/11/2026"}}
	if o.Assinaturas != quer {
		t.Fatalf("assinaturas = %+v", o.Assinaturas)
	}
	if o.Inspecao != (InspecaoImpressa{"Aprovado", "10 UN", "Eva", "16/11/2026"}) {
		t.Fatalf("inspeção = %+v", o.Inspecao)
	}
	if len(RenderFichaDaOrdem(f, Branding{}, h(14, 0))) == 0 {
		t.Fatal("PDF vazio")
	}
	for p, quer := range map[string]string{"": "Normal", "LOW": "Baixa", "URGENTE": "Urgente", "urgent": "Urgente", "3": "3"} {
		if got := rotuloPrioridade(p); got != quer {
			t.Errorf("rotuloPrioridade(%q) = %q, quer %q", p, got, quer)
		}
	}
}

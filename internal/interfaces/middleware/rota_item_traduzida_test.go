package middleware

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Rotas que têm um parâmetro com cara de item mas NÃO referenciam o cadastro de
// item. Cada uma precisa de justificativa — a lista existe para que ninguém
// silencie o teste por engano.
var rotasSemItem = map[string]string{
	// Identificador da LINHA do documento, não do cadastro de item.
	"/api/sales-order/items/{itemCode}":                "identificador da linha do pedido",
	"/api/sales-order/items/{itemCode}/cancel":         "identificador da linha do pedido",
	"/api/sales-quotation/items/{itemCode}":            "identificador da linha da cotação",
	"/api/sales-quotation/items/{itemCode}/cancel":     "identificador da linha da cotação",
	"/api/consumer-service/calls/checklist/{itemCode}": "identificador do item do checklist",

	// O handler já aceita o código público: resolve por request.TextCode /
	// itemresolution.Resolve. Traduzir aqui também converteria duas vezes e
	// gravaria no item errado.
	"/api/items/structure/{itemCode}/configuration-check": "handler resolve com request.TextCode",
	"/api/items/structure/where-used/{itemCode}":          "handler resolve com request.TextCode",
	"/api/items/structure/{itemCode}/history":             "handler resolve com request.TextCode",
	"/api/items/structure/{itemCode}/configurator":        "handler resolve com request.TextCode",
	"/api/items/structure/{itemCode}/configurator/apply":  "handler resolve com request.TextCode",

	// {code} aqui é código de máscara/classificação, não de item.
	"/api/items/classifications/masks/{code}":      "código de máscara de classificação",
	"/api/items/classifications/{maskCode}/{code}": "código de classificação",
}

var (
	reRoute  = regexp.MustCompile(`r\.Route\("([^"]+)"`)
	reMetodo = regexp.MustCompile(`\.(?:Get|Post|Put|Patch|Delete)\("([^"]+)"`)
	reParam  = regexp.MustCompile(`\{(itemCode|item_code)\}`)
)

// TestTodaRotaDeItemEhTraduzida cobra que toda rota que recebe código de item no
// caminho seja traduzida pelo middleware — ou esteja isenta explicitamente.
//
// Existe por causa de um defeito que passou despercebido por muito tempo: as
// rotas de estoque ficam atrás de `r.Route("/api/stock", …)`, um mount com
// curinga, e o middleware de tradução rodava DEPOIS do roteamento. O handler
// recebia o código público cru. Com código alfanumérico dava 400; com código
// numérico devolvia lista vazia e ATP zerado em silêncio, o que é pior, porque
// vira promessa de entrega errada sem nenhum erro na tela.
func TestTodaRotaDeItemEhTraduzida(t *testing.T) {
	fonte, err := os.ReadFile("../../../api/api.go")
	if err != nil {
		t.Fatalf("não consegui ler api.go: %v", err)
	}

	type quadro struct {
		prefixo string
		nivel   int
	}
	var pilha []quadro
	nivel := 0
	var faltando []string
	vistas := 0

	for _, linha := range strings.Split(string(fonte), "\n") {
		if m := reMetodo.FindStringSubmatch(linha); m != nil {
			var sb strings.Builder
			for _, q := range pilha {
				sb.WriteString(strings.TrimSuffix(q.prefixo, "/"))
			}
			rota := sb.String() + m[1]
			rota = strings.TrimSuffix(rota, "/")
			if referenciaItem(rota) {
				vistas++
				if _, isento := rotasSemItem[rota]; !isento && !rotaTraduzida(rota) {
					faltando = append(faltando, rota)
				}
			}
		}
		if m := reRoute.FindStringSubmatch(linha); m != nil {
			pilha = append(pilha, quadro{prefixo: m[1], nivel: nivel})
		}
		nivel += strings.Count(linha, "{") - strings.Count(linha, "}")
		for len(pilha) > 0 && nivel <= pilha[len(pilha)-1].nivel {
			pilha = pilha[:len(pilha)-1]
		}
	}

	if vistas == 0 {
		t.Fatal("nenhuma rota com código de item encontrada — o leitor de api.go quebrou, e um teste que não testa nada é pior que teste nenhum")
	}
	if len(faltando) > 0 {
		t.Errorf("%d de %d rotas recebem código de item no caminho mas não são traduzidas.\n"+
			"Inclua o prefixo em itemPathPatterns ou justifique em rotasSemItem:\n  %s",
			len(faltando), vistas, strings.Join(faltando, "\n  "))
	}
}

// referenciaItem diz se a rota recebe um código de item no caminho. `itemCode` e
// `item_code` são sempre item; `{code}` só é item debaixo de /api/items/ — nas
// demais rotas ele é o código de cliente, fornecedor, pedido e por aí vai.
func referenciaItem(rota string) bool {
	if reParam.MatchString(rota) {
		return true
	}
	return strings.HasPrefix(rota, "/api/items/") && strings.Contains(rota, "{code}")
}

// rotaTraduzida troca os parâmetros por um valor qualquer e pergunta ao mesmo
// código que o middleware usa se aquele caminho é traduzido.
func rotaTraduzida(rota string) bool {
	concreta := regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(rota, "X")
	partes := strings.Split(strings.Trim(concreta, "/"), "/")
	idx := itemPathSegmentIndex(partes)
	if idx < 0 {
		return false
	}
	// O segmento traduzido precisa ser justamente o que a rota declara como
	// parâmetro do item, não outro qualquer.
	posicoes := map[int]bool{}
	for i, parte := range strings.Split(strings.Trim(rota, "/"), "/") {
		if reParam.MatchString(parte) || (parte == "{code}" && strings.HasPrefix(rota, "/api/items/")) {
			posicoes[i] = true
		}
	}
	return posicoes[idx]
}

func TestTrocaSegmentoNaoPegaPedacoDeOutroSegmento(t *testing.T) {
	if got := trocaSegmento("movements/item/10", "10", "77"); got != "movements/item/77" {
		t.Fatalf("esperava movements/item/77, veio %s", got)
	}
	// "10" não pode casar dentro de "100".
	if got := trocaSegmento("movements/item/100", "10", "77"); got != "movements/item/100" {
		t.Fatalf("trocou pedaço de outro segmento: %s", got)
	}
}

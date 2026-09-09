package request

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestWhereNaoReusaParametroDoSet trava a falha que deixou a alteração de
// máquina e de tipo de máquina sem efeito: o WHERE citava `$6`, que era um
// parâmetro da própria cláusula SET. O UPDATE comparava a chave com o valor
// errado e nunca encontrava a linha.
//
// A regra: em UPDATE escrito com parâmetros posicionais, o WHERE não pode usar
// um `$n` já consumido pelo SET. Consultas com `sqlc.arg(...)` são nomeadas e
// não sofrem do problema.
func TestWhereNaoReusaParametroDoSet(t *testing.T) {
	raiz := filepath.Join("..", "..", "..", "infrastructure", "database", "queries")
	arquivos, err := filepath.Glob(filepath.Join(raiz, "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if len(arquivos) == 0 {
		t.Skip("nenhuma query encontrada")
	}

	consulta := regexp.MustCompile(`(?is)--\s*name:\s*(\w+)[^\n]*\n(.*?)(?:\n--\s*name:|\z)`)
	posicional := regexp.MustCompile(`\$(\d+)`)

	var problemas []string
	for _, caminho := range arquivos {
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range consulta.FindAllStringSubmatch(string(conteudo), -1) {
			nome, corpo := m[1], m[2]
			maiusculo := strings.ToUpper(corpo)
			if !strings.Contains(maiusculo, "UPDATE ") || !strings.Contains(maiusculo, "SET") {
				continue
			}
			iSet := strings.Index(maiusculo, "SET")
			iWhere := strings.Index(maiusculo, "WHERE")
			if iWhere <= iSet {
				continue
			}
			usadosNoSet := map[string]bool{}
			for _, p := range posicional.FindAllStringSubmatch(corpo[iSet:iWhere], -1) {
				usadosNoSet[p[1]] = true
			}
			for _, p := range posicional.FindAllStringSubmatch(corpo[iWhere:], -1) {
				if usadosNoSet[p[1]] {
					problemas = append(problemas,
						filepath.Base(caminho)+" · "+nome+": WHERE usa $"+p[1]+", que o SET já consumiu")
				}
			}
		}
	}

	if len(problemas) > 0 {
		t.Fatalf("UPDATE com parâmetro reaproveitado entre SET e WHERE:\n  %s",
			strings.Join(problemas, "\n  "))
	}
}

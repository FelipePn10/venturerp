package errorsuc_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Varredura estática: nenhuma mensagem que chega ao usuário pode estar em inglês.
//
// O requisito "mensagens em português e entendíveis" era verificado exercitando
// as rotas — o que só alcança o caminho feliz e as validações mais comuns. As
// específicas escapavam e só apareciam para o usuário, em produção, no pior
// momento. Aqui a regressão é barrada no CI: escrever
// `NewValidationError("item not found")` reprova o PR.
//
// O escopo é deliberadamente o que o usuário LÊ: erro de validação, conflito,
// não-encontrado e `jsonError`. O `fmt.Errorf` de embrulho interno
// (`"listing items: %w"`) fica de fora — ele vira "erro interno do servidor"
// antes de chegar à tela, e traduzi-lo só atrapalharia quem lê log.
func TestMensagensAoUsuarioEstaoEmPortugues(t *testing.T) {
	// Chamada de erro do usuário seguida do texto literal.
	chamada := regexp.MustCompile(`(?:NewValidationError|NewConflictError|NewNotFoundError|jsonError)\s*\([^"]{0,120}"([^"]{6,})"`)
	// Marcas fortes de português: acento ou palavra que não existe em inglês.
	portugues := regexp.MustCompile(`(?i)[áàâãéêíóôõúüç]|\b(nao|informe|cadastre|selecione|obrigatori|invalid[oa]|encontrad[oa]|empresa|pedido|ordem|nenhum|antes|já|pra|para o|do item)\b`)
	// Marcas fortes de inglês: palavras funcionais que não são palavras em português.
	ingles := regexp.MustCompile(`(?i)\b(the|not found|must be|is required|cannot|failed to|missing|already exists|unknown|expected|not allowed|does not|should be|invalid request|no rows)\b`)

	raiz := localizarRaiz(t)
	var suspeitas []string
	err := filepath.Walk(filepath.Join(raiz, "internal"), func(caminho string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(caminho, ".go") || strings.HasSuffix(caminho, "_test.go") {
			return err
		}
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			return err
		}
		for _, achado := range chamada.FindAllStringSubmatch(string(conteudo), -1) {
			msg := achado[1]
			// `%s`/`%w` sozinhos não são texto; e o que já tem marca de
			// português não interessa mesmo que carregue uma palavra inglesa
			// (nome de campo, sigla, enum citado na mensagem).
			if portugues.MatchString(msg) || !ingles.MatchString(msg) {
				continue
			}
			rel, _ := filepath.Rel(raiz, caminho)
			suspeitas = append(suspeitas, rel+" → "+msg)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(suspeitas) > 0 {
		t.Fatalf("mensagem ao usuário em inglês (%d):\n  %s\n\nTraduza para português, dizendo o que fazer — quem lê está no meio de um lançamento, não depurando.",
			len(suspeitas), strings.Join(suspeitas, "\n  "))
	}
}

// localizarRaiz sobe até achar o go.mod: o teste roda a partir do diretório do
// pacote, mas varre o repositório inteiro.
func localizarRaiz(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("go.mod não encontrado a partir do diretório do teste")
	return ""
}

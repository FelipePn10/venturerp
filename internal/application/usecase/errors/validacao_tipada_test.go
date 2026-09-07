package errorsuc_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Mensagem de validação escrita para o usuário. Quando devolvida como
// `fmt.Errorf` solta, o mapeador de erros não a reconhece e responde 500
// "erro interno do servidor" — jogando fora justamente a explicação que o
// usuário precisava ler. O caso que motivou a varredura foi o Gantt do APS:
// "ano 1 inválido" virava erro de servidor.
var validacao = regexp.MustCompile(`(?i)(inválid|obrigatóri|informe |deve ser|deve estar|não pode|precisa ser|já existe|não é aceito|selecione |escolha )`)

func TestValidacoesDeUsuarioSaoErrosTipados(t *testing.T) {
	raiz := filepath.Join("..", "..", "..", "application", "usecase")
	var soltas []string

	err := filepath.Walk(raiz, func(caminho string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(caminho, ".go") || strings.HasSuffix(caminho, "_test.go") {
			return err
		}
		fset := token.NewFileSet()
		arquivo, parseErr := parser.ParseFile(fset, caminho, nil, 0)
		if parseErr != nil {
			return nil // arquivo que não compila é problema de outro teste
		}
		ast.Inspect(arquivo, func(n ast.Node) bool {
			chamada, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := chamada.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Errorf" {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "fmt" {
				return true
			}
			if len(chamada.Args) == 0 {
				return true
			}
			literal, ok := chamada.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			texto := literal.Value
			// `%w` embrulha uma falha interna: não é validação de entrada.
			if strings.Contains(texto, "%w") || !validacao.MatchString(texto) {
				return true
			}
			soltas = append(soltas, caminho+": "+texto)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("varrendo os casos de uso: %v", err)
	}

	if len(soltas) > 0 {
		t.Fatalf("%d validação(ões) devolvidas como fmt.Errorf — use errorsuc.NewValidationError "+
			"para o usuário receber 422 com a mensagem, e não 500 genérico:\n  %s",
			len(soltas), strings.Join(soltas, "\n  "))
	}
}

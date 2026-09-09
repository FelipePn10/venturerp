package request

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSituacaoEmAtualizacaoEhOpcional trava a regra que fez o fornecedor sumir
// da listagem depois de qualquer edição: um `bool` de situação declarado num
// DTO de atualização chega como `false` quando o corpo não menciona o campo, e o
// caso de uso desativava o cadastro sem ninguém pedir.
//
// Em DTO de atualização, a situação é sempre ponteiro: nil = manter como está.
func TestSituacaoEmAtualizacaoEhOpcional(t *testing.T) {
	situacoes := map[string]bool{"IsActive": true, "Active": true, "Enabled": true}

	arquivos, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	var problemas []string
	for _, caminho := range arquivos {
		if strings.HasSuffix(caminho, "_test.go") {
			continue
		}
		fonte, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatal(err)
		}
		arquivo, err := parser.ParseFile(token.NewFileSet(), caminho, fonte, 0)
		if err != nil {
			t.Fatalf("%s: %v", caminho, err)
		}
		ast.Inspect(arquivo, func(n ast.Node) bool {
			especificacao, ok := n.(*ast.TypeSpec)
			if !ok || !strings.HasPrefix(especificacao.Name.Name, "Update") {
				return true
			}
			estrutura, ok := especificacao.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, campo := range estrutura.Fields.List {
				identificador, ok := campo.Type.(*ast.Ident)
				if !ok || identificador.Name != "bool" {
					continue
				}
				for _, nome := range campo.Names {
					if situacoes[nome.Name] {
						problemas = append(problemas, caminho+" · "+especificacao.Name.Name+"."+nome.Name)
					}
				}
			}
			return true
		})
	}

	if len(problemas) > 0 {
		t.Fatalf("situação como bool em DTO de atualização (use *bool, nil = manter):\n  %s",
			strings.Join(problemas, "\n  "))
	}
}

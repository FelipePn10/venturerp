// Package guardas concentra os testes que protegem invariantes que o
// compilador não vê.
package guardas

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestParamsComEmpresaSaoPreenchidos reprova o defeito mais silencioso que
// apareceu na auditoria de isolamento: um struct de parâmetros do sqlc que tem
// EnterpriseID/EnterpriseCode e o chamador deixa o campo em zero.
//
// Não há erro de compilação (campo ausente vira o valor zero), não há erro em
// tempo de execução (a consulta roda), e o efeito é uma escrita que não grava
// nada ou uma leitura que não acha nada — sempre em silêncio. Foi assim que a
// alteração de endereço de fornecedor parou de gravar sem avisar ninguém.
func TestParamsComEmpresaSaoPreenchidos(t *testing.T) {
	raiz, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	// 1. Quais structs gerados têm campo de empresa.
	comEmpresa := map[string]bool{}
	sqlcDir := filepath.Join(raiz, "internal", "infrastructure", "database", "sqlc")
	arquivos, err := filepath.Glob(filepath.Join(sqlcDir, "*.go"))
	if err != nil || len(arquivos) == 0 {
		t.Fatalf("não encontrei os arquivos gerados em %s: %v", sqlcDir, err)
	}
	declara := regexp.MustCompile(`(?s)type (\w+Params) struct \{(.*?)\n\}`)
	for _, f := range arquivos {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range declara.FindAllStringSubmatch(string(b), -1) {
			if strings.Contains(m[2], "EnterpriseID") || strings.Contains(m[2], "EnterpriseCode") {
				comEmpresa[m[1]] = true
			}
		}
	}
	if len(comEmpresa) == 0 {
		t.Fatal("nenhum struct de parâmetros com empresa encontrado — o teste perdeu o alvo")
	}

	// 2. Todo literal desses structs precisa informar a empresa.
	usa := regexp.MustCompile(`(?s)sqlc\.(\w+Params)\{(.*?)\n(\s*)\}`)
	var faltando []string
	err = filepath.Walk(filepath.Join(raiz, "internal"), func(caminho string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(caminho, ".go") || strings.HasSuffix(caminho, "_test.go") {
			return err
		}
		if strings.Contains(caminho, filepath.Join("database", "sqlc")) {
			return nil
		}
		b, err := os.ReadFile(caminho)
		if err != nil {
			return err
		}
		texto := string(b)
		// Alguns lugares montam o struct num helper e atribuem a empresa no
		// chamador (`params.EnterpriseID = ...`). Isso é correto, então a
		// atribuição no mesmo arquivo vale como preenchimento. É uma folga
		// deliberada: prefiro deixar passar esse caso a acusá-lo toda vez e
		// ensinar o time a ignorar o teste.
		atribuiNoArquivo := strings.Contains(texto, ".EnterpriseID =") ||
			strings.Contains(texto, ".EnterpriseCode =")
		for _, m := range usa.FindAllStringSubmatch(texto, -1) {
			if !comEmpresa[m[1]] {
				continue
			}
			if strings.Contains(m[2], "EnterpriseID") || strings.Contains(m[2], "EnterpriseCode") {
				continue
			}
			if atribuiNoArquivo {
				continue
			}
			rel, _ := filepath.Rel(raiz, caminho)
			faltando = append(faltando, rel+" → "+m[1])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range faltando {
		t.Errorf("parâmetros sem a empresa (grava/lê em silêncio na empresa 0): %s", f)
	}
}

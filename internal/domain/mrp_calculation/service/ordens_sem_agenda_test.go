package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

func TestOrdensSemAgendaResumeAsFalhas(t *testing.T) {
	if err := ordensSemAgenda(nil); err != nil {
		t.Fatalf("sem falha não há erro: %v", err)
	}
	err := ordensSemAgenda([]string{"item 10002: máquina 1006: não há janela suficiente"})
	if err == nil || !falhaDeCadastroOuCapacidade(err) {
		t.Fatalf("esperava erro de validação, veio %v", err)
	}
	for _, trecho := range []string{"1 ordem(ns)", "as demais foram programadas", "item 10002"} {
		if !strings.Contains(err.Error(), trecho) {
			t.Fatalf("%q não contém %q", err.Error(), trecho)
		}
	}
	// Muitas falhas: mostra as primeiras e conta o resto.
	muitas := make([]string, 8)
	for i := range muitas {
		muitas[i] = fmt.Sprintf("item %d: sem janela", 100+i)
	}
	texto := ordensSemAgenda(muitas).Error()
	if !strings.Contains(texto, "8 ordem(ns)") || !strings.Contains(texto, "e mais 3") || strings.Contains(texto, "item 107") {
		t.Fatalf("resumo = %q", texto)
	}
}

// Só o erro da própria ordem (capacidade, cadastro) deixa o planejamento
// seguir; falha de banco continua interrompendo e desfazendo tudo.
func TestFalhaDeCadastroOuCapacidade(t *testing.T) {
	capacidade := fmt.Errorf("máquina 1006: %w", domainerr.NewValidation("não há janela suficiente"))
	if !falhaDeCadastroOuCapacidade(capacidade) {
		t.Fatal("falha de capacidade embrulhada não foi reconhecida")
	}
	if falhaDeCadastroOuCapacidade(errors.New("connection reset")) {
		t.Fatal("erro de infraestrutura tratado como falha da ordem")
	}
	if falhaDeCadastroOuCapacidade(nil) {
		t.Fatal("nil não é falha")
	}
}

func TestNenhumaOrdemProgramadaMantemAProgramacaoAnterior(t *testing.T) {
	err := nenhumaOrdemProgramada([]string{"item 7: máquina 3: não há janela suficiente", "item 8: idem"})
	if !falhaDeCadastroOuCapacidade(err) || !strings.Contains(err.Error(), "programação anterior foi mantida") || !strings.Contains(err.Error(), "item 7") {
		t.Fatalf("erro = %v", err)
	}
}

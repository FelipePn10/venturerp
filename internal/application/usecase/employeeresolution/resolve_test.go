package employeeresolution

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
)

// A base da Tecnofer: códigos 1..28 e ids 31..58. A tela manda o CÓDIGO; as
// tabelas de produção exigem o ID. Sem tradução, apontar produção respondia
// "um dos vínculos informados não existe na empresa autenticada" para qualquer
// operador — o chão de fábrica não conseguia apontar nada.
type cadastroDeFuncionarios struct{ erroNaLista error }

var anaRita = &entity.Employee{ID: 31, Code: 1, Name: "ANA RITA DE JESUS SANTOS RAMOS"}
var cleber = &entity.Employee{ID: 34, Code: 4, Name: "CLEBER ALVES SANTANA"}

func (c cadastroDeFuncionarios) GetByCode(_ context.Context, code int64) (*entity.Employee, error) {
	switch code {
	case 1:
		return anaRita, nil
	case 4:
		return cleber, nil
	}
	return nil, errors.New("não encontrado")
}

func (c cadastroDeFuncionarios) List(_ context.Context) ([]*entity.Employee, error) {
	if c.erroNaLista != nil {
		return nil, c.erroNaLista
	}
	return []*entity.Employee{anaRita, cleber}, nil
}

func TestCodigoDoFuncionarioViraChaveInterna(t *testing.T) {
	id, err := ResolveID(context.Background(), cadastroDeFuncionarios{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if id != 31 {
		t.Fatalf("código 1 deveria virar o id 31, veio %d", id)
	}
}

// Cliente antigo (ou integração) que já manda a chave interna continua valendo.
func TestChaveInternaContinuaAceita(t *testing.T) {
	id, err := ResolveID(context.Background(), cadastroDeFuncionarios{}, 34)
	if err != nil {
		t.Fatal(err)
	}
	if id != 34 {
		t.Fatalf("id 34 deveria ser aceito, veio %d", id)
	}
}

// Funcionário que não existe precisa dizer isso, com o número tentado — e não
// estourar uma violação de chave estrangeira do banco na cara do usuário.
func TestFuncionarioInexistenteRecusaComMensagemClara(t *testing.T) {
	_, err := ResolveID(context.Background(), cadastroDeFuncionarios{}, 999)
	if err == nil {
		t.Fatal("esperava recusa")
	}
	if !contains(err.Error(), "999") || !contains(err.Error(), "cadastro de funcionários") {
		t.Fatalf("mensagem pouco útil: %s", err.Error())
	}
}

// Sem repositório configurado, o valor passa como está: é o comportamento
// anterior e não pode parar fluxo que já roda.
func TestSemCadastroConfiguradoNaoQuebra(t *testing.T) {
	id, err := ResolveID(context.Background(), nil, 77)
	if err != nil || id != 77 {
		t.Fatalf("id=%d err=%v", id, err)
	}
}

func TestOpcionalNuloSegueNulo(t *testing.T) {
	id, err := ResolveOptionalID(context.Background(), cadastroDeFuncionarios{}, nil)
	if err != nil || id != nil {
		t.Fatalf("id=%v err=%v", id, err)
	}
	zero := int64(0)
	id, err = ResolveOptionalID(context.Background(), cadastroDeFuncionarios{}, &zero)
	if err != nil || id != nil {
		t.Fatalf("zero deveria virar nulo: id=%v err=%v", id, err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

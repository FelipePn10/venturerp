package unit_of_measure_uc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
)

type repoExclusao struct {
	repository.UnitRepository
	err error
}

func (r repoExclusao) DeleteStandardConversion(context.Context, int64) error { return r.err }

type recalculoExclusao struct{ motivo string }

func (r *recalculoExclusao) RecalcularDoComponente(context.Context, int64, string) (int, []string, error) {
	return 0, nil, nil
}
func (r *recalculoExclusao) RecalcularDaEmpresa(_ context.Context, motivo string) (int, []string, error) {
	r.motivo = motivo
	return 0, []string{"estrutura do item RN-1: o componente TUBO é estocado em M e a linha está em MM, sem conversão entre as duas — cadastre a conversão ou corrija a linha"}, nil
}

// Excluir a conversão padrão deixa sem conversão a linha que dependia dela: o
// aviso volta para a tela (antes era descartado) e o motivo do histórico diz
// "excluída", não "alterada".
func TestExcluirConversaoPadraoDevolveAsEstruturasAfetadas(t *testing.T) {
	rec := &recalculoExclusao{}
	uc := &UseCase{repo: repoExclusao{}, Estruturas: rec}
	avisos, err := uc.DeleteStandardConversion(context.Background(), 9)
	if err != nil || len(avisos) != 1 || !strings.Contains(avisos[0], "RN-1") {
		t.Fatalf("err=%v avisos=%v", err, avisos)
	}
	if rec.motivo != "conversão padrão excluída" {
		t.Fatalf("motivo no histórico: %q", rec.motivo)
	}

	rec2 := &recalculoExclusao{}
	if _, err := (&UseCase{repo: repoExclusao{err: errors.New("não encontrada")}, Estruturas: rec2}).DeleteStandardConversion(context.Background(), 9); err == nil || rec2.motivo != "" {
		t.Fatalf("exclusão falhou e recalculou mesmo assim: err=%v", err)
	}
}

type recalculoLongo struct{}

func (recalculoLongo) RecalcularDoComponente(context.Context, int64, string) (int, []string, error) {
	return 0, nil, nil
}
func (recalculoLongo) RecalcularDaEmpresa(context.Context, string) (int, []string, error) {
	l := make([]string, 30)
	for i := range l {
		l[i] = "linha"
	}
	return 2, l, nil
}

// 30 linhas pendentes na empresa não viram 30 mensagens na tela.
func TestAvisosDaConversaoPadraoLimitados(t *testing.T) {
	av, err := (&UseCase{repo: repoExclusao{}, Estruturas: recalculoLongo{}}).DeleteStandardConversion(context.Background(), 1)
	if err != nil || len(av) != 7 || !strings.Contains(av[0], "2 linha(s)") || !strings.Contains(av[6], "e mais 25") {
		t.Fatalf("err=%v avisos=%v", err, av)
	}
}

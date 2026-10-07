package fiscal_uc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
)

type spedPeriodoFake struct {
	d           sped.DadosEFD
	inicio, fim time.Time
	inv         []sped.ItemInventario
	conta       string
}

func (f *spedPeriodoFake) InventarioEFD(context.Context, time.Time) ([]sped.ItemInventario, string, error) {
	return f.inv, f.conta, nil
}

func (f *spedPeriodoFake) CarregarPeriodoEFD(_ context.Context, inicio, fim time.Time) (*sped.DadosEFD, error) {
	f.inicio, f.fim = inicio, fim
	d := f.d
	return &d, nil
}

type spedConfigFake struct{ dia int }

func (c spedConfigFake) GetFiscalConfig(context.Context) (*fiscalEntity.FiscalConfig, error) {
	return &fiscalEntity.FiscalConfig{VencimentoIcmsDia: c.dia}, nil
}

func empresaSped() sped.EFDEmpresa {
	return sped.EFDEmpresa{CNPJ: "52454668000102", Nome: "TECNOFER", UF: "PR", IE: "9012345678", CodigoMunicipio: "4106902"}
}

func TestSpedAutomatico_Validacoes(t *testing.T) {
	uc := &SpedAutomaticoUseCase{Periodo: &spedPeriodoFake{d: sped.DadosEFD{Empresa: empresaSped()}}}
	ok := SpedAutomaticoRequest{Ano: 2026, Mes: 2, ContabilistaNome: "C", ContabilistaCPF: "11122233344"}
	casos := map[string]func(r *SpedAutomaticoRequest){
		"mês":        func(r *SpedAutomaticoRequest) { r.Mes = 13 },
		"finalidade": func(r *SpedAutomaticoRequest) { r.Finalidade = "2" },
		"perfil":     func(r *SpedAutomaticoRequest) { r.Perfil = "D" },
		"atividade":  func(r *SpedAutomaticoRequest) { r.IndAtividade = "5" },
		"cpf":        func(r *SpedAutomaticoRequest) { r.ContabilistaCPF = "123" },
		"saldo":      func(r *SpedAutomaticoRequest) { r.SaldoCredorAnteriorICMS = decimal.NewFromInt(-1) },
	}
	for nome, mudar := range casos {
		r := ok
		mudar(&r)
		_, err := uc.Gerar(context.Background(), r)
		var ve *errorsuc.ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: erro = %v, want ValidationError", nome, err)
		}
	}
}

func TestSpedAutomatico_PeriodoEVencimento(t *testing.T) {
	fake := &spedPeriodoFake{d: sped.DadosEFD{Empresa: empresaSped()}}
	uc := &SpedAutomaticoUseCase{Periodo: fake, Config: spedConfigFake{dia: 31}}
	res, err := uc.Gerar(context.Background(), SpedAutomaticoRequest{Ano: 2026, Mes: 1, ContabilistaNome: "C",
		ContabilistaCPF: "111.222.333-44", SaldoCredorAnteriorICMS: decimal.NewFromInt(5)})
	if err != nil {
		t.Fatal(err)
	}
	if fake.inicio.Format("2006-01-02") != "2026-01-01" || fake.fim.Format("2006-01-02") != "2026-01-31" {
		t.Errorf("período = %s..%s", fake.inicio, fake.fim)
	}
	if !strings.HasPrefix(res.Arquivo, "|0000|020|0|01012026|31012026|") || res.NomeArquivo != "EFD_ICMS_IPI_52454668000102_202601.txt" {
		t.Errorf("arquivo/nome: %q %s", res.Arquivo[:40], res.NomeArquivo)
	}
	// Sem movimento: o saldo anterior é transportado inteiro.
	if !res.Resumo.ICMSSaldoCredor.Equal(decimal.NewFromInt(5)) {
		t.Errorf("saldo credor = %s", res.Resumo.ICMSSaldoCredor)
	}
	// Dia 31 em fevereiro vira o último dia do mês.
	if got := uc.vencimentoPadrao(context.Background(), fake.fim); got.Format("2006-01-02") != "2026-02-28" {
		t.Errorf("vencimento = %s", got)
	}
	// Sem configuração: dia 10 do mês seguinte (dezembro → janeiro).
	uc.Config = nil
	if got := uc.vencimentoPadrao(context.Background(), time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)); got.Format("2006-01-02") != "2027-01-10" {
		t.Errorf("vencimento = %s", got)
	}
}

func TestSpedAutomatico_ConfiguracaoIncompleta(t *testing.T) {
	e := empresaSped()
	e.IE, e.CodigoMunicipio = "", "41"
	uc := &SpedAutomaticoUseCase{Periodo: &spedPeriodoFake{d: sped.DadosEFD{Empresa: e}}}
	_, err := uc.Gerar(context.Background(), SpedAutomaticoRequest{Ano: 2026, Mes: 1, ContabilistaNome: "C", ContabilistaCPF: "11122233344"})
	if err == nil || !strings.Contains(err.Error(), "inscrição estadual") || !strings.Contains(err.Error(), "município") {
		t.Errorf("erro = %v", err)
	}
}

func TestAvisosEFD(t *testing.T) {
	p := sped.EFDParams{
		Participantes: []sped.EFDParticipante{{CodPart: "1", Nome: "X"}},
		Itens:         []sped.EFDItem{{CodItem: "A", TipoItem: "01"}, {CodItem: "B", TipoItem: "07"}},
		Conhecimentos: []sped.EFDConhecimento{{NumDoc: "9"}},
		ApuracaoICMS:  &sped.EFDApuracaoICMS{Obrigacoes: []sped.EFDObrigacao{{VlOr: 1}}},
	}
	if a := avisosEFD(p, ""); len(a) != 4 {
		t.Errorf("avisos = %v", a)
	}
}

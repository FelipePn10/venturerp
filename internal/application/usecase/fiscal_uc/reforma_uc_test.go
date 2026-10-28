package fiscal_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
)

type authReforma struct{ ports.AuthService }

func (authReforma) CanManageFiscalConfig(context.Context) bool { return true }
func (authReforma) CanListFiscalExits(context.Context) bool    { return true }
func (authReforma) CanCreateFiscalExit(context.Context) bool   { return true }
func (authReforma) UserID(context.Context) (uuid.UUID, error)  { return uuid.New(), nil }

type reformaGravacao struct {
	reformaFake
	cfop, classTrib string
	ano             int
	cadastradas     []reforma.AliquotasCadastradas
}

func (f *reformaGravacao) SalvarCFOP(_ context.Context, cfop, c string, _ *string, _ uuid.UUID) error {
	f.cfop, f.classTrib = cfop, c
	return nil
}
func (f *reformaGravacao) SalvarAliquotas(_ context.Context, ano int, _ reforma.Aliquotas, _ string, _ uuid.UUID) error {
	f.ano = ano
	return nil
}
func (f *reformaGravacao) ListarAliquotas(context.Context) ([]reforma.AliquotasCadastradas, error) {
	return f.cadastradas, nil
}

func TestManutencaoDaReformaValida(t *testing.T) {
	repo := &reformaGravacao{reformaFake: reformaFake{tabela: tabelaTeste()}}
	uc := &ReformaTributariaUseCase{Repo: repo, Auth: authReforma{}}
	ctx := context.Background()

	if err := uc.SalvarCFOP(ctx, "5.910", SalvarCFOPClassTribDTO{ClassTrib: "410001"}); err != nil || repo.cfop != "5910" {
		t.Fatalf("CFOP com ponto deveria ser aceito como 5910: %v %q", err, repo.cfop)
	}
	for _, caso := range []struct{ cfop, classTrib, erro string }{
		{"1101", "000001", "saída"},
		{"510", "000001", "4 dígitos"},
		{"5101", "41000", "6 dígitos"},
		{"5101", "999999", "tabela oficial"},
	} {
		err := uc.SalvarCFOP(ctx, caso.cfop, SalvarCFOPClassTribDTO{ClassTrib: caso.classTrib})
		if err == nil || !strings.Contains(err.Error(), caso.erro) {
			t.Fatalf("%+v: %v", caso, err)
		}
	}
	d := decimal.RequireFromString
	if err := uc.SalvarAliquotas(ctx, 2027, SalvarAliquotasIBSCBSDTO{IBSUF: d("0.05"), IBSMun: d("0.05"), CBS: d("8.7")}); err == nil {
		t.Fatal("sem fundamento deveria recusar")
	}
	if err := uc.SalvarAliquotas(ctx, 2027, SalvarAliquotasIBSCBSDTO{IBSUF: d("0.05"), IBSMun: d("0.05"), CBS: d("100"), Fundamento: "Resolução"}); err == nil {
		t.Fatal("CBS de 100% deveria recusar")
	}
	if err := uc.SalvarAliquotas(ctx, 2027, SalvarAliquotasIBSCBSDTO{IBSUF: d("0.05"), IBSMun: d("0.05"), CBS: d("8.7"), Fundamento: "Resolução do Senado"}); err != nil || repo.ano != 2027 {
		t.Fatalf("cadastro válido: %v", err)
	}
}

func TestListaDeAliquotasMostraLeiEPendencias(t *testing.T) {
	uc := &ReformaTributariaUseCase{Repo: &reformaGravacao{}, Auth: authReforma{}}
	lista, err := uc.ListarAliquotas(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 8 || lista[0].Ano != 2026 || lista[0].Fonte != reforma.FonteLei || *lista[0].CBS != 0.9 {
		t.Fatalf("2026 pela lei: %+v", lista[0])
	}
	if lista[1].Fonte != "PENDENTE" || lista[1].CBS != nil || *lista[1].IBSUF != 0.05 || !strings.Contains(lista[1].Pendencia, "art. 347") {
		t.Fatalf("2027 sem CBS fica pendente com o IBS da lei: %+v", lista[1])
	}
	if lista[3].IBSUF != nil || lista[3].Fonte != "PENDENTE" {
		t.Fatalf("2029 sem lei nem cadastro: %+v", lista[3])
	}
}

//go:build integration

package enterprise_uc_test

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/enterprise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// A identificação da empresa vive num lugar só (VEMP0100); a configuração
// fiscal (VFIS0100) grava os parâmetros sem tocar nela. O token Focus de
// produção não muda pela tela.
func TestIdentificacaoUnicaETokenDeProducao(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	code := int(1_800_000_000 + testutil.UniqueCode()%100_000_000)
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_configs WHERE enterprise_id IN (SELECT id FROM enterprise WHERE code=$1)`, code)
		_, _ = pool.Exec(bg, `DELETE FROM user_enterprises WHERE enterprise_id IN (SELECT id FROM enterprise WHERE code=$1)`, code)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE code=$1`, code)
	})
	repo := enterprise.NewRepositoryEnterpriseSQLC(sqlc.New(pool)).ComPool(pool)
	e, err := mapper.ToEnterpriseEntity(request.CreateEnterpriseDTO{Code: code, CreatedBy: actor, CNPJ: "11222333000181",
		RazaoSocial: "IDENTIDADE TESTE LTDA", NomeFantasia: "IDENTIDADE", RegimeTributario: "3", UF: "PR", CodigoMunicipio: "4106902"})
	if err != nil {
		t.Fatal(err)
	}
	criada, err := repo.Create(bg, e)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &security.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: int64(criada.ID), EnterpriseCode: int64(code)})
	az := &auth.AuthService{}

	// 1. VEMP0100 altera a identificação: nome da empresa e fiscal_configs juntos.
	ident := &enterprise_uc.IdentificacaoUseCase{Repo: repo, Auth: az}
	if _, err := ident.Atualizar(ctx, request.IdentificacaoEmpresaDTO{CNPJ: "11222333000181", RazaoSocial: "IDENTIDADE TESTE LTDA",
		RegimeTributario: "3", UF: "PR", CodigoMunicipio: "3550308"}); err == nil || !strings.Contains(err.Error(), "não é de município do PR") {
		t.Fatalf("IBGE de outra UF deveria ser recusado: %v", err)
	}
	out, err := ident.Atualizar(ctx, request.IdentificacaoEmpresaDTO{CNPJ: "11.222.333/0001-81", RazaoSocial: "IDENTIDADE TESTE LTDA",
		NomeFantasia: "IDENTIDADE NOVA", InscricaoEstadual: "1234567890", RegimeTributario: "1", UF: "pr", Municipio: "Curitiba",
		CodigoMunicipio: "4106902", CEP: "80000-000", Logradouro: "Rua A", Numero: "10", Complemento: "Sala 2", Bairro: "Centro",
		Telefone: "4133330000", Email: "fiscal@identidade.test"})
	if err != nil {
		t.Fatal(err)
	}
	if out.RegimeTributario != "1" || out.UF != "PR" || out.Complemento != "Sala 2" || out.CEP != "80000000" {
		t.Fatalf("identificação gravada: %+v", out)
	}
	var nome string
	_ = pool.QueryRow(bg, `SELECT name FROM enterprise WHERE id=$1`, criada.ID).Scan(&nome)
	if nome != "IDENTIDADE NOVA" {
		t.Fatalf("o nome da empresa acompanha o nome fantasia: %q", nome)
	}

	// 2. VFIS0100 grava os parâmetros sem mandar a identificação: ela fica.
	fiscalRepo := fiscal.NewFiscalRepositoryPG(pool)
	cfgUC := &fiscal_uc.UpdateFiscalConfigUseCase{Repo: fiscalRepo, Auth: az}
	tokH := "token-de-homologacao"
	salvo, err := cfgUC.Execute(ctx, request.UpdateFiscalConfigDTO{IcmsInternoAliquota: 0.195, FocusNfeAmbiente: "homologacao",
		FocusNfeToken: &tokH, VencimentoIcmsDia: 10, VencimentoIPIDia: 15, VencimentoPisCofinsDia: 25})
	if err != nil {
		t.Fatal(err)
	}
	if salvo.CnpjEmpresa != "11222333000181" || salvo.RegimeTributario != "1" || salvo.Municipio != "Curitiba" || salvo.NomeFantasia != "IDENTIDADE NOVA" || !salvo.FocusNfeConfigured {
		t.Fatalf("a identificação deveria ficar como estava: %+v", salvo)
	}

	// 3. Produção só pela Venture: a tela não entra, não troca e não sai.
	if _, err := cfgUC.Execute(ctx, request.UpdateFiscalConfigDTO{FocusNfeAmbiente: "producao", VencimentoIcmsDia: 10, VencimentoIPIDia: 15, VencimentoPisCofinsDia: 25}); err == nil {
		t.Fatal("a tela não pode passar para produção")
	}
	testutil.Exec(t, pool, `UPDATE fiscal_configs SET focus_nfe_ambiente='producao', focus_nfe_token='token-de-producao' WHERE enterprise_id=$1`, criada.ID)
	outro := "outro-token"
	if _, err := cfgUC.Execute(ctx, request.UpdateFiscalConfigDTO{FocusNfeAmbiente: "producao", FocusNfeToken: &outro, VencimentoIcmsDia: 10, VencimentoIPIDia: 15, VencimentoPisCofinsDia: 25}); err == nil {
		t.Fatal("a tela não pode trocar o token de produção")
	}
	if _, err := cfgUC.Execute(ctx, request.UpdateFiscalConfigDTO{FocusNfeAmbiente: "homologacao", VencimentoIcmsDia: 10, VencimentoIPIDia: 15, VencimentoPisCofinsDia: 25}); err == nil {
		t.Fatal("a tela não pode sair da produção")
	}
	salvo, err = cfgUC.Execute(ctx, request.UpdateFiscalConfigDTO{IcmsInternoAliquota: 0.12, FocusNfeAmbiente: "producao", VencimentoIcmsDia: 12, VencimentoIPIDia: 15, VencimentoPisCofinsDia: 25})
	if err != nil {
		t.Fatalf("em produção os outros parâmetros continuam editáveis: %v", err)
	}
	var token string
	_ = pool.QueryRow(bg, `SELECT focus_nfe_token FROM fiscal_configs WHERE enterprise_id=$1`, criada.ID).Scan(&token)
	if token != "token-de-producao" || !salvo.FocusProducaoTravado || salvo.VencimentoIcmsDia != 12 {
		t.Fatalf("token de produção intocado e travado: %q %+v", token, salvo)
	}
}

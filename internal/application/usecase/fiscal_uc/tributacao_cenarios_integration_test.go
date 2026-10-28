//go:build integration

package fiscal_uc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// empresaTributavel cria uma empresa com configuração fiscal no PR (ICMS
// interno 19,5% e diferimento 38,46%, os padrões da tabela) no regime pedido.
func empresaTributavel(t *testing.T, nome, regime string, creditoSN float64, exigir bool) (context.Context, int64) {
	t.Helper()
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	code := 1_965_000_000 + testutil.UniqueCode()%10_000_000
	var emp int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,$2) RETURNING id`, code, nome).Scan(&emp); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,regime_tributario,aliquota_credito_simples,exigir_regra_icms_ipi,updated_by,enterprise_id,
		logradouro,numero,bairro,municipio,cep,focus_nfe_token)
		VALUES('11222333000181',$1,'PR',$2,$3,$4,$5,$6,'Rua A','1','Centro','Mandaguaçu','86975000','token-teste')`, nome, regime, creditoSN, exigir, ator, emp)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`, emp)
		testutil.Exec(t, pool, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM icms_ipi_tax_params WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM customers WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM items WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM enterprise WHERE id=$1`, emp)
	})
	return context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: emp, EnterpriseCode: code}), emp
}

type cenarioNota struct {
	uf, ie, finalidade string
	cliente            *int64
}

func perto(a, b float64) bool { return math.Abs(a-b) < 0.01 }

// A tributação por cenário de ponta a ponta: a nota é criada pelo caso de uso
// com os repositórios reais, a linha gravada é relida do banco, a prévia
// confere as pendências e o JSON enviado à Focus leva os grupos certos.
func TestIntegration_TributacaoPorCenario(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaTributavel(t, "Tecnofer teste", "3", 0, false)
	ator := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	item := u
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,1,$2,$3)`, item, ator, emp)
	clienteRevenda, clienteUso := u+1, u+2
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id,finalidade_compra) VALUES
		($1,'Revenda','44444444000191',$3,$4,'REVENDA'),($2,'Usa e consome','55555555000191',$3,$4,'USO_CONSUMO')`, clienteRevenda, clienteUso, ator, emp)

	repo := fiscalpg.NewFiscalRepositoryPG(pool)
	cadastro := fiscalpg.NewCadastroTributarioPG(pool)
	parametros := fiscalpg.NewParametroICMSIPIPG(pool)
	criar := &fiscal_uc.CreateFiscalExitUseCase{Repo: repo, Auth: &infraauth.AuthService{}, Parametros: parametros, Cadastro: cadastro}
	previa := &fiscal_uc.PreviaNFeUseCase{Repo: repo, Auth: &infraauth.AuthService{}}

	emitir := func(c cenarioNota) (*entity.FiscalExit, *entity.FiscalExitItem) {
		t.Helper()
		ncm, uf := "73269090", c.uf // NCM com IPI de 5% na tabela
		dto := request.CreateFiscalExitDTO{
			Serie: "1", DataEmissao: time.Now().Format("2006-01-02"), UFDestinatario: &uf, CustomerCode: c.cliente,
			CnpjDestinatario: strPtr("44444444000191"), RazaoSocialDestinatario: strPtr("Cliente"),
			Cfop: "5102", NaturezaOperacao: "Venda", ValorProdutos: 1000,
			Itens: []request.CreateFiscalExitItemDTO{{Sequence: 1, ItemCode: &item, Ncm: &ncm, Cfop: "5102", Quantity: 1, UnitPrice: 1000, TotalPrice: 1000, OrigemMercadoria: "0"}},
		}
		if c.ie != "" {
			dto.IEDestinatario = &c.ie
		}
		if c.finalidade != "" {
			dto.FinalidadeCompra = &c.finalidade
		}
		res, err := criar.Execute(ctx, dto)
		if err != nil {
			t.Fatal(err)
		}
		exit, err := repo.GetExitByID(ctx, res.ID)
		if err != nil {
			t.Fatal(err)
		}
		itens, err := repo.GetExitItems(ctx, res.ID)
		if err != nil || len(itens) != 1 {
			t.Fatalf("itens: %v %v", itens, err)
		}
		return exit, itens[0]
	}
	payload := func(id int64) map[string]any {
		t.Helper()
		out, err := previa.Execute(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(out.PayloadEnviado), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	testutil.Exec(t, pool, `UPDATE fiscal_configs SET cbenef_diferimento='PR830001' WHERE enterprise_id=$1`, emp)
	// 1. Tecnofer no PR para revendedor (finalidade do cadastro do cliente):
	// IPI fora da base, diferimento de 38,46% (CST 51), pDif gravado na linha.
	exit, it := emitir(cenarioNota{uf: "PR", ie: "9012345678", cliente: &clienteRevenda})
	if exit.FinalidadeCompra == nil || *exit.FinalidadeCompra != "REVENDA" {
		t.Fatalf("finalidade da nota: %v, quer REVENDA (do cliente)", exit.FinalidadeCompra)
	}
	if derefS(it.CstICMS) != "51" || !perto(it.BaseICMS, 1000) || !perto(it.ValorICMS, 195) || !perto(it.ValorICMSDiferido, 75) || !perto(it.ICMSDiferimentoPct, 38.46) {
		t.Fatalf("revenda PR: CST %s base %.2f ICMS %.2f diferido %.2f pDif %.2f", derefS(it.CstICMS), it.BaseICMS, it.ValorICMS, it.ValorICMSDiferido, it.ICMSDiferimentoPct)
	}
	if !perto(exit.ValorICMS, 120) {
		t.Fatalf("cabeçalho da nota com o ICMS da operação (%.2f), quer o destacado 120,00", exit.ValorICMS)
	}
	if derefS(it.CodigoBeneficioFiscal) != "PR830001" {
		t.Fatalf("cBenef do diferimento da configuração: %q, quer PR830001", derefS(it.CodigoBeneficioFiscal))
	}

	// 2. Contribuinte comprando para uso e consumo: consumidor final, IPI na
	// base, sem diferimento; o XML vai com consumidor_final = 1 e a IE.
	exit, it = emitir(cenarioNota{uf: "PR", ie: "9012345679", cliente: &clienteUso})
	if derefS(it.CstICMS) != "00" || !perto(it.BaseICMS, 1050) || !perto(it.ValorICMS, 204.75) || it.ValorICMSDiferido != 0 {
		t.Fatalf("uso e consumo PR: CST %s base %.2f ICMS %.2f diferido %.2f", derefS(it.CstICMS), it.BaseICMS, it.ValorICMS, it.ValorICMSDiferido)
	}
	if m := payload(exit.ID); m["consumidor_final"] != 1.0 || m["indicador_inscricao_estadual_destinatario"] != 1.0 {
		t.Fatalf("uso e consumo: consumidor_final %v indicador IE %v, quer 1 e 1", m["consumidor_final"], m["indicador_inscricao_estadual_destinatario"])
	}

	// 3. Não contribuinte em GO: 7% (Resolução SF 22/89) e o grupo do DIFAL
	// (interna de GO 19% − 7% sobre 1.050; GO não tem FCP geral) gravado e no XML.
	exit, it = emitir(cenarioNota{uf: "GO"})
	if !perto(it.AliqICMS, 0.07) || !perto(it.ValorICMS, 73.5) || !perto(it.ICMSUFDestBase, 1050) || !perto(it.ICMSUFDestValor, 126) || it.FCPUFDestValor != 0 {
		t.Fatalf("não contribuinte GO: aliq %.4f ICMS %.2f base dest %.2f DIFAL %.2f FCP %.2f", it.AliqICMS, it.ValorICMS, it.ICMSUFDestBase, it.ICMSUFDestValor, it.FCPUFDestValor)
	}
	m := payload(exit.ID)
	itemXML := m["items"].([]any)[0].(map[string]any)
	if itemXML["icms_valor_uf_destino"] != 126.0 || itemXML["icms_aliquota_interestadual"] != 7.0 || itemXML["icms_aliquota_interna_uf_destino"] != 19.0 ||
		itemXML["icms_percentual_partilha"] != 100.0 || itemXML["fcp_valor_uf_destino"] != nil {
		t.Fatalf("grupo ICMSUFDest no XML: %v", itemXML)
	}

	// 4. ST pela regra do cadastro, para SP, MVA 40% ajustada (12% → 18%).
	var regraST int64
	if err := pool.QueryRow(ctx, `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,mva_pct,mva_ajustar_interestadual,is_active)
		VALUES($1,'SP','SAIDA','73269090',40,true,true) RETURNING id`, emp).Scan(&regraST); err != nil {
		t.Fatal(err)
	}
	_, it = emitir(cenarioNota{uf: "SP", ie: "110042490114", finalidade: "REVENDA"})
	if it.RegraICMSIPIID == nil || *it.RegraICMSIPIID != regraST || derefS(it.CstICMS) != "10" || !perto(it.BaseICMSST, 1577.56) || !perto(it.ValorICMSST, 163.96) {
		t.Fatalf("ST da regra: regra %v CST %s base ST %.2f ST %.2f", it.RegraICMSIPIID, derefS(it.CstICMS), it.BaseICMSST, it.ValorICMSST)
	}

	// 5. Sem regra, a prévia avisa (empresa não exige regra): ATENÇÃO, não IMPEDE.
	exit, _ = emitir(cenarioNota{uf: "PR", ie: "9012345678"})
	out, err := previa.Execute(ctx, exit.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p := pendencia(out.Pendencias, "itens.regra_icms_ipi"); p == nil || p.Nivel != fiscal_uc.NivelAtencao {
		t.Fatalf("pendência da regra sem exigência: %+v", p)
	}

	// 6. Finalidade inválida é erro de validação, não 500.
	if _, err := criar.Execute(ctx, request.CreateFiscalExitDTO{Serie: "1", DataEmissao: time.Now().Format("2006-01-02"), UFDestinatario: strPtr("PR"),
		FinalidadeCompra: strPtr("REVENDER"), Itens: []request.CreateFiscalExitItemDTO{{Sequence: 1, Quantity: 1, UnitPrice: 1}}}); err == nil || !strings.Contains(err.Error(), "finalidade") {
		t.Fatalf("finalidade inválida: %v", err)
	}
}

// Com a exigência ligada, a linha sem regra impede a transmissão (prévia e
// autorização); com regra, a pendência some.
func TestIntegration_ExigenciaDeRegraImpedeTransmissao(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaTributavel(t, "Exige regra", "3", 0, true)
	ator := testutil.Actor(t, pool)
	item := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,1,$2,$3)`, item, ator, emp)
	repo := fiscalpg.NewFiscalRepositoryPG(pool)
	criar := &fiscal_uc.CreateFiscalExitUseCase{Repo: repo, Auth: &infraauth.AuthService{}, Parametros: fiscalpg.NewParametroICMSIPIPG(pool), Cadastro: fiscalpg.NewCadastroTributarioPG(pool)}
	previa := &fiscal_uc.PreviaNFeUseCase{Repo: repo, Auth: &infraauth.AuthService{}}
	autorizar := &fiscal_uc.AuthorizeFiscalExitUseCase{Repo: repo, Auth: &infraauth.AuthService{}}
	emitir := func() int64 {
		t.Helper()
		ncm := "73269090"
		res, err := criar.Execute(ctx, request.CreateFiscalExitDTO{Serie: "1", DataEmissao: time.Now().Format("2006-01-02"), UFDestinatario: strPtr("PR"), IEDestinatario: strPtr("9012345678"),
			CnpjDestinatario: strPtr("44444444000191"), RazaoSocialDestinatario: strPtr("Cliente"), Cfop: "5102", NaturezaOperacao: "Venda", ValorProdutos: 100,
			Itens: []request.CreateFiscalExitItemDTO{{Sequence: 1, ItemCode: &item, Ncm: &ncm, Cfop: "5102", Quantity: 1, UnitPrice: 100, TotalPrice: 100, OrigemMercadoria: "0"}}})
		if err != nil {
			t.Fatal(err)
		}
		return res.ID
	}
	semRegra := emitir()
	out, err := previa.Execute(ctx, semRegra)
	if err != nil {
		t.Fatal(err)
	}
	if p := pendencia(out.Pendencias, "itens.regra_icms_ipi"); p == nil || p.Nivel != fiscal_uc.NivelImpede {
		t.Fatalf("pendência com exigência: %+v, quer IMPEDE", p)
	}
	if _, err := autorizar.Execute(ctx, semRegra); err == nil || !strings.Contains(err.Error(), "regra de ICMS/IPI") {
		t.Fatalf("autorização sem regra: %v, quer recusa pela regra", err)
	}

	testutil.Exec(t, pool, `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,icms_pct_contrib,is_active) VALUES($1,'PR','SAIDA','7326',12,true)`, emp)
	comRegra := emitir()
	out, err = previa.Execute(ctx, comRegra)
	if err != nil {
		t.Fatal(err)
	}
	if p := pendencia(out.Pendencias, "itens.regra_icms_ipi"); p != nil {
		t.Fatalf("com regra não pode haver pendência: %+v", p)
	}
}

// Empresa do Simples: CSOSN 101 com crédito, sem ICMS/IPI/PIS/COFINS
// destacados; o XML leva pCredSN e vCredICMSSN e não leva base do ICMS.
func TestIntegration_SimplesNacionalCSOSN(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaTributavel(t, "Simples teste", "1", 2.56, false)
	ator := testutil.Actor(t, pool)
	item := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,1,$2,$3)`, item, ator, emp)
	repo := fiscalpg.NewFiscalRepositoryPG(pool)
	criar := &fiscal_uc.CreateFiscalExitUseCase{Repo: repo, Auth: &infraauth.AuthService{}, Parametros: fiscalpg.NewParametroICMSIPIPG(pool), Cadastro: fiscalpg.NewCadastroTributarioPG(pool)}
	previa := &fiscal_uc.PreviaNFeUseCase{Repo: repo, Auth: &infraauth.AuthService{}}
	ncm := "73269090"
	res, err := criar.Execute(ctx, request.CreateFiscalExitDTO{Serie: "1", DataEmissao: time.Now().Format("2006-01-02"), UFDestinatario: strPtr("PR"), IEDestinatario: strPtr("9012345678"),
		CnpjDestinatario: strPtr("44444444000191"), RazaoSocialDestinatario: strPtr("Cliente"), Cfop: "5102", NaturezaOperacao: "Venda", ValorProdutos: 1000,
		FinalidadeCompra: strPtr("REVENDA"),
		Itens:            []request.CreateFiscalExitItemDTO{{Sequence: 1, ItemCode: &item, Ncm: &ncm, Cfop: "5102", Quantity: 1, UnitPrice: 1000, TotalPrice: 1000, OrigemMercadoria: "0"}}})
	if err != nil {
		t.Fatal(err)
	}
	itens, _ := repo.GetExitItems(ctx, res.ID)
	it := itens[0]
	if derefS(it.CstICMS) != "101" || it.ValorICMS != 0 || it.ValorIPI != 0 || it.ValorPIS != 0 || !perto(it.AliqCreditoSimples, 2.56) || !perto(it.ValorCreditoSimples, 25.6) || derefS(it.CstPIS) != "49" {
		t.Fatalf("Simples: CSOSN %s ICMS %.2f IPI %.2f PIS %.2f (CST %s) crédito %.2f%% %.2f", derefS(it.CstICMS), it.ValorICMS, it.ValorIPI, it.ValorPIS, derefS(it.CstPIS), it.AliqCreditoSimples, it.ValorCreditoSimples)
	}
	out, err := previa.Execute(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range out.Pendencias {
		if p.Nivel == fiscal_uc.NivelImpede && strings.HasPrefix(p.Campo, "itens") {
			t.Fatalf("pendência inesperada no Simples: %+v", p)
		}
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out.PayloadEnviado), &m); err != nil {
		t.Fatal(err)
	}
	x := m["items"].([]any)[0].(map[string]any)
	if x["icms_situacao_tributaria"] != "101" || x["icms_aliquota_credito_simples"] != 2.56 || x["icms_valor_credito_simples"] != 25.6 {
		t.Fatalf("XML do Simples: %v", x)
	}
	if _, tem := x["icms_base_calculo"]; tem {
		t.Fatalf("CSOSN 101 não leva base de ICMS: %v", x)
	}
	if info, _ := m["informacoes_adicionais_contribuinte"].(string); !strings.Contains(info, "OPTANTE PELO SIMPLES NACIONAL") || !strings.Contains(info, "R$ 25,60") {
		t.Fatalf("infCpl do Simples: %q", info)
	}
}

// O simulador calcula sem gravar e explica cada linha.
func TestIntegration_SimuladorDeTributacao(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaTributavel(t, "Simulador", "3", 0, true)
	repo := fiscalpg.NewFiscalRepositoryPG(pool)
	sim := &fiscal_uc.SimularTributacaoUseCase{
		Tributador: &fiscal_uc.Tributador{Repo: repo, Parametros: fiscalpg.NewParametroICMSIPIPG(pool), Cadastro: fiscalpg.NewCadastroTributarioPG(pool)},
		Auth:       &infraauth.AuthService{},
	}
	var antes int64
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fiscal_exits WHERE enterprise_id=$1`, emp).Scan(&antes)
	out, err := sim.Execute(ctx, fiscal_uc.SimularTributacaoDTO{UFDestino: strPtr("GO"),
		Itens: []fiscal_uc.SimularTributacaoItemDTO{{Ncm: strPtr("73269090"), Quantidade: 2, ValorUnitario: 500}}})
	if err != nil {
		t.Fatal(err)
	}
	l := out.Itens[0]
	if out.TipoDestinatario != "nao_contribuinte" || !out.ConsumidorFinal || !perto(l.AliqICMSPct, 7) || !perto(l.ValorDIFAL, 126) || len(l.Notas) == 0 {
		t.Fatalf("simulação GO: %+v", out)
	}
	if len(out.Pendencias) == 0 || out.Pendencias[0].Nivel != fiscal_uc.NivelImpede {
		t.Fatalf("empresa exige regra: a simulação tem de apontar IMPEDE, deu %+v", out.Pendencias)
	}
	if !perto(out.ValorTotalNota, 1050) {
		t.Fatalf("total da nota simulada: %.2f, quer 1050 (produtos + IPI)", out.ValorTotalNota)
	}
	var depois int64
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fiscal_exits WHERE enterprise_id=$1`, emp).Scan(&depois)
	if depois != antes {
		t.Fatalf("o simulador gravou nota: %d → %d", antes, depois)
	}
	if _, err := sim.Execute(ctx, fiscal_uc.SimularTributacaoDTO{Itens: []fiscal_uc.SimularTributacaoItemDTO{{Ncm: strPtr("73269090"), Quantidade: 1, ValorUnitario: 1}}}); err == nil {
		t.Fatal("sem UF de destino a simulação tem de pedir a UF")
	}

	// Item informado pelo código comercial: o NCM vem da classificação fiscal
	// de venda do cadastro e a regra do item vale.
	ator := testutil.Actor(t, pool)
	interno := testutil.UniqueCode()
	comercial := fmt.Sprintf("SIM-%d", interno)
	classif := interno % 1_000_000
	testutil.Exec(t, pool, `INSERT INTO fiscal_classifications(code,description,ncm,enterprise_id,created_by) VALUES($1,'Peças','73269090',$2,$3)`, classif, emp, ator)
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM fiscal_classifications WHERE enterprise_id=$1`, emp) })
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,accounting_sale_fiscal_classification_code) VALUES($1,$2,1,$3,$4,$5)`,
		interno, comercial, ator, emp, fmt.Sprint(classif))
	var regra int64
	if err := pool.QueryRow(ctx, `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,item_code,icms_pct_non_contrib,is_active) VALUES($1,'GO','SAIDA',$2,4,true) RETURNING id`, emp, interno).Scan(&regra); err != nil {
		t.Fatal(err)
	}
	sim.ItemsRepo = item.NewRepositoryItemSQLC(sqlc.New(pool))
	sim.Itens = fiscalpg.NewFiscalEntryDocumentRepositoryPG(repo)
	codigo := request.TextCode(comercial)
	out, err = sim.Execute(ctx, fiscal_uc.SimularTributacaoDTO{UFDestino: strPtr("GO"),
		Itens: []fiscal_uc.SimularTributacaoItemDTO{{ItemCode: &codigo, Quantidade: 1, ValorUnitario: 1000}}})
	if err != nil {
		t.Fatal(err)
	}
	l = out.Itens[0]
	if l.Ncm != "73269090" || l.RegraICMSIPI == nil || *l.RegraICMSIPI != regra || !perto(l.AliqICMSPct, 4) || len(out.Pendencias) != 0 {
		t.Fatalf("simulação pelo código comercial: %+v pendências %+v", l, out.Pendencias)
	}

	// Sem a UF da empresa a operação interna parece interestadual: a simulação
	// tem de impedir em vez de mostrar alíquota e MVA errados sem aviso.
	testutil.Exec(t, pool, `UPDATE fiscal_configs SET uf_empresa='' WHERE enterprise_id=$1`, emp)
	out, err = sim.Execute(ctx, fiscal_uc.SimularTributacaoDTO{UFDestino: strPtr("PR"),
		Itens: []fiscal_uc.SimularTributacaoItemDTO{{Ncm: strPtr("73269090"), Quantidade: 1, ValorUnitario: 1000}}})
	if err != nil {
		t.Fatal(err)
	}
	semUF := false
	for _, p := range out.Pendencias {
		semUF = semUF || (p.Nivel == fiscal_uc.NivelImpede && strings.Contains(p.Mensagem, "UF da empresa"))
	}
	if !semUF {
		t.Fatalf("UF da empresa vazia sem IMPEDE: %+v", out.Pendencias)
	}
}

func pendencia(ps []response.PreviaPendenciaNFe, campo string) *response.PreviaPendenciaNFe {
	for i := range ps {
		if ps[i].Campo == campo {
			return &ps[i]
		}
	}
	return nil
}

func derefS(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

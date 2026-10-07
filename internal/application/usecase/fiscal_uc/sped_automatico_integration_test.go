//go:build integration

package fiscal_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func linhasEFD(arquivo, reg string) [][]string {
	var out [][]string
	for _, l := range strings.Split(arquivo, "\r\n") {
		if strings.HasPrefix(l, "|"+reg+"|") {
			out = append(out, strings.Split(l, "|"))
		}
	}
	return out
}

func TestIntegration_SpedAutomatico(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Sped',$2,'x')`, actor, actor.String()+"@example.test")

	novaEmpresa := func(n int64) (int64, context.Context) {
		code := int64(1_300_000_000+u%90_000_000) + n
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'SPED') RETURNING id`, code).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id, context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: id, EnterpriseCode: code})
	}
	empresa, ctx := novaEmpresa(0)
	outra, ctxOutra := novaEmpresa(1)
	semConfig, ctxSemConfig := novaEmpresa(2)
	t.Cleanup(func() {
		for _, e := range []int64{empresa, outra, semConfig} {
			for _, q := range []string{
				`DELETE FROM fiscal_freight_documents WHERE enterprise_id=$1`,
				`DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`,
				`DELETE FROM fiscal_exits WHERE enterprise_id=$1`, `DELETE FROM notification_outbox WHERE enterprise_id=$1`,
				`DELETE FROM fiscal_entries WHERE enterprise_id=$1`, `DELETE FROM stock_movements WHERE enterprise_id=$1`,
				`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
				`DELETE FROM fiscal_configs WHERE enterprise_id=$1`, `DELETE FROM special_adjustment_notes WHERE empresa_id=$1`,
				`DELETE FROM enterprise WHERE id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e); err != nil {
					t.Errorf("limpeza (%s): %v", q, err)
				}
			}
		}
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, actor)
	})

	for _, e := range []int64{empresa, outra} {
		testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,ie_empresa,uf_empresa,codigo_municipio,vencimento_icms_dia,updated_by,enterprise_id)
			VALUES('52454668000102','TECNOFER','9012345678','PR','4106902',12,$1,$2)`, actor, e)
	}
	chapa, produto := u+70, u+71
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WS-%d", u), actor, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,created_by,enterprise_id,name,warehouse_unit_of_measurement,engineering_type,warehouse_code)
		VALUES($1,($1::bigint)::text,$3,$4,'CHAPA ACO','KG',1,$5),($2,($2::bigint)::text,$3,$4,'SUPORTE','UN',0,$5)`, chapa, produto, actor, empresa, almox)

	// Estoque para o bloco H (inventário em 31/08): chapa entra 100 por 1.000,
	// sai 30 pelo custo médio (300), ganha 50 de frete (AJUSTE_CUSTO) e entra
	// mais 10 só em setembro (fora). Item de terceiro fica fora do inventário.
	terceiro := u + 72
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,created_by,enterprise_id,name,warehouse_unit_of_measurement,engineering_type,warehouse_code)
		VALUES($1,($1::bigint)::text,$2,$3,'MOLDE DO CLIENTE','UN',2,$4)`, terceiro, actor, empresa, almox)
	testutil.Exec(t, pool, `INSERT INTO stock_movements(item_code,warehouse_id,movement_type,quantity,unit_price,total_price,created_at,created_by,enterprise_id) VALUES
		($1,$3,'IN',100,10,1000,'2026-08-05 10:00:00-03',$4,$5),
		($1,$3,'OUT',30,10,300,'2026-08-10 10:00:00-03',$4,$5),
		($1,$3,'AJUSTE_CUSTO',0,0,50,'2026-08-12 10:00:00-03',$4,$5),
		($1,$3,'IN',10,11,110,'2026-09-02 10:00:00-03',$4,$5),
		($2,$3,'IN',5,0,0,'2026-08-05 10:00:00-03',$4,$5)`, chapa, terceiro, almox, actor, empresa)

	xmlNFe := `<nfeProc><NFe><infNFe><emit><CNPJ>11222333000181</CNPJ><IE>1234567</IE><enderEmit><xLgr>RUA A</xLgr><nro>10</nro><xBairro>CENTRO</xBairro><cMun>4106902</cMun></enderEmit></emit></infNFe></NFe></nfeProc>`
	insEntrada := func(e int64, numero int64, dataEntrada, status string) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_entries(chave_acesso,numero_nf,serie,modelo,data_emissao,data_entrada,cnpj_emitente,
			razao_social_emitente,ie_emitente,valor_produtos,valor_frete,valor_ipi,valor_icms,valor_total,status,created_by,enterprise_id,
			modalidade_frete,xml_content)
			VALUES($1,$2,'1','55',$3::date - 1,$3::date,'11222333000181','ACO SUL','1234567',1200,50,60,150,1310,$4,$5,$6,'1',$7) RETURNING id`,
			fmt.Sprintf("41%042d", u*10+numero), numero, dataEntrada, status, actor, e, xmlNFe).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	entrada := insEntrada(empresa, 77102, "2026-08-03", "APPROVED")
	insEntrada(empresa, 77103, "2026-08-04", "PENDING")  // não aprovada: fora
	insEntrada(empresa, 77104, "2026-09-01", "APPROVED") // mês seguinte: fora
	insEntrada(outra, 77105, "2026-08-05", "APPROVED")   // outra empresa: fora
	testutil.Exec(t, pool, `INSERT INTO fiscal_entry_items(fiscal_entry_id,sequence,item_code,ncm,cfop,cfop_entrada,quantity,unit_price,total_price,
		base_icms,aliq_icms,valor_icms,base_ipi,aliq_ipi,valor_ipi,cst_icms,origem_mercadoria,gera_credito_icms,gera_credito_ipi,
		valor_frete,valor_contabil,uom,fator_conversao)
		VALUES($1,1,$2,'72085100','5101','1101',1,1000,1000,1041.67,0.12,125,1000,0.05,50,'00','0',TRUE,TRUE,41.67,1091.67,'TN',1000),
		      ($1,2,NULL,'40151900','5102','1556',10,20,200,208.33,0.12,25,200,0.05,10,'00','0',FALSE,FALSE,8.33,218.33,'PC',NULL)`, entrada, chapa)

	insSaida := func(numero int64, status string) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_exits(chave_acesso,numero_nf,serie,data_emissao,cnpj_destinatario,razao_social_destinatario,
			cfop,natureza_operacao,valor_produtos,valor_frete,valor_desconto,valor_icms,valor_total,status,created_by,enterprise_id,
			dest_codigo_municipio,condicao_pagamento_id)
			VALUES($1,$2,'1','2026-08-10'::date,'44555666000190','CLIENTE B','6101','VENDA',2000,30,20,204,2010,$3,$4,$5,'4314902',NULL) RETURNING id`,
			fmt.Sprintf("41%042d", u*10+numero), numero, status, actor, empresa).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	saida := insSaida(501, "AUTHORIZED")
	cancelada := insSaida(502, "CANCELLED")
	insSaida(503, "DRAFT")
	for _, x := range []int64{saida, cancelada} {
		testutil.Exec(t, pool, `INSERT INTO fiscal_exit_items(fiscal_exit_id,sequence,item_code,ncm,cfop,quantity,unit_price,total_price,base_icms,aliq_icms,valor_icms,cst_icms,origem_mercadoria)
			VALUES($1,1,$2,'73269090','6101',10,100,1000,1000,0.12,120,'00','0'),($1,2,$2,'73269090','6101',7,100,700,700,0.12,84,'00','0'),
			      ($1,3,$2,'73269090','6102',3,100,300,0,0,0,'40','0')`, x, produto)
	}

	xmlCTe := `<cteProc><CTe><infCte><ide><cMunIni>4106902</cMunIni><cMunFim>4106902</cMunFim></ide><emit><CNPJ>77888999000100</CNPJ><enderEmit><cMun>4106902</cMun></enderEmit></emit></infCte></CTe></cteProc>`
	testutil.Exec(t, pool, `INSERT INTO fiscal_freight_documents(enterprise_id,chave_cte,numero,serie,data_emissao,cnpj_transportadora,nome_transportadora,
		cfop,valor_frete,base_icms,aliq_icms,valor_icms,credita_icms,data_vencimento,status,created_by,lancado_em,xml_content)
		VALUES($1,$2,900,'1','2026-08-03','77888999000100','TRANSP','5353',50,50,12,6,TRUE,'2026-09-03','LANCADO',$3,'2026-08-04 12:00:00-03',$4)`,
		empresa, fmt.Sprintf("41%042d", u*10+9), actor, xmlCTe)

	// Nota especial de ajuste emitida (crédito presumido de 13) entra no E111;
	// a em rascunho não.
	var codAjuste int64
	if err := pool.QueryRow(bg, `INSERT INTO icms_apuracao_adjustment_codes(code,uf,description,valid_from) VALUES('PR020009','PR','CREDITO PRESUMIDO TESTE','2020-01-01') RETURNING id`).Scan(&codAjuste); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Antes do código, as notas que o usam (a limpeza por empresa roda depois).
		if _, err := pool.Exec(bg, `DELETE FROM special_adjustment_notes WHERE adjustment_code_id=$1`, codAjuste); err != nil {
			t.Errorf("limpeza das notas de ajuste: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM icms_apuracao_adjustment_codes WHERE id=$1`, codAjuste); err != nil {
			t.Errorf("limpeza do código de ajuste: %v", err)
		}
	})
	testutil.Exec(t, pool, `INSERT INTO special_adjustment_notes(empresa_id,purpose,status,issue_date,period,adjustment_code_id,history,auto_generate_summary,total_value,total_icms,total_ipi)
		VALUES($1,'AJUSTE','EMITIDA','2026-08-20','2026-08',$2,'credito presumido',FALSE,0,13,0),
		      ($1,'AJUSTE','RASCUNHO','2026-08-21','2026-08',$2,'rascunho',FALSE,0,500,0)`, empresa, codAjuste)

	fiscalRepo := fiscalpg.NewFiscalRepositoryPG(pool)
	uc := &SpedAutomaticoUseCase{Periodo: fiscalpg.NewSpedRepositoryPG(fiscalRepo), Config: fiscalRepo}
	req := SpedAutomaticoRequest{Ano: 2026, Mes: 8, ContabilistaNome: "Contador", ContabilistaCPF: "111.222.333-44",
		SaldoCredorAnteriorICMS: decimal.NewFromInt(10), CodReceitaICMS: "1015"}
	res, err := uc.Gerar(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	r := res.Resumo
	if r.Entradas != 1 || r.Saidas != 1 || r.Canceladas != 1 || r.Fretes != 1 {
		t.Fatalf("resumo = %+v", r)
	}
	// Débitos 204; créditos 125 (só o item com crédito) + 6 do CT-e; saldo
	// anterior 10; ajuste de crédito presumido 13 (E111) → 50 a recolher.
	if !r.ICMSDebitos.Equal(decimal.NewFromInt(204)) || !r.ICMSCreditos.Equal(decimal.NewFromInt(131)) || !r.ICMSRecolher.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("apuração = %+v", r)
	}
	if !r.IPICreditos.Equal(decimal.NewFromInt(50)) {
		t.Errorf("crédito de IPI = %s, want 50", r.IPICreditos)
	}
	a := res.Arquivo
	if !strings.HasSuffix(a, "\r\n") || !strings.HasPrefix(a, "|0000|020|0|01082026|31082026|TECNOFER|52454668000102|") {
		t.Fatalf("cabeçalho: %q", a[:min(120, len(a))])
	}
	c100 := linhasEFD(a, "C100")
	if len(c100) != 3 {
		t.Fatalf("C100 = %d", len(c100))
	}
	if c100[0][2] != "0" || c100[0][8] != "77102" || c100[0][11] != "03082026" || c100[0][22] != "125,00" {
		t.Errorf("C100 da entrada = %v", c100[0])
	}
	c170 := linhasEFD(a, "C170")
	if len(c170) != 2 || c170[0][14] != "12,00" {
		t.Errorf("ALIQ_ICMS do C170 = %v (a nota guarda fração; a EFD quer %%)", c170)
	}
	for _, l := range linhasEFD(a, "C190") {
		if l[3] == "6101" && l[4] != "12,00" {
			t.Errorf("ALIQ_ICMS do C190 de saída = %s", l[4])
		}
	}
	if len(c170) != 2 || c170[0][11] != "1101" || c170[1][11] != "1556" || c170[1][15] != "0,00" {
		t.Errorf("C170 = %v", c170)
	}
	// O item sem cadastro entra com o código da linha da nota; o cadastrado, com o seu.
	if c170[0][3] != fmt.Sprint(chapa) || !strings.HasPrefix(c170[1][3], "NFE") {
		t.Errorf("COD_ITEM = %s / %s", c170[0][3], c170[1][3])
	}
	if l := linhasEFD(a, "0220"); len(l) != 1 || l[0][2] != "TN" || l[0][3] != "1000,000000" {
		t.Errorf("0220 = %v", l)
	}
	if l := linhasEFD(a, "E111"); len(l) != 1 || l[0][2] != "PR020009" || l[0][4] != "13,00" {
		t.Errorf("E111 = %v", l)
	}
	if l := linhasEFD(a, "E116"); len(l) != 1 || l[0][3] != "50,00" || l[0][4] != "12092026" || l[0][5] != "1015" {
		t.Errorf("E116 = %v", l)
	}
	d100 := linhasEFD(a, "D100")
	if len(d100) != 1 || d100[0][12] != "04082026" || d100[0][24] != "4106902" || d100[0][25] != "4106902" {
		t.Errorf("D100 = %v", d100)
	}
	for _, p := range linhasEFD(a, "0150") {
		if p[8] == "" {
			t.Errorf("0150 sem município: %v", p)
		}
	}
	if len(res.Avisos) != 0 {
		t.Errorf("avisos inesperados: %v", res.Avisos)
	}
	// Contagens do bloco 9 batem com o arquivo.
	total := strings.Count(a, "\r\n")
	if l := linhasEFD(a, "9999"); len(l) != 1 || l[0][2] != fmt.Sprint(total) {
		t.Errorf("9999 = %v, linhas = %d", l, total)
	}

	// Bloco H: inventário em 31/08 a pedido.
	inv := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	reqInv := req
	reqInv.InventarioData = &inv
	resInv, err := uc.Gerar(ctx, reqInv)
	if err != nil {
		t.Fatal(err)
	}
	if resInv.Resumo.ItensInventario != 1 || !resInv.Resumo.ValorInventario.Equal(decimal.NewFromInt(750)) {
		t.Fatalf("inventário: %d itens, %s (want 1 item, 750)", resInv.Resumo.ItensInventario, resInv.Resumo.ValorInventario)
	}
	h005 := linhasEFD(resInv.Arquivo, "H005")
	h010 := linhasEFD(resInv.Arquivo, "H010")
	if len(h005) != 1 || h005[0][2] != "31082026" || h005[0][3] != "750,00" || h005[0][4] != "01" {
		t.Errorf("H005 = %v", h005)
	}
	if len(h010) != 1 || h010[0][2] != fmt.Sprint(chapa) || h010[0][4] != "70,00000" || h010[0][5] != "10,714286" || h010[0][6] != "750,00" {
		t.Errorf("H010 = %v", h010)
	}
	// Sem a conta de estoque nos parâmetros contábeis, o aviso aparece.
	semConta := false
	for _, a := range resInv.Avisos {
		if strings.Contains(a, "COD_CTA") {
			semConta = true
		}
	}
	if !semConta {
		t.Errorf("faltou o aviso da conta de estoque: %v", resInv.Avisos)
	}
	posterior := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	reqInv.InventarioData = &posterior
	if _, err := uc.Gerar(ctx, reqInv); err == nil {
		t.Error("inventário depois do fim do período deveria ser recusado")
	}

	// A outra empresa só vê a sua nota.
	res2, err := uc.Gerar(ctxOutra, req)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Resumo.Entradas != 1 || res2.Resumo.Saidas != 0 || res2.Resumo.Fretes != 0 {
		t.Errorf("isolamento: %+v", res2.Resumo)
	}

	// Validações.
	if _, err := uc.Gerar(ctx, SpedAutomaticoRequest{Ano: 2026, Mes: 8}); err == nil {
		t.Error("sem contabilista deveria falhar")
	}
	// Empresa sem configuração fiscal: erro de validação, não 500.
	_, err = uc.Gerar(ctxSemConfig, req)
	var ve *errorsuc.ValidationError
	if !errors.As(err, &ve) || !strings.Contains(err.Error(), "VFIS0100") {
		t.Errorf("sem configuração fiscal: %v", err)
	}
	testutil.Exec(t, pool, `UPDATE fiscal_configs SET codigo_municipio='' WHERE enterprise_id=$1`, outra)
	if _, err := uc.Gerar(ctxOutra, req); err == nil || !strings.Contains(err.Error(), "município") {
		t.Errorf("config incompleta: %v", err)
	}
}

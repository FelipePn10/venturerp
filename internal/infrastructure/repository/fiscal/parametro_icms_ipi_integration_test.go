//go:build integration

package fiscal_test

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Hierarquia da regra de ICMS/IPI aplicada na nota de saída: item > NCM
// completo > capítulo do NCM; cliente > segmento do cliente > geral;
// preferencial desempata. Regras de outra UF, de entrada, de tipo de nota,
// de máscara, do Simples (empresa no Lucro Real) e de outra empresa não valem.
func TestParametroICMSIPIHierarquia(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaFiscal(t)
	ator := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	item := u
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,1,$2,$3)`, item, ator, emp)
	var segmento int64
	if err := pool.QueryRow(ctx, `INSERT INTO market_segments(code,description,enterprise_id) VALUES($1,'Revenda',$2) RETURNING id`, u%1_000_000, emp).Scan(&segmento); err != nil {
		t.Fatal(err)
	}
	clienteSeg, clienteSem := u+1, u+2
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id,market_segment_id) VALUES($1,'Revendedora','11111111000191',$2,$3,$4),($5,'Consumidor','22222222000191',$2,$3,NULL)`,
		clienteSeg, ator, emp, segmento, clienteSem)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM icms_ipi_tax_params WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM customers WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM market_segments WHERE enterprise_id=$1`, emp)
	})
	ins := `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,item_code,customer_code,market_segment_id,is_preferred,icms_pct_contrib,is_active)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,true) RETURNING id`
	regra := func(uf, op string, ncm *string, it, cli, seg *int64, pref bool, pct float64) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, ins, emp, uf, op, ncm, it, cli, seg, pref, pct).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	s := func(v string) *string { return &v }
	r := fiscalpg.NewParametroICMSIPIPG(pool)
	ncm := "7326.90.00"
	quer := func(nome string, it *int64, cli *int64, id int64, pct float64) {
		t.Helper()
		p, err := r.ParametroICMSIPI(ctx, engine.ConsultaParametroICMSIPI{ItemCode: it, NCM: &ncm, UF: "pr", Cliente: cli})
		if err != nil {
			t.Fatal(err)
		}
		if id == 0 {
			if p != nil {
				t.Fatalf("%s: casou a regra %d, quer nenhuma", nome, p.ID)
			}
			return
		}
		if p == nil || p.ID != id || p.Contrib.AliqICMSPct != pct {
			t.Fatalf("%s: regra %+v, quer %d (%.2f%%)", nome, p, id, pct)
		}
	}

	// Regras que nunca podem casar.
	regra("SP", "SAIDA", s("73269000"), nil, nil, nil, false, 1)
	regra("PR", "ENTRADA", s("73269000"), nil, nil, nil, false, 2)
	testutil.Exec(t, pool, `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,item_code,is_simples_optante,icms_pct_contrib,is_active) VALUES($1,'PR','SAIDA',$2,true,3,true)`, emp, item)
	testutil.Exec(t, pool, `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,item_code,item_config_mask,icms_pct_contrib,is_active) VALUES($1,'PR','SAIDA',$2,'AZUL',4,true)`, emp, item)
	quer("nada cadastrado que valha", &item, nil, 0, 0)

	capitulo := regra("PR", "AMBAS", s("7326"), nil, nil, nil, false, 12)
	quer("capítulo do NCM", &item, nil, capitulo, 12)
	completo := regra("PR", "SAIDA", s("7326.90.00"), nil, nil, nil, false, 17)
	quer("NCM completo vence o capítulo", &item, nil, completo, 17)
	porSegmento := regra("PR", "SAIDA", s("73269000"), nil, nil, &segmento, false, 18)
	quer("segmento do cliente", &item, &clienteSeg, porSegmento, 18)
	quer("cliente sem o segmento fica no NCM", &item, &clienteSem, completo, 17)
	porCliente := regra("PR", "SAIDA", s("73269000"), nil, &clienteSeg, nil, false, 19)
	quer("cliente vence segmento", &item, &clienteSeg, porCliente, 19)
	// Empresa do Simples: a regra de optante (do item) passa a valer e vence o NCM.
	if p, err := r.ParametroICMSIPI(ctx, engine.ConsultaParametroICMSIPI{ItemCode: &item, NCM: &ncm, UF: "PR", Simples: true}); err != nil || p == nil || p.Contrib.AliqICMSPct != 3 {
		t.Fatalf("empresa do Simples: %+v %v, quer a regra de optante (3%%)", p, err)
	}
	doItem := regra("PR", "SAIDA", nil, &item, nil, nil, false, 20)
	quer("item vence NCM, mesmo com cliente", &item, &clienteSeg, doItem, 20)
	preferido := regra("PR", "SAIDA", nil, &item, nil, nil, true, 21)
	quer("preferencial desempata", &item, nil, preferido, 21)

	// Outra empresa não enxerga as regras desta.
	outra, _ := testutil.DuasEmpresas(t, pool, nil)
	if p, err := r.ParametroICMSIPI(outra.Ctx, engine.ConsultaParametroICMSIPI{ItemCode: &item, NCM: &ncm, UF: "PR"}); err != nil || p != nil {
		t.Fatalf("outra empresa: %+v %v", p, err)
	}
}

// Filtros de cenário: regra de finalidade, de tipo de nota e da tributação do
// cliente só valem quando a nota tem o mesmo dado, e vencem a regra genérica;
// regra de entrada só vale na entrada. A regra traz CSOSN, MVA, ST e DIFAL.
func TestParametroICMSIPIFiltrosDeCenario(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaFiscal(t)
	ator := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	var tipoNota, tributacao int64
	if err := pool.QueryRow(ctx, `INSERT INTO invoice_types(code,description,enterprise_id) VALUES($1,'Venda ST',$2) RETURNING id`, u%1_000_000, emp).Scan(&tipoNota); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO tax_types(code,description,is_consumer,enterprise_id) VALUES($1,'Consumidor',true,$2) RETURNING id`, u%1_000_000, emp).Scan(&tributacao); err != nil {
		t.Fatal(err)
	}
	cliente := u + 1
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id,tax_type_id) VALUES($1,'Cliente','33333333000191',$2,$3,$4)`, cliente, ator, emp, tributacao)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM icms_ipi_tax_params WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM customers WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM invoice_types WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM tax_types WHERE enterprise_id=$1`, emp)
	})
	ncm := "8481.80.99"
	regra := func(sql string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, sql, append([]any{emp}, args...)...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	geral := regra(`INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,icms_pct_contrib,csosn_icms,mva_pct,mva_ajustar_interestadual,icms_subst_red_pct,icms_pct_for_st_contrib,icms_difal_type,icms_difal_red_pct,icms_acres_pct_non_contrib,icms_acres_type_non_contrib)
		VALUES($1,'SP','SAIDA','84818099',12,'101',40,false,10,18,'NAO_CONSIDERA',20,2,'FUNDO_COMBATE_POBREZA') RETURNING id`)
	usoConsumo := regra(`INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,icms_pct_contrib,finalidade_compra) VALUES($1,'SP','SAIDA','84818099',7,'USO_CONSUMO') RETURNING id`)
	doTipo := regra(`INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,icms_pct_contrib,invoice_type_exit_id) VALUES($1,'SP','SAIDA','84818099',4,$2) RETURNING id`, tipoNota)
	entrada := regra(`INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,icms_pct_contrib) VALUES($1,'SP','ENTRADA','84818099',2) RETURNING id`)

	r := fiscalpg.NewParametroICMSIPIPG(pool)
	busca := func(nome string, q engine.ConsultaParametroICMSIPI, quer int64) *engine.ParametroICMSIPI {
		t.Helper()
		q.NCM, q.UF = &ncm, "SP"
		p, err := r.ParametroICMSIPI(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		if p == nil || p.ID != quer {
			t.Fatalf("%s: regra %+v, quer %d", nome, p, quer)
		}
		return p
	}
	p := busca("sem finalidade nem tipo: a geral", engine.ConsultaParametroICMSIPI{}, geral)
	if p.CSOSN != "101" || p.MvaPct != 40 || p.MvaAjustar || p.RedBaseSTPct != 10 || p.Contrib.AliqSTPct != 18 || p.DifalTipo != "NAO_CONSIDERA" || p.DifalRedPct != 20 || p.FCPProdutoPct != 2 {
		t.Fatalf("campos de cenário não lidos: %+v", p)
	}
	busca("revenda não pega a regra de uso e consumo", engine.ConsultaParametroICMSIPI{Finalidade: "REVENDA"}, geral)
	busca("uso e consumo vence a geral", engine.ConsultaParametroICMSIPI{Finalidade: "USO_CONSUMO"}, usoConsumo)
	busca("tipo de nota vence a finalidade", engine.ConsultaParametroICMSIPI{Finalidade: "USO_CONSUMO", TipoNotaID: &tipoNota}, doTipo)
	outroTipo := tipoNota + 1_000_000
	busca("outro tipo de nota não pega a regra do tipo", engine.ConsultaParametroICMSIPI{TipoNotaID: &outroTipo}, geral)
	busca("entrada só pega regra de entrada", engine.ConsultaParametroICMSIPI{Entrada: true}, entrada)

	// Regra da tributação do cliente: só para clientes com essa tributação.
	daTributacao := regra(`INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,icms_pct_contrib,tax_type_id) VALUES($1,'SP','SAIDA','84818099',9,$2) RETURNING id`, tributacao)
	busca("cliente com a tributação", engine.ConsultaParametroICMSIPI{Cliente: &cliente}, daTributacao)
	busca("sem cliente não pega a regra da tributação", engine.ConsultaParametroICMSIPI{}, geral)
}

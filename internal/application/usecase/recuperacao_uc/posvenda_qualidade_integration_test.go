//go:build integration

package recuperacao_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/capa_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/indicadores_pv_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/ordem_servico_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/recuperacao_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/serie_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc"
	recdom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/recuperacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	capapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/capa"
	indpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/indicadorespv"
	item_ "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	ospg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/ordemservico"
	recpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/recuperacao"
	seriepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/serie"
	tapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// Do chamado em garantia aos indicadores: a OS concluída entra no custo da
// garantia por item, defeito e lote; a série que falhou entra na taxa de falha
// e no MTBF; o chamado abre a CAPA de reclamação do cliente (uma só); o custo
// é cobrado do fornecedor e o recuperado abate o custo líquido. Outra empresa
// não vê nem mexe em nada disso.
func TestPosVendaIndicadoresCAPAERecuperacao(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	equip := 91_000_000 + u%1_000_000
	cliente, fornecedor := 840_000+u%90_000, 850_000+u%90_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM recuperacoes_fornecedor WHERE enterprise_id=$1`,
				`DELETE FROM ordens_servico WHERE enterprise_id=$1`, `DELETE FROM os_parametros WHERE enterprise_id=$1`,
				`DELETE FROM technical_assistance_calls WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`ALTER TABLE quality_capa_eventos DISABLE TRIGGER trg_quality_capa_eventos_imutaveis`,
				`DELETE FROM quality_capas WHERE enterprise_id=$1`,
				`ALTER TABLE quality_capa_eventos ENABLE TRIGGER trg_quality_capa_eventos_imutaveis`,
				`DELETE FROM technical_assistance_defect_reasons WHERE enterprise_id=$1`, `DELETE FROM technical_assistance_defect_groups WHERE enterprise_id=$1`,
				`DELETE FROM numeros_serie WHERE enterprise_id=$1`, `DELETE FROM item_pos_venda WHERE enterprise_id=$1`,
				`DELETE FROM employees WHERE enterprise_id=$1`, `DELETE FROM suppliers WHERE enterprise_id=$1`, `DELETE FROM customers WHERE enterprise_id=$1`,
				`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			} {
				args := []any{e.ID}
				if strings.HasPrefix(q, "ALTER") {
					args = nil
				}
				if _, err := pool.Exec(bg, q, args...); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	// O responsável da CAPA precisa ser usuário ativo da empresa.
	lider := uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Líder PVQ',$2,'x')`, lider, lider.String()+"@example.test")
	defer func() { _, _ = pool.Exec(bg, `UPDATE users SET is_active=FALSE WHERE id=$1`, lider) }()
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id, enterprise_id, role) VALUES ($1,$2,'ADMIN') ON CONFLICT DO NOTHING`, lider, a.ID)
	var almox, tecnico, motivo int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Assistência',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("PVQ-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,commercial_warranty_days) VALUES($1,'PVQ'||($1::bigint)::text,$2,$3,$4,'BALANÇA PVQ',0)`, equip, almox, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE PVQ','11444777000161',$2,$3)`, cliente, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CELULAS LTDA','11222333000181',$2,$3)`, fornecedor, actor, a.ID)
	if err := pool.QueryRow(bg, `INSERT INTO employees(code,name,created_by,enterprise_id) VALUES($1,'Técnico PVQ',$2,$3) RETURNING code`, u%1_000_000_000, actor, a.ID).Scan(&tecnico); err != nil {
		t.Fatal(err)
	}
	var grupo int64
	if err := pool.QueryRow(bg, `INSERT INTO technical_assistance_defect_groups(description,created_by,enterprise_id) VALUES('Elétrico',$1,$2) RETURNING code`, actor, a.ID).Scan(&grupo); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO technical_assistance_defect_reasons(group_code,description,created_by,enterprise_id) VALUES($1,'Célula de carga queimada',$2,$3) RETURNING code`, grupo, actor, a.ID).Scan(&motivo); err != nil {
		t.Fatal(err)
	}

	items := item_.NewRepositoryItemSQLC(sqlc.New(pool))
	su := &serie_uc.UseCase{Repo: seriepg.New(pool), Auth: &auth.AuthService{}, Items: items}
	txt := fmt.Sprintf("PVQ%d", equip)
	if _, err := su.SalvarPerfil(a.Ctx, txt, serie_uc.PerfilDTO{ControlaSerie: true, GarantiaMeses: 12, CobrePecas: true, CobreMaoObra: true}); err != nil {
		t.Fatal(err)
	}
	venda := time.Now().AddDate(0, 0, -100).Format("2006-01-02")
	if _, err := su.Cadastrar(a.Ctx, serie_uc.CadastroDTO{ItemCode: request.TextCode(txt), Series: []string{"PVQ-1", "PVQ-2", "PVQ-3", "PVQ-4"}, Situacao: "NO_CLIENTE",
		CustomerCode: &cliente, DataVenda: &venda, Lote: ptr("L-2026-07")}); err != nil {
		t.Fatal(err)
	}
	ta := &technical_assistance_uc.UseCase{Repo: tapg.New(pool), Auth: &auth.AuthService{}, Garantia: su}
	serie := "PVQ-1"
	call, err := ta.CreateCall(a.Ctx, request.CreateTechnicalAssistanceCallDTO{EnterpriseCode: a.Code, CustomerCode: cliente, Subject: "Balança sem leitura", Priority: "NORMAL",
		Items: []request.CreateTechnicalAssistanceCallItemDTO{{ItemCode: equip, SerialNumber: &serie, Quantity: decimal.NewFromInt(1)}}})
	if err != nil {
		t.Fatal(err)
	}
	var itemDoChamado int64
	if err := pool.QueryRow(bg, `SELECT code FROM technical_assistance_call_items WHERE call_code=$1`, call.Code).Scan(&itemDoChamado); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `UPDATE technical_assistance_call_items SET defect_reason_code=$1 WHERE code=$2`, motivo, itemDoChamado)

	osu := &ordem_servico_uc.UseCase{Repo: ospg.New(pool), Auth: &auth.AuthService{}, Items: items}
	if _, err := osu.SalvarParametros(a.Ctx, ordem_servico_uc.ParametrosDTO{CustoHora: d("60"), PrecoHora: d("150"), WarehouseID: &almox}); err != nil {
		t.Fatal(err)
	}
	o, err := osu.Abrir(a.Ctx, ordem_servico_uc.AberturaDTO{CallCode: call.Code, CallItemCode: &itemDoChamado, Local: "CLIENTE", TecnicoCode: &tecnico})
	if err != nil {
		t.Fatal(err)
	}
	hoje := time.Now().Format("2006-01-02")
	for _, step := range []func() error{
		func() error { _, err := osu.Iniciar(a.Ctx, o.ID, nil); return err },
		func() error {
			_, err := osu.AddHora(a.Ctx, o.ID, ordem_servico_uc.HoraDTO{Data: hoje, Horas: d("3")})
			return err
		},
		func() error {
			_, err := osu.Concluir(a.Ctx, o.ID, ordem_servico_uc.ConclusaoDTO{DefeitoConstatado: "célula queimada", Solucao: "célula trocada"})
			return err
		},
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}

	// Indicadores: 3h × 60 = 180 de garantia no item, no defeito e no lote.
	ind := &indicadores_pv_uc.UseCase{Repo: indpg.New(pool)}
	p, err := ind.Painel(a.Ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Resumo.Chamados != 1 || p.Resumo.OSConcluidas != 1 || !p.Resumo.CustoGarantia.Equal(d("180")) || !p.Resumo.Cobrado.IsZero() {
		t.Fatalf("resumo: %+v", p.Resumo)
	}
	if len(p.PorItem) != 1 || p.PorItem[0].Chave != txt || !p.PorItem[0].CustoGarantia.Equal(d("180")) {
		t.Fatalf("por item: %+v", p.PorItem)
	}
	if len(p.PorDefeito) != 1 || p.PorDefeito[0].Descricao != "Célula de carga queimada" {
		t.Fatalf("por defeito: %+v", p.PorDefeito)
	}
	if len(p.PorLote) != 1 || p.PorLote[0].Descricao != "L-2026-07" {
		t.Fatalf("por lote: %+v", p.PorLote)
	}
	// 4 séries em campo há ~100 dias, 1 falhou: taxa 25%, MTBF ≈ 400 dias.
	if len(p.Confiabilidade) != 1 {
		t.Fatalf("confiabilidade: %+v", p.Confiabilidade)
	}
	cf := p.Confiabilidade[0]
	if cf.EmCampo != 4 || cf.Falharam != 1 || cf.FalhasTotais != 1 || cf.TaxaFalha == nil || !cf.TaxaFalha.Equal(d("25")) || cf.MTBFDias == nil ||
		cf.MTBFDias.LessThan(d("396")) || cf.MTBFDias.GreaterThan(d("404")) {
		t.Fatalf("confiabilidade: %+v taxa %v mtbf %v", cf, cf.TaxaFalha, cf.MTBFDias)
	}
	if pb, _ := ind.Painel(b.Ctx, "", ""); pb.Resumo.Chamados != 0 || len(pb.PorItem) != 0 || len(pb.Confiabilidade) != 0 {
		t.Fatalf("a outra empresa não vê os indicadores: %+v", pb.Resumo)
	}

	// Chamado → CAPA de reclamação do cliente, uma só.
	rec := &recuperacao_uc.UseCase{Repo: recpg.New(pool), Auth: &auth.AuthService{}, CAPA: &capa_uc.UseCase{Repo: capapg.New(pool), Auth: &auth.AuthService{}}}
	if _, err := rec.AbrirCAPA(b.Ctx, call.Code, recuperacao_uc.CAPADTO{Metodo: "SIMPLES"}); err == nil {
		t.Fatal("a outra empresa não abre CAPA pelo chamado alheio")
	}
	capa, err := rec.AbrirCAPA(a.Ctx, call.Code, recuperacao_uc.CAPADTO{Metodo: "SIMPLES", Responsavel: lider.String()})
	if err != nil {
		t.Fatal(err)
	}
	if capa.Origem != "RECLAMACAO_CLIENTE" || capa.CustomerCode == nil || *capa.CustomerCode != cliente || capa.ItemCode == nil || *capa.ItemCode != equip {
		t.Fatalf("CAPA herda cliente e item do chamado: %+v", capa.Capa)
	}
	if _, err := rec.AbrirCAPA(a.Ctx, call.Code, recuperacao_uc.CAPADTO{Responsavel: lider.String()}); err == nil {
		t.Fatal("o chamado só tem uma CAPA")
	}

	// Recuperação: pela OS, herda chamado, item e CAPA; recupera 120 dos 180.
	if _, err := rec.Abrir(b.Ctx, recuperacao_uc.AberturaDTO{SupplierCode: fornecedor, Motivo: "x", ValorReclamado: d("10")}); err == nil {
		t.Fatal("a outra empresa não usa o fornecedor desta")
	}
	r, err := rec.Abrir(a.Ctx, recuperacao_uc.AberturaDTO{SupplierCode: fornecedor, OSID: &o.ID, Motivo: "Célula de carga com defeito de fabricação", ValorReclamado: d("180")})
	if err != nil {
		t.Fatal(err)
	}
	if r.CallCode == nil || *r.CallCode != call.Code || r.ItemCode == nil || *r.ItemCode != equip || r.CapaID == nil || *r.CapaID != capa.ID || r.Situacao != "RECLAMADA" {
		t.Fatalf("reclamação: %+v", r)
	}
	if _, err := rec.Mudar(a.Ctx, r.ID, recuperacao_uc.MudancaDTO{Situacao: "RECUPERADA", ValorRecuperado: d("200"), Forma: ptr("nota_credito"), RecuperadoEm: hoje}); err == nil {
		t.Fatal("não recupera mais que o reclamado")
	}
	if _, err := rec.Mudar(b.Ctx, r.ID, recuperacao_uc.MudancaDTO{Situacao: "ACEITA"}); err == nil {
		t.Fatal("a outra empresa não muda a reclamação")
	}
	if _, err := rec.Mudar(a.Ctx, r.ID, recuperacao_uc.MudancaDTO{Situacao: "ACEITA"}); err != nil {
		t.Fatal(err)
	}
	r, err = rec.Mudar(a.Ctx, r.ID, recuperacao_uc.MudancaDTO{Situacao: "RECUPERADA", ValorRecuperado: d("120"), Forma: ptr("nota_credito"), Documento: ptr("NC 5521"), RecuperadoEm: hoje})
	if err != nil || r.Situacao != "RECUPERADA" || r.Forma == nil || *r.Forma != "NOTA_CREDITO" {
		t.Fatalf("recuperada: %+v %v", r, err)
	}
	if l, _ := rec.Listar(b.Ctx, recdom.Filtro{}); len(l) != 0 {
		t.Fatal("a outra empresa não lista as reclamações")
	}
	p, _ = ind.Painel(a.Ctx, "", "")
	if !p.Resumo.Reclamado.Equal(d("180")) || !p.Resumo.Recuperado.Equal(d("120")) || !p.Resumo.CustoLiquido.Equal(d("60")) || p.Resumo.ChamadosComCAPA != 1 {
		t.Fatalf("resumo com recuperação: %+v", p.Resumo)
	}
}

func ptr(s string) *string { return &s }

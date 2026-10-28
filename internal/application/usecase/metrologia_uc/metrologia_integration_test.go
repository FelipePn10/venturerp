//go:build integration

package metrologia_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/lote_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/metrologia_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/metrologia"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	amostragempg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/amostragem"
	lotepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/lote"
	metrologiapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/metrologia"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func dp(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}
func ip(v int64) *int64 { return &v }
func bp(v bool) *bool   { return &v }

func TestMetrologiaBloqueiaMedicaoEApontaImpacto(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	item := 94_000_000 + u%1_000_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`ALTER TABLE quality_calibrations DISABLE TRIGGER trg_quality_calibrations_imutaveis`,
				`DELETE FROM quality_inspection_lots WHERE enterprise_id=$1`,
				`DELETE FROM quality_instruments WHERE enterprise_id=$1`,
				`ALTER TABLE quality_calibrations ENABLE TRIGGER trg_quality_calibrations_imutaveis`,
				`DELETE FROM receiving_inspection_routes WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			} {
				if strings.HasPrefix(q, "ALTER") {
					_, _ = pool.Exec(bg, q)
				} else if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("MET-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,($1::bigint)::text,$2,$3,$4,'EIXO')`, item, almox, actor, a.ID)
	var rota int64
	if err := pool.QueryRow(bg, `INSERT INTO receiving_inspection_routes(enterprise_code,basis,item_code,mask,inspection_warehouse_id,valid_from,is_active,created_by)
		VALUES($1,'ITEM',$2,'',$3,$4::date,TRUE,$5) RETURNING id`, a.Code, item, almox, time.Now().AddDate(0, 0, -30), actor).Scan(&rota); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO receiving_inspection_route_steps(route_id,sequence,inspection_name,kind,appointment_mode,is_required,min_value,max_value,instrument_group)
		VALUES($1,1,'Diâmetro','VALUE','ALL_MEASUREMENTS',TRUE,19.9,20.1,'Paquímetro')`, rota)

	met := metrologiapg.New(pool)
	muc := &metrologia_uc.UseCase{Repo: met, Auth: &auth.AuthService{}}
	luc := &lote_uc.UseCase{Repo: lotepg.New(pool), Fontes: lotepg.NewFontes(pool), Amostragem: amostragempg.New(pool), Auth: &auth.AuthService{}, Metrologia: met}
	ctx := a.Ctx
	hoje := time.Now()

	abrirLote := func() *lote_uc.LoteResponse {
		t.Helper()
		l, err := luc.Abrir(ctx, lote_uc.AberturaDTO{Origem: "MANUAL", ItemCode: item, Quantidade: decimal.NewFromInt(5)})
		if err != nil || len(l.Caracteristicas) != 1 || l.Caracteristicas[0].Instrumento == nil {
			t.Fatalf("lote: %+v %v", l, err)
		}
		return l
	}
	medir := func(l *lote_uc.LoteResponse, unidade int, instr *int64) error {
		_, err := luc.GravarResultados(ctx, l.ID, []lote_uc.ResultadoDTO{{CaracteristicaID: l.Caracteristicas[0].ID, AmostraIndice: unidade, Valor: dp("20.0"), Conforme: bp(true), InstrumentoID: instr}})
		return err
	}

	// 1. Empresa sem instrumentos do grupo: mede sem informar (implantação gradual).
	l1 := abrirLote()
	if err := medir(l1, 1, nil); err != nil {
		t.Fatalf("sem metrologia implantada a medição passa: %v", err)
	}

	// 2. Cadastro e calibração aprovada há 10 dias.
	pq, err := muc.Salvar(ctx, 0, metrologia_uc.InstrumentoDTO{Codigo: "pq-01", Descricao: "Paquímetro 150", Grupo: "paquimetro", FrequenciaDias: 180, EMA: dp("0.03")})
	if err != nil || pq.Grupo != "PAQUIMETRO" || pq.Codigo != "PQ-01" {
		t.Fatalf("cadastro: %+v %v", pq, err)
	}
	mc, err := muc.Salvar(ctx, 0, metrologia_uc.InstrumentoDTO{Codigo: "MC-01", Descricao: "Micrômetro", Grupo: "MICROMETRO", FrequenciaDias: 365, EMA: dp("0.004")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := muc.Salvar(ctx, 0, metrologia_uc.InstrumentoDTO{Codigo: "PQ-01", Descricao: "dup", Grupo: "X", FrequenciaDias: 1}); err == nil {
		t.Fatal("código duplicado")
	}
	if _, err := muc.Obter(b.Ctx, pq.ID); err == nil {
		t.Fatal("B leu o instrumento de A")
	}
	d10 := hoje.AddDate(0, 0, -10).Format("2006-01-02")
	for _, id := range []int64{pq.ID, mc.ID} {
		if _, err := muc.RegistrarCalibracao(ctx, id, metrologia_uc.CalibracaoDTO{Data: d10, Executor: "INTERNO", ErroEncontrado: dp("0.001"), Incerteza: dp("0.001")}); err != nil {
			t.Fatal(err)
		}
	}

	// 3. Agora o grupo tem instrumento: medir sem informar é recusado; outro grupo também.
	l2 := abrirLote()
	if err := medir(l2, 1, nil); err == nil || !strings.Contains(err.Error(), "informe o instrumento") {
		t.Fatalf("sem instrumento: %v", err)
	}
	if err := medir(l2, 1, &mc.ID); err == nil || !strings.Contains(err.Error(), "grupo") {
		t.Fatalf("instrumento de outro grupo: %v", err)
	}
	if err := medir(l2, 1, &pq.ID); err != nil {
		t.Fatalf("paquímetro calibrado: %v", err)
	}
	if err := medir(l2, 2, &pq.ID); err != nil {
		t.Fatal(err)
	}
	if l, _ := luc.Obter(ctx, l2.ID); l.Resultados[0].InstrumentoID == nil || *l.Resultados[0].InstrumentoID != pq.ID {
		t.Fatalf("o resultado guarda o instrumento: %+v", l.Resultados)
	}

	// 4. Calibração reprovada: bloqueia e aponta o lote medido desde a última aprovada.
	r, err := muc.RegistrarCalibracao(ctx, pq.ID, metrologia_uc.CalibracaoDTO{Data: hoje.Format("2006-01-02"), Executor: "EXTERNO", Certificado: ip2s("RBC-77"),
		ErroEncontrado: dp("0.05"), Incerteza: dp("0.005")})
	if err != nil {
		t.Fatal(err)
	}
	if r.Calibracao.Resultado != "REPROVADO" || r.Instrumento.Status != "BLOQUEADO" || len(r.Impacto) != 1 || r.Impacto[0].LoteID != l2.ID || r.Impacto[0].Medicoes != 2 {
		t.Fatalf("reprovação: %+v impacto %+v", r.Instrumento, r.Impacto)
	}
	if err := medir(l2, 3, &pq.ID); err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("instrumento bloqueado: %v", err)
	}
	if bloq, _ := muc.Listar(ctx, metrologia.Filtro{Situacao: "bloqueados"}); len(bloq) != 1 || bloq[0].ID != pq.ID {
		t.Fatalf("lista de bloqueados: %+v", bloq)
	}

	// 5. Vencido: frequência de 1 dia calibrado há 10.
	ve, err := muc.Salvar(ctx, 0, metrologia_uc.InstrumentoDTO{Codigo: "PQ-02", Descricao: "Paquímetro 300", Grupo: "PAQUIMETRO", FrequenciaDias: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := muc.RegistrarCalibracao(ctx, ve.ID, metrologia_uc.CalibracaoDTO{Data: d10, Executor: "INTERNO", Resultado: "APROVADO"}); err != nil {
		t.Fatal(err)
	}
	if err := medir(l2, 3, &ve.ID); err == nil || !strings.Contains(err.Error(), "venceu") {
		t.Fatalf("instrumento vencido: %v", err)
	}
	if venc, _ := muc.Listar(ctx, metrologia.Filtro{Situacao: "VENCIDOS"}); len(venc) != 1 || venc[0].ID != ve.ID {
		t.Fatalf("lista de vencidos: %+v", venc)
	}

	// 6. Outra empresa não usa o instrumento de A; calibração é imutável.
	if _, err := muc.RegistrarCalibracao(b.Ctx, mc.ID, metrologia_uc.CalibracaoDTO{Data: d10, Executor: "INTERNO", Resultado: "APROVADO"}); err == nil {
		t.Fatal("B calibrou o instrumento de A")
	}
	if _, err := pool.Exec(bg, `UPDATE quality_calibrations SET resultado='APROVADO' WHERE instrument_id=$1`, pq.ID); err == nil {
		t.Fatal("a calibração registrada deveria ser imutável")
	}
	if _, err := muc.Baixar(ctx, ve.ID, "quebrado"); err != nil {
		t.Fatal(err)
	}
	if todos, _ := muc.Listar(ctx, metrologia.Filtro{}); len(todos) != 2 {
		t.Fatalf("o baixado sai da lista padrão: %d", len(todos))
	}
	_ = ip
}

func ip2s(s string) *string { return &s }

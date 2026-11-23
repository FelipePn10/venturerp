//go:build integration

package production_order_test

import (
	"context"
	"os"
	"testing"
	"time"

	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestFichaDaOrdemNaoAtrasaAData: em produção o banco fala UTC e a API roda em
// São Paulo. Data pura convertida para timestamptz virava 21h do dia anterior e
// a ficha imprimia o início previsto um dia antes da emissão (OF 3 da Tecnofer).
func TestFichaDaOrdemNaoAtrasaAData(t *testing.T) {
	sp, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skip("sem tzdata:", err)
	}
	antes := time.Local
	time.Local = sp
	t.Cleanup(func() { time.Local = antes })

	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, nil)
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, item)
	var id int64
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,end_date,created_by,enterprise_id)
		VALUES($1,$2,1,'OPEN','2026-11-23','2026-11-24',$3,$4) RETURNING id`, testutil.UniqueCode()%1_000_000, item, actor, a.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM production_orders WHERE id = $1`, id) })

	// pool com a sessão em UTC, como o banco de produção
	cfg, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "UTC"
	utc, err := pgxpool.NewWithConfig(base, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer utc.Close()

	f, err := productionrepo.NewProductionOrderRepositoryPGX(utc).FichaDaOrdem(a.Ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if f.Inicio == nil || f.Inicio.Format("02/01/2006") != "23/11/2026" {
		t.Fatalf("início previsto = %v, quer 23/11/2026", f.Inicio)
	}
	if f.Fim == nil || f.Fim.Format("02/01/2006") != "24/11/2026" {
		t.Fatalf("entrega prevista = %v, quer 24/11/2026", f.Fim)
	}
}

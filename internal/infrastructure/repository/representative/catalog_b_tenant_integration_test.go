//go:build integration

package representative_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	csentity "github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	taentity "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/consumer_service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/representative"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestCodeKeyedCatalogsAreIsolatedByTenant: tipos de representante, tipos de
// chamado do SAC e grupos/motivos de defeito eram do banco inteiro. Agora são
// da empresa, e o banco recusa registro apontando para o de outra empresa.
func TestCodeKeyedCatalogsAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := uuid.UUID(testutil.Actor(t, pool))
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids, codes := []int64{a.ID, b.ID}, []int64{a.Code, b.Code}
		_, _ = pool.Exec(base, `DELETE FROM representatives WHERE enterprise_code = ANY($1)`, codes)
		_, _ = pool.Exec(base, `DELETE FROM representative_types WHERE enterprise_id = ANY($1)`, ids)
		_, _ = pool.Exec(base, `DELETE FROM consumer_service_call_types WHERE enterprise_id = ANY($1)`, ids)
		_, _ = pool.Exec(base, `DELETE FROM technical_assistance_defect_reasons WHERE enterprise_id = ANY($1)`, ids)
		_, _ = pool.Exec(base, `DELETE FROM technical_assistance_defect_groups WHERE enterprise_id = ANY($1)`, ids)
	})

	reps := representative.New(pool)
	tipo, err := reps.CreateType(a.Ctx, &entity.RepresentativeType{Description: "TIPO A", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reps.GetType(b.Ctx, tipo.Code); err == nil {
		t.Fatal("empresa B leu o tipo de representante da A")
	}
	if l, err := reps.ListTypes(b.Ctx, false); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d tipo(s) da A (%v)", len(l), err)
	}
	if _, err := reps.Create(b.Ctx, &entity.Representative{Name: "REP B", DocumentNumber: "12345678909", TypeCode: &tipo.Code, IsActive: true, RegisterDate: time.Now()}); err == nil {
		t.Fatal("o representante da B foi gravado com o tipo da A")
	}
	if _, err := reps.Create(a.Ctx, &entity.Representative{Name: "REP A", DocumentNumber: "12345678909", TypeCode: &tipo.Code, IsActive: true, RegisterDate: time.Now()}); err != nil {
		t.Fatalf("o representante da A com o próprio tipo foi recusado: %v", err)
	}

	sac := consumer_service.New(pool)
	ct, err := sac.CreateCallType(a.Ctx, &csentity.CallType{Description: "RECLAMACAO A", IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sac.GetCallType(b.Ctx, ct.Code); err == nil {
		t.Fatal("empresa B leu o tipo de chamado da A")
	}
	if l, err := sac.ListCallTypes(b.Ctx, false); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d tipo(s) de chamado da A (%v)", len(l), err)
	}

	ta := technical_assistance.New(pool)
	grupo, err := ta.CreateDefectGroup(a.Ctx, &taentity.DefectGroup{Description: "GRUPO A", IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if l, err := ta.ListDefectGroups(b.Ctx, false); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d grupo(s) de defeito da A (%v)", len(l), err)
	}
	if _, err := ta.CreateDefectReason(b.Ctx, &taentity.DefectReason{GroupCode: grupo.Code, Description: "MOTIVO B", CreatedBy: actor}); err == nil {
		t.Fatal("empresa B criou motivo no grupo de defeito da A")
	}
	motivo, err := ta.CreateDefectReason(a.Ctx, &taentity.DefectReason{GroupCode: grupo.Code, Description: "MOTIVO A", CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ta.GetDefectReason(b.Ctx, motivo.Code); err == nil {
		t.Fatal("empresa B leu o motivo de defeito da A")
	}
}

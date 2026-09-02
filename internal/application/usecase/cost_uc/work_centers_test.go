package cost_uc

import (
	"context"
	"testing"

	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
)

type fakeWorkCenterReader struct {
	search string
	limit  int
	offset int
}

func (r *fakeWorkCenterReader) ListActiveWorkCenterTypes(_ context.Context, search string, limit, offset int) ([]*machineentity.MachineType, int64, error) {
	r.search, r.limit, r.offset = search, limit, offset
	description := "Corte de chapas"
	return []*machineentity.MachineType{{ID: 1, Code: 20, Name: "Laser", Description: &description, IsActive: true}}, 1, nil
}

func TestListWorkCentersPreservesSearchAndPaginationContract(t *testing.T) {
	reader := &fakeWorkCenterReader{}
	page, err := (&StandardCostUseCase{}).WithWorkCenters(reader).ListWorkCenters(context.Background(), "laser", 25, 5)
	if err != nil {
		t.Fatal(err)
	}
	if reader.search != "laser" || reader.limit != 25 || reader.offset != 5 || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("contrato inesperado: reader=%+v page=%+v", reader, page)
	}
}

func TestListWorkCentersRejectsInvalidPagination(t *testing.T) {
	_, err := (&StandardCostUseCase{}).WithWorkCenters(&fakeWorkCenterReader{}).ListWorkCenters(context.Background(), "", 501, 0)
	if err == nil {
		t.Fatal("esperava erro para limit acima de 500")
	}
}

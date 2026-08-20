package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

type fiscalInheritanceAuth struct{ ports.AuthService }

func (fiscalInheritanceAuth) CanCreateItem(context.Context) bool          { return true }
func (fiscalInheritanceAuth) EnterpriseID(context.Context) (int64, error) { return 41, nil }
func (fiscalInheritanceAuth) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000041"), nil
}

type fiscalInheritanceRepo struct {
	repository.ItemRepository
	master bool
}

func (r fiscalInheritanceRepo) Create(_ context.Context, item *entity.Item) (*entity.Item, error) {
	item.ID, item.Code = 9001, valueobject.ItemCode(9001)
	value, source := r.master, entity.FiscalSourceInherited
	if item.Accounting.CalculatePISCOFINS != nil {
		value, source = *item.Accounting.CalculatePISCOFINS, entity.FiscalSourceOverride
	}
	context := func(code int64) *entity.EffectiveFiscalContext {
		v := value
		return &entity.EffectiveFiscalContext{ClassificationID: code, ClassificationCode: code, CalculatePISCOFINS: &v, Sources: map[string]entity.FiscalValueSource{"calculate_pis_cofins": source}}
	}
	item.FiscalEffective.Purchase, item.FiscalEffective.Sale = context(101), context(102)
	return item, nil
}

func TestCreateItemHTTPPreservesPISCOFINSInheritanceAndOverrides(t *testing.T) {
	cases := []struct {
		name, jsonField, expectedSource string
		master, expected                bool
		expectRawOverride               bool
	}{
		{"herda verdadeiro", "", "HERDADO", true, true, false},
		{"herda falso", "", "HERDADO", false, false, false},
		{"sobrescreve verdadeiro", `,"calculate_pis_cofins":true`, "SOBRESCRITO", false, true, true},
		{"sobrescreve falso", `,"calculate_pis_cofins":false`, "SOBRESCRITO", true, false, true},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := fiscalInheritanceRepo{master: tc.master}
			uc := item_uc.NewCreateItemUseCase(repo, fiscalInheritanceAuth{})
			h := NewCreateItemHandler(uc, nil, nil, nil, nil)
			body := fmt.Sprintf(`{
                "code":"FISC-%d","name":"Item fiscal de teste","nature":0,
                "pdm":{"group_code":0,"modifier_code":0,"attributes":[],"description_technique":"Item fiscal"},
                "situation":"LINHA","health":"ATIVO",
                "warehouse":{"warehouse_code":1,"unit_of_measurement":"UN","automatic_low":false,"minimum_stock":0},
                "engineering":{"weight":{"gross":0,"net":0,"unit":"KG"},"type":"FABRICADO","type_struct":"INDUSTRIAL","oem":false},
                "planning":{"type_mrp":"NORMAL_MRP","llc":1,"ghost":false,"minimum_lot":1,"multiple_lot":1,"safety_stock":0,"critical":false,"exclusive":false,"active":true},
                "supplies":{"type_of_use":"INDUSTRIALIZACAO","receiving_checklist":false,"harvest":false},
                "accounting":{"purchase_fiscal_classification_code":"101","sale_fiscal_classification_code":"102"%s},
                "created_by":"00000000-0000-0000-0000-000000000041"
            }`, i+1, tc.jsonField)
			req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(body))
			rec := httptest.NewRecorder()
			h.CreateItem(rec, req)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var envelope struct {
				Data response.ItemResponse `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Data.Code != fmt.Sprintf("FISC-%d", i+1) {
				t.Fatalf("código comercial não foi preservado como string: %q", envelope.Data.Code)
			}
			for name, effective := range map[string]*response.EffectiveFiscalContextResponse{"purchase": envelope.Data.FiscalEffective.Purchase, "sale": envelope.Data.FiscalEffective.Sale} {
				if effective == nil || effective.CalculatePISCOFINS == nil || *effective.CalculatePISCOFINS != tc.expected || effective.Sources["calculate_pis_cofins"] != tc.expectedSource {
					t.Errorf("%s effective=%+v", name, effective)
				}
			}
			if tc.expectRawOverride != (envelope.Data.Accounting.CalculatePISCOFINS != nil) {
				t.Errorf("raw override=%v, expected presence=%v", envelope.Data.Accounting.CalculatePISCOFINS, tc.expectRawOverride)
			}
		})
	}
}

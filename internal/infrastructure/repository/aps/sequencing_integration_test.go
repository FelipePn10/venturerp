//go:build integration

package aps_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	apsdomain "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	apsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestSequencingConfigurationIsTenantIsolated(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var first int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&first); err != nil {
		t.Fatal(err)
	}
	var code int64
	if err := pool.QueryRow(base, "SELECT COALESCE(MAX(code),0)+1 FROM enterprise").Scan(&code); err != nil {
		t.Fatal(err)
	}
	var second int64
	if err := pool.QueryRow(base, "INSERT INTO enterprise(code,name) VALUES($1,'APS tenant test') RETURNING id", code).Scan(&second); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(base, "DELETE FROM enterprise WHERE id=$1", second) })
	ctx1 := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: first})
	ctx2 := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: second})
	repo := apsrepo.New(sqlc.New(pool), pool).(apsdomain.ConfigurationRepository)
	selection := apsrepo.New(sqlc.New(pool), pool).(apsdomain.SelectionRepository)
	group, err := repo.UpsertResourceGroup(ctx1, "LASER", "Célula laser")
	if err != nil {
		t.Fatal(err)
	}
	calendar, err := repo.UpsertMachineCalendar(ctx1, 9001, "Turno normal", []apsdomain.MachineCalendarInterval{{Weekday: 1, Start: "07:00", End: "17:00"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(base, "DELETE FROM machine_calendars WHERE id=$1", calendar.ID)
		pool.Exec(base, "DELETE FROM production_resource_groups WHERE id=$1", group.ID)
	})
	groups1, err := repo.ListResourceGroups(ctx1)
	if err != nil || len(groups1) == 0 {
		t.Fatalf("owner groups=%+v err=%v", groups1, err)
	}
	groups2, err := repo.ListResourceGroups(ctx2)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range groups2 {
		if v.ID == group.ID {
			t.Fatal("resource group leaked across tenants")
		}
	}
	calendars, err := repo.ListMachineCalendars(ctx1)
	if err != nil || len(calendars) == 0 || len(calendars[len(calendars)-1].Intervals) == 0 {
		t.Fatalf("calendars=%+v err=%v", calendars, err)
	}
	if _, err := selection.ListAvailabilityWindows(ctx1, 0, nil, time.Now(), time.Now().AddDate(0, 0, 7)); err != nil {
		t.Fatalf("availability query: %v", err)
	}
}

func TestSequencingIndustrialVolume(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := apsrepo.New(sqlc.New(pool), pool).(apsdomain.ConfigurationRepository)
	rowsCount := 2000
	if raw := os.Getenv("SEQUENCING_VOLUME_ROWS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			rowsCount = parsed
		}
	}
	prefix := fmt.Sprintf("VOL-%d-", time.Now().UnixNano())
	t.Cleanup(func() {
		pool.Exec(base, "DELETE FROM production_resource_groups WHERE enterprise_id=$1 AND code LIKE $2", enterpriseID, prefix+"%")
	})
	for i := 0; i < rowsCount; i++ {
		if _, err := repo.UpsertResourceGroup(ctx, fmt.Sprintf("%s%06d", prefix, i), "Volume"); err != nil {
			t.Fatal(err)
		}
	}
	started := time.Now()
	groups, err := repo.ListResourceGroups(ctx)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, g := range groups {
		if strings.HasPrefix(g.Code, prefix) {
			found++
		}
	}
	if found != rowsCount {
		t.Fatalf("found=%d want=%d", found, rowsCount)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("volume query too slow: %v", elapsed)
	}
	t.Logf("sequencing-volume rows=%d elapsed=%v", rowsCount, elapsed)
}

func TestSequencingSubregistriesConcurrencyAndDowntime(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := apsrepo.New(sqlc.New(pool), pool).(apsdomain.ConfigurationRepository)
	uid := uuid.New()
	mtCode := testutil.UniqueCode()
	var mtID int64
	if err := pool.QueryRow(ctx, `INSERT INTO machine_types(code,name,type,created_by,enterprise_id) SELECT $1,'APS WC',type,$2,$3 FROM machine_types LIMIT 1 RETURNING id`, mtCode, uid, enterpriseID).Scan(&mtID); err != nil {
		t.Fatal(err)
	}
	machineCode := testutil.UniqueCode()
	var machineID int64
	if err := pool.QueryRow(ctx, `INSERT INTO machines(code,name,machine_type_code,capacity,capacity_unit,capacity_period,efficiency_rate,created_by,enterprise_id) VALUES($1,'APS Machine',$2,8,'UN','HORA',1,$3,$4) RETURNING id`, machineCode, mtCode, uid, enterpriseID).Scan(&machineID); err != nil {
		t.Fatal(err)
	}
	employeeCode := testutil.UniqueCode()
	var employeeID int64
	if err := pool.QueryRow(ctx, `INSERT INTO employees(code,name,created_by,enterprise_id) VALUES($1,'APS Planner',$2,$3) RETURNING id`, employeeCode, uid, enterpriseID).Scan(&employeeID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(base, "DELETE FROM employees WHERE id=$1", employeeID)
		pool.Exec(base, "DELETE FROM machines WHERE id=$1", machineID)
		pool.Exec(base, "DELETE FROM machine_types WHERE id=$1", mtID)
	})
	if err := repo.UpsertEmployeeSequencingProfile(ctx, employeeID, apsdomain.EmployeeSequencingProfile{Contacts: []apsdomain.EmployeeContact{{ContactType: "EMAIL", Value: "planner@test.local", IsPrimary: true}}, Functions: []apsdomain.EmployeeFunction{{FunctionName: "PLANNER"}}, CreditLimit: "100.00"}); err != nil {
		t.Fatal(err)
	}
	serviceCode := fmt.Sprintf("MEC-%d", machineCode)
	fieldName := fmt.Sprintf("RPM-%d", machineCode)
	if err := repo.UpsertMachineIndustrialProfile(ctx, machineID, apsdomain.MachineIndustrialProfile{PreparationTime: "15", PreparationTimeUnit: "MINUTE", Services: []apsdomain.MachineService{{ServiceCode: serviceCode, Description: "Mecânica", ServiceType: "MECHANICAL", FrequencyValue: 30, FrequencyUnit: "DAY", ImplementedOn: time.Now(), Items: []apsdomain.ServiceItem{{ItemCode: 10, Quantity: "1.5"}}, ResponsibleEmployeeIDs: []int64{employeeID}}}, SpecialValues: []apsdomain.SpecialValue{{Name: fieldName, ValueType: "NUMBER", NumericValue: "1800"}}}); err != nil {
		t.Fatal(err)
	}
	employeeProfile, err := repo.GetEmployeeSequencingProfile(ctx, employeeID)
	if err != nil || len(employeeProfile.Contacts) != 1 || len(employeeProfile.Functions) != 1 {
		t.Fatalf("employee profile=%+v err=%v", employeeProfile, err)
	}
	contactID, functionID := employeeProfile.Contacts[0].ID, employeeProfile.Functions[0].ID
	if err = repo.UpdateEmployeeContact(ctx, employeeID, contactID, apsdomain.EmployeeContact{ContactType: "PHONE", Value: "+5511999999999", IsPrimary: true}); err != nil {
		t.Fatal(err)
	}
	if err = repo.UpdateEmployeeFunction(ctx, employeeID, functionID, apsdomain.EmployeeFunction{FunctionName: "MASTER PLANNER", IsManager: true}); err != nil {
		t.Fatal(err)
	}
	if err = repo.UpdateEmployeeContact(ctx, employeeID+999999, contactID, apsdomain.EmployeeContact{ContactType: "EMAIL", Value: "leak@test.local"}); err == nil {
		t.Fatal("contact update with wrong parent must fail")
	}
	machineProfile, err := repo.GetMachineIndustrialProfile(ctx, machineID)
	if err != nil || len(machineProfile.Services) != 1 || len(machineProfile.Services[0].Items) != 1 || len(machineProfile.SpecialValues) != 1 {
		t.Fatalf("machine profile=%+v err=%v", machineProfile, err)
	}
	serviceID, itemID, fieldID := machineProfile.Services[0].ID, machineProfile.Services[0].Items[0].ID, machineProfile.SpecialValues[0].FieldID
	service := machineProfile.Services[0]
	service.Description = "Mecânica revisada"
	service.FrequencyValue = 15
	if err = repo.UpdateMachineService(ctx, machineID, serviceID, service); err != nil {
		t.Fatal(err)
	}
	if err = repo.UpdateMachineServiceItem(ctx, machineID, serviceID, itemID, apsdomain.ServiceItem{ItemCode: 11, Quantity: "2.25", Notes: "estoque mínimo"}); err != nil {
		t.Fatal(err)
	}
	if err = repo.UpdateMachineSpecialValue(ctx, machineID, fieldID, apsdomain.SpecialValue{Name: fieldName + "-NOMINAL", ValueType: "NUMBER", NumericValue: "1750"}); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteMachineServiceItem(ctx, machineID+999999, serviceID, itemID); err == nil {
		t.Fatal("item delete with wrong machine must fail")
	}
	if err = repo.DeleteEmployeeContact(ctx, employeeID, contactID); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteEmployeeFunction(ctx, employeeID, functionID); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteMachineServiceItem(ctx, machineID, serviceID, itemID); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteMachineSpecialValue(ctx, machineID, fieldID); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteMachineService(ctx, machineID, serviceID); err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Truncate(time.Minute)
	down, err := repo.CreateMachineDowntime(ctx, apsdomain.MachineDowntime{MachineID: machineID, StartsAt: start, EndsAt: start.Add(time.Hour), DowntimeType: "UNPLANNED", Reason: "Teste"})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ListMachineDowntimes(ctx, machineID, start.Add(-time.Hour), start.Add(2*time.Hour))
	if err != nil || len(rows) != 1 {
		t.Fatalf("downtimes=%+v err=%v", rows, err)
	}
	if err := repo.DeleteMachineDowntime(ctx, down.ID); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := repo.UpsertResourceGroup(ctx, "CONCURRENT", "Grupo concorrente")
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	groups, err := repo.ListResourceGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, g := range groups {
		if g.Code == "CONCURRENT" {
			count++
			_ = repo.DeleteResourceGroup(ctx, g.ID)
		}
	}
	if count != 1 {
		t.Fatalf("concurrent group count=%d", count)
	}
}

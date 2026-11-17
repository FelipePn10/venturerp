//go:build integration

package shipping_carrier_uc_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipping_carrier_uc"
	carrierpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipping_carrier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/shopspring/decimal"
)

func ptr[T any](v T) *T { return &v }

// Duas empresas (códigos ≠ ids): cadastro de apoio, uso e exclusão da
// transportadora não atravessam a empresa, e a carga aberta trava inativação,
// exclusão e remoção do veículo.
func TestTransportadoraCompletaDuasEmpresas(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	actor := testutil.Actor(t, pool)
	uc := &shipping_carrier_uc.UseCase{Repo: carrierpg.New(pool)}

	fornecedor := testutil.UniqueCode()
	if _, err := pool.Exec(a.Ctx, `INSERT INTO suppliers(code,name,document_number,is_active,created_by,enterprise_id)
		VALUES($1,'Transportes A',$2,TRUE,$3,$4)`, fornecedor, strconv.FormatInt(fornecedor, 10), actor, a.ID); err != nil {
		t.Fatal(err)
	}

	// Apoio na empresa A.
	tipoA, err := uc.SalvarTipoVeiculo(a.Ctx, 0, &request.SalvarTipoVeiculoDTO{Code: "truck", Description: "Truck baú",
		WheelType: ptr("01"), BodyType: ptr("02"), Axles: ptr(int16(3))})
	if err != nil {
		t.Fatal(err)
	}
	tipoB, err := uc.SalvarTipoVeiculo(b.Ctx, 0, &request.SalvarTipoVeiculoDTO{Code: "TRUCK", Description: "Truck da B"})
	if err != nil {
		t.Fatalf("o mesmo código em outra empresa deveria ser aceito: %v", err)
	}
	if _, err := uc.SalvarTipoOcorrencia(a.Ctx, 0, &request.SalvarTipoOcorrenciaDTO{Code: "chuva forte", Description: "Atraso por chuva", CountsAsFailure: ptr(false)}); err != nil {
		t.Fatal(err)
	}
	if lista, _ := uc.ListarTiposVeiculo(b.Ctx); len(lista) != 1 || lista[0].ID != tipoB.ID {
		t.Fatalf("B viu tipos de veículo de A: %+v", lista)
	}
	if lista, _ := uc.ListarTiposOcorrencia(b.Ctx); len(lista) == 0 {
		t.Fatal("empresa sem tipos de ocorrência deveria receber os padrões")
	} else {
		for _, o := range lista {
			if o.Code == "CHUVA_FORTE" {
				t.Fatal("B viu tipo de ocorrência de A")
			}
		}
	}
	if err := uc.ExcluirTipoVeiculo(b.Ctx, tipoA.ID); err == nil {
		t.Fatal("B apagou tipo de veículo de A")
	}

	// Transportadora de A com veículo do tipo de B: recusado.
	dto := &request.SalvarTransportadoraDTO{SupplierCode: fornecedor, Modal: "AEREO",
		Vehicles: []request.TransportadoraVeiculoDTO{{Plate: "ABC1D23", VehicleTypeID: &tipoB.ID}}}
	if _, err := uc.Salvar(a.Ctx, 0, dto); err == nil {
		t.Fatal("veículo de A com tipo de B foi aceito")
	}
	dto.Vehicles[0].VehicleTypeID = &tipoA.ID
	salva, err := uc.Salvar(a.Ctx, 0, dto)
	if err != nil {
		t.Fatal(err)
	}
	if v := salva.Vehicles[0]; v.WheelType == nil || *v.WheelType != "01" || v.Axles == nil || *v.Axles != 3 {
		t.Fatalf("veículo não herdou o padrão do tipo: %+v", v)
	}
	if err := uc.ExcluirTipoVeiculo(a.Ctx, tipoA.ID); err == nil {
		t.Fatal("tipo de veículo em uso foi apagado")
	}

	// B não lê, não consulta uso, não inativa nem apaga a transportadora de A.
	if _, err := uc.Obter(b.Ctx, salva.ID); err == nil {
		t.Fatal("B leu a transportadora de A")
	}
	if _, err := uc.Uso(b.Ctx, salva.ID); err == nil {
		t.Fatal("B consultou o uso da transportadora de A")
	}
	if err := uc.Excluir(b.Ctx, salva.ID); err == nil {
		t.Fatal("B apagou a transportadora de A")
	}
	if _, err := uc.RegistrarOcorrencia(b.Ctx, salva.ID, &request.RegistrarOcorrenciaDTO{OccurrenceType: "ATRASO"}); err == nil {
		t.Fatal("B registrou ocorrência na transportadora de A")
	}

	// Alteração livre (sem uso) grava os campos novos.
	dto.CubageFactor = ptr(decimalDe("300"))
	dto.AcceptsDangerousGoods = ptr(true)
	dto.Vehicles[0].PlateState = ptr("PR")
	dto.Vehicles[0].DriverCNHExpiry = ptr("2030-01-31")
	if salva, err = uc.Salvar(a.Ctx, salva.ID, dto); err != nil {
		t.Fatal(err)
	}
	if !salva.CubageFactor.Equal(decimalDe("300")) || !salva.AcceptsDangerousGoods || *salva.Vehicles[0].DriverCNHExpiry != "2030-01-31" {
		t.Fatalf("campos novos não gravados: %+v", salva)
	}

	// Desktop antigo (min_client ainda o aceita) grava sem os campos novos: o
	// que está gravado é mantido, em vez de zerado.
	legado := &request.SalvarTransportadoraDTO{SupplierCode: fornecedor, Modal: "AEREO", Notes: ptr("gravado pela tela antiga"),
		Vehicles: []request.TransportadoraVeiculoDTO{{Plate: "ABC-1D23", DriverName: ptr("Novo motorista")}}}
	if salva, err = uc.Salvar(a.Ctx, salva.ID, legado); err != nil {
		t.Fatal(err)
	}
	if v := salva.Vehicles[0]; !salva.CubageFactor.Equal(decimalDe("300")) || !salva.AcceptsDangerousGoods ||
		v.PlateState == nil || *v.PlateState != "PR" || v.DriverCNHExpiry == nil || *v.DriverName != "Novo motorista" {
		t.Fatalf("tela antiga apagou os campos novos: %+v / veículo %+v", salva, v)
	}

	// Ocorrência: tipo precisa existir; o tipo que não conta como falha fica fora das falhas.
	if _, err := uc.RegistrarOcorrencia(a.Ctx, salva.ID, &request.RegistrarOcorrenciaDTO{OccurrenceType: "NAO_EXISTE"}); err == nil {
		t.Fatal("ocorrência com tipo não cadastrado foi aceita")
	}
	for _, tipo := range []string{"CHUVA_FORTE", "AVARIA"} {
		if _, err := uc.RegistrarOcorrencia(a.Ctx, salva.ID, &request.RegistrarOcorrenciaDTO{OccurrenceType: tipo}); err != nil {
			t.Fatal(err)
		}
	}
	// Romaneio pelo CÓDIGO (o que a tela mostra); o mesmo código na B não serve.
	codigoRomaneio := testutil.UniqueCode()
	codigoA, codigoB := codigoRomaneio, codigoRomaneio+1
	for codigo, empresa := range map[int64]int64{codigoA: a.ID, codigoB: b.ID} {
		if _, err := pool.Exec(context.Background(), `INSERT INTO shipments(code, created_by, enterprise_id) VALUES($1,$2,$3)`, codigo, actor, empresa); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM shipping_carrier_occurrences WHERE carrier_id=$1`, salva.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM shipments WHERE code IN ($1,$2)`, codigoRomaneio, codigoRomaneio+1)
	})
	pedidoInexistente := testutil.UniqueCode()
	if _, err := uc.RegistrarOcorrencia(a.Ctx, salva.ID, &request.RegistrarOcorrenciaDTO{OccurrenceType: "AVARIA", SalesOrderCode: &pedidoInexistente}); err == nil {
		t.Fatal("ocorrência aceitou pedido de venda inexistente")
	}
	if _, err := uc.RegistrarOcorrencia(a.Ctx, salva.ID, &request.RegistrarOcorrenciaDTO{OccurrenceType: "AVARIA", ShipmentCode: &codigoB}); err == nil {
		t.Fatal("ocorrência de A aceitou romaneio de B")
	}
	if _, err := uc.RegistrarOcorrencia(a.Ctx, salva.ID, &request.RegistrarOcorrenciaDTO{OccurrenceType: "REENTREGA", ShipmentCode: &codigoA}); err != nil {
		t.Fatal(err)
	}
	lista, err := uc.ListarOcorrencias(a.Ctx, salva.ID, 10)
	if err != nil || len(lista) != 3 || lista[0].ShipmentCode == nil || *lista[0].ShipmentCode != codigoA {
		t.Fatalf("ocorrência com romaneio não voltou com o código: %+v (err=%v)", lista, err)
	}

	detalhe, err := uc.Obter(a.Ctx, salva.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detalhe.Desempenho.Ocorrencias != 3 || detalhe.Desempenho.Falhas != 1 {
		t.Fatalf("desempenho: %+v", detalhe.Desempenho)
	}
	tipos, _ := uc.ListarTiposOcorrencia(a.Ctx)
	for _, o := range tipos {
		err := uc.ExcluirTipoOcorrencia(a.Ctx, o.ID)
		if o.Code == "CHUVA_FORTE" && err == nil {
			t.Fatal("tipo de ocorrência usado foi apagado")
		}
		if o.Code == "ELOGIO" && err != nil {
			t.Fatalf("tipo sem uso não foi apagado: %v", err)
		}
	}

	// Transportadora nunca usada se exclui.
	outroFornecedor := testutil.UniqueCode()
	if _, err := pool.Exec(a.Ctx, `INSERT INTO suppliers(code,name,document_number,is_active,created_by,enterprise_id)
		VALUES($1,'Transportes A2',$2,TRUE,$3,$4)`, outroFornecedor, strconv.FormatInt(outroFornecedor, 10), actor, a.ID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM suppliers WHERE code=$1`, outroFornecedor) })
	livre, err := uc.Salvar(a.Ctx, 0, &request.SalvarTransportadoraDTO{SupplierCode: outroFornecedor, Modal: "AEREO",
		Vehicles: []request.TransportadoraVeiculoDTO{{Plate: "XYZ9876"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.Excluir(a.Ctx, livre.ID); err != nil {
		t.Fatalf("exclusão de transportadora sem uso: %v", err)
	}
	if _, err := uc.Obter(a.Ctx, livre.ID); err == nil {
		t.Fatal("transportadora excluída ainda existe")
	}

	// Carga aberta de A com o veículo: trava inativação, exclusão e remoção da placa.
	if _, err := pool.Exec(a.Ctx, `INSERT INTO shipment_loads(code, status, created_by, enterprise_id, shipping_carrier_id, vehicle_plate)
		VALUES($1,'PLANNED',$2,$3,$4,'ABC-1D23')`, testutil.UniqueCode(), actor, a.ID, salva.ID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM shipment_loads WHERE shipping_carrier_id=$1`, salva.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM shipping_carrier_occurrences WHERE carrier_id=$1`, salva.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM shipping_carrier_vehicles WHERE carrier_id=$1`, salva.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM shipping_carriers WHERE id=$1`, salva.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM suppliers WHERE code=$1`, fornecedor)
	})
	uso, err := uc.Uso(a.Ctx, salva.ID)
	if err != nil || uso.PodeExcluir || uso.PodeInativar {
		t.Fatalf("uso com carga aberta: %+v (err=%v)", uso, err)
	}
	if err := uc.DefinirSituacao(a.Ctx, salva.ID, false); err == nil {
		t.Fatal("inativou com carga aberta")
	}
	if err := uc.Excluir(a.Ctx, salva.ID); err == nil {
		t.Fatal("excluiu com carga")
	}
	dto.Vehicles = nil
	if _, err := uc.Salvar(a.Ctx, salva.ID, dto); err == nil || !strings.Contains(err.Error(), "ABC1D23") {
		t.Fatalf("removeu veículo em carga aberta: %v", err)
	}

	// Carga expedida: inativa, mas não exclui (há histórico).
	if _, err := pool.Exec(a.Ctx, `UPDATE shipment_loads SET status='SHIPPED' WHERE shipping_carrier_id=$1`, salva.ID); err != nil {
		t.Fatal(err)
	}
	if err := uc.DefinirSituacao(a.Ctx, salva.ID, false); err != nil {
		t.Fatalf("inativação com carga expedida: %v", err)
	}
	if err := uc.Excluir(a.Ctx, salva.ID); err == nil {
		t.Fatal("excluiu transportadora com carga no histórico")
	}
}

func decimalDe(s string) decimal.Decimal { return decimal.RequireFromString(s) }

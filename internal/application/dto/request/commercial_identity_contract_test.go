package request

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCommercialCreationDTOsDoNotExposeCreatedBy(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "reprogramação", value: CreateDeliveryRescheduleDTO{}},
		{name: "reserva de tanque", value: DeliveryTankReservationDTO{}},
		{name: "reprogramação em lote", value: DeliveryRescheduleBatchDTO{}},
		{name: "conversão de orçamento", value: ConvertSalesQuotationDTO{}},
		{name: "nota de retorno da assistência", value: AddTechnicalAssistanceReturnNoteDTO{}},
		{name: "geração de pedidos da assistência", value: GenerateTechnicalAssistanceOrdersDTO{}},
		{name: "status da assistência", value: UpdateTechnicalAssistanceCallStatusDTO{}},
		{name: "retorno do chamado do SAC", value: AddConsumerServiceCallReturnDTO{}},
		{name: "pedido de venda", value: CreateSalesOrderDTO{}},
		{name: "análise do pedido", value: AnalyzeSalesOrderDTO{}},
		{name: "liberação do pedido", value: ReleaseSalesOrderDTO{}},
		{name: "atendimento do pedido", value: AttendSalesOrderDTO{}},
		{name: "conferência do pedido", value: ConferSalesOrderDTO{}},
		{name: "ação de atraso", value: SaveSalesOrderDelayReasonDTO{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "created_by") {
				t.Fatalf("created_by não deve fazer parte do contrato: %s", encoded)
			}
		})
	}
}

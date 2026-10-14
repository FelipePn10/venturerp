package mrp_uc

import "testing"

// A entrada é conferida antes de qualquer leitura ou envio.
func TestNotificarValidaEntrada(t *testing.T) {
	invalidos := []NotifyExceptionsDTO{
		{PlanCode: 0},
		{PlanCode: 1, WebhookURL: "http://example.com/x"},
		{PlanCode: 1, WebhookURL: "https://192.168.0.10/x"},
		{PlanCode: 1, WebhookURL: "https://100.64.1.1/x"},
		{PlanCode: 1, EmailTo: []string{"nao-e-email"}},
		{PlanCode: 1, EmailTo: []string{"Fulano <f@x.com>"}},
		{PlanCode: 1, EmailTo: []string{"a@semponto"}},
	}
	for _, d := range invalidos {
		if err := d.validar(); err == nil {
			t.Fatalf("entrada inválida aceita: %+v", d)
		}
	}
	muitos := NotifyExceptionsDTO{PlanCode: 1}
	for i := 0; i < 21; i++ {
		muitos.EmailTo = append(muitos.EmailTo, "pcp@empresa.com.br")
	}
	if err := muitos.validar(); err == nil {
		t.Fatal("aceitou mais de 20 e-mails")
	}
	ok := NotifyExceptionsDTO{PlanCode: 1, WebhookURL: " https://hooks.exemplo.com.br/mrp ", EmailTo: []string{" pcp@empresa.com.br ", ""}}
	if err := ok.validar(); err != nil || ok.WebhookURL != "https://hooks.exemplo.com.br/mrp" || len(ok.EmailTo) != 1 {
		t.Fatalf("entrada válida: %+v (%v)", ok, err)
	}
}

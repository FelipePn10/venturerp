package fiscal_uc

import "testing"

func TestDocumentoDoDestinatarioSemMascara(t *testing.T) {
	cnpj, ie, isento := "22.194.160/0001-51", "797.119.750.118", "isento"
	if got := *documentoSoDigitos(&cnpj); got != "22194160000151" {
		t.Fatalf("cnpj: %s", got)
	}
	if got := *inscricaoEstadual(&ie); got != "797119750118" {
		t.Fatalf("ie: %s", got)
	}
	if got := *inscricaoEstadual(&isento); got != "ISENTO" {
		t.Fatalf("isento: %s", got)
	}
	if documentoSoDigitos(nil) != nil {
		t.Fatal("nil continua nil")
	}
}

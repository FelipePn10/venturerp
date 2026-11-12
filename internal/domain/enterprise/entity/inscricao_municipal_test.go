package entity

import (
	"strings"
	"testing"
)

func dadosValidos() DadosFiscais {
	return DadosFiscais{CNPJ: "11.222.333/0001-81", RazaoSocial: "Empresa", UF: "PR", RegimeTributario: "3", CodigoMunicipio: "4115200"}
}

// A inscrição municipal vai para a NFS-e: chega sem espaços, e a tela que não
// manda o campo (nil) não apaga a gravada.
func TestInscricaoMunicipalDaEmpresa(t *testing.T) {
	d := dadosValidos()
	if err := d.Conferir(); err != nil || d.InscricaoMunicipal != nil {
		t.Fatalf("sem o campo: err=%v im=%v", err, d.InscricaoMunicipal)
	}
	im := "  123456 "
	d = dadosValidos()
	d.InscricaoMunicipal = &im
	if err := d.Conferir(); err != nil || *d.InscricaoMunicipal != "123456" {
		t.Fatalf("err=%v im=%q", err, *d.InscricaoMunicipal)
	}
	longa := strings.Repeat("9", 21)
	d = dadosValidos()
	d.InscricaoMunicipal = &longa
	if err := d.Conferir(); err == nil || !strings.Contains(err.Error(), "inscrição municipal") {
		t.Fatalf("aceitou inscrição com 21 caracteres: %v", err)
	}
}

package validation

import "testing"

func TestLerChaveNFe(t *testing.T) {
	// Chave de nota real autorizada em homologação.
	const valida = "41261052454668000102550010000000191556398072"
	c, ok := LerChaveNFe(valida)
	if !ok {
		t.Fatal("chave válida recusada")
	}
	if c.CodigoUF != "41" || c.AnoMes != "2610" || c.Documento != "52454668000102" || c.Modelo != "55" || c.Serie != 1 || c.Numero != 19 {
		t.Fatalf("chave lida errado: %+v", c)
	}
	for nome, chave := range map[string]string{
		"dígito errado": "41261052454668000102550010000000191556398073",
		"curta":         "4126105245466800010255",
		"vazia":         "",
		"com letras":    "4126105245466800010255001000000019155639807A",
	} {
		if _, ok := LerChaveNFe(chave); ok {
			t.Errorf("%s: deveria ser recusada", nome)
		}
	}
}

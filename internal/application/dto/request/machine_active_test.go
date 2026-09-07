package request

import "testing"

// Omitir `is_active` na criação passou a significar "ativa": antes o centro de
// trabalho nascia inativo, sumia das consultas e ainda fazia a criação de
// máquina falhar com uma mensagem que não dizia o motivo.
func TestTipoDeMaquinaNasceAtivoQuandoOCampoNaoVem(t *testing.T) {
	if !(CreateMachineTypeDTO{}).AtivoOuPadrao() {
		t.Fatal("tipo de máquina sem `is_active` deveria nascer ativo")
	}
	if !(CreateMachineDTO{}).AtivoOuPadrao() {
		t.Fatal("máquina sem `is_active` deveria nascer ativa")
	}
}

func TestInativaExplicitamenteContinuaValendo(t *testing.T) {
	falso := false
	if (CreateMachineTypeDTO{IsActive: &falso}).AtivoOuPadrao() {
		t.Fatal("`is_active: false` explícito deveria ser respeitado")
	}
	if (CreateMachineDTO{IsActive: &falso}).AtivoOuPadrao() {
		t.Fatal("`is_active: false` explícito deveria ser respeitado")
	}
}

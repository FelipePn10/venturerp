package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// O menu do perfil OPERATOR oferece a VPRO1200 (parada de máquina) e a tela só
// fala com as rotas de parada. Elas aceitavam apenas ADMIN e USER: o operador
// abria a tela, escolhia a máquina e recebia 403 no primeiro toque. O posto
// abre e fecha a parada — e nada mais da agenda do APS.
func TestPerfilDoPostoAbreEFechaParada(t *testing.T) {
	source, err := os.ReadFile("api.go")
	if err != nil {
		t.Fatalf("não foi possível ler api.go: %v", err)
	}
	linhaDa := func(rota string) string {
		re := regexp.MustCompile(`(?m)^.*` + regexp.QuoteMeta(rota) + `.*$`)
		return re.FindString(string(source))
	}

	for _, rota := range []string{
		`Post("/machine-stops/open", apsHandler.OpenMachineStop)`,
		`Post("/machine-stops/{machineID}/close", apsHandler.CloseMachineStop)`,
		`Get("/machine-stops/{machineID}", apsHandler.MachineStopStatus)`,
	} {
		linha := linhaDa(rota)
		if linha == "" {
			t.Fatalf("rota de parada ausente: %s", rota)
		}
		if !strings.Contains(linha, `"OPERATOR"`) {
			t.Errorf("o posto precisa usar %s: %s", rota, strings.TrimSpace(linha))
		}
		if strings.HasPrefix(rota, "Post") && !strings.Contains(linha, "PermProductionReport") {
			t.Errorf("gravar parada exige o escopo de apontamento: %s", strings.TrimSpace(linha))
		}
	}

	// A VPRO1000 também está no menu do posto: o operador lê a ficha da
	// ferramenta da ordem.
	for _, rota := range []string{
		`Get("/orders", toolSheetHandler.ListOrders)`,
		`Get("/substitutions", toolSheetHandler.ListSubstitutions)`,
		`Get("/{orderId}", toolSheetHandler.GetSheet)`,
	} {
		if !strings.Contains(linhaDa(rota), `"OPERATOR"`) {
			t.Errorf("o posto precisa ler a ficha da ferramenta: %s", rota)
		}
	}

	// A parada planejada, o parâmetro do sequenciamento e o vínculo de
	// ferramenta continuam do PCP.
	for _, rota := range []string{
		`Post("/machine-downtimes", apsHandler.CreateMachineDowntime)`,
		`Put("/sequence/settings", apsHandler.UpdateSequencingSettings)`,
		`Post("/assign", toolSheetHandler.Assign)`,
		`Post("/substitute", toolSheetHandler.Substitute)`,
	} {
		if strings.Contains(linhaDa(rota), `"OPERATOR"`) {
			t.Errorf("o perfil do posto não planeja a agenda: %s", rota)
		}
	}
}

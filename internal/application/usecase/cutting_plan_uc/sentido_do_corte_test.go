package cutting_plan_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func TestSentidoDoCorte(t *testing.T) {
	semVeio := &itementity.Item{}
	mdf := &itementity.Item{Engineering: itementity.Engineering{CuttingGrain: itementity.VeioComprimento}}
	peca := &itementity.Item{Engineering: itementity.Engineering{CuttingGrain: itementity.VeioLargura}}
	travada := &itementity.Item{Engineering: itementity.Engineering{CuttingNoRotation: true}}

	if g, r := sentidoDoCorte(semVeio, semVeio, true); g != entity.GrainNone || !r {
		t.Fatalf("sem veio gira: %s %v", g, r)
	}
	if g, r := sentidoDoCorte(semVeio, mdf, true); g != entity.GrainLength || r {
		t.Fatalf("o veio do material vale e trava o giro: %s %v", g, r)
	}
	if g, _ := sentidoDoCorte(peca, mdf, true); g != entity.GrainWidth {
		t.Fatalf("a peça manda sobre o material: %s", g)
	}
	if _, r := sentidoDoCorte(travada, semVeio, true); r {
		t.Fatal("peça marcada não gira")
	}
}

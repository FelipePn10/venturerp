package item_conversion_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
)

func dec(s string) decimal.Decimal   { return decimal.RequireFromString(s) }
func decp(s string) *decimal.Decimal { d := dec(s); return &d }

func criar(t *testing.T, uc *ItemConversionUseCase, de, para string, qde, qpara string) int64 {
	t.Helper()
	c, err := uc.Create(context.Background(), request.CreateItemConversionDTO{ItemCode: "TB-40X40", FromUOM: de, ToUOM: para,
		FromQuantity: decp(qde), ToQuantity: decp(qpara)})
	if err != nil {
		t.Fatalf("cadastro %s→%s recusado: %v", de, para, err)
	}
	return c.ID
}

// O bug que motivou a revisão: a tela alterava pelo POST, e o POST devolvia
// 409 porque o par já existia. Agora a alteração tem rota própria.
func TestUpdateCorrigeFatorNoLugarComMotivo(t *testing.T) {
	uc, repo := newContractUC(t)
	id := criar(t, uc, "UN", "KG", "1", "23")
	_, err := uc.Update(context.Background(), id, request.UpdateItemConversionDTO{ToQuantity: decp("23.5"), FromQuantity: decp("1")})
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("trocar o fator sem motivo deveria ser recusado, veio %v", err)
	}
	out, err := uc.Update(context.Background(), id, request.UpdateItemConversionDTO{FromQuantity: decp("1"), ToQuantity: decp("23.5"), Reason: "pesagem conferida"})
	if err != nil {
		t.Fatalf("alteração recusada: %v", err)
	}
	if out.ID != id || !out.FactorExact.Equal(dec("23.5")) {
		t.Fatalf("a correção deveria manter a mesma versão com o fator novo: %+v", out)
	}
	ultimo := repo.events[len(repo.events)-1]
	if ultimo.Action != entity.EventoAlterada || ultimo.Reason == nil || ultimo.Before["to_quantity"] != "23" {
		t.Fatalf("histórico sem o antes/motivo: %+v", ultimo)
	}
}

func TestUpdateSoPoliticaNaoExigeMotivo(t *testing.T) {
	uc, _ := newContractUC(t)
	id := criar(t, uc, "CX", "UN", "1", "12")
	if _, err := uc.Update(context.Background(), id, request.UpdateItemConversionDTO{RoundingPercent: decp("5")}); err != nil {
		t.Fatalf("alterar só o arredondamento não exige motivo: %v", err)
	}
}

func TestUpdateNovaVigenciaEncerraAAnterior(t *testing.T) {
	uc, repo := newContractUC(t)
	id := criar(t, uc, "UN", "KG", "1", "23")
	inicio := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	out, err := uc.Update(context.Background(), id, request.UpdateItemConversionDTO{FromQuantity: decp("1"), ToQuantity: decp("24"),
		ValidFrom: inicio, Reason: "chapa mais grossa a partir do próximo lote"})
	if err != nil {
		t.Fatalf("nova vigência recusada: %v", err)
	}
	if out.ID == id || out.ValidFrom != inicio || out.Situation != entity.SituacaoFutura {
		t.Fatalf("deveria nascer outra versão, futura: %+v", out)
	}
	antiga, _ := repo.GetByID(context.Background(), id)
	if antiga.ValidTo == nil || antiga.ValidTo.Format("2006-01-02") != time.Now().AddDate(0, 1, -1).Format("2006-01-02") {
		t.Fatalf("a versão anterior deveria terminar na véspera: %+v", antiga.ValidTo)
	}
}

func TestCreateRecusaInversaDoMesmoFato(t *testing.T) {
	uc, _ := newContractUC(t)
	criar(t, uc, "UN", "KG", "1", "23")
	_, err := uc.Create(context.Background(), request.CreateItemConversionDTO{ItemCode: "TB-40X40", FromUOM: "KG", ToUOM: "UN", Factor: dec("0.05")})
	if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("KG→UN é o mesmo fato que UN→KG: esperado conflito, veio %v", err)
	}
}

func TestCreateRecusaCaminhoContraditorio(t *testing.T) {
	uc, _ := newContractUC(t)
	criar(t, uc, "UN", "M", "1", "6")
	criar(t, uc, "M", "KG", "1", "4")
	_, err := uc.Create(context.Background(), request.CreateItemConversionDTO{ItemCode: "TB-40X40", FromUOM: "UN", ToUOM: "KG", Factor: dec("23")})
	v, ok := errorsuc.AsValidation(err)
	if !ok || !strings.Contains(v.Error(), "24") {
		t.Fatalf("UN→KG = 23 contradiz 6 × 4 = 24: esperado erro explicando, veio %v", err)
	}
}

func TestCreateRecusaUnidadeForaDoCatalogo(t *testing.T) {
	uc, repo := newContractUC(t)
	repo.missing = []string{"BARRA"}
	_, err := uc.Create(context.Background(), request.CreateItemConversionDTO{ItemCode: "TB-40X40", FromUOM: "BARRA", ToUOM: "M", Factor: dec("6")})
	if v, ok := errorsuc.AsValidation(err); !ok || !strings.Contains(v.Error(), "catálogo") {
		t.Fatalf("unidade fora do catálogo deveria ser recusada, veio %v", err)
	}
}

// Compro em KG, estoco em UN e a estrutura consome em MM — com duas
// conversões e a padrão M→MM o sistema chega a qualquer combinação.
func TestConversaoEmNiveis(t *testing.T) {
	uc, repo := newContractUC(t)
	repo.padrao = []service.Aresta{{ID: 900, De: "M", Para: "MM", QtdDe: dec("1"), QtdPara: dec("1000"), Origem: service.OrigemPadrao}}
	repo.acceptsFractional = true
	criar(t, uc, "UN", "KG", "1", "23.5")
	criar(t, uc, "UN", "M", "1", "6")
	q, ok, err := uc.ConvertQuantity(context.Background(), 4200, dec("47"), "KG", "MM")
	if err != nil || !ok || !q.Equal(dec("12000")) {
		t.Fatalf("47 KG = 2 UN = 12 M = 12000 MM, deu %v (ok=%v err=%v)", q, ok, err)
	}
	sim, err := uc.Simulate(context.Background(), 4200, "", "", "", "KG", "MM", dec("47"))
	if err != nil || len(sim.Steps) != 3 || !strings.Contains(sim.Explanation, "padrão") {
		t.Fatalf("simulação deveria mostrar o caminho de 3 passos: %+v err=%v", sim, err)
	}
}

func TestCoberturaApontaPapelSemConversao(t *testing.T) {
	uc, repo := newContractUC(t)
	repo.unidades = &repository.UnidadesDoItem{ItemCode: 4200, Estoque: "UN", Compra: "KG", VendaContabil: "CX",
		Usos: []repository.UsoDeUnidade{{Unidade: "MM", Onde: "ESTRUTURA", Quantidade: 3}}}
	criar(t, uc, "UN", "KG", "1", "23.5")
	cov, err := uc.Coverage(context.Background(), 4200, "")
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, r := range cov.Rows {
		status[r.Role+":"+r.Unit] = r.Status
	}
	if status["ESTOQUE:UN"] != "BASE" || status["COMPRA:KG"] != "OK" || status["VENDA:CX"] != "SEM_CONVERSAO" || status["ESTRUTURA:MM"] != "SEM_CONVERSAO" {
		t.Fatalf("matriz errada: %v", status)
	}
	if cov.Missing != 2 {
		t.Fatalf("faltam 2 conversões (CX e MM), deu %d", cov.Missing)
	}
}

func TestUpdateAvisaSPEDNoMesmoMes(t *testing.T) {
	uc, repo := newContractUC(t)
	id := criar(t, uc, "UN", "KG", "1", "23")
	repo.impacto = repository.Impacto{NotasAprovadasNoMes: 2}
	out, err := uc.Update(context.Background(), id, request.UpdateItemConversionDTO{FromQuantity: decp("1"), ToQuantity: decp("24"), Reason: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Warnings) == 0 || !strings.Contains(out.Warnings[0], "SPED") {
		t.Fatalf("deveria avisar do SPED: %v", out.Warnings)
	}
}

func TestDeleteRegistraHistorico(t *testing.T) {
	uc, repo := newContractUC(t)
	id := criar(t, uc, "UN", "KG", "1", "23")
	if err := uc.DeleteWithReason(context.Background(), id, "cadastro duplicado"); err != nil {
		t.Fatal(err)
	}
	ev := repo.events[len(repo.events)-1]
	if ev.Action != entity.EventoExcluida || ev.Reason == nil || *ev.Reason != "cadastro duplicado" {
		t.Fatalf("exclusão sem histórico: %+v", ev)
	}
	if list, _ := uc.ListByItem(context.Background(), 4200); len(list) != 0 {
		t.Fatalf("conversão excluída ainda aparece: %d", len(list))
	}
}

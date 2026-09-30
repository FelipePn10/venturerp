//go:build integration

// Testes da trilha de auditoria do beneficiamento (migração 000371).
//
// O levantamento da Usimac (documento 10) pede, por operação: usuário, data e
// hora, operação realizada, registro afetado, informação anterior, nova
// informação e motivo. O documento 6 pede, em particular, histórico de alteração
// de descrição e NCM do material do cliente. Cada teste aqui cobra um desses
// itens contra o banco de verdade — não contra o que o código diz que grava.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run Auditoria ./internal/infrastructure/repository/customer_material/
package customer_material_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
)

// trilha lê a auditoria pelo repositório, como a tela leria.
func (a *ambiente) trilha(t *testing.T, remessaID int64) []*entity.EventoDeAuditoria {
	t.Helper()
	eventos, err := a.repo.TrilhaDaRemessa(a.ctx, remessaID, 0)
	if err != nil {
		t.Fatalf("lendo a trilha: %v", err)
	}
	return eventos
}

// eventoDe encontra o evento de uma tabela e operação. Devolve nil se não houver —
// quem chama decide se a ausência é falha.
func eventoDe(eventos []*entity.EventoDeAuditoria, tabela, acao string) *entity.EventoDeAuditoria {
	for _, e := range eventos {
		if e.EntityType == tabela && e.Acao == acao {
			return e
		}
	}
	return nil
}

func campoJSON(t *testing.T, bruto []byte, campo string) string {
	t.Helper()
	if len(bruto) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(bruto, &m); err != nil {
		t.Fatalf("estado gravado não é JSON: %v", err)
	}
	if v, ok := m[campo]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func contem(lista []string, valor string) bool {
	for _, v := range lista {
		if v == valor {
			return true
		}
	}
	return false
}

// segundoUsuario cria outra pessoa para assinar a alteração. É o que torna a
// asserção de autor honesta: com um usuário só, atribuir a mudança a quem criou o
// registro dá o mesmo id e o teste passaria com o ator quebrado.
func (a *ambiente) segundoUsuario(t *testing.T) string {
	t.Helper()
	var id string
	if err := a.pool.QueryRow(a.ctx,
		`INSERT INTO users (id, name, email, password, role, is_active)
		 VALUES (gen_random_uuid(), 'Benef Revisor', 'benef-rev-'||$1||'@teste.local', 'x', 'ADMIN', true)
		 RETURNING id::text`, fmt.Sprint(a.enterpriseID)).Scan(&id); err != nil {
		t.Fatalf("criando segundo usuário: %v", err)
	}
	t.Cleanup(func() { _, _ = a.pool.Exec(a.ctx, `DELETE FROM users WHERE id = $1::uuid`, id) })
	return id
}

// TestAuditoriaRegistraRecebimentoComAutor: o recebimento da remessa e de cada
// linha entra na trilha assinado. Sem autor no insert, a trilha responde "alguém"
// — que é o mesmo que não responder.
func TestAuditoriaRegistraRecebimentoComAutor(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")

	eventos := a.trilha(t, r.ID)
	remessa := eventoDe(eventos, "customer_material_remittances", "INSERT")
	if remessa == nil {
		t.Fatal("recebimento da remessa não entrou na trilha")
	}
	if remessa.AtorID == nil || *remessa.AtorID != a.userID {
		t.Fatalf("autor do recebimento = %v, esperado %s", remessa.AtorID, a.userID)
	}
	if remessa.Antes != nil {
		t.Fatalf("INSERT não tem estado anterior, veio %s", remessa.Antes)
	}
	if got := campoJSON(t, remessa.Depois, "status"); got != "ABERTA" {
		t.Fatalf("estado novo gravou status %q, esperado ABERTA", got)
	}
	if remessa.OcorridoEm.IsZero() {
		t.Fatal("evento sem data e hora")
	}

	item := eventoDe(eventos, "customer_material_items", "INSERT")
	if item == nil {
		t.Fatal("linha da remessa não entrou na trilha")
	}
	if item.RemittanceID == nil || *item.RemittanceID != r.ID {
		t.Fatalf("linha auditada sem vínculo com a remessa: %v", item.RemittanceID)
	}
	if got := campoJSON(t, item.Depois, "ncm"); got != "76169900" {
		t.Fatalf("NCM gravado na trilha = %q", got)
	}
}

// TestAuditoriaGuardaDescricaoENCMAnteriores cobra o documento 6: alterar
// descrição ou NCM do material do cliente tem de deixar o valor antigo e o novo
// registrados, e apontar quais campos mudaram.
func TestAuditoriaGuardaDescricaoENCMAnteriores(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	item := r.Itens[0]

	testutilExec(t, a, `
UPDATE customer_material_items
SET description = 'SUB CJ SOLDADO DA LAMINA - REV 2', ncm = '73269090', updated_at = NOW()
WHERE id = $1`, item.ID)

	e := eventoDe(a.trilha(t, r.ID), "customer_material_items", "UPDATE")
	if e == nil {
		t.Fatal("alteração de descrição/NCM não entrou na trilha")
	}
	if got := campoJSON(t, e.Antes, "description"); got != "SUB CJ SOLDADO DA LAMINA" {
		t.Fatalf("descrição anterior = %q", got)
	}
	if got := campoJSON(t, e.Depois, "description"); got != "SUB CJ SOLDADO DA LAMINA - REV 2" {
		t.Fatalf("descrição nova = %q", got)
	}
	if got := campoJSON(t, e.Antes, "ncm"); got != "76169900" {
		t.Fatalf("NCM anterior = %q", got)
	}
	if got := campoJSON(t, e.Depois, "ncm"); got != "73269090" {
		t.Fatalf("NCM novo = %q", got)
	}
	if !contem(e.Alterados, "description") || !contem(e.Alterados, "ncm") {
		t.Fatalf("campos alterados = %v, esperado conter description e ncm", e.Alterados)
	}
	// updated_at muda em toda alteração; apontá-lo como mudança de conteúdo
	// esconde a alteração de verdade no meio do ruído.
	if contem(e.Alterados, "updated_at") {
		t.Fatalf("updated_at não deve contar como campo alterado: %v", e.Alterados)
	}
}

// TestAuditoriaIgnoraToqueSemConteudo: o recálculo de situação toca updated_at a
// cada movimento. Linha de trilha sem mudança de conteúdo é ruído.
func TestAuditoriaIgnoraToqueSemConteudo(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	antes := len(a.trilha(t, r.ID))

	testutilExec(t, a, `UPDATE customer_material_remittances SET updated_at = NOW() WHERE id = $1`, r.ID)

	if depois := len(a.trilha(t, r.ID)); depois != antes {
		t.Fatalf("trilha foi de %d para %d eventos por um toque sem conteúdo", antes, depois)
	}
}

// TestAuditoriaDoMovimentoGuardaMotivoEAutor: o ajuste é a operação que mais
// precisa de justificativa — é ela que reconcilia saldo divergente.
func TestAuditoriaDoMovimentoGuardaMotivoEAutor(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	item := r.Itens[0]

	conferente := a.segundoUsuario(t)
	motivo := "conferência física: 1 peça a menos que a nota"
	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoAjuste,
		Quantity: "1", Reason: &motivo, IdempotencyKey: "aj-" + fmt.Sprint(item.ID),
	}, conferente); err != nil {
		t.Fatalf("ajuste recusado: %v", err)
	}

	eventos := a.trilha(t, r.ID)
	mov := eventoDe(eventos, "customer_material_movements", "INSERT")
	if mov == nil {
		t.Fatal("movimento não entrou na trilha")
	}
	if mov.Motivo == nil || *mov.Motivo != motivo {
		t.Fatalf("motivo na trilha = %v, esperado %q", mov.Motivo, motivo)
	}
	if mov.AtorID == nil || *mov.AtorID != conferente {
		t.Fatalf("autor do movimento = %v, esperado %s", mov.AtorID, conferente)
	}
	if mov.RemittanceID == nil || *mov.RemittanceID != r.ID {
		t.Fatalf("movimento auditado sem vínculo com a remessa: %v", mov.RemittanceID)
	}
	// A baixa do consumo na linha é outra alteração, e também é auditada: é ela que
	// mostra o saldo antes e depois.
	linha := eventoDe(eventos, "customer_material_items", "UPDATE")
	if linha == nil {
		t.Fatal("baixa do consumo na linha não entrou na trilha")
	}
	if antes, depois := campoJSON(t, linha.Antes, "balance_qty"), campoJSON(t, linha.Depois, "balance_qty"); antes == depois {
		t.Fatalf("saldo anterior e novo iguais na trilha (%s)", antes)
	}
	if linha.AtorID == nil || *linha.AtorID != conferente {
		t.Fatalf("autor da baixa = %v, esperado %s — a linha não tem coluna de autor, o ator de sessão é que responde", linha.AtorID, conferente)
	}
}

// TestAuditoriaDoBloqueioEDoEncerramento: as duas decisões de pessoa sobre
// material de terceiro. Ambas exigem motivo, e a trilha tem de dizer quem decidiu.
func TestAuditoriaDoBloqueioEDoEncerramento(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")

	revisor := a.segundoUsuario(t)
	if err := a.repo.Bloquear(a.ctx, r.ID, "material sem pedido de venda", revisor); err != nil {
		t.Fatalf("bloqueio recusado: %v", err)
	}
	e := eventoDe(a.trilha(t, r.ID), "customer_material_remittances", "UPDATE")
	if e == nil {
		t.Fatal("bloqueio não entrou na trilha")
	}
	if e.Motivo == nil || *e.Motivo != "material sem pedido de venda" {
		t.Fatalf("motivo do bloqueio = %v", e.Motivo)
	}
	if e.AtorID == nil || *e.AtorID != revisor {
		t.Fatalf("autor do bloqueio = %v, esperado %s (quem bloqueou, não quem criou)", e.AtorID, revisor)
	}
	if campoJSON(t, e.Antes, "blocked") != "false" || campoJSON(t, e.Depois, "blocked") != "true" {
		t.Fatalf("trilha do bloqueio não mostra a virada: antes=%s depois=%s", e.Antes, e.Depois)
	}

	if err := a.repo.Desbloquear(a.ctx, r.ID, revisor); err != nil {
		t.Fatalf("desbloqueio recusado: %v", err)
	}
	if err := a.repo.Encerrar(a.ctx, r.ID, "cliente desistiu do beneficiamento", revisor); err != nil {
		t.Fatalf("encerramento recusado: %v", err)
	}

	var encerramento *entity.EventoDeAuditoria
	for _, ev := range a.trilha(t, r.ID) {
		if ev.EntityType == "customer_material_remittances" && ev.Acao == "UPDATE" &&
			campoJSON(t, ev.Depois, "status") == "ENCERRADA" {
			encerramento = ev
			break
		}
	}
	if encerramento == nil {
		t.Fatal("encerramento não entrou na trilha")
	}
	if encerramento.Motivo == nil || *encerramento.Motivo != "cliente desistiu do beneficiamento" {
		t.Fatalf("motivo do encerramento = %v", encerramento.Motivo)
	}
	if !contem(encerramento.Alterados, "status") {
		t.Fatalf("campos alterados no encerramento = %v", encerramento.Alterados)
	}
	if encerramento.AtorID == nil || *encerramento.AtorID != revisor {
		t.Fatalf("autor do encerramento = %v, esperado %s", encerramento.AtorID, revisor)
	}
}

// TestAuditoriaEhImutavel: trilha que pode ser reescrita não serve de prova em
// divergência de saldo com o cliente.
func TestAuditoriaEhImutavel(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	eventos := a.trilha(t, r.ID)
	if len(eventos) == 0 {
		t.Fatal("sem evento para tentar alterar")
	}

	_, err := a.pool.Exec(a.ctx, `UPDATE customer_material_audit SET reason = 'reescrito' WHERE id = $1`, eventos[0].ID)
	if err == nil {
		t.Fatal("a trilha aceitou ser alterada")
	}
	if !strings.Contains(err.Error(), "imutável") {
		t.Fatalf("erro inesperado ao alterar a trilha: %v", err)
	}

	if _, err := a.pool.Exec(a.ctx, `DELETE FROM customer_material_audit WHERE id = $1`, eventos[0].ID); err == nil {
		t.Fatal("a trilha aceitou ser apagada")
	}
}

// TestAuditoriaNaoVazaEntreEmpresas: a trilha guarda o registro inteiro, então um
// vazamento aqui entrega mais do que o vazamento da própria remessa.
func TestAuditoriaNaoVazaEntreEmpresas(t *testing.T) {
	a := montar(t)
	outra := montar(t)
	r := a.remessa(t, "6")

	eventos, err := outra.repo.TrilhaDaRemessa(outra.ctx, r.ID, 0)
	// Erro aqui não vale como isolamento: consulta quebrada também devolve erro, e
	// passaria o teste sem provar nada.
	if err != nil {
		t.Fatalf("lendo a trilha pela empresa vizinha: %v", err)
	}
	if len(eventos) > 0 {
		t.Fatalf("empresa vizinha leu %d eventos da trilha alheia", len(eventos))
	}
	// Controle positivo: a dona vê a própria trilha, senão o teste acima passaria
	// com a consulta devolvendo vazio para todo mundo.
	if proprios := a.trilha(t, r.ID); len(proprios) == 0 {
		t.Fatal("a dona da remessa não viu a própria trilha")
	}
}

// TestAuditoriaSobreviveAoRegistroApagado: a trilha é a defesa quando o registro
// não existe mais. Se ela cair junto, não era trilha.
func TestAuditoriaSobreviveAoRegistroApagado(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")

	testutilExec(t, a, `DELETE FROM customer_material_remittances WHERE id = $1`, r.ID)

	var total int
	if err := a.pool.QueryRow(a.ctx,
		`SELECT count(*) FROM customer_material_audit WHERE remittance_id = $1 AND action = 'DELETE'`,
		r.ID).Scan(&total); err != nil {
		t.Fatalf("contando a trilha da exclusão: %v", err)
	}
	// A remessa e a sua linha; o movimento não existe neste caso.
	if total < 2 {
		t.Fatalf("exclusão deixou %d evento(s) na trilha, esperado ao menos 2", total)
	}
}

// testutilExec roda SQL direto quando o teste precisa simular uma alteração que o
// repositório não expõe (correção de cadastro, toque em updated_at).
func testutilExec(t *testing.T, a *ambiente, sql string, args ...any) {
	t.Helper()
	if _, err := a.pool.Exec(a.ctx, sql, args...); err != nil {
		t.Fatalf("executando %s: %v", strings.Fields(sql)[0], err)
	}
}

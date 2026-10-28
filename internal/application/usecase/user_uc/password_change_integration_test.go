//go:build integration

package user_uc

import (
	"context"
	"errors"
	"os"
	"testing"

	userrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type fixedAuth struct {
	userID       uuid.UUID
	enterpriseID int64
	admin        bool
}

func (a fixedAuth) UserID(context.Context) (uuid.UUID, error)   { return a.userID, nil }
func (a fixedAuth) EnterpriseID(context.Context) (int64, error) { return a.enterpriseID, nil }
func (a fixedAuth) IsAdmin(context.Context) bool                { return a.admin }

// Fluxo completo da troca de senha no banco: duas empresas, segregação de
// funções, aprovação vencida que não pode travar o usuário e mensagens certas
// para cada recusa.
func TestIntegrationPasswordChangeFlow(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definida")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	// Cleanup roda em ordem inversa: a limpeza (registrada depois) roda antes
	// de fechar o pool. Com defer, o pool fechava antes e a limpeza falhava.
	t.Cleanup(pool.Close)

	const emp1, emp2 int64 = 930000001, 930000002
	const senha = "SenhaAtual#2026"
	hash, _ := bcrypt.GenerateFromPassword([]byte(senha), 4)
	usuario, admin1, admin2, adminOutra := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	todos := []uuid.UUID{usuario, admin1, admin2, adminOutra}
	limpar := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM password_change_requests WHERE enterprise_id IN ($1,$2)`, emp1, emp2)
		for _, id := range todos {
			_, _ = pool.Exec(ctx, `DELETE FROM password_change_requests WHERE user_id=$1`, id)
			_, _ = pool.Exec(ctx, `DELETE FROM user_enterprises WHERE user_id=$1`, id)
			_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
		}
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE id IN ($1,$2)`, emp1, emp2)
	}
	limpar()
	t.Cleanup(limpar)
	for _, e := range []int64{emp1, emp2} {
		if _, err := pool.Exec(ctx, `INSERT INTO enterprise(id,code,name) VALUES($1,$2,'Empresa troca de senha')`, e, e-929000000); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []struct {
		id   uuid.UUID
		nome string
		role string
		emp  int64
	}{{usuario, "Usuária Teste", "USER", emp1}, {admin1, "Admin Um", "ADMIN", emp1}, {admin2, "Admin Dois", "ADMIN", emp1}, {adminOutra, "Admin Outra", "ADMIN", emp2}} {
		if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password,role,auth_version,is_active) VALUES($1,$2,$3,$4,$5,1,true)`,
			u.id, u.nome, u.id.String()+"@senha.test", string(hash), u.role); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,$3)`, u.id, u.emp, u.role); err != nil {
			t.Fatal(err)
		}
	}
	repo := userrepo.NewRepositoryUserSQLC(nil, pool)
	como := func(id uuid.UUID, emp int64, admin bool) *PasswordChangeUseCase {
		return NewPasswordChangeUseCase(repo, fixedAuth{id, emp, admin})
	}
	uUsuario, uAdmin1, uAdmin2, uOutra := como(usuario, emp1, false), como(admin1, emp1, true), como(admin2, emp1, true), como(adminOutra, emp2, true)

	// Solicita com motivo; a segunda solicitação é recusada com mensagem clara.
	req, err := uUsuario.Request(ctx, "  Primeiro acesso  ")
	if err != nil || req.Status != "PENDING" || req.Reason == nil || *req.Reason != "Primeiro acesso" || !req.Own {
		t.Fatalf("solicitação: %+v %v", req, err)
	}
	if _, err := uUsuario.Request(ctx, ""); !errors.Is(err, ErrPasswordChangeActive) {
		t.Fatalf("segunda solicitação: %v", err)
	}
	if mine, err := uUsuario.Mine(ctx); err != nil || len(mine) != 1 || mine[0].ID != req.ID {
		t.Fatalf("minhas: %+v %v", mine, err)
	}
	// Usuário comum não lista nem aprova.
	if _, err := uUsuario.List(ctx, ""); !errors.Is(err, ErrPasswordChangeForbidden) {
		t.Fatalf("lista por usuário comum: %v", err)
	}
	// Outra empresa não vê nem decide.
	if lista, err := uOutra.List(ctx, ""); err != nil || len(lista) != 0 {
		t.Fatalf("outra empresa listou: %+v %v", lista, err)
	}
	if err := uOutra.Approve(ctx, req.ID); !errors.Is(err, ErrPasswordChangeNotFound) {
		t.Fatalf("aprovação por outra empresa: %v", err)
	}
	lista, err := uAdmin1.List(ctx, "PENDING")
	if err != nil || len(lista) != 1 || lista[0].UserName != "Usuária Teste" || lista[0].Own {
		t.Fatalf("lista do admin: %+v %v", lista, err)
	}
	// Concluir antes da aprovação.
	if err := uUsuario.Complete(ctx, req.ID, senha, "NovaSenha#2026x"); !errors.Is(err, ErrPasswordChangeNotReady) {
		t.Fatalf("concluir sem aprovação: %v", err)
	}
	if err := uAdmin1.Reject(ctx, req.ID, "x"); !errors.Is(err, ErrRejectionReason) {
		t.Fatalf("rejeição sem motivo: %v", err)
	}
	if err := uAdmin1.Approve(ctx, req.ID); err != nil {
		t.Fatal(err)
	}
	if err := uAdmin2.Approve(ctx, req.ID); !errors.Is(err, ErrPasswordChangeDecided) {
		t.Fatalf("aprovar de novo: %v", err)
	}
	lista, _ = uAdmin1.List(ctx, "APPROVED")
	if len(lista) != 1 || lista[0].ApprovedByName == nil || *lista[0].ApprovedByName != "Admin Um" || lista[0].ExpiresAt == nil {
		t.Fatalf("aprovada: %+v", lista)
	}

	// Aprovação vencida: aparece expirada, não conclui e não trava novo pedido.
	if _, err := pool.Exec(ctx, `UPDATE password_change_requests SET expires_at = now() - interval '1 minute' WHERE id=$1`, req.ID); err != nil {
		t.Fatal(err)
	}
	if lista, _ = uAdmin1.List(ctx, "EXPIRED"); len(lista) != 1 {
		t.Fatalf("vencida não aparece como expirada: %+v", lista)
	}
	if err := uUsuario.Complete(ctx, req.ID, senha, "NovaSenha#2026x"); !errors.Is(err, ErrPasswordChangeExpired) {
		t.Fatalf("concluir vencida: %v", err)
	}
	req2, err := uUsuario.Request(ctx, "Nova tentativa")
	if err != nil {
		t.Fatalf("pedido depois de aprovação vencida: %v", err)
	}
	if err := uAdmin2.Approve(ctx, req2.ID); err != nil {
		t.Fatal(err)
	}
	if err := uUsuario.Complete(ctx, req2.ID, "errada#2026X", "NovaSenha#2026x"); !errors.Is(err, ErrCurrentPasswordInvalid) {
		t.Fatalf("senha atual errada: %v", err)
	}
	if err := uUsuario.Complete(ctx, req2.ID, senha, senha); !errors.Is(err, ErrSamePassword) {
		t.Fatalf("mesma senha: %v", err)
	}
	if err := uUsuario.Complete(ctx, req2.ID, senha, "curta"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("senha fraca: %v", err)
	}
	// Outro usuário não usa a aprovação alheia.
	if err := uAdmin1.Complete(ctx, req2.ID, senha, "NovaSenha#2026x"); !errors.Is(err, ErrPasswordChangeNotFound) {
		t.Fatalf("aprovação alheia: %v", err)
	}
	if err := uUsuario.Complete(ctx, req2.ID, senha, "NovaSenha#2026x"); err != nil {
		t.Fatal(err)
	}
	var versao int64
	var novoHash string
	if err := pool.QueryRow(ctx, `SELECT auth_version, password FROM users WHERE id=$1`, usuario).Scan(&versao, &novoHash); err != nil {
		t.Fatal(err)
	}
	if versao != 2 || bcrypt.CompareHashAndPassword([]byte(novoHash), []byte("NovaSenha#2026x")) != nil {
		t.Fatalf("senha não trocada: versão %d", versao)
	}
	if err := uUsuario.Complete(ctx, req2.ID, "NovaSenha#2026x", "OutraSenha#2026y"); !errors.Is(err, ErrPasswordChangeNotReady) {
		t.Fatalf("reuso da aprovação: %v", err)
	}

	// Administrador não aprova a própria solicitação; outro administrador sim.
	propria, err := uAdmin1.Request(ctx, "")
	if err != nil || propria.Reason != nil {
		t.Fatalf("solicitação do admin: %+v %v", propria, err)
	}
	if lista, _ = uAdmin1.List(ctx, "PENDING"); len(lista) != 1 || !lista[0].Own {
		t.Fatalf("própria não marcada: %+v", lista)
	}
	if err := uAdmin1.Approve(ctx, propria.ID); !errors.Is(err, ErrPasswordChangeOwn) {
		t.Fatalf("autoaprovação: %v", err)
	}
	if err := uAdmin2.Reject(ctx, propria.ID, "Sem justificativa"); err != nil {
		t.Fatal(err)
	}
	mine, _ := uAdmin1.Mine(ctx)
	if len(mine) != 1 || mine[0].Status != "REJECTED" || mine[0].RejectionReason == nil || mine[0].RejectedByName == nil || *mine[0].RejectedByName != "Admin Dois" {
		t.Fatalf("rejeitada: %+v", mine)
	}
}

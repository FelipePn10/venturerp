//go:build integration

package user_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	userrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/user"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Usuário com acesso a duas empresas: com a senha certa e sem escolher, o
// login devolve a lista (para a tela perguntar); escolhendo, entra na
// escolhida; senha errada não revela a lista. Com uma empresa só, entra direto.
func TestLoginComMaisDeUmaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	a, b := testutil.DuasEmpresas(t, pool, nil)
	id := uuid.New()
	email := id.String() + "@login.test"
	hash, _ := bcrypt.GenerateFromPassword([]byte("Senha#2026"), bcrypt.MinCost)
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password,role,auth_version,is_active) VALUES($1,'Multi',$2,$3,'USER',1,true)`, id, email, string(hash))
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM user_enterprises WHERE user_id=$1`, id)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, id)
	})
	uc := &LoginUserUseCase{Repo: userrepo.NewRepositoryUserSQLC(sqlc.New(pool), pool)}

	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'USER')`, id, a.ID)
	if _, _, _, _, emp, _, err := uc.Execute(bg, request.LoginUserDTO{Email: email, Password: "Senha#2026"}); err != nil || emp != a.ID {
		t.Fatalf("uma empresa entra direto: %d %v", emp, err)
	}

	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'ADMIN')`, id, b.ID)
	_, _, _, _, _, _, err := uc.Execute(bg, request.LoginUserDTO{Email: email, Password: "Senha#2026"})
	var escolha *ErrEmpresaObrigatoria
	if !errors.As(err, &escolha) || len(escolha.Empresas) != 2 {
		t.Fatalf("duas empresas sem escolha devolvem a lista: %v", err)
	}
	if _, _, _, _, _, _, err := uc.Execute(bg, request.LoginUserDTO{Email: email, Password: "errada"}); errors.As(err, &escolha) {
		t.Fatal("senha errada não revela as empresas")
	}
	code := b.Code
	_, role, _, _, emp, _, err := uc.Execute(bg, request.LoginUserDTO{Email: email, Password: "Senha#2026", EnterpriseCode: &code})
	if err != nil || emp != b.ID || role != "ADMIN" {
		t.Fatalf("escolhendo a B: %d %s %v", emp, role, err)
	}
}

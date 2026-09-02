package user_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	userentity "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	userrepo "github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type loginRepo struct {
	userrepo.UserRepository
	user          *userentity.User
	authorization userrepo.Authorization
}

func (r loginRepo) FindByEmail(context.Context, string) (*userentity.User, error) {
	return r.user, nil
}

func (r loginRepo) ResolveAuthorization(context.Context, string, *int64) (userrepo.Authorization, error) {
	return r.authorization, nil
}

type recordingMirror struct {
	userID       string
	enterpriseID int64
	err          error
}

func (m *recordingMirror) SyncIdentity(_ context.Context, userID string, enterpriseID int64) error {
	m.userID, m.enterpriseID = userID, enterpriseID
	return m.err
}

func TestTrainingLoginMirrorsSelectedIdentityBeforeReturning(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.New()
	mirror := &recordingMirror{}
	uc := NewLoginUserUseCase(loginRepo{
		user:          &userentity.User{ID: userID, Name: "Usuário", Email: "user@example.com", Password: string(hash)},
		authorization: userrepo.Authorization{EnterpriseID: 42, EnterpriseCode: 7, Role: "USER", AuthVersion: 3},
	})
	uc.Mirror = mirror
	_, _, _, _, enterpriseID, _, err := uc.Execute(context.Background(), request.LoginUserDTO{Email: "user@example.com", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if mirror.userID != userID.String() || mirror.enterpriseID != 42 || enterpriseID != 42 {
		t.Fatalf("mirror = (%s,%d), login enterprise = %d", mirror.userID, mirror.enterpriseID, enterpriseID)
	}
}

func TestTrainingLoginFailsClosedWhenIdentityCannotBeMirrored(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	uc := NewLoginUserUseCase(loginRepo{
		user:          &userentity.User{ID: uuid.New(), Password: string(hash)},
		authorization: userrepo.Authorization{EnterpriseID: 42, Role: "USER", AuthVersion: 1},
	})
	uc.Mirror = &recordingMirror{err: errors.New("training database unavailable")}
	_, _, _, _, _, _, err = uc.Execute(context.Background(), request.LoginUserDTO{Password: "secret"})
	if !errors.Is(err, ErrIdentitySync) {
		t.Fatalf("error = %v, want ErrIdentitySync", err)
	}
}

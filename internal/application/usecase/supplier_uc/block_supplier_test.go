package supplier_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/supplier/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/supplier/repository"
)

// blockRepo é um repositório de fornecedores mínimo: só o suficiente para o
// bloqueio/desbloqueio, com o mesmo comportamento do banco (o UPDATE não avisa
// quando não encontra o código).
type blockRepo struct {
	repository.SupplierRepository
	suppliers map[int64]*entity.Supplier
	// silent reproduz o UPDATE que não encontra a linha e mesmo assim não falha.
	silent bool
}

func (r *blockRepo) GetSupplierByCode(_ context.Context, code int64) (*entity.Supplier, error) {
	s, ok := r.suppliers[code]
	if !ok {
		return nil, errors.New("no rows in result set")
	}
	copied := *s
	return &copied, nil
}

func (r *blockRepo) BlockSupplier(_ context.Context, code int64, reason string) error {
	if r.silent {
		return nil
	}
	if s, ok := r.suppliers[code]; ok {
		s.Blocked, s.BlockReason = true, &reason
	}
	return nil
}

func (r *blockRepo) UnblockSupplier(_ context.Context, code int64) error {
	if r.silent {
		return nil
	}
	if s, ok := r.suppliers[code]; ok {
		s.Blocked, s.BlockReason = false, nil
	}
	return nil
}

func newBlockUC(blocked bool) (*SupplierUseCase, *blockRepo) {
	reason := "inadimplência"
	repo := &blockRepo{suppliers: map[int64]*entity.Supplier{
		10: {Code: 10, Name: "MetalFix", Blocked: blocked, BlockReason: &reason},
	}}
	if !blocked {
		repo.suppliers[10].BlockReason = nil
	}
	return NewSupplierUseCase(repo), repo
}

func TestBlockSupplier_PersistsReason(t *testing.T) {
	uc, repo := newBlockUC(false)
	err := uc.BlockSupplier(context.Background(), request.BlockSupplierDTO{Code: 10, Reason: " documentação vencida "})
	if err != nil {
		t.Fatalf("bloqueio recusado: %v", err)
	}
	if !repo.suppliers[10].Blocked || repo.suppliers[10].BlockReason == nil ||
		*repo.suppliers[10].BlockReason != "documentação vencida" {
		t.Fatalf("estado após o bloqueio: %+v", repo.suppliers[10])
	}
}

func TestBlockSupplier_RequiresReason(t *testing.T) {
	uc, _ := newBlockUC(false)
	err := uc.BlockSupplier(context.Background(), request.BlockSupplierDTO{Code: 10, Reason: "   "})
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
}

func TestBlockSupplier_UnknownSupplierIsNotFound(t *testing.T) {
	uc, _ := newBlockUC(false)
	err := uc.BlockSupplier(context.Background(), request.BlockSupplierDTO{Code: 99, Reason: "teste"})
	if _, ok := errorsuc.AsNotFound(err); !ok {
		t.Fatalf("esperado NotFoundError, veio %T (%v)", err, err)
	}
}

func TestBlockSupplier_AlreadyBlockedIsConflict(t *testing.T) {
	uc, _ := newBlockUC(true)
	err := uc.BlockSupplier(context.Background(), request.BlockSupplierDTO{Code: 10, Reason: "teste"})
	if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("esperado ConflictError, veio %T (%v)", err, err)
	}
}

func TestUnblockSupplier_ClearsReason(t *testing.T) {
	uc, repo := newBlockUC(true)
	if err := uc.UnblockSupplier(context.Background(), 10); err != nil {
		t.Fatalf("desbloqueio recusado: %v", err)
	}
	if repo.suppliers[10].Blocked || repo.suppliers[10].BlockReason != nil {
		t.Fatalf("estado após o desbloqueio: %+v", repo.suppliers[10])
	}
}

func TestUnblockSupplier_NotBlockedIsConflict(t *testing.T) {
	uc, _ := newBlockUC(false)
	err := uc.UnblockSupplier(context.Background(), 10)
	if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("esperado ConflictError, veio %T (%v)", err, err)
	}
}

// Um UPDATE que não altera nada não pode responder sucesso à tela — era esse o
// "bloquear sem efeito" relatado.
func TestBlockSupplier_SilentUpdateIsReported(t *testing.T) {
	uc, repo := newBlockUC(false)
	repo.silent = true
	if err := uc.BlockSupplier(context.Background(), request.BlockSupplierDTO{Code: 10, Reason: "teste"}); err == nil {
		t.Fatal("bloqueio sem efeito respondeu sucesso")
	}
	repo.suppliers[10].Blocked = true
	if err := uc.UnblockSupplier(context.Background(), 10); err == nil {
		t.Fatal("desbloqueio sem efeito respondeu sucesso")
	}
}

package nfse_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	nfseentity "github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
	nfserepo "github.com/FelipePn10/panossoerp/internal/domain/nfse/repository"
	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeNFSeRepo struct {
	nfserepo.NFSeRepository
	created   *nfseentity.NFSe
	errCreate error
	errGetID  error
}

func (r *fakeNFSeRepo) Create(_ context.Context, n *nfseentity.NFSe) (*nfseentity.NFSe, error) {
	if r.errCreate != nil {
		return nil, r.errCreate
	}
	n.ID = 101
	r.created = n
	return n, nil
}
func (r *fakeNFSeRepo) GetByID(_ context.Context, id int64) (*nfseentity.NFSe, error) {
	if r.errGetID != nil {
		return nil, r.errGetID
	}
	if r.created != nil && r.created.ID == id {
		return r.created, nil
	}
	return nil, errors.New("not found")
}
func (r *fakeNFSeRepo) List(_ context.Context) ([]*nfseentity.NFSe, error) {
	if r.created != nil {
		return []*nfseentity.NFSe{r.created}, nil
	}
	return nil, nil
}
func (r *fakeNFSeRepo) UpdateStatus(_ context.Context, id int64, status nfseentity.NFSeStatus) (*nfseentity.NFSe, error) {
	if r.created != nil {
		r.created.Status = status
		return r.created, nil
	}
	return nil, errors.New("not found")
}
func (r *fakeNFSeRepo) UpdateAuthorization(_ context.Context, _ int64, num, _, _, _ string) (*nfseentity.NFSe, error) {
	if r.created != nil {
		r.created.NumeroNFSe = &num
		return r.created, nil
	}
	return nil, errors.New("not found")
}
func (r *fakeNFSeRepo) SaveFocusLog(_ context.Context, _, _, _, _ string, _, _ int) error {
	return nil
}

type fakeNFSeAuth struct {
	ports.AuthService
	canCreate bool
	canGet    bool
}

func (a *fakeNFSeAuth) CanCreateFiscalExit(_ context.Context) bool { return a.canCreate }
func (a *fakeNFSeAuth) CanGetFiscalExit(_ context.Context) bool    { return a.canGet }
func (a *fakeNFSeAuth) UserID(_ context.Context) (uuid.UUID, error) {
	return uuid.New(), nil
}

func validCreateDTO() request.CreateNFSeDTO {
	return request.CreateNFSeDTO{
		DataEmissao:      "2026-06-16",
		NaturezaOperacao: 1,
		ItemListaServico: "1.01",
		Discriminacao:    "Servico de montagem industrial",
		CodigoMunicipio:  "3550308",
		ValorServicos:    1200.00,
		AliquotaISS:      0.05,
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCreateNFSe_Success(t *testing.T) {
	repo := &fakeNFSeRepo{}
	uc := &CreateNFSeUseCase{Repo: repo, Auth: &fakeNFSeAuth{canCreate: true}}

	result, err := uc.Execute(context.Background(), validCreateDTO())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != string(nfseentity.NFSeStatusRascunho) {
		t.Errorf("new NFS-e must be RASCUNHO, got %q", result.Status)
	}
	if repo.created == nil {
		t.Fatal("NFS-e was not persisted")
	}
}

func TestCreateNFSe_Unauthorized(t *testing.T) {
	uc := &CreateNFSeUseCase{
		Repo: &fakeNFSeRepo{},
		Auth: &fakeNFSeAuth{canCreate: false},
	}
	_, err := uc.Execute(context.Background(), validCreateDTO())
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

func TestCreateNFSe_ValidationErrors(t *testing.T) {
	auth := &fakeNFSeAuth{canCreate: true}

	cases := []struct {
		name   string
		mutate func(*request.CreateNFSeDTO)
	}{
		{"missing valor_servicos", func(d *request.CreateNFSeDTO) { d.ValorServicos = 0 }},
		{"missing item_lista_servico", func(d *request.CreateNFSeDTO) { d.ItemListaServico = "" }},
		{"missing discriminacao", func(d *request.CreateNFSeDTO) { d.Discriminacao = "" }},
		{"missing codigo_municipio", func(d *request.CreateNFSeDTO) { d.CodigoMunicipio = "" }},
		{"invalid data_emissao", func(d *request.CreateNFSeDTO) { d.DataEmissao = "not-a-date" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dto := validCreateDTO()
			tc.mutate(&dto)
			uc := &CreateNFSeUseCase{Repo: &fakeNFSeRepo{}, Auth: auth}
			_, err := uc.Execute(context.Background(), dto)
			if err == nil {
				t.Errorf("%s: expected validation error, got nil", tc.name)
			}
		})
	}
}

func TestCreateNFSe_ISSCalculation(t *testing.T) {
	repo := &fakeNFSeRepo{}
	uc := &CreateNFSeUseCase{Repo: repo, Auth: &fakeNFSeAuth{canCreate: true}}

	dto := validCreateDTO()
	dto.ValorServicos = 1000
	dto.ValorDeducoes = 200
	dto.AliquotaISS = 0.05

	_, err := uc.Execute(context.Background(), dto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// base = 1000 − 200 = 800; ISS = 800 × 0.05 = 40
	if repo.created.ValorISS != 40 {
		t.Errorf("ValorISS = %v, want 40", repo.created.ValorISS)
	}
}

func TestCreateNFSe_RepoError(t *testing.T) {
	uc := &CreateNFSeUseCase{
		Repo: &fakeNFSeRepo{errCreate: errors.New("db error")},
		Auth: &fakeNFSeAuth{canCreate: true},
	}
	_, err := uc.Execute(context.Background(), validCreateDTO())
	if err == nil {
		t.Fatal("expected repo error, got nil")
	}
}

func TestGetNFSe_NotFound(t *testing.T) {
	uc := &GetNFSeUseCase{
		Repo: &fakeNFSeRepo{errGetID: errors.New("not found")},
		Auth: &fakeNFSeAuth{canGet: true},
	}
	_, err := uc.Execute(context.Background(), 999)
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestListNFSe_Unauthorized(t *testing.T) {
	uc := &ListNFSeUseCase{
		Repo: &fakeNFSeRepo{},
		Auth: &fakeNFSeAuth{canGet: false},
	}
	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

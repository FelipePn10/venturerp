package shipping_carrier_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/repository"
	"github.com/shopspring/decimal"
)

// repoFalso implementa só o que as travas usam; o resto entra em pânico se
// for chamado por engano.
type repoFalso struct {
	domrepo.Repository
	atual    *entity.Transportadora
	usos     []domrepo.Uso
	placas   []string
	salvou   bool
	excluiu  bool
	situacao *bool
	lista    []*entity.Transportadora
}

func (r *repoFalso) Listar(context.Context, domrepo.Filtro) ([]*entity.Transportadora, error) {
	return r.lista, nil
}

func (r *repoFalso) Obter(context.Context, int64) (*entity.Transportadora, error) {
	c := *r.atual
	return &c, nil
}
func (r *repoFalso) Uso(context.Context, int64) ([]domrepo.Uso, error)    { return r.usos, nil }
func (r *repoFalso) PlacasEmUso(context.Context, int64) ([]string, error) { return r.placas, nil }
func (r *repoFalso) ListarTiposVeiculo(context.Context) ([]*domrepo.TipoVeiculo, error) {
	return nil, nil
}
func (r *repoFalso) FornecedorExiste(context.Context, int64) (string, string, bool, error) {
	return "Transportes Teste", "", true, nil
}
func (r *repoFalso) Salvar(_ context.Context, t *entity.Transportadora) (*entity.Transportadora, error) {
	r.salvou = true
	return t, nil
}
func (r *repoFalso) HojeDaEmpresa(context.Context) (time.Time, error) {
	return time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC), nil
}
func (r *repoFalso) Excluir(context.Context, int64) error { r.excluiu = true; return nil }
func (r *repoFalso) DefinirSituacao(_ context.Context, _ int64, ativa bool) error {
	r.situacao = &ativa
	return nil
}

func novoRepo() *repoFalso {
	return &repoFalso{atual: &entity.Transportadora{ID: 7, SupplierCode: 100, Modal: entity.ModalAereo, IsActive: true,
		Vehicles: []*entity.Veiculo{{Plate: "ABC1D23", IsActive: true}}}}
}

func dtoBase() *request.SalvarTransportadoraDTO {
	ativo := true
	return &request.SalvarTransportadoraDTO{SupplierCode: 100, Modal: "AEREO", IsActive: &ativo,
		Vehicles: []request.TransportadoraVeiculoDTO{{Plate: "ABC1D23"}}}
}

func TestNaoTrocaFornecedorDeTransportadoraUsada(t *testing.T) {
	repo := novoRepo()
	repo.usos = []domrepo.Uso{{Documento: "notas fiscais", Total: 3}}
	dto := dtoBase()
	dto.SupplierCode = 200
	_, err := (&UseCase{Repo: repo}).Salvar(context.Background(), 7, dto)
	if err == nil || !strings.Contains(err.Error(), "3 notas fiscais") || repo.salvou {
		t.Fatalf("troca de fornecedor de transportadora usada passou: %v", err)
	}
	// Sem uso, a troca é permitida.
	repo.usos = nil
	if _, err := (&UseCase{Repo: repo}).Salvar(context.Background(), 7, dto); err != nil {
		t.Fatal(err)
	}
}

func TestNaoInativaComDocumentoEmAberto(t *testing.T) {
	repo := novoRepo()
	repo.usos = []domrepo.Uso{{Documento: "cargas", Total: 5, Abertos: 1}, {Documento: "notas fiscais", Total: 9}}
	uc := &UseCase{Repo: repo}
	err := uc.DefinirSituacao(context.Background(), 7, false)
	if err == nil || !strings.Contains(err.Error(), "1 cargas") || repo.situacao != nil {
		t.Fatalf("inativação com carga aberta passou: %v", err)
	}
	// Pelo formulário (is_active=false) vale a mesma regra.
	dto := dtoBase()
	inativo := false
	dto.IsActive = &inativo
	if _, err := uc.Salvar(context.Background(), 7, dto); err == nil || repo.salvou {
		t.Fatal("inativação pelo formulário com carga aberta passou")
	}
	// Só documentos encerrados: inativa.
	repo.usos = []domrepo.Uso{{Documento: "notas fiscais", Total: 9}}
	if err := uc.DefinirSituacao(context.Background(), 7, false); err != nil || repo.situacao == nil || *repo.situacao {
		t.Fatalf("inativação sem documento aberto falhou: %v", err)
	}
}

func TestNaoRemoveVeiculoEmCargaAberta(t *testing.T) {
	repo := novoRepo()
	repo.placas = []string{"ABC1D23"}
	dto := dtoBase()
	dto.Vehicles = nil
	if _, err := (&UseCase{Repo: repo}).Salvar(context.Background(), 7, dto); err == nil || !strings.Contains(err.Error(), "ABC1D23") {
		t.Fatalf("remoção de veículo em carga aberta passou: %v", err)
	}
	inativo := false
	dto.Vehicles = []request.TransportadoraVeiculoDTO{{Plate: "abc-1d23", IsActive: &inativo}}
	if _, err := (&UseCase{Repo: repo}).Salvar(context.Background(), 7, dto); err == nil {
		t.Fatal("inativação de veículo em carga aberta passou")
	}
}

func TestExcluiSoTransportadoraSemUso(t *testing.T) {
	repo := novoRepo()
	repo.usos = []domrepo.Uso{{Documento: "ocorrências de entrega", Total: 1}}
	uc := &UseCase{Repo: repo}
	if err := uc.Excluir(context.Background(), 7); err == nil || repo.excluiu {
		t.Fatal("exclusão de transportadora com histórico passou")
	}
	repo.usos = nil
	if err := uc.Excluir(context.Background(), 7); err != nil || !repo.excluiu {
		t.Fatalf("exclusão de transportadora sem uso falhou: %v", err)
	}
}

func TestDestinoDaCotacao(t *testing.T) {
	casos := []struct {
		nome, uf, cep, querUF string
		erro                  bool
	}{
		{"só CEP deduz a UF", "", "80010-000", "PR", false},
		{"UF e CEP coerentes", "pr", "80010000", "PR", false},
		{"CEP de outro estado", "PR", "01310-100", "", true},
		{"UF inexistente", "XX", "", "", true},
		{"CEP curto", "", "8001", "", true},
		{"nada", "", "", "", true},
	}
	for _, c := range casos {
		carga := entity.Carga{UF: c.uf, CEP: c.cep}
		err := conferirDestino(&carga)
		if (err != nil) != c.erro || (!c.erro && carga.UF != c.querUF) {
			t.Errorf("%s: err=%v uf=%q", c.nome, err, carga.UF)
		}
	}
}

// A mais barata com RNTRC vencido continua listada (o preço é referência),
// mas vai para o fim e não recebe o selo de mais barata nem de mais rápida.
func TestCotacaoNaoIndicaTransportadoraImpedida(t *testing.T) {
	sp := "SP"
	venceu := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	barataVencida := &entity.Transportadora{ID: 1, SupplierName: "Barata", IsActive: true, Modal: entity.ModalAereo,
		FreightKgRate: decimal.NewFromInt(1), ANTTExpiry: &venceu,
		ServiceAreas: []*entity.RegiaoAtendida{{IsActive: true, State: &sp, LeadDays: 1}}}
	cara := &entity.Transportadora{ID: 2, SupplierName: "Cara", IsActive: true, Modal: entity.ModalAereo,
		FreightKgRate: decimal.NewFromInt(2),
		ServiceAreas:  []*entity.RegiaoAtendida{{IsActive: true, State: &sp, LeadDays: 5}}}
	uc := &UseCase{Repo: &repoFalso{lista: []*entity.Transportadora{barataVencida, cara}}}
	out, err := uc.Cotar(context.Background(), entity.Carga{PesoKg: decimal.NewFromInt(10), UF: "SP"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cotacoes) != 2 || out.Cotacoes[0].CarrierID != 2 || out.Cotacoes[1].CarrierID != 1 {
		t.Fatalf("ordem errada: %+v", out.Cotacoes)
	}
	if out.MaisBarata == nil || *out.MaisBarata != 2 || out.MaisRapida == nil || *out.MaisRapida != 2 {
		t.Fatalf("selos para a impedida: barata=%v rápida=%v", out.MaisBarata, out.MaisRapida)
	}
}

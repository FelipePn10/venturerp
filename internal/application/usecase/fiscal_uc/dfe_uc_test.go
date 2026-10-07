package fiscal_uc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type authDFe struct{ ports.AuthService }

func (authDFe) CanCreateFiscalEntry(context.Context) bool { return true }
func (authDFe) CanGetFiscalEntry(context.Context) bool    { return true }

type configDFe struct {
	repository.FiscalRepository
	cfg *entity.FiscalConfig
}

func (c configDFe) GetFiscalConfig(context.Context) (*entity.FiscalConfig, error) { return c.cfg, nil }

type recebidasMem struct {
	versao     int64
	docs       map[string]repository.ReceivedDocument
	manifest   map[string]string
	resultados []*string
	filtro     repository.ReceivedDocumentsFilter
}

func (r *recebidasMem) ReservarSincronizacaoDFe(context.Context, time.Duration) ([]repository.EmpresaDFe, error) {
	return nil, nil
}
func (r *recebidasMem) RegistrarResultadoDFe(_ context.Context, erro *string) error {
	r.resultados = append(r.resultados, erro)
	return nil
}
func (r *recebidasMem) DFeStatus(context.Context, time.Time) (*repository.DFeStatus, error) {
	return &repository.DFeStatus{}, nil
}
func (r *recebidasMem) SetDFeAutomatico(context.Context, bool) error { return nil }

func (r *recebidasMem) DFeVersion(context.Context) (int64, error) { return r.versao, nil }
func (r *recebidasMem) UpsertReceivedDocuments(_ context.Context, docs []repository.ReceivedDocument, max int64) (int, error) {
	novas := 0
	for _, d := range docs {
		if _, ok := r.docs[d.ChaveAcesso]; !ok {
			novas++
		}
		r.docs[d.ChaveAcesso] = d
	}
	if max > r.versao {
		r.versao = max
	}
	return novas, nil
}
func (r *recebidasMem) ListReceivedDocuments(_ context.Context, f repository.ReceivedDocumentsFilter) ([]repository.ReceivedDocument, error) {
	r.filtro = f
	out := []repository.ReceivedDocument{}
	for k, d := range r.docs {
		if f.Busca == "" || f.Busca == k {
			out = append(out, d)
		}
	}
	return out, nil
}
func (r *recebidasMem) SetManifestacao(_ context.Context, chave, tipo string) error {
	r.manifest[chave] = tipo
	return nil
}

// fonteFake simula a Focus: 250 notas em páginas de 100, versões 1..250.
type fonteFake struct {
	total       int
	chamadas    int
	manifestos  []focusnfe.ManifestacaoPayload
	falhaListar error
}

func chaveFake(i int) string { return fmt.Sprintf("4126101234567800019055001%09d1%08d0", i, i) }

func (f *fonteFake) ListarNFesRecebidas(_ context.Context, _ string, versao int64) ([]focusnfe.NFeRecebida, int64, error) {
	f.chamadas++
	if f.falhaListar != nil {
		return nil, 0, f.falhaListar
	}
	out := []focusnfe.NFeRecebida{}
	max := versao
	for i := int(versao) + 1; i <= f.total && len(out) < 100; i++ {
		out = append(out, focusnfe.NFeRecebida{ChaveNFe: chaveFake(i), NomeEmitente: "F", DocumentoEmitente: "12345678000190",
			ValorTotal: json.Number("10.50"), Situacao: "autorizada", Versao: json.Number(fmt.Sprint(i)), DataEmissao: "2026-10-01T10:00:00-03:00"})
		max = int64(i)
	}
	return out, max, nil
}

func (f *fonteFake) ManifestarDestinatario(_ context.Context, p focusnfe.ManifestacaoPayload) (map[string]interface{}, error) {
	f.manifestos = append(f.manifestos, p)
	return map[string]interface{}{}, nil
}

func novoDFe(fonte *fonteFake) (*DFeUseCase, *recebidasMem) {
	token := "tok"
	mem := &recebidasMem{docs: map[string]repository.ReceivedDocument{}, manifest: map[string]string{}}
	return &DFeUseCase{
		Repo:     configDFe{cfg: &entity.FiscalConfig{CnpjEmpresa: "98765432000110", FocusNfeToken: &token}},
		Recebida: mem,
		Auth:     authDFe{},
		Fonte:    func(string, string) RecebidasFonte { return fonte },
	}, mem
}

func TestDFeSincronizaPaginandoEContinuaDaVersao(t *testing.T) {
	fonte := &fonteFake{total: 250}
	uc, mem := novoDFe(fonte)
	r, err := uc.Sincronizar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.Novas != 250 || r.Versao != 250 || fonte.chamadas != 3 {
		t.Fatalf("novas=%d versão=%d chamadas=%d", r.Novas, r.Versao, fonte.chamadas)
	}
	d := mem.docs[chaveFake(7)]
	if d.NumeroNF == nil || *d.NumeroNF != 7 || d.Serie == nil || *d.Serie != "1" || !d.ValorTotal.Equal(d.ValorTotal.Round(2)) {
		t.Fatalf("número/série lidos da chave: %+v", d)
	}
	// Segunda sincronização: nada novo, parte da versão gravada.
	fonte.total = 260
	r, err = uc.Sincronizar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.Novas != 10 || r.Versao != 260 {
		t.Fatalf("incremental: novas=%d versão=%d", r.Novas, r.Versao)
	}
}

func TestDFeErroDaFonteViraErroDeServicoExterno(t *testing.T) {
	uc, mem := novoDFe(&fonteFake{falhaListar: errors.New("CNPJ do emitente não autorizado")})
	_, err := uc.Sincronizar(context.Background())
	var ext *errorsuc.ExternalServiceError
	if !errors.As(err, &ext) {
		t.Fatalf("esperava erro de serviço externo, veio %T %v", err, err)
	}
	// O erro fica registrado: a sincronização agendada não tem quem leia a resposta.
	if len(mem.resultados) != 1 || mem.resultados[0] == nil || !strings.Contains(*mem.resultados[0], "não autorizado") {
		t.Fatalf("erro não registrado: %v", mem.resultados)
	}
}

func TestDFeSucessoLimpaOErroRegistrado(t *testing.T) {
	uc, mem := novoDFe(&fonteFake{total: 3})
	if _, err := uc.Sincronizar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(mem.resultados) != 1 || mem.resultados[0] != nil {
		t.Fatalf("sucesso deveria limpar o erro: %v", mem.resultados)
	}
}

// Prazo de manifestação: emissão + 180 dias. Ciência não encerra o prazo;
// confirmação sim; nota cancelada não precisa.
func TestDFePrazoDeManifestacao(t *testing.T) {
	uc, mem := novoDFe(&fonteFake{})
	hoje := time.Date(2026, 10, 6, 15, 0, 0, 0, time.UTC)
	uc.Agora = func() time.Time { return hoje }
	data := func(s string) *time.Time { d, _ := time.Parse("2006-01-02", s); return &d }
	ciencia, confirmacao := "ciencia", "confirmacao"
	mem.docs["a"] = repository.ReceivedDocument{ChaveAcesso: "a", DataEmissao: data("2026-04-20"), Manifestacao: &ciencia}     // vence 17/10 → 11 dias
	mem.docs["b"] = repository.ReceivedDocument{ChaveAcesso: "b", DataEmissao: data("2026-09-01")}                             // vence 28/02 → 145 dias
	mem.docs["c"] = repository.ReceivedDocument{ChaveAcesso: "c", DataEmissao: data("2026-03-01")}                             // venceu em 28/08
	mem.docs["d"] = repository.ReceivedDocument{ChaveAcesso: "d", DataEmissao: data("2026-04-20"), Manifestacao: &confirmacao} // encerrada
	mem.docs["e"] = repository.ReceivedDocument{ChaveAcesso: "e", DataEmissao: data("2026-03-01"), Situacao: "cancelada"}
	lista, err := uc.Listar(context.Background(), FiltroRecebidas{Prazo: true})
	if err != nil {
		t.Fatal(err)
	}
	if !mem.filtro.SomentePrazo || !mem.filtro.Hoje.Equal(hoje) {
		t.Fatalf("filtro de prazo não repassado ao repositório: %+v", mem.filtro)
	}
	por := map[string]struct {
		dias   *int
		alerta bool
	}{}
	for _, r := range lista {
		por[r.ChaveAcesso] = struct {
			dias   *int
			alerta bool
		}{r.DiasParaPrazo, r.AlertaPrazo}
	}
	if d := por["a"]; d.dias == nil || *d.dias != 11 || !d.alerta {
		t.Fatalf("nota com ciência a 11 dias do prazo: %+v", d)
	}
	if d := por["b"]; d.dias == nil || *d.dias != 145 || d.alerta {
		t.Fatalf("nota recente: %+v", d)
	}
	if d := por["c"]; d.dias == nil || *d.dias >= 0 || !d.alerta {
		t.Fatalf("nota com prazo vencido: %+v", d)
	}
	if d := por["d"]; d.dias != nil || d.alerta {
		t.Fatalf("confirmação encerra o prazo: %+v", d)
	}
	if d := por["e"]; d.dias != nil || d.alerta {
		t.Fatalf("nota cancelada não tem prazo: %+v", d)
	}
}

func TestDFeSemTokenPedeConfiguracao(t *testing.T) {
	uc, _ := novoDFe(&fonteFake{})
	uc.Repo = configDFe{cfg: &entity.FiscalConfig{CnpjEmpresa: "98765432000110"}}
	_, err := uc.Sincronizar(context.Background())
	var v *errorsuc.ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("esperava validação, veio %v", err)
	}
}

func TestDFeManifestacaoValidaTipoEJustificativa(t *testing.T) {
	fonte := &fonteFake{}
	uc, mem := novoDFe(fonte)
	chave := chaveFake(1)
	ctx := context.Background()
	if err := uc.Manifestar(ctx, chave, "aprovar", ""); err == nil {
		t.Fatal("tipo inválido deveria ser recusado")
	}
	if err := uc.Manifestar(ctx, "123", "ciencia", ""); err == nil {
		t.Fatal("chave curta deveria ser recusada")
	}
	if err := uc.Manifestar(ctx, chave, "desconhecimento", "não sei"); err == nil {
		t.Fatal("desconhecimento sem 15 caracteres de justificativa deveria ser recusado")
	}
	if len(fonte.manifestos) != 0 {
		t.Fatal("nada inválido pode chegar à SEFAZ")
	}
	if err := uc.Manifestar(ctx, chave, "Confirmacao", ""); err != nil {
		t.Fatal(err)
	}
	if mem.manifest[chave] != "confirmacao" || fonte.manifestos[0].CNPJ != "98765432000110" {
		t.Fatalf("manifestação não registrada: %+v %+v", mem.manifest, fonte.manifestos)
	}
}

func TestDFeImportarRecusaNotaCancelada(t *testing.T) {
	uc, mem := novoDFe(&fonteFake{})
	uc.ByKey = &ImportNFeByKeyUseCase{}
	chave := chaveFake(3)
	mem.docs[chave] = repository.ReceivedDocument{ChaveAcesso: chave, Situacao: "cancelada"}
	_, err := uc.Importar(context.Background(), chave)
	var v *errorsuc.ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("nota cancelada não pode ser importada: %v", err)
	}
}

// A tela VFIS0620 mandava "CIENCIA"; a Focus só aceita a grafia minúscula.
func TestDFeManifestacaoAceitaGrafiaAntiga(t *testing.T) {
	fonte := &fonteFake{}
	uc, mem := novoDFe(fonte)
	chave := chaveFake(2)
	if err := uc.Manifestar(context.Background(), chave, "CIENCIA", ""); err != nil {
		t.Fatal(err)
	}
	if err := uc.Manifestar(context.Background(), chave, "OPERACAO_NAO_REALIZADA", "mercadoria nunca foi pedida"); err != nil {
		t.Fatal(err)
	}
	if fonte.manifestos[0].Tipo != "ciencia" || fonte.manifestos[1].Tipo != "nao_realizada" || mem.manifest[chave] != "nao_realizada" {
		t.Fatalf("tipos enviados: %+v", fonte.manifestos)
	}
}

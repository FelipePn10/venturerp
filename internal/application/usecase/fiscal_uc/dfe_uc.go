package fiscal_uc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// RecebidasFonte é a fonte das NF-e emitidas contra o CNPJ (Focus NF-e).
// Interface para os testes não dependerem da Focus.
type RecebidasFonte interface {
	ListarNFesRecebidas(ctx context.Context, cnpj string, versao int64) ([]focusnfe.NFeRecebida, int64, error)
	ManifestarDestinatario(ctx context.Context, p focusnfe.ManifestacaoPayload) (map[string]interface{}, error)
}

// DFeUseCase é a caixa de entrada fiscal: as NF-e emitidas contra o CNPJ da
// empresa (distribuição DF-e), a manifestação do destinatário e a importação
// de cada uma como nota de entrada — sem esperar o fornecedor mandar o XML.
type DFeUseCase struct {
	Repo     repository.FiscalRepository
	Recebida repository.ReceivedDocumentsRepository
	ByKey    *ImportNFeByKeyUseCase
	Auth     ports.AuthService
	// Fonte, quando nula, é a Focus NF-e com o token da configuração fiscal.
	Fonte func(token, ambiente string) RecebidasFonte
	// Agora, quando nulo, é time.Now (os testes fixam a data).
	Agora func() time.Time
}

func (uc *DFeUseCase) hoje() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

var tiposManifestacao = map[string]bool{"ciencia": true, "confirmacao": true, "desconhecimento": true, "nao_realizada": true}

func (uc *DFeUseCase) fonte(ctx context.Context) (RecebidasFonte, string, error) {
	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, "", err
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, "", errorsuc.NewValidationError("o token da Focus NF-e não está configurado — informe-o em Configuração Fiscal (VFIS0100)")
	}
	if strings.TrimSpace(cfg.CnpjEmpresa) == "" {
		return nil, "", errorsuc.NewValidationError("o CNPJ da empresa não está na configuração fiscal")
	}
	if uc.Fonte != nil {
		return uc.Fonte(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente), cfg.CnpjEmpresa, nil
	}
	return focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente), cfg.CnpjEmpresa, nil
}

type SincronizacaoDFe struct {
	Recebidas int   `json:"recebidas"`
	Novas     int   `json:"novas"`
	Versao    int64 `json:"versao"`
}

// Sincronizar busca na SEFAZ (via Focus) as notas novas desde a última versão.
// O resultado (sucesso ou o erro da Focus/SEFAZ) fica registrado na
// configuração fiscal para a tela mostrar — a sincronização agendada não tem
// ninguém olhando a resposta.
func (uc *DFeUseCase) Sincronizar(ctx context.Context) (*SincronizacaoDFe, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	out, err := uc.sincronizar(ctx)
	var msg *string
	if err != nil {
		m := err.Error()
		msg = &m
	}
	if errReg := uc.Recebida.RegistrarResultadoDFe(ctx, msg); errReg != nil && err == nil {
		return nil, errReg
	}
	return out, err
}

func (uc *DFeUseCase) sincronizar(ctx context.Context) (*SincronizacaoDFe, error) {
	fonte, cnpj, err := uc.fonte(ctx)
	if err != nil {
		return nil, err
	}
	versao, err := uc.Recebida.DFeVersion(ctx)
	if err != nil {
		return nil, err
	}
	out := &SincronizacaoDFe{Versao: versao}
	// Cada página traz até 100; 50 páginas por vez é folga para qualquer
	// volume diário sem prender a requisição indefinidamente.
	for pagina := 0; pagina < 50; pagina++ {
		lista, max, err := fonte.ListarNFesRecebidas(ctx, cnpj, out.Versao)
		if err != nil {
			return nil, errorsuc.NewExternalServiceError("Focus NF-e (notas recebidas)", err.Error())
		}
		if max < out.Versao {
			max = out.Versao // a versão nunca anda para trás
		}
		docs := make([]repository.ReceivedDocument, 0, len(lista))
		for _, n := range lista {
			if d, ok := documentoRecebido(n); ok {
				docs = append(docs, d)
			}
		}
		novas, err := uc.Recebida.UpsertReceivedDocuments(ctx, docs, max)
		if err != nil {
			return nil, err
		}
		out.Recebidas += len(docs)
		out.Novas += novas
		if max <= out.Versao || len(lista) < 100 {
			out.Versao = max
			break
		}
		out.Versao = max
	}
	return out, nil
}

func documentoRecebido(n focusnfe.NFeRecebida) (repository.ReceivedDocument, bool) {
	chave := soDigitos(n.ChaveNFe)
	if len(chave) != 44 {
		return repository.ReceivedDocument{}, false
	}
	d := repository.ReceivedDocument{
		ChaveAcesso:  chave,
		CNPJEmitente: soDigitos(n.DocumentoEmitente),
		NomeEmitente: strings.TrimSpace(n.NomeEmitente),
		Situacao:     strings.ToLower(strings.TrimSpace(n.Situacao)),
		XMLCompleto:  n.NFeCompleta,
	}
	// Série e número estão dentro da chave (posições 23–25 e 26–34).
	serie := strings.TrimLeft(chave[22:25], "0")
	if serie == "" {
		serie = "0"
	}
	d.Serie = &serie
	if num, err := strconv.ParseInt(chave[25:34], 10, 64); err == nil {
		d.NumeroNF = &num
	}
	if v, err := decimal.NewFromString(n.ValorTotal.String()); err == nil {
		d.ValorTotal = v
	}
	if v, err := n.Versao.Int64(); err == nil {
		d.Versao = v
	}
	if t := lerDataNFe(n.DataEmissao); !t.IsZero() {
		d.DataEmissao = &t
	}
	if n.ManifestacaoDestinatario != nil && *n.ManifestacaoDestinatario != "" {
		m := strings.ToLower(*n.ManifestacaoDestinatario)
		d.Manifestacao = &m
	}
	return d, true
}

// FiltroRecebidas: pendentes = ainda não lançadas; prazo = sem manifestação
// conclusiva com o prazo vencendo ou vencido.
type FiltroRecebidas struct {
	Pendentes bool
	Prazo     bool
	Busca     string
}

// manifestacaoConclusiva: ciência só libera o XML; o prazo de 180 dias se
// encerra com confirmação, desconhecimento ou operação não realizada.
func manifestacaoConclusiva(m *string) bool {
	return m != nil && (*m == "confirmacao" || *m == "desconhecimento" || *m == "nao_realizada")
}

func (uc *DFeUseCase) Listar(ctx context.Context, f FiltroRecebidas) ([]response.NFeRecebidaResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	hoje := uc.hoje()
	docs, err := uc.Recebida.ListReceivedDocuments(ctx, repository.ReceivedDocumentsFilter{
		SomentePendentes: f.Pendentes, SomentePrazo: f.Prazo, Busca: f.Busca, Hoje: hoje,
	})
	if err != nil {
		return nil, err
	}
	dia := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
	out := make([]response.NFeRecebidaResponse, 0, len(docs))
	for _, d := range docs {
		r := response.NFeRecebidaResponse{
			ChaveAcesso: d.ChaveAcesso, CNPJEmitente: d.CNPJEmitente, NomeEmitente: d.NomeEmitente,
			NumeroNF: d.NumeroNF, Serie: d.Serie, ValorTotal: d.ValorTotal.InexactFloat64(), Situacao: d.Situacao,
			Manifestacao: d.Manifestacao, XMLCompleto: d.XMLCompleto, FiscalEntryID: d.FiscalEntryID, SyncedAt: d.SyncedAt,
		}
		if d.DataEmissao != nil {
			s := d.DataEmissao.Format("2006-01-02")
			r.DataEmissao = &s
			if d.Situacao != "cancelada" && !manifestacaoConclusiva(d.Manifestacao) {
				prazo := dia(*d.DataEmissao).AddDate(0, 0, repository.PrazoManifestacaoDias)
				p := prazo.Format("2006-01-02")
				dias := int(prazo.Sub(dia(hoje)).Hours() / 24)
				r.PrazoManifestacao, r.DiasParaPrazo = &p, &dias
				r.AlertaPrazo = dias <= repository.AlertaPrazoDias
			}
		}
		out = append(out, r)
	}
	return out, nil
}

// Manifestar registra a manifestação do destinatário. Desconhecimento e
// operação não realizada exigem justificativa (mínimo 15 caracteres, regra
// da SEFAZ).
func (uc *DFeUseCase) Manifestar(ctx context.Context, chave, tipo, justificativa string) error {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return errorsuc.ErrUnauthorized
	}
	chave = soDigitos(chave)
	tipo = normalizarTipoManifestacao(tipo)
	if len(chave) != 44 {
		return errorsuc.NewValidationError("a chave de acesso deve ter 44 dígitos")
	}
	if !tiposManifestacao[tipo] {
		return errorsuc.NewValidationError("tipo de manifestação inválido: ciencia, confirmacao, desconhecimento ou nao_realizada")
	}
	if (tipo == "desconhecimento" || tipo == "nao_realizada") && len([]rune(strings.TrimSpace(justificativa))) < 15 {
		return errorsuc.NewValidationError("informe a justificativa (pelo menos 15 caracteres)")
	}
	fonte, cnpj, err := uc.fonte(ctx)
	if err != nil {
		return err
	}
	if _, err := fonte.ManifestarDestinatario(ctx, focusnfe.ManifestacaoPayload{CNPJ: cnpj, ChaveNFe: chave, Tipo: tipo, Justificativa: justificativa}); err != nil {
		return errorsuc.NewExternalServiceError("Focus NF-e (manifestação)", err.Error())
	}
	return uc.Recebida.SetManifestacao(ctx, chave, tipo)
}

// Importar baixa o XML da nota recebida e segue o fluxo da nota de entrada.
// Sem ciência da operação a SEFAZ não libera o XML completo: a ciência é
// registrada antes, quando ainda não houve manifestação.
func (uc *DFeUseCase) Importar(ctx context.Context, chave string) (*response.FiscalEntryResponse, error) {
	if uc.ByKey == nil {
		return nil, fmt.Errorf("importação por chave não configurada")
	}
	chave = soDigitos(chave)
	docs, err := uc.Recebida.ListReceivedDocuments(ctx, repository.ReceivedDocumentsFilter{Busca: chave, Limite: 1})
	if err != nil {
		return nil, err
	}
	if len(docs) == 1 && docs[0].Situacao == "cancelada" {
		return nil, errorsuc.NewValidationError("esta NF-e foi cancelada pelo emitente e não pode ser lançada")
	}
	if len(docs) == 1 && docs[0].Manifestacao == nil && !docs[0].XMLCompleto {
		if err := uc.Manifestar(ctx, chave, "ciencia", ""); err != nil {
			return nil, err
		}
	}
	return uc.ByKey.Execute(ctx, ImportNFeByKeyDTO{ChaveAcesso: chave})
}

// StatusDFe: última sincronização, erro da última tentativa e o resumo dos
// prazos de manifestação.
func (uc *DFeUseCase) Status(ctx context.Context) (*response.DFeStatusResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	st, err := uc.Recebida.DFeStatus(ctx, uc.hoje())
	if err != nil {
		return nil, err
	}
	return &response.DFeStatusResponse{
		Automatico: st.Automatico, SincronizadoEm: st.SincronizadoEm, UltimaTentativa: st.UltimaTentativa,
		UltimoErro: st.UltimoErro, PrazoProximo: st.PrazoProximo, PrazoVencido: st.PrazoVencido,
		IntervaloMinutos: int(IntervaloSincronizacaoDFe.Minutes()), AlertaPrazoDias: repository.AlertaPrazoDias,
	}, nil
}

// SetAutomatico liga ou desliga a sincronização agendada da empresa.
func (uc *DFeUseCase) SetAutomatico(ctx context.Context, ativo bool) error {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Recebida.SetDFeAutomatico(ctx, ativo)
}

// IntervaloSincronizacaoDFe: a SEFAZ limita as consultas de distribuição (o
// excesso bloqueia o CNPJ por uma hora); uma por hora fica bem abaixo.
const IntervaloSincronizacaoDFe = time.Hour

// Package portal_uc é o Portal do Cliente, fase 0 (.ai/tasks/portal-do-cliente.md):
// o vendedor gera um link por cliente; com ele, o cliente vê os próprios pedidos e
// notas e pede a repetição de um pedido, que vira rascunho para o comercial confirmar.
package portal_uc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
)

// ErrNaoEncontrado: link inexistente, vencido ou revogado, ou pedido de outro
// cliente — a resposta é a mesma para não revelar o que existe.
var ErrNaoEncontrado = errorsuc.NewNotFoundError("link inválido ou vencido: peça um novo à empresa")

type Link struct {
	ID           int64      `json:"id"`
	CustomerCode int64      `json:"customer_code"`
	Descricao    string     `json:"description,omitempty"`
	ExpiraEm     time.Time  `json:"expires_at"`
	RevogadoEm   *time.Time `json:"revoked_at,omitempty"`
	UltimoAcesso *time.Time `json:"last_access_at,omitempty"`
	Acessos      int        `json:"access_count"`
	CriadoEm     time.Time  `json:"created_at"`
	// Token e URL só voltam na criação: o banco guarda apenas o hash.
	Token string `json:"token,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Sessao é o que o token abre: a empresa, o cliente e quem assina o que o portal grava.
type Sessao struct {
	LinkID         int64
	EnterpriseID   int64
	EnterpriseCode int64
	Empresa        string
	CustomerCode   int64
	Cliente        string
	ExpiraEm       time.Time
	Ator           uuid.UUID
}

type Linha struct {
	Item          string          `json:"item"`
	Descricao     string          `json:"description"`
	Mascara       string          `json:"mask,omitempty"`
	Unidade       string          `json:"uom,omitempty"`
	Quantidade    decimal.Decimal `json:"quantity"`
	Atendido      decimal.Decimal `json:"delivered"`
	PrecoUnitario decimal.Decimal `json:"unit_price"`
	Total         decimal.Decimal `json:"total"`
	Entrega       *time.Time      `json:"delivery_date,omitempty"`
	// APreco: item pedido pelo portal que o comercial ainda vai precificar.
	APreco bool `json:"to_be_priced,omitempty"`
}

type Nota struct {
	ID       int64           `json:"id"`
	Numero   int64           `json:"number"`
	Serie    string          `json:"series,omitempty"`
	Chave    string          `json:"key"`
	Emissao  time.Time       `json:"issued_at"`
	Valor    decimal.Decimal `json:"value"`
	Situacao string          `json:"status"`
	// Documentos: a DANFE/XML oficiais existem (ou dá para buscar na emissora).
	Documentos bool `json:"documents_available"`
}

// Titulo é uma parcela a pagar do pedido (o boleto/PIX vai junto quando emitido).
type Titulo struct {
	Documento      string          `json:"document"`
	Parcela        string          `json:"installment,omitempty"`
	Vencimento     time.Time       `json:"due_date"`
	Valor          decimal.Decimal `json:"value"`
	Saldo          decimal.Decimal `json:"balance"`
	Situacao       string          `json:"status"`
	LinhaDigitavel string          `json:"digitable_line,omitempty"`
	PixCopiaECola  string          `json:"pix_copy_paste,omitempty"`
}

// Entrega é um romaneio do pedido: separado, conferido, despachado.
type Entrega struct {
	Romaneio       int64      `json:"shipment"`
	Situacao       string     `json:"status"`
	DespachadoEm   *time.Time `json:"shipped_at,omitempty"`
	Transportadora string     `json:"carrier,omitempty"`
	NotaNumero     *int64     `json:"invoice_number,omitempty"`
}

type Pedido struct {
	Code            int64           `json:"code"`
	Numero          int64           `json:"number"`
	Situacao        string          `json:"status"`
	Emissao         *time.Time      `json:"issued_at,omitempty"`
	Entrega         *time.Time      `json:"delivery_date,omitempty"`
	Total           decimal.Decimal `json:"total"`
	PedidoDoCliente string          `json:"customer_po,omitempty"`
	Origem          string          `json:"origin,omitempty"`
	Condicao        string          `json:"payment_terms,omitempty"`
	Frete           string          `json:"freight,omitempty"`
	Transportadora  string          `json:"carrier,omitempty"`
	LocalEntrega    string          `json:"delivery_address,omitempty"`
	Observacao      string          `json:"note,omitempty"`
	Linhas          []Linha         `json:"lines"`
	Notas           []Nota          `json:"invoices"`
	Titulos         []Titulo        `json:"installments"`
	Entregas        []Entrega       `json:"shipments"`
}

type Resumo struct {
	Empresa  string    `json:"company"`
	Cliente  string    `json:"customer"`
	ExpiraEm time.Time `json:"link_expires_at"`
	Pedidos  []Pedido  `json:"orders"`
	// AoPagar: soma dos saldos em aberto (para o cartão do topo).
	AoPagar decimal.Decimal `json:"open_balance"`
	Vencido decimal.Decimal `json:"overdue_balance"`
}

// MaxPendentesDoPortal: rascunhos do portal aguardando confirmação por cliente.
const MaxPendentesDoPortal = 5

type Repo interface {
	CriarLink(ctx context.Context, cliente int64, hash []byte, expira time.Time, descricao string, autor uuid.UUID) (int64, error)
	ListarLinks(ctx context.Context, cliente int64) ([]Link, error)
	RevogarLink(ctx context.Context, id int64) error
	SessaoPorToken(ctx context.Context, hash []byte) (*Sessao, error)
	Evento(ctx context.Context, s *Sessao, acao, detalhe string, pedido, origem *int64, ip, agente string) error
	Pedidos(ctx context.Context, s *Sessao) ([]Pedido, error)
	PedidoDoCliente(ctx context.Context, s *Sessao, code int64) (bool, error)
	PendentesDoPortal(ctx context.Context, s *Sessao, origem int64) (int, int64, error)
	NotaDoCliente(ctx context.Context, s *Sessao, nota int64) (int64, bool, error)
	AvisarEmpresa(ctx context.Context, s *Sessao, novo, numero, original int64, observacao string) (string, error)
}

// Documentos: os endereços da DANFE e do XML oficiais da nota (a mesma consulta
// da tela fiscal, que busca na emissora quando ainda não estão gravados).
type Documentos interface {
	Execute(ctx context.Context, exitID int64) (*fiscal_uc.DANFEResult, error)
}

type Repetidor interface {
	RepetirComOrigem(ctx context.Context, code int64, origem, observacao string) (*sales_order_uc.RepetirResultado, error)
}

type UseCase struct {
	Repo    Repo
	Auth    ports.AuthService
	Repetir Repetidor
	BaseURL string // PORTAL_BASE_URL: monta o link completo; vazio = só o token
	Empresa string // PORTAL_EMPRESA: id no catálogo de empresas; vai no link
	Agora   func() time.Time
	// Documentos e Email são opcionais: sem eles o download e o e-mail ao
	// representante não acontecem (a pendência e o alerta continuam).
	Documentos Documentos
	Email      ports.EmailProvider
	// XMLGuardado: a cópia do XML autorizado que o ERP guarda. Opcional: sem
	// ela o XML vem da emissora, como a DANFE.
	XMLGuardado XMLGuardado
}

type XMLGuardado interface {
	ExitXML(ctx context.Context, id int64) (autorizado, cancelamento string, err error)
}

var travas sync.Map // "empresa:cliente" -> *sync.Mutex

func (uc *UseCase) travaDoCliente(empresa, cliente int64) *sync.Mutex {
	m, _ := travas.LoadOrStore(fmt.Sprintf("%d:%d", empresa, cliente), &sync.Mutex{})
	return m.(*sync.Mutex)
}

func (uc *UseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

// Hash é o que o banco guarda do token.
func Hash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// CriarLink gera o token (32 bytes aleatórios) do cliente, válido por dias (1 a 365).
func (uc *UseCase) CriarLink(ctx context.Context, cliente int64, dias int, descricao string) (*Link, error) {
	// Sem o endereço público (PORTAL_BASE_URL) o link não teria onde abrir.
	if strings.TrimSpace(uc.BaseURL) == "" {
		return nil, errorsuc.NewValidationError("o portal do cliente não está ligado neste servidor (endereço público não configurado): fale com o suporte")
	}
	if dias <= 0 {
		dias = 30
	}
	if dias > 365 {
		return nil, errorsuc.NewValidationError("o link vale no máximo 365 dias")
	}
	descricao = strings.TrimSpace(descricao)
	if utf8.RuneCountInString(descricao) > 120 {
		return nil, errorsuc.NewValidationError("a descrição do link tem no máximo 120 caracteres")
	}
	autor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("gerando o token do portal: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	expira := uc.agora().Add(time.Duration(dias) * 24 * time.Hour)
	id, err := uc.Repo.CriarLink(ctx, cliente, Hash(token), expira, descricao, autor)
	if err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d não encontrado nesta empresa ou inativo: só cliente ativo recebe link do portal", cliente))
		}
		return nil, err
	}
	l := &Link{ID: id, CustomerCode: cliente, Descricao: descricao, ExpiraEm: expira, CriadoEm: uc.agora(), Token: token}
	if base := strings.TrimRight(uc.BaseURL, "/"); base != "" {
		l.URL = base + "/#/portal/" + token
		if uc.Empresa != "" {
			l.URL = base + "/#/portal/" + url.PathEscape(uc.Empresa) + "/" + token
		}
	}
	return l, nil
}

func (uc *UseCase) ListarLinks(ctx context.Context, cliente int64) ([]Link, error) {
	l, err := uc.Repo.ListarLinks(ctx, cliente)
	if l == nil {
		l = []Link{}
	}
	return l, err
}

func (uc *UseCase) RevogarLink(ctx context.Context, id int64) error {
	if err := uc.Repo.RevogarLink(ctx, id); err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			return errorsuc.NewNotFoundError("link não encontrado")
		}
		return err
	}
	return nil
}

// Abrir resolve o token. Sem ADMIN ativo na empresa o portal não grava nada.
func (uc *UseCase) Abrir(ctx context.Context, token, ip, agente string) (*Sessao, error) {
	token = strings.TrimSpace(token)
	if len(token) < 20 || len(token) > 100 {
		return nil, ErrNaoEncontrado
	}
	s, err := uc.Repo.SessaoPorToken(ctx, Hash(token))
	if err != nil {
		return nil, err
	}
	if s.Ator == uuid.Nil {
		return nil, fmt.Errorf("a empresa %d não tem ADMIN ativo para assinar o portal", s.EnterpriseID)
	}
	return s, nil
}

// Resumo: o que o cliente vê. ctx já é o da empresa do link.
func (uc *UseCase) Resumo(ctx context.Context, s *Sessao, ip, agente string) (*Resumo, error) {
	p, err := uc.Repo.Pedidos(ctx, s)
	if err != nil {
		return nil, err
	}
	if p == nil {
		p = []Pedido{}
	}
	for i := range p {
		if p[i].Linhas == nil {
			p[i].Linhas = []Linha{}
		}
		if p[i].Notas == nil {
			p[i].Notas = []Nota{}
		}
		if p[i].Titulos == nil {
			p[i].Titulos = []Titulo{}
		}
		if p[i].Entregas == nil {
			p[i].Entregas = []Entrega{}
		}
	}
	_ = uc.Repo.Evento(ctx, s, "ACESSO", "", nil, nil, ip, agente) // auditoria não derruba a consulta
	out := &Resumo{Empresa: s.Empresa, Cliente: s.Cliente, ExpiraEm: s.ExpiraEm, Pedidos: p}
	hoje := uc.agora().Truncate(24 * time.Hour)
	for _, pe := range p {
		for _, t := range pe.Titulos {
			if t.Saldo.IsPositive() {
				out.AoPagar = out.AoPagar.Add(t.Saldo)
				if t.Vencimento.Before(hoje) {
					out.Vencido = out.Vencido.Add(t.Saldo)
				}
			}
		}
	}
	return out, nil
}

// PedirDeNovo cria o rascunho (origem PORTAL) com as linhas do pedido e o preço
// de hoje. O comercial confirma pela Central de Pendências.
func (uc *UseCase) PedirDeNovo(ctx context.Context, s *Sessao, code int64, observacao, ip, agente string) (*sales_order_uc.RepetirResultado, error) {
	observacao = strings.TrimSpace(observacao)
	if utf8.RuneCountInString(observacao) > 500 {
		return nil, errorsuc.NewValidationError("a observação tem no máximo 500 caracteres")
	}
	ok, err := uc.Repo.PedidoDoCliente(ctx, s, code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNaoEncontrado
	}
	// O portal pede, a empresa confirma: sem limite, um link vazado (ou um
	// clique repetido) enche a carteira de rascunhos. A trava por cliente faz a
	// conferência e a gravação valerem juntas (dois cliques no mesmo instante).
	// shortcut: trava no processo; vale porque cada empresa roda uma API só. Com réplicas, trocar por pg_advisory_xact_lock.
	trava := uc.travaDoCliente(s.EnterpriseID, s.CustomerCode)
	trava.Lock()
	defer trava.Unlock()
	abertos, mesmo, err := uc.Repo.PendentesDoPortal(ctx, s, code)
	if err != nil {
		return nil, err
	}
	if mesmo > 0 {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("você já pediu este pedido de novo (pedido %d) e a empresa ainda vai confirmar: se precisar mudar algo, fale com o seu vendedor", mesmo))
	}
	if abertos >= MaxPendentesDoPortal {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("você já tem %d pedidos aguardando a confirmação da empresa: espere a confirmação ou fale com o seu vendedor", abertos))
	}
	res, err := uc.Repetir.RepetirComOrigem(ctx, code, "PORTAL", observacao)
	if err != nil {
		return nil, err
	}
	novo := res.Pedido.Code
	_ = uc.Repo.Evento(ctx, s, "PEDIDO_REPETIDO", fmt.Sprintf("repetição do pedido %d (%d linha(s))", code, res.Linhas), &novo, &code, ip, agente)
	// A empresa fica sabendo por três caminhos: o cartão da Central de
	// Pendências (sempre), o alerta para quem assina e o e-mail ao representante.
	// Falha de aviso não desfaz o pedido: o rascunho já está gravado.
	if email, err := uc.Repo.AvisarEmpresa(ctx, s, novo, res.Pedido.OrderNumber, code, observacao); err != nil {
		slog.Error("pedido do portal gravado, mas o alerta falhou", "pedido", novo, "erro", err)
	} else if email != "" && uc.Email != nil {
		texto := fmt.Sprintf("%s pediu pelo portal a repetição do pedido de origem %d.\n\nO pedido %d foi criado como RASCUNHO e aguarda a sua confirmação no ERP (Pedido de Venda, VVND0200).", s.Cliente, code, res.Pedido.OrderNumber)
		if observacao != "" {
			texto += "\n\nObservação do cliente: " + observacao
		}
		if err := uc.Email.Send(ctx, ports.EmailMessage{EnterpriseID: s.EnterpriseID, To: []string{email}, Subject: fmt.Sprintf("Pedido pelo portal — %s (pedido %d)", s.Cliente, res.Pedido.OrderNumber), Text: texto}); err != nil {
			slog.Warn("e-mail do pedido do portal ao representante não saiu", "pedido", novo, "erro", err)
		}
	}
	return res, nil
}

// XMLDaNota devolve a cópia guardada do XML autorizado da nota do cliente;
// conteúdo vazio quando ainda não há cópia (o XML então vem da emissora).
func (uc *UseCase) XMLDaNota(ctx context.Context, s *Sessao, nota int64, ip, agente string) (string, string, error) {
	if uc.XMLGuardado == nil {
		return "", "", nil
	}
	numero, ok, err := uc.Repo.NotaDoCliente(ctx, s, nota)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", ErrNaoEncontrado
	}
	xml, _, err := uc.XMLGuardado.ExitXML(ctx, nota)
	if err != nil || xml == "" {
		return "", "", err
	}
	_ = uc.Repo.Evento(ctx, s, "DOWNLOAD_XML", fmt.Sprintf("NF-e %d", numero), nil, nil, ip, agente)
	return xml, fmt.Sprintf("NF-e %d.xml", numero), nil
}

// Documento devolve o endereço da DANFE (pdf) ou do XML da nota do cliente.
func (uc *UseCase) Documento(ctx context.Context, s *Sessao, nota int64, tipo, ip, agente string) (string, string, error) {
	numero, ok, err := uc.Repo.NotaDoCliente(ctx, s, nota)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", ErrNaoEncontrado
	}
	if uc.Documentos == nil {
		return "", "", errorsuc.NewValidationError("o download de notas não está disponível neste servidor")
	}
	d, err := uc.Documentos.Execute(ctx, nota)
	if err != nil {
		return "", "", errorsuc.NewValidationError("a DANFE e o XML desta nota ainda não estão disponíveis: peça à empresa")
	}
	url, nome := d.DanfeURL, fmt.Sprintf("NF-e %d.pdf", numero)
	if tipo == "xml" {
		url, nome = d.XMLURL, fmt.Sprintf("NF-e %d.xml", numero)
	}
	if url == "" {
		return "", "", errorsuc.NewValidationError("este documento da nota ainda não está disponível: peça à empresa")
	}
	_ = uc.Repo.Evento(ctx, s, "DOWNLOAD_"+strings.ToUpper(tipo), fmt.Sprintf("NF-e %d", numero), nil, nil, ip, agente)
	return url, nome, nil
}

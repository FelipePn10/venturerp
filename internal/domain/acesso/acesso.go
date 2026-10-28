// Package acesso modela os perfis de acesso da empresa: o que cada perfil pode
// fazer em cada módulo do ERP.
//
// O papel do usuário (ADMIN, USER, OPERATOR, VIEWER) continua valendo. O perfil
// restringe dentro dele: usuário com perfil na empresa só faz o que algum dos
// seus perfis permite; usuário sem perfil segue só com o papel, como antes.
// ADMIN não é restringido — é quem administra os perfis.
package acesso

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/google/uuid"
)

// Modulo é uma área do ERP — as mesmas do menu do desktop.
type Modulo string

const (
	Comercial     Modulo = "comercial"
	Suprimentos   Modulo = "suprimentos"
	Qualidade     Modulo = "qualidade"
	Estoque       Modulo = "estoque"
	Engenharia    Modulo = "engenharia"
	Planejamento  Modulo = "planejamento"
	Producao      Modulo = "producao"
	Manutencao    Modulo = "manutencao"
	Assistencia   Modulo = "assistencia"
	Fiscal        Modulo = "fiscal"
	Financeiro    Modulo = "financeiro"
	Contabilidade Modulo = "contabilidade"
	Custos        Modulo = "custos"
	Cadastros     Modulo = "cadastros"
	Administracao Modulo = "administracao"
)

// Acao é o que se faz no módulo.
type Acao string

const (
	Ler            Acao = "LER"
	IncluirAlterar Acao = "INCLUIR_ALTERAR"
	Excluir        Acao = "EXCLUIR"
	// Aprovar: alçadas e liberações (pedido de compra, nota de entrada,
	// título a pagar, liberação comercial do pedido).
	Aprovar Acao = "APROVAR"
	// Autorizar: transmitir ou cancelar documento na SEFAZ/prefeitura.
	Autorizar Acao = "AUTORIZAR"
	// Executar: rodar cálculos que mexem no plano (MRP, CRP, cenários).
	Executar Acao = "EXECUTAR"
)

// ModuloInfo descreve o módulo e as ações que fazem sentido nele.
type ModuloInfo struct {
	Modulo Modulo
	Nome   string
	Acoes  []Acao
}

var basicas = []Acao{Ler, IncluirAlterar, Excluir}

func com(extras ...Acao) []Acao { return append(append([]Acao{}, basicas...), extras...) }

// Catalogo é a lista de módulos e ações que um perfil pode receber.
var Catalogo = []ModuloInfo{
	{Comercial, "Comercial (clientes, orçamentos, pedidos, representantes)", com(Aprovar)},
	{Suprimentos, "Suprimentos (compras, fornecedores, cotações, importação)", com(Aprovar)},
	{Qualidade, "Qualidade e inspeção", com()},
	{Estoque, "Estoque, almoxarifado e expedição", com()},
	{Engenharia, "Engenharia (itens, estruturas, roteiros, configurador)", com()},
	{Planejamento, "Planejamento (MRP, CRP, previsão, demandas)", com(Executar)},
	{Producao, "Produção (ordens, plano de corte, APS)", com()},
	{Manutencao, "Manutenção", com()},
	{Assistencia, "Assistência técnica e SAC", com()},
	{Fiscal, "Fiscal (NF-e, CT-e, NFS-e, SPED, parâmetros)", com(Aprovar, Autorizar)},
	{Financeiro, "Financeiro (contas a pagar e receber, bancos)", com(Aprovar)},
	{Contabilidade, "Contabilidade", com()},
	{Custos, "Custos e margem", com()},
	{Cadastros, "Cadastros gerais (empresa, funcionários, localização)", com()},
	{Administracao, "Administração (auditoria, atualização, senhas)", com(Aprovar)},
}

// InfoDo devolve o módulo do catálogo.
func InfoDo(m Modulo) (ModuloInfo, bool) {
	for _, i := range Catalogo {
		if i.Modulo == m {
			return i, true
		}
	}
	return ModuloInfo{}, false
}

// RotuloAcao é o nome da ação para a mensagem ao usuário.
func RotuloAcao(a Acao) string {
	switch a {
	case Ler:
		return "consultar"
	case IncluirAlterar:
		return "incluir ou alterar"
	case Excluir:
		return "excluir"
	case Aprovar:
		return "aprovar"
	case Autorizar:
		return "autorizar documentos fiscais"
	case Executar:
		return "executar cálculos"
	}
	return strings.ToLower(string(a))
}

// Permissoes é o conjunto módulo → ações.
type Permissoes map[Modulo]map[Acao]bool

// Pode diz se o conjunto permite a ação no módulo.
func (p Permissoes) Pode(m Modulo, a Acao) bool { return p[m][a] }

// Unir soma as permissões de outro perfil.
func (p Permissoes) Unir(o Permissoes) {
	for m, acoes := range o {
		if p[m] == nil {
			p[m] = map[Acao]bool{}
		}
		for a, ok := range acoes {
			if ok {
				p[m][a] = true
			}
		}
	}
}

// Lista devolve as permissões ordenadas (para resposta e comparação).
func (p Permissoes) Lista() []Permissao {
	out := []Permissao{}
	for m, acoes := range p {
		for a, ok := range acoes {
			if ok {
				out = append(out, Permissao{Modulo: m, Acao: a})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Modulo != out[j].Modulo {
			return out[i].Modulo < out[j].Modulo
		}
		return out[i].Acao < out[j].Acao
	})
	return out
}

// Permissao é um par módulo × ação.
type Permissao struct {
	Modulo Modulo
	Acao   Acao
}

// NovasPermissoes valida a lista contra o catálogo. Quem pode alterar, excluir,
// aprovar, autorizar ou executar também consulta: sem LER a tela nem abre.
func NovasPermissoes(lista []Permissao) (Permissoes, error) {
	p := Permissoes{}
	for _, item := range lista {
		info, ok := InfoDo(item.Modulo)
		if !ok {
			return nil, domainerr.NewValidation(fmt.Sprintf("módulo %q não existe", item.Modulo))
		}
		valida := false
		for _, a := range info.Acoes {
			if a == item.Acao {
				valida = true
				break
			}
		}
		if !valida {
			return nil, domainerr.NewValidation(fmt.Sprintf("a ação %q não existe no módulo %s", item.Acao, info.Nome))
		}
		if p[item.Modulo] == nil {
			p[item.Modulo] = map[Acao]bool{}
		}
		p[item.Modulo][item.Acao] = true
		p[item.Modulo][Ler] = true
	}
	return p, nil
}

// Perfil é um conjunto de permissões da empresa.
type Perfil struct {
	ID           int64
	EnterpriseID int64
	Nome         string
	Descricao    *string
	Ativo        bool
	Permissoes   Permissoes
	Usuarios     int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	UpdatedBy    uuid.UUID
}

// Validar normaliza nome e descrição.
func (p *Perfil) Validar() error {
	p.Nome = strings.TrimSpace(p.Nome)
	if p.Nome == "" {
		return domainerr.NewValidation("informe o nome do perfil")
	}
	if len([]rune(p.Nome)) > 80 {
		return domainerr.NewValidation("o nome do perfil tem no máximo 80 caracteres")
	}
	if p.Descricao != nil {
		d := strings.TrimSpace(*p.Descricao)
		if d == "" {
			p.Descricao = nil
		} else {
			p.Descricao = &d
		}
	}
	if p.Permissoes == nil {
		p.Permissoes = Permissoes{}
	}
	return nil
}

// AcessoDoUsuario é o que vale para o usuário na empresa.
type AcessoDoUsuario struct {
	// Restrito é verdadeiro quando o usuário tem perfil ativo na empresa (e não
	// é ADMIN): só vale o que os perfis permitem.
	Restrito   bool
	Perfis     []string
	Permissoes Permissoes
}

// Pode aplica a regra: sem restrição, vale o papel (decidido em outro lugar).
func (a AcessoDoUsuario) Pode(m Modulo, acao Acao) bool {
	return !a.Restrito || a.Permissoes.Pode(m, acao)
}

// UsuarioDaEmpresa é o usuário com os perfis que tem na empresa.
type UsuarioDaEmpresa struct {
	UserID uuid.UUID
	Nome   string
	Email  string
	Papel  string
	Ativo  bool
	Perfis []int64
}

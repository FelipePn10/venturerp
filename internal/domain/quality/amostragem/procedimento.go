package amostragem

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// TipoProcedimento: pela norma, plano fixo da empresa ou inspeção 100 %.
type TipoProcedimento string

const (
	TipoNorma TipoProcedimento = "NORMA"
	TipoFixo  TipoProcedimento = "FIXO"
	TipoCem   TipoProcedimento = "CEM"
)

// Procedimento de amostragem da empresa.
type Procedimento struct {
	ID              int64            `json:"id"`
	Codigo          string           `json:"codigo"`
	Descricao       string           `json:"descricao"`
	Tipo            TipoProcedimento `json:"tipo"`
	Nivel           Nivel            `json:"nivel,omitempty"`
	NQA             string           `json:"nqa,omitempty"`
	PermiteAtenuada bool             `json:"permite_atenuada"`
	AmostraFixa     *int64           `json:"amostra_fixa,omitempty"`
	AcFixo          *int             `json:"ac_fixo,omitempty"`
	ReFixo          *int             `json:"re_fixo,omitempty"`
	Skip            SkipLot          `json:"skip_lot"`
	Ativo           bool             `json:"ativo"`
	UpdatedAt       time.Time        `json:"updated_at"`
	UpdatedBy       uuid.UUID        `json:"updated_by"`
}

// Normalizar limpa o que não pertence ao tipo e padroniza o NQA.
func (p *Procedimento) Normalizar() {
	p.Codigo = strings.ToUpper(strings.TrimSpace(p.Codigo))
	p.Descricao = strings.TrimSpace(p.Descricao)
	switch p.Tipo {
	case TipoNorma:
		if j := indiceDoNQA(p.NQA); j >= 0 {
			p.NQA = NQAs[j]
		}
		p.AmostraFixa, p.AcFixo, p.ReFixo = nil, nil, nil
	case TipoFixo:
		p.Nivel, p.NQA, p.PermiteAtenuada = "", "", false
	case TipoCem:
		p.Nivel, p.NQA, p.PermiteAtenuada = "", "", false
		p.AmostraFixa, p.AcFixo, p.ReFixo = nil, nil, nil
		p.Skip = SkipLot{}
	}
}

// Validar confere o procedimento antes de gravar.
func (p Procedimento) Validar() error {
	if p.Codigo == "" || len(p.Codigo) > 20 {
		return domainerr.NewValidation("informe o código do procedimento (até 20 caracteres)")
	}
	if p.Descricao == "" {
		return domainerr.NewValidation("informe a descrição do procedimento")
	}
	switch p.Tipo {
	case TipoNorma:
		if _, ok := colunaDoNivel[p.Nivel]; !ok {
			return domainerr.NewValidation("informe o nível de inspeção (S1, S2, S3, S4, I, II ou III)")
		}
		if indiceDoNQA(p.NQA) < 0 {
			return domainerr.NewValidation(fmt.Sprintf("NQA %q não é um valor da norma (%s)", p.NQA, strings.Join(NQAs, ", ")))
		}
	case TipoFixo:
		if p.AmostraFixa == nil || *p.AmostraFixa <= 0 || p.AcFixo == nil || *p.AcFixo < 0 || p.ReFixo == nil || *p.ReFixo <= *p.AcFixo {
			return domainerr.NewValidation("plano fixo exige amostra > 0, Ac ≥ 0 e Re > Ac")
		}
	case TipoCem:
	default:
		return domainerr.NewValidation("tipo de procedimento inválido (NORMA, FIXO ou CEM)")
	}
	if (p.Skip.AposAceitos != 0 || p.Skip.Frequencia != 0) && !p.Skip.Ativo() {
		return domainerr.NewValidation("skip-lot exige quantidade de lotes aceitos > 0 e frequência > 1")
	}
	return nil
}

// TamanhoDoLote converte a quantidade do lote em unidades para a amostragem
// por atributos: a norma conta peças, não quilos. Quantidade fracionada só
// cabe em plano fixo ou 100 %.
func TamanhoDoLote(quantidade string) (int64, error) {
	q := strings.TrimSpace(strings.ReplaceAll(quantidade, ",", "."))
	if i := strings.IndexByte(q, '.'); i >= 0 && strings.Trim(q[i+1:], "0") == "" {
		q = q[:i]
	}
	n, err := strconv.ParseInt(q, 10, 64)
	if err != nil || n <= 0 {
		return 0, domainerr.NewValidation(fmt.Sprintf("a amostragem pela NBR 5426 conta unidades inteiras; o lote de %s não é inteiro — use plano fixo ou 100 %%", quantidade))
	}
	return n, nil
}

// PlanoPara devolve o plano do lote na severidade do estado.
func (p Procedimento) PlanoPara(tamanhoLote int64, e Estado) (Plano, error) {
	if tamanhoLote <= 0 {
		return Plano{}, domainerr.NewValidation("o lote precisa ter quantidade maior que zero")
	}
	switch p.Tipo {
	case TipoCem:
		return Plano{N: tamanhoLote, Ac: 0, Re: 1, CemPorCento: true, Severidade: Normal}, nil
	case TipoFixo:
		pl := Plano{N: *p.AmostraFixa, Ac: *p.AcFixo, Re: *p.ReFixo, Severidade: Normal}
		if pl.N >= tamanhoLote {
			pl.N, pl.CemPorCento = tamanhoLote, true
		}
		return pl, nil
	}
	sev := e.Severidade
	if sev == "" {
		sev = Normal
	}
	return PlanoParaLote(tamanhoLote, p.Nivel, p.NQA, sev)
}

// ResultadoDoLote monta o Resultado de um lote inspecionado com d não
// conformes na amostra, incluindo o teste do NQA um passo mais apertado que a
// pontuação de troca usa (sempre na tabela normal, 9.3.3.2).
func (p Procedimento) ResultadoDoLote(pl Plano, d int) Resultado {
	r := Resultado{Aceito: pl.Aceita(d), Ac: pl.Ac}
	if p.Tipo == TipoNorma && pl.Severidade == Normal && pl.Letra != "" {
		if mais := NQAMaisApertado(pl.NQA); mais != "" {
			if alt, err := PlanoDaTabela(pl.Letra, mais, Normal); err == nil {
				r.AceitoComNQAMaisApertado = alt.Aceita(d)
			}
		}
	}
	return r
}

// Registrar atualiza o estado com o lote. Só o procedimento pela norma troca
// severidade; plano fixo e 100 % só alimentam o skip-lot.
func (p Procedimento) Registrar(e Estado, r Resultado, producaoRegular bool) (Estado, *Transicao) {
	if p.Tipo == TipoNorma {
		return e.Registrar(r, Regras{PermiteAtenuada: p.PermiteAtenuada, ProducaoRegular: producaoRegular})
	}
	if e.Severidade == "" {
		e.Severidade = Normal
	}
	if r.Aceito {
		e.AceitosSeguidos++
	} else {
		e.AceitosSeguidos = 0
	}
	e.LotesDesdeInspecao = 0
	return e, nil
}

// Chave da unidade de controle do estado.
func ChaveFornecedor(fornecedor int64) string { return fmt.Sprintf("F:%d", fornecedor) }
func ChaveOperacao(operacao int64) string     { return fmt.Sprintf("O:%d", operacao) }

const ChaveGeral = "-"

// EstadoDoItem é o estado de uma unidade de controle, para consulta.
type EstadoDoItem struct {
	ProcedimentoID int64     `json:"procedure_id"`
	Procedimento   string    `json:"procedure_codigo"`
	ItemCode       int64     `json:"item_code"`
	Chave          string    `json:"chave"`
	Estado         Estado    `json:"estado"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Evento do histórico de severidade.
type Evento struct {
	ID             int64      `json:"id"`
	ProcedimentoID int64      `json:"procedure_id"`
	ItemCode       int64      `json:"item_code"`
	Chave          string     `json:"chave"`
	De             Severidade `json:"de"`
	Para           Severidade `json:"para"`
	Tipo           string     `json:"evento"`
	Motivo         string     `json:"motivo"`
	Referencia     *string    `json:"referencia,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
}

// Atualizacao altera o estado sob trava; devolve o novo estado, a transição
// (para o histórico) e o erro que desfaz tudo.
type Atualizacao func(Estado) (Estado, *Transicao, error)

// Repository da amostragem, sempre na empresa da sessão.
type Repository interface {
	ListarProcedimentos(ctx context.Context, somenteAtivos bool) ([]Procedimento, error)
	ObterProcedimento(ctx context.Context, id int64) (*Procedimento, error)
	SalvarProcedimento(ctx context.Context, p *Procedimento) error
	// Estado lê sem travar (Novo quando ainda não existe).
	Estado(ctx context.Context, procedimentoID, item int64, chave string) (Estado, error)
	// AtualizarEstado lê com trava, aplica fn e grava estado e evento na mesma
	// transação: dois lotes do mesmo fornecedor não perdem a troca um do outro.
	AtualizarEstado(ctx context.Context, procedimentoID, item int64, chave string, referencia *string, ator uuid.UUID, fn Atualizacao) (Estado, *Transicao, error)
	EstadosDoItem(ctx context.Context, item int64) ([]EstadoDoItem, error)
	Eventos(ctx context.Context, item int64, limite int) ([]Evento, error)
}

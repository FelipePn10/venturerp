// Package serie_uc: perfil de pós-venda do item, número de série e base
// instalada (Fase 3.1).
package serie_uc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/posvenda/garantia"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/serie"
)

type UseCase struct {
	Repo dom.Repository
	Auth ports.AuthService
	// Items traduz o código de negócio do item (o que a tela mostra) para a
	// chave interna. Sem ele, o código é lido como a chave interna.
	Items any
}

// Item: a chave interna a partir do código que o usuário vê.
func (uc *UseCase) Item(ctx context.Context, codigo string) (int64, error) {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return 0, errorsuc.NewValidationError("informe o item")
	}
	if uc.Items == nil {
		n, err := strconv.ParseInt(codigo, 10, 64)
		if err != nil || n <= 0 {
			return 0, errorsuc.NewValidationError("o código do item informado é inválido")
		}
		return n, nil
	}
	item, err := itemresolution.Resolve(ctx, uc.Items, request.TextCode(codigo))
	if err != nil {
		return 0, err
	}
	return int64(item.Code), nil
}

func (uc *UseCase) ator(ctx context.Context) *uuid.UUID {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return nil
	}
	return &a
}

func limpo(s *string) *string {
	if s == nil {
		return nil
	}
	if v := strings.TrimSpace(*s); v != "" {
		return &v
	}
	return nil
}

func dia(s *string, campo string) (*time.Time, error) {
	v := limpo(s)
	if v == nil {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", (*v)[:min(10, len(*v))])
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s inválida (use AAAA-MM-DD)", campo))
	}
	return &d, nil
}

// ─── Perfil ──────────────────────────────────────────────────────────────────

type PerfilDTO struct {
	ControlaSerie  bool    `json:"controla_serie"`
	MascaraSerieID *int64  `json:"mascara_serie_id,omitempty"`
	Observacoes    *string `json:"observacoes,omitempty"`
	// Garantia (3.2).
	GarantiaMeses     int    `json:"garantia_meses"`
	GarantiaInicio    string `json:"garantia_inicio,omitempty"`
	CobrePecas        bool   `json:"cobre_pecas"`
	CobreMaoObra      bool   `json:"cobre_mao_obra"`
	CobreDeslocamento bool   `json:"cobre_deslocamento"`
}

func (uc *UseCase) Perfil(ctx context.Context, codigo string) (*dom.Perfil, error) {
	item, err := uc.Item(ctx, codigo)
	if err != nil {
		return nil, err
	}
	return uc.Repo.Perfil(ctx, item)
}

func (uc *UseCase) Perfis(ctx context.Context) ([]dom.Perfil, error) { return uc.Repo.Perfis(ctx) }

func (uc *UseCase) SalvarPerfil(ctx context.Context, codigo string, dto PerfilDTO) (*dom.Perfil, error) {
	item, err := uc.Item(ctx, codigo)
	if err != nil {
		return nil, err
	}
	if dto.MascaraSerieID != nil && *dto.MascaraSerieID <= 0 {
		dto.MascaraSerieID = nil
	}
	inicio := strings.ToUpper(strings.TrimSpace(dto.GarantiaInicio))
	if inicio == "" {
		inicio = garantia.InicioVenda
	}
	if inicio != garantia.InicioVenda && inicio != garantia.InicioInstalacao {
		return nil, errorsuc.NewValidationError("a garantia começa na venda ou na instalação")
	}
	if dto.GarantiaMeses < 0 || dto.GarantiaMeses > 240 {
		return nil, errorsuc.NewValidationError("o prazo de garantia vai de 0 a 240 meses")
	}
	p := &dom.Perfil{ItemCode: item, ControlaSerie: dto.ControlaSerie, MascaraSerieID: dto.MascaraSerieID,
		Observacoes: limpo(dto.Observacoes), UpdatedBy: uc.ator(ctx),
		Garantia: garantia.Regra{Meses: dto.GarantiaMeses, Inicio: inicio, CobrePecas: dto.CobrePecas, CobreMaoObra: dto.CobreMaoObra,
			CobreDeslocamento: dto.CobreDeslocamento}}
	if !p.ControlaSerie {
		p.MascaraSerieID = nil
	}
	if err := uc.Repo.SalvarPerfil(ctx, p); err != nil {
		return nil, err
	}
	return uc.Repo.Perfil(ctx, item)
}

// ─── Séries ──────────────────────────────────────────────────────────────────

// CadastroDTO: uma ou várias séries do mesmo item (a base legada costuma vir
// numa planilha: um item, dezenas de séries no mesmo cliente).
type CadastroDTO struct {
	ItemCode        request.TextCode `json:"item_code"`
	Mask            *string          `json:"mask,omitempty"`
	Series          []string         `json:"series"`
	Situacao        string           `json:"situacao"`
	Origem          string           `json:"origem,omitempty"`
	WarehouseID     *int64           `json:"warehouse_id,omitempty"`
	Lote            *string          `json:"lote,omitempty"`
	CustomerCode    *int64           `json:"customer_code,omitempty"`
	DataVenda       *string          `json:"data_venda,omitempty"`
	DataInstalacao  *string          `json:"data_instalacao,omitempty"`
	LocalInstalacao *string          `json:"local_instalacao,omitempty"`
	Observacoes     *string          `json:"observacoes,omitempty"`
}

func (uc *UseCase) Cadastrar(ctx context.Context, dto CadastroDTO) ([]dom.Serie, error) {
	item, err := uc.Item(ctx, dto.ItemCode.String())
	if err != nil {
		return nil, err
	}
	perfil, err := uc.Repo.Perfil(ctx, item)
	if err != nil {
		return nil, err
	}
	if !perfil.ControlaSerie {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %s não controla número de série: ligue no perfil de pós-venda (VATC0110)", dto.ItemCode.String()))
	}
	series, err := dom.ConferirLista(dto.Series, decimal.NewFromInt(int64(contar(dto.Series))))
	if err != nil {
		return nil, err
	}
	if len(series) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos uma série")
	}
	situacao := strings.ToUpper(strings.TrimSpace(dto.Situacao))
	switch situacao {
	case dom.EmEstoque:
		if dto.WarehouseID == nil {
			return nil, errorsuc.NewValidationError("série em estoque precisa do almoxarifado")
		}
		dto.CustomerCode = nil
	case dom.NoCliente:
		if dto.CustomerCode == nil {
			return nil, errorsuc.NewValidationError("série no cliente precisa do cliente")
		}
	default:
		return nil, errorsuc.NewValidationError("a situação do cadastro é em estoque ou no cliente")
	}
	origem := strings.ToUpper(strings.TrimSpace(dto.Origem))
	if origem == "" {
		origem = dom.OrigemCadastro
	}
	if origem != dom.OrigemCadastro && origem != dom.OrigemCompra {
		return nil, errorsuc.NewValidationError("a origem do cadastro manual é cadastro ou compra")
	}
	venda, err := dia(dto.DataVenda, "data da venda")
	if err != nil {
		return nil, err
	}
	instalacao, err := dia(dto.DataInstalacao, "data da instalação")
	if err != nil {
		return nil, err
	}
	por := uc.ator(ctx)
	out := make([]dom.Serie, 0, len(series))
	motivo := "cadastro manual"
	for _, n := range series {
		s := &dom.Serie{ItemCode: item, Mask: limpo(dto.Mask), NumeroSerie: n, Situacao: situacao, Origem: origem,
			WarehouseID: dto.WarehouseID, Lote: limpo(dto.Lote), CustomerCode: dto.CustomerCode, DataVenda: venda,
			DataInstalacao: instalacao, LocalInstalacao: limpo(dto.LocalInstalacao), Observacoes: limpo(dto.Observacoes)}
		if err := uc.Repo.Cadastrar(ctx, s, por, &motivo); err != nil {
			if len(out) > 0 {
				return nil, fmt.Errorf("%w (as %d série(s) anteriores ficaram gravadas)", err, len(out))
			}
			return nil, err
		}
		gravada, err := uc.Repo.Obter(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *gravada)
	}
	return out, nil
}

func contar(series []string) int {
	n := 0
	for _, s := range series {
		if strings.TrimSpace(s) != "" {
			n++
		}
	}
	return n
}

type SerieResponse struct {
	*dom.Serie
	Historico []dom.Evento `json:"historico"`
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*SerieResponse, error) {
	s, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	h, err := uc.Repo.Historico(ctx, id)
	if err != nil {
		return nil, err
	}
	return &SerieResponse{Serie: s, Historico: h}, nil
}

// Listar aceita o item pelo código de negócio.
func (uc *UseCase) Listar(ctx context.Context, item string, f dom.Filtro) ([]dom.Serie, error) {
	if strings.TrimSpace(item) != "" {
		code, err := uc.Item(ctx, item)
		if err != nil {
			return nil, err
		}
		f.ItemCode = &code
	}
	return uc.Repo.Listar(ctx, f)
}

type InstalacaoDTO struct {
	Data  string  `json:"data"`
	Local *string `json:"local,omitempty"`
}

func (uc *UseCase) Instalar(ctx context.Context, id int64, dto InstalacaoDTO) (*SerieResponse, error) {
	d, err := dia(&dto.Data, "data da instalação")
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errorsuc.NewValidationError("informe a data da instalação")
	}
	if _, err := uc.Repo.Mudar(ctx, id, dom.Mudanca{Evento: dom.EventoInstalacao, DataInstalacao: d, LocalInstalacao: limpo(dto.Local), Por: uc.ator(ctx)}); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

type TransferenciaDTO struct {
	CustomerCode int64  `json:"customer_code"`
	Motivo       string `json:"motivo"`
}

func (uc *UseCase) Transferir(ctx context.Context, id int64, dto TransferenciaDTO) (*SerieResponse, error) {
	motivo := strings.TrimSpace(dto.Motivo)
	if dto.CustomerCode <= 0 || motivo == "" {
		return nil, errorsuc.NewValidationError("informe o novo cliente e o motivo da transferência")
	}
	atual, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if atual.CustomerCode != nil && *atual.CustomerCode == dto.CustomerCode {
		return nil, errorsuc.NewValidationError("a série já está nesse cliente")
	}
	cliente := dto.CustomerCode
	if _, err := uc.Repo.Mudar(ctx, id, dom.Mudanca{Evento: dom.EventoTransferencia, CustomerCode: &cliente, Motivo: &motivo, Por: uc.ator(ctx)}); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

func (uc *UseCase) Baixar(ctx context.Context, id int64, motivo string) (*SerieResponse, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo da baixa")
	}
	if _, err := uc.Repo.Mudar(ctx, id, dom.Mudanca{Evento: dom.EventoBaixa, Motivo: &motivo, Por: uc.ator(ctx)}); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

// ─── NF de saída ─────────────────────────────────────────────────────────────

func (uc *UseCase) LinhasDaNota(ctx context.Context, exitID int64) ([]dom.LinhaNota, error) {
	return uc.Repo.LinhasComSerie(ctx, exitID)
}

type SeriesDaLinhaDTO struct {
	Series []int64 `json:"series"`
}

func (uc *UseCase) DefinirSeriesDaLinha(ctx context.Context, exitID, itemID int64, dto SeriesDaLinhaDTO) ([]dom.LinhaNota, error) {
	vistas := map[int64]bool{}
	ids := make([]int64, 0, len(dto.Series))
	for _, id := range dto.Series {
		if id <= 0 || vistas[id] {
			continue
		}
		vistas[id] = true
		ids = append(ids, id)
	}
	linhas, err := uc.Repo.LinhasComSerie(ctx, exitID)
	if err != nil {
		return nil, err
	}
	for _, l := range linhas {
		if l.FiscalExitItemID != itemID {
			continue
		}
		if !l.Quantidade.Equal(l.Quantidade.Truncate(0)) {
			return nil, errorsuc.NewValidationError("item com número de série só aceita quantidade inteira na nota")
		}
		if int64(len(ids)) > l.Quantidade.IntPart() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha tem %s unidade(s) e foram escolhidas %d série(s)", l.Quantidade.String(), len(ids)))
		}
		if err := uc.Repo.DefinirSeriesDaLinha(ctx, exitID, itemID, ids, uc.ator(ctx)); err != nil {
			return nil, err
		}
		return uc.Repo.LinhasComSerie(ctx, exitID)
	}
	return nil, errorsuc.NewValidationError("a linha não é de item com número de série")
}

// ─── Garantia (3.2) ──────────────────────────────────────────────────────────

// ConsultaGarantiaDTO: o que se sabe do produto no atendimento.
type ConsultaGarantiaDTO struct {
	Item       string  `json:"item"`
	Serie      string  `json:"serie,omitempty"`
	Cliente    *int64  `json:"cliente,omitempty"`
	NumeroNF   *int64  `json:"numero_nf,omitempty"`
	DataCompra *string `json:"data_compra,omitempty"`
}

// GarantiaResponse: a decisão e de onde ela veio.
type GarantiaResponse struct {
	garantia.Resultado
	Regra  garantia.Regra   `json:"regra"`
	Serie  *dom.Serie       `json:"serie,omitempty"`
	Nota   *dom.NotaVendida `json:"nota,omitempty"`
	Avisos []string         `json:"avisos,omitempty"`
}

// Hoje é trocável no teste.
var Hoje = time.Now

func (uc *UseCase) ConsultarGarantia(ctx context.Context, dto ConsultaGarantiaDTO) (*GarantiaResponse, error) {
	item, err := uc.Item(ctx, dto.Item)
	if err != nil {
		return nil, err
	}
	compra, err := dia(dto.DataCompra, "data da compra")
	if err != nil {
		return nil, err
	}
	var serie *string
	if s := strings.TrimSpace(dto.Serie); s != "" {
		serie = &s
	}
	return uc.Determinar(ctx, item, serie, dto.Cliente, dto.NumeroNF, compra)
}

// Determinar: série da base instalada > NF de saída do cliente > data
// informada. Usado pela consulta e pelo item do chamado.
func (uc *UseCase) Determinar(ctx context.Context, item int64, serie *string, cliente, numeroNF *int64, compra *time.Time) (*GarantiaResponse, error) {
	perfil, err := uc.Repo.Perfil(ctx, item)
	if err != nil {
		return nil, err
	}
	out := &GarantiaResponse{Regra: perfil.Garantia}
	base := garantia.Base{}
	if serie != nil {
		s, err := uc.Repo.PorNumero(ctx, item, *serie)
		if err != nil && !errorsuc.EhNaoEncontrado(err) {
			return nil, err
		}
		if s != nil {
			out.Serie = s
			base = garantia.Base{Origem: garantia.OrigemSerie, Venda: s.DataVenda, Instalacao: s.DataInstalacao, EstendidaAte: s.EstendidaAte}
			if cliente != nil && s.CustomerCode != nil && *s.CustomerCode != *cliente {
				out.Avisos = append(out.Avisos, fmt.Sprintf("a série está registrada para outro cliente (%s)", s.ClienteNome))
			}
			if s.Situacao == dom.Baixado {
				out.Avisos = append(out.Avisos, "a série está baixada")
			}
		} else if perfil.ControlaSerie {
			out.Avisos = append(out.Avisos, fmt.Sprintf("a série %s não está na base instalada (VATC0100)", *serie))
		}
	}
	if base.Origem == "" && numeroNF != nil && cliente != nil {
		n, err := uc.Repo.NotaDoCliente(ctx, *cliente, *numeroNF)
		if err != nil && !errorsuc.EhNaoEncontrado(err) {
			return nil, err
		}
		if n != nil {
			out.Nota = n
			v := n.Emissao
			base = garantia.Base{Origem: garantia.OrigemNF, Venda: &v}
		} else {
			out.Avisos = append(out.Avisos, fmt.Sprintf("a NF %d não foi encontrada como venda autorizada para o cliente", *numeroNF))
		}
	}
	if base.Origem == "" && compra != nil {
		base = garantia.Base{Origem: garantia.OrigemInformada, Venda: compra}
	}
	out.Resultado = garantia.Determinar(perfil.Garantia, base, Hoje())
	return out, nil
}

type GarantiaEstendidaDTO struct {
	Ate    string `json:"ate"`
	Motivo string `json:"motivo"`
}

func (uc *UseCase) EstenderGarantia(ctx context.Context, id int64, dto GarantiaEstendidaDTO) (*SerieResponse, error) {
	ate, err := dia(&dto.Ate, "data final da garantia estendida")
	if err != nil {
		return nil, err
	}
	motivo := strings.TrimSpace(dto.Motivo)
	if ate == nil || motivo == "" {
		return nil, errorsuc.NewValidationError("informe até quando vai a garantia estendida e o motivo (contrato, cortesia)")
	}
	s, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.Situacao == dom.Baixado {
		return nil, errorsuc.NewValidationError("série baixada não recebe garantia estendida")
	}
	if err := uc.Repo.EstenderGarantia(ctx, id, *ate, motivo, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

// GarantiaParaChamado: a decisão de garantia do item do chamado, com a série da
// base instalada e a data da venda que a fundamentou.
func (uc *UseCase) GarantiaParaChamado(ctx context.Context, item int64, serie *string, cliente, numeroNF *int64, compra *time.Time) (garantia.Resultado, *int64, *time.Time, error) {
	g, err := uc.Determinar(ctx, item, serie, cliente, numeroNF, compra)
	if err != nil {
		return garantia.Resultado{}, nil, nil, err
	}
	var serieID *int64
	var venda *time.Time
	if g.Serie != nil {
		id := g.Serie.ID
		serieID, venda = &id, g.Serie.DataVenda
	}
	if g.Nota != nil {
		v := g.Nota.Emissao
		venda = &v
	}
	return g.Resultado, serieID, venda, nil
}

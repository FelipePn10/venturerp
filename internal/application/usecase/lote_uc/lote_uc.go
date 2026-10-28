// Package lote_uc é o lote de inspeção: abertura (recebimento, processo,
// manual) com as características do plano e a amostragem NBR 5426,
// resultados com a conformidade decidida pelo servidor, decisão de uso que
// move o estoque e alimenta a severidade, e cancelamento.
package lote_uc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/metrologia"
)

type UseCase struct {
	Repo       dom.Repository
	Fontes     dom.Fontes
	Amostragem amostragem.Repository
	Auth       ports.AuthService
	// Metrologia confere o instrumento das medições (opcional).
	Metrologia metrologia.Repository
	Agora      func() time.Time
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type AberturaDTO struct {
	Origem                string          `json:"origem"`
	ItemCode              int64           `json:"item_code"`
	Mask                  string          `json:"mask"`
	Lote                  *string         `json:"lote,omitempty"`
	Serie                 *string         `json:"serie,omitempty"`
	Quantidade            decimal.Decimal `json:"quantidade"`
	WarehouseID           *int64          `json:"warehouse_id,omitempty"`
	SupplierCode          *int64          `json:"supplier_code,omitempty"`
	PurchaseOrderCode     *int64          `json:"purchase_order_code,omitempty"`
	PurchaseOrderItemCode *int64          `json:"purchase_order_item_code,omitempty"`
	FiscalEntryCode       *int64          `json:"fiscal_entry_code,omitempty"`
	ReceivingNoticeCode   *int64          `json:"receiving_notice_code,omitempty"`
	ProductionOrderID     *int64          `json:"production_order_id,omitempty"`
	RouteOperationID      *int64          `json:"route_operation_id,omitempty"`
	InspectionPlanID      *int64          `json:"inspection_plan_id,omitempty"`
	// SemRoteiro: abrir sem copiar o roteiro de recebimento do item.
	SemRoteiro     bool    `json:"sem_roteiro,omitempty"`
	ProcedimentoID *int64  `json:"sampling_procedure_id,omitempty"`
	Certificado    *string `json:"certificado,omitempty"`
	NotaFornecedor *string `json:"nota_fornecedor,omitempty"`
	Modelo         *string `json:"modelo,omitempty"`
	Observacoes    *string `json:"observacoes,omitempty"`
	// Origem legada (rotas de compatibilidade): não vem do cliente.
	LegacyKind *string `json:"-"`
	LegacyID   *int64  `json:"-"`
	// Registro antigo de qualidade: sem amostragem; características só se
	// vieram medições.
	SemAmostragem      bool `json:"-"`
	SemCaracteristicas bool `json:"-"`
}

type ResultadoDTO struct {
	CaracteristicaID int64            `json:"characteristic_id"`
	AmostraIndice    int              `json:"amostra_indice"`
	Valor            *decimal.Decimal `json:"valor,omitempty"`
	Conforme         *bool            `json:"conforme,omitempty"`
	Atributo         *string          `json:"atributo,omitempty"`
	Observacao       *string          `json:"observacao,omitempty"`
	InstrumentoID    *int64           `json:"instrument_id,omitempty"`
}

type DecisaoDTO struct {
	Aceita            decimal.Decimal `json:"qtd_aceita"`
	Restrita          decimal.Decimal `json:"qtd_restrita"`
	Retrabalho        decimal.Decimal `json:"qtd_retrabalho"`
	Rejeitada         decimal.Decimal `json:"qtd_rejeitada"`
	Sucata            decimal.Decimal `json:"qtd_sucata"`
	DestinoAceito     *int64          `json:"destino_aceito,omitempty"`
	DestinoRestrito   *int64          `json:"destino_restrito,omitempty"`
	DestinoRetrabalho *int64          `json:"destino_retrabalho,omitempty"`
	DestinoRejeitado  *int64          `json:"destino_rejeitado,omitempty"`
	DestinoSucata     *int64          `json:"destino_sucata,omitempty"`
	Tratamento        *string         `json:"tratamento,omitempty"`
	AfetaIQF          *bool           `json:"afeta_iqf,omitempty"`
	// ProducaoIrregular devolve a atenuada para normal (9.3.4 b).
	ProducaoIrregular bool    `json:"producao_irregular,omitempty"`
	Motivo            *string `json:"motivo,omitempty"`
}

// LoteResponse: o lote com a avaliação corrente.
type LoteResponse struct {
	*dom.Lote
	Avaliacao  dom.Avaliacao `json:"avaliacao"`
	Relatorios []int64       `json:"relatorios_fornecedor"`
}

// ─── abertura ─────────────────────────────────────────────────────────────────

func (uc *UseCase) ator(ctx context.Context) uuid.UUID {
	a, _ := uc.Auth.UserID(ctx)
	return a
}

// Abrir abre o lote: copia as características do plano e calcula o plano de
// amostragem pela severidade corrente do item × fornecedor (ou × operação).
func (uc *UseCase) Abrir(ctx context.Context, dto AberturaDTO) (*LoteResponse, error) {
	origem := dom.Origem(strings.ToUpper(strings.TrimSpace(dto.Origem)))
	if origem == "" {
		origem = dom.OrigemManual
	}
	if !origem.Valida() {
		return nil, errorsuc.NewValidationError("origem inválida (RECEBIMENTO, PROCESSO, FINAL, DEVOLUCAO ou MANUAL)")
	}
	if dto.ItemCode <= 0 || !dto.Quantidade.IsPositive() {
		return nil, errorsuc.NewValidationError("informe o item e a quantidade do lote (maior que zero)")
	}
	ator := uc.ator(ctx)
	l := &dom.Lote{
		Origem: origem, Status: dom.Aberto, ItemCode: dto.ItemCode, Mask: dto.Mask, Lote: limpo(dto.Lote), Serie: limpo(dto.Serie),
		Quantidade: dto.Quantidade.Round(6), WarehouseID: dto.WarehouseID, SupplierCode: dto.SupplierCode,
		PurchaseOrderCode: dto.PurchaseOrderCode, PurchaseOrderItemCode: dto.PurchaseOrderItemCode, FiscalEntryCode: dto.FiscalEntryCode,
		ReceivingNoticeCode: dto.ReceivingNoticeCode, ProductionOrderID: dto.ProductionOrderID, RouteOperationID: dto.RouteOperationID,
		Certificado: limpo(dto.Certificado), NotaFornecedor: limpo(dto.NotaFornecedor), Modelo: limpo(dto.Modelo), Observacoes: limpo(dto.Observacoes),
		LegacyKind: dto.LegacyKind, LegacyID: dto.LegacyID,
	}
	if ator != uuid.Nil {
		l.CreatedBy = &ator
	}
	var fonte *dom.Fonte
	var err error
	switch {
	case dto.InspectionPlanID != nil && *dto.InspectionPlanID > 0:
		fonte, err = uc.Fontes.PlanoDeInspecao(ctx, *dto.InspectionPlanID)
	case !dto.SemRoteiro && (origem == dom.OrigemRecebimento || origem == dom.OrigemManual || origem == dom.OrigemDevolucao):
		fonte, err = uc.Fontes.RoteiroDeRecebimento(ctx, dto.ItemCode, dto.Mask)
	}
	if err != nil {
		return nil, err
	}
	proc := dto.ProcedimentoID
	if fonte != nil {
		l.ReceivingRouteID, l.InspectionPlanID = fonte.RouteID, fonte.PlanID
		if !dto.SemCaracteristicas {
			l.Caracteristicas = fonte.Caracteristicas
		}
		if l.RouteOperationID == nil {
			l.RouteOperationID = fonte.RouteOperation
		}
		if proc == nil {
			proc = fonte.ProcedimentoID
		}
	}
	if dto.SemAmostragem {
		proc = nil
	}
	if proc != nil && *proc > 0 {
		chave := amostragem.ChaveGeral
		switch {
		case l.SupplierCode != nil && *l.SupplierCode > 0:
			chave = amostragem.ChaveFornecedor(*l.SupplierCode)
		case l.RouteOperationID != nil && *l.RouteOperationID > 0:
			chave = amostragem.ChaveOperacao(*l.RouteOperationID)
		}
		p, err := uc.Amostragem.ObterProcedimento(ctx, *proc)
		if err != nil {
			return nil, err
		}
		if !p.Ativo {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o procedimento de amostragem %s está inativo", p.Codigo))
		}
		e, err := uc.Amostragem.Estado(ctx, p.ID, l.ItemCode, chave)
		if err != nil {
			return nil, err
		}
		tamanho := int64(1)
		if p.Tipo == amostragem.TipoNorma {
			if tamanho, err = amostragem.TamanhoDoLote(l.Quantidade.String()); err != nil {
				return nil, err
			}
		} else if n, errN := amostragem.TamanhoDoLote(l.Quantidade.String()); errN == nil {
			tamanho = n
		}
		pl, err := p.PlanoPara(tamanho, e)
		if err != nil {
			return nil, err
		}
		l.ProcedimentoID, l.ChaveAmostragem, l.Plano = &p.ID, &chave, &pl
		l.Pulado = !e.DeveInspecionar(p.Skip)
	}
	if err := uc.Repo.Abrir(ctx, l); err != nil {
		return nil, err
	}
	return uc.resposta(ctx, l)
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

func (uc *UseCase) resposta(ctx context.Context, l *dom.Lote) (*LoteResponse, error) {
	rel, err := uc.Repo.Relatorios(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	return &LoteResponse{Lote: l, Avaliacao: l.Avaliar(), Relatorios: rel}, nil
}

// ─── consulta ─────────────────────────────────────────────────────────────────

func (uc *UseCase) Obter(ctx context.Context, id int64) (*LoteResponse, error) {
	l, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.resposta(ctx, l)
}

func (uc *UseCase) ObterPorLegado(ctx context.Context, kind string, id int64) (*dom.Lote, error) {
	return uc.Repo.ObterPorLegado(ctx, kind, id)
}

func (uc *UseCase) Listar(ctx context.Context, f dom.Filtro) ([]dom.Lote, error) {
	return uc.Repo.Listar(ctx, f)
}

// ─── resultados ───────────────────────────────────────────────────────────────

// GravarResultados registra (ou corrige) resultados do lote. A conformidade
// da característica variável é calculada pelos limites copiados do plano.
func (uc *UseCase) GravarResultados(ctx context.Context, id int64, dtos []ResultadoDTO) (*LoteResponse, error) {
	if len(dtos) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos um resultado")
	}
	l, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if l.Status != dom.Aberto {
		return nil, errorsuc.NewValidationError("o lote não está aberto: os resultados não podem mais ser alterados")
	}
	carac := map[int64]dom.Caracteristica{}
	for _, c := range l.Caracteristicas {
		carac[c.ID] = c
	}
	n := l.TamanhoDaAmostra()
	if l.Plano == nil {
		// Sem procedimento, a amostra é o que o inspetor medir, até o lote.
		if q := l.Quantidade.Ceil().IntPart(); q > n {
			n = q
		}
	}
	rs := make([]dom.Resultado, 0, len(dtos))
	for _, x := range dtos {
		c, ok := carac[x.CaracteristicaID]
		if !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a característica %d não pertence ao lote %d", x.CaracteristicaID, l.Numero))
		}
		if x.AmostraIndice <= 0 || int64(x.AmostraIndice) > n {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("%s: a unidade da amostra vai de 1 a %d", c.Nome, n))
		}
		conf, err := c.Conformidade(x.Valor, x.Conforme)
		if err != nil {
			return nil, err
		}
		var valor *decimal.Decimal
		if x.Valor != nil {
			v := x.Valor.Round(6)
			valor = &v
		}
		rs = append(rs, dom.Resultado{CaracteristicaID: c.ID, AmostraIndice: x.AmostraIndice, Valor: valor, Atributo: limpo(x.Atributo),
			Conforme: conf, Observacao: limpo(x.Observacao), InstrumentoID: x.InstrumentoID})
	}
	if err := uc.conferirInstrumentos(ctx, carac, rs); err != nil {
		return nil, err
	}
	if err := uc.Repo.GravarResultados(ctx, id, rs, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

// ─── decisão ──────────────────────────────────────────────────────────────────

func (uc *UseCase) Decidir(ctx context.Context, id int64, dto DecisaoDTO) (*LoteResponse, error) {
	ator := uc.ator(ctx)
	d := &dom.Decisao{
		Aceita: dto.Aceita.Round(6), Restrita: dto.Restrita.Round(6), Retrabalho: dto.Retrabalho.Round(6),
		Rejeitada: dto.Rejeitada.Round(6), Sucata: dto.Sucata.Round(6),
		DestinoAceito: dto.DestinoAceito, DestinoRestrito: dto.DestinoRestrito, DestinoRetrabalho: dto.DestinoRetrabalho,
		DestinoRejeitado: dto.DestinoRejeitado, DestinoSucata: dto.DestinoSucata,
		Tratamento: dto.Tratamento, AfetaIQF: dto.AfetaIQF == nil || *dto.AfetaIQF, Motivo: dto.Motivo,
	}
	// O procedimento é lido antes (não muda com o lote); o resto é decidido
	// sobre o lote travado.
	atual, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	var proc *amostragem.Procedimento
	if atual.ProcedimentoID != nil {
		if proc, err = uc.Amostragem.ObterProcedimento(ctx, *atual.ProcedimentoID); err != nil {
			return nil, err
		}
	}
	l, err := uc.Repo.Decidir(ctx, id, d, ator, func(l *dom.Lote, almox map[int64]dom.Almoxarifado) (*dom.AtualizacaoDaAmostragem, *dom.NaoConformidade, error) {
		if err := l.PrepararDecisao(d, almox); err != nil {
			return nil, nil, err
		}
		return efeitos(l, d, proc, !dto.ProducaoIrregular)
	})
	if err != nil {
		return nil, err
	}
	return uc.resposta(ctx, l)
}

// efeitos calcula a troca de severidade e a NC da decisão.
func efeitos(l *dom.Lote, d *dom.Decisao, proc *amostragem.Procedimento, producaoRegular bool) (*dom.AtualizacaoDaAmostragem, *dom.NaoConformidade, error) {
	var at *dom.AtualizacaoDaAmostragem
	if proc != nil && l.Plano != nil && l.ChaveAmostragem != nil {
		ref := fmt.Sprintf("lote de inspeção %d", l.Numero)
		aproveita := d.Aproveitada().IsPositive()
		av := l.Avaliar()
		var r amostragem.Resultado
		if !l.Pulado {
			r = proc.ResultadoDoLote(*l.Plano, av.NaoConformes)
			if av.CriticaReprovada || !av.Completa {
				r.Aceito, r.AceitoComNQAMaisApertado = false, false
			}
		}
		pulado := l.Pulado
		at = &dom.AtualizacaoDaAmostragem{
			ProcedimentoID: proc.ID, ItemCode: l.ItemCode, Chave: *l.ChaveAmostragem, Referencia: ref,
			Fn: func(e amostragem.Estado) (amostragem.Estado, *amostragem.Transicao, error) {
				if e.Suspensa {
					if aproveita {
						return e, nil, errorsuc.NewValidationError("a inspeção deste item/fornecedor está suspensa (5 lotes recusados em apertada): o lote só pode ser rejeitado ou devolvido até a liberação na VINS0110")
					}
					return e, nil, nil
				}
				if pulado {
					return e.Pular(), nil, nil
				}
				novo, tr := proc.Registrar(e, r, producaoRegular)
				return novo, tr, nil
			},
		}
	}
	naoAproveitada := d.Retrabalho.Add(d.Rejeitada).Add(d.Sucata)
	concessao := d.Tratamento != nil && *d.Tratamento == "CONCESSAO"
	if !naoAproveitada.IsPositive() && !d.Restrita.IsPositive() && !concessao {
		return at, nil, nil
	}
	qtd := naoAproveitada.Add(d.Restrita)
	if concessao && !qtd.IsPositive() {
		qtd = l.Quantidade
	}
	sev := "MAIOR"
	if l.Avaliar().CriticaReprovada {
		sev = "CRITICA"
	} else if naoAproveitada.IsZero() {
		sev = "MENOR"
	}
	motivo := ""
	if d.Motivo != nil {
		motivo = ": " + *d.Motivo
	}
	nc := &dom.NaoConformidade{
		Descricao:  fmt.Sprintf("Lote de inspeção %d (%s) — %s%s", l.Numero, strings.ToLower(string(l.Origem)), strings.ToLower(strings.ReplaceAll(d.Tipo, "_", " ")), motivo),
		Quantidade: qtd.String(),
		Severidade: sev,
	}
	return at, nc, nil
}

// Cancelar cancela o lote aberto cuja entrada foi estornada (ou o lote sem
// estoque, como o de processo de uma ordem cancelada).
func (uc *UseCase) Cancelar(ctx context.Context, id int64, motivo string) (*LoteResponse, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do cancelamento")
	}
	if err := uc.Repo.Cancelar(ctx, id, motivo, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

// ─── laudos do fornecedor ─────────────────────────────────────────────────────

func (uc *UseCase) VincularRelatorio(ctx context.Context, id, relatorio int64) error {
	ator := uc.ator(ctx)
	if ator == uuid.Nil {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.VincularRelatorio(ctx, id, relatorio, ator)
}

func (uc *UseCase) DesvincularRelatorio(ctx context.Context, id, relatorio int64) error {
	return uc.Repo.DesvincularRelatorio(ctx, id, relatorio)
}

// ─── portas do recebimento ────────────────────────────────────────────────────

// AbrirDoRecebimento abre o lote do material que o recebimento (pedido de
// compra ou NF de entrada) colocou no almoxarifado de inspeção. Implementa as
// portas de purchase_order_uc e fiscal_uc.
func (uc *UseCase) AbrirDoRecebimento(ctx context.Context, item int64, mask string, quantidade decimal.Decimal, almoxInspecao int64,
	fornecedor, pedido, linha *int64, lote *string, notaEntrada *int64) (int64, int64, error) {
	r, err := uc.Abrir(ctx, AberturaDTO{
		Origem: string(dom.OrigemRecebimento), ItemCode: item, Mask: mask, Quantidade: quantidade, WarehouseID: &almoxInspecao,
		SupplierCode: fornecedor, PurchaseOrderCode: pedido, PurchaseOrderItemCode: linha, Lote: lote, FiscalEntryCode: notaEntrada,
	})
	if err != nil {
		return 0, 0, err
	}
	return r.ID, r.Numero, nil
}

// ResolveInspectionRoute: almoxarifado de inspeção do roteiro vigente do item
// na empresa da sessão.
func (uc *UseCase) ResolveInspectionRoute(ctx context.Context, item int64, mask string) (int64, bool) {
	f, err := uc.Fontes.RoteiroDeRecebimento(ctx, item, mask)
	if err != nil || f == nil || f.AlmoxInspecao == nil {
		return 0, false
	}
	return *f.AlmoxInspecao, true
}

// AmostragemDoProcesso: plano de amostragem do lote de processo pelo
// procedimento do plano de inspeção e a severidade do item × operação.
// Implementa production_order_uc.AmostragemDeProcesso.
func (uc *UseCase) AmostragemDoProcesso(ctx context.Context, planoID, item, operacao int64, quantidade decimal.Decimal) (*production_order_uc.AmostragemDoLote, error) {
	f, err := uc.Fontes.PlanoDeInspecao(ctx, planoID)
	if err != nil || f.ProcedimentoID == nil {
		return nil, err
	}
	p, err := uc.Amostragem.ObterProcedimento(ctx, *f.ProcedimentoID)
	if err != nil {
		return nil, err
	}
	if !p.Ativo {
		return nil, nil
	}
	chave := amostragem.ChaveOperacao(operacao)
	e, err := uc.Amostragem.Estado(ctx, p.ID, item, chave)
	if err != nil {
		return nil, err
	}
	tamanho, errN := amostragem.TamanhoDoLote(quantidade.String())
	if errN != nil {
		// Etapa fracionada (kg, m) com procedimento pela norma: o lote abre
		// sem plano de amostragem — a inspeção é controle sobre a ordem e não
		// pode impedir que ela nasça.
		if p.Tipo == amostragem.TipoNorma {
			return nil, nil
		}
		tamanho = 1
	}
	pl, err := p.PlanoPara(tamanho, e)
	if err != nil {
		return nil, err
	}
	bruto, err := json.Marshal(pl)
	if err != nil {
		return nil, err
	}
	return &production_order_uc.AmostragemDoLote{ProcedimentoID: p.ID, Chave: chave, Plano: bruto, Pulado: !e.DeveInspecionar(p.Skip)}, nil
}

// conferirInstrumentos: instrumento informado tem de poder medir (ativo, do
// grupo da característica, calibração em dia). Característica com grupo exige
// instrumento quando a empresa já cadastrou instrumentos daquele grupo — quem
// ainda não usa a metrologia não fica impedido de inspecionar.
func (uc *UseCase) conferirInstrumentos(ctx context.Context, carac map[int64]dom.Caracteristica, rs []dom.Resultado) error {
	if uc.Metrologia == nil {
		return nil
	}
	hoje := time.Now()
	if uc.Agora != nil {
		hoje = uc.Agora()
	}
	ids := []int64{}
	for _, r := range rs {
		if r.InstrumentoID != nil {
			ids = append(ids, *r.InstrumentoID)
		}
	}
	instr := map[int64]metrologia.Instrumento{}
	if len(ids) > 0 {
		var err error
		if instr, err = uc.Metrologia.PorIDs(ctx, ids); err != nil {
			return err
		}
	}
	grupoTem := map[string]bool{}
	for _, r := range rs {
		c := carac[r.CaracteristicaID]
		grupo := ""
		if c.Instrumento != nil {
			grupo = metrologia.NormalizarGrupo(*c.Instrumento)
		}
		if r.InstrumentoID == nil {
			if grupo == "" {
				continue
			}
			tem, ok := grupoTem[grupo]
			if !ok {
				lista, err := uc.Metrologia.Listar(ctx, metrologia.Filtro{Grupo: grupo})
				if err != nil {
					return err
				}
				tem = len(lista) > 0
				grupoTem[grupo] = tem
			}
			if tem {
				return errorsuc.NewValidationError(fmt.Sprintf("%s: informe o instrumento usado (grupo %s)", c.Nome, grupo))
			}
			continue
		}
		i, ok := instr[*r.InstrumentoID]
		if !ok {
			return errorsuc.NewValidationError(fmt.Sprintf("o instrumento %d não existe nesta empresa", *r.InstrumentoID))
		}
		if err := i.PodeMedir(grupo, hoje); err != nil {
			return errorsuc.NewValidationError(c.Nome + ": " + err.Error())
		}
	}
	return nil
}

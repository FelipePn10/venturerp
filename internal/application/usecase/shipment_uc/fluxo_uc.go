package shipment_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
)

// Fluxo guiado da expedição (melhoria 6): pedido → separação → conferência →
// NF-e → despacho numa tela só. O estado de cada romaneio vem amarrado ao da
// nota (cancelar a nota reabre o romaneio — ver EfeitosNFe) e a resposta diz a
// próxima ação, com o endpoint que a executa.

// SituacaoDaNota diz o status da NF-e (fiscal_exits.status).
type SituacaoDaNota interface {
	SituacaoDaNota(ctx context.Context, exitID int64) (string, error)
}

// Etapa do romaneio no fluxo.
type Etapa struct {
	Nome     string     `json:"name"`
	Situacao string     `json:"state"` // FEITA, ATUAL, PENDENTE
	Quando   *time.Time `json:"at,omitempty"`
}

// Acao é o próximo passo, com o endpoint que o executa.
type Acao struct {
	Codigo    string `json:"code"`
	Descricao string `json:"description"`
	Metodo    string `json:"method"`
	Caminho   string `json:"path"`
}

// RomaneioNoFluxo é um romaneio do pedido e onde ele está.
type RomaneioNoFluxo struct {
	Code        int64   `json:"code"`
	Status      string  `json:"status"`
	NFeID       *int64  `json:"nfe_id,omitempty"`
	NFeNumero   *int64  `json:"nfe_number,omitempty"`
	NFeStatus   string  `json:"nfe_status,omitempty"`
	Etapas      []Etapa `json:"steps"`
	ProximaAcao *Acao   `json:"next_action,omitempty"`
}

// FluxoDaExpedicao é o pedido inteiro.
type FluxoDaExpedicao struct {
	SalesOrderCode  int64             `json:"sales_order_code"`
	PendenteExpedir decimal.Decimal   `json:"pending_to_ship"`
	EmRomaneio      decimal.Decimal   `json:"in_open_shipments"`
	Romaneios       []RomaneioNoFluxo `json:"shipments"`
	ProximaAcao     *Acao             `json:"next_action,omitempty"`
}

type FluxoDaExpedicaoUseCase struct {
	Repo    repository.ShipmentRepository
	Pedidos SalesOrderReader
	Notas   SituacaoDaNota
}

// Fluxo monta o fluxo do pedido de venda.
func (uc *FluxoDaExpedicaoUseCase) Fluxo(ctx context.Context, pedido int64) (*FluxoDaExpedicao, error) {
	so, err := uc.Pedidos.GetByCode(ctx, pedido)
	if err != nil {
		return nil, err
	}
	out := &FluxoDaExpedicao{SalesOrderCode: pedido, Romaneios: []RomaneioNoFluxo{}}
	for _, it := range so.Items {
		if it.PendingQty.IsPositive() {
			out.PendenteExpedir = out.PendenteExpedir.Add(it.PendingQty)
		}
	}
	lista, err := uc.Repo.ListBySalesOrder(ctx, pedido)
	if err != nil {
		return nil, err
	}
	for _, resumo := range lista {
		if resumo.Status == entity.ShipmentStatusCancelled {
			continue
		}
		// A listagem não traz as linhas: lê o romaneio inteiro.
		s, err := uc.Repo.GetByCode(ctx, resumo.Code)
		if err != nil {
			return nil, err
		}
		r := RomaneioNoFluxo{Code: s.Code, Status: string(s.Status), NFeID: s.FiscalExitID, NFeNumero: s.NFeNumber}
		if s.FiscalExitID != nil && uc.Notas != nil {
			if r.NFeStatus, err = uc.Notas.SituacaoDaNota(ctx, *s.FiscalExitID); err != nil {
				return nil, err
			}
		}
		r.Etapas, r.ProximaAcao = etapasDoRomaneio(s, r.NFeStatus)
		if s.Status != entity.ShipmentStatusShipped {
			for _, it := range s.Items {
				out.EmRomaneio = out.EmRomaneio.Add(it.Quantity)
			}
		}
		if out.ProximaAcao == nil && r.ProximaAcao != nil {
			out.ProximaAcao = r.ProximaAcao
		}
		out.Romaneios = append(out.Romaneios, r)
	}
	// Falta expedir e nada está em romaneio aberto: o próximo passo é montar um.
	if out.ProximaAcao == nil && out.PendenteExpedir.GreaterThan(out.EmRomaneio) {
		out.ProximaAcao = &Acao{Codigo: "MONTAR_ROMANEIO", Descricao: "Montar o romaneio com o saldo a expedir do pedido",
			Metodo: "POST", Caminho: "/api/shipments/auto-fill/sales-order"}
	}
	return out, nil
}

// etapasDoRomaneio: separação → conferência → NF-e → despacho. A NF-e pode vir
// antes da separação (faturar antes de separar), então ela vale por si.
func etapasDoRomaneio(s *entity.Shipment, nfe string) ([]Etapa, *Acao) {
	acao := func(codigo, desc, metodo, caminho string) *Acao {
		return &Acao{Codigo: codigo, Descricao: desc, Metodo: metodo, Caminho: fmt.Sprintf(caminho, s.Code)}
	}
	separado := s.Status == entity.ShipmentStatusSeparated || s.Status == entity.ShipmentStatusConferred || s.Status == entity.ShipmentStatusShipped
	conferido := s.Status == entity.ShipmentStatusConferred || s.Status == entity.ShipmentStatusShipped
	autorizada := nfe == "AUTHORIZED"
	despachado := s.Status == entity.ShipmentStatusShipped
	sit := func(feito bool, atual bool) string {
		switch {
		case feito:
			return "FEITA"
		case atual:
			return "ATUAL"
		}
		return "PENDENTE"
	}
	var prox *Acao
	switch {
	case despachado:
	case !separado:
		prox = acao("SEPARAR", "Separar (reserva o estoque)", "POST", "/api/shipments/%d/separate")
	case !conferido:
		prox = acao("CONFERIR", "Conferir as quantidades e encerrar a conferência", "POST", "/api/shipments/%d/confer")
	case s.FiscalExitID == nil:
		prox = acao("EMITIR_NFE", "Emitir a NF-e do romaneio", "POST", "/api/shipments/%d/nfe")
	case !autorizada:
		prox = &Acao{Codigo: "AUTORIZAR_NFE", Descricao: "Transmitir a NF-e à SEFAZ", Metodo: "POST", Caminho: fmt.Sprintf("/api/fiscal/exits/%d/authorize", *s.FiscalExitID)}
	default:
		prox = acao("DESPACHAR", "Despachar a carga", "POST", "/api/shipments/%d/ship")
	}
	etapas := []Etapa{
		{Nome: "SEPARACAO", Situacao: sit(separado, !separado), Quando: s.SeparatedAt},
		{Nome: "CONFERENCIA", Situacao: sit(conferido, separado && !conferido), Quando: s.ConferredAt},
		{Nome: "NFE", Situacao: sit(autorizada, conferido && !autorizada)},
		{Nome: "DESPACHO", Situacao: sit(despachado, conferido && autorizada && !despachado), Quando: s.ShippedAt},
	}
	return etapas, prox
}

// SituacaoDaNotaFunc adapta uma função ao SituacaoDaNota (montagem na API).
type SituacaoDaNotaFunc func(ctx context.Context, exitID int64) (string, error)

func (f SituacaoDaNotaFunc) SituacaoDaNota(ctx context.Context, exitID int64) (string, error) {
	return f(ctx, exitID)
}

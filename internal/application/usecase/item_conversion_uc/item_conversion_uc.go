package item_conversion_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

// ErrNoConversion is returned when no path is registered.
var ErrNoConversion = errors.New("não existe conversão cadastrada entre essas unidades para o item — cadastre em Conversões por Item (VSUP0110)")

// ItemConversionUseCase é a VSUP0110 e, ao mesmo tempo, o conversor de
// unidades (ports.UOMConverter) usado por compra e estrutura.
type ItemConversionUseCase struct {
	repo repository.ManagedItemConversionRepository
	auth ports.AuthService
	// items resolve o código de negócio do item (texto) para a chave legada.
	items any
	now   func() time.Time
}

func NewItemConversionUseCase(repo repository.ManagedItemConversionRepository, auth ports.AuthService, items ...any) *ItemConversionUseCase {
	uc := &ItemConversionUseCase{repo: repo, auth: auth, now: time.Now}
	if len(items) > 0 {
		uc.items = items[0]
	}
	return uc
}

// ResolveItem devolve a chave legada do item a partir do código de negócio.
func (uc *ItemConversionUseCase) ResolveItem(ctx context.Context, code request.TextCode) (int64, error) {
	item, err := itemresolution.Resolve(ctx, uc.items, code)
	if err != nil {
		return 0, err
	}
	return int64(item.Code), nil
}

// legacyItemFinder é a fatia do repositório de itens usada para enriquecer a
// resposta a partir da chave legada.
type legacyItemFinder interface {
	FindItemByCode(context.Context, valueobject.ItemCode) (*itementity.Item, error)
}

func (uc *ItemConversionUseCase) itemOf(ctx context.Context, legacy int64) *itementity.Item {
	finder, ok := uc.items.(legacyItemFinder)
	if !ok || legacy <= 0 {
		return nil
	}
	item, err := finder.FindItemByCode(ctx, valueobject.ItemCode(legacy))
	if err != nil {
		return nil
	}
	return item
}

func (uc *ItemConversionUseCase) actor(ctx context.Context) (uuid.UUID, error) {
	if uc.auth == nil {
		return uuid.Nil, nil
	}
	return uc.auth.UserID(ctx)
}

func (uc *ItemConversionUseCase) hoje() time.Time {
	n := uc.now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

func parseData(s, campo string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s inválida: use AAAA-MM-DD", campo))
	}
	return &t, nil
}

// fracao escolhe entre a fração (quantidades) e o fator único.
func fracao(fator decimal.Decimal, de, para *decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if de != nil && para != nil && de.IsPositive() && para.IsPositive() {
		return *de, *para
	}
	return decimal.NewFromInt(1), fator
}

func (uc *ItemConversionUseCase) checarCatalogo(ctx context.Context, itemCode int64, unidades ...string) error {
	faltam, err := uc.repo.MissingUnits(ctx, itemCode, unidades...)
	if err != nil {
		return err
	}
	if len(faltam) > 0 {
		return errorsuc.NewValidationError(fmt.Sprintf("a unidade %s não existe (ou está inativa) no catálogo da empresa: cadastre-a na aba Unidades antes de usá-la",
			strings.Join(faltam, ", ")))
	}
	return nil
}

// checarMascara: conversão de máscara só para máscara que existe no item —
// uma máscara digitada errada criaria uma conversão que nunca é usada.
func (uc *ItemConversionUseCase) checarMascara(ctx context.Context, itemCode int64, mask string) error {
	if mask == "" {
		return nil
	}
	m, ok := uc.repo.(interface {
		MaskExists(ctx context.Context, itemCode int64, mask string) (bool, error)
	})
	if !ok {
		return nil
	}
	existe, err := m.MaskExists(ctx, itemCode, mask)
	if err != nil {
		return err
	}
	if !existe {
		return errorsuc.NewValidationError(fmt.Sprintf("a máscara %q não existe para este item: gere-a no configurador ou deixe em branco para valer para o item todo", mask))
	}
	return nil
}

func sobrepoe(aIni time.Time, aFim *time.Time, bIni time.Time, bFim *time.Time) bool {
	if aFim != nil && aFim.Before(bIni) {
		return false
	}
	if bFim != nil && bFim.Before(aIni) {
		return false
	}
	return true
}

// mesmoFato: mesmo item/máscara e mesmo par em qualquer sentido.
func mesmoFato(a, b *entity.ItemUnitConversion) bool {
	return a.ItemCode == b.ItemCode && a.Mask == b.Mask &&
		((a.FromUOM == b.FromUOM && a.ToUOM == b.ToUOM) || (a.FromUOM == b.ToUOM && a.ToUOM == b.FromUOM))
}

// consistencia recusa a conversão que contradiz o caminho já existente.
func (uc *ItemConversionUseCase) consistencia(ctx context.Context, c *entity.ItemUnitConversion) error {
	arestas, err := uc.repo.Graph(ctx, c.ItemCode, c.Mask, "", c.ValidFrom)
	if err != nil {
		return err
	}
	nova := service.Aresta{De: c.FromUOM, Para: c.ToUOM, QtdDe: c.FromQty, QtdPara: c.ToQty, Origem: service.OrigemItem}
	if conf := service.VerificarConsistencia(arestas, nova); conf != nil {
		return errorsuc.NewValidationError(conf.Error())
	}
	return nil
}

// Create cadastra uma conversão. O par (nos dois sentidos) só pode ter uma
// versão vigente por período: para trocar o fator, use Update.
func (uc *ItemConversionUseCase) Create(ctx context.Context, dto request.CreateItemConversionDTO) (*response.ItemUnitConversionResponse, error) {
	actor, err := uc.actor(ctx)
	if err != nil {
		return nil, err
	}
	if actor != uuid.Nil {
		dto.CreatedBy = actor
	}
	itemCode, err := uc.ResolveItem(ctx, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	de, para := fracao(dto.Factor, dto.FromQuantity, dto.ToQuantity)
	c, err := entity.NewItemUnitConversionFraction(itemCode, dto.Mask, dto.FromUOM, dto.ToUOM, de, para, dto.RoundingPercent, dto.ToleranceValue, dto.ToleranceType, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	if inicio, err := parseData(dto.ValidFrom, "data de início da vigência"); err != nil {
		return nil, err
	} else if inicio != nil {
		c.ValidFrom = *inicio
	}
	if dto.Origin != "" {
		c.Origin = dto.Origin
	}
	c.Notes = limparTexto(dto.Notes)
	if err := uc.checarCatalogo(ctx, itemCode, c.FromUOM, c.ToUOM); err != nil {
		return nil, err
	}
	if err := uc.checarMascara(ctx, itemCode, c.Mask); err != nil {
		return nil, err
	}
	versoes, err := uc.repo.ListVersions(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	for _, v := range versoes {
		if mesmoFato(v, c) && sobrepoe(v.ValidFrom, v.ValidTo, c.ValidFrom, nil) {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("já existe conversão de %s para %s neste item (%s) — use Alterar para trocar o fator ou abrir uma nova vigência",
				v.FromUOM, v.ToUOM, descreverFracao(v)))
		}
	}
	if err := uc.consistencia(ctx, c); err != nil {
		return nil, err
	}
	ev := &entity.Event{ItemCode: itemCode, Action: entity.EventoCriada, CreatedBy: c.CreatedBy}
	created, err := uc.repo.CreateWithEvent(ctx, c, ev)
	if err != nil {
		return nil, err
	}
	out := toItemConversionResponse(created, uc.itemOf(ctx, itemCode), uc.hoje())
	out.Warnings = append(append(uc.avisosDeUso(ctx, created), uc.segundaUnidade(ctx, created)...), uc.avisoEmbalagem(ctx, created)...)
	return out, nil
}

// sincronizadorDeSegundaUnidade é opcional no repositório (os fakes de teste
// não o têm): grava no cadastro do item o fator da segunda unidade.
type sincronizadorDeSegundaUnidade interface {
	SincronizarSegundaUnidade(ctx context.Context, itemCode int64, de, para string, fator decimal.Decimal) (*decimal.Decimal, bool, error)
}

// segundaUnidade mantém o cadastro do item (VENT0200) igual à conversão do par
// estoque ↔ segunda unidade que vale hoje. Os dois eram fontes separadas: o
// gatilho do estoque congelava o peso com o fator do cadastro enquanto compra
// e nota usavam o da VSUP0110. A conversão de máscara ou com vigência futura
// não mexe no cadastro (ele tem um fator só, o de hoje).
//
// Roda depois de a conversão estar gravada: uma falha aqui vira aviso na
// resposta (a conversão vale) em vez de um erro que faria a pessoa gravar de novo.
func (uc *ItemConversionUseCase) segundaUnidade(ctx context.Context, c *entity.ItemUnitConversion) []string {
	if c == nil || c.Mask != "" || c.ValidFrom.After(uc.hoje()) || (c.ValidTo != nil && c.ValidTo.Before(uc.hoje())) {
		return nil
	}
	novo := service.Aresta{De: c.FromUOM, Para: c.ToUOM, QtdDe: c.FromQty, QtdPara: c.ToQty}.Fator()
	s, ok := uc.repo.(sincronizadorDeSegundaUnidade)
	if !ok {
		return uc.avisoSegundaUnidade(ctx, c, novo)
	}
	antes, mudou, err := s.SincronizarSegundaUnidade(ctx, c.ItemCode, c.FromUOM, c.ToUOM, novo)
	if err != nil {
		return []string{fmt.Sprintf("a conversão foi gravada, mas o fator da segunda unidade do cadastro do item não foi atualizado (%v) — confira-o no cadastro do item", err)}
	}
	if !mudou || antes == nil {
		return nil
	}
	return []string{fmt.Sprintf("o fator da segunda unidade no cadastro do item (estoque em duas unidades) era %s e passou a seguir esta conversão; os movimentos já gravados mantêm o peso que tinham",
		service.Numero(*antes))}
}

// fatorDaOrigem: o fator de/para que o cadastro do item (segunda unidade ou
// embalagem) dá ao mesmo par da conversão, em qualquer sentido.
func fatorDaOrigem(arestas []service.Aresta, origem service.Origem, de, para string) (decimal.Decimal, bool) {
	for _, a := range arestas {
		if a.Origem != origem {
			continue
		}
		switch {
		case service.Normalizar(a.De) == de && service.Normalizar(a.Para) == para:
			return a.Fator(), true
		case service.Normalizar(a.De) == para && service.Normalizar(a.Para) == de && !a.Fator().IsZero():
			return decimal.NewFromInt(1).DivRound(a.Fator(), 10), true
		}
	}
	return decimal.Zero, false
}

func diverge(novo, cadastro decimal.Decimal) bool {
	return !cadastro.IsZero() && novo.Sub(cadastro).Abs().DivRound(cadastro, 10).GreaterThan(service.ToleranciaConsistencia)
}

// avisoSegundaUnidade: sem como sincronizar, ao menos avisa a divergência.
func (uc *ItemConversionUseCase) avisoSegundaUnidade(ctx context.Context, c *entity.ItemUnitConversion, novo decimal.Decimal) []string {
	arestas, err := uc.repo.Graph(ctx, c.ItemCode, "", "", time.Time{})
	if err != nil {
		return nil
	}
	cadastro, ok := fatorDaOrigem(arestas, service.OrigemSegundaUnidade, c.FromUOM, c.ToUOM)
	if !ok || !diverge(novo, cadastro) {
		return nil
	}
	return []string{fmt.Sprintf("o cadastro do item (segunda unidade) diz 1 %s = %s %s; vale esta conversão (1 %s = %s %s) — corrija o fator da segunda unidade no cadastro do item",
		c.FromUOM, service.Numero(cadastro), c.ToUOM, c.FromUOM, service.Numero(novo), c.ToUOM)}
}

// avisoEmbalagem: a embalagem do item (VENT0200 › Embalagens e GTIN) diz
// outra quantidade para o mesmo par. A conversão explícita ganha dela no
// grafo; a embalagem não é alterada sozinha porque é também o GTIN que vai na
// nota de saída — quem decide qual está certa é a pessoa.
func (uc *ItemConversionUseCase) avisoEmbalagem(ctx context.Context, c *entity.ItemUnitConversion) []string {
	if c == nil || c.Mask != "" {
		return nil
	}
	arestas, err := uc.repo.Graph(ctx, c.ItemCode, "", "", time.Time{})
	if err != nil {
		return nil
	}
	cadastro, ok := fatorDaOrigem(arestas, service.OrigemEmbalagem, c.FromUOM, c.ToUOM)
	novo := service.Aresta{De: c.FromUOM, Para: c.ToUOM, QtdDe: c.FromQty, QtdPara: c.ToQty}.Fator()
	if !ok || !diverge(novo, cadastro) {
		return nil
	}
	return []string{fmt.Sprintf("a embalagem do item (cadastro do item › Embalagens e GTIN) diz 1 %s = %s %s, e esta conversão diz %s — nas notas e nos pedidos vale a conversão; se a embalagem estiver errada, corrija-a no cadastro do item",
		c.FromUOM, service.Numero(cadastro), c.ToUOM, service.Numero(novo))}
}

// avisosDeUso: a conversão não toca nenhuma unidade que o item usa.
func (uc *ItemConversionUseCase) avisosDeUso(ctx context.Context, c *entity.ItemUnitConversion) []string {
	u, err := uc.repo.UnitsOfItem(ctx, c.ItemCode)
	if err != nil || u == nil {
		return nil
	}
	usadas := map[string]bool{u.Estoque: true, u.Compra: true, u.CompraContabil: true, u.VendaContabil: true, u.Secundaria: true}
	for _, x := range u.Usos {
		usadas[x.Unidade] = true
	}
	delete(usadas, "")
	if usadas[c.FromUOM] || usadas[c.ToUOM] {
		return nil
	}
	return []string{fmt.Sprintf("o item é estocado em %s e nenhuma das duas unidades (%s, %s) aparece no cadastro, nas estruturas, nos fornecedores ou nos pedidos dele — confira se é o item certo",
		u.Estoque, c.FromUOM, c.ToUOM)}
}

func limparTexto(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func descreverFracao(c *entity.ItemUnitConversion) string {
	return fmt.Sprintf("%s %s = %s %s", service.Numero(c.FromQty), c.FromUOM, service.Numero(c.ToQty), c.ToUOM)
}

// Update altera uma conversão. Ver UpdateItemConversionDTO para a diferença
// entre corrigir o fator e abrir nova vigência.
func (uc *ItemConversionUseCase) Update(ctx context.Context, id int64, dto request.UpdateItemConversionDTO) (*response.ItemUnitConversionResponse, error) {
	actor, err := uc.actor(ctx)
	if err != nil {
		return nil, err
	}
	atual, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	nova := *atual
	if actor != uuid.Nil {
		nova.UpdatedBy = &actor
	}
	de, para := atual.FromQty, atual.ToQty
	if dto.FromQuantity != nil || dto.ToQuantity != nil || dto.Factor != nil {
		f := atual.Factor
		if dto.Factor != nil {
			f = *dto.Factor
		}
		de, para = fracao(f, dto.FromQuantity, dto.ToQuantity)
	}
	if err := nova.SetFraction(de, para); err != nil {
		return nil, err
	}
	arred, tol, tipo := atual.RoundingPercent, atual.ToleranceValue, atual.ToleranceType
	if dto.RoundingPercent != nil {
		arred = *dto.RoundingPercent
	}
	if dto.ToleranceValue != nil {
		tol = *dto.ToleranceValue
	}
	if dto.ToleranceType != nil {
		tipo = *dto.ToleranceType
	}
	if err := nova.SetPolicy(arred, tol, tipo); err != nil {
		return nil, err
	}
	if dto.Notes != nil {
		nova.Notes = limparTexto(dto.Notes)
	}
	inicio, err := parseData(dto.ValidFrom, "data de início da nova vigência")
	if err != nil {
		return nil, err
	}
	mudouFator := !atual.SameFraction(nova.FromQty, nova.ToQty)
	motivo := strings.TrimSpace(dto.Reason)
	if mudouFator && motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo da troca do fator: ela fica no histórico da conversão")
	}
	if inicio != nil && !mudouFator {
		return nil, errorsuc.NewValidationError("nova vigência só faz sentido com um fator diferente do atual")
	}
	var avisos []string
	if mudouFator {
		if inicio != nil {
			nova.ValidFrom = *inicio
		}
		if err := uc.consistencia(ctx, &nova); err != nil {
			return nil, err
		}
		if avisos, err = uc.avisosDeTroca(ctx, atual, &nova, inicio); err != nil {
			return nil, err
		}
	}
	var r *string
	if motivo != "" {
		r = &motivo
	}
	ev := &entity.Event{ItemCode: atual.ItemCode, Action: entity.EventoAlterada, Before: atual.Snapshot(), Reason: r, CreatedBy: actor}
	var salvo *entity.ItemUnitConversion
	if inicio != nil {
		if !inicio.After(atual.ValidFrom) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a nova vigência precisa começar depois de %s, o início da versão atual",
				atual.ValidFrom.Format("02/01/2006")))
		}
		if atual.ValidTo != nil && inicio.After(*atual.ValidTo) {
			return nil, errorsuc.NewValidationError("a versão atual já termina antes dessa data: cadastre uma conversão nova para o período")
		}
		nova.ID, nova.CreatedBy, nova.ValidTo = 0, actor, atual.ValidTo
		ev.Action = entity.EventoNovaVigencia
		salvo, err = uc.repo.NewVersion(ctx, atual, inicio.AddDate(0, 0, -1), &nova, ev)
	} else {
		salvo, err = uc.repo.Update(ctx, &nova, ev)
	}
	if err != nil {
		return nil, err
	}
	out := toItemConversionResponse(salvo, uc.itemOf(ctx, salvo.ItemCode), uc.hoje())
	if mudouFator {
		avisos = append(append(avisos, uc.segundaUnidade(ctx, salvo)...), uc.avisoEmbalagem(ctx, salvo)...)
	}
	out.Warnings = avisos
	return out, nil
}

// avisosDeTroca: o que a troca de fator deixa para trás.
func (uc *ItemConversionUseCase) avisosDeTroca(ctx context.Context, atual, nova *entity.ItemUnitConversion, inicio *time.Time) ([]string, error) {
	imp, err := uc.repo.Impact(ctx, atual)
	if err != nil {
		return nil, err
	}
	var avisos []string
	hoje := uc.hoje()
	mesmoMes := inicio == nil || (inicio.Year() == hoje.Year() && inicio.Month() == hoje.Month())
	if imp.NotasAprovadasNoMes > 0 && mesmoMes {
		avisos = append(avisos, fmt.Sprintf("o item já teve %d nota(s) aprovada(s) neste mês com o fator anterior; o SPED (registro 0220) espera um fator por mês — o ideal é abrir nova vigência a partir do dia 1º do próximo mês", imp.NotasAprovadasNoMes))
	}
	if imp.PedidosAbertos > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linha(s) de pedido de compra em aberto foram convertidas com o fator anterior e não mudam sozinhas", imp.PedidosAbertos))
	}
	if imp.NotasPendentes > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linha(s) de nota de entrada pendente terão a conversão reaplicada na aprovação", imp.NotasPendentes))
	}
	if n := uc.estruturasDesatualizadas(ctx, nova, imp); n > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linha(s) de estrutura guardaram um fator diferente do novo: revise-as no cadastro da estrutura (o fator fica congelado de propósito)", n))
	}
	return avisos, nil
}

// estruturasDesatualizadas compara o fator congelado em cada unidade das
// estruturas com o fator que o grafo dá considerando a conversão `nova`.
func (uc *ItemConversionUseCase) estruturasDesatualizadas(ctx context.Context, nova *entity.ItemUnitConversion, imp *repository.Impacto) int64 {
	// A estrutura congela o fator sem máscara: conversão de máscara não a afeta.
	if nova == nil || len(imp.FatoresCongelados) == 0 || nova.Mask != "" {
		return 0
	}
	unidades, err := uc.repo.UnitsOfItem(ctx, nova.ItemCode)
	if err != nil {
		return 0
	}
	arestas, err := uc.repo.Graph(ctx, nova.ItemCode, "", "", time.Time{})
	if err != nil {
		return 0
	}
	filtradas := arestas[:0:0]
	for _, a := range arestas {
		if a.Origem != service.OrigemPadrao && ((a.De == nova.FromUOM && a.Para == nova.ToUOM) || (a.De == nova.ToUOM && a.Para == nova.FromUOM)) {
			continue
		}
		filtradas = append(filtradas, a)
	}
	arestas = append(filtradas, service.Aresta{De: nova.FromUOM, Para: nova.ToUOM, QtdDe: nova.FromQty, QtdPara: nova.ToQty, Origem: service.OrigemItem})
	var n int64
	for _, f := range imp.FatoresCongelados {
		c, ok := service.Resolver(arestas, f.Unidade, unidades.Estoque)
		if !ok {
			n += f.Linhas
			continue
		}
		if !c.Fator().Round(6).Equal(f.Fator.Round(6)) {
			n += f.Linhas
		}
	}
	return n
}

// Delete exclui (inativa) a conversão, registrando o motivo.
func (uc *ItemConversionUseCase) Delete(ctx context.Context, id int64) error {
	return uc.DeleteWithReason(ctx, id, "excluída por integração, sem motivo informado")
}

func (uc *ItemConversionUseCase) DeleteWithReason(ctx context.Context, id int64, reason string) error {
	actor, err := uc.actor(ctx)
	if err != nil {
		return err
	}
	atual, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	t := strings.TrimSpace(reason)
	if t == "" {
		return errorsuc.NewValidationError("informe o motivo da exclusão: ele fica no histórico da conversão")
	}
	r := &t
	return uc.repo.Deactivate(ctx, id, &entity.Event{ItemCode: atual.ItemCode, ConversionID: &atual.ID,
		Action: entity.EventoExcluida, Before: atual.Snapshot(), Reason: r, CreatedBy: actor})
}

func (uc *ItemConversionUseCase) ListByItem(ctx context.Context, itemCode int64) ([]*response.ItemUnitConversionResponse, error) {
	list, err := uc.repo.ListByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toItemConversionResponses(list, uc.itemOf(ctx, itemCode), uc.hoje()), nil
}

// ListVersions inclui as versões encerradas.
func (uc *ItemConversionUseCase) ListVersions(ctx context.Context, itemCode int64) ([]*response.ItemUnitConversionResponse, error) {
	list, err := uc.repo.ListVersions(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toItemConversionResponses(list, uc.itemOf(ctx, itemCode), uc.hoje()), nil
}

func (uc *ItemConversionUseCase) History(ctx context.Context, itemCode int64) ([]*response.ConversionEventResponse, error) {
	evs, err := uc.repo.Events(ctx, itemCode, 200)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ConversionEventResponse, 0, len(evs))
	for _, e := range evs {
		out = append(out, &response.ConversionEventResponse{ID: e.ID, ConversionID: e.ConversionID, LotConversionID: e.LotConversionID,
			Action: e.Action, Before: e.Before, After: e.After, Reason: e.Reason, CreatedBy: e.CreatedBy, CreatedByName: e.CreatedByName, CreatedAt: e.CreatedAt})
	}
	return out, nil
}

// Impact mostra o que depende da conversão antes de alterá-la ou excluí-la.
func (uc *ItemConversionUseCase) Impact(ctx context.Context, id int64) (*response.ConversionImpactResponse, error) {
	c, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	imp, err := uc.repo.Impact(ctx, c)
	if err != nil {
		return nil, err
	}
	out := &response.ConversionImpactResponse{ConversionID: id, StructureLines: imp.EstruturasNaUnidade,
		OpenPurchaseLines: imp.PedidosAbertos, PendingEntryLines: imp.NotasPendentes, SupplierLinks: imp.VinculosFornecedor,
		ApprovedEntriesInMonth: imp.NotasAprovadasNoMes, Messages: []string{}}
	out.StructureLinesStale = uc.estruturasDesatualizadas(ctx, c, imp)
	if imp.EstruturasNaUnidade > 0 {
		out.Messages = append(out.Messages, fmt.Sprintf("%d linha(s) de estrutura usam o item em %s ou %s; elas guardam o fator do dia em que foram gravadas", imp.EstruturasNaUnidade, c.FromUOM, c.ToUOM))
	}
	if imp.PedidosAbertos > 0 {
		out.Messages = append(out.Messages, fmt.Sprintf("%d linha(s) de pedido de compra em aberto compram nessa unidade", imp.PedidosAbertos))
	}
	if imp.NotasPendentes > 0 {
		out.Messages = append(out.Messages, fmt.Sprintf("%d linha(s) de nota de entrada pendente: sem a conversão, a aprovação fica bloqueada", imp.NotasPendentes))
	}
	if imp.VinculosFornecedor > 0 {
		out.Messages = append(out.Messages, fmt.Sprintf("%d vínculo(s) com fornecedor recebem o item nessa unidade", imp.VinculosFornecedor))
	}
	if imp.NotasAprovadasNoMes > 0 {
		out.Messages = append(out.Messages, fmt.Sprintf("%d nota(s) aprovada(s) neste mês usaram o fator atual (SPED 0220)", imp.NotasAprovadasNoMes))
	}
	return out, nil
}

// Coverage é a matriz de cobertura: cada papel de unidade do item e se ele
// chega à unidade de estoque.
func (uc *ItemConversionUseCase) Coverage(ctx context.Context, itemCode int64, mask string) (*response.UnitCoverageResponse, error) {
	u, err := uc.repo.UnitsOfItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	arestas, err := uc.repo.Graph(ctx, itemCode, mask, "", time.Time{})
	if err != nil {
		return nil, err
	}
	out := &response.UnitCoverageResponse{StockUnit: u.Estoque, AcceptsFractional: u.AceitaFracao, Rows: []response.UnitCoverageRow{}}
	if it := uc.itemOf(ctx, itemCode); it != nil {
		out.ItemCode, out.ItemName = string(it.BusinessCode), it.Name
	}
	todas := []string{u.Estoque, u.Compra, u.CompraContabil, u.VendaContabil, u.Secundaria}
	for _, x := range u.Usos {
		todas = append(todas, x.Unidade)
	}
	faltam, err := uc.repo.MissingUnits(ctx, itemCode, todas...)
	if err != nil {
		return nil, err
	}
	fora := map[string]bool{}
	for _, f := range faltam {
		fora[f] = true
	}
	linha := func(papel, unidade string, ocorr int64) {
		if unidade == "" {
			return
		}
		row := response.UnitCoverageRow{Role: papel, Unit: unidade, Occurrences: ocorr}
		switch {
		case unidade == u.Estoque:
			row.Status = "BASE"
		case fora[unidade]:
			row.Status = "FORA_DO_CATALOGO"
		default:
			if c, ok := service.Resolver(arestas, unidade, u.Estoque); ok {
				f := c.Fator()
				row.Status, row.Factor, row.Explanation, row.Source = "OK", &f, c.Explicacao(), string(c.OrigemPrincipal())
			} else {
				row.Status = "SEM_CONVERSAO"
				out.Missing++
			}
		}
		out.Rows = append(out.Rows, row)
	}
	linha("ESTOQUE", u.Estoque, 0)
	linha("COMPRA", u.Compra, 0)
	linha("COMPRA_CONTABIL", u.CompraContabil, 0)
	linha("VENDA", u.VendaContabil, 0)
	linha("SECUNDARIA", u.Secundaria, 0)
	for _, x := range u.Usos {
		linha(x.Onde, x.Unidade, x.Quantidade)
	}
	return out, nil
}

// Simulate resolve e explica uma conversão (simulador da tela).
func (uc *ItemConversionUseCase) Simulate(ctx context.Context, itemCode int64, mask, lot, date, from, to string, qty decimal.Decimal) (*response.ConversionSimulationResponse, error) {
	data, err := parseData(date, "data")
	if err != nil {
		return nil, err
	}
	at := uc.hoje()
	if data != nil {
		at = *data
	}
	arestas, err := uc.repo.Graph(ctx, itemCode, mask, lot, at)
	if err != nil {
		return nil, err
	}
	c, ok := service.Resolver(arestas, from, to)
	if !ok {
		return nil, errorsuc.NewNotFoundError(ErrNoConversion.Error())
	}
	out := &response.ConversionSimulationResponse{FromUOM: c.De, ToUOM: c.Para, Quantity: qty, Converted: c.Converter(qty),
		Factor: c.Fator(), Explanation: c.Explicacao(), Source: string(c.OrigemPrincipal()), Date: at.Format("2006-01-02"),
		Lot: strings.TrimSpace(lot), Mask: strings.TrimSpace(mask), Steps: []response.ConversionStepResponse{}}
	if it := uc.itemOf(ctx, itemCode); it != nil {
		out.ItemCode = string(it.BusinessCode)
	}
	for _, p := range c.Passos {
		out.Steps = append(out.Steps, response.ConversionStepResponse{FromUOM: p.De, ToUOM: p.Para, Factor: p.Fator(), Source: string(p.Origem), Inverted: p.Invertido, RecordID: p.ID})
	}
	fracionado, err := uc.repo.AcceptsFractional(ctx, itemCode)
	if err == nil && !fracionado {
		if r, err := aplicarPolitica(c, out.Converted); err != nil {
			out.RoundingNote = err.Error()
		} else if !r.Equal(out.Converted) {
			out.Rounded = &r
			out.RoundingNote = "o item não aceita fração: a quantidade foi arredondada dentro da política da conversão"
		}
	}
	return out, nil
}

// ─── Lote ────────────────────────────────────────────────────────────────────

func (uc *ItemConversionUseCase) ListLots(ctx context.Context, itemCode int64) ([]*response.LotConversionResponse, error) {
	list, err := uc.repo.ListLotConversions(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	codigo := uc.codigoDeNegocio(ctx, itemCode)
	out := make([]*response.LotConversionResponse, 0, len(list))
	for _, l := range list {
		out = append(out, toLotResponse(l, codigo))
	}
	return out, nil
}

func (uc *ItemConversionUseCase) SaveLot(ctx context.Context, dto request.LotConversionDTO) (*response.LotConversionResponse, error) {
	actor, err := uc.actor(ctx)
	if err != nil {
		return nil, err
	}
	itemCode, err := uc.ResolveItem(ctx, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	l, err := entity.NewLotConversion(itemCode, dto.Lot, dto.FromUOM, dto.ToUOM, dto.FromQuantity, dto.ToQuantity, limparTexto(dto.Notes), actor)
	if err != nil {
		return nil, err
	}
	if err := uc.checarCatalogo(ctx, itemCode, l.FromUOM, l.ToUOM); err != nil {
		return nil, err
	}
	salvo, err := uc.repo.UpsertLotConversion(ctx, l, &entity.Event{ItemCode: itemCode, Action: entity.EventoCriada, CreatedBy: actor})
	if err != nil {
		return nil, err
	}
	out := toLotResponse(salvo, uc.codigoDeNegocio(ctx, itemCode))
	if existe, err := uc.repo.LotExists(ctx, itemCode, l.Lot); err == nil && !existe {
		out.Warnings = []string{fmt.Sprintf("o lote %s ainda não existe no estoque deste item; o fator passa a valer quando ele entrar", l.Lot)}
	}
	return out, nil
}

func (uc *ItemConversionUseCase) DeleteLot(ctx context.Context, id int64, reason string) error {
	actor, err := uc.actor(ctx)
	if err != nil {
		return err
	}
	l, err := uc.repo.GetLotConversion(ctx, id)
	if err != nil {
		return err
	}
	var r *string
	if t := strings.TrimSpace(reason); t != "" {
		r = &t
	}
	return uc.repo.DeactivateLotConversion(ctx, id, &entity.Event{ItemCode: l.ItemCode, LotConversionID: &l.ID,
		Action: entity.EventoExcluida, Before: l.Snapshot(), Reason: r, CreatedBy: actor})
}

func (uc *ItemConversionUseCase) codigoDeNegocio(ctx context.Context, itemCode int64) string {
	if it := uc.itemOf(ctx, itemCode); it != nil {
		return string(it.BusinessCode)
	}
	return fmt.Sprint(itemCode)
}

// ─── ports.UOMConverter ──────────────────────────────────────────────────────

// Path resolve o caminho de conversão (máscara e lote opcionais, data zero = hoje).
func (uc *ItemConversionUseCase) Path(ctx context.Context, itemCode int64, mask, lot string, at time.Time, fromUOM, toUOM string) (*service.Caminho, error) {
	if service.Normalizar(fromUOM) == service.Normalizar(toUOM) {
		c, _ := service.Resolver(nil, fromUOM, toUOM)
		return &c, nil
	}
	arestas, err := uc.repo.Graph(ctx, itemCode, mask, lot, at)
	if err != nil {
		return nil, err
	}
	c, ok := service.Resolver(arestas, fromUOM, toUOM)
	if !ok {
		return nil, nil
	}
	return &c, nil
}

// Factor returns f where 1 fromUOM = f × toUOM.
func (uc *ItemConversionUseCase) Factor(ctx context.Context, itemCode int64, fromUOM, toUOM string) (decimal.Decimal, bool, error) {
	return uc.FactorConfigured(ctx, itemCode, "", fromUOM, toUOM)
}

func (uc *ItemConversionUseCase) FactorConfigured(ctx context.Context, itemCode int64, mask, fromUOM, toUOM string) (decimal.Decimal, bool, error) {
	c, err := uc.Path(ctx, itemCode, mask, "", time.Time{}, fromUOM, toUOM)
	if err != nil || c == nil {
		return decimal.Zero, false, err
	}
	return c.Fator(), true, nil
}

func (uc *ItemConversionUseCase) ConvertQuantity(ctx context.Context, itemCode int64, qty decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error) {
	return uc.ConvertQuantityConfigured(ctx, itemCode, "", qty, fromUOM, toUOM)
}

// ConvertQuantityExact converte pela fração exata do caminho, sem a política de
// arredondamento do item que não aceita fração: a estrutura guarda quanto de
// chapa uma peça consome (0,5 UN), e multiplicar pelo fator já arredondado
// dava 0,5000000008.
func (uc *ItemConversionUseCase) ConvertQuantityExact(ctx context.Context, itemCode int64, qty decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error) {
	c, err := uc.Path(ctx, itemCode, "", "", time.Time{}, fromUOM, toUOM)
	if err != nil || c == nil {
		return decimal.Zero, false, err
	}
	return c.Converter(qty), true, nil
}

// ConvertQuantityConfigured converte e, para item que não aceita fração,
// aplica a política de arredondamento/tolerância da conversão do item.
func (uc *ItemConversionUseCase) ConvertQuantityConfigured(ctx context.Context, itemCode int64, mask string, qty decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error) {
	if service.Normalizar(fromUOM) == service.Normalizar(toUOM) {
		return qty, true, nil
	}
	c, err := uc.Path(ctx, itemCode, mask, "", time.Time{}, fromUOM, toUOM)
	if err != nil || c == nil {
		return decimal.Zero, false, err
	}
	converted := c.Converter(qty)
	fractional, err := uc.repo.AcceptsFractional(ctx, itemCode)
	if err != nil {
		return decimal.Zero, false, err
	}
	if fractional {
		return converted, true, nil
	}
	r, err := aplicarPolitica(*c, converted)
	if err != nil {
		return decimal.Zero, false, err
	}
	return r, true, nil
}

// aplicarPolitica arredonda para o inteiro mais próximo se a sobra couber no
// arredondamento + tolerância da conversão; senão recusa.
func aplicarPolitica(c service.Caminho, converted decimal.Decimal) (decimal.Decimal, error) {
	arred, tol, tipo := c.PoliticaDeArredondamento()
	rounded := converted.Round(0)
	allowed := converted.Abs().Mul(arred).Div(decimal.NewFromInt(100))
	if tipo == "PERCENT" {
		allowed = allowed.Add(converted.Abs().Mul(tol).Div(decimal.NewFromInt(100)))
	} else {
		allowed = allowed.Add(tol)
	}
	if converted.Sub(rounded).Abs().GreaterThan(allowed) {
		return decimal.Zero, errorsuc.NewValidationError(fmt.Sprintf(
			"o item não aceita fração e a conversão daria %s: ajuste o arredondamento ou a tolerância da conversão", service.Numero(converted)))
	}
	return rounded, nil
}

// ConvertUnitPrice: preço por fromUOM → preço por toUOM.
func (uc *ItemConversionUseCase) ConvertUnitPrice(ctx context.Context, itemCode int64, price decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error) {
	c, err := uc.Path(ctx, itemCode, "", "", time.Time{}, fromUOM, toUOM)
	if err != nil || c == nil {
		return decimal.Zero, false, err
	}
	return c.ConverterPreco(price), true, nil
}

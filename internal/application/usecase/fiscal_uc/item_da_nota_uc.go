package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

// CadastroDeItem é o cadastro de itens (VENT0200) com as suas regras:
// código automático, PDM e classificações conferidos.
type CadastroDeItem interface {
	Execute(ctx context.Context, item *itementity.Item) (*response.ItemResponse, error)
}

// ItemDaNotaUseCase cadastra, a partir de uma linha da nota de entrada, o item
// que ainda não existe — com a descrição, a unidade, a origem e o CEST da nota
// — e devolve o código para a conciliação. O que a nota não diz (grupo PDM,
// classificação fiscal, dados de engenharia) se completa depois em VENT0200.
type ItemDaNotaUseCase struct {
	Docs     repository.FiscalEntryDocumentRepository
	Cadastro CadastroDeItem
	Auth     ports.AuthService
}

type ItemDaNotaDTO struct {
	Nome        string `json:"nome,omitempty"`
	Unidade     string `json:"unidade,omitempty"`      // unidade de estoque (KG, UN, CX...)
	WarehouseID *int64 `json:"warehouse_id,omitempty"` // almoxarifado padrão do item
	// TipoUso: INDUSTRIALIZACAO, CONSUMO ou IMOBILIZADO (padrão: pelo CFOP de entrada).
	TipoUso string `json:"tipo_uso,omitempty"`
	Revenda *bool  `json:"revenda,omitempty"`
}

type ItemDaNotaResultado struct {
	ItemCode int64  `json:"item_code"`
	Codigo   string `json:"codigo"`
	Nome     string `json:"nome"`
	Unidade  string `json:"unidade"`
	Aviso    string `json:"aviso"`
}

// SugestaoItemDaNota: o que a tela pré-preenche (e o que vale quando o corpo
// vem vazio), deduzido da linha da nota.
func SugestaoItemDaNota(it *entity.FiscalEntryItem) ItemDaNotaDTO {
	cfop := deref(it.CfopEntrada)
	if cfop == "" {
		cfop = it.Cfop
	}
	tipo, revenda := "INDUSTRIALIZACAO", false
	if len(cfop) == 4 {
		switch cfop[1:] {
		case "556", "407", "653":
			tipo = "CONSUMO"
		case "551", "406":
			tipo = "IMOBILIZADO"
		case "102", "403", "117", "118", "121":
			revenda = true
		}
	}
	return ItemDaNotaDTO{Nome: strings.TrimSpace(deref(it.Description)), Unidade: UnidadeDoCadastro(deref(it.UOM)),
		WarehouseID: it.WarehouseID, TipoUso: tipo, Revenda: &revenda}
}

// UnidadeDoCadastro traduz a unidade da nota para a do cadastro; vazio quando
// não há equivalente (a tela pede para escolher).
func UnidadeDoCadastro(u string) string {
	u = strings.ToUpper(strings.TrimSpace(u))
	switch u {
	case "UND", "UNID", "UNIDADE", "UNI":
		u = "UN"
	case "PCA", "PECA", "PÇ", "PCS":
		u = "PC"
	case "TON", "TN", "T":
		u = "TONELADA"
	case "LT", "LTS":
		u = "L"
	case "MT", "MTS":
		u = "M"
	case "CXA", "CAIXA":
		u = "CX"
	}
	if types.TypeUnitOfMeasurementItem(u).IsValid() {
		return u
	}
	return ""
}

func (uc *ItemDaNotaUseCase) Execute(ctx context.Context, entryID, itemID int64, dto ItemDaNotaDTO) (*ItemDaNotaResultado, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if doc.Status != entity.EntryStatusPending && doc.Status != entity.EntryStatusConferred {
		return nil, errorsuc.NewValidationError("só nota pendente ou conferida tem item a cadastrar")
	}
	var linha *entity.FiscalEntryItem
	for _, it := range doc.Itens {
		if it.ID == itemID {
			linha = it
		}
	}
	if linha == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado na nota %d", itemID, entryID))
	}
	if linha.ItemCode != nil {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("o item %d da nota já está conciliado com o item %d", linha.Sequence, *linha.ItemCode))
	}

	s := SugestaoItemDaNota(linha)
	nome := firstNonEmpty(strings.TrimSpace(dto.Nome), s.Nome)
	if nome == "" {
		return nil, errorsuc.NewValidationError("informe o nome do item")
	}
	unidade := UnidadeDoCadastro(firstNonEmpty(dto.Unidade, s.Unidade))
	if unidade == "" {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a unidade %q da nota não existe no cadastro: escolha a unidade de estoque do item", deref(linha.UOM)))
	}
	almox := dto.WarehouseID
	if almox == nil {
		almox = s.WarehouseID
	}
	if almox == nil {
		return nil, errorsuc.NewValidationError("informe o almoxarifado padrão do item")
	}
	tipoUso := types.INDUSTRIALIZACAO
	switch strings.ToUpper(firstNonEmpty(dto.TipoUso, s.TipoUso)) {
	case "CONSUMO":
		tipoUso = types.CONSUMO
	case "IMOBILIZADO":
		tipoUso = types.IMOBILIZADO
	}
	revenda := s.Revenda != nil && *s.Revenda
	if dto.Revenda != nil {
		revenda = *dto.Revenda
	}

	unid := types.TypeUnitOfMeasurementItem(unidade)
	estrutura := types.INDUSTRIAL
	var tipoVenda *string
	if revenda {
		estrutura = types.COMERCIAL
		v := "REVENDA"
		tipoVenda = &v
	}
	var origem *int
	if o := strings.TrimSpace(deref(linha.Origem)); len(o) == 1 && o[0] >= '0' && o[0] <= '8' {
		v := int(o[0] - '0')
		origem = &v
	}
	var cest *string
	if c := soDigitos(deref(linha.CEST)); len(c) == 7 {
		cest = &c
	}
	userID, _ := uc.Auth.UserID(ctx)
	criador, _ := uuid.Parse(userID.String())
	item, err := itementity.NewItem("", nome, nil, itementity.ItemGeneric,
		itementity.PDM{DescriptionTechnique: nome}, types.LINHA, types.ACTIVE,
		itementity.Warehouse{WarehouseCode: int(*almox), UnitOfMeasurement: unid},
		itementity.Engineering{Type: types.COMPRADO, TypeStruct: estrutura, Weight: valueobject.Weight{Unit: "KG"}},
		itementity.Planning{TypeMRP: types.NORMAL_MRP, Active: true},
		itementity.Supplies{TypeOfUse: tipoUso, PurchaseUOM: &unid},
		itementity.Commercial{SaleType: tipoVenda},
		itementity.Accounting{Origin: origem, CEST: cest, PurchaseUnitOfMeasurement: &unid},
		criador)
	if err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	criado, err := uc.Cadastro.Execute(ctx, item)
	if err != nil {
		return nil, err
	}
	return &ItemDaNotaResultado{ItemCode: criado.LegacyCode, Codigo: criado.Code, Nome: criado.Name, Unidade: unidade,
		Aviso: "Item cadastrado com os dados da nota. Complete grupo PDM, classificação fiscal (NCM " + deref(linha.Ncm) + ") e engenharia em VENT0200."}, nil
}

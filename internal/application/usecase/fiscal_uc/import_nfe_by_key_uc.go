package fiscal_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// ImportNFeByKeyUseCase importa a NF-e de entrada pela chave de acesso:
// baixa o XML da nota recebida na Focus NF-e e segue o mesmo caminho da
// importação do arquivo (conciliação, plano de contas, parcelas). A nota fica
// pendente para conferência — nada entra no financeiro sem aprovação.
type ImportNFeByKeyUseCase struct {
	Upload *UploadNFEEntryUseCase
}

type ImportNFeByKeyDTO struct {
	ChaveAcesso       string  `json:"chave_acesso"`
	AccessKey         string  `json:"access_key"`
	PurchaseOrderCode *int64  `json:"purchase_order_code,omitempty"`
	DataEntrada       *string `json:"data_entrada,omitempty"`
}

func (uc *ImportNFeByKeyUseCase) Execute(ctx context.Context, dto ImportNFeByKeyDTO) (*response.FiscalEntryResponse, error) {
	if uc.Upload == nil {
		return nil, fmt.Errorf("importação de NF-e não configurada")
	}
	if !uc.Upload.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	chave := soDigitos(firstNonEmpty(dto.ChaveAcesso, dto.AccessKey))
	if len(chave) != 44 {
		return nil, errorsuc.NewValidationError("a chave de acesso deve ter exatamente 44 dígitos")
	}
	existente, err := uc.Upload.Docs.FindEntryByChave(ctx, chave)
	if err != nil {
		return nil, err
	}
	if existente != nil {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("esta NF-e já foi importada como entrada %d", existente.ID))
	}
	cfg, err := uc.Upload.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, errorsuc.NewValidationError("o token da Focus NF-e não está configurado — informe-o em Configuração Fiscal (VFIS0100) ou importe o arquivo XML")
	}
	conteudo, err := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente).BaixarXMLNFeRecebida(ctx, chave)
	if err != nil {
		return nil, err
	}
	return uc.Upload.ExecuteFile(ctx, conteudo, request.UploadNFEDTO{
		PurchaseOrderCode: dto.PurchaseOrderCode,
		DataEntrada:       dto.DataEntrada,
	})
}

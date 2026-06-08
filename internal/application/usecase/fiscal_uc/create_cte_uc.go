package fiscal_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type CreateCTeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *CreateCTeUseCase) Execute(ctx context.Context, dto request.CreateCTeDTO) (*entity.FiscalCTe, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	dataEmissao, err := time.Parse("2006-01-02", dto.DataEmissao)
	if err != nil {
		return nil, fmt.Errorf("data_emissao inválida: %w", err)
	}
	dataEntrada, err := time.Parse("2006-01-02", dto.DataEntrada)
	if err != nil {
		return nil, fmt.Errorf("data_entrada inválida: %w", err)
	}

	if dto.TipoRateio == "" {
		dto.TipoRateio = "VALOR"
	}

	var emissionData *string
	if len(dto.EmissionData) > 0 {
		s := string(dto.EmissionData)
		emissionData = &s
	}

	cte := &entity.FiscalCTe{
		NumeroCTe:           dto.NumeroCTe,
		Serie:               dto.Serie,
		DataEmissao:         dataEmissao,
		DataEntrada:         dataEntrada,
		CnpjEmitente:        dto.CnpjEmitente,
		RazaoSocialEmitente: dto.RazaoSocialEmitente,
		IEEmitente:          dto.IEEmitente,
		UFEmitente:          dto.UFEmitente,
		Cfop:                dto.Cfop,
		ValorFrete:          dto.ValorFrete,
		ValorSeguro:         dto.ValorSeguro,
		ValorOutros:         dto.ValorOutros,
		ValorTotal:          dto.ValorTotal,
		ValorICMS:           dto.ValorICMS,
		BaseICMS:            dto.BaseICMS,
		AliqICMS:            dto.AliqICMS,
		CstICMS:             dto.CstICMS,
		TipoRateio:          dto.TipoRateio,
		FiscalEntryID:       dto.FiscalEntryID,
		Status:              "PENDENTE",
		EmissionData:        emissionData,
		Notes:               dto.Notes,
		IsActive:            true,
		CreatedBy:           userID,
	}

	return uc.Repo.CreateCTe(ctx, cte)
}

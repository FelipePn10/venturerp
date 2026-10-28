package entry_operation_uc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/repository"
	"github.com/FelipePn10/panossoerp/internal/shared/ptrutil"
)

type EntryOperationUseCase struct {
	repo repository.EntryOperationRepository
	// cfops confere a natureza contra a tabela oficial de CFOP (opcional).
	cfops CatalogoCFOP
}

// CatalogoCFOP: o cadastro de CFOP com a tabela oficial.
type CatalogoCFOP interface {
	GetCFOPByCode(ctx context.Context, code int32) (*fiscalentity.CFOP, error)
}

// ComCFOPs liga a conferência da natureza com a tabela oficial de CFOP.
func (uc *EntryOperationUseCase) ComCFOPs(c CatalogoCFOP) *EntryOperationUseCase {
	uc.cfops = c
	return uc
}

// conferirOperacao: natureza é CFOP de entrada da tabela oficial, vigente, e os
// créditos marcados cabem nela.
func (uc *EntryOperationUseCase) conferirOperacao(ctx context.Context, o *entity.EntryOperationType) error {
	if err := o.ConferirCreditos(); err != nil {
		return err
	}
	if uc.cfops == nil {
		return nil
	}
	n, _ := strconv.Atoi(o.NatureOperation)
	cf, err := uc.cfops.GetCFOPByCode(ctx, int32(n))
	if err != nil && !errorsuc.EhNaoEncontrado(err) {
		return err
	}
	if cf == nil {
		return errorsuc.NewValidationError(fmt.Sprintf("a natureza %s não existe na tabela de CFOP (VFIS0300)", o.NatureOperation))
	}
	if !cf.IsActive || !cf.VigenteEm(time.Now()) {
		return errorsuc.NewValidationError(fmt.Sprintf("o CFOP %s está inativo ou fora da vigência (VFIS0300)", o.NatureOperation))
	}
	return nil
}

func NewEntryOperationUseCase(repo repository.EntryOperationRepository) *EntryOperationUseCase {
	return &EntryOperationUseCase{repo: repo}
}

// ─── State Groups ─────────────────────────────────────────────────────────────

func (uc *EntryOperationUseCase) CreateStateGroup(ctx context.Context, dto request.CreateStateGroupDTO) (*response.StateGroupResponse, error) {
	code, err := uc.repo.NextStateGroupCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	g, err := entity.NewStateGroup(code, dto.Description, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateStateGroup(ctx, g)
	if err != nil {
		return nil, err
	}
	for _, uf := range dto.UFs {
		uf = strings.ToUpper(strings.TrimSpace(uf))
		if uf == "" {
			continue
		}
		if aerr := uc.repo.AddStateGroupUF(ctx, created.Code, uf); aerr != nil {
			return nil, aerr
		}
		created.UFs = append(created.UFs, uf)
	}
	return toStateGroupResponse(created), nil
}

func (uc *EntryOperationUseCase) GetStateGroup(ctx context.Context, code int64) (*response.StateGroupResponse, error) {
	g, err := uc.repo.GetStateGroupByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toStateGroupResponse(g), nil
}

func (uc *EntryOperationUseCase) ListStateGroups(ctx context.Context) ([]*response.StateGroupResponse, error) {
	list, err := uc.repo.ListStateGroups(ctx)
	if err != nil {
		return nil, err
	}
	return toStateGroupResponses(list), nil
}

func (uc *EntryOperationUseCase) AddStateGroupUF(ctx context.Context, dto request.AddStateGroupUFDTO) error {
	uf := strings.ToUpper(strings.TrimSpace(dto.UF))
	if uf == "" {
		return errorsuc.NewValidationError("informe a UF")
	}
	return uc.repo.AddStateGroupUF(ctx, dto.StateGroupCode, uf)
}

// ─── Entry Operation Types ────────────────────────────────────────────────────

func (uc *EntryOperationUseCase) CreateEntryOperation(ctx context.Context, dto request.CreateEntryOperationDTO) (*response.EntryOperationTypeResponse, error) {
	code, err := uc.repo.NextEntryOperationCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	o, err := entity.NewEntryOperationType(code, dto.Description, dto.NatureOperation, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	o.InvoiceTypeCode = dto.InvoiceTypeCode
	o.ClassificationType = dto.ClassificationType
	o.ClassificationCode = dto.ClassificationCode
	o.StateGroupCode = dto.StateGroupCode
	o.SupplierTypeCode = dto.SupplierTypeCode
	o.MovimentaEstoque = ptrutil.BoolOr(dto.MovimentaEstoque, true)
	o.GeraFinanceiro = ptrutil.BoolOr(dto.GeraFinanceiro, true)
	o.CreditaICMS = ptrutil.BoolOr(dto.CreditaICMS, true)
	o.CreditaIPI = ptrutil.BoolOr(dto.CreditaIPI, true)
	o.CreditaPISCOFINS = ptrutil.BoolOr(dto.CreditaPISCOFINS, true)
	if err := uc.conferirOperacao(ctx, o); err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateEntryOperation(ctx, o)
	if err != nil {
		return nil, err
	}
	return toEntryOperationTypeResponse(created), nil
}

func (uc *EntryOperationUseCase) UpdateEntryOperation(ctx context.Context, dto request.UpdateEntryOperationDTO) (*response.EntryOperationTypeResponse, error) {
	o, err := uc.repo.GetEntryOperationByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	o.Description = dto.Description
	if dto.NatureOperation != "" {
		n, err := entity.NormalizarNatureza(dto.NatureOperation)
		if err != nil {
			return nil, err
		}
		o.NatureOperation = n
	}
	o.InvoiceTypeCode = dto.InvoiceTypeCode
	o.ClassificationType = dto.ClassificationType
	o.ClassificationCode = dto.ClassificationCode
	o.StateGroupCode = dto.StateGroupCode
	o.SupplierTypeCode = dto.SupplierTypeCode
	o.IsActive = ptrutil.BoolOr(dto.IsActive, o.IsActive)
	o.MovimentaEstoque = ptrutil.BoolOr(dto.MovimentaEstoque, o.MovimentaEstoque)
	o.GeraFinanceiro = ptrutil.BoolOr(dto.GeraFinanceiro, o.GeraFinanceiro)
	o.CreditaICMS = ptrutil.BoolOr(dto.CreditaICMS, o.CreditaICMS)
	o.CreditaIPI = ptrutil.BoolOr(dto.CreditaIPI, o.CreditaIPI)
	o.CreditaPISCOFINS = ptrutil.BoolOr(dto.CreditaPISCOFINS, o.CreditaPISCOFINS)
	if err := uc.conferirOperacao(ctx, o); err != nil {
		return nil, err
	}
	updated, err := uc.repo.UpdateEntryOperation(ctx, o)
	if err != nil {
		return nil, err
	}
	return toEntryOperationTypeResponse(updated), nil
}

func (uc *EntryOperationUseCase) GetEntryOperation(ctx context.Context, code int64) (*response.EntryOperationTypeResponse, error) {
	o, err := uc.repo.GetEntryOperationByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toEntryOperationTypeResponse(o), nil
}

func (uc *EntryOperationUseCase) ListEntryOperations(ctx context.Context, onlyActive bool) ([]*response.EntryOperationTypeResponse, error) {
	list, err := uc.repo.ListEntryOperations(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toEntryOperationTypeResponses(list), nil
}

// ValidateUF applies the UF × Grupo de Estado rule for an entry operation type.
func (uc *EntryOperationUseCase) ValidateUF(ctx context.Context, code int64, enterpriseUF string) error {
	o, err := uc.repo.GetEntryOperationByCode(ctx, code)
	if err != nil {
		return err
	}
	enterpriseUF = strings.ToUpper(strings.TrimSpace(enterpriseUF))
	ufInGroup := false
	if o.StateGroupCode != nil {
		in, gerr := uc.repo.UFInGroup(ctx, *o.StateGroupCode, enterpriseUF)
		if gerr != nil {
			// Antes virava "UF fora do grupo" e a operação era recusada com a
			// mensagem errada.
			return fmt.Errorf("conferindo o grupo de estados da operação: %w", gerr)
		}
		ufInGroup = in
	}
	return o.ValidateUF(enterpriseUF, ufInGroup)
}

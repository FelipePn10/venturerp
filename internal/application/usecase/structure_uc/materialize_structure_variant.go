package structure_uc

//
//import (
//	"context"
//	"errors"
//	"fmt"
//
//	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
//
//	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
//
//	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
//	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
//	"github.com/FelipePn10/panossoerp/internal/application/ports"
//	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
//	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
//	"github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject"
//	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure"
//)
//
//type ResolveStructureForMaskUseCase struct {
//	repo repository.ItemStructureRepository
//	auth ports.AuthService
//}
//
//func (uc *ResolveStructureForMaskUseCase) Execute(
//	ctx context.Context,
//	dto request.ResolveStructureForMaskDTO,
//) (*response.StructureTreeResponse, error) {
//
//	if !uc.auth.ResolveStructureForMask(ctx) {
//		return nil, errorsuc.ErrUnauthorized
//	}
//
//	if dto.RootMaskValue == "" {
//		return nil, errors.New("root mask value is required")
//	}
//
//	rootExists, err := uc.repo.ItemExists(ctx, dto.RootItemCode)
//	if err != nil {
//		return nil, err
//	}
//	if !rootExists {
//		return nil, fmt.Errorf("item %d not found", dto.RootItemCode)
//	}
//
//	rootAnswers, err := uc.repo.GetMaskAnswersByItemAndValue(ctx, dto.RootItemCode, dto.RootMaskValue)
//	if err != nil {
//		return nil, err
//	}
//
//	nodes, err := uc.resolve(
//		ctx,
//		dto.RootItemCode,
//		dto.RootMaskValue,
//		rootAnswers,
//		1,
//		map[int64]bool{},
//	)
//	if err != nil {
//		return nil, err
//	}
//
//	respNodes := mapper.MapNodes(nodes)
//
//	return &response.StructureTreeResponse{
//		RootItemCode: dto.RootItemCode,
//		RootMask:     &dto.RootMaskValue,
//		Components:   respNodes,
//		TotalNodes:   mapper.CountNodes(respNodes),
//		TotalLevels:  mapper.MaxLevel(respNodes) + 1,
//	}, nil
//}

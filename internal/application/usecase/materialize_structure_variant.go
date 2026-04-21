package usecase

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
//
//func (uc *ResolveStructureForMaskUseCase) resolve(
//	ctx context.Context,
//	parentCode int64,
//	parentMask string,
//	parentAnswers []maskvo.MaskAnswer,
//	level int,
//	visited map[int64]bool,
//) ([]*valueobject.StructureNode, error) {
//
//	if visited[parentCode] {
//		return nil, nil
//	}
//
//	visited[parentCode] = true
//	defer delete(visited, parentCode)
//
//	children, err := uc.repo.GetDirectChildrenForMask(ctx, parentCode, parentMask)
//	if err != nil {
//		return nil, err
//	}
//
//	nodes := []*valueobject.StructureNode{}
//
//	for _, comp := range children {
//
//		code, desc, _ := uc.repo.GetItemCodeAndDesc(ctx, comp.ChildCode)
//
//		var childMask *string
//		var childAnswers []maskvo.MaskAnswer
//
//		if comp.Inherit {
//
//			questions, _ := uc.repo.GetItemQuestions(ctx, comp.ChildCode)
//
//			childMask = maskservice.PropagateMask(parentAnswers, questions)
//
//			if childMask != nil {
//				childAnswers, _ = uc.repo.GetMaskAnswersByItemAndValue(ctx, comp.ChildCode, *childMask)
//			}
//		}
//
//		node := valueobject.NewStructureNode(
//			comp,
//			code,
//			desc,
//			level,
//			childMask,
//		)
//
//		var sub []*valueobject.StructureNode
//
//		if childMask != nil {
//			sub, err = uc.resolve(ctx, comp.ChildCode, *childMask, childAnswers, level+1, visited)
//		} else {
//			sub, err = uc.resolveGeneric(ctx, comp.ChildCode, level+1, visited)
//		}
//
//		if err != nil {
//			return nil, err
//		}
//
//		for _, s := range sub {
//			node.AddChild(s)
//		}
//
//		nodes = append(nodes, node)
//	}
//
//	return nodes, nil
//}
//
//func (uc *ResolveStructureForMaskUseCase) resolveGeneric(
//	ctx context.Context,
//	parentCode int64,
//	level int,
//	visited map[int64]bool,
//) ([]*valueobject.StructureNode, error) {
//
//	children, err := uc.repo.GetAllDirectChildren(ctx, parentCode)
//	if err != nil {
//		return nil, err
//	}
//
//	nodes := []*valueobject.StructureNode{}
//
//	for _, comp := range children {
//
//		code, desc, _ := uc.repo.GetItemCodeAndDesc(ctx, comp.ChildCode)
//
//		node := valueobject.NewStructureNode(comp, code, desc, level, nil)
//
//		sub, _ := uc.resolveGeneric(ctx, comp.ChildCode, level+1, visited)
//
//		for _, s := range sub {
//			node.AddChild(s)
//		}
//
//		nodes = append(nodes, node)
//	}
//
//	return nodes, nil
//}

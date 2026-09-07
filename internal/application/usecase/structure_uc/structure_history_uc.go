package structure_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// StructureHistoryEntry é uma alteração registrada da estrutura.
type StructureHistoryEntry struct {
	ID            int64     `json:"id"`
	ParentCode    string    `json:"parent_code"`
	ChildCode     string    `json:"child_code"`
	Action        string    `json:"action"`
	ChangedBy     *string   `json:"changed_by,omitempty"`
	ChangedByName string    `json:"changed_by_name,omitempty"`
	ChangedAt     time.Time `json:"changed_at"`
	// Changes lista apenas o que mudou, em português, já comparado.
	Changes []FieldChange `json:"changes"`
}

// FieldChange é um campo que mudou, com o antes e o depois.
type FieldChange struct {
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// StructureHistoryRepository lê a trilha de auditoria da estrutura.
type StructureHistoryRepository interface {
	ListByParent(ctx context.Context, parentCode int64, limit int) ([]StructureHistoryEntry, error)
}

// ListStructureHistoryUseCase responde "quem mudou o quê e quando".
//
// Uma estrutura errada gera ordem errada e compra errada. Nem FoccoERP nem SAP
// mostram essa trilha na própria tela de estrutura; sem ela, descobrir quem
// trocou a quantidade depende de olhar log de banco.
type ListStructureHistoryUseCase struct {
	Repo StructureHistoryRepository
	// Items é o repositório de itens; o código de negócio informado pela tela é
	// resolvido para o código interno como nas demais consultas de estrutura.
	Items any
}

func (uc *ListStructureHistoryUseCase) Execute(
	ctx context.Context,
	parent request.TextCode,
	limit int,
) ([]StructureHistoryEntry, error) {
	if uc.Repo == nil {
		return nil, errorsuc.NewValidationError("o histórico da estrutura não está disponível nesta instalação")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	code, err := resolveItemCode(ctx, uc.Items, parent)
	if err != nil {
		return nil, err
	}
	return uc.Repo.ListByParent(ctx, code, limit)
}

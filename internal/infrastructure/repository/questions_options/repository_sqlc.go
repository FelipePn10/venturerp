package questionsoptions

import (
	"context"
	"database/sql"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/questions_options/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryQuestionOptionsSQLC) Save(
	ctx context.Context,
	qstops *entity.QuestionsOptions,
) (*entity.QuestionsOptions, error) {
	dbQuestionOption, err := r.q.CreateQuestionOption(ctx, sqlc.CreateQuestionOptionParams{
		Value:      qstops.Value,
		CreatedBy:  pgutil.ToPgUUID(qstops.CreatedBy),
		QuestionID: qstops.QuestionId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	return &entity.QuestionsOptions{
		ID:         dbQuestionOption.ID,
		QuestionId: dbQuestionOption.QuestionID,
		CreatedBy:  pgutil.FromPgUUID(dbQuestionOption.CreatedBy),
		Value:      dbQuestionOption.Value,
	}, nil
}

func (r *repositoryQuestionOptionsSQLC) ExistsQuestionOptionByValue(
	ctx context.Context,
	value string,
	questionID int64,
) (bool, error) {

	params := sqlc.ExistsQuestionOptionByValueParams{
		Value:      value,
		QuestionID: questionID,
	}

	exists, err := r.q.ExistsQuestionOptionByValue(ctx, params)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *repositoryQuestionOptionsSQLC) Delete(
	ctx context.Context,
	questionid int64,
) error {
	return r.q.DeleteQuestionOption(ctx, questionid)
}

func (r *repositoryQuestionOptionsSQLC) ListByQuestionID(
	ctx context.Context,
	questionID int64,
) ([]entity.QuestionsOptions, error) {
	rows, err := r.q.ListOptionsByQuestionID(ctx, questionID)
	if err != nil {
		return nil, err
	}

	result := make([]entity.QuestionsOptions, 0, len(rows))
	for _, row := range rows {
		result = append(result, entity.QuestionsOptions{
			ID:         row.ID,
			QuestionId: row.QuestionID,
			CreatedBy:  pgutil.FromPgUUID(row.CreatedBy),
			Value:      row.Value,
		})
	}
	return result, nil
}

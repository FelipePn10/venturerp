package questions

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/product/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/questions/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryQuestionSQLC) Save(
	ctx context.Context,
	qst *entity.Question,
) (*entity.Question, error) {

	params := sqlc.CreateQuestionParams{
		Name:      qst.Name,
		Createdby: pgutil.ToPgUUID(qst.CreatedBy),
	}

	dbQuestion, err := r.q.CreateQuestion(ctx, params)
	if err != nil {
		return nil, err
	}

	return &entity.Question{
		Name:      dbQuestion.Name,
		CreatedBy: pgutil.FromPgUUID(dbQuestion.Createdby),
	}, nil
}

func (r *repositoryQuestionSQLC) Delete(
	ctx context.Context,
	id int64,
) error {
	return r.q.DeleteQuestion(ctx, id)
}

func (r *repositoryQuestionSQLC) FindQuestionByName(
	ctx context.Context,
	name string,
) (*entity.Question, error) {

	dbQuestions, err := r.q.FindQuestionByName(ctx, name)
	if err != nil {
		return nil, err
	}

	if len(dbQuestions) == 0 {
		return nil, repository.ErrNotFound
	}

	dbQuestion := dbQuestions[0]

	return &entity.Question{
		Name:      dbQuestion.Name,
		CreatedBy: pgutil.FromPgUUID(dbQuestion.Createdby),
	}, nil
}

func (r *repositoryQuestionSQLC) ExistsQuestionByName(
	ctx context.Context,
	name string,
) (bool, error) {

	exists, err := r.q.ExistsQuestionByName(ctx, name)
	if err != nil {
		return false, err
	}

	return exists, nil
}

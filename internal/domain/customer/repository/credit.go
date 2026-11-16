package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
)

// CustomerLifecycleRepository reúne o que o cadastro do cliente precisa além do
// CRUD: crédito, situação na Receita, uso do cliente pelo sistema e o painel.
// Fica separado de CustomerRepository para não obrigar quem só lê cadastro a
// implementar tudo isto.
type CustomerLifecycleRepository interface {
	GetCreditPolicy(ctx context.Context) (entity.CreditPolicy, error)
	SaveCreditPolicy(ctx context.Context, p entity.CreditPolicy, by uuid.UUID) error

	// GroupCustomerCodes devolve os clientes da empresa com a mesma raiz de
	// CNPJ (grupo econômico), incluindo o próprio; para CPF, só o próprio.
	GroupCustomerCodes(ctx context.Context, docType entity.DocumentType, document string) ([]int64, error)
	GetCreditHistory(ctx context.Context, customerCodes []int64, historyMonths int, today time.Time) (entity.CreditHistory, error)

	// ApplyCreditReview grava a trilha e o limite do cliente na mesma transação.
	ApplyCreditReview(ctx context.Context, r *entity.CreditReview) error
	ListCreditReviews(ctx context.Context, customerCode int64) ([]*entity.CreditReview, error)

	SetCustomerRegistration(ctx context.Context, code int64, r entity.RegistrationSnapshot) error
	// MarkRegistrationAttempt registra a tentativa (com ou sem resposta) para
	// a fila do agendador andar.
	MarkRegistrationAttempt(ctx context.Context, code int64) error
	// ListCustomersForRegistrationCheck vê todas as empresas: só o agendador usa.
	ListCustomersForRegistrationCheck(ctx context.Context, max int) ([]entity.RegistrationCheckTarget, error)

	// FindCustomerByDocumentDigits acha o cliente da empresa pelo CNPJ/CPF sem
	// pontuação.
	FindCustomerByDocumentDigits(ctx context.Context, digits string) (code int64, name string, found bool, err error)
	CustomerUsage(ctx context.Context, code int64) ([]entity.CustomerUsage, error)
	SetCustomerActive(ctx context.Context, code int64, active bool) error
	// DeleteCustomer confere o uso com check e exclui numa só transação.
	DeleteCustomer(ctx context.Context, code int64, check func([]entity.CustomerUsage) error) error
	// DeleteSupport exclui um cadastro de apoio que ninguém usa.
	DeleteSupport(ctx context.Context, resource string, code int64) error
	// SetSupportActive inativa ou reativa um cadastro de apoio que tem situação.
	SetSupportActive(ctx context.Context, resource string, code int64, active bool) error
	CustomerOverview(ctx context.Context, code int64, groupCodes []int64, today time.Time) (*entity.CustomerOverview, error)
}

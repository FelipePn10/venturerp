package user_uc

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	userentity "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordChangeForbidden = errors.New("você não tem permissão para esta operação de troca de senha")
	ErrPasswordChangeInvalid   = errors.New("o pedido de troca de senha é inválido ou expirou")
	ErrCurrentPasswordInvalid  = errors.New("a senha atual está incorreta")
	ErrWeakPassword            = errors.New("a nova senha precisa de ao menos 12 caracteres, com maiúscula, minúscula, número e caractere especial")
	ErrSamePassword            = errors.New("a nova senha precisa ser diferente da senha atual")
	ErrPasswordChangeActive    = errors.New("você já tem uma solicitação de troca de senha em andamento; aguarde a decisão do administrador ou conclua a troca aprovada")
	ErrPasswordChangeNotFound  = errors.New("solicitação de troca de senha não encontrada")
	ErrPasswordChangeOwn       = errors.New("você não pode aprovar a própria solicitação; peça a outro administrador")
	ErrPasswordChangeDecided   = errors.New("esta solicitação não está mais aguardando aprovação; atualize a lista")
	ErrPasswordChangeExpired   = errors.New("a aprovação expirou sem uso; faça uma nova solicitação")
	ErrPasswordChangeNotReady  = errors.New("não há troca de senha aprovada para concluir; a solicitação precisa ser aprovada por um administrador")
	ErrRejectionReason         = errors.New("informe o motivo da rejeição (de 3 a 500 caracteres)")
	ErrRequestReason           = errors.New("o motivo da solicitação pode ter no máximo 500 caracteres")
)

// ApprovalWindow é por quanto tempo a aprovação vale para concluir a troca.
const ApprovalWindow = 15 * time.Minute

type PasswordChangeAuth interface {
	UserID(context.Context) (uuid.UUID, error)
	EnterpriseID(context.Context) (int64, error)
	IsAdmin(context.Context) bool
}

type PasswordChangeUseCase struct {
	repo repository.UserRepository
	auth PasswordChangeAuth
	now  func() time.Time
}

func NewPasswordChangeUseCase(repo repository.UserRepository, auth PasswordChangeAuth) *PasswordChangeUseCase {
	return &PasswordChangeUseCase{repo: repo, auth: auth, now: time.Now}
}

func (uc *PasswordChangeUseCase) identity(ctx context.Context) (uuid.UUID, int64, error) {
	userID, err := uc.auth.UserID(ctx)
	if err != nil {
		return uuid.Nil, 0, ErrPasswordChangeForbidden
	}
	enterpriseID, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return uuid.Nil, 0, ErrPasswordChangeForbidden
	}
	return userID, enterpriseID, nil
}

func (uc *PasswordChangeUseCase) Request(ctx context.Context, reason string) (*userentity.PasswordChangeRequest, error) {
	userID, enterpriseID, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > 500 {
		return nil, ErrRequestReason
	}
	request, err := uc.repo.CreatePasswordChangeRequest(ctx, userID, enterpriseID, reason)
	switch {
	case errors.Is(err, userentity.ErrPasswordChangeActive):
		return nil, ErrPasswordChangeActive
	case errors.Is(err, userentity.ErrPasswordChangeNotMember):
		return nil, ErrPasswordChangeForbidden
	case err != nil:
		return nil, err
	}
	request.Own = true
	return request, nil
}

// Mine: as solicitações de quem está logado, a mais recente primeiro.
func (uc *PasswordChangeUseCase) Mine(ctx context.Context) ([]userentity.PasswordChangeRequest, error) {
	userID, enterpriseID, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	requests, err := uc.repo.ListUserPasswordChangeRequests(ctx, userID, enterpriseID)
	if err != nil {
		return nil, err
	}
	for i := range requests {
		requests[i].Own = true
	}
	return requests, nil
}

func (uc *PasswordChangeUseCase) List(ctx context.Context, status string) ([]userentity.PasswordChangeRequest, error) {
	adminID, enterpriseID, err := uc.identity(ctx)
	if err != nil || !uc.auth.IsAdmin(ctx) {
		return nil, ErrPasswordChangeForbidden
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	switch status {
	case "", userentity.PasswordChangePending, userentity.PasswordChangeApproved, userentity.PasswordChangeRejected,
		userentity.PasswordChangeUsed, userentity.PasswordChangeExpired:
	default:
		return nil, ErrPasswordChangeInvalid
	}
	requests, err := uc.repo.ListPasswordChangeRequests(ctx, enterpriseID, status)
	if err != nil {
		return nil, err
	}
	for i := range requests {
		requests[i].Own = requests[i].UserID == adminID
	}
	return requests, nil
}

// pendingForDecision carrega a solicitação e diz por que ela não pode ser
// decidida, com a mensagem certa para cada caso.
func (uc *PasswordChangeUseCase) pendingForDecision(ctx context.Context, requestID uuid.UUID, enterpriseID int64) (*userentity.PasswordChangeRequest, error) {
	request, err := uc.repo.GetPasswordChangeRequest(ctx, requestID, enterpriseID)
	if errors.Is(err, userentity.ErrPasswordChangeNotFound) {
		return nil, ErrPasswordChangeNotFound
	}
	if err != nil {
		return nil, err
	}
	if request.Status != userentity.PasswordChangePending {
		return nil, ErrPasswordChangeDecided
	}
	return request, nil
}

func (uc *PasswordChangeUseCase) Approve(ctx context.Context, requestID uuid.UUID) error {
	adminID, enterpriseID, err := uc.identity(ctx)
	if err != nil || !uc.auth.IsAdmin(ctx) {
		return ErrPasswordChangeForbidden
	}
	request, err := uc.pendingForDecision(ctx, requestID, enterpriseID)
	if err != nil {
		return err
	}
	if request.UserID == adminID {
		return ErrPasswordChangeOwn
	}
	if err := uc.repo.ApprovePasswordChangeRequest(ctx, requestID, enterpriseID, adminID, uc.now().Add(ApprovalWindow)); err != nil {
		// Outra sessão decidiu entre a leitura e a gravação.
		return ErrPasswordChangeDecided
	}
	return nil
}

func (uc *PasswordChangeUseCase) Reject(ctx context.Context, requestID uuid.UUID, reason string) error {
	adminID, enterpriseID, err := uc.identity(ctx)
	if err != nil || !uc.auth.IsAdmin(ctx) {
		return ErrPasswordChangeForbidden
	}
	reason = strings.TrimSpace(reason)
	if n := utf8.RuneCountInString(reason); n < 3 || n > 500 {
		return ErrRejectionReason
	}
	if _, err := uc.pendingForDecision(ctx, requestID, enterpriseID); err != nil {
		return err
	}
	if err := uc.repo.RejectPasswordChangeRequest(ctx, requestID, enterpriseID, adminID, reason); err != nil {
		return ErrPasswordChangeDecided
	}
	return nil
}

var (
	upperPattern   = regexp.MustCompile(`[A-Z]`)
	lowerPattern   = regexp.MustCompile(`[a-z]`)
	digitPattern   = regexp.MustCompile(`[0-9]`)
	specialPattern = regexp.MustCompile(`[^A-Za-z0-9]`)
)

func validNewPassword(password string) bool {
	n := utf8.RuneCountInString(password)
	return n >= 12 && n <= 128 && upperPattern.MatchString(password) &&
		lowerPattern.MatchString(password) && digitPattern.MatchString(password) && specialPattern.MatchString(password)
}

func (uc *PasswordChangeUseCase) Complete(ctx context.Context, requestID uuid.UUID, currentPassword, newPassword string) error {
	userID, enterpriseID, err := uc.identity(ctx)
	if err != nil {
		return err
	}
	request, err := uc.repo.GetPasswordChangeRequest(ctx, requestID, enterpriseID)
	if errors.Is(err, userentity.ErrPasswordChangeNotFound) || (err == nil && request.UserID != userID) {
		return ErrPasswordChangeNotFound
	}
	if err != nil {
		return err
	}
	switch request.Status {
	case userentity.PasswordChangeApproved:
	case userentity.PasswordChangeExpired:
		return ErrPasswordChangeExpired
	default:
		return ErrPasswordChangeNotReady
	}
	if currentPassword == newPassword {
		return ErrSamePassword
	}
	if !validNewPassword(newPassword) {
		return ErrWeakPassword
	}
	currentHash, err := uc.repo.PasswordHash(ctx, userID, enterpriseID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(currentPassword)) != nil {
		return ErrCurrentPasswordInvalid
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	err = uc.repo.CompletePasswordChange(ctx, requestID, userID, enterpriseID, currentHash, string(newHash))
	switch {
	case errors.Is(err, userentity.ErrPasswordChangeExpired):
		return ErrPasswordChangeExpired
	case err != nil:
		return ErrPasswordChangeInvalid
	}
	return nil
}

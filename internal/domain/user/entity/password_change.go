package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Situações da solicitação de troca de senha.
const (
	PasswordChangePending  = "PENDING"
	PasswordChangeApproved = "APPROVED"
	PasswordChangeRejected = "REJECTED"
	PasswordChangeUsed     = "USED"
	PasswordChangeExpired  = "EXPIRED"
)

var (
	// ErrPasswordChangeActive: o usuário já tem uma solicitação aguardando
	// aprovação ou aprovada e ainda válida.
	ErrPasswordChangeActive = errors.New("já existe uma solicitação de troca de senha em andamento")
	// ErrPasswordChangeNotFound: a solicitação não existe nesta empresa.
	ErrPasswordChangeNotFound = errors.New("solicitação de troca de senha não encontrada")
	// ErrPasswordChangeExpired: a aprovação passou do prazo de uso.
	ErrPasswordChangeExpired = errors.New("a aprovação da troca de senha expirou")
	// ErrPasswordChangeNotMember: o usuário não pertence à empresa da sessão.
	ErrPasswordChangeNotMember = errors.New("usuário não vinculado a esta empresa")
)

// PasswordChangeRequest é a solicitação de troca de senha. Os identificadores
// internos (usuário, empresa) não saem na resposta: a tela mostra nome e
// e-mail; o ID da solicitação só serve para as ações.
type PasswordChangeRequest struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"-"`
	EnterpriseID    int64      `json:"-"`
	UserName        string     `json:"user_name,omitempty"`
	UserEmail       string     `json:"user_email,omitempty"`
	Status          string     `json:"status"`
	Reason          *string    `json:"reason,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	ApprovedByName  *string    `json:"approved_by_name,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	RejectedByName  *string    `json:"rejected_by_name,omitempty"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	UsedAt          *time.Time `json:"used_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	// Own: a solicitação é de quem está consultando (o administrador não aprova
	// a própria).
	Own bool `json:"own"`
}

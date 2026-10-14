package entity

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"

	"github.com/google/uuid"
)

var (
	ErrInvalidItemCode = domainerr.NewValidation("informe o código do item")
	ErrInvalidMask     = domainerr.NewValidation("informe a máscara")
	ErrInvalidMaskHash = domainerr.NewValidation("o identificador da máscara não pode ficar vazio")
	ErrInvalidHash     = domainerr.NewValidation("o identificador não corresponde à máscara informada")
	ErrInvalidUser     = domainerr.NewValidation("é preciso identificar o usuário que está criando o registro")
)

func NewItemMask(
	itemCode int64,
	mask string,
	maskHash string,
	createdBy uuid.UUID,
) (*ItemMask, error) {

	// Validação estrutural
	if itemCode < 0 {
		return nil, ErrInvalidItemCode
	}

	if mask == "" {
		return nil, ErrInvalidMask
	}

	if maskHash == "" {
		return nil, ErrInvalidMaskHash
	}

	if createdBy == uuid.Nil {
		return nil, ErrInvalidUser
	}

	expectedHash := generateHash(mask)
	if maskHash != expectedHash {
		return nil, ErrInvalidHash
	}

	return &ItemMask{
		ItemCode:  itemCode,
		Mask:      mask,
		MaskHash:  maskHash,
		CreatedBy: createdBy,
	}, nil
}

func generateHash(mask string) string {
	h := sha256.Sum256([]byte(mask))
	return hex.EncodeToString(h[:])[:8]
}

package entity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidItemCode = errors.New("informe o código do item")
	ErrInvalidMask     = errors.New("informe a máscara")
	ErrInvalidMaskHash = errors.New("o identificador da máscara não pode ficar vazio")
	ErrInvalidHash     = errors.New("o identificador não corresponde à máscara informada")
	ErrInvalidUser     = errors.New("é preciso identificar o usuário que está criando o registro")
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

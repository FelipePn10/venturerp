package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// UpdateBrandingUseCase stores the company logo and/or brand colour used to
// brand exported reports (PDF letterheads, table headers).
type UpdateBrandingUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

// Execute persists the provided branding. Empty logo or colour leaves the
// existing value untouched, so the logo and colour can be updated independently.
func (uc *UpdateBrandingUseCase) Execute(ctx context.Context, logo []byte, logoMime, brandColor string) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Repo.SetBranding(ctx, logo, logoMime, brandColor, userID)
}

// GetBrandingUseCase exposes the current logo bytes (for preview/download).
type GetBrandingUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

// Logo returns the stored logo bytes and its mime type. It returns (nil, "")
// when no logo is configured.
func (uc *GetBrandingUseCase) Logo(ctx context.Context) ([]byte, string, error) {
	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil || cfg == nil {
		return nil, "", err
	}
	mime := "image/png"
	if cfg.LogoMime != nil && *cfg.LogoMime != "" {
		mime = *cfg.LogoMime
	}
	return cfg.Logo, mime, nil
}

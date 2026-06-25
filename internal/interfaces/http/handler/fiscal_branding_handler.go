package handler

import (
	"bytes"
	"image/jpeg"
	"image/png"
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// FiscalBrandingHandler manages the company logo and brand colour that brand
// every exported report. Kept separate from the (large) FiscalHandler so the
// upload/serve plumbing stays self-contained.
type FiscalBrandingHandler struct {
	*security.BaseHandler
	update *fiscal_uc.UpdateBrandingUseCase
	get    *fiscal_uc.GetBrandingUseCase
}

func NewFiscalBrandingHandler(update *fiscal_uc.UpdateBrandingUseCase, get *fiscal_uc.GetBrandingUseCase) *FiscalBrandingHandler {
	return &FiscalBrandingHandler{BaseHandler: &security.BaseHandler{}, update: update, get: get}
}

const maxLogoBytes = 2 << 20 // 2 MiB

// Update handles POST /api/fiscal/config/branding as multipart/form-data with an
// optional `logo` file (PNG/JPEG) and an optional `brand_color` hex field.
func (h *FiscalBrandingHandler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxLogoBytes + (1 << 20)); err != nil {
		h.BadRequest(w, "envie multipart/form-data com 'logo' e/ou 'brand_color'")
		return
	}

	brandColor := strings.TrimSpace(r.FormValue("brand_color"))
	if brandColor != "" && !validHexColor(brandColor) {
		h.BadRequest(w, "brand_color deve ser hex no formato #RRGGBB")
		return
	}

	var logo []byte
	var logoMime string
	if file, _, err := r.FormFile("logo"); err == nil {
		defer file.Close()
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(http.MaxBytesReader(w, file, maxLogoBytes)); err != nil {
			h.BadRequest(w, "logo excede o tamanho máximo de 2 MB")
			return
		}
		logo = buf.Bytes()
		mime, ok := sniffImageMime(logo)
		if !ok {
			h.BadRequest(w, "logo deve ser PNG ou JPEG")
			return
		}
		logoMime = mime
	}

	if len(logo) == 0 && brandColor == "" {
		h.BadRequest(w, "informe 'logo' e/ou 'brand_color'")
		return
	}

	if err := h.update.Execute(r.Context(), logo, logoMime, brandColor); err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, map[string]any{"logo_updated": len(logo) > 0, "brand_color": brandColor}, "branding atualizado")
}

// Logo handles GET /api/fiscal/config/logo, serving the stored image so the
// front-end can preview it. Returns 404 when no logo is configured.
func (h *FiscalBrandingHandler) Logo(w http.ResponseWriter, r *http.Request) {
	data, mime, err := h.get.Logo(r.Context())
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	if len(data) == 0 {
		h.NotFound(w, "logo não configurado")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// sniffImageMime validates the bytes are a real PNG or JPEG and returns the mime.
func sniffImageMime(data []byte) (string, bool) {
	if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		if _, err := png.DecodeConfig(bytes.NewReader(data)); err == nil {
			return "image/png", true
		}
	}
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		if _, err := jpeg.DecodeConfig(bytes.NewReader(data)); err == nil {
			return "image/jpeg", true
		}
	}
	return "", false
}

func validHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for i := 1; i < 7; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

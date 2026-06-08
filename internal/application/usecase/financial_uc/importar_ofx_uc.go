package financial_uc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ImportarOFXUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

type ImportarOFXResult struct {
	Importados  int `json:"importados"`
	Duplicados  int `json:"duplicados"`
	Conciliados int `json:"conciliados"`
}

type ofxTransaction struct {
	TrnType  string
	DtPosted string
	TrnAmt   string
	FitID    string
	Memo     string
}

// Execute parses an OFX file content and imports transactions into extrato_bancario.
// Content may be OFX 1.x (SGML) or OFX 2.x (XML).
func (uc *ImportarOFXUseCase) Execute(ctx context.Context, contaBancariaID int64, ofxContent string) (*ImportarOFXResult, error) {
	if !uc.Auth.CanImportarOFX(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	txns, err := parseOFX(ofxContent)
	if err != nil {
		return nil, fmt.Errorf("parsing OFX: %w", err)
	}

	result := &ImportarOFXResult{}

	for _, t := range txns {
		data, err := parseOFXDate(t.DtPosted)
		if err != nil {
			continue
		}

		valor, err := strconv.ParseFloat(strings.TrimSpace(t.TrnAmt), 64)
		if err != nil {
			continue
		}

		tipo := "CREDIT"
		if valor < 0 {
			tipo = "DEBIT"
			valor = -valor
		}

		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(
			fmt.Sprintf("%d|%s|%s|%s", contaBancariaID, t.DtPosted, t.TrnAmt, t.FitID),
		)))

		err = uc.Repo.SaveExtratoItem(ctx, contaBancariaID, data, valor, tipo, t.Memo, t.FitID, hash)
		if err != nil {
			// ON CONFLICT DO NOTHING means duplicate = no error, but count it
			result.Duplicados++
			continue
		}
		result.Importados++
	}

	// Auto-match after import
	matched, _ := uc.Repo.AutoMatchExtrato(ctx, contaBancariaID)
	result.Conciliados = matched

	return result, nil
}

// parseOFX handles both OFX 1.x SGML and OFX 2.x XML formats.
func parseOFX(content string) ([]ofxTransaction, error) {
	// Detect OFX 2.x (starts with <?xml or <OFX>)
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<OFX>") {
		return parseOFXXML(trimmed)
	}
	// OFX 1.x: strip SGML headers, then parse tag content
	return parseOFXSGML(content)
}

var tagRe = regexp.MustCompile(`(?i)<([A-Z0-9.]+)>([^<]*)`)

func parseOFXSGML(content string) ([]ofxTransaction, error) {
	// Find the OFX body after the header lines
	idx := strings.Index(strings.ToUpper(content), "<OFX>")
	if idx >= 0 {
		content = content[idx:]
	}

	// Collect all tags and their values
	tags := make(map[string]string)
	matches := tagRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		tags[strings.ToUpper(m[1])] = strings.TrimSpace(m[2])
	}

	// Find all STMTTRN blocks
	re := regexp.MustCompile(`(?is)<STMTTRN>(.*?)</STMTTRN>`)
	blocks := re.FindAllStringSubmatch(content, -1)

	var txns []ofxTransaction
	for _, b := range blocks {
		block := b[1]
		blockTags := make(map[string]string)
		for _, m := range tagRe.FindAllStringSubmatch(block, -1) {
			blockTags[strings.ToUpper(m[1])] = strings.TrimSpace(m[2])
		}
		txns = append(txns, ofxTransaction{
			TrnType:  blockTags["TRNTYPE"],
			DtPosted: blockTags["DTPOSTED"],
			TrnAmt:   blockTags["TRNAMT"],
			FitID:    blockTags["FITID"],
			Memo:     blockTags["MEMO"],
		})
	}
	return txns, nil
}

func parseOFXXML(content string) ([]ofxTransaction, error) {
	// Same approach as SGML — use regex on XML content
	return parseOFXSGML(content)
}

// parseOFXDate parses OFX date formats: YYYYMMDD, YYYYMMDDHHMMSS, YYYYMMDDHHMMSS.XXX[-TZ:Name]
func parseOFXDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	// Strip timezone suffix after [
	if idx := strings.Index(s, "["); idx > 0 {
		s = s[:idx]
	}
	// Try various formats
	formats := []string{"20060102150405.000", "20060102150405", "20060102"}
	for _, f := range formats {
		if len(s) >= len(f) {
			t, err := time.Parse(f, s[:len(f)])
			if err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse OFX date: %q", s)
}

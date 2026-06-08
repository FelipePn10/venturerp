package ibpt_uc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/ibpt/repository"
)

// IBPTUseCase imports the IBPT/SCI table and looks up approximate tax burden.
type IBPTUseCase struct {
	Repo repository.IBPTRepository
}

// ImportFromCSV parses the official IBPT TabelaIBPTax CSV (semicolon-delimited,
// comma decimals) for a UF and bulk-upserts it. The expected columns are:
// codigo;ex;tipo;descricao;nacionalfederal;importadosfederal;estadual;municipal;
// vigenciainicio;vigenciafim;chave;versao;fonte
func (uc *IBPTUseCase) ImportFromCSV(ctx context.Context, uf, csvText string) (int, error) {
	uf = strings.ToUpper(strings.TrimSpace(uf))
	if len(uf) != 2 {
		return 0, fmt.Errorf("UF inválida: %q", uf)
	}

	lines := strings.Split(strings.ReplaceAll(csvText, "\r\n", "\n"), "\n")
	rates := make([]*entity.IBPTRate, 0, len(lines))
	for i, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		cols := strings.Split(ln, ";")
		// Skip a header row if present.
		if i == 0 && (strings.EqualFold(cols[0], "codigo") || strings.EqualFold(cols[0], "ncm")) {
			continue
		}
		if len(cols) < 8 {
			continue
		}
		r := &entity.IBPTRate{
			NCM:              strings.TrimSpace(cols[0]),
			Ex:               firstNonEmpty(strings.TrimSpace(cols[1]), "0"),
			UF:               uf,
			Tipo:             int16(parseIntSafe(cols[2])),
			Descricao:        strings.TrimSpace(cols[3]),
			NacionalFederal:  parseBRDecimal(cols[4]),
			ImportadoFederal: parseBRDecimal(cols[5]),
			Estadual:         parseBRDecimal(cols[6]),
			Municipal:        parseBRDecimal(cols[7]),
			Fonte:            "IBPT",
		}
		if len(cols) > 8 {
			r.VigenciaInicio = parseBRDate(cols[8])
		}
		if len(cols) > 9 {
			r.VigenciaFim = parseBRDate(cols[9])
		}
		if len(cols) > 10 {
			c := strings.TrimSpace(cols[10])
			if c != "" {
				r.Chave = &c
			}
		}
		if len(cols) > 11 {
			r.Versao = strings.TrimSpace(cols[11])
		}
		if len(cols) > 12 && strings.TrimSpace(cols[12]) != "" {
			r.Fonte = strings.TrimSpace(cols[12])
		}
		rates = append(rates, r)
	}

	return uc.Repo.BulkUpsert(ctx, rates)
}

// Lookup returns the most recent IBPT rate for an NCM in a UF.
func (uc *IBPTUseCase) Lookup(ctx context.Context, ncm, uf string) (*entity.IBPTRate, error) {
	return uc.Repo.GetByNCM(ctx, strings.TrimSpace(ncm), strings.ToUpper(strings.TrimSpace(uf)))
}

func firstNonEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func parseIntSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// parseBRDecimal parses "12,50" or "12.50" into 12.5.
func parseBRDecimal(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ".", ""))
	s = strings.ReplaceAll(s, ",", ".")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseBRDate parses DD/MM/YYYY; returns nil when empty/invalid.
func parseBRDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := time.Parse("02/01/2006", s); err == nil {
		return &t
	}
	return nil
}

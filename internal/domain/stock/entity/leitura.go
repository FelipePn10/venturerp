package entity

import (
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// O que um código lido pode ser.
const (
	LeituraItem     = "ITEM"
	LeituraEndereco = "ENDERECO"
)

// CodigoGS1 são os dados de uma etiqueta GS1-128 ou Data Matrix.
type CodigoGS1 struct {
	GTIN       string
	Lote       string
	Validade   *time.Time
	Quantidade *decimal.Decimal
}

// LeituraResolvida diz o que o coletor leu.
type LeituraResolvida struct {
	Tipo        string           `json:"kind"`
	ItemCode    *int64           `json:"item_code,omitempty"`
	Nome        string           `json:"name,omitempty"`
	Unidade     string           `json:"uom,omitempty"`
	Lote        string           `json:"lot,omitempty"`
	Validade    *time.Time       `json:"expires_at,omitempty"`
	Quantidade  *decimal.Decimal `json:"quantity,omitempty"`
	Endereco    string           `json:"address,omitempty"`
	WarehouseID *int64           `json:"warehouse_id,omitempty"`
	Bloqueado   bool             `json:"blocked,omitempty"`
}

const separadorGS1 = '\x1d'

// LerGS1 interpreta os campos GS1 que o estoque usa: 01 (GTIN), 10 (lote),
// 17 (validade), 30 e 37 (quantidade). Aceita a forma com parênteses e o prefixo
// de simbologia. Devolve false quando o texto não é GS1.
func LerGS1(texto string) (CodigoGS1, bool) {
	s := strings.TrimSpace(texto)
	for _, prefixo := range []string{"]C1", "]d2", "]Q3"} {
		s = strings.TrimPrefix(s, prefixo)
	}
	s = strings.NewReplacer("(", "", ")", "").Replace(s)
	if len(s) < 16 || !strings.HasPrefix(s, "01") || !soDigitos(s[2:16]) {
		return CodigoGS1{}, false
	}
	var g CodigoGS1
	for len(s) >= 2 {
		ai := s[:2]
		s = s[2:]
		switch ai {
		case "01", "17":
			tamanho := 14
			if ai == "17" {
				tamanho = 6
			}
			if len(s) < tamanho || !soDigitos(s[:tamanho]) {
				return CodigoGS1{}, false
			}
			if ai == "01" {
				g.GTIN = s[:tamanho]
			} else {
				g.Validade = validadeGS1(s[:tamanho])
			}
			s = s[tamanho:]
		case "10", "30", "37":
			fim := strings.IndexRune(s, separadorGS1)
			valor, resto := s, ""
			if fim >= 0 {
				valor, resto = s[:fim], s[fim+1:]
			}
			if ai == "10" {
				g.Lote = valor
			} else if q, err := decimal.NewFromString(valor); err == nil {
				g.Quantidade = &q
			}
			s = resto
		default:
			// Campo que o estoque não usa: o resto não dá para separar com segurança.
			return g, true
		}
		s = strings.TrimLeft(s, string(separadorGS1))
	}
	return g, true
}

func soDigitos(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// validadeGS1: AAMMDD; dia 00 é o último do mês, pela regra GS1.
func validadeGS1(s string) *time.Time {
	ano, _ := strconv.Atoi(s[:2])
	mes, _ := strconv.Atoi(s[2:4])
	dia, _ := strconv.Atoi(s[4:6])
	if mes < 1 || mes > 12 || dia > 31 {
		return nil
	}
	var t time.Time
	if dia == 0 {
		t = time.Date(2000+ano, time.Month(mes)+1, 0, 0, 0, 0, 0, time.UTC)
	} else {
		t = time.Date(2000+ano, time.Month(mes), dia, 0, 0, 0, 0, time.UTC)
	}
	return &t
}

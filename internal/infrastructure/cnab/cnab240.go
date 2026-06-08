// Package cnab generates bank exchange files. cnab240.go implements a CNAB 240
// (FEBRABAN) "remessa" for boleto registration.
//
// IMPORTANT: this follows the FEBRABAN 240 standard layout. Individual banks
// (Itaú, Bradesco, Santander, BB, Caixa) deviate in some positions — notably the
// "convênio"/carteira block of segment P and the "nosso número" composition.
// The output must be homologated with the target bank before production use.
package cnab

import (
	"fmt"
	"strings"
	"time"
)

// RemessaConfig holds the issuer (cedente/beneficiário) and bank data.
type RemessaConfig struct {
	BankCode    string // 3 digits, e.g. "341"
	BankName    string
	CompanyName string
	CompanyCNPJ string // digits only (14)
	Agencia     string
	AgenciaDV   string
	Conta       string
	ContaDV     string
	Convenio    string // código do convênio/cedente
	SequenceNSA int    // número sequencial do arquivo (NSA)

	// Bank-specific overrides. When left blank, they are resolved from the
	// BankProfile registry by BankCode (see bankProfile / resolveProfile).
	Carteira      string // 1 char, código da carteira de cobrança
	EspecieTitulo string // 2 chars, código da espécie (02 = DM, 01 = duplicata)
	LayoutArquivo string // 3 chars, versão do layout do arquivo
	LayoutLote    string // 3 chars, versão do layout do lote
}

// BankProfile carries the positions/codes that diverge between banks within the
// FEBRABAN 240 standard. The defaults below cover the five largest Brazilian
// banks; values still require homologation with the bank before production.
type BankProfile struct {
	Carteira      string
	EspecieTitulo string
	LayoutArquivo string
	LayoutLote    string
}

// bankProfiles maps a bank code to its known CNAB 240 deviations.
var bankProfiles = map[string]BankProfile{
	"341": {Carteira: "1", EspecieTitulo: "01", LayoutArquivo: "083", LayoutLote: "045"}, // Itaú
	"237": {Carteira: "9", EspecieTitulo: "02", LayoutArquivo: "084", LayoutLote: "045"}, // Bradesco
	"033": {Carteira: "1", EspecieTitulo: "02", LayoutArquivo: "040", LayoutLote: "030"}, // Santander
	"001": {Carteira: "1", EspecieTitulo: "02", LayoutArquivo: "083", LayoutLote: "042"}, // Banco do Brasil
	"104": {Carteira: "1", EspecieTitulo: "02", LayoutArquivo: "101", LayoutLote: "060"}, // Caixa (SIGCB)
}

// defaultProfile is used when the bank code is not in the registry.
var defaultProfile = BankProfile{Carteira: "1", EspecieTitulo: "02", LayoutArquivo: "103", LayoutLote: "030"}

// resolveProfile returns the effective profile for cfg, with explicit
// RemessaConfig overrides taking precedence over the registry defaults.
func resolveProfile(cfg RemessaConfig) BankProfile {
	p, ok := bankProfiles[onlyDigits(cfg.BankCode)]
	if !ok {
		p = defaultProfile
	}
	if cfg.Carteira != "" {
		p.Carteira = cfg.Carteira
	}
	if cfg.EspecieTitulo != "" {
		p.EspecieTitulo = cfg.EspecieTitulo
	}
	if cfg.LayoutArquivo != "" {
		p.LayoutArquivo = cfg.LayoutArquivo
	}
	if cfg.LayoutLote != "" {
		p.LayoutLote = cfg.LayoutLote
	}
	return p
}

// Titulo is a single boleto to register.
type Titulo struct {
	NossoNumero  string
	NumeroDoc    string
	Vencimento   time.Time
	Valor        float64
	Emissao      time.Time
	SacadoNome   string
	SacadoTipo   int // 1=CPF, 2=CNPJ
	SacadoDoc    string
	SacadoEnd    string
	SacadoBairro string
	SacadoCidade string
	SacadoUF     string
	SacadoCEP    string // digits only (8)
}

// GenerateRemessa240 builds the full 240-column remessa text (CRLF-terminated
// lines) for the given titles.
func GenerateRemessa240(cfg RemessaConfig, titulos []Titulo) (string, error) {
	if len(titulos) == 0 {
		return "", fmt.Errorf("nenhum título para gerar remessa")
	}
	var b strings.Builder
	now := time.Now()
	profile := resolveProfile(cfg)

	writeLine(&b, headerArquivo(cfg, profile, now))
	writeLine(&b, headerLote(cfg, profile))

	seq := 1 // sequential within the lote (registro detalhe)
	totalValor := 0.0
	for _, t := range titulos {
		writeLine(&b, segmentoP(cfg, profile, t, seq))
		seq++
		writeLine(&b, segmentoQ(cfg, t, seq))
		seq++
		totalValor += t.Valor
	}

	// Lote records = 2 header/trailer + 2 per título.
	loteRecords := 2 + seq - 1
	writeLine(&b, trailerLote(cfg, loteRecords, len(titulos), totalValor))
	// File records = header arquivo + lote records + trailer lote + trailer arquivo.
	fileRecords := 1 + loteRecords + 1
	writeLine(&b, trailerArquivo(cfg, fileRecords))

	return b.String(), nil
}

func writeLine(b *strings.Builder, line string) {
	b.WriteString(padAlpha(line, 240))
	b.WriteString("\r\n")
}

// ── Records ────────────────────────────────────────────────────────────────

func headerArquivo(cfg RemessaConfig, profile BankProfile, now time.Time) string {
	var s strings.Builder
	s.WriteString(padNum(cfg.BankCode, 3))                       // 001-003 banco
	s.WriteString("0000")                                        // 004-007 lote = 0000
	s.WriteString("0")                                           // 008 registro = 0
	s.WriteString(strings.Repeat(" ", 9))                        // 009-017 CNAB brancos
	s.WriteString("2")                                           // 018 tipo inscrição empresa = 2 (CNPJ)
	s.WriteString(padNum(cfg.CompanyCNPJ, 14))                   // 019-032 CNPJ
	s.WriteString(padNum(cfg.Convenio, 20))                      // 033-052 convênio
	s.WriteString(padNum(cfg.Agencia, 5))                        // 053-057 agência
	s.WriteString(padAlpha(cfg.AgenciaDV, 1))                    // 058 DV agência
	s.WriteString(padNum(cfg.Conta, 12))                         // 059-070 conta
	s.WriteString(padAlpha(cfg.ContaDV, 1))                      // 071 DV conta
	s.WriteString(" ")                                           // 072 DV ag/conta
	s.WriteString(padAlpha(cfg.CompanyName, 30))                 // 073-102 nome empresa
	s.WriteString(padAlpha(cfg.BankName, 30))                    // 103-132 nome banco
	s.WriteString(strings.Repeat(" ", 10))                       // 133-142 CNAB
	s.WriteString("1")                                           // 143 código remessa = 1
	s.WriteString(now.Format("02012006"))                        // 144-151 data geração DDMMAAAA
	s.WriteString(now.Format("150405"))                          // 152-157 hora geração HHMMSS
	s.WriteString(padNum(fmt.Sprintf("%d", cfg.SequenceNSA), 6)) // 158-163 NSA
	s.WriteString(padNum(profile.LayoutArquivo, 3))              // 164-166 versão layout arquivo (por banco)
	s.WriteString(strings.Repeat("0", 5))                        // 167-171 densidade
	s.WriteString(strings.Repeat(" ", 69))                       // 172-240 reservado
	return s.String()
}

func headerLote(cfg RemessaConfig, profile BankProfile) string {
	var s strings.Builder
	s.WriteString(padNum(cfg.BankCode, 3))       // 001-003 banco
	s.WriteString("0001")                        // 004-007 lote
	s.WriteString("1")                           // 008 registro = 1
	s.WriteString("R")                           // 009 tipo operação = Remessa
	s.WriteString("01")                          // 010-011 tipo serviço = Cobrança
	s.WriteString("  ")                          // 012-013 CNAB
	s.WriteString(padNum(profile.LayoutLote, 3)) // 014-016 versão layout lote (por banco)
	s.WriteString(" ")                           // 017 CNAB
	s.WriteString("2")                           // 018 tipo inscrição = CNPJ
	s.WriteString(padNum(cfg.CompanyCNPJ, 15))   // 019-033 CNPJ (15)
	s.WriteString(padNum(cfg.Convenio, 20))      // 034-053 convênio
	s.WriteString(padNum(cfg.Agencia, 5))        // 054-058 agência
	s.WriteString(padAlpha(cfg.AgenciaDV, 1))    // 059 DV
	s.WriteString(padNum(cfg.Conta, 12))         // 060-071 conta
	s.WriteString(padAlpha(cfg.ContaDV, 1))      // 072 DV
	s.WriteString(" ")                           // 073 DV ag/conta
	s.WriteString(padAlpha(cfg.CompanyName, 30)) // 074-103 nome empresa
	s.WriteString(strings.Repeat(" ", 40))       // 104-143 mensagem 1
	s.WriteString(strings.Repeat(" ", 40))       // 144-183 mensagem 2
	s.WriteString(strings.Repeat("0", 8))        // 184-191 nº remessa
	s.WriteString(strings.Repeat("0", 8))        // 192-199 data gravação
	s.WriteString(strings.Repeat("0", 8))        // 200-207 data crédito
	s.WriteString(strings.Repeat(" ", 33))       // 208-240 CNAB
	return s.String()
}

func segmentoP(cfg RemessaConfig, profile BankProfile, t Titulo, seq int) string {
	var s strings.Builder
	s.WriteString(padNum(cfg.BankCode, 3))           // 001-003 banco
	s.WriteString("0001")                            // 004-007 lote
	s.WriteString("3")                               // 008 registro = 3
	s.WriteString(padNum(fmt.Sprintf("%d", seq), 5)) // 009-013 nº sequencial registro
	s.WriteString("P")                               // 014 segmento
	s.WriteString(" ")                               // 015 CNAB
	s.WriteString("01")                              // 016-017 código movimento = 01 (entrada)
	s.WriteString(padNum(cfg.Agencia, 5))            // 018-022 agência
	s.WriteString(padAlpha(cfg.AgenciaDV, 1))        // 023 DV
	s.WriteString(padNum(cfg.Conta, 12))             // 024-035 conta
	s.WriteString(padAlpha(cfg.ContaDV, 1))          // 036 DV
	s.WriteString(" ")                               // 037 DV ag/conta
	s.WriteString(padNum(t.NossoNumero, 20))         // 038-057 nosso número
	s.WriteString(padAlpha(profile.Carteira, 1))     // 058 carteira (por banco)
	s.WriteString("1")                               // 059 cadastramento
	s.WriteString("2")                               // 060 documento = escritural
	s.WriteString("2")                               // 061 emissão boleto
	s.WriteString("A")                               // 062 distribuição
	s.WriteString(padAlpha(t.NumeroDoc, 15))         // 063-077 número documento
	s.WriteString(dateOrZero(t.Vencimento))          // 078-085 vencimento DDMMAAAA
	s.WriteString(padNum(valorStr(t.Valor), 15))     // 086-100 valor título
	s.WriteString(strings.Repeat("0", 5))            // 101-105 agência cobradora
	s.WriteString(" ")                               // 106 DV
	s.WriteString(padNum(profile.EspecieTitulo, 2))  // 107-108 espécie título (por banco)
	s.WriteString("N")                               // 109 aceite
	s.WriteString(dateOrZero(t.Emissao))             // 110-117 data emissão
	s.WriteString("0")                               // 118 código juros
	s.WriteString(strings.Repeat("0", 8))            // 119-126 data juros
	s.WriteString(strings.Repeat("0", 15))           // 127-141 valor juros
	s.WriteString("0")                               // 142 código desconto
	s.WriteString(strings.Repeat("0", 8))            // 143-150 data desconto
	s.WriteString(strings.Repeat("0", 15))           // 151-165 valor desconto
	s.WriteString(strings.Repeat("0", 15))           // 166-180 valor IOF
	s.WriteString(strings.Repeat("0", 15))           // 181-195 valor abatimento
	s.WriteString(padAlpha(t.NumeroDoc, 25))         // 196-220 identificação no banco/uso empresa
	s.WriteString("3")                               // 221 código protesto = não protestar
	s.WriteString("00")                              // 222-223 dias protesto
	s.WriteString("1")                               // 224 código baixa
	s.WriteString("000")                             // 225-227 dias baixa
	s.WriteString("09")                              // 228-229 moeda = real
	s.WriteString(strings.Repeat("0", 10))           // 230-239 contrato
	s.WriteString(" ")                               // 240 CNAB
	return s.String()
}

func segmentoQ(cfg RemessaConfig, t Titulo, seq int) string {
	tipoInsc := "1"
	if t.SacadoTipo == 2 {
		tipoInsc = "2"
	}
	var s strings.Builder
	s.WriteString(padNum(cfg.BankCode, 3))           // 001-003 banco
	s.WriteString("0001")                            // 004-007 lote
	s.WriteString("3")                               // 008 registro = 3
	s.WriteString(padNum(fmt.Sprintf("%d", seq), 5)) // 009-013 nº sequencial
	s.WriteString("Q")                               // 014 segmento
	s.WriteString(" ")                               // 015 CNAB
	s.WriteString("01")                              // 016-017 código movimento
	s.WriteString(tipoInsc)                          // 018 tipo inscrição sacado
	s.WriteString(padNum(t.SacadoDoc, 15))           // 019-033 documento sacado
	s.WriteString(padAlpha(t.SacadoNome, 40))        // 034-073 nome sacado
	s.WriteString(padAlpha(t.SacadoEnd, 40))         // 074-113 endereço
	s.WriteString(padAlpha(t.SacadoBairro, 15))      // 114-128 bairro
	s.WriteString(padNum(t.SacadoCEP, 8))            // 129-136 CEP (8)
	s.WriteString(padAlpha(t.SacadoCidade, 15))      // 137-151 cidade
	s.WriteString(padAlpha(t.SacadoUF, 2))           // 152-153 UF
	s.WriteString("0")                               // 154 tipo inscrição sacador/avalista
	s.WriteString(strings.Repeat("0", 15))           // 155-169 documento sacador
	s.WriteString(strings.Repeat(" ", 40))           // 170-209 nome sacador
	s.WriteString(strings.Repeat("0", 3))            // 210-212 banco correspondente
	s.WriteString(strings.Repeat(" ", 20))           // 213-232 nosso número banco corresp.
	s.WriteString(strings.Repeat(" ", 8))            // 233-240 CNAB
	return s.String()
}

func trailerLote(cfg RemessaConfig, qtdRegistros, qtdTitulos int, totalValor float64) string {
	var s strings.Builder
	s.WriteString(padNum(cfg.BankCode, 3))                    // 001-003 banco
	s.WriteString("0001")                                     // lote
	s.WriteString("5")                                        // registro = 5
	s.WriteString(strings.Repeat(" ", 9))                     // CNAB
	s.WriteString(padNum(fmt.Sprintf("%d", qtdRegistros), 6)) // qtd registros lote
	s.WriteString(padNum(fmt.Sprintf("%d", qtdTitulos), 6))   // qtd títulos cobrança
	s.WriteString(padNum(valorStr(totalValor), 17))           // valor total títulos
	s.WriteString(strings.Repeat("0", 6+17+6+17))             // demais totais zerados
	s.WriteString(strings.Repeat(" ", 31))                    // referência débito / CNAB
	// Pad/truncate to 240 happens in writeLine.
	return s.String()
}

func trailerArquivo(cfg RemessaConfig, qtdRegistros int) string {
	var s strings.Builder
	s.WriteString(padNum(cfg.BankCode, 3))                    // 001-003 banco
	s.WriteString("9999")                                     // lote = 9999
	s.WriteString("9")                                        // registro = 9
	s.WriteString(strings.Repeat(" ", 9))                     // CNAB
	s.WriteString("000001")                                   // qtd lotes
	s.WriteString(padNum(fmt.Sprintf("%d", qtdRegistros), 6)) // qtd registros arquivo
	s.WriteString("000000")                                   // qtd contas concil.
	return s.String()
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// padNum right-aligns digits in a field of width n, zero-filled on the left.
func padNum(s string, n int) string {
	s = onlyDigits(s)
	if len(s) > n {
		return s[len(s)-n:]
	}
	return strings.Repeat("0", n-len(s)) + s
}

// padAlpha left-aligns text in a field of width n, space-filled, upper-cased.
func padAlpha(s string, n int) string {
	s = strings.ToUpper(s)
	if len(s) > n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

// valorStr renders a monetary value as cents with no separators.
func valorStr(v float64) string {
	cents := int64(v*100 + 0.5)
	return fmt.Sprintf("%d", cents)
}

func dateOrZero(t time.Time) string {
	if t.IsZero() {
		return "00000000"
	}
	return t.Format("02012006")
}

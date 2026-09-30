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

// ImportarOFXResult é o que a tela mostra depois da importação.
//
// `Ignorados` e `Avisos` existem porque antes o caso de uso descartava linha com
// data ou valor ilegível em silêncio: a pessoa via "importados: 12" num extrato de
// 15 lançamentos e não tinha como saber dos 3 que faltaram.
type ImportarOFXResult struct {
	Importados  int      `json:"importados"`
	Duplicados  int      `json:"duplicados"`
	Conciliados int      `json:"conciliados"`
	Ignorados   int      `json:"ignorados"`
	Avisos      []string `json:"avisos,omitempty"`
	// Identificação que veio no próprio arquivo, para a pessoa conferir na tela
	// que importou o extrato da conta certa.
	BancoDoArquivo string `json:"banco_do_arquivo,omitempty"`
	ContaDoArquivo string `json:"conta_do_arquivo,omitempty"`
	PeriodoInicio  string `json:"periodo_inicio,omitempty"`
	PeriodoFim     string `json:"periodo_fim,omitempty"`
}

type ofxTransaction struct {
	TrnType  string
	DtPosted string
	TrnAmt   string
	FitID    string
	Memo     string
}

// extratoOFX é o arquivo inteiro: os lançamentos e a identificação da conta.
type extratoOFX struct {
	Transacoes []ofxTransaction
	BankID     string
	AccountID  string
	DtStart    string
	DtEnd      string
}

// ErrArquivoNaoEhOFX é o que a tela recebe quando o arquivo escolhido não é um
// extrato OFX. Existe como erro de validação (422), não 500: é escolha errada da
// pessoa, não falha do sistema.
const maxOFXBytes = 8 << 20 // 8 MiB — extrato de um mês não passa disso

// Execute importa um extrato OFX (1.x SGML ou 2.x XML) para a conta bancária.
//
// A validação do formato é a primeira coisa: antes, `parseOFX` nunca devolvia
// erro e um arquivo qualquer — JSON, PDF, planilha — produzia zero lançamento e
// resposta de SUCESSO. Quem importava concluía que o extrato estava vazio.
func (uc *ImportarOFXUseCase) Execute(ctx context.Context, contaBancariaID int64, ofxContent string) (*ImportarOFXResult, error) {
	if !uc.Auth.CanImportarOFX(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(ofxContent) > maxOFXBytes {
		return nil, errorsuc.NewValidationError(
			"o arquivo tem mais de 8 MB; envie o extrato de um período menor")
	}

	extrato, err := parseOFX(ofxContent)
	if err != nil {
		return nil, err
	}

	result := &ImportarOFXResult{
		BancoDoArquivo: extrato.BankID,
		ContaDoArquivo: extrato.AccountID,
		PeriodoInicio:  dataOFXLegivel(extrato.DtStart),
		PeriodoFim:     dataOFXLegivel(extrato.DtEnd),
		Avisos:         []string{},
	}

	// A conta do arquivo contra a conta escolhida na tela. Importar o extrato de
	// uma conta em outra concilia pagamento com movimento que nunca existiu ali —
	// é o erro mais caro desta tela, e o arquivo carrega a informação necessária
	// para avisar. Aviso e não recusa: o cadastro pode ter a conta com dígito,
	// máscara ou agência junto, e recusar por divergência de formatação
	// impediria a importação legítima.
	if aviso := uc.conferirConta(ctx, contaBancariaID, extrato); aviso != "" {
		result.Avisos = append(result.Avisos, aviso)
	}

	for _, t := range extrato.Transacoes {
		data, err := parseOFXDate(t.DtPosted)
		if err != nil {
			result.Ignorados++
			result.Avisos = append(result.Avisos, fmt.Sprintf(
				"lançamento ignorado: data %q não é uma data OFX válida", t.DtPosted))
			continue
		}

		valor, err := strconv.ParseFloat(strings.TrimSpace(t.TrnAmt), 64)
		if err != nil {
			result.Ignorados++
			result.Avisos = append(result.Avisos, fmt.Sprintf(
				"lançamento de %s ignorado: valor %q não é um número",
				data.Format("02/01/2006"), t.TrnAmt))
			continue
		}
		if valor == 0 {
			result.Ignorados++
			result.Avisos = append(result.Avisos, fmt.Sprintf(
				"lançamento de %s ignorado: valor zero não movimenta a conta",
				data.Format("02/01/2006")))
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

		inserido, err := uc.Repo.SaveExtratoItem(ctx, contaBancariaID, data, valor, tipo, t.Memo, t.FitID, hash)
		if err != nil {
			// Falha de gravação era contada como "duplicado" e a importação seguia
			// como se tivesse dado certo. Um extrato gravado pela metade produz
			// conciliação errada; interromper é a resposta honesta.
			return nil, fmt.Errorf("gravando o lançamento de %s: %w", data.Format("02/01/2006"), err)
		}
		if inserido {
			result.Importados++
		} else {
			result.Duplicados++
		}
	}

	matched, err := uc.Repo.AutoMatchExtrato(ctx, contaBancariaID)
	if err != nil {
		// A importação já valeu; só a conciliação automática falhou. Dizer isso é
		// melhor que devolver "conciliados: 0" como se nada tivesse casado.
		result.Avisos = append(result.Avisos,
			"os lançamentos foram importados, mas a conciliação automática falhou: "+err.Error())
	}
	result.Conciliados = matched

	if len(result.Avisos) == 0 {
		result.Avisos = nil
	}
	return result, nil
}

// conferirConta compara banco e conta do arquivo com o cadastro escolhido.
func (uc *ImportarOFXUseCase) conferirConta(ctx context.Context, contaBancariaID int64, extrato *extratoOFX) string {
	if extrato.AccountID == "" && extrato.BankID == "" {
		return ""
	}
	contas, err := uc.Repo.ListContasBancarias(ctx)
	if err != nil {
		return ""
	}
	for _, c := range contas {
		if c.ID != contaBancariaID {
			continue
		}
		if extrato.AccountID != "" && !mesmoNumero(c.Conta, extrato.AccountID) {
			return fmt.Sprintf(
				"atenção: o arquivo é da conta %s (banco %s) e a conta selecionada é %s (banco %s) — confirme antes de conciliar",
				extrato.AccountID, extrato.BankID, c.Conta, c.Banco)
		}
		return ""
	}
	return ""
}

// mesmoNumero compara contas bancárias escritas de formas diferentes.
//
// Compara só os dígitos, e aceita que um número seja SUFIXO do outro: o cadastro
// pode guardar "12345-6", "0001/12345-6" ou "123456", e o arquivo do banco manda
// normalmente sem máscara. O sufixo cobre o caso da agência colada na frente.
//
// Isso alimenta um AVISO, não uma recusa — e a escolha entre errar para mais ou
// para menos importa: um aviso a mais é incômodo, um aviso que não aparece deixa
// conciliar o extrato de uma conta contra os pagamentos de outra. Por isso a
// comparação é tolerante. O piso de 4 dígitos evita que contas curtas casem com
// qualquer coisa por acidente.
func mesmoNumero(a, b string) bool {
	da, db := somenteDigitos(a), somenteDigitos(b)
	if da == "" || db == "" {
		return true // sem número para comparar, não há divergência a afirmar
	}
	if da == db {
		return true
	}
	menor, maior := da, db
	if len(menor) > len(maior) {
		menor, maior = maior, menor
	}
	if len(menor) < 4 {
		return false
	}
	return strings.HasSuffix(maior, menor)
}

func somenteDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var (
	tagRe      = regexp.MustCompile(`(?i)<([A-Z0-9.]+)>([^<]*)`)
	stmttrnRe  = regexp.MustCompile(`(?is)<STMTTRN>(.*?)</STMTTRN>`)
	temRaizOFX = regexp.MustCompile(`(?i)<OFX>`)
	temHeader  = regexp.MustCompile(`(?i)OFXHEADER\s*[:=]`)
)

// parseOFX lê OFX 1.x (SGML) e 2.x (XML) e RECUSA o que não for OFX.
//
// A recusa é o ponto: o formato não é adivinhado pela extensão do arquivo — que
// qualquer um renomeia — mas pela estrutura. Um OFX tem a raiz <OFX> ou o
// cabeçalho OFXHEADER; um JSON, um PDF ou uma planilha não têm nenhum dos dois.
func parseOFX(content string) (*extratoOFX, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, errorsuc.NewValidationError(
			"o arquivo está vazio — selecione o extrato OFX baixado no site do banco")
	}
	if !temRaizOFX.MatchString(trimmed) && !temHeader.MatchString(trimmed) {
		return nil, errorsuc.NewValidationError(
			"o arquivo não é um extrato OFX: não foi encontrado o cabeçalho OFXHEADER nem a marcação <OFX>. " +
				"Baixe o extrato no formato OFX (Money/Open Financial Exchange) no site do banco — " +
				"CSV, PDF, JSON e planilha não servem para a conciliação. " +
				"Recebido: " + descreverConteudo(trimmed))
	}

	extrato := lerTagsOFX(trimmed)
	if len(extrato.Transacoes) == 0 {
		return nil, errorsuc.NewValidationError(
			"o arquivo é um OFX válido, mas não contém nenhum lançamento (<STMTTRN>). " +
				"Verifique se o período escolhido no banco tem movimento.")
	}
	return extrato, nil
}

// descreverConteudo diz o que parece ter sido enviado, para a mensagem ajudar em
// vez de só recusar. Só olha o começo do arquivo: nada do conteúdo é ecoado.
func descreverConteudo(s string) string {
	switch {
	case strings.HasPrefix(s, "{"), strings.HasPrefix(s, "["):
		return "um arquivo JSON"
	case strings.HasPrefix(s, "%PDF"):
		return "um arquivo PDF"
	case strings.HasPrefix(s, "PK\x03\x04"):
		return "um arquivo ZIP, XLSX ou DOCX"
	case strings.HasPrefix(s, "<?xml"):
		return "um XML que não é OFX (nota fiscal, talvez)"
	case strings.Contains(s, ";") || strings.Contains(s, ","):
		return "um arquivo de texto separado por delimitador (CSV)"
	default:
		return "um arquivo de texto sem estrutura OFX"
	}
}

// lerTagsOFX extrai os lançamentos e a identificação da conta. SGML e XML são
// lidos do mesmo jeito: em OFX as duas variantes têm as mesmas tags, e o que muda
// é apenas o fechamento — que esta leitura não exige.
func lerTagsOFX(content string) *extratoOFX {
	if idx := strings.Index(strings.ToUpper(content), "<OFX>"); idx >= 0 {
		content = content[idx:]
	}

	// Cabeçalho do extrato: banco, conta e período. Lidos ANTES dos blocos de
	// lançamento para a identificação da conta não ser sobrescrita por tag
	// homônima dentro de um lançamento.
	cabecalho := strings.SplitN(content, "<STMTTRN>", 2)[0]
	tagsCabecalho := make(map[string]string)
	for _, m := range tagRe.FindAllStringSubmatch(cabecalho, -1) {
		chave := strings.ToUpper(m[1])
		if _, existe := tagsCabecalho[chave]; !existe {
			tagsCabecalho[chave] = strings.TrimSpace(m[2])
		}
	}

	extrato := &extratoOFX{
		BankID:    tagsCabecalho["BANKID"],
		AccountID: tagsCabecalho["ACCTID"],
		DtStart:   tagsCabecalho["DTSTART"],
		DtEnd:     tagsCabecalho["DTEND"],
	}

	for _, b := range stmttrnRe.FindAllStringSubmatch(content, -1) {
		blockTags := make(map[string]string)
		for _, m := range tagRe.FindAllStringSubmatch(b[1], -1) {
			blockTags[strings.ToUpper(m[1])] = strings.TrimSpace(m[2])
		}
		extrato.Transacoes = append(extrato.Transacoes, ofxTransaction{
			TrnType:  blockTags["TRNTYPE"],
			DtPosted: blockTags["DTPOSTED"],
			TrnAmt:   blockTags["TRNAMT"],
			FitID:    blockTags["FITID"],
			Memo:     blockTags["MEMO"],
		})
	}
	return extrato
}

// parseOFXDate lê os formatos de data do OFX: AAAAMMDD, AAAAMMDDHHMMSS e
// AAAAMMDDHHMMSS.XXX[-TZ:Nome].
func parseOFXDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "["); idx > 0 {
		s = s[:idx]
	}
	for _, f := range []string{"20060102150405.000", "20060102150405", "20060102"} {
		if len(s) >= len(f) {
			if t, err := time.Parse(f, s[:len(f)]); err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("data OFX não reconhecida: %q", s)
}

// dataOFXLegivel devolve a data do cabeçalho em DD/MM/AAAA, ou vazio.
func dataOFXLegivel(bruto string) string {
	t, err := parseOFXDate(bruto)
	if err != nil {
		return ""
	}
	return t.Format("02/01/2006")
}

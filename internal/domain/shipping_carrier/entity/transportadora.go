// Package entity é o cadastro de transportadora: o perfil de transporte do
// fornecedor, a frota, as regiões atendidas e as ocorrências de entrega.
//
// A transportadora continua sendo um fornecedor (o frete é pago pelo contas a
// pagar de sempre). O que existe aqui é o que faltava para operar frete:
// habilitação na ANTT, modal, tabela de frete, seguro, veículo/motorista e
// prazo por região.
package entity

import (
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Modal string

const (
	ModalRodoviario  Modal = "RODOVIARIO"
	ModalAereo       Modal = "AEREO"
	ModalMaritimo    Modal = "MARITIMO"
	ModalFerroviario Modal = "FERROVIARIO"
	ModalDutoviario  Modal = "DUTOVIARIO"
	ModalMultimodal  Modal = "MULTIMODAL"
)

var modais = map[Modal]string{
	ModalRodoviario:  "Rodoviário",
	ModalAereo:       "Aéreo",
	ModalMaritimo:    "Marítimo",
	ModalFerroviario: "Ferroviário",
	ModalDutoviario:  "Dutoviário",
	ModalMultimodal:  "Multimodal",
}

func (m Modal) Valido() bool { _, ok := modais[m]; return ok }
func (m Modal) Rotulo() string {
	if r, ok := modais[m]; ok {
		return r
	}
	return string(m)
}

// TipoTransportador são as categorias da ANTT: ETC (empresa), CTC (cooperativa)
// e TAC (autônomo). O tipo muda a obrigação fiscal do CT-e.
type TipoTransportador string

const (
	TipoETC TipoTransportador = "ETC"
	TipoCTC TipoTransportador = "CTC"
	TipoTAC TipoTransportador = "TAC"
)

var tipos = map[TipoTransportador]string{
	TipoETC: "ETC — Empresa de Transporte de Carga",
	TipoCTC: "CTC — Cooperativa de Transporte de Carga",
	TipoTAC: "TAC — Transportador Autônomo",
}

func (t TipoTransportador) Valido() bool { _, ok := tipos[t]; return ok }
func (t TipoTransportador) Rotulo() string {
	if r, ok := tipos[t]; ok {
		return r
	}
	return string(t)
}

var (
	somenteDigitos   = regexp.MustCompile(`^[0-9]+$`)
	placaAntiga      = regexp.MustCompile(`^[A-Z]{3}[0-9]{4}$`)
	placaMercosul    = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)
	cepFormatoValido = regexp.MustCompile(`^[0-9]{8}$`)
)

type Transportadora struct {
	ID                 int64
	EnterpriseID       int64
	SupplierCode       int64
	SupplierName       string // leitura, vem da junção com suppliers
	SupplierDocument   string // leitura
	ANTTRNTRC          *string
	ANTTExpiry         *time.Time
	ShipperType        *TipoTransportador
	Modal              Modal
	IssuesCTe          bool
	DefaultFreightType *string
	FreightMinValue    decimal.Decimal
	FreightKgRate      decimal.Decimal
	FreightPctValue    decimal.Decimal
	GrisPct            decimal.Decimal
	TollPer100Kg       decimal.Decimal
	InsuranceCompany   *string
	InsurancePolicy    *string
	InsuranceExpiry    *time.Time
	InsuranceCoverage  decimal.Decimal
	AverageLeadDays    *int16
	TrackingURL        *string
	ContactName        *string
	ContactPhone       *string
	ContactEmail       *string
	Notes              *string
	// CubageFactor é o kg por m³ do peso cubado; zero cobra só o peso real.
	CubageFactor          decimal.Decimal
	DispatchFee           decimal.Decimal
	AcceptsDangerousGoods bool
	IsActive              bool
	CreatedAt             time.Time
	UpdatedAt             time.Time

	Vehicles     []*Veiculo
	ServiceAreas []*RegiaoAtendida
}

func (t *Transportadora) Normalizar() {
	if strings.TrimSpace(string(t.Modal)) == "" {
		t.Modal = ModalRodoviario
	}
	t.Modal = Modal(strings.ToUpper(strings.TrimSpace(string(t.Modal))))
	if t.ShipperType != nil {
		v := TipoTransportador(strings.ToUpper(strings.TrimSpace(string(*t.ShipperType))))
		if v == "" {
			t.ShipperType = nil
		} else {
			t.ShipperType = &v
		}
	}
	if t.ANTTRNTRC != nil {
		// O usuário digita "123.456-78"; o que vale é o número.
		limpo := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, *t.ANTTRNTRC)
		if limpo == "" {
			t.ANTTRNTRC = nil
		} else {
			t.ANTTRNTRC = &limpo
		}
	}
	if t.DefaultFreightType != nil {
		v := strings.ToUpper(strings.TrimSpace(*t.DefaultFreightType))
		if v == "" {
			t.DefaultFreightType = nil
		} else {
			t.DefaultFreightType = &v
		}
	}
	t.ContactEmail = limparTexto(t.ContactEmail)
	t.ContactName = limparTexto(t.ContactName)
	t.ContactPhone = limparTexto(t.ContactPhone)
	t.InsuranceCompany = limparTexto(t.InsuranceCompany)
	t.InsurancePolicy = limparTexto(t.InsurancePolicy)
	t.TrackingURL = limparTexto(t.TrackingURL)
	t.Notes = limparTexto(t.Notes)
}

func limparTexto(v *string) *string {
	if v == nil {
		return nil
	}
	limpo := strings.TrimSpace(*v)
	if limpo == "" {
		return nil
	}
	return &limpo
}

func (t *Transportadora) Validar() error {
	if t.SupplierCode <= 0 {
		return domainerr.NewValidation("informe o fornecedor da transportadora")
	}
	if !t.Modal.Valido() {
		return domainerr.NewValidation(fmt.Sprintf("modal de transporte inválido: %s", t.Modal))
	}
	if t.ShipperType != nil && !t.ShipperType.Valido() {
		return domainerr.NewValidation(fmt.Sprintf("tipo de transportador inválido: %s", *t.ShipperType))
	}
	// RNTRC tem 8 dígitos. Com número errado o CT-e é rejeitado na SEFAZ, e o
	// erro aparece na expedição, não no cadastro.
	if t.ANTTRNTRC != nil {
		if !somenteDigitos.MatchString(*t.ANTTRNTRC) || len(*t.ANTTRNTRC) != 8 {
			return domainerr.NewValidation("o RNTRC deve ter 8 dígitos")
		}
	}
	// Modal rodoviário emitindo CT-e sem RNTRC não passa na SEFAZ.
	if t.Modal == ModalRodoviario && t.IssuesCTe && t.ANTTRNTRC == nil {
		return domainerr.NewValidation("transportadora rodoviária que emite CT-e precisa do RNTRC")
	}
	if t.DefaultFreightType != nil {
		switch *t.DefaultFreightType {
		case "CIF", "FOB", "TERCEIROS", "SEM_FRETE":
		default:
			return domainerr.NewValidation(fmt.Sprintf("tipo de frete padrão inválido: %s", *t.DefaultFreightType))
		}
	}
	for _, d := range []struct {
		nome  string
		valor decimal.Decimal
	}{
		{"valor mínimo de frete", t.FreightMinValue},
		{"valor por quilo", t.FreightKgRate},
		{"pedágio por 100 kg", t.TollPer100Kg},
		{"cobertura do seguro", t.InsuranceCoverage},
		{"fator de cubagem", t.CubageFactor},
		{"taxa de despacho", t.DispatchFee},
	} {
		if d.valor.IsNegative() {
			return domainerr.NewValidation(fmt.Sprintf("o %s não pode ser negativo", d.nome))
		}
	}
	cem := decimal.NewFromInt(100)
	if t.FreightPctValue.IsNegative() || t.FreightPctValue.GreaterThan(cem) {
		return domainerr.NewValidation("o percentual de frete sobre o valor deve estar entre 0 e 100")
	}
	if t.GrisPct.IsNegative() || t.GrisPct.GreaterThan(cem) {
		return domainerr.NewValidation("o percentual de GRIS deve estar entre 0 e 100")
	}
	if t.AverageLeadDays != nil && *t.AverageLeadDays < 0 {
		return domainerr.NewValidation("o prazo médio não pode ser negativo")
	}
	if t.InsurancePolicy != nil && t.InsuranceCompany == nil {
		return domainerr.NewValidation("informe a seguradora da apólice")
	}
	for _, v := range t.Vehicles {
		if err := v.Validar(); err != nil {
			return err
		}
	}
	for _, a := range t.ServiceAreas {
		if err := a.Validar(); err != nil {
			return err
		}
	}
	return nil
}

// Alertas são as pendências que não impedem gravar mas impedem transportar:
// habilitação ou seguro vencido, frota sem veículo ativo, tabela zerada. É o
// que a tela mostra em amarelo em vez de descobrir na hora de despachar.
func (t *Transportadora) Alertas(hoje time.Time) []string {
	var out []string
	hoje = hoje.Truncate(24 * time.Hour)
	if t.ANTTRNTRC == nil && t.Modal == ModalRodoviario {
		out = append(out, "transportadora rodoviária sem RNTRC informado")
	}
	out = append(out, vencimento("RNTRC", "vencido", t.ANTTExpiry, hoje)...)
	out = append(out, vencimento("seguro", "vencido", t.InsuranceExpiry, hoje)...)
	if t.InsuranceCompany == nil {
		out = append(out, "sem seguro de carga cadastrado")
	}
	if t.FreightMinValue.IsZero() && t.FreightKgRate.IsZero() && t.FreightPctValue.IsZero() {
		out = append(out, "tabela de frete não preenchida: o frete terá de ser digitado a cada pedido")
	}
	if t.Modal == ModalRodoviario {
		ativos := 0
		for _, v := range t.Vehicles {
			if v.IsActive {
				ativos++
			}
		}
		if ativos == 0 {
			out = append(out, "nenhum veículo ativo cadastrado")
		}
	}
	for _, v := range t.Vehicles {
		if !v.IsActive {
			continue
		}
		// O MDF-e exige UF da placa e tara do veículo de tração.
		if v.PlateState == nil || v.TareKg.IsZero() {
			out = append(out, fmt.Sprintf("veículo %s sem UF da placa ou tara: o MDF-e não pode ser emitido com ele", v.Plate))
		}
		if v.DriverName != nil {
			out = append(out, vencimento(fmt.Sprintf("CNH do motorista %s (%s)", *v.DriverName, v.Plate), "vencida", v.DriverCNHExpiry, hoje)...)
		}
	}
	if len(t.ServiceAreas) == 0 {
		out = append(out, "nenhuma região atendida cadastrada: o prazo de entrega não pode ser calculado")
	}
	return out
}

// AvisoDeVencimento é a antecedência com que o documento a vencer vira
// pendência: dá tempo de pedir a renovação antes de a carga parar.
const AvisoDeVencimento = 30

// vencimento: `vencido` vem no gênero do documento ("RNTRC vencido", "CNH vencida").
func vencimento(oque, vencido string, validade *time.Time, hoje time.Time) []string {
	if validade == nil {
		return nil
	}
	if validade.Before(hoje) {
		return []string{fmt.Sprintf("%s %s em %s", oque, vencido, validade.Format("02/01/2006"))}
	}
	if validade.Before(hoje.AddDate(0, 0, AvisoDeVencimento+1)) {
		return []string{fmt.Sprintf("%s vence em %s", oque, validade.Format("02/01/2006"))}
	}
	return nil
}

var (
	tiposRodado     = map[string]bool{"01": true, "02": true, "03": true, "04": true, "05": true, "06": true}
	tiposCarroceria = map[string]bool{"00": true, "01": true, "02": true, "03": true, "04": true, "05": true}
	categoriasCNH   = map[string]bool{"A": true, "B": true, "C": true, "D": true, "E": true, "AB": true, "AC": true, "AD": true, "AE": true}
)

// RodadoValido e CarroceriaValida conferem os códigos tpRod e tpCar do MDF-e.
func RodadoValido(c string) bool     { return tiposRodado[c] }
func CarroceriaValida(c string) bool { return tiposCarroceria[c] }

// RenavamValido confere o dígito verificador do RENAVAM (11 dígitos; o antigo,
// de 9, é completado com zeros à esquerda como o DETRAN faz).
func RenavamValido(r string) bool {
	if !somenteDigitos.MatchString(r) || (len(r) != 9 && len(r) != 11) {
		return false
	}
	if len(r) == 9 {
		r = "00" + r
	}
	pesos := []int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	soma := 0
	for i, p := range pesos {
		soma += int(r[i]-'0') * p
	}
	dv := (soma * 10) % 11
	if dv == 10 {
		dv = 0
	}
	return dv == int(r[10]-'0')
}

type Veiculo struct {
	ID             int64
	EnterpriseID   int64
	CarrierID      int64
	Plate          string
	Description    *string
	VehicleType    *string
	Axles          *int16
	CapacityKg     decimal.Decimal
	CapacityM3     decimal.Decimal
	ANTTOwner      *string
	DriverName     *string
	DriverDocument *string
	DriverLicense  *string
	VehicleTypeID  *int64
	PlateState     *string
	Renavam        *string
	TareKg         decimal.Decimal
	WheelType      *string // MDF-e tpRod
	BodyType       *string // MDF-e tpCar
	// OwnerType é o tpProp do MDF-e: 0 TAC agregado, 1 TAC independente, 2
	// outros; vazio = veículo da própria transportadora.
	OwnerType         *string
	DriverCNHCategory *string
	DriverCNHExpiry   *time.Time
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (v *Veiculo) Normalizar() {
	v.Plate = strings.ToUpper(strings.NewReplacer("-", "", " ", "", ".", "").Replace(strings.TrimSpace(v.Plate)))
	v.Description = limparTexto(v.Description)
	v.VehicleType = limparTexto(v.VehicleType)
	v.ANTTOwner = limparTexto(v.ANTTOwner)
	v.DriverName = limparTexto(v.DriverName)
	v.DriverDocument = limparTexto(v.DriverDocument)
	v.DriverLicense = limparTexto(v.DriverLicense)
	v.PlateState = maiusculo(v.PlateState)
	v.Renavam = soDigitos(v.Renavam)
	v.WheelType = limparTexto(v.WheelType)
	v.BodyType = limparTexto(v.BodyType)
	v.OwnerType = limparTexto(v.OwnerType)
	v.DriverCNHCategory = maiusculo(v.DriverCNHCategory)
	if v.ANTTOwner != nil {
		v.ANTTOwner = soDigitos(v.ANTTOwner)
	}
	if v.DriverDocument != nil {
		v.DriverDocument = soDigitos(v.DriverDocument)
	}
}

func maiusculo(v *string) *string {
	v = limparTexto(v)
	if v == nil {
		return nil
	}
	u := strings.ToUpper(*v)
	return &u
}

func soDigitos(v *string) *string { return limparCEP(v) }

func (v *Veiculo) Validar() error {
	if v.Plate == "" {
		return domainerr.NewValidation("informe a placa do veículo")
	}
	// Aceita o padrão antigo (AAA9999) e o Mercosul (AAA9A99): a frota real tem
	// os dois, e recusar um dos formatos é travar o cadastro sem motivo.
	if !placaAntiga.MatchString(v.Plate) && !placaMercosul.MatchString(v.Plate) {
		return domainerr.NewValidation(fmt.Sprintf("placa inválida: %s (use AAA9999 ou AAA9A99)", v.Plate))
	}
	if v.CapacityKg.IsNegative() || v.CapacityM3.IsNegative() {
		return domainerr.NewValidation("a capacidade do veículo não pode ser negativa")
	}
	if v.Axles != nil && (*v.Axles < 2 || *v.Axles > 12) {
		return domainerr.NewValidation("o número de eixos deve estar entre 2 e 12")
	}
	if v.TareKg.IsNegative() {
		return domainerr.NewValidation(fmt.Sprintf("a tara do veículo %s não pode ser negativa", v.Plate))
	}
	if v.PlateState != nil && !validation.UFValida(*v.PlateState) {
		return domainerr.NewValidation(fmt.Sprintf("UF da placa %s inválida: %s", v.Plate, *v.PlateState))
	}
	if v.Renavam != nil && !RenavamValido(*v.Renavam) {
		return domainerr.NewValidation(fmt.Sprintf("RENAVAM do veículo %s inválido (dígito verificador não confere)", v.Plate))
	}
	if v.WheelType != nil && !tiposRodado[*v.WheelType] {
		return domainerr.NewValidation(fmt.Sprintf("tipo de rodado inválido no veículo %s: %s", v.Plate, *v.WheelType))
	}
	if v.BodyType != nil && !tiposCarroceria[*v.BodyType] {
		return domainerr.NewValidation(fmt.Sprintf("tipo de carroceria inválido no veículo %s: %s", v.Plate, *v.BodyType))
	}
	if v.OwnerType != nil {
		switch *v.OwnerType {
		case "0", "1":
			// Veículo de TAC: o MDF-e leva o RNTRC do proprietário.
			if v.ANTTOwner == nil || len(*v.ANTTOwner) != 8 {
				return domainerr.NewValidation(fmt.Sprintf("veículo %s é de TAC: informe o RNTRC do proprietário (8 dígitos)", v.Plate))
			}
		case "2":
		default:
			return domainerr.NewValidation(fmt.Sprintf("tipo de proprietário inválido no veículo %s: %s", v.Plate, *v.OwnerType))
		}
	}
	if v.ANTTOwner != nil && len(*v.ANTTOwner) != 8 {
		return domainerr.NewValidation(fmt.Sprintf("o RNTRC do proprietário do veículo %s deve ter 8 dígitos", v.Plate))
	}
	if v.DriverDocument != nil && !validation.ValidateCPF(*v.DriverDocument) {
		return domainerr.NewValidation(fmt.Sprintf("CPF do motorista do veículo %s inválido", v.Plate))
	}
	if v.DriverCNHCategory != nil && !categoriasCNH[*v.DriverCNHCategory] {
		return domainerr.NewValidation(fmt.Sprintf("categoria de CNH inválida no veículo %s: %s", v.Plate, *v.DriverCNHCategory))
	}
	return nil
}

// Cabe responde se a carga caberia neste veículo. É a pergunta da expedição.
func (v *Veiculo) Cabe(pesoKg, volumeM3 decimal.Decimal) bool {
	if v.CapacityKg.IsPositive() && pesoKg.GreaterThan(v.CapacityKg) {
		return false
	}
	if v.CapacityM3.IsPositive() && volumeM3.GreaterThan(v.CapacityM3) {
		return false
	}
	return true
}

type RegiaoAtendida struct {
	ID             int64
	EnterpriseID   int64
	CarrierID      int64
	State          *string
	City           *string
	PostalCodeFrom *string
	PostalCodeTo   *string
	LeadDays       int16
	MinValue       decimal.Decimal
	KgRate         decimal.Decimal
	PctValue       decimal.Decimal
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (a *RegiaoAtendida) Normalizar() {
	if a.State != nil {
		uf := strings.ToUpper(strings.TrimSpace(*a.State))
		if uf == "" {
			a.State = nil
		} else {
			a.State = &uf
		}
	}
	a.City = limparTexto(a.City)
	a.PostalCodeFrom = limparCEP(a.PostalCodeFrom)
	a.PostalCodeTo = limparCEP(a.PostalCodeTo)
}

func limparCEP(v *string) *string {
	if v == nil {
		return nil
	}
	limpo := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, *v)
	if limpo == "" {
		return nil
	}
	return &limpo
}

func (a *RegiaoAtendida) Validar() error {
	if a.State == nil && a.PostalCodeFrom == nil {
		return domainerr.NewValidation("informe a UF ou a faixa de CEP da região atendida")
	}
	if a.State != nil && len(*a.State) != 2 {
		return domainerr.NewValidation(fmt.Sprintf("UF inválida: %s", *a.State))
	}
	for _, cep := range []*string{a.PostalCodeFrom, a.PostalCodeTo} {
		if cep != nil && !cepFormatoValido.MatchString(*cep) {
			return domainerr.NewValidation(fmt.Sprintf("CEP inválido: %s (use 8 dígitos)", *cep))
		}
	}
	if a.PostalCodeFrom != nil && a.PostalCodeTo != nil && *a.PostalCodeFrom > *a.PostalCodeTo {
		return domainerr.NewValidation("a faixa de CEP começa depois de terminar")
	}
	if a.LeadDays < 0 {
		return domainerr.NewValidation("o prazo da região não pode ser negativo")
	}
	if a.MinValue.IsNegative() || a.KgRate.IsNegative() {
		return domainerr.NewValidation("os valores da região não podem ser negativos")
	}
	if a.PctValue.IsNegative() || a.PctValue.GreaterThan(decimal.NewFromInt(100)) {
		return domainerr.NewValidation("o percentual da região deve estar entre 0 e 100")
	}
	return nil
}

// Atende diz se o destino cai nesta região. A faixa de CEP é a mais
// específica; depois a cidade (dentro da UF); por último a UF inteira.
func (a *RegiaoAtendida) Atende(uf, cidade, cep string) bool {
	if !a.IsActive {
		return false
	}
	cep = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cep)
	// Região com faixa de CEP é específica, mesmo que também traga a UF. Sem um
	// CEP de oito dígitos não há como confirmar que o destino cai na faixa, e
	// cair na comparação por estado fazia uma transportadora cadastrada para uma
	// faixa estreita ser cotada como se atendesse o estado inteiro.
	if a.PostalCodeFrom != nil {
		if len(cep) != 8 {
			return false
		}
		fim := *a.PostalCodeFrom
		if a.PostalCodeTo != nil {
			fim = *a.PostalCodeTo
		}
		return cep >= *a.PostalCodeFrom && cep <= fim
	}
	if a.State == nil || !strings.EqualFold(*a.State, uf) {
		return false
	}
	// Região de uma cidade não vale para o estado todo: antes a cidade era
	// ignorada e o preço e o prazo de "Curitiba" eram aplicados a todo o PR.
	if a.City != nil {
		return cidade != "" && semAcento(*a.City) == semAcento(cidade)
	}
	return true
}

// Especificidade ordena as regiões que atendem o mesmo destino: CEP, cidade, UF.
func (a *RegiaoAtendida) Especificidade() int {
	switch {
	case a.PostalCodeFrom != nil:
		return 3
	case a.City != nil:
		return 2
	default:
		return 1
	}
}

var acentos = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "ê", "e", "è", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n",
)

func semAcento(s string) string {
	return strings.Join(strings.Fields(acentos.Replace(strings.ToLower(s))), " ")
}

// SomarDiasUteis conta o prazo em dias úteis (segunda a sexta), como as
// transportadoras informam. Feriado não entra: cada praça tem os seus.
func SomarDiasUteis(base time.Time, dias int) time.Time {
	d := base
	for dias > 0 {
		d = d.AddDate(0, 0, 1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			dias--
		}
	}
	return d
}

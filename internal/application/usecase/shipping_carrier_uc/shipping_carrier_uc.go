// Package shipping_carrier_uc é o cadastro de transportadora e a cotação de frete.
package shipping_carrier_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/repository"
	"github.com/shopspring/decimal"
)

const formatoData = "2006-01-02"

type UseCase struct {
	Repo domrepo.Repository
}

func (uc *UseCase) Listar(ctx context.Context, f domrepo.Filtro) ([]response.TransportadoraResponse, error) {
	lista, err := uc.Repo.Listar(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]response.TransportadoraResponse, 0, len(lista))
	for _, t := range lista {
		out = append(out, *montar(t))
	}
	return out, nil
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*response.TransportadoraResponse, error) {
	t, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	out := montar(t)
	// Doze meses é a janela que o comercial usa para renegociar frete.
	desde := time.Now().AddDate(-1, 0, 0)
	if d, err := uc.Repo.Desempenho(ctx, id, desde); err == nil {
		out.Desempenho = &response.TransportadoraDesempenhoResponse{
			Ocorrencias: d.Ocorrencias,
			AtrasoMedio: d.AtrasoMedio,
			CustoTotal:  d.CustoTotal,
			Desde:       desde.Format(formatoData),
		}
		if d.UltimaData != nil {
			s := d.UltimaData.Format(formatoData)
			out.Desempenho.UltimaData = &s
		}
	}
	return out, nil
}

func (uc *UseCase) ObterPorFornecedor(ctx context.Context, supplierCode int64) (*response.TransportadoraResponse, error) {
	t, err := uc.Repo.ObterPorFornecedor(ctx, supplierCode)
	if err != nil {
		return nil, err
	}
	return montar(t), nil
}

func (uc *UseCase) Salvar(ctx context.Context, id int64, dto *request.SalvarTransportadoraDTO) (*response.TransportadoraResponse, error) {
	if dto == nil {
		return nil, errorsuc.NewValidationError("informe os dados da transportadora")
	}
	t := &entity.Transportadora{
		ID:                 id,
		SupplierCode:       dto.SupplierCode,
		ANTTRNTRC:          dto.ANTTRNTRC,
		ShipperType:        tipo(dto.ShipperType),
		Modal:              entity.Modal(dto.Modal),
		IssuesCTe:          boolOu(dto.IssuesCTe, true),
		DefaultFreightType: dto.DefaultFreightType,
		FreightMinValue:    num(dto.FreightMinValue),
		FreightKgRate:      num(dto.FreightKgRate),
		FreightPctValue:    num(dto.FreightPctValue),
		GrisPct:            num(dto.GrisPct),
		TollPer100Kg:       num(dto.TollPer100Kg),
		InsuranceCompany:   dto.InsuranceCompany,
		InsurancePolicy:    dto.InsurancePolicy,
		InsuranceCoverage:  num(dto.InsuranceCoverage),
		AverageLeadDays:    dto.AverageLeadDays,
		TrackingURL:        dto.TrackingURL,
		ContactName:        dto.ContactName,
		ContactPhone:       dto.ContactPhone,
		ContactEmail:       dto.ContactEmail,
		Notes:              dto.Notes,
		IsActive:           boolOu(dto.IsActive, true),
	}
	var err error
	if t.ANTTExpiry, err = data(dto.ANTTExpiry, "validade do RNTRC"); err != nil {
		return nil, err
	}
	if t.InsuranceExpiry, err = data(dto.InsuranceExpiry, "validade do seguro"); err != nil {
		return nil, err
	}
	for _, v := range dto.Vehicles {
		veiculo := &entity.Veiculo{
			Plate:          v.Plate,
			Description:    v.Description,
			VehicleType:    v.VehicleType,
			Axles:          v.Axles,
			CapacityKg:     num(v.CapacityKg),
			CapacityM3:     num(v.CapacityM3),
			ANTTOwner:      v.ANTTOwner,
			DriverName:     v.DriverName,
			DriverDocument: v.DriverDocument,
			DriverLicense:  v.DriverLicense,
			IsActive:       boolOu(v.IsActive, true),
		}
		veiculo.Normalizar()
		t.Vehicles = append(t.Vehicles, veiculo)
	}
	for _, a := range dto.ServiceAreas {
		regiao := &entity.RegiaoAtendida{
			State:          a.State,
			City:           a.City,
			PostalCodeFrom: a.PostalCodeFrom,
			PostalCodeTo:   a.PostalCodeTo,
			LeadDays:       int16Ou(a.LeadDays),
			MinValue:       num(a.MinValue),
			KgRate:         num(a.KgRate),
			PctValue:       num(a.PctValue),
			IsActive:       boolOu(a.IsActive, true),
		}
		regiao.Normalizar()
		t.ServiceAreas = append(t.ServiceAreas, regiao)
	}
	t.Normalizar()
	if err := t.Validar(); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	// Placa repetida na própria grade: o banco recusaria, mas o erro aqui diz
	// qual placa é.
	vistas := map[string]bool{}
	for _, v := range t.Vehicles {
		if vistas[v.Plate] {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a placa %s aparece duas vezes na frota", v.Plate))
		}
		vistas[v.Plate] = true
	}
	nome, _, ativo, err := uc.Repo.FornecedorExiste(ctx, t.SupplierCode)
	if err != nil {
		return nil, err
	}
	if nome == "" {
		return nil, errorsuc.NewValidationError("o fornecedor informado não existe nesta empresa")
	}
	if !ativo {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o fornecedor %s está inativo: reative o cadastro antes de usá-lo como transportadora", nome))
	}
	salva, err := uc.Repo.Salvar(ctx, t)
	if err != nil {
		return nil, err
	}
	return montar(salva), nil
}

func (uc *UseCase) DefinirSituacao(ctx context.Context, id int64, ativa bool) error {
	return uc.Repo.DefinirSituacao(ctx, id, ativa)
}

func (uc *UseCase) RegistrarOcorrencia(ctx context.Context, carrierID int64, dto *request.RegistrarOcorrenciaDTO) (*response.TransportadoraOcorrenciaResponse, error) {
	if dto == nil || strings.TrimSpace(dto.OccurrenceType) == "" {
		return nil, errorsuc.NewValidationError("informe o tipo da ocorrência")
	}
	quando := time.Now()
	if d, err := data(dto.OccurrenceDate, "data da ocorrência"); err != nil {
		return nil, err
	} else if d != nil {
		quando = *d
	}
	if quando.After(time.Now().AddDate(0, 0, 1)) {
		return nil, errorsuc.NewValidationError("a ocorrência não pode ser lançada no futuro")
	}
	o := &domrepo.Ocorrencia{
		CarrierID:      carrierID,
		OccurrenceDate: quando,
		OccurrenceType: strings.ToUpper(strings.TrimSpace(dto.OccurrenceType)),
		SalesOrderCode: dto.SalesOrderCode,
		DelayDays:      int16Ou(dto.DelayDays),
		CostImpact:     num(dto.CostImpact),
		Description:    dto.Description,
	}
	if o.DelayDays < 0 {
		return nil, errorsuc.NewValidationError("o atraso não pode ser negativo")
	}
	gravada, err := uc.Repo.RegistrarOcorrencia(ctx, o)
	if err != nil {
		return nil, err
	}
	return montarOcorrencia(gravada), nil
}

func (uc *UseCase) ListarOcorrencias(ctx context.Context, carrierID int64, limite int) ([]response.TransportadoraOcorrenciaResponse, error) {
	lista, err := uc.Repo.ListarOcorrencias(ctx, carrierID, limite)
	if err != nil {
		return nil, err
	}
	out := make([]response.TransportadoraOcorrenciaResponse, 0, len(lista))
	for _, o := range lista {
		out = append(out, *montarOcorrencia(o))
	}
	return out, nil
}

// Cotar compara o frete de todas as transportadoras ativas que atendem o
// destino. É a pergunta que a expedição faz: quem leva, por quanto e em quantos
// dias.
func (uc *UseCase) Cotar(ctx context.Context, carga entity.Carga) (*response.ComparativoFreteResponse, error) {
	if carga.PesoKg.IsNegative() || carga.ValorMercadoria.IsNegative() {
		return nil, errorsuc.NewValidationError("peso e valor da carga não podem ser negativos")
	}
	if strings.TrimSpace(carga.UF) == "" && strings.TrimSpace(carga.CEP) == "" {
		return nil, errorsuc.NewValidationError("informe a UF ou o CEP do destino")
	}
	lista, err := uc.Repo.Listar(ctx, domrepo.Filtro{SomenteAtivas: true})
	if err != nil {
		return nil, err
	}
	out := &response.ComparativoFreteResponse{
		Destino:    strings.TrimSpace(strings.ToUpper(carga.UF) + " " + carga.CEP),
		PesoKg:     carga.PesoKg,
		ValorCarga: carga.ValorMercadoria,
		Cotacoes:   []response.CotacaoFreteResponse{},
		NaoAtendem: []string{},
	}
	for _, t := range lista {
		cot, err := t.Cotar(carga)
		if err != nil {
			// Não atender o destino não é erro da requisição: é informação.
			out.NaoAtendem = append(out.NaoAtendem, fmt.Sprintf("%s: %s", t.SupplierName, err.Error()))
			continue
		}
		out.Cotacoes = append(out.Cotacoes, *montarCotacao(cot))
	}
	sort.SliceStable(out.Cotacoes, func(i, j int) bool {
		if out.Cotacoes[i].Total.Equal(out.Cotacoes[j].Total) {
			return out.Cotacoes[i].PrazoDias < out.Cotacoes[j].PrazoDias
		}
		return out.Cotacoes[i].Total.LessThan(out.Cotacoes[j].Total)
	})
	if len(out.Cotacoes) > 0 {
		barata := out.Cotacoes[0].CarrierID
		out.MaisBarata = &barata
		rapida := out.Cotacoes[0]
		for _, c := range out.Cotacoes[1:] {
			if c.PrazoDias < rapida.PrazoDias {
				rapida = c
			}
		}
		out.MaisRapida = &rapida.CarrierID
	}
	return out, nil
}

func montar(t *entity.Transportadora) *response.TransportadoraResponse {
	out := &response.TransportadoraResponse{
		ID:                 t.ID,
		SupplierCode:       t.SupplierCode,
		SupplierName:       t.SupplierName,
		SupplierDocument:   t.SupplierDocument,
		ANTTRNTRC:          t.ANTTRNTRC,
		ANTTExpiry:         dataTexto(t.ANTTExpiry),
		Modal:              string(t.Modal),
		ModalLabel:         t.Modal.Rotulo(),
		IssuesCTe:          t.IssuesCTe,
		DefaultFreightType: t.DefaultFreightType,
		FreightMinValue:    t.FreightMinValue,
		FreightKgRate:      t.FreightKgRate,
		FreightPctValue:    t.FreightPctValue,
		GrisPct:            t.GrisPct,
		TollPer100Kg:       t.TollPer100Kg,
		InsuranceCompany:   t.InsuranceCompany,
		InsurancePolicy:    t.InsurancePolicy,
		InsuranceExpiry:    dataTexto(t.InsuranceExpiry),
		InsuranceCoverage:  t.InsuranceCoverage,
		AverageLeadDays:    t.AverageLeadDays,
		TrackingURL:        t.TrackingURL,
		ContactName:        t.ContactName,
		ContactPhone:       t.ContactPhone,
		ContactEmail:       t.ContactEmail,
		Notes:              t.Notes,
		IsActive:           t.IsActive,
		Alertas:            t.Alertas(time.Now()),
		Vehicles:           make([]response.TransportadoraVeiculoResponse, 0, len(t.Vehicles)),
		ServiceAreas:       make([]response.TransportadoraRegiaoResponse, 0, len(t.ServiceAreas)),
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
	}
	// Lista vazia em vez de nulo: a tela faz `.map` no alerta sem conferir.
	if out.Alertas == nil {
		out.Alertas = []string{}
	}
	if t.ShipperType != nil {
		valor := string(*t.ShipperType)
		rotulo := t.ShipperType.Rotulo()
		out.ShipperType = &valor
		out.ShipperTypeLabel = &rotulo
	}
	for _, v := range t.Vehicles {
		out.Vehicles = append(out.Vehicles, response.TransportadoraVeiculoResponse{
			ID: v.ID, Plate: v.Plate, Description: v.Description, VehicleType: v.VehicleType,
			Axles: v.Axles, CapacityKg: v.CapacityKg, CapacityM3: v.CapacityM3, ANTTOwner: v.ANTTOwner,
			DriverName: v.DriverName, DriverDocument: v.DriverDocument, DriverLicense: v.DriverLicense,
			IsActive: v.IsActive,
		})
	}
	for _, a := range t.ServiceAreas {
		out.ServiceAreas = append(out.ServiceAreas, response.TransportadoraRegiaoResponse{
			ID: a.ID, State: a.State, City: a.City, PostalCodeFrom: a.PostalCodeFrom,
			PostalCodeTo: a.PostalCodeTo, LeadDays: a.LeadDays, MinValue: a.MinValue,
			KgRate: a.KgRate, PctValue: a.PctValue, IsActive: a.IsActive,
		})
	}
	return out
}

func montarOcorrencia(o *domrepo.Ocorrencia) *response.TransportadoraOcorrenciaResponse {
	return &response.TransportadoraOcorrenciaResponse{
		ID:             o.ID,
		CarrierID:      o.CarrierID,
		OccurrenceDate: o.OccurrenceDate.Format(formatoData),
		OccurrenceType: o.OccurrenceType,
		SalesOrderCode: o.SalesOrderCode,
		DelayDays:      o.DelayDays,
		CostImpact:     o.CostImpact,
		Description:    o.Description,
		CreatedAt:      o.CreatedAt,
	}
}

func montarCotacao(c *entity.Cotacao) *response.CotacaoFreteResponse {
	out := &response.CotacaoFreteResponse{
		CarrierID:       c.CarrierID,
		SupplierCode:    c.SupplierCode,
		SupplierName:    c.SupplierName,
		Modal:           string(c.Modal),
		ModalLabel:      c.Modal.Rotulo(),
		ValorPorPeso:    c.ValorPorPeso,
		ValorAdValorem:  c.ValorAdValorem,
		ValorGris:       c.ValorGris,
		ValorPedagio:    c.ValorPedagio,
		PisoAplicado:    c.PisoAplicado,
		Total:           c.Total,
		PrazoDias:       c.PrazoDias,
		PrevisaoEntrega: c.PrevisaoEntrega.Format(formatoData),
		Alertas:         c.Alertas,
	}
	if out.Alertas == nil {
		out.Alertas = []string{}
	}
	if c.Regiao != nil {
		out.RegiaoID = &c.Regiao.ID
		out.RegiaoDescricao = descreverRegiao(c.Regiao)
	} else {
		out.RegiaoDescricao = "tabela geral da transportadora"
	}
	return out
}

func descreverRegiao(a *entity.RegiaoAtendida) string {
	partes := []string{}
	if a.City != nil {
		partes = append(partes, *a.City)
	}
	if a.State != nil {
		partes = append(partes, *a.State)
	}
	if a.PostalCodeFrom != nil {
		faixa := *a.PostalCodeFrom
		if a.PostalCodeTo != nil {
			faixa += " a " + *a.PostalCodeTo
		}
		partes = append(partes, "CEP "+faixa)
	}
	if len(partes) == 0 {
		return "região sem identificação"
	}
	return strings.Join(partes, " · ")
}

func num(v *float64) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	return decimal.NewFromFloat(*v)
}

func boolOu(v *bool, padrao bool) bool {
	if v == nil {
		return padrao
	}
	return *v
}

func int16Ou(v *int16) int16 {
	if v == nil {
		return 0
	}
	return *v
}

func tipo(v *string) *entity.TipoTransportador {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	t := entity.TipoTransportador(strings.ToUpper(strings.TrimSpace(*v)))
	return &t
}

func data(v *string, campo string) (*time.Time, error) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil, nil
	}
	d, err := time.Parse(formatoData, strings.TrimSpace(*v))
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s inválida: use o formato AAAA-MM-DD", campo))
	}
	return &d, nil
}

func dataTexto(d *time.Time) *string {
	if d == nil {
		return nil
	}
	s := d.Format(formatoData)
	return &s
}

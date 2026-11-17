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
	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
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
	hoje, err := uc.Repo.HojeDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]response.TransportadoraResponse, 0, len(lista))
	for _, t := range lista {
		out = append(out, *montar(t, hoje))
	}
	return out, nil
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*response.TransportadoraResponse, error) {
	t, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	hoje, err := uc.Repo.HojeDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	out := montar(t, hoje)
	// Doze meses é a janela que o comercial usa para renegociar frete.
	desde := hoje.AddDate(-1, 0, 0)
	d, err := uc.Repo.Desempenho(ctx, id, desde)
	if err != nil {
		return nil, err
	}
	out.Desempenho = &response.TransportadoraDesempenhoResponse{
		Ocorrencias: d.Ocorrencias,
		Falhas:      d.Falhas,
		AtrasoMedio: d.AtrasoMedio,
		CustoTotal:  d.CustoTotal,
		Desde:       desde.Format(formatoData),
	}
	if d.UltimaData != nil {
		s := d.UltimaData.Format(formatoData)
		out.Desempenho.UltimaData = &s
	}
	return out, nil
}

func (uc *UseCase) ObterPorFornecedor(ctx context.Context, supplierCode int64) (*response.TransportadoraResponse, error) {
	t, err := uc.Repo.ObterPorFornecedor(ctx, supplierCode)
	if err != nil {
		return nil, err
	}
	hoje, err := uc.Repo.HojeDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	return montar(t, hoje), nil
}

func (uc *UseCase) Salvar(ctx context.Context, id int64, dto *request.SalvarTransportadoraDTO) (*response.TransportadoraResponse, error) {
	if dto == nil {
		return nil, errorsuc.NewValidationError("informe os dados da transportadora")
	}
	t := &entity.Transportadora{
		ID:                    id,
		SupplierCode:          dto.SupplierCode,
		ANTTRNTRC:             dto.ANTTRNTRC,
		ShipperType:           tipo(dto.ShipperType),
		Modal:                 entity.Modal(dto.Modal),
		IssuesCTe:             boolOu(dto.IssuesCTe, true),
		DefaultFreightType:    dto.DefaultFreightType,
		FreightMinValue:       num(dto.FreightMinValue),
		FreightKgRate:         num(dto.FreightKgRate),
		FreightPctValue:       num(dto.FreightPctValue),
		GrisPct:               num(dto.GrisPct),
		TollPer100Kg:          num(dto.TollPer100Kg),
		InsuranceCompany:      dto.InsuranceCompany,
		InsurancePolicy:       dto.InsurancePolicy,
		InsuranceCoverage:     num(dto.InsuranceCoverage),
		AverageLeadDays:       dto.AverageLeadDays,
		TrackingURL:           dto.TrackingURL,
		ContactName:           dto.ContactName,
		ContactPhone:          dto.ContactPhone,
		ContactEmail:          dto.ContactEmail,
		Notes:                 dto.Notes,
		CubageFactor:          num(dto.CubageFactor),
		DispatchFee:           num(dto.DispatchFee),
		AcceptsDangerousGoods: boolOu(dto.AcceptsDangerousGoods, false),
		IsActive:              boolOu(dto.IsActive, true),
	}
	var err error
	if t.ANTTExpiry, err = data(dto.ANTTExpiry, "validade do RNTRC"); err != nil {
		return nil, err
	}
	if t.InsuranceExpiry, err = data(dto.InsuranceExpiry, "validade do seguro"); err != nil {
		return nil, err
	}
	tiposVeiculo, err := uc.Repo.ListarTiposVeiculo(ctx)
	if err != nil {
		return nil, err
	}
	var atual *entity.Transportadora
	if id > 0 {
		if atual, err = uc.Repo.Obter(ctx, id); err != nil {
			return nil, err
		}
		// Desktop anterior a estes campos (ainda aceito pelo min_client) não os
		// envia. Sem o campo no corpo, vale o que está gravado — senão gravar
		// pela tela antiga zeraria cubagem, despacho e produto perigoso.
		if dto.CubageFactor == nil {
			t.CubageFactor = atual.CubageFactor
		}
		if dto.DispatchFee == nil {
			t.DispatchFee = atual.DispatchFee
		}
		if dto.AcceptsDangerousGoods == nil {
			t.AcceptsDangerousGoods = atual.AcceptsDangerousGoods
		}
	}
	for _, v := range dto.Vehicles {
		veiculo := &entity.Veiculo{
			Plate:             v.Plate,
			Description:       v.Description,
			VehicleType:       v.VehicleType,
			Axles:             v.Axles,
			CapacityKg:        num(v.CapacityKg),
			CapacityM3:        num(v.CapacityM3),
			ANTTOwner:         v.ANTTOwner,
			DriverName:        v.DriverName,
			DriverDocument:    v.DriverDocument,
			DriverLicense:     v.DriverLicense,
			VehicleTypeID:     v.VehicleTypeID,
			PlateState:        v.PlateState,
			Renavam:           v.Renavam,
			TareKg:            num(v.TareKg),
			WheelType:         v.WheelType,
			BodyType:          v.BodyType,
			OwnerType:         v.OwnerType,
			DriverCNHCategory: v.DriverCNHCategory,
			IsActive:          boolOu(v.IsActive, true),
		}
		if veiculo.DriverCNHExpiry, err = data(v.DriverCNHExpiry, "validade da CNH"); err != nil {
			return nil, err
		}
		// Veículo vindo sem `tare_kg` é de tela antiga (a nova sempre manda):
		// os dados do MDF-e e da CNH gravados para a mesma placa são mantidos.
		if v.TareKg == nil && atual != nil {
			herdarDadosNovos(veiculo, atual.Vehicles)
		}
		if err := aplicarTipo(veiculo, tiposVeiculo); err != nil {
			return nil, err
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
	if atual != nil {
		if err := uc.conferirAlteracao(ctx, atual, t); err != nil {
			return nil, err
		}
	}
	nome, _, ativo, err := uc.Repo.FornecedorExiste(ctx, t.SupplierCode)
	if err != nil {
		return nil, err
	}
	if nome == "" {
		return nil, errorsuc.NewValidationError("o fornecedor informado não existe nesta empresa")
	}
	if !ativo && t.IsActive {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o fornecedor %s está inativo: reative o cadastro antes de usá-lo como transportadora", nome))
	}
	salva, err := uc.Repo.Salvar(ctx, t)
	if err != nil {
		return nil, err
	}
	hoje, err := uc.Repo.HojeDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	return montar(salva, hoje), nil
}

// DefinirSituacao inativa ou reativa. Inativar com documento em aberto deixaria
// carga, romaneio, pedido ou aviso apontando para quem não pode mais ser usado
// (a NF-e da carga seria recusada na emissão), e cliente com ela como padrão
// passaria a faturar sem transportadora válida.
func (uc *UseCase) DefinirSituacao(ctx context.Context, id int64, ativa bool) error {
	if ativa {
		t, err := uc.Repo.Obter(ctx, id)
		if err != nil {
			return err
		}
		nome, _, fornecedorAtivo, err := uc.Repo.FornecedorExiste(ctx, t.SupplierCode)
		if err != nil {
			return err
		}
		if !fornecedorAtivo {
			return errorsuc.NewValidationError(fmt.Sprintf("o fornecedor %s está inativo: reative-o na VSUP0500 antes de reativar a transportadora", nome))
		}
	} else if err := uc.conferirInativacao(ctx, id); err != nil {
		return err
	}
	return uc.Repo.DefinirSituacao(ctx, id, ativa)
}

func (uc *UseCase) conferirInativacao(ctx context.Context, id int64) error {
	usos, err := uc.Repo.Uso(ctx, id)
	if err != nil {
		return err
	}
	if abertos := descreverUso(usos, true); abertos != "" {
		return errorsuc.NewConflictError("a transportadora não pode ser inativada: está em " + abertos +
			". Conclua ou troque a transportadora nesses documentos antes")
	}
	return nil
}

// conferirAlteracao protege o que já foi gravado em documento: o fornecedor da
// transportadora não muda depois de usada (mudaria quem transportou em todo o
// histórico), a inativação segue a mesma regra do botão Inativar e veículo em
// carga ou romaneio aberto não sai da frota.
func (uc *UseCase) conferirAlteracao(ctx context.Context, atual, novo *entity.Transportadora) error {
	if atual.SupplierCode != novo.SupplierCode {
		usos, err := uc.Repo.Uso(ctx, novo.ID)
		if err != nil {
			return err
		}
		if todos := descreverUso(usos, false); todos != "" {
			return errorsuc.NewConflictError("o fornecedor da transportadora não pode ser trocado: ela já está em " + todos +
				". Cadastre outra transportadora para o novo fornecedor")
		}
	}
	if atual.IsActive && !novo.IsActive {
		if err := uc.conferirInativacao(ctx, novo.ID); err != nil {
			return err
		}
	}
	placas, err := uc.Repo.PlacasEmUso(ctx, novo.ID)
	if err != nil {
		return err
	}
	for _, placa := range placas {
		ok := false
		for _, v := range novo.Vehicles {
			if v.Plate == placa && v.IsActive {
				ok = true
				break
			}
		}
		if !ok {
			return errorsuc.NewConflictError(fmt.Sprintf("o veículo %s está em carga ou romaneio ainda não expedido: não pode ser removido nem inativado", placa))
		}
	}
	return nil
}

// herdarDadosNovos copia, do veículo gravado com a mesma placa, os campos que a
// tela antiga não conhece.
func herdarDadosNovos(v *entity.Veiculo, gravados []*entity.Veiculo) {
	placa := strings.ToUpper(strings.NewReplacer("-", "", " ", "", ".", "").Replace(strings.TrimSpace(v.Plate)))
	for _, g := range gravados {
		if g.Plate != placa {
			continue
		}
		// Só o que veio ausente: campo presente no corpo vale como enviado.
		v.TareKg = g.TareKg
		v.VehicleTypeID = ouGravado(v.VehicleTypeID, g.VehicleTypeID)
		v.PlateState = ouGravado(v.PlateState, g.PlateState)
		v.Renavam = ouGravado(v.Renavam, g.Renavam)
		v.WheelType = ouGravado(v.WheelType, g.WheelType)
		v.BodyType = ouGravado(v.BodyType, g.BodyType)
		v.OwnerType = ouGravado(v.OwnerType, g.OwnerType)
		v.DriverCNHCategory = ouGravado(v.DriverCNHCategory, g.DriverCNHCategory)
		v.DriverCNHExpiry = ouGravado(v.DriverCNHExpiry, g.DriverCNHExpiry)
		return
	}
}

func ouGravado[T any](enviado, gravado *T) *T {
	if enviado != nil {
		return enviado
	}
	return gravado
}

// Uso devolve onde a transportadora está gravada e o que a tela pode oferecer.
func (uc *UseCase) Uso(ctx context.Context, id int64) (*response.TransportadoraUsoResponse, error) {
	if _, err := uc.Repo.Obter(ctx, id); err != nil {
		return nil, err
	}
	usos, err := uc.Repo.Uso(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &response.TransportadoraUsoResponse{Documentos: []response.TransportadoraUsoItem{}, PodeExcluir: len(usos) == 0, PodeInativar: descreverUso(usos, true) == ""}
	for _, u := range usos {
		out.Documentos = append(out.Documentos, response.TransportadoraUsoItem{Documento: u.Documento, Total: u.Total, Abertos: u.Abertos})
	}
	return out, nil
}

// Excluir só apaga transportadora que nunca foi usada: com histórico, o
// caminho é inativar.
func (uc *UseCase) Excluir(ctx context.Context, id int64) error {
	if _, err := uc.Repo.Obter(ctx, id); err != nil {
		return err
	}
	usos, err := uc.Repo.Uso(ctx, id)
	if err != nil {
		return err
	}
	if todos := descreverUso(usos, false); todos != "" {
		return errorsuc.NewConflictError("a transportadora não pode ser excluída: já está em " + todos + ". Inative-a em vez de excluir")
	}
	return uc.Repo.Excluir(ctx, id)
}

// descreverUso monta "2 romaneio(s), 1 cliente(s)…"; com somenteAbertos, conta
// só documento em aberto e cliente ativo.
func descreverUso(usos []domrepo.Uso, somenteAbertos bool) string {
	var partes []string
	for _, u := range usos {
		n := u.Total
		if somenteAbertos {
			n = u.Abertos
		}
		if n > 0 {
			partes = append(partes, fmt.Sprintf("%d %s", n, u.Documento))
		}
	}
	return strings.Join(partes, ", ")
}

// aplicarTipo confere o tipo de veículo e completa o que o veículo deixou em
// branco com o padrão do tipo (rodado, carroceria, eixos, capacidade).
func aplicarTipo(v *entity.Veiculo, tipos []*domrepo.TipoVeiculo) error {
	if v.VehicleTypeID == nil {
		return nil
	}
	for _, t := range tipos {
		if t.ID != *v.VehicleTypeID {
			continue
		}
		if !t.IsActive {
			return errorsuc.NewValidationError(fmt.Sprintf("o tipo de veículo %s está inativo", t.Code))
		}
		if v.VehicleType == nil || strings.TrimSpace(*v.VehicleType) == "" {
			v.VehicleType = &t.Code
		}
		if v.WheelType == nil {
			v.WheelType = t.WheelType
		}
		if v.BodyType == nil {
			v.BodyType = t.BodyType
		}
		if v.Axles == nil {
			v.Axles = t.Axles
		}
		if v.CapacityKg.IsZero() {
			v.CapacityKg = t.CapacityKg
		}
		if v.CapacityM3.IsZero() {
			v.CapacityM3 = t.CapacityM3
		}
		return nil
	}
	return errorsuc.NewValidationError(fmt.Sprintf("o tipo de veículo %d não existe nesta empresa", *v.VehicleTypeID))
}

func (uc *UseCase) RegistrarOcorrencia(ctx context.Context, carrierID int64, dto *request.RegistrarOcorrenciaDTO) (*response.TransportadoraOcorrenciaResponse, error) {
	if dto == nil || strings.TrimSpace(dto.OccurrenceType) == "" {
		return nil, errorsuc.NewValidationError("informe o tipo da ocorrência")
	}
	// Comparação por DIA de calendário da empresa: a data digitada chega como
	// meia-noite, e "hoje" é o do fuso da empresa, não o do servidor.
	hoje, err := uc.Repo.HojeDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	quando := hoje
	if d, err := data(dto.OccurrenceDate, "data da ocorrência"); err != nil {
		return nil, err
	} else if d != nil {
		quando = *d
	}
	if quando.After(hoje) {
		return nil, errorsuc.NewValidationError("a ocorrência não pode ser lançada no futuro")
	}
	o := &domrepo.Ocorrencia{
		CarrierID:      carrierID,
		OccurrenceDate: quando,
		OccurrenceType: strings.ToUpper(strings.TrimSpace(dto.OccurrenceType)),
		SalesOrderCode: dto.SalesOrderCode,
		ShipmentCode:   dto.ShipmentCode,
		DelayDays:      int16Ou(dto.DelayDays),
		CostImpact:     num(dto.CostImpact),
		Description:    dto.Description,
	}
	if o.DelayDays < 0 {
		return nil, errorsuc.NewValidationError("o atraso não pode ser negativo")
	}
	if o.CostImpact.IsNegative() {
		return nil, errorsuc.NewValidationError("o custo da ocorrência não pode ser negativo")
	}
	tipos, err := uc.Repo.ListarTiposOcorrencia(ctx)
	if err != nil {
		return nil, err
	}
	valido := false
	for _, t := range tipos {
		if t.Code == o.OccurrenceType {
			if !t.IsActive {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("o tipo de ocorrência %s está inativo", t.Code))
			}
			valido = true
		}
	}
	if !valido {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("tipo de ocorrência %s não cadastrado: cadastre-o em Cadastros de apoio", o.OccurrenceType))
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
	if carga.PesoKg.IsNegative() || carga.ValorMercadoria.IsNegative() || carga.VolumeM3.IsNegative() {
		return nil, errorsuc.NewValidationError("peso, volume e valor da carga não podem ser negativos")
	}
	if !carga.PesoKg.IsPositive() {
		return nil, errorsuc.NewValidationError("informe o peso da carga: sem ele o frete sai só pelo piso e a comparação engana")
	}
	if err := conferirDestino(&carga); err != nil {
		return nil, err
	}
	if carga.Base.IsZero() {
		// Prazo e pendências contados a partir de hoje no fuso da empresa.
		hoje, err := uc.Repo.HojeDaEmpresa(ctx)
		if err != nil {
			return nil, err
		}
		carga.Base = hoje
	}
	lista, err := uc.Repo.Listar(ctx, domrepo.Filtro{SomenteAtivas: true})
	if err != nil {
		return nil, err
	}
	out := &response.ComparativoFreteResponse{
		Destino:    strings.TrimSpace(strings.Join(strings.Fields(carga.Cidade+" "+carga.UF+" "+carga.CEP), " ")),
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
	// Quem pode levar vem primeiro; entre iguais, o menor total e depois o
	// menor prazo. A impedida continua visível (o preço serve de referência),
	// mas nunca é indicada como a mais barata ou a mais rápida.
	sort.SliceStable(out.Cotacoes, func(i, j int) bool {
		a, b := out.Cotacoes[i], out.Cotacoes[j]
		if (len(a.Impedimentos) == 0) != (len(b.Impedimentos) == 0) {
			return len(a.Impedimentos) == 0
		}
		if a.Total.Equal(b.Total) {
			return a.PrazoDias < b.PrazoDias
		}
		return a.Total.LessThan(b.Total)
	})
	var rapida *response.CotacaoFreteResponse
	for i := range out.Cotacoes {
		c := &out.Cotacoes[i]
		if len(c.Impedimentos) > 0 {
			continue
		}
		if out.MaisBarata == nil {
			id := c.CarrierID
			out.MaisBarata = &id
		}
		if rapida == nil || c.PrazoDias < rapida.PrazoDias {
			rapida = c
		}
	}
	if rapida != nil {
		id := rapida.CarrierID
		out.MaisRapida = &id
	}
	return out, nil
}

// conferirDestino deixa o destino coerente antes de cotar: o CEP diz a UF
// (destino só com CEP não achava as regiões por estado), CEP de outro estado é
// erro de digitação, e UF inexistente não pode virar "ninguém atende".
func conferirDestino(c *entity.Carga) error {
	c.UF = strings.ToUpper(strings.TrimSpace(c.UF))
	c.Cidade = strings.TrimSpace(c.Cidade)
	c.CEP = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, c.CEP)
	if c.UF == "" && c.CEP == "" {
		return errorsuc.NewValidationError("informe a UF ou o CEP do destino")
	}
	if c.CEP != "" {
		if len(c.CEP) != 8 {
			return errorsuc.NewValidationError("CEP do destino inválido: use 8 dígitos")
		}
		ufDoCEP := validation.UFDoCEP(c.CEP)
		if ufDoCEP == "" {
			return errorsuc.NewValidationError(fmt.Sprintf("o CEP %s não pertence a nenhum estado", c.CEP))
		}
		if c.UF != "" && c.UF != ufDoCEP {
			return errorsuc.NewValidationError(fmt.Sprintf("o CEP %s é de %s, não de %s: confira o destino", c.CEP, ufDoCEP, c.UF))
		}
		c.UF = ufDoCEP
	}
	if !validation.UFValida(c.UF) {
		return errorsuc.NewValidationError(fmt.Sprintf("UF do destino inválida: %s", c.UF))
	}
	return nil
}

func montar(t *entity.Transportadora, hoje time.Time) *response.TransportadoraResponse {
	out := &response.TransportadoraResponse{
		ID:                    t.ID,
		SupplierCode:          t.SupplierCode,
		SupplierName:          t.SupplierName,
		SupplierDocument:      t.SupplierDocument,
		ANTTRNTRC:             t.ANTTRNTRC,
		ANTTExpiry:            dataTexto(t.ANTTExpiry),
		Modal:                 string(t.Modal),
		ModalLabel:            t.Modal.Rotulo(),
		IssuesCTe:             t.IssuesCTe,
		DefaultFreightType:    t.DefaultFreightType,
		FreightMinValue:       t.FreightMinValue,
		FreightKgRate:         t.FreightKgRate,
		FreightPctValue:       t.FreightPctValue,
		GrisPct:               t.GrisPct,
		TollPer100Kg:          t.TollPer100Kg,
		InsuranceCompany:      t.InsuranceCompany,
		InsurancePolicy:       t.InsurancePolicy,
		InsuranceExpiry:       dataTexto(t.InsuranceExpiry),
		InsuranceCoverage:     t.InsuranceCoverage,
		AverageLeadDays:       t.AverageLeadDays,
		TrackingURL:           t.TrackingURL,
		ContactName:           t.ContactName,
		ContactPhone:          t.ContactPhone,
		ContactEmail:          t.ContactEmail,
		Notes:                 t.Notes,
		CubageFactor:          t.CubageFactor,
		DispatchFee:           t.DispatchFee,
		AcceptsDangerousGoods: t.AcceptsDangerousGoods,
		IsActive:              t.IsActive,
		Alertas:               t.Alertas(hoje),
		Vehicles:              make([]response.TransportadoraVeiculoResponse, 0, len(t.Vehicles)),
		ServiceAreas:          make([]response.TransportadoraRegiaoResponse, 0, len(t.ServiceAreas)),
		CreatedAt:             t.CreatedAt,
		UpdatedAt:             t.UpdatedAt,
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
			VehicleTypeID: v.VehicleTypeID, PlateState: v.PlateState, Renavam: v.Renavam, TareKg: v.TareKg,
			WheelType: v.WheelType, BodyType: v.BodyType, OwnerType: v.OwnerType,
			DriverCNHCategory: v.DriverCNHCategory, DriverCNHExpiry: dataTexto(v.DriverCNHExpiry),
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
		ShipmentCode:   o.ShipmentCode,
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
		PesoTaxado:      c.PesoTaxado,
		PesoCubado:      c.PesoCubado,
		ValorDespacho:   c.ValorDespacho,
		ValorPorPeso:    c.ValorPorPeso,
		ValorAdValorem:  c.ValorAdValorem,
		ValorGris:       c.ValorGris,
		ValorPedagio:    c.ValorPedagio,
		PisoAplicado:    c.PisoAplicado,
		Total:           c.Total,
		PrazoDias:       c.PrazoDias,
		PrevisaoEntrega: c.PrevisaoEntrega.Format(formatoData),
		Alertas:         c.Alertas,
		Impedimentos:    c.Impedimentos,
	}
	if out.Alertas == nil {
		out.Alertas = []string{}
	}
	if out.Impedimentos == nil {
		out.Impedimentos = []string{}
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

func num(v *decimal.Decimal) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	return *v
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

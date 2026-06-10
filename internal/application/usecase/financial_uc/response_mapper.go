package financial_uc

import (
	"encoding/json"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
)

func toCondicaoPagamentoResponse(c *entity.CondicaoPagamento) *response.CondicaoPagamentoResponse {
	if c == nil {
		return nil
	}
	return &response.CondicaoPagamentoResponse{
		ID:        c.ID,
		Nome:      c.Nome,
		Parcelas:  json.RawMessage(c.Parcelas),
		Ativo:     c.Ativo,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func toCondicaoPagamentoResponses(list []*entity.CondicaoPagamento) []*response.CondicaoPagamentoResponse {
	out := make([]*response.CondicaoPagamentoResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toCondicaoPagamentoResponse(c))
	}
	return out
}

func toPlanoContasResponse(p *entity.PlanoContas) *response.PlanoContasResponse {
	if p == nil {
		return nil
	}
	return &response.PlanoContasResponse{
		ID:         p.ID,
		Codigo:     p.Codigo,
		Descricao:  p.Descricao,
		Tipo:       p.Tipo,
		Natureza:   p.Natureza,
		ParentCode: p.ParentCode,
		Nivel:      p.Nivel,
		IsActive:   p.IsActive,
		CreatedAt:  p.CreatedAt,
	}
}

func toPlanoContasResponses(list []*entity.PlanoContas) []*response.PlanoContasResponse {
	out := make([]*response.PlanoContasResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toPlanoContasResponse(p))
	}
	return out
}

func toCentroCustoResponse(c *entity.CentroCusto) *response.CentroCustoResponse {
	if c == nil {
		return nil
	}
	return &response.CentroCustoResponse{
		ID:        c.ID,
		Codigo:    c.Codigo,
		Descricao: c.Descricao,
		Tipo:      c.Tipo,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
	}
}

func toCentroCustoResponses(list []*entity.CentroCusto) []*response.CentroCustoResponse {
	out := make([]*response.CentroCustoResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toCentroCustoResponse(c))
	}
	return out
}

func toContaBancariaResponse(c *entity.ContaBancaria) *response.ContaBancariaResponse {
	if c == nil {
		return nil
	}
	return &response.ContaBancariaResponse{
		ID:           c.ID,
		Banco:        c.Banco,
		Agencia:      c.Agencia,
		Conta:        c.Conta,
		Digito:       c.Digito,
		Descricao:    c.Descricao,
		Titular:      c.Titular,
		SaldoInicial: c.SaldoInicial,
		ChavePix:     c.ChavePix,
		TipoChavePix: c.TipoChavePix,
		IsActive:     c.IsActive,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		CreatedBy:    c.CreatedBy,
	}
}

func toContaBancariaResponses(list []*entity.ContaBancaria) []*response.ContaBancariaResponse {
	out := make([]*response.ContaBancariaResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toContaBancariaResponse(c))
	}
	return out
}

func toContaPagarResponse(c *entity.ContaPagar) *response.ContaPagarResponse {
	if c == nil {
		return nil
	}
	return &response.ContaPagarResponse{
		ID:                       c.ID,
		NumeroDocumento:          c.NumeroDocumento,
		TipoDocumento:            c.TipoDocumento,
		FornecedorID:             c.FornecedorID,
		FiscalEntryID:            c.FiscalEntryID,
		PurchaseOrderID:          c.PurchaseOrderID,
		DataLancamento:           c.DataLancamento,
		DataEmissao:              c.DataEmissao,
		DataVencimento:           c.DataVencimento,
		DataPagamento:            c.DataPagamento,
		ValorBruto:               c.ValorBruto,
		Desconto:                 c.Desconto,
		Juros:                    c.Juros,
		Multa:                    c.Multa,
		ValorPago:                c.ValorPago,
		ParcelaNumero:            c.ParcelaNumero,
		ParcelaTotal:             c.ParcelaTotal,
		ParcelaPaiID:             c.ParcelaPaiID,
		ContaBancariaID:          c.ContaBancariaID,
		FormaPagamento:           c.FormaPagamento,
		PlanoContasID:            c.PlanoContasID,
		CentroCustoID:            c.CentroCustoID,
		StatusAprovacao:          string(c.StatusAprovacao),
		AprovadoPor:              c.AprovadoPor,
		DataAprovacao:            c.DataAprovacao,
		MotivoRejeicao:           c.MotivoRejeicao,
		Status:                   string(c.Status),
		AdiantamentoID:           c.AdiantamentoID,
		ValorAdiantamentoAbatido: c.ValorAdiantamentoAbatido,
		ComprovantePath:          c.ComprovantePath,
		Observacao:               c.Observacao,
		IsActive:                 c.IsActive,
		CriadoPor:                c.CriadoPor,
		BaixadoPor:               c.BaixadoPor,
		CreatedAt:                c.CreatedAt,
		UpdatedAt:                c.UpdatedAt,
	}
}

func toContaReceberResponse(c *entity.ContaReceber) *response.ContaReceberResponse {
	if c == nil {
		return nil
	}
	return &response.ContaReceberResponse{
		ID:              c.ID,
		NumeroDocumento: c.NumeroDocumento,
		ClienteID:       c.ClienteID,
		FiscalExitID:    c.FiscalExitID,
		SalesOrderID:    c.SalesOrderID,
		DataLancamento:  c.DataLancamento,
		DataEmissao:     c.DataEmissao,
		DataVencimento:  c.DataVencimento,
		DataRecebimento: c.DataRecebimento,
		ValorBruto:      c.ValorBruto,
		Desconto:        c.Desconto,
		Juros:           c.Juros,
		Multa:           c.Multa,
		ValorRecebido:   c.ValorRecebido,
		ParcelaNumero:   c.ParcelaNumero,
		ParcelaTotal:    c.ParcelaTotal,
		ParcelaPaiID:    c.ParcelaPaiID,
		ContaBancariaID: c.ContaBancariaID,
		FormaPagamento:  c.FormaPagamento,
		NossoNumero:     c.NossoNumero,
		LinhaDigitavel:  c.LinhaDigitavel,
		CodigoBarras:    c.CodigoBarras,
		ChavePixGerada:  c.ChavePixGerada,
		PlanoContasID:   c.PlanoContasID,
		CentroCustoID:   c.CentroCustoID,
		Status:          string(c.Status),
		EmProtesto:      c.EmProtesto,
		IsActive:        c.IsActive,
		CriadoPor:       c.CriadoPor,
		BaixadoPor:      c.BaixadoPor,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

func toTaxAssessmentResponse(t *entity.TaxAssessment) *response.TaxAssessmentResponse {
	if t == nil {
		return nil
	}
	return &response.TaxAssessmentResponse{
		ID:             t.ID,
		Imposto:        t.Imposto,
		Competencia:    t.Competencia,
		Debitos:        t.Debitos,
		Creditos:       t.Creditos,
		SaldoDevedor:   t.SaldoDevedor,
		SaldoCredor:    t.SaldoCredor,
		Status:         string(t.Status),
		CpID:           t.CpID,
		DataVencimento: t.DataVencimento,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

func toTaxAssessmentResponses(list []*entity.TaxAssessment) []*response.TaxAssessmentResponse {
	out := make([]*response.TaxAssessmentResponse, 0, len(list))
	for _, t := range list {
		out = append(out, toTaxAssessmentResponse(t))
	}
	return out
}

func toFluxoCaixaResponses(list []*entity.FluxoCaixa) []*response.FluxoCaixaResponse {
	out := make([]*response.FluxoCaixaResponse, 0, len(list))
	for _, f := range list {
		out = append(out, &response.FluxoCaixaResponse{
			ID:                     f.ID,
			Data:                   f.Data,
			Tipo:                   string(f.Tipo),
			Valor:                  f.Valor,
			ContaBancariaID:        f.ContaBancariaID,
			ContaBancariaDestinoID: f.ContaBancariaDestinoID,
			ContasPagarID:          f.ContasPagarID,
			ContasReceberID:        f.ContasReceberID,
			Descricao:              f.Descricao,
			Conciliado:             f.Conciliado,
			ExtratoHash:            f.ExtratoHash,
			CreatedAt:              f.CreatedAt,
		})
	}
	return out
}

func toAdiantamentoResponse(a *entity.Adiantamento) *response.AdiantamentoResponse {
	if a == nil {
		return nil
	}
	return &response.AdiantamentoResponse{
		ID:               a.ID,
		Tipo:             string(a.Tipo),
		ParceiroID:       a.ParceiroID,
		ContaBancariaID:  a.ContaBancariaID,
		NumeroDocumento:  a.NumeroDocumento,
		DataAdiantamento: a.DataAdiantamento,
		ValorOriginal:    a.ValorOriginal,
		ValorUtilizado:   a.ValorUtilizado,
		Saldo:            a.Saldo(),
		Status:           string(a.Status),
		Descricao:        a.Descricao,
		IsActive:         a.IsActive,
		CreatedBy:        a.CreatedBy,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

func toAdiantamentoResponses(list []*entity.Adiantamento) []*response.AdiantamentoResponse {
	out := make([]*response.AdiantamentoResponse, 0, len(list))
	for _, a := range list {
		out = append(out, toAdiantamentoResponse(a))
	}
	return out
}

func toAdiantamentoAplicacaoResponse(a *entity.AdiantamentoAplicacao) *response.AdiantamentoAplicacaoResponse {
	if a == nil {
		return nil
	}
	return &response.AdiantamentoAplicacaoResponse{
		ID:             a.ID,
		AdiantamentoID: a.AdiantamentoID,
		ContaTipo:      a.ContaTipo,
		ContaID:        a.ContaID,
		ValorAplicado:  a.ValorAplicado,
		DataAplicacao:  a.DataAplicacao,
		CreatedBy:      a.CreatedBy,
		CreatedAt:      a.CreatedAt,
	}
}

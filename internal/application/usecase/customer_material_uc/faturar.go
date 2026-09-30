package customer_material_uc

// Faturamento do beneficiamento: cria a nota de saída que cobra o serviço e devolve
// o material do cliente, e baixa o saldo de terceiro que essa nota representa.
//
// A ORDEM importa e é deliberada:
//
//	1. monta e valida as linhas (puro, sem efeito)
//	2. cria a nota de saída — em RASCUNHO, ainda não transmitida
//	3. baixa o saldo no razão de terceiros, vinculado à nota
//	4. (passo separado) autoriza na SEFAZ, conferindo antes que a baixa existe
//
// Não há transação entre os dois módulos, então a ordem é a proteção. Uma falha
// entre 2 e 3 deixa uma nota em rascunho sem baixa: recuperável repetindo, porque
// a baixa é idempotente pela nota, e inofensiva porque nada foi transmitido. O
// inverso — nota autorizada na SEFAZ sem o saldo baixado — seria material do
// cliente que saiu fiscalmente e continua aparecendo como presente, e é justamente
// isso que a ordem torna impossível.

import (
	"context"
	"fmt"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EscritorDeNota é o mínimo que o faturamento precisa do módulo fiscal. A
// interface é estreita de propósito: o beneficiamento não depende do repositório
// fiscal inteiro, e assim o caso de uso é testável com um dublê pequeno.
type EscritorDeNota interface {
	GetNextNFNumber(ctx context.Context) (int64, error)
	CreateExit(ctx context.Context, e *fiscalentity.FiscalExit) (*fiscalentity.FiscalExit, error)
	CreateExitItem(ctx context.Context, item *fiscalentity.FiscalExitItem) (*fiscalentity.FiscalExitItem, error)
}

// ResolvedorDeDestinatario resolve os dados fiscais do cliente. Vem de fora porque
// a regra de qual endereço vale (entrega → cobrança → padrão → primeiro) mora no
// módulo fiscal; duplicá-la faria a nota de beneficiamento sair com endereço
// diferente da nota de venda do mesmo cliente.
type ResolvedorDeDestinatario interface {
	DadosFiscaisDoCliente(ctx context.Context, code int64) (*DadosDoDestinatario, error)
}

// DadosDoDestinatario é o que a NF-e exige do cliente.
//
// É um ALIAS do tipo do módulo fiscal, não uma cópia: os dois lados falam a mesma
// estrutura e o resolvedor de lá satisfaz a interface daqui sem adaptador. Uma cópia
// com os mesmos campos exigiria conversão manual, que é onde um campo novo some.
type DadosDoDestinatario = fiscal_uc.DadosFiscaisDoDestinatario

// PedidoDeFaturamento é o que a tela envia.
type PedidoDeFaturamento struct {
	RemittanceID int64
	Servico      ServicoFaturado
	Devolucoes   []DevolucaoDeMaterial
	// Destinatario é opcional: vazio, o cadastro do cliente da remessa é consultado.
	// A tela não precisa montar endereço fiscal, e não deve — isso é do cadastro.
	Destinatario *DadosDoDestinatario
	// Serie da nota. Vazio usa "1", a série da operação.
	Serie string
	// DataEmissao vazia usa hoje.
	DataEmissao *time.Time
}

// ResultadoDoFaturamento devolve o que a tela precisa mostrar.
type ResultadoDoFaturamento struct {
	FiscalExitID int64
	NumeroNF     int64
	Serie        string
	Nota         *NotaDeRetorno
	Movimentos   []*entity.Movimento
	// SaldoRestante é o que continua em poder da empresa depois desta nota.
	SaldoRestante decimal.Decimal
}

// Faturar cria a nota e baixa o saldo. A nota nasce em rascunho: a transmissão é
// um passo separado, feito pelo autorizador fiscal.
func (uc *UseCase) Faturar(ctx context.Context, pedido PedidoDeFaturamento, usuario string) (*ResultadoDoFaturamento, error) {
	if uc.Notas == nil {
		return nil, errorsuc.NewValidationError("emissão de nota não configurada neste ambiente")
	}
	if strings.TrimSpace(usuario) == "" {
		return nil, errorsuc.NewValidationError("sessão sem usuário identificado")
	}
	autor, err := uuid.Parse(strings.TrimSpace(usuario))
	if err != nil {
		return nil, errorsuc.NewValidationError("usuário da sessão inválido")
	}

	remessa, err := uc.Repo.BuscarPorID(ctx, pedido.RemittanceID)
	if err != nil {
		return nil, err
	}

	// Destinatário: o informado vence; sem ele, resolve do cadastro do cliente da
	// remessa. Sem nenhum dos dois, a nota não teria destinatário.
	destinatario := pedido.Destinatario
	if destinatario == nil {
		if uc.Clientes == nil {
			return nil, errorsuc.NewValidationError(
				"informe o destinatário da nota: a resolução pelo cadastro não está disponível neste ambiente")
		}
		resolvido, err := uc.Clientes.DadosFiscaisDoCliente(ctx, remessa.CustomerCode)
		if err != nil {
			return nil, err
		}
		destinatario = resolvido
	}

	// 1. Monta e valida. Nada foi gravado ainda, então uma recusa aqui não deixa
	//    rastro — é onde CFOP, CST, saldo e UF são conferidos.
	nota, err := MontarNotaDeRetorno(remessa, destinatario.UF, pedido.Servico, pedido.Devolucoes)
	if err != nil {
		return nil, err
	}

	numero, err := uc.Notas.GetNextNFNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("obter o número da nota: %w", err)
	}
	serie := strings.TrimSpace(pedido.Serie)
	if serie == "" {
		serie = "1"
	}
	emissao := uc.agora()
	if pedido.DataEmissao != nil {
		emissao = *pedido.DataEmissao
	}

	// 2. Cria a nota em rascunho.
	saida := &fiscalentity.FiscalExit{
		NumeroNF:         numero,
		Serie:            serie,
		DataEmissao:      emissao,
		Cfop:             CFOPServico,
		NaturezaOperacao: nota.NaturezaOperacao,
		ValorProdutos:    paraFloat(nota.ValorTotal),
		ValorPIS:         paraFloat(nota.ValorPIS),
		ValorCOFINS:      paraFloat(nota.ValorCOFINS),
		ValorTotal:       paraFloat(nota.ValorTotal),
		// Rascunho de propósito: a transmissão é passo separado, e é ela que confere
		// se o saldo já foi baixado.
		Status:         fiscalentity.ExitStatusDraft,
		IsActive:       true,
		CreatedBy:      autor,
		CustomerCode:   &remessa.CustomerCode,
		SalesOrderCode: remessa.SalesOrderCode,
		SourceType:     textoOuNil("BENEFICIAMENTO"),
	}
	aplicarDestinatario(saida, *destinatario)

	criada, err := uc.Notas.CreateExit(ctx, saida)
	if err != nil {
		return nil, fmt.Errorf("criar a nota de beneficiamento: %w", err)
	}

	for _, linha := range nota.Linhas {
		item := &fiscalentity.FiscalExitItem{
			FiscalExitID:     criada.ID,
			Sequence:         linha.Sequencia,
			Ncm:              textoOuNil(linha.NCM),
			Cfop:             linha.CFOP,
			Quantity:         paraFloat(linha.Quantidade),
			UnitPrice:        paraFloat(linha.ValorUnit),
			TotalPrice:       paraFloat(linha.ValorTotal),
			CstICMS:          textoOuNil(linha.CSTICMS),
			Description:      textoOuNil(linha.Descricao),
			OrigemMercadoria: "0",
			// ICMS zero nas duas pontas: diferimento no serviço, suspensão no
			// material. Explícito para ninguém supor que faltou calcular.
			BaseICMS:  0,
			AliqICMS:  0,
			ValorICMS: 0,
		}
		if linha.CSTPIS != "" {
			item.CstPIS = textoOuNil(linha.CSTPIS)
			item.AliqPIS = paraFloat(linha.AliqPIS)
			item.ValorPIS = paraFloat(linha.ValorPIS)
		}
		if linha.CSTCOFINS != "" {
			item.CstCOFINS = textoOuNil(linha.CSTCOFINS)
			item.AliqCOFINS = paraFloat(linha.AliqCOFINS)
			item.ValorCOFINS = paraFloat(linha.ValorCOFINS)
		}
		if _, err := uc.Notas.CreateExitItem(ctx, item); err != nil {
			return nil, fmt.Errorf("gravar a linha %d da nota: %w", linha.Sequencia, err)
		}
	}

	// 3. Baixa o saldo, vinculado à nota. Idempotente: se esta etapa falhar, basta
	//    repetir o faturamento da mesma nota.
	baixas := make([]domrepo.MovimentoDaNota, 0, len(nota.Linhas))
	for _, linha := range nota.Linhas {
		if linha.ItemDaRemessa == nil {
			continue // linha de serviço não baixa saldo
		}
		baixas = append(baixas, domrepo.MovimentoDaNota{
			RemittanceItemID: linha.ItemDaRemessa.ID,
			MovementType:     linha.Movimento,
			Quantity:         linha.Quantidade.String(),
			UnitValue:        linha.ValorUnit.String(),
			CFOP:             linha.CFOP,
		})
	}

	movimentos, err := uc.Repo.RegistrarMovimentosDaNota(ctx, criada.ID, baixas, usuario)
	if err != nil {
		// A nota fica em rascunho, sem baixa. A mensagem diz o que fazer, porque
		// deixar o operador adivinhando é o que gera nota duplicada.
		return nil, fmt.Errorf(
			"a nota %d/%s foi criada, mas o saldo do cliente NÃO foi baixado: %w. "+
				"Repita o faturamento desta nota para concluir — a baixa não duplica",
			criada.NumeroNF, criada.Serie, err)
	}

	atualizada, err := uc.Repo.BuscarPorID(ctx, pedido.RemittanceID)
	if err != nil {
		return nil, err
	}

	return &ResultadoDoFaturamento{
		FiscalExitID:  criada.ID,
		NumeroNF:      criada.NumeroNF,
		Serie:         criada.Serie,
		Nota:          nota,
		Movimentos:    movimentos,
		SaldoRestante: atualizada.SaldoTotal(),
	}, nil
}

// ConferirBaixaDaNota é a trava antes de transmitir: uma nota de beneficiamento só
// pode ir para a SEFAZ se o saldo que ela devolve já foi baixado. Sem isso, o
// material sai fiscalmente e continua aparecendo como presente.
func (uc *UseCase) ConferirBaixaDaNota(ctx context.Context, fiscalExitID int64) error {
	movimentos, err := uc.Repo.MovimentosDaNota(ctx, fiscalExitID)
	if err != nil {
		return err
	}
	if len(movimentos) == 0 {
		return errorsuc.NewConflictError(fmt.Sprintf(
			"a nota %d é de beneficiamento e o saldo do cliente ainda não foi baixado; "+
				"refaça o faturamento antes de transmitir", fiscalExitID))
	}
	return nil
}

func aplicarDestinatario(saida *fiscalentity.FiscalExit, d DadosDoDestinatario) {
	saida.CnpjDestinatario = textoOuNil(d.CNPJ)
	saida.RazaoSocialDestinatario = textoOuNil(d.RazaoSocial)
	saida.IEDestinatario = textoOuNil(d.IE)
	saida.UFDestinatario = textoOuNil(d.UF)
	saida.DestLogradouro = textoOuNil(d.Logradouro)
	saida.DestNumero = textoOuNil(d.Numero)
	saida.DestComplemento = textoOuNil(d.Complemento)
	saida.DestBairro = textoOuNil(d.Bairro)
	saida.DestMunicipio = textoOuNil(d.Municipio)
	saida.DestCodigoMunicipio = textoOuNil(d.CodigoMunicipio)
	saida.DestCEP = textoOuNil(d.CEP)
	saida.DestEmail = textoOuNil(d.Email)
	saida.DestTelefone = textoOuNil(d.Telefone)
}

func textoOuNil(valor string) *string {
	limpo := strings.TrimSpace(valor)
	if limpo == "" {
		return nil
	}
	return &limpo
}

// paraFloat converte para o tipo que a entidade fiscal usa. A conversão acontece
// só aqui, na borda: todo o cálculo até este ponto foi decimal.
func paraFloat(v decimal.Decimal) float64 {
	f, _ := v.Float64()
	return f
}

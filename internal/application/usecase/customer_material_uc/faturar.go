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
	"log/slog"
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
	// RascunhoDeBeneficiamento devolve o rascunho já criado para a remessa (nulo se
	// não houver). É o que torna a repetição do faturamento segura de verdade: sem
	// ela, repetir pedia número novo e criava OUTRA nota em vez de concluir a
	// original.
	RascunhoDeBeneficiamento(ctx context.Context, remessaID int64) (*fiscalentity.FiscalExit, error)
	// GetExitItems diz quais linhas o rascunho já tem, para a retomada criar apenas
	// as que faltam.
	GetExitItems(ctx context.Context, fiscalExitID int64) ([]*fiscalentity.FiscalExitItem, error)
	// GetFiscalConfig dá a UF da empresa e o texto legal da nota de retorno.
	GetFiscalConfig(ctx context.Context) (*fiscalentity.FiscalConfig, error)
	// GetExitByID e CancelExitWithMotivo servem ao descarte da nota não autorizada.
	GetExitByID(ctx context.Context, id int64) (*fiscalentity.FiscalExit, error)
	CancelExitWithMotivo(ctx context.Context, id int64, motivo string, userID uuid.UUID) (*fiscalentity.FiscalExit, error)
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
		return nil, errorsuc.NewValidationError("a emissão de notas não está disponível nesta instalação: fale com o suporte")
	}
	if strings.TrimSpace(usuario) == "" {
		return nil, errorsuc.NewValidationError("sua sessão não identificou o usuário: entre no sistema novamente")
	}
	autor, err := uuid.Parse(strings.TrimSpace(usuario))
	if err != nil {
		return nil, errorsuc.NewValidationError("sua sessão não identificou o usuário: entre no sistema novamente")
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
				"os dados do cliente não estão disponíveis para esta nota: fale com o suporte")
		}
		resolvido, err := uc.Clientes.DadosFiscaisDoCliente(ctx, remessa.CustomerCode)
		if err != nil {
			return nil, err
		}
		destinatario = resolvido
	}

	// O serviço vem do pedido de beneficiamento: a linha, o preço e o saldo. O
	// que a tela mandou prevalece; a quantidade não passa do saldo da linha.
	servico, err := uc.servicoDoPedido(ctx, remessa, pedido.Servico)
	if err != nil {
		return nil, err
	}

	// Serviço com diferimento é operação entre contribuintes: a SEFAZ rejeita o CST 51
	// sem inscrição estadual (508). Recusar aqui evita baixar o saldo de uma nota que
	// não vai passar.
	if strings.TrimSpace(destinatario.IE) == "" {
		return nil, errorsuc.NewValidationError(
			"cadastre a inscrição estadual do cliente: a nota de beneficiamento é operação entre contribuintes")
	}

	// 1. Monta e valida. Nada foi gravado ainda, então uma recusa aqui não deixa
	//    rastro — é onde CFOP, CST, saldo e UF são conferidos.
	config, err := uc.Notas.GetFiscalConfig(ctx)
	if err != nil {
		if _, naoTem := errorsuc.AsNotFound(err); naoTem {
			return nil, errorsuc.NewValidationError("preencha a tela Configuração Fiscal (VFIS0100) antes de faturar o beneficiamento")
		}
		return nil, err
	}
	emitente := EmitenteDaNota{UF: config.UFEmpresa}
	if config.ObsRetornoIndustrializacao != nil {
		emitente.ObservacaoLegal = *config.ObsRetornoIndustrializacao
	}
	if config.CBenefDiferimento != nil {
		emitente.CBenefDiferimento = *config.CBenefDiferimento
	}
	if config.CBenefSuspensao != nil {
		emitente.CBenefSuspensao = *config.CBenefSuspensao
	}
	nota, err := MontarNotaDeRetorno(remessa, emitente, destinatario.UF, servico, pedido.Devolucoes)
	if err != nil {
		return nil, err
	}

	// 2. RETOMA o rascunho pendente desta remessa, se houver.
	//
	// Este passo é a correção de um defeito real: quando a criação das linhas ou a
	// baixa falhava, o rascunho ficava gravado e a mensagem mandava repetir — mas
	// repetir pedia número novo e criava OUTRA nota. A repetição anunciada como
	// segura produzia duplicata. Não há transação possível entre os dois módulos;
	// retomar é o que resolve.
	criada, err := uc.Notas.RascunhoDeBeneficiamento(ctx, pedido.RemittanceID)
	if err != nil {
		return nil, fmt.Errorf("procurar nota em aberto desta remessa: %w", err)
	}
	retomada := criada != nil

	if !retomada {
		serie := strings.TrimSpace(pedido.Serie)
		if serie == "" {
			serie = "1"
		}
		emissao := uc.agora()
		if pedido.DataEmissao != nil {
			emissao = *pedido.DataEmissao
		}
		criada, err = uc.criarRascunhoNumerado(ctx, pedido, remessa, nota, destinatario, serie, emissao, autor)
		if err != nil {
			return nil, err
		}
	}

	// 3. Cria as linhas que ainda faltam. Numa retomada, as já gravadas ficam como
	//    estão: recriá-las duplicaria a linha na nota.
	jaGravadas := map[int]bool{}
	if retomada {
		existentes, err := uc.Notas.GetExitItems(ctx, criada.ID)
		if err != nil {
			return nil, fmt.Errorf("ler as linhas da nota em aberto: %w", err)
		}
		for _, it := range existentes {
			jaGravadas[it.Sequence] = true
		}
	}

	if err := uc.gravarLinhas(ctx, criada.ID, nota, jaGravadas); err != nil {
		return nil, err
	}

	return uc.concluirFaturamento(ctx, pedido, nota, criada, usuario, retomada)
}

// tentativasDeNumerar cobre a disputa rara pelo número; passar disso é outra falha.
const tentativasDeNumerar = 3

// criarRascunhoNumerado pede o número e grava a capa. O número é o maior + 1: duas
// notas ao mesmo tempo tiram o mesmo e uma perde no índice único. Esta tenta de novo
// com o próximo, a não ser que a disputa tenha sido pela própria remessa.
func (uc *UseCase) criarRascunhoNumerado(
	ctx context.Context, pedido PedidoDeFaturamento, remessa *entity.Remessa,
	nota *NotaDeRetorno, destinatario *DadosDoDestinatario,
	serie string, emissao time.Time, autor uuid.UUID,
) (*fiscalentity.FiscalExit, error) {
	var ultimoErro error
	for tentativa := 0; tentativa < tentativasDeNumerar; tentativa++ {
		numero, err := uc.Notas.GetNextNFNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("obter o número da nota: %w", err)
		}
		criada, err := uc.criarRascunho(ctx, pedido, remessa, nota, destinatario, numero, serie, emissao, autor)
		if err == nil {
			return criada, nil
		}
		ultimoErro = err
		if outra, errBusca := uc.Notas.RascunhoDeBeneficiamento(ctx, pedido.RemittanceID); errBusca == nil && outra != nil {
			return nil, errorsuc.NewConflictError("esta remessa acabou de ser faturada: atualize a tela para ver a nota")
		}
	}
	return nil, ultimoErro
}

// criarRascunho grava a capa da nota em RASCUNHO, vinculada à remessa.
func (uc *UseCase) criarRascunho(
	ctx context.Context, pedido PedidoDeFaturamento, remessa *entity.Remessa,
	nota *NotaDeRetorno, destinatario *DadosDoDestinatario,
	numero int64, serie string, emissao time.Time, autor uuid.UUID,
) (*fiscalentity.FiscalExit, error) {
	// 2. Cria a nota em rascunho.
	saida := &fiscalentity.FiscalExit{
		NumeroNF:         numero,
		Serie:            serie,
		DataEmissao:      emissao,
		Cfop:             CFOPServico,
		NaturezaOperacao: nota.NaturezaOperacao,
		ValorProdutos:    nota.ValorTotal,
		ValorPIS:         nota.ValorPIS,
		ValorCOFINS:      nota.ValorCOFINS,
		ValorTotal:       nota.ValorTotal,
		// Rascunho de propósito: a transmissão é passo separado, e é ela que confere
		// se o saldo já foi baixado.
		Status:         fiscalentity.ExitStatusDraft,
		IsActive:       true,
		CreatedBy:      autor,
		CustomerCode:   &remessa.CustomerCode,
		SalesOrderCode: remessa.SalesOrderCode,
		SourceType:     textoOuNil("BENEFICIAMENTO"),
		// O texto legal do diferimento e da suspensão é exigência da nota: sem ele a
		// operação sai sem o fundamento do ICMS zero.
		InformacoesComplementares: textoOuNil(nota.Observacao),
		// A nota do cliente que originou o material, quando a chave foi informada.
		NFeReferenciada: chaveDaRemessa(remessa),
		// O vínculo com a remessa é o que permite RETOMAR este rascunho quando o
		// faturamento falha depois daqui.
		CustomerMaterialRemittanceID: &pedido.RemittanceID,
	}
	aplicarDestinatario(saida, *destinatario)

	criada, err := uc.Notas.CreateExit(ctx, saida)
	if err != nil {
		return nil, fmt.Errorf("criar a nota de beneficiamento: %w", err)
	}
	return criada, nil
}

// gravarLinhas cria as linhas da nota, pulando as sequências que já existem.
func (uc *UseCase) gravarLinhas(ctx context.Context, notaID int64, nota *NotaDeRetorno, jaGravadas map[int]bool) error {
	for _, linha := range nota.Linhas {
		if jaGravadas[linha.Sequencia] {
			continue // retomada: recriar duplicaria a linha na nota
		}
		item := &fiscalentity.FiscalExitItem{
			FiscalExitID:     notaID,
			Sequence:         linha.Sequencia,
			Ncm:              textoOuNil(linha.NCM),
			Cfop:             linha.CFOP,
			Quantity:         linha.Quantidade,
			UnitPrice:        linha.ValorUnit,
			TotalPrice:       linha.ValorTotal,
			CstICMS:          textoOuNil(linha.CSTICMS),
			Description:      textoOuNil(linha.Descricao),
			OrigemMercadoria: "0",
			// ⚠️ Unidade e código do produto (migração 000374). Sem eles o autorizador
			// manda `uCom` fixo em "UN" e `cProd` "0": a nota de retorno do
			// beneficiamento tem linhas em KG com o código DO CLIENTE, e sairia
			// descrevendo outra mercadoria.
			UnidadeComercial: textoOuNil(linha.Unidade),
			CodigoProduto:    textoOuNil(linha.CodigoProduto),
			// Linha do pedido de beneficiamento (só no serviço).
			SalesOrderItemCode:    linha.SalesOrderItemCode,
			CodigoBeneficioFiscal: textoOuNil(linha.CBenef),
			// Só no serviço: é o que leva o pDif de 100% à nota (rejeição 929 sem ele).
			ICMSDiferimentoPct: diferimentoDaLinha(linha.CSTICMS),
			// ICMS zero nas duas pontas: diferimento no serviço, suspensão no
			// material. Explícito para ninguém supor que faltou calcular.
			BaseICMS:  decimal.NewFromFloat(0),
			AliqICMS:  decimal.NewFromFloat(0),
			ValorICMS: decimal.NewFromFloat(0),
		}
		// O CST de PIS/COFINS vai SEMPRE, mesmo com valor zero: no material que volta
		// é o CST 49 ("outras operações de saída"), e deixar o campo vazio faria o
		// autorizador cair no padrão "01", declarando o material como tributado.
		if linha.CSTPIS != "" {
			item.CstPIS = textoOuNil(linha.CSTPIS)
			item.AliqPIS = linha.AliqPIS
			item.ValorPIS = linha.ValorPIS
		}
		if linha.CSTCOFINS != "" {
			item.CstCOFINS = textoOuNil(linha.CSTCOFINS)
			item.AliqCOFINS = linha.AliqCOFINS
			item.ValorCOFINS = linha.ValorCOFINS
		}
		if _, err := uc.Notas.CreateExitItem(ctx, item); err != nil {
			return fmt.Errorf("gravar a linha %d da nota: %w", linha.Sequencia, err)
		}
	}
	return nil
}

// concluirFaturamento baixa o saldo vinculado à nota e monta o resultado.
func (uc *UseCase) concluirFaturamento(
	ctx context.Context, pedido PedidoDeFaturamento, nota *NotaDeRetorno,
	criada *fiscalentity.FiscalExit, usuario string, retomada bool,
) (*ResultadoDoFaturamento, error) {
	// 4. Baixa o saldo, vinculado à nota. Idempotente: se esta etapa falhar, basta
	//    repetir o faturamento — a nota é RETOMADA, não recriada.
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
		// Requisição simultânea já baixou por esta nota: a baixa aconteceu.
		if existentes, errBusca := uc.Repo.MovimentosDaNota(ctx, criada.ID); errBusca == nil && len(existentes) > 0 {
			movimentos, err = existentes, nil
		}
	}
	if err != nil {
		// A nota fica em rascunho, sem baixa. A mensagem diz o que fazer, porque
		// deixar o operador adivinhando é o que gera nota duplicada.
		slog.ErrorContext(ctx, "nota de beneficiamento criada sem a baixa do saldo", "nota", criada.ID, "erro", err)
		return nil, errorsuc.NewConflictError(
			"a nota foi criada, mas o saldo do cliente NÃO foi baixado. " +
				"Repita o faturamento desta remessa para concluir: o sistema retoma esta " +
				"mesma nota em vez de criar outra, e a baixa não duplica")
	}

	_ = retomada // a retomada já foi decidida antes; aqui só o resultado importa
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
		return errorsuc.NewConflictError(
			"o saldo do cliente desta nota de beneficiamento ainda não foi baixado: " +
				"refaça o faturamento da remessa antes de transmitir")
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

func diferimentoDaLinha(cst string) decimal.Decimal {
	if cst == CSTICMSServico {
		return decimal.NewFromInt(100)
	}
	return decimal.Zero
}

// DescartarNota desfaz a nota de beneficiamento que não foi autorizada e devolve o
// saldo ao cliente. Sem isto, nota rejeitada ou rascunho errado prendiam o material:
// só a nota autorizada pode ser cancelada na SEFAZ. Devolve o número que ficou sem uso.
//
// O estorno vem antes do cancelamento porque é idempotente: se a segunda etapa falhar,
// repetir a operação conclui sem somar saldo de novo.
func (uc *UseCase) DescartarNota(ctx context.Context, fiscalExitID int64, usuario, motivo string) (int64, error) {
	if uc.Notas == nil {
		return 0, errorsuc.NewValidationError("a emissão de notas não está disponível nesta instalação: fale com o suporte")
	}
	autor, err := uuid.Parse(strings.TrimSpace(usuario))
	if err != nil {
		return 0, errorsuc.NewValidationError("sua sessão não identificou o usuário: entre no sistema novamente")
	}
	nota, err := uc.Notas.GetExitByID(ctx, fiscalExitID)
	if err != nil {
		return 0, err
	}
	if nota.SourceType == nil || *nota.SourceType != "BENEFICIAMENTO" {
		return 0, errorsuc.NewConflictError("só nota de beneficiamento é descartada por aqui")
	}
	if nota.Status != fiscalentity.ExitStatusDraft && nota.Status != fiscalentity.ExitStatusRejected {
		return 0, errorsuc.NewConflictError(
			"só nota em rascunho ou rejeitada pode ser descartada; a autorizada é cancelada na SEFAZ")
	}
	if _, err := uc.EstornarNota(ctx, fiscalExitID, usuario, motivo); err != nil {
		return 0, err
	}
	if _, err := uc.Notas.CancelExitWithMotivo(ctx, fiscalExitID, strings.TrimSpace(motivo), autor); err != nil {
		return 0, fmt.Errorf("descartar a nota %d: %w", nota.NumeroNF, err)
	}
	return nota.NumeroNF, nil
}

// chaveDaRemessa devolve a chave da NF-e do cliente, só quando tem os 44 dígitos.
func chaveDaRemessa(remessa *entity.Remessa) *string {
	if remessa.NFeKey == nil || len(strings.TrimSpace(*remessa.NFeKey)) != 44 {
		return nil
	}
	return textoOuNil(*remessa.NFeKey)
}

func textoOuNil(valor string) *string {
	limpo := strings.TrimSpace(valor)
	if limpo == "" {
		return nil
	}
	return &limpo
}

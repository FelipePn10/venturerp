// Package customer_material_uc é o beneficiamento: material do cliente em poder da
// empresa, do recebimento da NF-e de remessa até o retorno fiscal.
//
// A camada é fina de propósito. As invariantes do saldo vivem no repositório e no
// banco, porque são elas que não podem ser contornadas — nem por outro caminho de
// código, nem por SQL manual numa correção às pressas.
package customer_material_uc

import (
	"context"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	"github.com/shopspring/decimal"
)

// PrazoFiscalPadraoDias é o prazo de retorno da operação de beneficiamento.
// Fica aqui, e não fixado na tela, para mudar num lugar só se a regra mudar.
const PrazoFiscalPadraoDias = 30

type UseCase struct {
	Repo domrepo.Repository
	// Notas emite a nota de saída do beneficiamento. Opcional: sem ela o módulo
	// controla o saldo e a trilha, e só o faturamento fica indisponível.
	Notas EscritorDeNota
	// Clientes resolve o destinatário da nota pelo cadastro. Opcional: sem ele a
	// tela precisa enviar o destinatário completo.
	Clientes ResolvedorDeDestinatario
	// Agora existe para o cálculo do prazo ser testável sem esperar o relógio.
	Agora func() time.Time
}

func (uc *UseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

// ItemRecebido é uma linha da nota do cliente, como a conferência a informa.
type ItemRecebido struct {
	LineNumber       int32
	CustomerItemCode string
	ItemCode         *int64
	Description      string
	NCM              string
	CST              *string
	UOM              string
	QtyInvoiced      string
	QtyReceived      string
	UnitValue        string
	DivergenceReason *string
	WarehouseID      *int64
	Address          *string
}

// NovaRemessa é o recebimento de material do cliente.
type NovaRemessa struct {
	CustomerCode         int64
	NFeNumber            int64
	NFeSeries            string
	NFeKey               *string
	CFOP                 string
	IssueDate            time.Time
	ReceivedAt           time.Time
	FiscalReturnDeadline *time.Time
	TotalValue           string
	SalesOrderCode       *int64
	Notes                *string
	Itens                []ItemRecebido
}

// Receber registra a remessa. Duas regras de negócio moram aqui porque dependem do
// contexto da operação, não da forma do dado:
//
//  1. material sem pedido cadastrado entra BLOQUEADO — as regras do cliente proíbem
//     usá-lo na produção antes da regularização;
//  2. conferência divergente da nota também bloqueia, pelo mesmo motivo.
func (uc *UseCase) Receber(ctx context.Context, nova NovaRemessa, usuario string) (*entity.Remessa, error) {
	// Quem recebeu o material fica gravado: num controle de material que não é da
	// empresa, "quem deu entrada nisso" precisa ter resposta. Sem isso o insert
	// chegava ao banco com created_by vazio e falhava na conversão para uuid.
	if strings.TrimSpace(usuario) == "" {
		return nil, errorsuc.NewValidationError("sessão sem usuário identificado")
	}
	if len(nova.Itens) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos um item da nota de remessa")
	}
	if nova.CustomerCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o cliente proprietário do material")
	}
	if nova.NFeNumber <= 0 {
		return nil, errorsuc.NewValidationError("informe o número da NF-e de remessa")
	}
	if nova.IssueDate.IsZero() {
		return nil, errorsuc.NewValidationError("informe a data de emissão da NF-e")
	}

	recebimento := nova.ReceivedAt
	if recebimento.IsZero() {
		recebimento = uc.agora()
	}
	prazo := nova.FiscalReturnDeadline
	if prazo == nil {
		calculado := nova.IssueDate.AddDate(0, 0, PrazoFiscalPadraoDias)
		prazo = &calculado
	}
	if prazo.Before(nova.IssueDate) {
		return nil, errorsuc.NewValidationError("o prazo de retorno não pode ser anterior à emissão da nota")
	}

	total, err := decimalOuZero(nova.TotalValue, "valor total da nota")
	if err != nil {
		return nil, err
	}

	remessa := &entity.Remessa{
		CustomerCode:         nova.CustomerCode,
		NFeNumber:            nova.NFeNumber,
		NFeSeries:            primeiroNaoVazio(nova.NFeSeries, "1"),
		NFeKey:               nova.NFeKey,
		CFOP:                 primeiroNaoVazio(nova.CFOP, "5901"),
		IssueDate:            nova.IssueDate,
		ReceivedAt:           recebimento,
		FiscalReturnDeadline: *prazo,
		TotalValue:           total,
		Status:               entity.StatusAberta,
		SalesOrderCode:       nova.SalesOrderCode,
		Notes:                nova.Notes,
		CreatedBy:            usuario,
	}

	divergente := false
	for i, bruto := range nova.Itens {
		item, err := montarItem(bruto, i)
		if err != nil {
			return nil, err
		}
		if !item.QtyReceived.Equal(item.QtyInvoiced) {
			divergente = true
		}
		remessa.Itens = append(remessa.Itens, item)
	}

	switch {
	case nova.SalesOrderCode == nil:
		remessa.Blocked = true
		motivo := "material recebido sem pedido de beneficiamento cadastrado"
		remessa.BlockReason = &motivo
	case divergente:
		remessa.Blocked = true
		motivo := "quantidade conferida diferente da NF-e; regularize antes de produzir"
		remessa.BlockReason = &motivo
	}

	return uc.Repo.Registrar(ctx, remessa)
}

func montarItem(bruto ItemRecebido, indice int) (*entity.ItemRemessa, error) {
	if strings.TrimSpace(bruto.CustomerItemCode) == "" {
		return nil, errorsuc.NewValidationError("informe o código do item como consta na nota do cliente")
	}
	if strings.TrimSpace(bruto.Description) == "" {
		return nil, errorsuc.NewValidationError("informe a descrição do item " + bruto.CustomerItemCode)
	}
	// O NCM da entrada é o que volta no retorno fiscal: sem ele a nota de retorno
	// sai com classificação diferente da que o cliente enviou.
	if strings.TrimSpace(bruto.NCM) == "" {
		return nil, errorsuc.NewValidationError("informe o NCM do item " + bruto.CustomerItemCode + " como consta na nota")
	}
	if strings.TrimSpace(bruto.UOM) == "" {
		return nil, errorsuc.NewValidationError("informe a unidade do item " + bruto.CustomerItemCode)
	}

	faturada, err := decimalObrigatorio(bruto.QtyInvoiced, "quantidade da nota do item "+bruto.CustomerItemCode)
	if err != nil {
		return nil, err
	}
	if !faturada.IsPositive() {
		return nil, errorsuc.NewValidationError("a quantidade da nota do item " + bruto.CustomerItemCode + " deve ser maior que zero")
	}
	// Sem conferência informada, vale o que a nota diz.
	recebida := faturada
	if strings.TrimSpace(bruto.QtyReceived) != "" {
		recebida, err = decimalObrigatorio(bruto.QtyReceived, "quantidade conferida do item "+bruto.CustomerItemCode)
		if err != nil {
			return nil, err
		}
		if recebida.IsNegative() {
			return nil, errorsuc.NewValidationError("a quantidade conferida do item " + bruto.CustomerItemCode + " não pode ser negativa")
		}
	}
	if !recebida.Equal(faturada) && (bruto.DivergenceReason == nil || strings.TrimSpace(*bruto.DivergenceReason) == "") {
		return nil, errorsuc.NewValidationError(
			"a quantidade conferida do item " + bruto.CustomerItemCode + " difere da nota; informe o motivo da divergência")
	}

	valor, err := decimalOuZero(bruto.UnitValue, "valor unitário do item "+bruto.CustomerItemCode)
	if err != nil {
		return nil, err
	}

	linha := bruto.LineNumber
	if linha <= 0 {
		linha = int32(indice + 1)
	}
	return &entity.ItemRemessa{
		LineNumber:       linha,
		CustomerItemCode: strings.TrimSpace(bruto.CustomerItemCode),
		ItemCode:         bruto.ItemCode,
		Description:      strings.TrimSpace(bruto.Description),
		NCM:              strings.TrimSpace(bruto.NCM),
		CST:              bruto.CST,
		UOM:              strings.ToUpper(strings.TrimSpace(bruto.UOM)),
		QtyInvoiced:      faturada,
		QtyReceived:      recebida,
		UnitValue:        valor,
		DivergenceReason: bruto.DivergenceReason,
		WarehouseID:      bruto.WarehouseID,
		Address:          bruto.Address,
	}, nil
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*entity.Remessa, error) {
	return uc.Repo.BuscarPorID(ctx, id)
}

func (uc *UseCase) Listar(ctx context.Context, f domrepo.FiltroRemessa) ([]*entity.Remessa, error) {
	return uc.Repo.Listar(ctx, f)
}

func (uc *UseCase) Saldo(ctx context.Context, f domrepo.FiltroSaldo) ([]*entity.SaldoPorItem, error) {
	return uc.Repo.SaldoPorItem(ctx, f)
}

func (uc *UseCase) Movimentos(ctx context.Context, itemID int64) ([]*entity.Movimento, error) {
	return uc.Repo.MovimentosDoItem(ctx, itemID)
}

// Movimentar lança retorno, sobra, sucata ou ajuste. O CFOP entra automático
// quando não informado: 5902 no retorno com o faturamento do serviço, 5903 em
// sobra e sucata devolvidas — é a regra que o cliente documentou, e deixá-la ao
// operador em cada lançamento é convite a erro fiscal.
func (uc *UseCase) Movimentar(ctx context.Context, mov domrepo.NovoMovimento, usuario string) (*entity.Movimento, error) {
	if mov.CFOP == nil || strings.TrimSpace(*mov.CFOP) == "" {
		if padrao := cfopPadrao(mov.MovementType); padrao != "" {
			mov.CFOP = &padrao
		}
	}
	return uc.Repo.RegistrarMovimento(ctx, mov, usuario)
}

func cfopPadrao(tipo entity.TipoMovimento) string {
	switch tipo {
	case entity.MovimentoRetorno:
		return "5902"
	case entity.MovimentoSobra, entity.MovimentoSucata:
		return "5903"
	default:
		return ""
	}
}

// Bloquear, Desbloquear e Encerrar exigem quem assina: material de terceiro
// bloqueado ou encerrado com saldo é decisão de pessoa, e a trilha da migração
// 000371 registra qual.
func (uc *UseCase) Bloquear(ctx context.Context, id int64, motivo, usuario string) error {
	if strings.TrimSpace(usuario) == "" {
		return errorsuc.NewValidationError("não foi possível identificar o usuário da sessão")
	}
	return uc.Repo.Bloquear(ctx, id, motivo, usuario)
}

func (uc *UseCase) Desbloquear(ctx context.Context, id int64, usuario string) error {
	if strings.TrimSpace(usuario) == "" {
		return errorsuc.NewValidationError("não foi possível identificar o usuário da sessão")
	}
	return uc.Repo.Desbloquear(ctx, id, usuario)
}

func (uc *UseCase) Encerrar(ctx context.Context, id int64, motivo, usuario string) error {
	if strings.TrimSpace(usuario) == "" {
		return errorsuc.NewValidationError("não foi possível identificar o usuário da sessão")
	}
	return uc.Repo.Encerrar(ctx, id, motivo, usuario)
}

// EstornarNota devolve ao cliente o saldo que uma nota de retorno baixou. Chamada
// no cancelamento da NF-e: o documento 6 da Usimac exige estornar as movimentações
// relacionadas, e sem isso o saldo afirma que o material voltou quando ele
// continua no pátio.
//
// O motivo mínimo é o mesmo do cancelamento fiscal: quinze caracteres. Estorno com
// "erro" escrito no motivo não explica nada a quem conferir seis meses depois.
func (uc *UseCase) EstornarNota(ctx context.Context, fiscalExitID int64, usuario, motivo string) ([]*entity.Movimento, error) {
	if strings.TrimSpace(usuario) == "" {
		return nil, errorsuc.NewValidationError("não foi possível identificar o usuário da sessão")
	}
	if len(strings.TrimSpace(motivo)) < 15 {
		return nil, errorsuc.NewValidationError("descreva o motivo do estorno com pelo menos 15 caracteres")
	}
	return uc.Repo.EstornarMovimentosDaNota(ctx, fiscalExitID, usuario, strings.TrimSpace(motivo))
}

// EstornarNotaCancelada é o que o cancelamento fiscal chama. Existe separada de
// EstornarNota porque o módulo fiscal não deve depender das entidades do
// beneficiamento para desfazer uma baixa — só precisa saber se deu certo.
func (uc *UseCase) EstornarNotaCancelada(ctx context.Context, fiscalExitID int64, usuario, motivo string) error {
	_, err := uc.EstornarNota(ctx, fiscalExitID, usuario, motivo)
	return err
}

// TrilhaDaRemessa devolve o histórico da remessa. Passa pela busca da remessa
// antes para o 404 vir de um registro que não existe na empresa da sessão, e não
// de uma trilha vazia — trilha vazia e remessa de outra empresa não podem
// responder a mesma coisa.
func (uc *UseCase) TrilhaDaRemessa(ctx context.Context, id int64, limite int) ([]*entity.EventoDeAuditoria, error) {
	if _, err := uc.Repo.BuscarPorID(ctx, id); err != nil {
		return nil, err
	}
	return uc.Repo.TrilhaDaRemessa(ctx, id, limite)
}

func decimalObrigatorio(bruto, campo string) (decimal.Decimal, error) {
	v, err := decimal.NewFromString(strings.TrimSpace(bruto))
	if err != nil {
		return decimal.Zero, errorsuc.NewValidationError(campo + " inválido")
	}
	return v, nil
}

func decimalOuZero(bruto, campo string) (decimal.Decimal, error) {
	if strings.TrimSpace(bruto) == "" {
		return decimal.Zero, nil
	}
	return decimalObrigatorio(bruto, campo)
}

func primeiroNaoVazio(valor, padrao string) string {
	if strings.TrimSpace(valor) == "" {
		return padrao
	}
	return strings.TrimSpace(valor)
}

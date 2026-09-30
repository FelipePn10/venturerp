// Package repository declara o contrato de persistência do material de terceiro.
package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
)

// FiltroRemessa restringe a consulta de remessas. Todos os campos são opcionais;
// a empresa da sessão é sempre aplicada pelo repositório.
type FiltroRemessa struct {
	CustomerCode     *int64
	NFeNumber        *int64
	SalesOrderCode   *int64
	Status           []entity.StatusRemessa
	SomenteBloqueada bool
	// SomenteComSaldo traz apenas remessas com material ainda em poder da empresa.
	SomenteComSaldo bool
	// VencendoAte filtra pelo prazo fiscal de retorno: remessas cujo prazo cai até
	// esta data. É a consulta que evita perder o prazo de 30 dias.
	VencendoAte *time.Time
	Busca       string
	Limit       int
	Offset      int
}

// FiltroSaldo restringe a visão agregada por item.
type FiltroSaldo struct {
	CustomerCode     *int64
	CustomerItemCode string
	ItemCode         *int64
	Busca            string
}

// NovoMovimento é o pedido de lançamento no razão. Quantidade e valor chegam como
// texto para não perder precisão decimal no caminho da API até o banco.
type NovoMovimento struct {
	RemittanceItemID  int64
	MovementType      entity.TipoMovimento
	Quantity          string
	UnitValue         string
	CFOP              *string
	ProductionOrderID *int64
	FiscalExitID      *int64
	ScrapDestination  *entity.DestinoSucata
	Reason            *string
	IdempotencyKey    string
}

// Repository é o contrato usado pelos casos de uso.
type Repository interface {
	// Registrar grava a remessa e seus itens atomicamente. Uma remessa sem item, ou
	// um item que falhe, desfaz tudo.
	Registrar(ctx context.Context, remessa *entity.Remessa) (*entity.Remessa, error)
	// BuscarPorID traz a remessa com os itens, ou NotFound.
	BuscarPorID(ctx context.Context, id int64) (*entity.Remessa, error)
	// Listar devolve remessas com os itens carregados numa consulta por lote.
	Listar(ctx context.Context, filtro FiltroRemessa) ([]*entity.Remessa, error)
	// SaldoPorItem agrega o material de terceiro em poder da empresa.
	SaldoPorItem(ctx context.Context, filtro FiltroSaldo) ([]*entity.SaldoPorItem, error)
	// RegistrarMovimento lança no razão, atualiza o consumo da linha e recalcula a
	// situação da remessa — tudo na mesma transação. Repetir a mesma
	// `IdempotencyKey` devolve o movimento original.
	RegistrarMovimento(ctx context.Context, mov NovoMovimento, usuario string) (*entity.Movimento, error)
	// MovimentosDoItem devolve a trilha de um item, da mais antiga para a mais nova.
	MovimentosDoItem(ctx context.Context, itemID int64) ([]*entity.Movimento, error)
	// Bloquear e Desbloquear governam o material que não pode ir para a produção.
	Bloquear(ctx context.Context, id int64, motivo string, usuario string) error
	Desbloquear(ctx context.Context, id int64, usuario string) error
	// Encerrar fecha a remessa. Com saldo remanescente, exige motivo — é exceção
	// aprovada, e fica registrado quem aprovou.
	Encerrar(ctx context.Context, id int64, motivo string, usuario string) error
	// RegistrarMovimentosDaNota baixa, numa única transação, todo o material que a
	// nota de beneficiamento devolve, já vinculado a ela. Idempotente pela nota:
	// repetir a chamada com a mesma nota devolve os movimentos originais, porque a
	// falha entre criar a nota e baixar o saldo tem de ser recuperável por repetição.
	RegistrarMovimentosDaNota(ctx context.Context, fiscalExitID int64, linhas []MovimentoDaNota, usuario string) ([]*entity.Movimento, error)
	// MovimentosDaNota devolve o que uma nota baixou. É a conferência que impede
	// autorizar na SEFAZ uma nota cujo saldo não foi baixado.
	MovimentosDaNota(ctx context.Context, fiscalExitID int64) ([]*entity.Movimento, error)
	// TrilhaDaRemessa devolve a auditoria da remessa e de tudo que pende dela —
	// itens e movimentos —, do evento mais recente para o mais antigo. A trilha é
	// gravada por trigger, não por este pacote; aqui só se lê.
	TrilhaDaRemessa(ctx context.Context, remittanceID int64, limite int) ([]*entity.EventoDeAuditoria, error)
	// EstornarMovimentosDaNota devolve ao saldo do cliente o que a nota baixou,
	// quando a nota é cancelada. Idempotente: repetir não devolve duas vezes.
	EstornarMovimentosDaNota(ctx context.Context, fiscalExitID int64, usuario, motivo string) ([]*entity.Movimento, error)
}

// MovimentoDaNota é uma baixa que a nota de beneficiamento representa.
type MovimentoDaNota struct {
	RemittanceItemID int64
	MovementType     entity.TipoMovimento
	Quantity         string
	UnitValue        string
	CFOP             string
}

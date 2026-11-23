package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// LocalTerceiro é o local do almoxarifado "em poder de terceiros".
const LocalTerceiro = "TERCEIRO"

// AlmoxDoTerceiroTx devolve o almoxarifado TERCEIRO do fornecedor, criando-o na
// primeira remessa. Ele não conta para promessa nem aceita reserva: a peça está
// fora da fábrica.
func AlmoxDoTerceiroTx(ctx context.Context, tx pgx.Tx, empresa, fornecedor int64, ator uuid.UUID) (int64, error) {
	var nome string
	err := tx.QueryRow(ctx, `SELECT name FROM suppliers WHERE code=$1 AND enterprise_id=$2`, fornecedor, empresa).Scan(&nome)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errorsuc.NewValidationError(fmt.Sprintf("o fornecedor %d não existe nesta empresa", fornecedor))
	}
	if err != nil {
		return 0, fmt.Errorf("lendo o fornecedor da remessa: %w", err)
	}
	var id int64
	err = tx.QueryRow(ctx, `SELECT id FROM warehouse WHERE enterprise_id=$1 AND supplier_code=$2`, empresa, fornecedor).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("lendo o almoxarifado do terceiro: %w", err)
	}
	descricao := "Em poder de " + nome
	if len([]rune(descricao)) > 150 {
		descricao = string([]rune(descricao)[:150])
	}
	// warehouse.code é único no banco: leva a empresa.
	err = tx.QueryRow(ctx,
		`INSERT INTO warehouse (code, description, created_by, location, type, disposition, reservations_allowed, enterprise_id, supplier_code)
		 VALUES ($1, $2, $3, 'TERCEIRO', 'NORMAL', FALSE, FALSE, $4, $5)
		 ON CONFLICT (enterprise_id, supplier_code) WHERE supplier_code IS NOT NULL DO UPDATE SET supplier_code = EXCLUDED.supplier_code
		 RETURNING id`,
		fmt.Sprintf("TERC-%d-%d", empresa, fornecedor), descricao, ator, empresa, fornecedor).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("criando o almoxarifado do terceiro: %w", err)
	}
	return id, nil
}

// TransferirTx lança a transferência entre almoxarifados na transação de quem
// chamou (a OS de terceiro grava o movimento e o estoque juntos).
func TransferirTx(ctx context.Context, tx pgx.Tx, empresa int64, t entity.TransferenciaEstoque) (*entity.ResultadoTransferencia, error) {
	if t.DeAlmox == t.ParaAlmox {
		return nil, errorsuc.NewValidationError("origem e destino são o mesmo almoxarifado")
	}
	return transferirTx(ctx, tx, empresa, t, false)
}

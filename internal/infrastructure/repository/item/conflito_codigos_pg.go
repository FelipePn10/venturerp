package item

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

// conferirConflitoDeCodigos impede que um código digitado leve ao item errado.
// Os campos de código resolvem sempre pelo código atual do item; o código
// anterior e a referência do fabricante só servem de busca. Se um deles fosse o
// código atual de outro item, quem digitasse pensando no primeiro levaria o
// segundo sem aviso. Por isso: o código anterior e a referência do fabricante não
// podem ser o código atual de outro item; e, ao criar um item, o código dele não
// pode ser o código anterior, a referência do fabricante ou o código de
// fornecedor de outro item. (O código atual não muda depois de criado, então a
// segunda conferência é só na criação.)
func (r *RepositoryItemSQLC) conferirConflitoDeCodigos(ctx context.Context, it *entity.Item, criando bool) error {
	if r.pool == nil {
		return nil
	}
	proprio := strings.TrimSpace(string(it.BusinessCode))
	for rotulo, valor := range map[string]*string{"código anterior": it.PreviousCode, "referência do fabricante": it.ManufacturerPartNumber} {
		if valor == nil || strings.TrimSpace(*valor) == "" {
			continue
		}
		var outro string
		err := r.pool.QueryRow(ctx, `SELECT business_code FROM items
			 WHERE enterprise_id = $1 AND upper(business_code) = upper(btrim($2)) AND business_code <> $3 LIMIT 1`,
			it.EnterpriseID, *valor, proprio).Scan(&outro)
		if err == nil {
			return errorsuc.NewValidationError(fmt.Sprintf("o %s %s é o código atual do item %s: quem o digitasse num pedido ou numa nota levaria o outro item. Corrija ou deixe em branco", rotulo, strings.TrimSpace(*valor), outro))
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("conferindo o %s do item: %w", rotulo, err)
		}
	}
	// O tanque / linha precisa estar cadastrado e ativo (VPME0102): um número
	// solto fazia a promessa de entrega somar a ocupação num tanque que ninguém via.
	if t := it.Planning.TankCode; t != nil && *t > 0 {
		var ativa *bool
		err := r.pool.QueryRow(ctx, `SELECT is_active FROM delivery_capacity_lines WHERE enterprise_id = $1 AND code = $2`, it.EnterpriseID, int64(*t)).Scan(&ativa)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewValidationError(fmt.Sprintf("o tanque / linha %d não está cadastrado: cadastre nos Parâmetros da VPME0102ITE ou escolha outro", *t))
		}
		if err != nil {
			return fmt.Errorf("conferindo o tanque do item: %w", err)
		}
		if ativa != nil && !*ativa {
			return errorsuc.NewValidationError(fmt.Sprintf("o tanque / linha %d está desativado", *t))
		}
	}
	if !criando || proprio == "" {
		return nil
	}
	apelidos, err := ApelidosDoCodigo(ctx, r.pool, it.EnterpriseID, proprio)
	if err != nil {
		return err
	}
	for _, a := range apelidos {
		// GTIN não conflita: o código do item não é numérico de 8 a 14 dígitos por acaso.
		if strings.Contains(a, "GTIN") {
			continue
		}
		return errorsuc.NewValidationError(fmt.Sprintf("o código %s %s: um código digitado precisa apontar para um item só. Use outro código", proprio, a))
	}
	return nil
}

// conferirValidadeComLote: validade e vida útil mínima são do lote. Sem controle
// de lote não há lote para receber a data, e a validade ficava gravada sem efeito.
func (r *RepositoryItemSQLC) conferirValidadeComLote(ctx context.Context, it *entity.Item) error {
	if r.pool == nil || (it.Warehouse.ShelfLifeDays == nil && it.Warehouse.MinRemainingLifeDays == nil) {
		return nil
	}
	var controla bool
	err := r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT c.controls_lot FROM manufacturing_stock_item_controls c
		  JOIN items i ON i.code = c.item_code AND i.enterprise_id = c.enterprise_id
		 WHERE c.enterprise_id = $1 AND i.business_code = $2), FALSE)`, it.EnterpriseID, string(it.BusinessCode)).Scan(&controla)
	if err != nil {
		return fmt.Errorf("conferindo o controle de lote do item: %w", err)
	}
	if !controla {
		return errorsuc.NewValidationError("validade e vida útil mínima são do lote: ligue o controle de lote do item (Estoque e compras) antes de informá-las")
	}
	return nil
}

package third_party_service

import (
	"context"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscaluc "github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	domain "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var (
	_ fiscaluc.LeitorDaRemessa  = (*Repo)(nil)
	_ fiscaluc.EstornoDaRemessa = (*Repo)(nil)
)

// RemessaDaOS lê o que a NF-e de remessa precisa da OS de terceiro.
func (r *Repo) RemessaDaOS(ctx context.Context, osID int64, movimento *int64) (*fiscaluc.RemessaParaIndustrializar, error) {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var x fiscaluc.RemessaParaIndustrializar
	var fornecedor *int64
	var status string
	var ambiguo bool
	err = r.db.QueryRow(ctx, `
		SELECT o.id, o.code, COALESCE(rem.item_code, o.item_code), COALESCE(rem.mask, o.mask), COALESCE(i.name,''),
		       COALESCE((SELECT fc.ncm FROM fiscal_classifications fc
		                  WHERE fc.enterprise_id = i.enterprise_id
		                    AND fc.code::text = COALESCE(NULLIF(i.accounting_sale_fiscal_classification_code,''), i.accounting_purchase_fiscal_classification_code)
		                  LIMIT 1), ''),
		       COALESCE(i.warehouse_unit_of_measurement::text, o.uom), o.supplier_code, o.status, o.due_date,
		       COALESCE((SELECT b.avg_cost FROM stock_balances b WHERE b.enterprise_id = o.enterprise_id AND b.item_code = COALESCE(rem.item_code, o.item_code)
		                  AND b.avg_cost > 0 ORDER BY b.quantity DESC LIMIT 1), 0),
		       `+remAmbiguoSQL+`
		  FROM third_party_service_orders o`+itemRemetidoSQL+`
		  LEFT JOIN items i ON i.code = COALESCE(rem.item_code, o.item_code) AND i.enterprise_id = o.enterprise_id
		 WHERE o.id = $1 AND o.enterprise_id = $2`, osID, eid).
		Scan(&x.OSID, &x.OSCode, &x.ItemCode, &x.Mask, &x.Descricao, &x.NCM, &x.Unidade, &fornecedor, &status, &x.PrevisaoRetorno, &x.CustoMedio, &ambiguo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("ordem de serviço de terceiro não encontrada")
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a OS de terceiro: %w", err)
	}
	if status == "CANCELLED" {
		return nil, errorsuc.NewValidationError("a OS de terceiro está cancelada")
	}
	if ambiguo {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a OF da OS de terceiro %d tem vários materiais e nenhum ligado à etapa externa: informe na estrutura do item (etapa do componente) qual peça vai ao terceiro", x.OSCode))
	}
	if fornecedor == nil {
		return nil, errorsuc.NewValidationError("informe o fornecedor da OS antes de emitir a remessa")
	}
	x.Fornecedor = *fornecedor
	if movimento != nil {
		var qtd decimal.Decimal
		err := r.db.QueryRow(ctx, `SELECT m.quantity,
			       EXISTS (SELECT 1 FROM fiscal_exits x WHERE x.id = m.fiscal_exit_id AND x.status NOT IN ('CANCELLED','DENIED','REJECTED'))
			  FROM third_party_service_movements m
			 WHERE m.id=$1 AND m.service_order_id=$2 AND m.enterprise_id=$3 AND m.movement_type='REMITTANCE'`, *movimento, osID, eid).Scan(&qtd, &x.RemessaJaTemNota)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewValidationError("a remessa informada não é desta OS")
		}
		if err != nil {
			return nil, fmt.Errorf("conferindo a remessa da OS: %w", err)
		}
		x.QuantidadeDaRemessa = &qtd
	}
	return &x, nil
}

// FornecedorDestinatario lê o fornecedor e o endereço principal dele.
func (r *Repo) FornecedorDestinatario(ctx context.Context, fornecedor int64) (*fiscaluc.DestinatarioDaRemessa, error) {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var d fiscaluc.DestinatarioDaRemessa
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(s.document_number,''), COALESCE(s.name,''), COALESCE(s.state_registration,''),
		       COALESCE(a.uf,''), COALESCE(a.street,''), COALESCE(a.number,''), COALESCE(a.complement,''),
		       COALESCE(a.neighborhood,''), COALESCE(a.city,''), COALESCE(a.zip_code,'')
		  FROM suppliers s
		  LEFT JOIN LATERAL (SELECT * FROM supplier_addresses x WHERE x.supplier_id = s.id
		                      ORDER BY x.is_default DESC, x.id LIMIT 1) a ON TRUE
		 WHERE s.code = $1 AND s.enterprise_id = $2`, fornecedor, eid).
		Scan(&d.CNPJ, &d.RazaoSocial, &d.IE, &d.UF, &d.Logradouro, &d.Numero, &d.Complemento, &d.Bairro, &d.Municipio, &d.CEP)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("fornecedor %d não encontrado nesta empresa", fornecedor))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o fornecedor da remessa: %w", err)
	}
	var falta []string
	for _, c := range []struct{ nome, v string }{{"CNPJ", d.CNPJ}, {"UF", d.UF}, {"logradouro", d.Logradouro}, {"número", d.Numero},
		{"bairro", d.Bairro}, {"município", d.Municipio}, {"CEP", d.CEP}} {
		if strings.TrimSpace(c.v) == "" {
			falta = append(falta, c.nome)
		}
	}
	if len(falta) > 0 {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o cadastro do fornecedor %d está incompleto para a NF-e (%s): complete-o em Fornecedores", fornecedor, strings.Join(falta, ", ")))
	}
	return &d, nil
}

// LigarNotaAoMovimento grava a NF-e na remessa da OS.
func (r *Repo) LigarNotaAoMovimento(ctx context.Context, osID, movimentoID, exitID int64) error {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE third_party_service_movements SET fiscal_exit_id = $4
		WHERE id = $1 AND service_order_id = $2 AND enterprise_id = $3 AND movement_type = 'REMITTANCE'`, movimentoID, osID, eid, exitID)
	if err != nil {
		return fmt.Errorf("ligando a NF-e à remessa: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("a remessa informada não é desta OS")
	}
	return nil
}

// EstornarRemessasDaNota: a NF-e de remessa foi cancelada — a peça não saiu.
// Registra o RETURN de cada remessa ligada à nota (chave fixa: repetir não
// duplica), o que traz o estoque de volta do almoxarifado do terceiro.
func (r *Repo) EstornarRemessasDaNota(ctx context.Context, exitID int64) error {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	rows, err := r.db.Query(ctx, `SELECT id, service_order_id, quantity, warehouse_id, COALESCE(lot,''), created_by
		FROM third_party_service_movements WHERE enterprise_id = $1 AND fiscal_exit_id = $2 AND movement_type = 'REMITTANCE'`, eid, exitID)
	if err != nil {
		return fmt.Errorf("lendo as remessas da nota cancelada: %w", err)
	}
	var lista []domain.Movement
	for rows.Next() {
		var m domain.Movement
		if err := rows.Scan(&m.ID, &m.ServiceOrderID, &m.Quantity, &m.WarehouseID, &m.Lot, &m.CreatedBy); err != nil {
			rows.Close()
			return err
		}
		lista = append(lista, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, m := range lista {
		if _, err := r.AddMovement(ctx, m.ServiceOrderID, domain.Movement{MovementType: "RETURN", Quantity: m.Quantity, OccurredAt: time.Now(),
			ReferenceType: "NF_SAIDA_CANCELADA", ReferenceCode: fmt.Sprint(exitID), Notes: "NF-e de remessa cancelada: a peça não saiu",
			IdempotencyKey: fmt.Sprintf("estorno-remessa-%d-%d", exitID, m.ID), WarehouseID: m.WarehouseID, Lot: m.Lot, CreatedBy: m.CreatedBy}); err != nil {
			return fmt.Errorf("devolvendo a remessa %d da OS: %w", m.ID, err)
		}
	}
	return nil
}

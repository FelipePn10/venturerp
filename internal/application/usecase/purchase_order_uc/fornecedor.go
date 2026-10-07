package purchase_order_uc

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// resolverFornecedorDoTenant confirma que o fornecedor do pedido é da empresa
// autenticada e está ativo, e devolve os padrões do cadastro dele.
//
// O cadastro de fornecedor é por empresa (suppliers.enterprise_id), mas
// suppliers.code é único no banco inteiro e a chave estrangeira do pedido não
// olha a empresa. Sem esta conferência, informar o código de um fornecedor de
// OUTRA empresa gravava o pedido: o leitor do documento (que filtra por
// empresa) devolvia fornecedor em branco, não havia e-mail para enviar, e o
// pedido ainda assim podia receber material.
//
// Só "não existe nesta empresa" vira recusa de validação. Uma falha de banco
// sobe como é: calar a diferença transformaria indisponibilidade em "cadastre
// o fornecedor".
func resolverFornecedorDoTenant(ctx context.Context, fonte ports.SupplierPurchasingDefaultsProvider, supplierCode, enterpriseCode int64) (*ports.SupplierPurchasingDefaults, error) {
	def, err := fonte.GetPurchasingDefaults(ctx, supplierCode, enterpriseCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("fornecedor %d não encontrado no cadastro desta empresa", supplierCode))
		}
		return nil, err
	}
	if def == nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("fornecedor %d não encontrado no cadastro desta empresa", supplierCode))
	}
	if !def.IsActive {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o fornecedor %d (%s) está inativo; reative o cadastro ou escolha outro", supplierCode, def.SupplierName))
	}
	return def, nil
}

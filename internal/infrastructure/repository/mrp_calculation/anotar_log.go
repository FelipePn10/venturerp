package mrp_calculation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// AnotarNoLog junta uma chave ao JSON de erros de um cálculo já encerrado, sem
// apagar o que o cálculo gravou. É assim que o aviso das estruturas sem
// conversão (apurado antes do cálculo) chega ao log guardado, que o impresso do
// plano de materiais lê.
func (r *MRPCalculationRepositorySQLC) AnotarNoLog(ctx context.Context, logCode int64, chave string, valor any) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	corpo, err := json.Marshal(valor)
	if err != nil {
		return fmt.Errorf("anotando o log do MRP: %w", err)
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE mrp_calculation_logs
		   SET errors = COALESCE(errors, '{}'::jsonb) || jsonb_build_object($3::text, $4::jsonb)
		 WHERE code = $1 AND enterprise_id = $2`, logCode, enterpriseID, chave, string(corpo))
	if err != nil {
		return fmt.Errorf("anotando o log do MRP: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("anotando o log do MRP: cálculo %d não encontrado nesta empresa", logCode)
	}
	return nil
}

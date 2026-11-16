package handler

import (
	"context"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type setCustomerPO interface {
	Execute(ctx context.Context, code int64, po *string) error
}

// SalesOrderCustomerPO atende PATCH /api/sales-order/{code}/customer-po
// {"customer_po_number": "..."}.
func SalesOrderCustomerPO(uc setCustomerPO) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code, ok := pathID(w, r, "code")
		var body struct {
			CustomerPONumber *string `json:"customer_po_number"`
		}
		if !ok || !security.DecodeBody(w, r, &body) {
			return
		}
		noContent(w, uc.Execute(r.Context(), code, body.CustomerPONumber))
	}
}

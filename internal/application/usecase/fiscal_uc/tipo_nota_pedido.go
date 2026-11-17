package fiscal_uc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
)

// linhaTipoNota: uma linha do pedido que vai para a nota e o tipo de NF dela.
type linhaTipoNota struct {
	rotulo string
	tipo   *string
}

// resolverTipoNota: o tipo de NF da nota quando ninguém escolheu um. Vale o
// tipo das linhas (todas com o mesmo); linha sem tipo segue o "Tipo de NF
// padrão" do pedido. Uma nota tem um tipo só: linhas de tipos diferentes vão em
// notas separadas. Os tipos são códigos gravados como texto; tipo que não existe
// ou está inativo devolve o motivo, para o usuário corrigir.
func resolverTipoNota(ctx context.Context, clientes customerrepo.CustomerRepository, padraoDoPedido *string, linhas []linhaTipoNota) (*int64, string, error) {
	padrao := ""
	if padraoDoPedido != nil {
		padrao = strings.TrimSpace(*padraoDoPedido)
	}
	tipos := map[string][]string{}
	var ordem []string
	for _, l := range linhas {
		t := padrao
		if l.tipo != nil && strings.TrimSpace(*l.tipo) != "" {
			t = strings.TrimSpace(*l.tipo)
		}
		if _, ok := tipos[t]; !ok {
			ordem = append(ordem, t)
		}
		tipos[t] = append(tipos[t], l.rotulo)
	}
	if len(ordem) > 1 {
		partes := make([]string, 0, len(ordem))
		for _, t := range ordem {
			rotulo := "tipo " + t
			if t == "" {
				rotulo = "sem tipo"
			}
			partes = append(partes, fmt.Sprintf("%s (itens %s)", rotulo, strings.Join(tipos[t], ", ")))
		}
		return nil, "as linhas têm tipos de NF diferentes — " + strings.Join(partes, "; ") + ": fature cada tipo numa nota ou escolha o tipo no faturamento", nil
	}
	codigoTexto := padrao
	if len(ordem) == 1 {
		codigoTexto = ordem[0]
	}
	if codigoTexto == "" || clientes == nil {
		return nil, "", nil
	}
	codigo, err := strconv.ParseInt(codigoTexto, 10, 64)
	if err != nil {
		return nil, fmt.Sprintf("o tipo de NF %q do pedido não é um código válido", codigoTexto), nil
	}
	tipo, err := clientes.GetInvoiceTypeByCode(ctx, codigo)
	if err != nil && !errorsuc.EhNaoEncontrado(err) {
		return nil, "", fmt.Errorf("lendo o tipo de NF do pedido: %w", err)
	}
	if err != nil || tipo == nil {
		return nil, fmt.Sprintf("o tipo de NF %d do pedido não existe nesta empresa", codigo), nil
	}
	if !tipo.IsActive {
		return nil, fmt.Sprintf("o tipo de NF %d do pedido está inativo", codigo), nil
	}
	return &tipo.ID, "", nil
}

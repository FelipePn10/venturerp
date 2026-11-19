package entity

// IdentificacaoOF: o que faz uma OF ser reconhecida sem decorar o número —
// item, cliente, projeto, pedido ou demanda de origem, a OF pai que ela
// abastece e o produto final da cadeia (raiz). Só leitura.
type IdentificacaoOF struct {
	OrderID          int64
	ItemBusinessCode string
	ItemName         string
	CustomerCode     *int64
	CustomerName     string
	ProjectID        *int64
	ProjectCode      string
	ProjectName      string
	// Pedidos de venda e demandas independentes da cadeia (vêm da raiz).
	SalesOrders []int64
	Demands     []int64
	// OF pai (ou a ordem planejada pai, quando ainda não virou OF).
	ParentOrderID      *int64
	ParentPlannedCode  *int64
	ParentItemBusiness string
	ParentOrderNumber  *int64
	// Raiz da cadeia: a OF do produto final (a própria OF quando não tem pai).
	RootOrderID      *int64
	RootItemBusiness string
	RootOrderNumber  *int64
	// Nível na cadeia: 0 = produto final.
	Level int
	// Quantas OFs filhas (componentes fabricados) ela tem.
	Children int
}

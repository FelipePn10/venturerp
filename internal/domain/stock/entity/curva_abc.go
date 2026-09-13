package entity

// ClassificarABC decide a classe pelo acumulado de Pareto.
//
// A comparação é com o acumulado ANTES do próprio item (`acumulado` menos a
// `participacao` dele). Comparar com o acumulado já incluindo o item joga para a
// classe seguinte justamente quem cruza o corte — no limite, um item que sozinho
// responde por 100% do consumo virava C, que é o oposto do que a curva quer
// dizer. É também a prática corrente: o item que atravessa a linha dos 80% ainda
// é A.
func ClassificarABC(acumuladoPct, participacaoPct, corteA, corteB float64) string {
	anterior := acumuladoPct - participacaoPct
	switch {
	case anterior < corteA:
		return "A"
	case anterior < corteB:
		return "B"
	default:
		return "C"
	}
}

package amostragem

// Estado da severidade de uma unidade de controle (item × fornecedor no
// recebimento, item × operação no processo). Atualizado a cada lote em
// inspeção original — lote reinspecionado não conta (9.3.1).
type Estado struct {
	Severidade Severidade `json:"severidade"`
	// Pontuacao de troca para atenuada (9.3.3.2); só corre em normal.
	Pontuacao int `json:"pontuacao"`
	// Ultimos guarda os resultados dos últimos lotes (true = aceito) na
	// severidade atual, no máximo 5: base de 9.3.1 e 9.3.2.
	Ultimos []bool `json:"ultimos"`
	// RecusasEmApertada acumula as recusas desde a entrada em apertada (9.4).
	RecusasEmApertada int `json:"recusas_em_apertada"`
	// Suspensa: 5 recusas acumuladas em apertada. A inspeção pela norma para
	// até a ação do fornecedor ser aceita; a liberação volta em apertada.
	Suspensa bool `json:"suspensa"`
	// AceitosSeguidos e LotesDesdeInspecao alimentam o skip-lot.
	AceitosSeguidos    int `json:"aceitos_seguidos"`
	LotesDesdeInspecao int `json:"lotes_desde_inspecao"`
}

// Novo estado: a inspeção começa em normal (9.1).
func Novo() Estado { return Estado{Severidade: Normal} }

// Resultado de um lote em inspeção original.
type Resultado struct {
	Aceito bool
	// Ac do plano usado e se o lote seria aceito com o NQA um passo mais
	// exigente (pontuação quando Ac ≥ 2).
	Ac                       int
	AceitoComNQAMaisApertado bool
}

// Regras de quem configura o procedimento.
type Regras struct {
	// PermiteAtenuada é a "autoridade responsável" de 9.3.3 c): sem ela a
	// pontuação corre mas a troca para atenuada não acontece.
	PermiteAtenuada bool
	// ProducaoRegular é 9.3.3 b) e 9.3.4 b); falso impede a atenuada e devolve
	// a atenuada para normal.
	ProducaoRegular bool
}

// Transicao descreve a troca feita pelo lote, para o histórico e a tela.
type Transicao struct {
	De, Para  Severidade
	Suspensao bool
	// Liberacao: a suspensão foi levantada por quem tem alçada (9.4).
	Liberacao bool
	Motivo    string
}

// Registrar aplica as regras de troca da cláusula 9 ao resultado do lote.
func (e Estado) Registrar(r Resultado, regras Regras) (Estado, *Transicao) {
	if e.Severidade == "" {
		e = Novo()
	}
	de := e.Severidade
	e.Ultimos = append(e.Ultimos, r.Aceito)
	if len(e.Ultimos) > 5 {
		e.Ultimos = e.Ultimos[len(e.Ultimos)-5:]
	}
	if r.Aceito {
		e.AceitosSeguidos++
	} else {
		e.AceitosSeguidos = 0
	}
	e.LotesDesdeInspecao = 0

	trocar := func(para Severidade, motivo string) (Estado, *Transicao) {
		e.Severidade, e.Ultimos, e.Pontuacao = para, nil, 0
		if para != Apertada {
			e.RecusasEmApertada = 0
		}
		return e, &Transicao{De: de, Para: para, Motivo: motivo}
	}

	switch e.Severidade {
	case Normal:
		recusas := 0
		for _, ok := range e.Ultimos {
			if !ok {
				recusas++
			}
		}
		if recusas >= 2 {
			e.RecusasEmApertada = 0
			return trocar(Apertada, "2 de até 5 lotes consecutivos recusados (9.3.1)")
		}
		switch {
		case r.Ac >= 2 && r.AceitoComNQAMaisApertado, r.Ac < 2 && r.Aceito:
			if r.Ac >= 2 {
				e.Pontuacao += 3
			} else {
				e.Pontuacao += 2
			}
		default:
			e.Pontuacao = 0
		}
		if e.Pontuacao >= 30 && regras.PermiteAtenuada && regras.ProducaoRegular {
			return trocar(Atenuada, "pontuação de troca ≥ 30, produção regular e atenuada autorizada (9.3.3)")
		}
	case Apertada:
		if !r.Aceito {
			e.RecusasEmApertada++
			if e.RecusasEmApertada >= 5 {
				e.Suspensa = true
				return e, &Transicao{De: de, Para: Apertada, Suspensao: true,
					Motivo: "5 lotes recusados acumulados em apertada: inspeção suspensa até ação do fornecedor (9.4)"}
			}
		}
		if len(e.Ultimos) == 5 && todosAceitos(e.Ultimos) {
			return trocar(Normal, "5 lotes consecutivos aceitos em apertada (9.3.2)")
		}
	case Atenuada:
		if !r.Aceito {
			return trocar(Normal, "lote recusado em atenuada (9.3.4 a)")
		}
		if !regras.ProducaoRegular {
			return trocar(Normal, "produção irregular (9.3.4 b)")
		}
	}
	return e, nil
}

// Liberar tira a suspensão depois que a ação do fornecedor foi aceita; a
// inspeção recomeça em apertada (9.4).
func (e Estado) Liberar() Estado {
	e.Suspensa = false
	e.Severidade = Apertada
	e.Ultimos = nil
	e.RecusasEmApertada = 0
	e.Pontuacao = 0
	return e
}

func todosAceitos(xs []bool) bool {
	for _, x := range xs {
		if !x {
			return false
		}
	}
	return true
}

// SkipLot é regra da empresa, não da ISO 2859-1 (o skip-lot normativo é a
// ISO 2859-3): depois de AposAceitos lotes aceitos seguidos, só 1 lote a cada
// Frequencia é inspecionado; os outros são liberados sem amostra. Uma recusa
// zera a sequência e volta à inspeção de todo lote.
type SkipLot struct {
	AposAceitos int `json:"apos_aceitos"`
	Frequencia  int `json:"frequencia"`
}

// Ativo diz se a regra está configurada.
func (s SkipLot) Ativo() bool { return s.AposAceitos > 0 && s.Frequencia > 1 }

// DeveInspecionar decide se o próximo lote é inspecionado. Lote suspenso
// sempre exige decisão; apertada nunca pula.
func (e Estado) DeveInspecionar(s SkipLot) bool {
	if !s.Ativo() || e.Suspensa || e.Severidade == Apertada {
		return true
	}
	if e.AceitosSeguidos < s.AposAceitos {
		return true
	}
	return e.LotesDesdeInspecao+1 >= s.Frequencia
}

// Pular registra um lote liberado pelo skip-lot.
func (e Estado) Pular() Estado {
	e.LotesDesdeInspecao++
	return e
}

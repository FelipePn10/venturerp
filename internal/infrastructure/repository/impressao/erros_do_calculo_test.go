package impressao

import "testing"

func TestErrosDoCalculoNoImpresso(t *testing.T) {
	erros, avisos := errosDoCalculo(`{"estrutura_sem_conversao":["estrutura do item RN-1: tubo em M, linha em KG","estrutura do item RN-2: chapa"],"itens_sem_lead_time":["TP-1","TP-2"],"interrompido":"o serviço caiu"}`)
	if len(avisos) != 2 || avisos[0] != "estrutura do item RN-1: tubo em M, linha em KG" {
		t.Fatalf("avisos: %v", avisos)
	}
	if erros != "interrompido: o serviço caiu · itens sem lead time: TP-1; TP-2" {
		t.Fatalf("erros: %q", erros)
	}
	for _, vazio := range []string{"", "{}", "null"} {
		if e, a := errosDoCalculo(vazio); e != "" || a != nil {
			t.Fatalf("%q virou %q/%v", vazio, e, a)
		}
	}
	if e, a := errosDoCalculo(`{"estrutura_sem_conversao":["x"]}`); e != "" || len(a) != 1 {
		t.Fatalf("só aviso: %q/%v", e, a)
	}
}

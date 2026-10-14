package datetime

import "testing"

// Data vazia é "sem data"; data preenchida e inválida precisa virar mensagem,
// não sumir em silêncio.
func TestPrimeiraInvalida(t *testing.T) {
	vazio, ok, rfc, br, ruim := " ", "2026-10-13", "2026-10-13T10:00:00Z", "13/10/2026", "31/02/2026"
	if msg := PrimeiraInvalida(Campo("a", nil), Campo("b", &vazio), Campo("c", &ok), Campo("d", &rfc), Campo("e", &br)); msg != "" {
		t.Fatalf("datas válidas recusadas: %s", msg)
	}
	if msg := PrimeiraInvalida(Campo("data de entrega", &ok), Campo("validade", &ruim)); msg != `validade inválida ("31/02/2026"): use AAAA-MM-DD ou DD/MM/AAAA` {
		t.Fatalf("mensagem = %q", msg)
	}
}

func TestObrigatoriaOpcionalHora(t *testing.T) {
	if _, msg := Obrigatoria("data de vencimento", " "); msg != "informe a data de vencimento" {
		t.Fatalf("vazia: %q", msg)
	}
	if _, msg := Obrigatoria("data de vencimento", "31/02/2026"); msg == "" {
		t.Fatal("31/02 deveria ser recusada")
	}
	if d, msg := Obrigatoria("data", "13/10/2026"); msg != "" || d.Day() != 13 || d.Month() != 10 {
		t.Fatalf("DD/MM/AAAA: %v %q", d, msg)
	}
	if d, msg := Opcional("validade", nil); d != nil || msg != "" {
		t.Fatal("nil deveria ser vazio")
	}
	v := "2026-13-01"
	if _, msg := Opcional("validade", &v); msg == "" {
		t.Fatal("mês 13 deveria ser recusado")
	}
	h := "07:30"
	if r, msg := Hora("início", &h); msg != "" || r.Hour() != 7 || r.Minute() != 30 {
		t.Fatalf("HH:MM: %v %q", r, msg)
	}
	h = "25:00"
	if _, msg := Hora("início", &h); msg == "" {
		t.Fatal("25:00 deveria ser recusado")
	}
}

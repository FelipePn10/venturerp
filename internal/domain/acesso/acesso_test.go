package acesso

import "testing"

func TestNovasPermissoesValidaEImplicaLer(t *testing.T) {
	p, err := NovasPermissoes([]Permissao{{Fiscal, Autorizar}})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Pode(Fiscal, Autorizar) || !p.Pode(Fiscal, Ler) {
		t.Fatalf("quem autoriza também consulta: %+v", p)
	}
	if _, err := NovasPermissoes([]Permissao{{"inexistente", Ler}}); err == nil {
		t.Fatal("módulo fora do catálogo")
	}
	if _, err := NovasPermissoes([]Permissao{{Estoque, Autorizar}}); err == nil {
		t.Fatal("estoque não tem a ação AUTORIZAR")
	}
	perfil := &Perfil{Nome: "  "}
	if perfil.Validar() == nil {
		t.Fatal("nome em branco")
	}
	sem := AcessoDoUsuario{}
	if !sem.Pode(Financeiro, Excluir) {
		t.Fatal("sem perfil vale o papel")
	}
}

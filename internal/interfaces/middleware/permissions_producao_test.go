package middleware

import "testing"

// O apontamento de chão de fábrica deixou de depender da permissão de
// planejamento. Quem já operava continua operando; o perfil de posto passa a
// existir sem ganhar nada além do que aponta.
func TestPermissaoDeApontamentoSeparaChaoDeFabricaDePlanejamento(t *testing.T) {
	for _, perfil := range []string{"ADMIN", "USER"} {
		if !RoleHasPermission(perfil, PermProductionReport) {
			t.Fatalf("%s perdeu o apontamento de produção", perfil)
		}
	}
	if !RoleHasPermission("OPERATOR", PermProductionReport) {
		t.Fatal("OPERATOR precisa apontar produção — é a única razão do perfil existir")
	}
	if !RoleHasPermission("OPERATOR", PermRead) {
		t.Fatal("OPERATOR precisa ler a ordem que vai apontar")
	}
	for _, escopo := range []string{PermWrite, PermPlanningRun, PermPurchaseApprove, PermFiscalAuthorize, PermFinancialManage, PermItemActivate, PermAdmin} {
		if RoleHasPermission("OPERATOR", escopo) {
			t.Fatalf("OPERATOR não pode receber %q junto com o apontamento", escopo)
		}
	}
	if RoleHasPermission("VIEWER", PermProductionReport) {
		t.Fatal("VIEWER é somente leitura")
	}
	if RoleHasPermission("DESCONHECIDO", PermProductionReport) {
		t.Fatal("perfil desconhecido não recebe escopo algum")
	}
}

// O papel aceito pelo JWT sai do MESMO mapa de permissões.
//
// Eram duas listas independentes: o mapa ganhou OPERATOR e o middleware
// continuou com `role != "ADMIN" && role != "USER"` escrito à mão. O perfil
// existia, tinha escopo, e tomava 401 antes de chegar a qualquer rota — falha
// que nenhum teste de permissão pegaria, porque a permissão estava certa.
func TestPerfilAceitoPeloTokenSaiDoMapaDePermissoes(t *testing.T) {
	for _, perfil := range []string{"ADMIN", "USER", "OPERATOR", "VIEWER"} {
		if !perfilConhecido(perfil) {
			t.Fatalf("%s está no mapa de permissões mas o token o recusaria", perfil)
		}
	}
	for _, desconhecido := range []string{"", "ROOT", "admin", "OPERADOR"} {
		if perfilConhecido(desconhecido) {
			t.Fatalf("perfil %q não deveria ser aceito", desconhecido)
		}
	}
}

package portal_uc

import (
	"context"
	"errors"
	"testing"
)

type repoPortalXML struct {
	Repo
	doCliente bool
	eventos   int
}

func (r *repoPortalXML) NotaDoCliente(context.Context, *Sessao, int64) (int64, bool, error) {
	return 987, r.doCliente, nil
}
func (r *repoPortalXML) Evento(context.Context, *Sessao, string, string, *int64, *int64, string, string) error {
	r.eventos++
	return nil
}

type xmlGuardadoFake struct{ lido bool }

func (x *xmlGuardadoFake) ExitXML(context.Context, int64) (string, string, error) {
	x.lido = true
	return "<nfeProc/>", "", nil
}

// O cliente só baixa o XML guardado da própria nota; a de outro cliente nem é lida.
func TestXMLDaNota_SoDaNotaDoCliente(t *testing.T) {
	guardado := &xmlGuardadoFake{}
	alheio := &UseCase{Repo: &repoPortalXML{}, XMLGuardado: guardado}
	if _, _, err := alheio.XMLDaNota(context.Background(), &Sessao{}, 1, "", ""); !errors.Is(err, ErrNaoEncontrado) || guardado.lido {
		t.Fatalf("nota de outro cliente: err=%v lido=%v", err, guardado.lido)
	}
	repo := &repoPortalXML{doCliente: true}
	xml, nome, err := (&UseCase{Repo: repo, XMLGuardado: guardado}).XMLDaNota(context.Background(), &Sessao{}, 1, "", "")
	if err != nil || xml != "<nfeProc/>" || nome != "NF-e 987.xml" || repo.eventos != 1 {
		t.Fatalf("xml=%q nome=%q err=%v eventos=%d", xml, nome, err, repo.eventos)
	}
}

package focusnfe

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestTrainingEnvironmentRejectsProductionFocusEndpoint(t *testing.T) {
	t.Setenv("DATA_ENVIRONMENT", "training")
	client := NewClient("token", "producao")
	if client.configErr == nil {
		t.Fatal("expected production Focus endpoint to be rejected in training")
	}
	if _, _, err := client.do(context.Background(), http.MethodGet, "/nfe/test", nil); err == nil {
		t.Fatal("expected request to fail before network access")
	}
}

func TestEmitirNFeStopsOnHTTPErrorWithoutPolling(t *testing.T) {
	t.Parallel()

	var postCalls, getCalls int
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case http.MethodPost:
			postCalls++
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`{"codigo":"permissao_negada","mensagem":"CNPJ do emitente não autorizado."}`)),
				Request:    r,
			}, nil
		case http.MethodGet:
			getCalls++
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found")), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusMethodNotAllowed, Body: io.NopCloser(strings.NewReader("method not allowed")), Request: r}, nil
		}
	})}

	client := NewClient("token-de-teste", "homologacao")
	client.baseURL = "https://focus.invalid/v2"
	client.httpCli = httpClient

	_, err := client.EmitirNFe(context.Background(), "ref-403", NFEPayload{})
	if err == nil {
		t.Fatal("EmitirNFe() error = nil, want HTTP error")
	}
	if !strings.Contains(err.Error(), "HTTP 403") || !strings.Contains(err.Error(), "permissao_negada") {
		t.Fatalf("EmitirNFe() error = %q, want status and Focus error code", err)
	}
	if postCalls != 1 {
		t.Fatalf("POST calls = %d, want 1", postCalls)
	}
	if getCalls != 0 {
		t.Fatalf("GET polling calls = %d, want 0 after terminal HTTP error", getCalls)
	}
}

// A SEFAZ que demora não é rejeição: a nota fica aguardando e a consulta
// conclui. Rejeição da SEFAZ volta tipada, com o motivo.
func TestEmitirNFeClassificaRejeicaoEProcessamento(t *testing.T) {
	t.Parallel()
	resposta := func(body string) *http.Response {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(body))}
	}
	rejeita := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return resposta(`{"status":"erro_autorizacao","status_sefaz":"930","mensagem_sefaz":"CST com beneficio fiscal e nao informado o codigo"}`), nil
	})}
	c := NewClient("t", "homologacao")
	c.baseURL, c.httpCli = "https://focus.invalid/v2", rejeita
	_, err := c.EmitirNFe(context.Background(), "r1", NFEPayload{})
	var rej *RejeicaoError
	if !errors.As(err, &rej) || !strings.Contains(rej.Motivo, "930") {
		t.Fatalf("erro = %v, quer RejeicaoError com o código 930", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // a espera termina na hora: o que importa é a classificação
	processando := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return resposta(`{"status":"processando_autorizacao"}`), nil
	})}
	c2 := NewClient("t", "homologacao")
	c2.baseURL, c2.httpCli = "https://focus.invalid/v2", processando
	_, err = c2.EmitirNFe(ctx, "r2", NFEPayload{})
	var proc *EmProcessamentoError
	if !errors.As(err, &proc) {
		t.Fatalf("erro = %v, quer EmProcessamentoError (não é rejeição)", err)
	}
}

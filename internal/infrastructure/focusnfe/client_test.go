package focusnfe

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
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

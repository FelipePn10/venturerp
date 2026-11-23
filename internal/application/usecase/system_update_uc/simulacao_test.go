package system_update_uc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPedidoDeSimulacao(t *testing.T) {
	dir := t.TempDir()
	m := NewSimulacaoManager(dir, "venturerp_tecnofer", false)
	st, err := m.Pedir(context.Background(), "admin@x")
	if err != nil || st.State != StateQueued {
		t.Fatalf("%+v %v", st, err)
	}
	var p PedidoSimulacao
	data, _ := os.ReadFile(filepath.Join(dir, "request.json"))
	if err := json.Unmarshal(data, &p); err != nil || p.SourceDatabase != "venturerp_tecnofer" || p.RequestedBy != "admin@x" {
		t.Fatalf("pedido: %+v %v", p, err)
	}
	if _, err := m.Pedir(context.Background(), "x"); !errors.Is(err, ErrSimulacaoEmAndamento) {
		t.Fatalf("segundo pedido: %v", err)
	}
	if _, err := NewSimulacaoManager(t.TempDir(), "venturerp_training", true).Pedir(context.Background(), "x"); !errors.Is(err, ErrSimulacaoNoTreino) {
		t.Fatalf("pedido feito pelo treino: %v", err)
	}
	if _, err := NewSimulacaoManager("", "x", false).Pedir(context.Background(), "x"); !errors.Is(err, ErrSimulacaoIndisponivel) {
		t.Fatalf("sem diretório: %v", err)
	}
	if _, err := NewSimulacaoManager(t.TempDir(), "x; DROP", false).Pedir(context.Background(), "x"); err == nil {
		t.Fatal("nome de banco inválido deveria ser recusado")
	}
}

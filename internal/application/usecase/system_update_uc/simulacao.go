package system_update_uc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// Modo simulação (melhoria 2): um clique pede ao agente da VPS uma cópia fiel do
// banco desta empresa no banco de treino (instância DATA_ENVIRONMENT=training,
// que já recusa a SEFAZ de produção e marca os e-mails). A API não toca no banco
// de treino nem no Docker: grava o pedido; o agente root (systemd .path) faz a
// cópia com flock, saneia e sobe a API de treino — ver scripts/refresh-training.sh.

var (
	ErrSimulacaoEmAndamento  = errorsuc.NewConflictError("uma cópia para o treino já está em andamento")
	ErrSimulacaoIndisponivel = errorsuc.NewValidationError("o modo simulação não está configurado nesta instalação (SIMULATION_DIR)")
	ErrSimulacaoNoTreino     = errorsuc.NewValidationError("a cópia para o treino é pedida pelo ambiente de produção, não pelo próprio treino")
	nomeDeBanco              = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)
)

// PedidoSimulacao é o que o agente lê. Só o NOME do banco de origem: o agente
// confere contra a lista do host e usa as próprias credenciais.
type PedidoSimulacao struct {
	SourceDatabase string    `json:"source_database"`
	RequestedBy    string    `json:"requested_by"`
	RequestedAt    time.Time `json:"requested_at"`
}

// SimulacaoManager grava o pedido e lê a situação da cópia.
type SimulacaoManager struct {
	dir    string
	origem string
	treino bool
	now    func() time.Time
}

// NewSimulacaoManager: dir vazio desliga o recurso; origem é o nome do banco
// desta API; treino diz se esta API já é a de treino.
func NewSimulacaoManager(dir, origem string, treino bool) *SimulacaoManager {
	return &SimulacaoManager{dir: dir, origem: origem, treino: treino, now: time.Now}
}

// StatusSimulacao é a situação da última cópia.
type StatusSimulacao struct {
	State       State      `json:"state"`
	Progress    int        `json:"progress"`
	Message     string     `json:"message,omitempty"`
	RequestedAt *time.Time `json:"requested_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	// TrainingURL: onde entrar depois da cópia (o agente grava).
	TrainingURL string `json:"training_url,omitempty"`
}

func (m *SimulacaoManager) Status(context.Context) (StatusSimulacao, error) {
	if m.dir == "" {
		return StatusSimulacao{}, ErrSimulacaoIndisponivel
	}
	st := StatusSimulacao{State: StateIdle}
	data, err := os.ReadFile(filepath.Join(m.dir, "status.json"))
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return StatusSimulacao{}, fmt.Errorf("ler a situação da simulação: %w", err)
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return StatusSimulacao{}, fmt.Errorf("decodificar a situação da simulação: %w", err)
	}
	return st, nil
}

// Pedir grava o pedido da cópia. Repetir enquanto há um na fila devolve 409.
func (m *SimulacaoManager) Pedir(_ context.Context, quem string) (StatusSimulacao, error) {
	switch {
	case m.dir == "":
		return StatusSimulacao{}, ErrSimulacaoIndisponivel
	case m.treino:
		return StatusSimulacao{}, ErrSimulacaoNoTreino
	case !nomeDeBanco.MatchString(m.origem):
		return StatusSimulacao{}, errorsuc.NewValidationError(fmt.Sprintf("nome do banco de origem inválido: %q", m.origem))
	}
	if err := os.MkdirAll(m.dir, 0o750); err != nil {
		if isReadOnlyQueue(err) {
			return StatusSimulacao{}, ErrSimulacaoIndisponivel
		}
		return StatusSimulacao{}, fmt.Errorf("criar a fila da simulação: %w", err)
	}
	lockPath := filepath.Join(m.dir, "active.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return StatusSimulacao{}, ErrSimulacaoEmAndamento
	}
	if isReadOnlyQueue(err) {
		return StatusSimulacao{}, ErrSimulacaoIndisponivel
	}
	if err != nil {
		return StatusSimulacao{}, fmt.Errorf("criar a trava da simulação: %w", err)
	}
	_ = lock.Close()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(lockPath)
		}
	}()
	agora := m.now().UTC()
	st := StatusSimulacao{State: StateQueued, Message: "Cópia para o treino aguardando o agente da VPS", RequestedAt: &agora}
	if err := writeJSONAtomic(filepath.Join(m.dir, "status.json"), st); err != nil {
		return StatusSimulacao{}, err
	}
	if err := writeJSONAtomic(filepath.Join(m.dir, "request.json"), PedidoSimulacao{SourceDatabase: m.origem, RequestedBy: quem, RequestedAt: agora}); err != nil {
		return StatusSimulacao{}, err
	}
	ok = true
	return st, nil
}

package system_update_uc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	appversion "github.com/FelipePn10/panossoerp/internal/version"
)

var (
	ErrUpdateInProgress = errors.New("uma atualização já está em andamento")
	ErrInvalidVersion   = errors.New("versão inválida")
	ErrNoRelease        = errors.New("nenhuma versão publicada foi encontrada")
	semverPattern       = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`)
)

type State string

const (
	StateIdle       State = "idle"
	StateQueued     State = "queued"
	StateRunning    State = "running"
	StateSucceeded  State = "succeeded"
	StateFailed     State = "failed"
	StateRolledBack State = "rolled_back"
)

type Status struct {
	State           State      `json:"state"`
	CurrentVersion  string     `json:"current_version"`
	LatestVersion   string     `json:"latest_version,omitempty"`
	TargetVersion   string     `json:"target_version,omitempty"`
	UpdateAvailable bool       `json:"update_available"`
	Progress        int        `json:"progress"`
	Message         string     `json:"message,omitempty"`
	RequestedAt     *time.Time `json:"requested_at,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

type updateRequest struct {
	Version     string    `json:"version"`
	RequestedAt time.Time `json:"requested_at"`
}

type Manager struct {
	dir        string
	releaseURL string
	client     *http.Client
	now        func() time.Time
}

func NewManager(dir, releaseURL string, client *http.Client) *Manager {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &Manager{dir: dir, releaseURL: releaseURL, client: client, now: time.Now}
}

func (m *Manager) Status(ctx context.Context) (Status, error) {
	status := Status{State: StateIdle, CurrentVersion: appversion.Current().Version}
	data, err := os.ReadFile(filepath.Join(m.dir, "status.json"))
	if err == nil {
		if err := json.Unmarshal(data, &status); err != nil {
			return Status{}, fmt.Errorf("decodificar status da atualização: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Status{}, fmt.Errorf("ler status da atualização: %w", err)
	}
	status.CurrentVersion = appversion.Current().Version
	if status.State == StateQueued || status.State == StateRunning {
		return status, nil
	}
	latest, err := m.latestVersion(ctx)
	if err == nil {
		status.LatestVersion = latest
		status.UpdateAvailable = compareSemver(latest, status.CurrentVersion) > 0
	}
	return status, nil
}

func (m *Manager) Request(ctx context.Context, requestedVersion string) (Status, error) {
	version := normalizeVersion(requestedVersion)
	if version == "" {
		var err error
		version, err = m.latestVersion(ctx)
		if err != nil {
			return Status{}, err
		}
	}
	if !semverPattern.MatchString(version) {
		return Status{}, ErrInvalidVersion
	}
	current := appversion.Current().Version
	if semverPattern.MatchString(current) && compareSemver(version, current) <= 0 {
		return Status{}, fmt.Errorf("%w: a versão deve ser superior a %s", ErrInvalidVersion, current)
	}
	if err := os.MkdirAll(m.dir, 0o750); err != nil {
		return Status{}, fmt.Errorf("criar diretório de atualização: %w", err)
	}
	lockPath := filepath.Join(m.dir, "active.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return Status{}, ErrUpdateInProgress
	}
	if err != nil {
		return Status{}, fmt.Errorf("criar trava de atualização: %w", err)
	}
	_ = lock.Close()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(lockPath)
		}
	}()

	now := m.now().UTC()
	request := updateRequest{Version: version, RequestedAt: now}
	status := Status{State: StateQueued, CurrentVersion: current, TargetVersion: version, Progress: 0, Message: "Atualização aguardando o agente da VPS", RequestedAt: &now}
	if err := writeJSONAtomic(filepath.Join(m.dir, "status.json"), status); err != nil {
		return Status{}, err
	}
	if err := writeJSONAtomic(filepath.Join(m.dir, "request.json"), request); err != nil {
		return Status{}, err
	}
	cleanup = false
	return status, nil
}

func (m *Manager) latestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.releaseURL, nil)
	if err != nil {
		return "", fmt.Errorf("criar consulta de release: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "VentureERP-version-check")
	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("consultar release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: servidor respondeu HTTP %d", ErrNoRelease, resp.StatusCode)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decodificar release: %w", err)
	}
	version := normalizeVersion(release.TagName)
	if !semverPattern.MatchString(version) {
		return "", ErrNoRelease
	}
	return version, nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("codificar arquivo de atualização: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".update-*.tmp")
	if err != nil {
		return fmt.Errorf("criar arquivo temporário: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("publicar arquivo de atualização: %w", err)
	}
	return nil
}

func normalizeVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

func compareSemver(a, b string) int {
	parse := func(value string) [3]int {
		value = strings.SplitN(value, "-", 2)[0]
		parts := strings.Split(value, ".")
		var result [3]int
		for i := 0; i < len(parts) && i < 3; i++ {
			result[i], _ = strconv.Atoi(parts[i])
		}
		return result
	}
	av, bv := parse(a), parse(b)
	for i := range av {
		if av[i] < bv[i] {
			return -1
		}
		if av[i] > bv[i] {
			return 1
		}
	}
	return 0
}

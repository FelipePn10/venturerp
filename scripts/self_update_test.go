package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSelfUpdateSuccessAndRollback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell updater is Linux-only")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required")
	}
	for _, test := range []struct {
		name      string
		failRun   bool
		wantState string
	}{
		{name: "success", wantState: "succeeded"},
		{name: "migration failure restores backup", failRun: true, wantState: "rolled_back"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			bin := filepath.Join(root, "bin")
			updateDir := filepath.Join(root, "update")
			backupDir := filepath.Join(root, "backups")
			mustMkdir(t, bin)
			mustMkdir(t, updateDir)
			mustWrite(t, filepath.Join(updateDir, "request.json"), `{"version":"1.2.3","requested_at":"2026-07-15T12:00:00Z"}`, 0o600)
			mustWrite(t, filepath.Join(updateDir, "active.lock"), "", 0o600)
			mustWrite(t, filepath.Join(bin, "systemctl"), "#!/bin/sh\n[ \"$1\" = is-active ] && exit 1\nexit 0\n", 0o755)
			mustWrite(t, filepath.Join(bin, "curl"), "#!/bin/sh\nexit 0\n", 0o755)
			mustWrite(t, filepath.Join(bin, "docker"), dockerMock, 0o755)
			config := "IMAGE_REPOSITORY=ghcr.io/test/api\nCOMPOSE_FILE=" + filepath.Join(root, "compose.yml") + "\nAPI_ENV_FILE=" + filepath.Join(root, ".env") + "\nUPDATE_DIR=" + updateDir + "\nBACKUP_DIR=" + backupDir + "\nLOCK_FILE=" + filepath.Join(root, "update.lock") + "\nDATABASE_CONTAINER=db\nDATABASE_USER=user\nDATABASE_NAME=erp\nDATABASE_URL=postgres://test\nLEGACY_SERVICE=legacy.service\nHEALTH_ATTEMPTS=1\nHEALTH_INTERVAL_SECONDS=0\n"
			configPath := filepath.Join(root, "update.env")
			mustWrite(t, configPath, config, 0o600)

			command := exec.Command("bash", filepath.Join("self-update.sh"))
			command.Dir = "."
			command.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "VENTURERP_UPDATE_CONFIG="+configPath)
			if test.failRun {
				command.Env = append(command.Env, "MOCK_FAIL_MIGRATION=1")
			}
			err := command.Run()
			if test.failRun && err == nil {
				t.Fatal("self-update succeeded despite forced migration failure")
			}
			if !test.failRun && err != nil {
				t.Fatalf("self-update failed: %v", err)
			}
			data, readErr := os.ReadFile(filepath.Join(updateDir, "status.json"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			var status struct {
				State string `json:"state"`
			}
			if err := json.Unmarshal(data, &status); err != nil {
				t.Fatal(err)
			}
			if status.State != test.wantState {
				t.Fatalf("state = %q, want %q", status.State, test.wantState)
			}
		})
	}
}

const dockerMock = `#!/bin/sh
printf 'docker %s\n' "$*" >>"${MOCK_LOG:-/dev/null}"
if [ "$1" = compose ]; then
  printf 'composeenv profiles=%s main=%s usimac=%s\n' \
    "${COMPOSE_PROFILES:-}" "${VENTURERP_API_ENV:-}" "${VENTURERP_USIMAC_API_ENV:-}" >>"${MOCK_LOG:-/dev/null}"
fi
case "$1" in
  inspect|pull|rm|compose) exit 0 ;;
  create) echo mock-container; exit 0 ;;
  cp) mkdir -p "$3"; echo '-- migration' >"$3/000001_init.up.sql"; exit 0 ;;
  run) [ "${MOCK_FAIL_MIGRATION:-0}" = 1 ] && exit 42; exit 0 ;;
  exec)
    case "$*" in
      *pg_dump*) echo mock-backup ;;
      *) cat >/dev/null 2>&1 || true ;;
    esac
    exit 0 ;;
esac
exit 0
`

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

// TestSelfUpdateMultiTenant cobre o ciclo com duas empresas clientes na mesma
// VPS: cada base precisa de backup próprio, migration própria e health-check
// próprio, e uma falha precisa restaurar as duas — nunca só a principal.
func TestSelfUpdateMultiTenant(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell updater is Linux-only")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required")
	}
	for _, test := range []struct {
		name      string
		failRun   bool
		wantState string
	}{
		{name: "success", wantState: "succeeded"},
		{name: "failure restores every tenant", failRun: true, wantState: "rolled_back"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			bin := filepath.Join(root, "bin")
			updateDir := filepath.Join(root, "update")
			backupDir := filepath.Join(root, "backups")
			logPath := filepath.Join(root, "docker.log")
			mainEnv := filepath.Join(root, ".env")
			usimacEnv := filepath.Join(root, ".env.usimac")
			mustMkdir(t, bin)
			mustMkdir(t, updateDir)
			mustWrite(t, filepath.Join(updateDir, "request.json"), `{"version":"1.2.3","requested_at":"2026-07-15T12:00:00Z"}`, 0o600)
			mustWrite(t, filepath.Join(updateDir, "active.lock"), "", 0o600)
			mustWrite(t, mainEnv, "", 0o600)
			mustWrite(t, usimacEnv, "", 0o600)
			mustWrite(t, filepath.Join(bin, "systemctl"), "#!/bin/sh\n[ \"$1\" = is-active ] && exit 1\nexit 0\n", 0o755)
			mustWrite(t, filepath.Join(bin, "curl"), "#!/bin/sh\nexit 0\n", 0o755)
			mustWrite(t, filepath.Join(bin, "docker"), dockerMock, 0o755)

			config := "IMAGE_REPOSITORY=ghcr.io/test/api\n" +
				"COMPOSE_FILE=" + filepath.Join(root, "compose.yml") + "\n" +
				"API_ENV_FILE=" + mainEnv + "\n" +
				"UPDATE_DIR=" + updateDir + "\n" +
				"BACKUP_DIR=" + backupDir + "\n" +
				"LOCK_FILE=" + filepath.Join(root, "update.lock") + "\n" +
				"DATABASE_CONTAINER=db\nDATABASE_USER=user\nDATABASE_NAME=erp\n" +
				"DATABASE_URL=postgres://user:secret@127.0.0.1:5432/erp?sslmode=disable\n" +
				"VENTURERP_TENANTS=usimac\n" +
				"TENANT_USIMAC_DATABASE_URL=postgres://venturerp_usimac:outra@127.0.0.1:5432/venturerp_usimac?sslmode=disable\n" +
				"TENANT_USIMAC_API_ENV_FILE=" + usimacEnv + "\n" +
				"TENANT_USIMAC_HEALTH_URL=http://127.0.0.1:5075/health/ready\n" +
				"LEGACY_SERVICE=legacy.service\nHEALTH_ATTEMPTS=1\nHEALTH_INTERVAL_SECONDS=0\n"
			configPath := filepath.Join(root, "update.env")
			mustWrite(t, configPath, config, 0o600)

			command := exec.Command("bash", "self-update.sh")
			command.Dir = "."
			command.Env = append(os.Environ(),
				"PATH="+bin+":"+os.Getenv("PATH"),
				"VENTURERP_UPDATE_CONFIG="+configPath,
				"MOCK_LOG="+logPath,
			)
			if test.failRun {
				command.Env = append(command.Env, "MOCK_FAIL_MIGRATION=1")
			}
			err := command.Run()
			if test.failRun && err == nil {
				t.Fatal("self-update succeeded despite forced migration failure")
			}
			if !test.failRun && err != nil {
				t.Fatalf("self-update failed: %v", err)
			}

			status := readStatus(t, filepath.Join(updateDir, "status.json"))
			if status != test.wantState {
				t.Fatalf("state = %q, want %q", status, test.wantState)
			}

			logged := readFile(t, logPath)

			// Uma base sem backup é uma base que o rollback não sabe restaurar.
			assertLoggedCall(t, logged, "pg_dump -U user", "-d erp ")
			assertLoggedCall(t, logged, "pg_dump -U venturerp_usimac", "-d venturerp_usimac ")
			assertBackupExists(t, backupDir, "pre-update-1.2.3-")
			assertBackupExists(t, backupDir, "pre-update-usimac-1.2.3-")

			if !test.failRun {
				// Migration precisa rodar contra as duas bases, não só a principal.
				assertLoggedCall(t, logged, "migrate/migrate", "-database=postgres://user:secret@127.0.0.1:5432/erp?sslmode=disable")
				assertLoggedCall(t, logged, "migrate/migrate", "-database=postgres://venturerp_usimac:outra@127.0.0.1:5432/venturerp_usimac?sslmode=disable")
				// O contêiner da Usimac só sobe se o perfil e o env file chegarem ao compose.
				if !strings.Contains(logged, "composeenv profiles=usimac main="+mainEnv+" usimac="+usimacEnv) {
					t.Fatalf("compose não recebeu perfil/env da Usimac:\n%s", logged)
				}
				return
			}

			// No rollback, as duas bases precisam ser restauradas.
			assertLoggedCall(t, logged, "pg_restore -U user", "-d erp ")
			assertLoggedCall(t, logged, "pg_restore -U venturerp_usimac", "-d venturerp_usimac ")
		})
	}
}

// TestSelfUpdateRejectsTenantsSharingDatabase garante que um erro de
// copiar-e-colar no update.env pare o processo antes do primeiro backup: dois
// tenants na mesma base fariam o rollback restaurar o dump errado por cima de
// dados de produção.
func TestSelfUpdateRejectsTenantsSharingDatabase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell updater is Linux-only")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	updateDir := filepath.Join(root, "update")
	mainEnv := filepath.Join(root, ".env")
	mustMkdir(t, bin)
	mustMkdir(t, updateDir)
	mustWrite(t, filepath.Join(updateDir, "request.json"), `{"version":"1.2.3","requested_at":"2026-07-15T12:00:00Z"}`, 0o600)
	mustWrite(t, mainEnv, "", 0o600)
	mustWrite(t, filepath.Join(bin, "systemctl"), "#!/bin/sh\nexit 0\n", 0o755)
	mustWrite(t, filepath.Join(bin, "curl"), "#!/bin/sh\nexit 0\n", 0o755)
	mustWrite(t, filepath.Join(bin, "docker"), dockerMock, 0o755)

	config := "IMAGE_REPOSITORY=ghcr.io/test/api\n" +
		"COMPOSE_FILE=" + filepath.Join(root, "compose.yml") + "\n" +
		"API_ENV_FILE=" + mainEnv + "\n" +
		"UPDATE_DIR=" + updateDir + "\n" +
		"BACKUP_DIR=" + filepath.Join(root, "backups") + "\n" +
		"LOCK_FILE=" + filepath.Join(root, "update.lock") + "\n" +
		"DATABASE_CONTAINER=db\nDATABASE_USER=user\nDATABASE_NAME=erp\n" +
		"DATABASE_URL=postgres://user:secret@127.0.0.1:5432/erp?sslmode=disable\n" +
		"VENTURERP_TENANTS=usimac\n" +
		"TENANT_USIMAC_DATABASE_URL=postgres://venturerp_usimac:outra@127.0.0.1:5432/erp?sslmode=disable\n" +
		"TENANT_USIMAC_API_ENV_FILE=" + mainEnv + "\n" +
		"HEALTH_ATTEMPTS=1\nHEALTH_INTERVAL_SECONDS=0\n"
	configPath := filepath.Join(root, "update.env")
	mustWrite(t, configPath, config, 0o600)

	command := exec.Command("bash", "self-update.sh")
	command.Dir = "."
	command.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"VENTURERP_UPDATE_CONFIG="+configPath,
	)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("self-update aceitou dois tenants na mesma base")
	}
	if !strings.Contains(string(output), "apontam para a mesma base") {
		t.Fatalf("mensagem não explica o conflito: %s", output)
	}
	// O pedido precisa continuar na fila: nada foi feito, nada foi consumido.
	if _, statErr := os.Stat(filepath.Join(updateDir, "request.json")); statErr != nil {
		t.Fatalf("pedido foi consumido apesar da configuração inválida: %v", statErr)
	}
}

func readStatus(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var status struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	return status.State
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertBackupExists(t *testing.T, dir, prefix string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, prefix+"*.dump"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("nenhum backup com prefixo %q em %s", prefix, dir)
	}
}

// assertLoggedCall exige que UMA linha do log contenha todos os trechos. Checar
// os trechos soltos no log inteiro é o que torna a asserção vácua: "pg_restore"
// de uma base casa com "-d outra_base" de uma linha de pg_dump.
func assertLoggedCall(t *testing.T, logged string, parts ...string) {
	t.Helper()
	for _, line := range strings.Split(logged, "\n") {
		matched := true
		for _, part := range parts {
			if !strings.Contains(line, part) {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("nenhuma linha do log combina %q:\n%s", parts, logged)
}

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

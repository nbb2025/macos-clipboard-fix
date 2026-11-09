package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type fakeRunner struct {
	mu        sync.Mutex
	prefs     map[string]string
	pids      map[string]int
	clipboard string
	calls     int
}

func (f *fakeRunner) Run(_ context.Context, name string, args []string, input string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	switch filepath.Base(name) {
	case "defaults":
		if len(args) >= 3 && args[0] == "read" {
			v, ok := f.prefs[args[2]]
			if !ok {
				return "", errors.New("unset")
			}
			return v, nil
		}
		if len(args) >= 5 && args[0] == "write" {
			f.prefs[args[2]] = args[len(args)-1]
			return "", nil
		}
	case "pgrep":
		n := f.pids[args[len(args)-1]]
		if n == 0 {
			return "", errors.New("not running")
		}
		return strconv.Itoa(n), nil
	case "kill":
		pid, _ := strconv.Atoi(args[len(args)-1])
		for n, p := range f.pids {
			if p == pid {
				f.pids[n] = pid + 1
			}
		}
		return "", nil
	case "launchctl":
		return "", nil
	case "pbcopy":
		f.clipboard = input
		return "", nil
	case "pbpaste":
		return f.clipboard, nil
	}
	return "", errors.New("unexpected command: " + name)
}

func fixture(t *testing.T) (*Engine, *fakeRunner, string) {
	t.Helper()
	home := t.TempDir()
	e, err := New(home)
	if err != nil {
		t.Fatal(err)
	}
	dir := cacheDirectory(home)
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "12345678-1234-1234-1234-123456789ABC")
	if err = os.WriteFile(p, []byte("PRIVATE-CLIPBOARD-FIXTURE"), 0600); err != nil {
		t.Fatal(err)
	}
	f := &fakeRunner{prefs: map[string]string{"kLocalPasteboardBlobName": p}, pids: map[string]int{"useractivityd": 4000}}
	e.Runner = f
	return e, f, p
}

func TestBackupThenRestorePreservesNewerCache(t *testing.T) {
	e, _, original := fixture(t)
	ctx := context.Background()
	b, err := e.BackupCache(ctx, "send")
	if err != nil {
		t.Fatal(err)
	}
	if !b.Ready {
		t.Fatal("backup not committed")
	}
	if _, err = os.Stat(original); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("original should be moved")
	}
	if data, _ := os.ReadFile(b.Path); string(data) != "PRIVATE-CLIPBOARD-FIXTURE" {
		t.Fatal("original bytes not preserved")
	}
	if err = os.WriteFile(original, []byte("NEWER-CACHE-DATA"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = e.RestoreBackup(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(original); string(data) != "PRIVATE-CLIPBOARD-FIXTURE" {
		t.Fatal("old cache not restored")
	}
	var preserved bool
	for _, other := range e.Session().Backups {
		if other.ID != b.ID && other.Ready {
			data, _ := os.ReadFile(other.Path)
			preserved = string(data) == "NEWER-CACHE-DATA"
		}
	}
	if !preserved {
		t.Fatal("newer cache was not backed up before restoration")
	}
	reloaded, err := New(e.Home)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Session().Backups) != 2 || !reloaded.Session().Backups[0].Restored {
		t.Fatal("recovery journal not persisted")
	}
	data, err := e.Export()
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"PRIVATE-CLIPBOARD-FIXTURE", "NEWER-CACHE-DATA", e.Home} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("report leaked %q", secret)
		}
	}
}

func TestJournalFailureDoesNotMoveCache(t *testing.T) {
	e, _, original := fixture(t)
	bad := filepath.Join(t.TempDir(), "not-a-directory")
	_ = os.WriteFile(bad, []byte("x"), 0600)
	e.DataDir = bad
	if _, err := e.BackupCache(context.Background(), "send"); err == nil {
		t.Fatal("expected journal failure")
	}
	if data, _ := os.ReadFile(original); string(data) != "PRIVATE-CLIPBOARD-FIXTURE" {
		t.Fatal("source changed before recovery journal was saved")
	}
}

func TestRefusesOverwriteAndSymlinks(t *testing.T) {
	e, _, p := fixture(t)
	target := p + ".clipboard-fix-backup-20260101T000000-0123456789abcdef"
	if err := os.WriteFile(target, []byte("DO-NOT-OVERWRITE"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := e.moveCache(context.Background(), p, target); err == nil {
		t.Fatal("overwrote existing destination")
	}
	if data, _ := os.ReadFile(target); string(data) != "DO-NOT-OVERWRITE" {
		t.Fatal("existing destination changed")
	}
	link := filepath.Join(filepath.Dir(p), "ABCDEF12-1234-1234-1234-123456789ABC")
	if err := os.Symlink(p, link); err != nil {
		t.Fatal(err)
	}
	if _, err := regularFile(link); err == nil {
		t.Fatal("accepted symlink")
	}
	if err := renameExclusive(p, target); err == nil {
		t.Fatal("exclusive move overwrote existing destination")
	}
}

func TestRejectsPathsOutsideExactCacheDirectory(t *testing.T) {
	home := t.TempDir()
	good := filepath.Join(cacheDirectory(home), "12345678-1234-1234-1234-123456789ABC")
	if err := validateCachePath(home, good); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{filepath.Join(home, "Documents", "12345678-1234-1234-1234-123456789ABC"), good + "/../state", filepath.Join(cacheDirectory(home), "not-a-cache"), good + ".txt", "../" + good} {
		if validateCachePath(home, bad) == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	for _, bad := range []string{good + ".backup", filepath.Join(home, "elsewhere", filepath.Base(good)), good + ".clipboard-fix-backup-20260101T000000-../../a"} {
		if validateBackupPath(home, good, bad) == nil {
			t.Fatalf("accepted backup %q", bad)
		}
	}
}

func TestProbeOverwriteCannotBeMarkedSuccessful(t *testing.T) {
	e, f, _ := fixture(t)
	ctx := context.Background()
	p, err := e.CreateProbe(ctx)
	if err != nil || !p.Matches {
		t.Fatal(p, err)
	}
	if err = e.RecordVerification("mac-iphone", true); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.clipboard = "USER-SECRET-CHANGED"
	f.mu.Unlock()
	p, err = e.CheckProbe(ctx)
	if err != nil || p.Matches {
		t.Fatal("failed to detect overwritten probe")
	}
	if e.RecordVerification("mac-ipad", true) == nil {
		t.Fatal("recorded success for overwritten probe")
	}
	b, _ := e.Export()
	if strings.Contains(string(b), "USER-SECRET-CHANGED") {
		t.Fatal("leaked clipboard text")
	}
}

func TestPreparedJournalRecoveredAfterInterruptedCommit(t *testing.T) {
	e, _, p := fixture(t)
	b, err := e.BackupCache(context.Background(), "send")
	if err != nil {
		t.Fatal(err)
	}
	e.mu.Lock()
	e.backups[0].Ready = false
	err = e.persistLocked()
	e.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(e.Home)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Session().Backups[0].Ready {
		t.Fatal("did not recover completed file move")
	}
	if _, err = os.Stat(b.Path); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(p); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unexpected source")
	}
}

func TestArbitraryCommandsAndToolsAreRejected(t *testing.T) {
	e, f, _ := fixture(t)
	ctx := context.Background()
	for _, err := range []error{e.Repair(ctx, "quick; touch /tmp/injected"), e.Tool(ctx, "Maccy; rm -rf /", false), e.OpenSettings(ctx, "https://untrusted.invalid"), e.restart(ctx, "securityd")} {
		if err == nil {
			t.Fatal("accepted arbitrary operation")
		}
	}
	if f.calls != 0 {
		t.Fatal("rejected operation reached command runner")
	}
}

func TestDiagnosticParsersDoNotGuessUnknownStates(t *testing.T) {
	for text, want := range map[string]string{"Wi-Fi Power (en0): On": "ok", "Wi-Fi Power (en0): Off": "warning", "failed to get power": "unknown", "": "unknown"} {
		if got := parsePower(text); got != want {
			t.Fatal(text, got, want)
		}
	}
	if got := wifiInterface("Hardware Port: Wi-Fi\nDevice: en0\nEthernet Address: aa\n"); got != "en0" {
		t.Fatal(got)
	}
	if got := wifiInterface("Hardware Port: Wi-Fi\nDevice: en0;injected\n"); got != "" {
		t.Fatal("accepted unsafe interface")
	}
}

func TestOutputIsBoundedWithoutBreakingWriter(t *testing.T) {
	b := &cappedBuffer{limit: 16}
	n, err := b.Write([]byte(strings.Repeat("x", 1000)))
	if err != nil || n != 1000 || b.Len() != 16 {
		t.Fatal(n, err, b.Len())
	}
}

func TestReadOnlyDiagnosticCannotWriteOrRepair(t *testing.T) {
	home := t.TempDir()
	e := NewReadOnly(home)
	f := &fakeRunner{prefs: map[string]string{}, pids: map[string]int{}}
	e.Runner = f
	e.log("info", "read-only diagnostic")
	ctx := context.Background()
	if _, err := e.CreateProbe(ctx); !errors.Is(err, errReadOnly) {
		t.Fatal(err)
	}
	if _, err := e.BackupCache(ctx, "send"); !errors.Is(err, errReadOnly) {
		t.Fatal(err)
	}
	if err := e.RestoreBackup(ctx, "id"); !errors.Is(err, errReadOnly) {
		t.Fatal(err)
	}
	if err := e.Repair(ctx, "quick"); !errors.Is(err, errReadOnly) {
		t.Fatal(err)
	}
	if err := e.Tool(ctx, "Maccy", false); !errors.Is(err, errReadOnly) {
		t.Fatal(err)
	}
	if err := e.RecordVerification("iphone-mac", true); !errors.Is(err, errReadOnly) {
		t.Fatal(err)
	}
	if f.calls != 0 {
		t.Fatal("read-only mode reached a mutating command")
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatal("read-only mode created state files", err)
	}
}

func TestExportRedactsHomeWithJSONSpecialCharacters(t *testing.T) {
	home := filepath.Join(t.TempDir(), `user"quoted`)
	e, err := New(home)
	if err != nil {
		t.Fatal(err)
	}
	e.lastReport.Caches = []CacheState{{Path: filepath.Join(cacheDirectory(home), "example")}}
	data, err := e.Export()
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err = json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "quoted") || !strings.Contains(string(data), "~/Library") {
		t.Fatal("home path was not redacted")
	}
}

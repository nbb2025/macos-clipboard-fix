package core

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var cacheNamePattern = regexp.MustCompile(`^[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12}$`)
var backupSuffixPattern = regexp.MustCompile(`^\.clipboard-fix-backup-[0-9]{8}T[0-9]{6}-[a-f0-9]{16}$`)

func validateCachePath(home, path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Dir(path) != cacheDirectory(home) || !cacheNamePattern.MatchString(filepath.Base(path)) {
		return errors.New("拒绝访问系统剪贴板目录以外的文件")
	}
	return nil
}

func validateBackupPath(home, original, backup string) error {
	if err := validateCachePath(home, original); err != nil {
		return err
	}
	if filepath.Dir(backup) != cacheDirectory(home) || filepath.Clean(backup) != backup {
		return errors.New("备份路径无效")
	}
	suffix := strings.TrimPrefix(filepath.Base(backup), filepath.Base(original))
	if !backupSuffixPattern.MatchString(suffix) {
		return errors.New("备份名称无效")
	}
	return nil
}

func regularFile(path string) (fs.FileInfo, error) {
	s, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !s.Mode().IsRegular() {
		return nil, errors.New("只允许备份常规缓存文件，已拒绝符号链接和目录")
	}
	p, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if !p.IsDir() || p.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("缓存目录不能是符号链接")
	}
	return s, nil
}

// moveCache never overwrites an existing destination. Protected containers are
// handled through Finder, with the user's normal macOS Automation permission.
// File paths are AppleScript argv data, never interpolated as script source.
func (e *Engine) moveCache(ctx context.Context, from, to string) error {
	if filepath.Dir(from) != cacheDirectory(e.Home) || filepath.Dir(to) != cacheDirectory(e.Home) {
		return errors.New("缓存移动超出允许目录")
	}
	if _, err := regularFile(from); err != nil {
		return err
	}
	if _, err := os.Lstat(to); err == nil {
		return errors.New("目标文件已存在，已保留两份文件")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	err := renameExclusive(from, to)
	if err != nil && errors.Is(err, fs.ErrPermission) {
		script := `on run argv
tell application "Finder"
if exists (POSIX file (item 3 of argv)) then error "Destination already exists"
if not (exists (POSIX file (item 1 of argv))) then error "Source cache does not exist"
set name of (POSIX file (item 1 of argv)) to (item 2 of argv)
end tell
end run`
		c, cancel := context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
		_, err = e.Runner.Run(c, "/usr/bin/osascript", []string{"-e", script, from, filepath.Base(to), to}, "")
	}
	if err != nil {
		return err
	}
	if _, err = os.Lstat(to); err != nil {
		return fmt.Errorf("未确认备份文件：%w", err)
	}
	if _, err = os.Lstat(from); err == nil {
		return errors.New("原文件仍然存在，请重新诊断")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (e *Engine) BackupCache(ctx context.Context, kind string) (Backup, error) {
	if e.readOnly {
		return Backup{}, errReadOnly
	}
	key := "kLocalPasteboardBlobName"
	if kind == "receive" {
		key = "kRemotePasteboardBlobName"
	} else if kind != "send" {
		return Backup{}, errors.New("缓存方向无效")
	}
	path, ok := e.pref(ctx, key, false)
	if !ok || path == "" {
		return Backup{}, errors.New("系统尚未登记此方向的缓存，无需重建")
	}
	if err := validateCachePath(e.Home, path); err != nil {
		return Backup{}, err
	}
	s, err := regularFile(path)
	if err != nil {
		return Backup{}, fmt.Errorf("缓存暂不可访问：%w", err)
	}
	id := newID()
	backup := Backup{ID: id, Kind: kind, Original: path, Path: path + ".clipboard-fix-backup-" + time.Now().Format("20060102T150405") + "-" + id, Created: time.Now().Format(time.RFC3339), Bytes: s.Size()}
	if err = validateBackupPath(e.Home, path, backup.Path); err != nil {
		return Backup{}, err
	}
	// Persist a recovery record before touching the source file.
	e.mu.Lock()
	e.backups = append(e.backups, backup)
	err = e.persistLocked()
	e.mu.Unlock()
	if err != nil {
		return Backup{}, err
	}
	if err = e.moveCache(ctx, path, backup.Path); err != nil {
		return Backup{}, err
	}
	e.mu.Lock()
	for i := range e.backups {
		if e.backups[i].ID == id {
			e.backups[i].Ready = true
			backup = e.backups[i]
			break
		}
	}
	err = e.persistLocked()
	e.mu.Unlock()
	if err != nil {
		rollbackErr := e.moveCache(ctx, backup.Path, path)
		return backup, errors.Join(fmt.Errorf("备份登记失败：%w", err), rollbackErr)
	}
	e.log("info", "已保留 "+kind+" 缓存备份："+filepath.Base(backup.Path))
	return backup, nil
}

func (e *Engine) RestoreBackup(ctx context.Context, id string) (err error) {
	if e.readOnly {
		return errReadOnly
	}
	if !e.op.TryLock() {
		return errors.New("已有操作正在进行")
	}
	defer e.op.Unlock()
	e.mu.Lock()
	var found Backup
	for _, b := range e.backups {
		if b.ID == id {
			found = b
			break
		}
	}
	e.mu.Unlock()
	if found.ID == "" || found.Restored {
		return errors.New("备份不存在或已经恢复")
	}
	if err = validateBackupPath(e.Home, found.Original, found.Path); err != nil {
		return err
	}
	if _, err = regularFile(found.Path); err != nil {
		return fmt.Errorf("备份文件不可访问：%w", err)
	}
	e.progress("restore", 10, "正在恢复缓存备份", "running")
	defer func() {
		if err != nil {
			e.progress("restore", 100, "恢复未完成："+err.Error(), "error")
		} else {
			e.progress("restore", 100, "备份已恢复，请重新跨设备复测", "done")
		}
	}()
	var displaced string
	if s, statErr := os.Lstat(found.Original); statErr == nil {
		if !s.Mode().IsRegular() {
			return errors.New("原路径不再是常规文件，已停止恢复")
		}
		newID := newID()
		displaced = found.Original + ".clipboard-fix-backup-" + time.Now().Format("20060102T150405") + "-" + newID
		b := Backup{ID: newID, Kind: found.Kind, Original: found.Original, Path: displaced, Created: time.Now().Format(time.RFC3339), Bytes: s.Size()}
		e.mu.Lock()
		e.backups = append(e.backups, b)
		err = e.persistLocked()
		e.mu.Unlock()
		if err != nil {
			return err
		}
		if err = e.moveCache(ctx, found.Original, displaced); err != nil {
			return err
		}
		e.mu.Lock()
		for i := range e.backups {
			if e.backups[i].ID == newID {
				e.backups[i].Ready = true
			}
		}
		err = e.persistLocked()
		e.mu.Unlock()
		if err != nil {
			rollbackErr := e.moveCache(ctx, displaced, found.Original)
			if rollbackErr == nil {
				e.mu.Lock()
				for i := range e.backups {
					if e.backups[i].ID == newID {
						e.backups[i].Ready = false
					}
				}
				e.mu.Unlock()
			}
			return errors.Join(fmt.Errorf("恢复登记失败，已尝试还原当前缓存：%w", err), rollbackErr)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	if err = e.moveCache(ctx, found.Path, found.Original); err != nil {
		if displaced != "" {
			err = errors.Join(err, e.moveCache(ctx, displaced, found.Original))
		}
		return err
	}
	e.mu.Lock()
	for i := range e.backups {
		if e.backups[i].ID == id {
			e.backups[i].Restored = true
		}
	}
	e.verifications = []Verification{}
	e.probe = Probe{}
	err = e.persistLocked()
	e.mu.Unlock()
	if err != nil {
		return err
	}
	key := "kLocalPasteboardBlobName"
	if found.Kind == "receive" {
		key = "kRemotePasteboardBlobName"
	}
	if _, err = e.command(ctx, "/usr/bin/defaults", "write", activityDomain, key, "-string", found.Original); err != nil {
		return err
	}
	e.progress("restore", 75, "缓存文件已恢复，正在刷新接力服务", "running")
	return e.restart(ctx, "useractivityd")
}

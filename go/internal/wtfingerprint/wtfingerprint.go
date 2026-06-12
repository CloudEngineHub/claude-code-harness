package wtfingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxReadBytes = 4096

// Snapshot holds fingerprints for monitored paths. Keys are paths relative to $HOME
// (e.g. ".claude/settings.json"). Used by Capture and Diff.
type Snapshot struct {
	Files map[string]string `json:"files"`
}

// Capture records fingerprints for sensitive paths outside the worker worktree.
// When paths is empty, DefaultWatchPaths() is used. Missing paths are omitted
// from Files (absence is OK; escape detection targets "change").
func Capture(paths []string) (Snapshot, error) {
	if len(paths) == 0 {
		paths = DefaultWatchPaths()
	}
	snap := Snapshot{Files: make(map[string]string)}
	home := homeDir()

	for _, root := range paths {
		if err := captureRoot(root, home, &snap); err != nil {
			return Snapshot{}, err
		}
	}
	return snap, nil
}

// DefaultWatchPaths returns v1 sensitive paths under $HOME.
func DefaultWatchPaths() []string {
	home := homeDir()
	return []string{
		filepath.Join(home, ".claude"),
		filepath.Join(home, ".aws"),
		filepath.Join(home, ".ssh"),
		filepath.Join(home, ".gnupg"),
		filepath.Join(home, ".config", "gcloud"),
		filepath.Join(home, ".netrc"),
	}
}

// Diff compares before and after snapshots and returns changed relative paths.
// An empty result means the invariant holds.
func Diff(before, after Snapshot) []string {
	seen := make(map[string]struct{})
	var changed []string

	for path, fp := range before.Files {
		seen[path] = struct{}{}
		if after.Files[path] != fp {
			changed = append(changed, path)
		}
	}
	for path, fp := range after.Files {
		if _, ok := seen[path]; ok {
			continue
		}
		if before.Files[path] != fp {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return dir
}

func captureRoot(root, home string, snap *Snapshot) error {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return nil
	}

	if !info.IsDir() {
		rel, relErr := relativeHomePath(root, home)
		if relErr != nil {
			return nil
		}
		fp, fpErr := fingerprintFile(root, info)
		if fpErr != nil || fp == "" {
			return nil
		}
		snap.Files[rel] = fp
		return nil
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		rel, relErr := relativeHomePath(path, home)
		if relErr != nil {
			return nil
		}
		fp, fpErr := fingerprintFile(path, info)
		if fpErr != nil || fp == "" {
			return nil
		}
		snap.Files[rel] = fp
		return nil
	})
}

func relativeHomePath(absPath, home string) (string, error) {
	if home == "" {
		return "", fmt.Errorf("home not set")
	}
	home = filepath.Clean(home)
	absPath = filepath.Clean(absPath)
	rel, err := filepath.Rel(home, absPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path outside home: %s", absPath)
	}
	return rel, nil
}

func fingerprintFile(path string, info fs.FileInfo) (string, error) {
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		hash := sha256.Sum256([]byte(target))
		return fmt.Sprintf("%d-%d-%s", info.ModTime().UnixNano(), info.Size(), hex.EncodeToString(hash[:])), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, maxReadBytes)
	n, readErr := io.ReadFull(f, buf)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		return "", readErr
	}
	hash := sha256.Sum256(buf[:n])
	return fmt.Sprintf("%d-%d-%s", info.ModTime().UnixNano(), info.Size(), hex.EncodeToString(hash[:])), nil
}

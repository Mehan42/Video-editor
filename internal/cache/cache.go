// Package cache persists completed pipeline runs so a repeat invocation with
// the same input bytes and same effective parameters can reuse the previous
// transcript/audio/chunks instead of re-running ffmpeg and whisper. The cache
// is strictly local (no network), lives below the operator-selected output
// root, and can be disabled per run with --no-cache.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Key identifies a cached run. InputHash is the SHA-256 of the raw input file;
// ParamsHash is a stable hash over every parameter that changes the outputs
// (dependencies, language, chunk geometry, etc.). Two runs map to the same
// cache dir only when both match.
type Key struct {
	InputHash  string `json:"input_hash"`
	ParamsHash string `json:"params_hash"`
}

// DirName is the on-disk directory name for a cache entry: short, stable and
// filesystem-safe. Short inputs are zero-padded on the right so the function
// never panics on bad metadata.
func (k Key) DirName() string {
	return shortHash(k.InputHash) + "-" + shortHash(k.ParamsHash)
}

func shortHash(s string) string {
	if len(s) >= 16 {
		return s[:16]
	}
	// Right-pad with '0' — safe because hex chars never include '0'..'f' gaps
	// in a way that could collide with any real 16-char prefix.
	buf := make([]byte, 16)
	copy(buf, s)
	for i := len(s); i < 16; i++ {
		buf[i] = '0'
	}
	return string(buf)
}

// ParamsHash computes a stable hash over everything that can change the
// pipeline artifacts for the same input. Anything not included here is either
// cosmetic (log output) or independent of the byte content of the artifacts
// (run id, wall clock).
func ParamsHash(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = io.WriteString(h, part)
		_, _ = io.WriteString(h, "\x00")
	}
	return hex.EncodeToString(h.Sum(nil))
}

// RootDir returns the cache root below the operator's output directory.
func RootDir(outputRoot string) string {
	return filepath.Join(outputRoot, ".cache")
}

// Dir returns the directory that holds (or will hold) the cached run for key.
func Dir(outputRoot string, key Key) string {
	return filepath.Join(RootDir(outputRoot), key.DirName())
}

// meta records enough information to validate a cache hit against the caller's
// parameters and to explain a miss in diagnostics. Hashes seal the artifacts
// the hit path intends to reuse; they are verified before any copy happens.
type meta struct {
	InputHash      string    `json:"input_hash"`
	ParamsHash     string    `json:"params_hash"`
	StoredAt       time.Time `json:"stored_at"`
	Version        int       `json:"version"`
	AudioHash      string    `json:"audio_hash"`
	TranscriptHash string    `json:"transcript_hash"`
	SubtitleHash   string    `json:"subtitle_hash"`
}

const metaFile = "cache.json"
const metaVersion = 1

// Load verifies that dir is a complete, self-consistent cache entry for key
// and returns the absolute paths of its artifacts. It returns ok=false on any
// inconsistency — a corrupt cache must never crash a run, only force a
// recompute.
func Load(outputRoot string, key Key) (dir string, ok bool) {
	dir = Dir(outputRoot, key)
	data, err := os.ReadFile(filepath.Join(dir, metaFile))
	if err != nil {
		return "", false
	}
	var m meta
	if err := json.Unmarshal(data, &m); err != nil {
		return "", false
	}
	if m.Version != metaVersion || m.InputHash != key.InputHash || m.ParamsHash != key.ParamsHash {
		return "", false
	}
	// Every file we plan to reuse must exist and hash to its recorded value.
	required := []struct {
		name string
		hash string
	}{
		{"audio.wav", m.AudioHash},
		{"transcript.txt", m.TranscriptHash},
		{"transcript.srt", m.SubtitleHash},
	}
	for _, r := range required {
		if r.hash == "" {
			return "", false
		}
		actual, err := HashFile(filepath.Join(dir, r.name))
		if err != nil || actual != r.hash {
			return "", false
		}
	}
	return dir, true
}

// HashFile returns the SHA-256 hex digest of the file at path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SaveMetadata writes the meta record for an already-populated cache dir. The
// caller writes the artifact files first; SaveMetadata only seals the entry.
func SaveMetadata(dir string, key Key, audioHash, transcriptHash, subtitleHash string) error {
	m := meta{
		InputHash:      key.InputHash,
		ParamsHash:     key.ParamsHash,
		StoredAt:       time.Now().UTC(),
		Version:        metaVersion,
		AudioHash:      audioHash,
		TranscriptHash: transcriptHash,
		SubtitleHash:   subtitleHash,
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, metaFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, metaFile))
}

// copyDir recursively copies src into dst, preserving regular files only
// (no symlinks followed, no device files). Mode 0700 on dirs, 0600 on files.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o700)
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("refusing to copy non-regular file %s", path)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// RestoreHit copies a validated cache dir into a fresh run root. The manifest
// file is intentionally NOT copied: the caller rewrites it with the new run id
// and new absolute paths after the copy completes.
func RestoreHit(cacheDir, runRoot string) error {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(runRoot, 0o700); err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if name == metaFile || name == "manifest.json" || strings.HasSuffix(name, ".tmp") {
			continue
		}
		src := filepath.Join(cacheDir, name)
		dst := filepath.Join(runRoot, name)
		if e.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return err
			}
			continue
		}
		if !e.Type().IsRegular() {
			continue
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// ErrCorrupt is returned by callers when a cache directory fails validation;
// callers treat it as "miss" and recompute.
var ErrCorrupt = errors.New("cache entry failed validation")

// ListKeys is a debugging helper: it returns the sorted keys of every cache
// entry below outputRoot. Not used by the hot path.
func ListKeys(outputRoot string) ([]string, error) {
	entries, err := os.ReadDir(RootDir(outputRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

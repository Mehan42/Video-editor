package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestParamsHashDeterministic(t *testing.T) {
	a := ParamsHash("ffmpeg-A", "whisper-A", "model-A", "auto", "1800", "40")
	b := ParamsHash("ffmpeg-A", "whisper-A", "model-A", "auto", "1800", "40")
	if a != b {
		t.Fatalf("params hash not deterministic: %s vs %s", a, b)
	}
	c := ParamsHash("ffmpeg-A", "whisper-A", "model-A", "ru", "1800", "40")
	if a == c {
		t.Fatal("params hash must change when language changes")
	}
	d := ParamsHash("ffmpeg-B", "whisper-A", "model-A", "auto", "1800", "40")
	if a == d {
		t.Fatal("params hash must change when ffmpeg bytes change")
	}
}

func TestKeyDirName(t *testing.T) {
	key := Key{InputHash: "0123456789abcdef_fedcba9876543210", ParamsHash: "aabbccddeeff0011_1100ffeeffddccbb"}
	got := key.DirName()
	want := "0123456789abcdef-aabbccddeeff0011"
	if got != want {
		t.Fatalf("DirName() = %q, want %q", got, want)
	}
}

func TestLoadMissWhenNothingStored(t *testing.T) {
	root := t.TempDir()
	key := Key{
		InputHash:  hex.EncodeToString([]byte("input-hash-must-be-long-enough----")),
		ParamsHash: hex.EncodeToString([]byte("params-hash-is-also-long----------")),
	}
	if _, ok := Load(root, key); ok {
		t.Fatal("expected a miss for an empty cache root")
	}
	if got := key.DirName(); len(got) != 33 {
		t.Fatalf("DirName() length = %d, want 33", len(got))
	}
}

func TestStoreThenLoadHits(t *testing.T) {
	root := t.TempDir()
	key := Key{
		InputHash:  hex.EncodeToString([]byte("input-hash-must-be-long-enough----")),
		ParamsHash: hex.EncodeToString([]byte("params-hash-is-also-long----------")),
	}
	dir := Dir(root, key)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Populate artifacts.
	contents := map[string][]byte{
		"audio.wav":      []byte("fake wav bytes for test"),
		"transcript.txt": []byte("fake transcript"),
		"transcript.srt": []byte("fake subtitles"),
	}
	hashes := map[string]string{}
	for name, data := range contents {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		hashes[name] = hex.EncodeToString(sum[:])
	}
	if err := SaveMetadata(dir, key, hashes["audio.wav"], hashes["transcript.txt"], hashes["transcript.srt"]); err != nil {
		t.Fatal(err)
	}
	gotDir, ok := Load(root, key)
	if !ok {
		t.Fatal("expected a hit after SaveMetadata")
	}
	if gotDir != dir {
		t.Fatalf("Load returned dir %q, want %q", gotDir, dir)
	}
}

func TestLoadRejectsCorruptedArtifact(t *testing.T) {
	root := t.TempDir()
	key := Key{
		InputHash:  hex.EncodeToString([]byte("input-hash-must-be-long-enough----")),
		ParamsHash: hex.EncodeToString([]byte("params-hash-is-also-long----------")),
	}
	dir := Dir(root, key)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "audio.wav"), []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "transcript.txt"), []byte("transcript"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "transcript.srt"), []byte("subs"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Save metadata with deliberately wrong hashes.
	bad := hex.EncodeToString(make([]byte, sha256.Size))
	if err := SaveMetadata(dir, key, bad, bad, bad); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(root, key); ok {
		t.Fatal("corrupted artifact hashes must force a miss")
	}
}

func TestRestoreHitCopiesArtifactsButNotManifestOrMeta(t *testing.T) {
	root := t.TempDir()
	key := Key{
		InputHash:  hex.EncodeToString([]byte("input-hash-must-be-long-enough----")),
		ParamsHash: hex.EncodeToString([]byte("params-hash-is-also-long----------")),
	}
	cacheDir := Dir(root, key)
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"audio.wav":      "wav-bytes",
		"transcript.txt": "transcript bytes",
		"transcript.srt": "subtitle bytes",
		"cache.json":     `{"stale":"meta"}`,
		"manifest.json":  `{"stale":"manifest"}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(cacheDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runRoot := filepath.Join(t.TempDir(), "new-run")
	if err := RestoreHit(cacheDir, runRoot); err != nil {
		t.Fatal(err)
	}
	// Artifacts must be present with identical contents.
	for name, want := range map[string]string{
		"audio.wav":      "wav-bytes",
		"transcript.txt": "transcript bytes",
		"transcript.srt": "subtitle bytes",
	} {
		got, err := os.ReadFile(filepath.Join(runRoot, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != want {
			t.Fatalf("%s restored content = %q, want %q", name, got, want)
		}
	}
	// Manifest and cache meta must NOT be copied; the caller rewrites them.
	if _, err := os.Stat(filepath.Join(runRoot, "manifest.json")); !os.IsNotExist(err) {
		t.Fatal("manifest.json must not be restored from cache")
	}
	if _, err := os.Stat(filepath.Join(runRoot, "cache.json")); !os.IsNotExist(err) {
		t.Fatal("cache.json must not be restored from cache")
	}
}

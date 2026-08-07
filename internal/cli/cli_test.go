package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecute_BareLocalPathIsProcessed(t *testing.T) {
	// A bare path to a non-video file must become a --input and then fail
	// validation with a clear extension error, not "unknown command".
	tmp := filepath.Join(t.TempDir(), "video.mp4")
	if err := os.WriteFile(tmp, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errBuf bytes.Buffer
	err := Execute(context.Background(), []string{tmp}, &out, &errBuf)
	if err == nil {
		t.Fatal("expected validation error for stub mp4 (missing ffmpeg)")
	}
	var ue *UsageError
	if strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("bare path was not recognized as input: %v", err)
	}
	if !strings.Contains(err.Error(), "ffmpeg") && !strings.Contains(err.Error(), "stat") {
		t.Fatalf("unexpected error for bare path: %v (want ffmpeg/stat validation)", err)
	}
	_ = ue
}

func TestExecute_HTTPUrlOpensProcessAndReportsNotImplemented(t *testing.T) {
	var out, errBuf bytes.Buffer
	err := Execute(context.Background(), []string{"https://youtube.com/watch?v=abc"}, &out, &errBuf)
	if err == nil {
		t.Fatal("expected error for URL input")
	}
	if strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("URL was not recognized as input: %v", err)
	}
	if !strings.Contains(err.Error(), "not supported yet") || !strings.Contains(err.Error(), "downloader") {
		t.Fatalf("unexpected error for URL input: %v", err)
	}
}

func TestExecute_UnknownCommandStillFails(t *testing.T) {
	var out, errBuf bytes.Buffer
	err := Execute(context.Background(), []string{"definitely-not-a-path-xyz"}, &out, &errBuf)
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command, got: %v", err)
	}
}

func TestLooksLikeInput(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "a.mp4")
	if err := os.WriteFile(tmp, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{
		"http://a.b/v.mp4":  true,
		"https://a.b/v.mp4": true,
		tmp:                 true,
		"":                  false,
		"missing-file.mp4":  false,
	}
	for in, want := range cases {
		if got := looksLikeInput(in); got != want {
			t.Errorf("looksLikeInput(%q) = %v, want %v", in, got, want)
		}
	}
}

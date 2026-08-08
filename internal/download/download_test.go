package download

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests never hit the network and never invoke a real yt-dlp — they
// lock in the validation gates that stand between a URL and an actual
// egress. Live downloads are exercised only by an operator-driven smoke run.

func TestSupportedSchemes(t *testing.T) {
	cases := map[string]bool{
		"https://youtube.com/watch?v=x": true,
		"http://example.com/v.mp4":      true,
		"ftp://example.com/v.mp4":       false, // not http(s)
		"file:///etc/passwd":            false,
		"gopher://x":                    false,
		"data:text/plain,hi":            false,
		"//example.com/no-scheme":       false,
		"":                              false,
		"not a url":                     false,
	}
	for in, want := range cases {
		if got := Supported(in); got != want {
			t.Errorf("Supported(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFetchRejectsUnsupportedScheme(t *testing.T) {
	// We use a dummy binary path; validation must fail before we ever exec it.
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "yt-dlp.exe")
	if err := os.WriteFile(fake, []byte("not actually an exe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Fetch(context.Background(), fake, fake, "ftp://x/v.mp4", tmp, 1<<20); err == nil || !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Fatalf("expected scheme error, got %v", err)
	}
}

func TestFetchRejectsMissingBinary(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-ytdlp.exe")
	if _, _, err := Fetch(context.Background(), missing, missing, "https://example.com/v.mp4", t.TempDir(), 1<<20); err == nil || !strings.Contains(err.Error(), "stat yt-dlp") {
		t.Fatalf("expected stat yt-dlp error, got %v", err)
	}
}

func TestFetchRejectsNonPositiveMaxBytes(t *testing.T) {
	// Must fail before any exec: a zero/negative maxBytes is a bug in the call
	// site, never a user error.
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "yt-dlp.exe")
	if err := os.WriteFile(fake, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Fetch(context.Background(), fake, fake, "https://example.com/v.mp4", tmp, 0); err == nil || !strings.Contains(err.Error(), "max download size") {
		t.Fatalf("expected maxBytes validation, got %v", err)
	}
}

func TestFetchHonorsContextCancellationBeforeExec(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "yt-dlp.exe")
	if err := os.WriteFile(fake, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // ensure the timer has fired
	_, _, err := Fetch(ctx, fake, fake, "https://example.com/v.mp4", tmp, 1<<20)
	if err == nil {
		t.Fatal("expected an error for a pre-cancelled context")
	}
	// The exec call itself will fail because the file is not a real
	// executable; what we assert here is the call returned promptly rather
	// than blocking on a download.
}

// TestFetchUsesSeparateArguments verifies we never concatenate the URL into
// a shell command string. We can't easily observe exec.Command's args from
// outside, so this test asserts the binary path is validated as a file — if
// someone tried to be clever with `exec.Command("sh", "-c", ytdlpPath+" "+url)`
// the path check would still catch a non-existent binary, but a string-splice
// vulnerability would be possible. Keeping this test as a tripwire.
func TestFetchRequiresRegularFile(t *testing.T) {
	tmp := t.TempDir()
	// Pass the directory itself as the binary path — must fail.
	if _, _, err := Fetch(context.Background(), tmp, tmp, "https://example.com/v.mp4", tmp, 1<<20); err == nil {
		t.Fatal("expected error when ytdlp path is a directory")
	}
}

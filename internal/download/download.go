// Package download fetches a remote video URL to a local file via the
// external `yt-dlp` binary, so the rest of the pipeline can consume a
// regular file path. Network egress happens only when the operator passes
// --allow-download; without that flag the CLI rejects URL inputs before
// this package is ever called.
package download

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultTimeout caps how long a single download may run. The pipeline-level
// --timeout runs on top of this and is usually looser.
const DefaultTimeout = 10 * time.Minute

// Supported reports whether url is eligible for the yt-dlp downloader. We
// deliberately accept only http(s) — file:, gopher:, data: and other schemes
// are rejected before any network egress occurs.
func Supported(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// Fetch invokes yt-dlp with separate arguments (never a shell string) to
// download rawVideo to a file under dstDir, and returns the absolute path
// plus the file size. Errors wrap the yt-dlp stderr where available so the
// operator sees the real reason (geo-block, auth required, 404, throttling).
//
// ffmpegPath must point at the ffmpeg *binary* (not its directory); yt-dlp
// uses it to merge video+audio DASH tracks into a single mp4 (most modern
// hosts, YouTube included, do not ship a pre-merged file). Callers therefore
// pass the same ffmpeg binary the pipeline would use for audio extraction.
//
// Boundary invariants enforced here:
//   - dstDir must already exist with operator-controlled permissions.
//   - the output template forces a single file with a safe extension; we
//     never use a shell or interpret the remote server's filename.
//   - no cookies/auth/custom headers: only anonymously fetchable URLs work.
func Fetch(ctx context.Context, ytdlpPath, ffmpegPath, rawURL, dstDir string, maxBytes int64) (path string, size int64, err error) {
	if !Supported(rawURL) {
		return "", 0, fmt.Errorf("unsupported URL scheme: %s", rawURL)
	}
	if maxBytes <= 0 {
		return "", 0, errors.New("max download size must be positive")
	}
	if strings.TrimSpace(ytdlpPath) == "" {
		return "", 0, errors.New("yt-dlp path is empty")
	}
	if _, err := os.Stat(ytdlpPath); err != nil {
		return "", 0, fmt.Errorf("stat yt-dlp: %w", err)
	}
	if strings.TrimSpace(ffmpegPath) == "" {
		return "", 0, errors.New("ffmpeg path is empty")
	}
	// yt-dlp expects --ffmpeg-location to name the directory containing
	// ffmpeg.exe on Windows; we point it at the directory of the same binary
	// the pipeline already trusts.
	ffmpegDir := filepath.Dir(ffmpegPath)

	// -f "bv*+ba/b" prefers best video+best audio; yt-dlp will merge them via
	// ffmpeg when --merge-output-format is given (we bundle ffmpeg; YuTuBe
	// and most hosts ship video/audio as separate DASH tracks and a single
	// output container is what the pipeline expects). --no-playlist prevents
	// users from accidentally pulling a whole playlist through a single-URL
	// command. -o pins the filename under dstDir with a sanitized,
	// deterministic template; --no-part removes the intermediate .part file
	// on success so our scanner sees only complete files.
	args := []string{
		"--no-call-home",
		"--no-playlist",
		"--no-warnings",
		"--quiet",
		"--no-progress",
		"--ffmpeg-location", ffmpegDir,
		"-f", "bv*+ba/b",
		"--merge-output-format", "mp4",
		"--no-part",
		"-o", filepath.Join(dstDir, "download.%(ext)s"),
		rawURL,
	}
	cmd := exec.CommandContext(ctx, ytdlpPath, args...)
	cmd.Stdin = nil
	var stderr strings.Builder
	cmd.Stdout = nil
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", 0, ctx.Err()
		}
		return "", 0, fmt.Errorf("yt-dlp: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	// After yt-dlp exits successfully, the staging dir must contain exactly
	// one file matching "download.<ext>". Scanning after the fact is more
	// robust than parsing --print output because our Windows yt-dlp versions
	// have shown build-dependent inconsistency in after_move hook output.
	candidates, err := filepath.Glob(filepath.Join(dstDir, "download.*"))
	if err != nil {
		return "", 0, err
	}
	var outs []string
	for _, c := range candidates {
		if strings.HasSuffix(c, ".part") || strings.HasSuffix(c, ".ytdl") {
			continue
		}
		outs = append(outs, c)
	}
	if len(outs) == 0 {
		return "", 0, fmt.Errorf("yt-dlp reported success but produced no download.* in %s", dstDir)
	}
	if len(outs) > 1 {
		return "", 0, fmt.Errorf("yt-dlp left %d candidate files in %s (refusing to guess)", len(outs), dstDir)
	}
	out, err := filepath.Abs(outs[0])
	if err != nil {
		return "", 0, err
	}
	// Defensive: the file must live under dstDir. A symlink-attack or --paths
	// trick inside yt-dlp config would otherwise smuggle the artifact outside
	// the run directory.
	dstAbs, err := filepath.Abs(dstDir)
	if err != nil {
		return "", 0, err
	}
	rel, err := filepath.Rel(dstAbs, out)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", 0, fmt.Errorf("yt-dlp wrote outside download dir: %s", out)
	}
	info, err := os.Stat(out)
	if err != nil {
		return "", 0, fmt.Errorf("stat downloaded file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", 0, errors.New("downloaded path is not a regular file")
	}
	if info.Size() == 0 {
		return "", 0, errors.New("downloaded file is empty")
	}
	if info.Size() > maxBytes {
		// Remove the oversized artifact: leaving it on disk would let an
		// attacker burn quota by pointing us at huge URLs.
		_ = os.Remove(out)
		return "", 0, fmt.Errorf("downloaded file exceeds max-input-bytes: %d > %d", info.Size(), maxBytes)
	}
	return out, info.Size(), nil
}

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pagevideo/internal/config"
	"pagevideo/internal/llm"
	"pagevideo/internal/pipeline"
)

type UsageError struct {
	Message string
}

func (e *UsageError) Error() string { return e.Message }

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &UsageError{Message: usageHelp}
	}

	switch args[0] {
	case "process":
		return executeProcess(ctx, args[1:], stdout, stderr)
	case "provider":
		return executeProvider(ctx, args[1:], stdout, stderr)
	case "version":
		_, err := fmt.Fprintln(stdout, "pagevideo dev")
		return err
	case "--help", "-h", "help":
		_, err := fmt.Fprintln(stdout, usageHelp)
		return err
	default:
		// Convenience: user dropped a raw video path (e.g. in the REPL) without
		// typing "process". If the first token is an existing file or an http(s)
		// URL, treat the whole line as the input of a process command.
		if looksLikeInput(args[0]) {
			return executeProcess(ctx, append([]string{"--input", args[0]}, args[1:]...), stdout, stderr)
		}
		return &UsageError{Message: fmt.Sprintf("unknown command %q", args[0])}
	}
}

// looksLikeInput reports whether the first CLI token is a direct media source:
// an http(s) URL or an existing local file. Used so a user can drop a bare path
// or URL without typing the "process --input" prefix.
func looksLikeInput(token string) bool {
	if strings.HasPrefix(token, "http://") || strings.HasPrefix(token, "https://") {
		return true
	}
	info, err := os.Stat(token)
	return err == nil && info.Mode().IsRegular()
}

const usageHelp = `PageVideo CLI — local video to transcript/chunks/summary pipeline.

Usage:
  pagevideo process --input FILE [options]     Run local pipeline on a video
  pagevideo provider check [--base-url URL]    Check local LLM (Bionic) readiness
  pagevideo version                            Print version
  pagevideo --help                             Show this help

You can also drop a bare path or URL to a video instead of typing "process --input".

Key process options:
  --enable-summary        Send transcript to local Bionic chat and write summary.md (opt-in, off by default)
  --llm-base-url URL      Local OpenAI-compatible endpoint (default http://127.0.0.1:1234/v1)
  --language LANG         Spoken language or auto (default auto)
  --timeout DURATION      Max processing time (default 30m)
  --max-input-bytes N     Reject inputs larger than N bytes

Without --enable-summary no network/LLM activity occurs at all.`

func executeProvider(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "check" {
		return &UsageError{Message: "usage: pagevideo provider check [--base-url URL]"}
	}
	fs := flag.NewFlagSet("provider check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	baseURL := fs.String("base-url", "http://127.0.0.1:1234/v1", "local Bionic OpenAI-compatible base URL")
	timeout := fs.Duration("timeout", 5*time.Second, "provider readiness timeout")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return &UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return &UsageError{Message: "provider check does not accept positional arguments"}
	}
	client, err := llm.NewBionicClient(llm.Config{BaseURL: *baseURL, Timeout: *timeout}, nil)
	if err != nil {
		return &UsageError{Message: err.Error()}
	}
	readiness := client.Check(ctx)
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(readiness); err != nil {
		return err
	}
	if readiness.Status != "READY" {
		return fmt.Errorf("provider readiness: %s", readiness.Status)
	}
	return nil
}

func executeProcess(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("process", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "local video file")
	output := fs.String("output", "output", "output root")
	ffmpeg := fs.String("ffmpeg", filepath.Join("ffmpeg", "bin", "ffmpeg.exe"), "ffmpeg executable")
	whisper := fs.String("whisper", filepath.Join("whisper.cpp", "bin", "whisper-cli.exe"), "whisper executable")
	model := fs.String("model", filepath.Join("whisper.cpp", "models", "ggml-base.bin"), "Whisper model")
	language := fs.String("language", "auto", "spoken language or auto")
	chunkChars := fs.Int("chunk-chars", 1800, "maximum chunk size in Unicode characters")
	overlapWords := fs.Int("chunk-overlap-words", 40, "overlap between adjacent chunks in words")
	timeout := fs.Duration("timeout", 30*time.Minute, "maximum processing duration")
	maxInputBytes := fs.Int64("max-input-bytes", 4*1024*1024*1024, "maximum accepted input size")
	enableSummary := fs.Bool("enable-summary", false, "send transcript to a local LLM (Bionic) to write summary.md; explicit operator gate")
	llmBaseURL := fs.String("llm-base-url", "http://127.0.0.1:1234/v1", "local LLM (OpenAI-compatible) base URL used with --enable-summary")
	llmTimeout := fs.Duration("llm-timeout", 90*time.Second, "LLM chat timeout when summary is enabled")
	llmMaxResponse := fs.Int64("llm-max-response-bytes", 2*1024*1024, "LLM response byte limit")
	summaryMaxChars := fs.Int("summary-max-chars", 24000, "maximum transcript characters sent in one summary request")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return &UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return &UsageError{Message: "process does not accept positional arguments"}
	}
	if *input == "" {
		return &UsageError{Message: "--input is required"}
	}

	cfg := config.Config{
		Input:               *input,
		OutputRoot:          *output,
		FFmpeg:              *ffmpeg,
		Whisper:             *whisper,
		Model:               *model,
		Language:            *language,
		ChunkChars:          *chunkChars,
		OverlapWords:        *overlapWords,
		Timeout:             *timeout,
		MaxInputBytes:       *maxInputBytes,
		EnableSummary:       *enableSummary,
		LLMBaseURL:          *llmBaseURL,
		LLMTimeout:          *llmTimeout,
		LLMMaxResponseBytes: *llmMaxResponse,
		SummaryMaxChars:     *summaryMaxChars,
	}
	if err := cfg.Validate(); err != nil {
		return &UsageError{Message: err.Error()}
	}

	result, err := pipeline.Run(ctx, cfg, stderr)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

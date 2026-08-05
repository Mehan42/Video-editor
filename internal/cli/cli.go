package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"pagevideo/internal/config"
	"pagevideo/internal/pipeline"
)

type UsageError struct {
	Message string
}

func (e *UsageError) Error() string { return e.Message }

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &UsageError{Message: "usage: pagevideo process --input FILE [options]"}
	}

	switch args[0] {
	case "process":
		return executeProcess(ctx, args[1:], stdout, stderr)
	case "version":
		_, err := fmt.Fprintln(stdout, "pagevideo dev")
		return err
	default:
		return &UsageError{Message: fmt.Sprintf("unknown command %q", args[0])}
	}
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
		Input:         *input,
		OutputRoot:    *output,
		FFmpeg:        *ffmpeg,
		Whisper:       *whisper,
		Model:         *model,
		Language:      *language,
		ChunkChars:    *chunkChars,
		OverlapWords:  *overlapWords,
		Timeout:       *timeout,
		MaxInputBytes: *maxInputBytes,
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

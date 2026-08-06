package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"pagevideo/internal/chunk"
	"pagevideo/internal/config"
	"pagevideo/internal/llm"
)

type Result struct {
	RunID      string        `json:"run_id"`
	Status     string        `json:"status"`
	Input      Artifact      `json:"input"`
	Audio      Artifact      `json:"audio"`
	Transcript Artifact      `json:"transcript"`
	Subtitles  Artifact      `json:"subtitles"`
	Chunks     []chunk.Chunk `json:"chunks"`
	// Summary is present only when EnableSummary was requested AND the LLM
	// call succeeded. Its content is untrusted model output, never authority.
	Summary      *Artifact `json:"summary,omitempty"`
	ManifestPath string    `json:"manifest_path"`
}

type Artifact struct {
	Path string `json:"path"`
	Hash string `json:"sha256"`
}

func Run(ctx context.Context, cfg config.Config, logWriter io.Writer) (Result, error) {
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}
	abs, err := cfg.AbsolutePaths()
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, abs.Timeout)
	defer cancel()
	inputHash, err := fileHash(abs.Input)
	if err != nil {
		return Result{}, fmt.Errorf("hash input: %w", err)
	}
	runID := fmt.Sprintf("%s-%d", inputHash[:16], time.Now().UTC().UnixNano())
	runRoot := filepath.Join(abs.OutputRoot, runID)
	if err := os.MkdirAll(runRoot, 0o700); err != nil {
		return Result{}, fmt.Errorf("create run root: %w", err)
	}
	logf(logWriter, "pagevideo: run=%s input=%s", runID, abs.Input)

	input := Artifact{Path: abs.Input, Hash: inputHash}
	audioPath := filepath.Join(runRoot, "audio.wav")
	if err := runCommand(ctx, logWriter, abs.FFmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-i", abs.Input, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-y", audioPath); err != nil {
		return Result{}, fmt.Errorf("extract audio: %w", err)
	}
	audioHash, err := fileHash(audioPath)
	if err != nil {
		return Result{}, fmt.Errorf("hash audio: %w", err)
	}

	transcriptPrefix := filepath.Join(runRoot, "transcript")
	if err := runCommand(ctx, logWriter, abs.Whisper, "-m", abs.Model, "-f", audioPath, "-l", abs.Language, "-otxt", "-osrt", "-nt", "-of", transcriptPrefix); err != nil {
		return Result{}, fmt.Errorf("transcribe audio: %w", err)
	}
	transcriptPath := transcriptPrefix + ".txt"
	subtitlesPath := transcriptPrefix + ".srt"
	transcript, err := os.ReadFile(transcriptPath)
	if err != nil {
		return Result{}, fmt.Errorf("read transcript: %w", err)
	}
	transcriptHash := hashBytes(transcript)
	subtitleHash, err := fileHash(subtitlesPath)
	if err != nil {
		return Result{}, fmt.Errorf("hash subtitles: %w", err)
	}

	chunks, err := chunk.Split(string(transcript), inputHash[:16], inputHash, runID, abs.ChunkChars, abs.OverlapWords)
	if err != nil {
		return Result{}, fmt.Errorf("split transcript: %w", err)
	}
	result := Result{
		RunID:      runID,
		Status:     "READY",
		Input:      input,
		Audio:      Artifact{Path: audioPath, Hash: audioHash},
		Transcript: Artifact{Path: transcriptPath, Hash: transcriptHash},
		Subtitles:  Artifact{Path: subtitlesPath, Hash: subtitleHash},
		Chunks:     chunks,
	}
	if summary, serr := maybeSummarize(ctx, cfg, string(transcript), runRoot); serr != nil {
		// Never fail the whole pipeline for a summary error: the transcript and
		// chunks are already safely written. Degrade to READY without summary.
		logf(logWriter, "pagevideo: summary skipped: %v", serr)
	} else if summary != nil {
		result.Summary = summary
	}
	manifestPath := filepath.Join(runRoot, "manifest.json")
	result.ManifestPath = manifestPath
	if err := writeJSON(manifestPath, result); err != nil {
		return Result{}, fmt.Errorf("write manifest: %w", err)
	}
	return result, nil
}

func runCommand(ctx context.Context, logWriter io.Writer, executable string, args ...string) error {
	command := exec.CommandContext(ctx, executable, args...)
	command.Stdin = nil
	command.Stdout = logWriter
	command.Stderr = logWriter
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%s: %w", filepath.Base(executable), err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	tmpPath := path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(value)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmpPath, path)
}

func fileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func logf(writer io.Writer, format string, args ...any) {
	if writer == nil {
		return
	}
	_, _ = fmt.Fprintf(writer, format+"\n", args...)
}

// maybeSummarize returns nil,nil when summarization is disabled. When enabled
// it calls the local LLM gateway, writes the result as summary.md below the
// run root with hash, and returns the artifact. Any failure is returned so the
// caller can degrade gracefully; nothing here can affect provider, policy or
// capability choice other than through Config.
func maybeSummarize(ctx context.Context, cfg config.Config, transcript, runRoot string) (*Artifact, error) {
	if !cfg.EnableSummary {
		return nil, nil
	}
	client, err := llm.NewBionicClient(llm.Config{
		BaseURL:          cfg.LLMBaseURL,
		Timeout:          cfg.LLMTimeout,
		MaxResponseBytes: cfg.LLMMaxResponseBytes,
		AllowChat:        true, // reached only when EnableSummary gate passed
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("llm client: %w", err)
	}
	summary, err := client.SummarizeTranscript(ctx, transcript)
	if err != nil {
		return nil, fmt.Errorf("summarize: %w", err)
	}
	summaryPath := filepath.Join(runRoot, "summary.md")
	if err := os.WriteFile(summaryPath, []byte(summary), 0o600); err != nil {
		return nil, fmt.Errorf("write summary: %w", err)
	}
	hash, err := fileHash(summaryPath)
	if err != nil {
		return nil, fmt.Errorf("hash summary: %w", err)
	}
	return &Artifact{Path: summaryPath, Hash: hash}, nil
}

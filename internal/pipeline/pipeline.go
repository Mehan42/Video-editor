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
	"strconv"
	"time"

	"pagevideo/internal/cache"
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
	// Summary, Study, FAQ, Glossary are present only when EnableSummary was
	// requested AND the corresponding LLM call succeeded. Their content is
	// untrusted model output, never authority.
	Summary      *Artifact `json:"summary,omitempty"`
	Study        *Artifact `json:"study,omitempty"`
	FAQ          *Artifact `json:"faq,omitempty"`
	Glossary     *Artifact `json:"glossary,omitempty"`
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
	// Dependency state participates in the cache key: if the operator swaps
	// ffmpeg, whisper or the model file, prior runs must not be reused.
	ffmpegHash, err := hashDependency(abs.FFmpeg)
	if err != nil {
		return Result{}, fmt.Errorf("hash ffmpeg: %w", err)
	}
	whisperHash, err := hashDependency(abs.Whisper)
	if err != nil {
		return Result{}, fmt.Errorf("hash whisper: %w", err)
	}
	modelHash, err := hashDependency(abs.Model)
	if err != nil {
		return Result{}, fmt.Errorf("hash model: %w", err)
	}
	paramsHash := cache.ParamsHash(
		ffmpegHash,
		whisperHash,
		modelHash,
		abs.Language,
		strconv.Itoa(abs.ChunkChars),
		strconv.Itoa(abs.OverlapWords),
	)
	runID := fmt.Sprintf("%s-%d", inputHash[:16], time.Now().UTC().UnixNano())
	runRoot := filepath.Join(abs.OutputRoot, runID)
	if abs.UseCache {
		key := cache.Key{InputHash: inputHash, ParamsHash: paramsHash}
		if cacheDir, ok := cache.Load(abs.OutputRoot, key); ok {
			logf(logWriter, "pagevideo: cache hit run=%s source=%s", runID, cacheDir)
			if err := cache.RestoreHit(cacheDir, runRoot); err != nil {
				logf(logWriter, "pagevideo: cache restore failed, falling back to full run: %v", err)
			} else {
				return finishFromCache(ctx, cfg, abs, runID, runRoot, inputHash)
			}
		}
	}
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
	if err := maybeArtifacts(ctx, cfg, string(transcript), runRoot, &result); err != nil {
		// Never fail the whole pipeline for an LLM error: the transcript and
		// chunks are already safely written. Degrade to READY without artifacts.
		logf(logWriter, "pagevideo: artifact generation skipped: %v", err)
	}
	manifestPath := filepath.Join(runRoot, "manifest.json")
	result.ManifestPath = manifestPath
	if err := writeJSON(manifestPath, result); err != nil {
		return Result{}, fmt.Errorf("write manifest: %w", err)
	}
	if abs.UseCache {
		key := cache.Key{InputHash: inputHash, ParamsHash: paramsHash}
		if err := saveCacheEntry(abs.OutputRoot, key, audioPath, transcriptPath, subtitlesPath, audioHash, transcriptHash, subtitleHash); err != nil {
			// Cache persistence is optimization only: a failure to save must
			// never fail an otherwise-successful run. The next invocation will
			// simply re-run from scratch.
			logf(logWriter, "pagevideo: cache save skipped: %v", err)
		}
	}
	return result, nil
}

// hashDependency resolves the dependency path and hashes the file. Returned
// hash participates in the cache params key, so swapping an ffmpeg build or a
// whisper model invalidates previously cached runs.
func hashDependency(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return fileHash(abs)
}

// saveCacheEntry mirrors the produced artifacts into OutputRoot/.cache so a
// future run with the same input+params can skip ffmpeg/whisper. The manifest
// file is not copied: on a hit the caller recomputes and rewrites it with the
// new run id and new absolute paths.
func saveCacheEntry(outputRoot string, key cache.Key, audioPath, transcriptPath, subtitlesPath, audioHash, transcriptHash, subtitleHash string) error {
	cacheDir := cache.Dir(outputRoot, key)
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return err
	}
	type srcDst struct{ src, name string }
	for _, pair := range []srcDst{
		{audioPath, "audio.wav"},
		{transcriptPath, "transcript.txt"},
		{subtitlesPath, "transcript.srt"},
	} {
		if err := copyOne(pair.src, filepath.Join(cacheDir, pair.name)); err != nil {
			return err
		}
	}
	return cache.SaveMetadata(cacheDir, key, audioHash, transcriptHash, subtitleHash)
}

func copyOne(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// finishFromCache builds a fresh Result from a validated cache directory: read
// transcript/subtitles, re-split chunks (deterministic, depends only on
// transcript bytes + chunk params + runID), optionally summarize, then write
// the manifest for the new run.
func finishFromCache(ctx context.Context, cfg config.Config, abs config.Config, runID, runRoot, inputHash string) (Result, error) {
	transcriptPath := filepath.Join(runRoot, "transcript.txt")
	subtitlesPath := filepath.Join(runRoot, "transcript.srt")
	transcript, err := os.ReadFile(transcriptPath)
	if err != nil {
		return Result{}, fmt.Errorf("cached transcript unreadable: %w", err)
	}
	transcriptHash := hashBytes(transcript)
	subtitleHash, err := fileHash(subtitlesPath)
	if err != nil {
		return Result{}, fmt.Errorf("cached subtitles unreadable: %w", err)
	}
	audioPath := filepath.Join(runRoot, "audio.wav")
	audioHash, err := fileHash(audioPath)
	if err != nil {
		return Result{}, fmt.Errorf("cached audio unreadable: %w", err)
	}
	chunks, err := chunk.Split(string(transcript), inputHash[:16], inputHash, runID, abs.ChunkChars, abs.OverlapWords)
	if err != nil {
		return Result{}, fmt.Errorf("re-split cached transcript: %w", err)
	}
	result := Result{
		RunID:      runID,
		Status:     "READY",
		Input:      Artifact{Path: abs.Input, Hash: inputHash},
		Audio:      Artifact{Path: audioPath, Hash: audioHash},
		Transcript: Artifact{Path: transcriptPath, Hash: transcriptHash},
		Subtitles:  Artifact{Path: subtitlesPath, Hash: subtitleHash},
		Chunks:     chunks,
	}
	if err := maybeArtifacts(ctx, cfg, string(transcript), runRoot, &result); err != nil {
		logf(nil, "pagevideo: artifact generation skipped: %v", err)
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

// artifactGenerator names a single LLM-produced Markdown file together with
// the LLM call that produces its body. Adding a new generated artifact is a
// matter of appending one struct literal — no per-kind plumbing in the
// pipeline itself.
type artifactGenerator struct {
	filename string
	generate func(ctx context.Context, c *llm.Client, transcript string) (string, error)
	assign   func(r *Result, a *Artifact)
}

var artifactGenerators = []artifactGenerator{
	{"summary.md", func(ctx context.Context, c *llm.Client, t string) (string, error) { return c.SummarizeTranscript(ctx, t) }, func(r *Result, a *Artifact) { r.Summary = a }},
	{"study.md", func(ctx context.Context, c *llm.Client, t string) (string, error) { return c.GenerateStudy(ctx, t) }, func(r *Result, a *Artifact) { r.Study = a }},
	{"faq.md", func(ctx context.Context, c *llm.Client, t string) (string, error) { return c.GenerateFAQ(ctx, t) }, func(r *Result, a *Artifact) { r.FAQ = a }},
	{"glossary.md", func(ctx context.Context, c *llm.Client, t string) (string, error) { return c.GenerateGlossary(ctx, t) }, func(r *Result, a *Artifact) { r.Glossary = a }},
}

// maybeArtifacts runs each generator against the local Bionic chat endpoint
// when cfg.EnableSummary is set. A per-artifact failure is logged and the
// artifact is skipped — it never fails the run, matching the policy that the
// pipeline output is useful even without any LLM involvement. When
// EnableSummary is false, this is a no-op (returns nil).
func maybeArtifacts(ctx context.Context, cfg config.Config, transcript, runRoot string, result *Result) error {
	if !cfg.EnableSummary {
		return nil
	}
	client, err := llm.NewBionicClient(llm.Config{
		BaseURL:          cfg.LLMBaseURL,
		Timeout:          cfg.LLMTimeout,
		MaxResponseBytes: cfg.LLMMaxResponseBytes,
		AllowChat:        true, // reached only when EnableSummary gate passed
	}, nil)
	if err != nil {
		return fmt.Errorf("llm client: %w", err)
	}
	var firstErr error
	for _, gen := range artifactGenerators {
		body, err := gen.generate(ctx, client, transcript)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", gen.filename, err)
			}
			continue
		}
		path := filepath.Join(runRoot, gen.filename)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("write %s: %w", gen.filename, err)
			}
			continue
		}
		hash, err := fileHash(path)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("hash %s: %w", gen.filename, err)
			}
			continue
		}
		gen.assign(result, &Artifact{Path: path, Hash: hash})
	}
	return firstErr
}

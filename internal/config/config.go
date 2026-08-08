package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Input         string
	OutputRoot    string
	FFmpeg        string
	Whisper       string
	Model         string
	Language      string
	ChunkChars    int
	OverlapWords  int
	Timeout       time.Duration
	MaxInputBytes int64
	// LLM / summary options. Summary generation is opt-in: it sends the
	// transcript to a provider and therefore needs an explicit operator gate.
	EnableSummary       bool
	LLMBaseURL          string
	LLMTimeout          time.Duration
	LLMMaxResponseBytes int64
	SummaryMaxChars     int
	// Cache controls reuse of previously completed runs. When false the run
	// directory under OutputRoot/.cache is ignored and the pipeline recomputes
	// from scratch. Default true (set by the CLI layer); Config.Validate does
	// not depend on it.
	UseCache bool
	// AllowDownload and YtDlp control the URL input path. AllowDownload is
	// the operator gate: without it Validate rejects http(s) inputs. YtDlp is
	// the path to the yt-dlp binary; it is only required when AllowDownload
	// is true and the input is a URL.
	AllowDownload bool
	YtDlp         string
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Input) == "" {
		return errors.New("input path is empty")
	}
	isURL := strings.HasPrefix(c.Input, "http://") || strings.HasPrefix(c.Input, "https://")
	if isURL {
		if !c.AllowDownload {
			return fmt.Errorf("URL input (%q) requires --allow-download (and --ytdlp PATH if yt-dlp is not at the default tools/yt-dlp.exe); remote download is opt-in", c.Input)
		}
		if strings.TrimSpace(c.YtDlp) == "" {
			return errors.New("--ytdlp is required when --allow-download is set")
		}
		info, err := os.Stat(c.YtDlp)
		if err != nil {
			return fmt.Errorf("stat ytdlp: %w", err)
		}
		if !info.Mode().IsRegular() {
			return errors.New("ytdlp must be a regular file")
		}
		// Skip local-file validations; size/extension checks apply *after*
		// the download completes inside the pipeline.
		if c.MaxInputBytes <= 0 {
			return errors.New("max input size must be positive")
		}
	} else {
		inputInfo, err := os.Stat(c.Input)
		if err != nil {
			return fmt.Errorf("stat input: %w", err)
		}
		if !inputInfo.Mode().IsRegular() {
			return errors.New("input must be a regular file")
		}
		if inputInfo.Size() == 0 {
			return errors.New("input is empty")
		}
		if c.MaxInputBytes <= 0 {
			return errors.New("max input size must be positive")
		}
		if inputInfo.Size() > c.MaxInputBytes {
			return fmt.Errorf("input exceeds max-input-bytes: %d > %d", inputInfo.Size(), c.MaxInputBytes)
		}
		ext := strings.ToLower(filepath.Ext(c.Input))
		allowed := map[string]bool{
			".avi": true,
			".mkv": true,
			".mov": true,
			".mp4": true,
		}
		if !allowed[ext] {
			return fmt.Errorf("unsupported input extension %q", ext)
		}
	}
	for name, path := range map[string]string{
		"ffmpeg":  c.FFmpeg,
		"whisper": c.Whisper,
		"model":   c.Model,
	} {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return fmt.Errorf("stat %s: %w", name, statErr)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", name)
		}
	}
	if strings.TrimSpace(c.Language) == "" {
		return errors.New("language is empty")
	}
	if c.ChunkChars < 100 {
		return errors.New("chunk-chars must be at least 100")
	}
	if c.OverlapWords < 0 {
		return errors.New("chunk-overlap-words cannot be negative")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if c.EnableSummary {
		if strings.TrimSpace(c.LLMBaseURL) == "" {
			return errors.New("llm-base-url is required when --enable-summary is set")
		}
		if c.LLMTimeout <= 0 {
			c.LLMTimeout = 60 * time.Second
		}
		if c.LLMMaxResponseBytes <= 0 {
			c.LLMMaxResponseBytes = 2 * 1024 * 1024
		}
		if c.SummaryMaxChars <= 0 {
			c.SummaryMaxChars = 24000
		}
	}
	// UseCache defaults to true; --no-cache is the only way to disable it.
	// There is no Validate-time work for the cache itself: a corrupt or stale
	// entry is treated as a miss, never as an error.
	return nil
}

func (c Config) AbsolutePaths() (Config, error) {
	// URL inputs must survive path resolution untouched: filepath.Abs would
	// mangle "https://..." into a bogus filesystem path. The pipeline's
	// download stage consumes the URL before any path operation.
	isURL := strings.HasPrefix(c.Input, "http://") || strings.HasPrefix(c.Input, "https://")
	paths := []*string{&c.OutputRoot, &c.FFmpeg, &c.Whisper, &c.Model}
	if !isURL {
		paths = append([]*string{&c.Input}, paths...)
	}
	for _, path := range paths {
		absolute, err := filepath.Abs(*path)
		if err != nil {
			return Config{}, fmt.Errorf("absolute path: %w", err)
		}
		*path = absolute
	}
	return c, nil
}

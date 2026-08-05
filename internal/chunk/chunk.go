package chunk

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"
)

type Chunk struct {
	ChunkID     string `json:"chunk_id"`
	Ordinal     int    `json:"ordinal"`
	Text        string `json:"text"`
	ContentHash string `json:"content_hash"`
	SourceID    string `json:"source_id"`
	SourceHash  string `json:"source_hash"`
	TrustClass  string `json:"trust_class"`
	RunID       string `json:"run_id"`
}

func Split(text, sourceID, sourceHash, runID string, maxChars, overlapWords int) ([]Chunk, error) {
	if maxChars < 1 {
		return nil, errors.New("maxChars must be positive")
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []Chunk{}, nil
	}
	if overlapWords >= len(words) && len(words) > 0 {
		overlapWords = len(words) - 1
	}

	chunks := []Chunk{}
	start := 0
	for start < len(words) {
		end := start
		length := 0
		for end < len(words) {
			candidate := words[end]
			extra := utf8.RuneCountInString(candidate)
			if length > 0 {
				extra++
			}
			if length+extra > maxChars && end > start {
				break
			}
			length += extra
			end++
		}
		if end == start {
			end++
		}
		chunkText := strings.Join(words[start:end], " ")
		contentHash := hash(chunkText)
		chunks = append(chunks, Chunk{
			ChunkID:     "chunk-" + contentHash[:16],
			Ordinal:     len(chunks),
			Text:        chunkText,
			ContentHash: contentHash,
			SourceID:    sourceID,
			SourceHash:  sourceHash,
			TrustClass:  "untrusted_source",
			RunID:       runID,
		})
		if end == len(words) {
			break
		}
		next := end - overlapWords
		if next <= start {
			next = end
		}
		start = next
	}
	return chunks, nil
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

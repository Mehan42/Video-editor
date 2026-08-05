package chunk

import "testing"

func TestSplit(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		maxChars   int
		overlap    int
		wantChunks int
	}{
		{name: "empty", text: "", maxChars: 100, overlap: 0, wantChunks: 0},
		{name: "single", text: "one two three", maxChars: 100, overlap: 1, wantChunks: 1},
		{name: "split", text: "one two three four five six", maxChars: 12, overlap: 1, wantChunks: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Split(tt.text, "source", "hash", "run", tt.maxChars, tt.overlap)
			if err != nil {
				t.Fatalf("Split() error = %v", err)
			}
			if len(got) != tt.wantChunks {
				t.Fatalf("Split() chunks = %d, want %d", len(got), tt.wantChunks)
			}
			for _, item := range got {
				if item.TrustClass != "untrusted_source" || item.ContentHash == "" {
					t.Fatalf("chunk provenance missing: %+v", item)
				}
			}
		})
	}
}

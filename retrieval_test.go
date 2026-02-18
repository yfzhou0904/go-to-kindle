package main

import "testing"

func TestNormalizeLocalPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes surrounding single quotes",
			input: "'/tmp/hello world.html'",
			want:  "/tmp/hello world.html",
		},
		{
			name:  "unescapes common shell escaped path",
			input: "/tmp/hello\\ world\\.html",
			want:  "/tmp/hello world.html",
		},
		{
			name:  "unescapes escaped pipe",
			input: "/tmp/a\\|b.html",
			want:  "/tmp/a|b.html",
		},
		{
			name:  "unescapes escaped slash",
			input: "\\/tmp\\/foo\\/bar.html",
			want:  "/tmp/foo/bar.html",
		},
		{
			name:  "returns trimmed input when shell parse fails",
			input: "  '/tmp/unclosed  ",
			want:  "'/tmp/unclosed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeLocalPath(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeLocalPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

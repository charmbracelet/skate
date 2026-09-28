package main

import (
	"strings"
	"testing"
)

func TestPreviewValue(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "short single line is unchanged",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "multiline short value escapes newlines",
			in:   "line1\nline2\nline3",
			want: `line1\nline2\nline3`,
		},
		{
			name: "carriage return is escaped too",
			in:   "a\r\nb",
			want: `a\r\nb`,
		},
		{
			name: "long single line is truncated with suffix",
			in:   strings.Repeat("a", previewLen+10),
			want: strings.Repeat("a", previewLen) + "  (10 more chars)",
		},
		{
			name: "long multiline value is truncated and escaped",
			in:   "#!/bin/sh\n" + strings.Repeat("x", previewLen),
			// The first previewLen chars are "#!/bin/sh\n" + 70 "x"s, which after
			// escaping the newline becomes "#!/bin/sh\\n" + 70 "x"s.
			want: `#!/bin/sh\n` + strings.Repeat("x", previewLen-len("#!/bin/sh\n")) + "  (10 more chars)",
		},
		{
			name: "exactly previewLen single-line value is not truncated",
			in:   strings.Repeat("a", previewLen),
			want: strings.Repeat("a", previewLen),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(previewValue([]byte(tt.in)))
			if got != tt.want {
				t.Errorf("previewValue() got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPreviewValue_BinaryIsUnchanged(t *testing.T) {
	// Non-UTF-8 input: the binary detection in printFromKV is what masks
	// these, previewValue should leave them alone so that path keeps working.
	binary := []byte{0xff, 0xfe, 0xfd, 0x00, 0x01}
	got := previewValue(binary)
	if string(got) != string(binary) {
		t.Errorf("previewValue mangled binary input: %v -> %v", binary, got)
	}
}

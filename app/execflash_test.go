package app

import (
	"bytes"
	"testing"
)

// Tests for ScanCROrLFLines and dropCROrLF.
// The tests use a byte-slice expected token where `nil` means the
// splitter should return a nil token (i.e. request more data or no token).
func TestScanCROrLFLines(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       string
		atEOF       bool
		wantAdvance int
		wantToken   []byte // nil means we expect a nil token
	}{
		{
			name:        "splits text starting with \\r",
			input:       "\rfoo",
			atEOF:       false,
			wantAdvance: 1,
			wantToken:   []byte(""),
		},
		{
			name:        "splits text starting with \\n",
			input:       "\nfoo",
			atEOF:       false,
			wantAdvance: 1,
			wantToken:   []byte(""),
		},
		{
			name:        "splits text with \\r midway",
			input:       "abc\rdef",
			atEOF:       false,
			wantAdvance: 4,
			wantToken:   []byte("abc"),
		},
		{
			name:        "splits text with \\n midway",
			input:       "abc\ndef",
			atEOF:       false,
			wantAdvance: 4,
			wantToken:   []byte("abc"),
		},
		{
			name:        "splits text ending with \\r at EOF",
			input:       "abc\r",
			atEOF:       true,
			wantAdvance: 4,
			wantToken:   []byte("abc"),
		},
		{
			name:        "splits text ending with \\n at EOF",
			input:       "abc\n",
			atEOF:       true,
			wantAdvance: 4,
			wantToken:   []byte("abc"),
		},
		{
			name:        "final non-terminated line at EOF",
			input:       "lastline",
			atEOF:       true,
			wantAdvance: 8,
			wantToken:   []byte("lastline"),
		},
		{
			name:        "incomplete data no EOF requests more data",
			input:       "incomplete",
			atEOF:       false,
			wantAdvance: 0,
			wantToken:   nil, // expect nil token because function should request more data
		},
		{
			name:  "handles CRLF pair",
			input: "abc\r\ndef",
			atEOF: false,
			// advance should be index of '\n' + 1, i.e. 4 + 1 = 5
			wantAdvance: 5,
			wantToken:   []byte("abc"),
		},
		{
			name:        "empty input at EOF returns nil",
			input:       "",
			atEOF:       true,
			wantAdvance: 0,
			wantToken:   nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			advance, token, err := ScanCROrLFLines([]byte(testCase.input), testCase.atEOF)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if advance != testCase.wantAdvance {
				t.Fatalf("expected advance %d, got %d", testCase.wantAdvance, advance)
			}
			if !bytes.Equal(token, testCase.wantToken) {
				gotStr := "<nil>"
				if token != nil {
					gotStr = string(token)
				}
				wantStr := "<nil>"
				if testCase.wantToken != nil {
					wantStr = string(testCase.wantToken)
				}
				t.Fatalf("expected token %v, got %v", wantStr, gotStr)
			}
		})
	}
}

func TestDropCR(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  []byte
		output []byte
	}{
		{"no terminator", []byte("abc"), []byte("abc")},
		{"trailing CR", []byte("abc\r"), []byte("abc")},
		{"empty slice", []byte(""), []byte("")},
		{"single CR", []byte("\r"), []byte("")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := dropCR(testCase.input)
			if !bytes.Equal(got, testCase.output) {
				t.Fatalf("expected %q, got %q", string(testCase.output), string(got))
			}
		})
	}
}

package faherror

import (
	"encoding/json"
	"testing"
)

func TestFahError(t *testing.T) {
	tests := []struct {
		name    string
		err     *FahError
		wantLog string
	}{
		{
			name: "basic error",
			err: &FahError{
				Code:   "fah/basic",
				Reason: "Just a reason",
				RawLog: "A simple log",
			},
			wantLog: "A simple log",
		},
		{
			name: "unescaped quotes in raw log",
			err: &FahError{
				Code:   "fah/quotes",
				Reason: "Quotes reason",
				RawLog: `log with "double" and 'single' quotes`,
			},
			wantLog: `log with "double" and 'single' quotes`,
		},
		{
			name: "newlines and control chars",
			err: &FahError{
				Code:   "fah/control",
				Reason: "Control reason",
				RawLog: "line1\nline2\rline3\thorizontal",
			},
			wantLog: "line1\nline2\rline3\thorizontal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()

			// Verify it's valid JSON
			var parsed FahError
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Fatalf("Error() output is not valid JSON: %v\nOutput: %s", err, got)
			}

			// Verify RawLog is preserved exactly (because json.Marshal handled the escaping)
			if parsed.RawLog != tt.wantLog {
				t.Errorf("RawLog mismatch. Got: %q, Want: %q", parsed.RawLog, tt.wantLog)
			}
		})
	}
}

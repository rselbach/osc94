package osc94

import (
	"bytes"
	"io"
	"testing"
)

func TestEscapeWithTerminator(t *testing.T) {
	tests := map[string]struct {
		state      State
		percent    int
		terminator terminator
		want       string
	}{
		"clear_bel": {
			state:      StateClear,
			percent:    0,
			terminator: terminatorBEL,
			want:       "\x1b]9;4;0;0\a",
		},
		"normal_bel": {
			state:      StateNormal,
			percent:    42,
			terminator: terminatorBEL,
			want:       "\x1b]9;4;1;42\a",
		},
		"error_st": {
			state:      StateError,
			percent:    7,
			terminator: terminatorST,
			want:       "\x1b]9;4;2;7\x1b\\",
		},
		"indeterminate_st": {
			state:      StateIndeterminate,
			percent:    0,
			terminator: terminatorST,
			want:       "\x1b]9;4;3;0\x1b\\",
		},
		"warning_bel": {
			state:      StateWarning,
			percent:    99,
			terminator: terminatorBEL,
			want:       "\x1b]9;4;4;99\a",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			got, err := escapeWithTerminator(tc.state, tc.percent, tc.terminator)
			if err != nil {
				t.Fatalf("escapeWithTerminator() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("escapeWithTerminator() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEscapePercentValidation(t *testing.T) {
	tests := map[string]struct {
		state   State
		percent int
		wantErr bool
	}{
		"normal_low": {
			state:   StateNormal,
			percent: -1,
			wantErr: true,
		},
		"normal_ok": {
			state:   StateNormal,
			percent: 0,
			wantErr: false,
		},
		"normal_ok_max": {
			state:   StateNormal,
			percent: 100,
			wantErr: false,
		},
		"normal_high": {
			state:   StateNormal,
			percent: 101,
			wantErr: true,
		},
		"indeterminate_out_of_range": {
			state:   StateIndeterminate,
			percent: 123,
			wantErr: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			_, err := escapeWithTerminator(tc.state, tc.percent, terminatorBEL)
			if tc.wantErr && err == nil {
				t.Fatalf("escapeWithTerminator() expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("escapeWithTerminator() error = %v", err)
			}
		})
	}
}

func TestEscapeInvalidState(t *testing.T) {
	_, err := escapeWithTerminator(State(99), 0, terminatorBEL)
	if err == nil {
		t.Fatalf("escapeWithTerminator() expected error for invalid state")
	}
}

func TestTerminatorSequence(t *testing.T) {
	tests := map[string]struct {
		terminator terminator
		want       string
		wantErr    bool
	}{
		"bel": {
			terminator: terminatorBEL,
			want:       "\a",
			wantErr:    false,
		},
		"st": {
			terminator: terminatorST,
			want:       "\x1b\\",
			wantErr:    false,
		},
		"invalid": {
			terminator: terminator(99),
			want:       "",
			wantErr:    true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			got, err := terminatorSequence(tc.terminator)
			if tc.wantErr && err == nil {
				t.Fatalf("terminatorSequence() expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("terminatorSequence() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("terminatorSequence() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProgressSetEnabled(t *testing.T) {
	var buffer bytes.Buffer
	progress := New(&buffer, WithTerminatorST())

	if err := progress.Set(StateNormal, 25); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	want := "\x1b]9;4;1;25\x1b\\"
	if got := buffer.String(); got != want {
		t.Fatalf("Set() wrote %q, want %q", got, want)
	}
}

func TestProgressSetDisabled(t *testing.T) {
	var buffer bytes.Buffer
	progress := New(&buffer, WithEnabled(false))

	if err := progress.Set(StateNormal, 10); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if buffer.Len() != 0 {
		t.Fatalf("Set() wrote %q, want empty", buffer.String())
	}
}

func TestWithDetector(t *testing.T) {
	tests := map[string]struct {
		detector func(io.Writer) bool
		want     string
	}{
		"disabled": {
			detector: func(io.Writer) bool { return false },
			want:     "",
		},
		"enabled": {
			detector: func(io.Writer) bool { return true },
			want:     "\x1b]9;4;1;5\a",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var buffer bytes.Buffer
			progress := New(&buffer, WithDetector(tc.detector))

			if err := progress.Set(StateNormal, 5); err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			if got := buffer.String(); got != tc.want {
				t.Fatalf("Set() wrote %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDetectOverrides(t *testing.T) {
	tests := map[string]struct {
		disable string
		force   string
		want    bool
	}{
		"disable": {
			disable: "1",
			force:   "",
			want:    false,
		},
		"force": {
			disable: "",
			force:   "1",
			want:    true,
		},
		"disable_wins": {
			disable: "1",
			force:   "1",
			want:    false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Setenv("OSC94_DISABLE", tc.disable)
			t.Setenv("OSC94_FORCE", tc.force)

			if got := Detect(&bytes.Buffer{}); got != tc.want {
				t.Fatalf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetectTTYCheck(t *testing.T) {
	tests := map[string]struct {
		tty  bool
		want bool
	}{
		"tty_false": {
			tty:  false,
			want: false,
		},
		"tty_true": {
			tty:  true,
			want: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Setenv("OSC94_DISABLE", "")
			t.Setenv("OSC94_FORCE", "")
			t.Setenv("TERM", "xterm-256color")
			t.Setenv("WT_SESSION", "1")

			got := detect(&bytes.Buffer{}, func(io.Writer) bool {
				return tc.tty
			})
			if got != tc.want {
				t.Fatalf("detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsDumbTerm(t *testing.T) {
	tests := map[string]struct {
		term string
		want bool
	}{
		"dumb": {
			term: "dumb",
			want: true,
		},
		"dumb_spaced": {
			term: " DUMB ",
			want: true,
		},
		"xterm": {
			term: "xterm-256color",
			want: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Setenv("TERM", tc.term)

			if got := isDumbTerm(); got != tc.want {
				t.Fatalf("isDumbTerm() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHasOSC94SupportHint(t *testing.T) {
	tests := map[string]struct {
		env  map[string]string
		want bool
	}{
		"none": {
			env:  map[string]string{},
			want: false,
		},
		"wt_session": {
			env:  map[string]string{"WT_SESSION": "1"},
			want: true,
		},
		"conemu": {
			env:  map[string]string{"ConEmuANSI": "ON"},
			want: true,
		},
		"vte_version": {
			env:  map[string]string{"VTE_VERSION": "7001"},
			want: true,
		},
		"term_program": {
			env:  map[string]string{"TERM_PROGRAM": "vscode"},
			want: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			keys := []string{"WT_SESSION", "ConEmuANSI", "VTE_VERSION", "TERM_PROGRAM"}
			for _, key := range keys {
				t.Setenv(key, "")
			}
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			if got := hasOSC94SupportHint(); got != tc.want {
				t.Fatalf("hasOSC94SupportHint() = %v, want %v", got, tc.want)
			}
		})
	}
}

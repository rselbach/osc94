// Package osc94 provides OSC 9;4 progress reporting helpers.
//
// The OSC 9;4 sequence lets terminals show a progress indicator in tabs
// or taskbars. This package focuses on safe, opt-in output for CLI apps.
package osc94

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	minPercent = 0
	maxPercent = 100
)

// State represents the OSC 9;4 progress state.
//
// States map to the terminal-defined values used by Windows Terminal and
// other compatible emulators.
type State int

const (
	StateClear State = iota
	StateNormal
	StateError
	StateIndeterminate
	StateWarning
)

const (
	terminatorBEL terminator = iota
	terminatorST
)

type terminator int

// Progress writes OSC 9;4 sequences to an output stream.
type Progress struct {
	writer     io.Writer
	enabled    bool
	terminator terminator
}

// Option configures a Progress instance.
type Option func(*Progress)

// New returns a Progress writer bound to the provided output.
//
// The default behavior is enabled output with a BEL terminator.
func New(writer io.Writer, opts ...Option) *Progress {
	progress := &Progress{
		writer:     writer,
		enabled:    true,
		terminator: terminatorBEL,
	}

	for _, opt := range opts {
		opt(progress)
	}

	return progress
}

// WithEnabled forces progress output on or off.
func WithEnabled(enabled bool) Option {
	return func(progress *Progress) {
		progress.enabled = enabled
	}
}

// WithAutoEnable enables output only when Detect reports support.
func WithAutoEnable() Option {
	return func(progress *Progress) {
		progress.enabled = Detect(progress.writer)
	}
}

// WithDetector uses a custom detector to decide enablement.
func WithDetector(detector func(io.Writer) bool) Option {
	return func(progress *Progress) {
		progress.enabled = detector(progress.writer)
	}
}

// WithTerminatorBEL uses BEL (\a) to terminate OSC sequences.
func WithTerminatorBEL() Option {
	return func(progress *Progress) {
		progress.terminator = terminatorBEL
	}
}

// WithTerminatorST uses ST (ESC \\) to terminate OSC sequences.
func WithTerminatorST() Option {
	return func(progress *Progress) {
		progress.terminator = terminatorST
	}
}

// Set writes a progress update using the provided state and percentage.
//
// Percent must be 0-100 unless state is StateIndeterminate.
func (progress *Progress) Set(state State, percent int) error {
	if !progress.enabled {
		return nil
	}

	escape, err := escapeWithTerminator(state, percent, progress.terminator)
	if err != nil {
		return err
	}

	_, err = io.WriteString(progress.writer, escape)
	return err
}

// SetPercent updates progress using the normal state.
func (progress *Progress) SetPercent(percent int) error {
	return progress.Set(StateNormal, percent)
}

// Indeterminate switches to the indeterminate state.
func (progress *Progress) Indeterminate() error {
	return progress.Set(StateIndeterminate, 0)
}

// Error updates progress using the error state.
func (progress *Progress) Error(percent int) error {
	return progress.Set(StateError, percent)
}

// Warning updates progress using the warning state.
func (progress *Progress) Warning(percent int) error {
	return progress.Set(StateWarning, percent)
}

// Clear hides any active progress indicator.
func (progress *Progress) Clear() error {
	return progress.Set(StateClear, 0)
}

// Escape returns an OSC 9;4 sequence terminated with BEL.
func Escape(state State, percent int) (string, error) {
	return escapeWithTerminator(state, percent, terminatorBEL)
}

func escapeWithTerminator(state State, percent int, seqTerminator terminator) (string, error) {
	if state != StateIndeterminate && (percent < minPercent || percent > maxPercent) {
		return "", fmt.Errorf("osc94: percent %d out of range", percent)
	}

	if !isValidState(state) {
		return "", fmt.Errorf("osc94: invalid state %d", state)
	}

	terminatorValue, err := terminatorSequence(seqTerminator)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("\x1b]9;4;%d;%d%s", state, percent, terminatorValue), nil
}

// Detect reports whether OSC 9;4 support is likely available.
//
// The check is conservative: it requires a TTY, excludes TERM=dumb,
// and matches known terminal hints. OSC94_DISABLE=1 always disables
// output; OSC94_FORCE=1 always enables it.
func Detect(writer io.Writer) bool {
	return detect(writer, isTTY)
}

func detect(writer io.Writer, ttyCheck func(io.Writer) bool) bool {
	if os.Getenv("OSC94_DISABLE") == "1" {
		return false
	}

	if os.Getenv("OSC94_FORCE") == "1" {
		return true
	}

	if !ttyCheck(writer) {
		return false
	}

	if isDumbTerm() {
		return false
	}

	return hasOSC94SupportHint()
}

func isValidState(state State) bool {
	switch state {
	case StateClear, StateNormal, StateError, StateIndeterminate, StateWarning:
		return true
	default:
		return false
	}
}

func terminatorSequence(seqTerminator terminator) (string, error) {
	switch seqTerminator {
	case terminatorBEL:
		return "\a", nil
	case terminatorST:
		return "\x1b\\", nil
	default:
		return "", errors.New("osc94: unknown terminator")
	}
}

// isTTY returns true when the writer is a character device.
func isTTY(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// isDumbTerm reports whether TERM indicates a basic terminal.
func isDumbTerm() bool {
	term := strings.TrimSpace(os.Getenv("TERM"))
	return strings.EqualFold(term, "dumb")
}

// hasOSC94SupportHint checks environment hints for OSC 9;4 support.
func hasOSC94SupportHint() bool {
	if os.Getenv("WT_SESSION") != "" {
		return true
	}

	if strings.EqualFold(os.Getenv("ConEmuANSI"), "ON") {
		return true
	}

	if os.Getenv("VTE_VERSION") != "" {
		return true
	}

	termProgram := os.Getenv("TERM_PROGRAM")
	termPrograms := []string{
		"ghostty",
		"iTerm.app",
		"vscode",
		"vscode-insiders",
	}

	for _, candidate := range termPrograms {
		if strings.EqualFold(termProgram, candidate) {
			return true
		}
	}

	return false
}

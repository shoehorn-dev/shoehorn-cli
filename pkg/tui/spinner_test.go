package tui

import (
	"context"
	"errors"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestErrCancelled_MatchesContextCanceled: errors.Is(err, context.Canceled) must catch TTY cancellation too.
func TestErrCancelled_MatchesContextCanceled(t *testing.T) {
	if !errors.Is(ErrCancelled, context.Canceled) {
		t.Fatal("want ErrCancelled to satisfy errors.Is(_, context.Canceled)")
	}
}

func TestInterpretSpinnerResult_SignalInterrupted_ReturnsCancelled(t *testing.T) {
	_, err := interpretSpinnerResult(nil, tea.ErrInterrupted)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("want ErrCancelled for a real SIGINT, got %v", err)
	}
}

func TestInterpretSpinnerResult_SignalInterrupted_Wrapped_ReturnsCancelled(t *testing.T) {
	// bubbletea's Program.Run wraps a real SIGINT as ErrProgramKilled: ErrInterrupted.
	wrapped := fmt.Errorf("%w: %w", tea.ErrProgramKilled, tea.ErrInterrupted)
	_, err := interpretSpinnerResult(nil, wrapped)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("want ErrCancelled for a wrapped SIGINT, got %v", err)
	}
}

func TestInterpretSpinnerResult_QuitWithoutDone_ReturnsCancelled(t *testing.T) {
	// SIGTERM maps to bubbletea's QuitMsg, which Run returns as (model, nil).
	_, err := interpretSpinnerResult(spinnerModel{done: false}, nil)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("want ErrCancelled when the program quit before finishing, got %v", err)
	}
}

func TestInterpretSpinnerResult_NormalCompletion_Unaffected(t *testing.T) {
	result, err := interpretSpinnerResult(spinnerModel{done: true, result: "x"}, nil)
	if err != nil || result != "x" {
		t.Fatalf("got (%v, %v), want (\"x\", nil)", result, err)
	}

	boom := errors.New("boom")
	_, err = interpretSpinnerResult(spinnerModel{done: true, err: boom}, nil)
	if !errors.Is(err, boom) {
		t.Fatalf("want the fn's own error passed through unchanged, got %v", err)
	}
}

func TestInterpretSpinnerResult_UnexpectedModelType(t *testing.T) {
	_, err := interpretSpinnerResult(tableModel{}, nil)
	if err == nil {
		t.Fatal("want an error for an unexpected final model type")
	}
}

// TestSpinnerModel_CtrlC_SetsCancelledErr: a cancelled fetch must set a non-nil err.
func TestSpinnerModel_CtrlC_SetsCancelledErr(t *testing.T) {
	m := newSpinnerModel("test", func() (any, error) { return nil, nil })

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	sm, ok := updated.(spinnerModel)
	if !ok {
		t.Fatalf("Update returned %T, want spinnerModel", updated)
	}
	if sm.err == nil {
		t.Fatal("want err set after ctrl+c, got nil")
	}
	if !errors.Is(sm.err, ErrCancelled) {
		t.Fatalf("want ErrCancelled, got %v", sm.err)
	}
	if cmd == nil {
		t.Fatal("want a Quit command returned after ctrl+c")
	}
}

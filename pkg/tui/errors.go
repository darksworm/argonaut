package tui

import (
	"time"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
)

// UpdateAppErrorState records err as the current error and appends it to the
// bounded error history, scheduling auto-hide for low-severity errors.
func UpdateAppErrorState(state *model.AppState, err *apperrors.ArgonautError) {
	if state.ErrorState == nil {
		state.ErrorState = &model.ErrorState{
			History: make([]apperrors.ArgonautError, 0),
		}
	}

	state.ErrorState.Current = err
	// Clear any pending auto-hide from a previous low-severity error so it
	// can't hide the error we're recording now.
	state.ErrorState.AutoHideAt = nil

	if err != nil {
		state.ErrorState.History = append(state.ErrorState.History, *err)

		maxHistory := 20
		if len(state.ErrorState.History) > maxHistory {
			state.ErrorState.History = state.ErrorState.History[1:]
		}

		if err.Severity == apperrors.SeverityLow {
			autoHide := time.Now().Add(5 * time.Second)
			state.ErrorState.AutoHideAt = &autoHide
		}
	}
}

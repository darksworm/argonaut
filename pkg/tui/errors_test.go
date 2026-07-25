package tui

import (
	"testing"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
)

func TestUpdateAppErrorState_HighSeverityError_ClearsPendingAutoHide(t *testing.T) {
	state := &model.AppState{}

	low := apperrors.New(apperrors.ErrorAPI, "LOW", "transient hiccup")
	low.Severity = apperrors.SeverityLow
	UpdateAppErrorState(state, low)
	if state.ErrorState.AutoHideAt == nil {
		t.Fatal("low-severity error should schedule auto-hide")
	}

	high := apperrors.New(apperrors.ErrorAPI, "HIGH", "sync failed")
	high.Severity = apperrors.SeverityHigh
	UpdateAppErrorState(state, high)
	if state.ErrorState.AutoHideAt != nil {
		t.Error("high-severity error must not inherit the previous error's auto-hide deadline")
	}
}

package main

import (
	"testing"
)

func TestCommitValidateFlagLogic(t *testing.T) {
	tests := []struct {
		name    string
		task    string
		noTask  bool
		reason  string
		wantErr string
	}{
		{"both task and no-task", "t1", true, "", "mutually exclusive"},
		{"no-task without reason", "", true, "", "requires --reason"},
		{"neither task nor no-task", "", false, "", "requires --task"},
		{"valid task", "t1", false, "", ""},
		{"valid no-task with reason", "", true, "infra change", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommitFlags(tt.task, tt.noTask, tt.reason)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !scopeContains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

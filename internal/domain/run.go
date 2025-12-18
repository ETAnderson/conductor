package domain

type RunStatus string

const (
	RunStatusCompleted        RunStatus = "completed"
	RunStatusNoChangeDetected RunStatus = "no_change_detected"
	RunStatusHasChanges       RunStatus = "has_changes"
)

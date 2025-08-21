package collector

import "GWatch/internal/entity"

// ProcessInspector defines capabilities to query top CPU/memory processes.
type ProcessInspector interface {
	GetTopProcesses(n int) ([]entity.ProcessInfo, []entity.ProcessInfo, error)
}



package entity

// ProcessInfo represents a process usage snapshot
type ProcessInfo struct {
	PID        int32
	Name       string
	CPUPercent float64
	MemPercent float32
	MemRSS     uint64 // MB
}



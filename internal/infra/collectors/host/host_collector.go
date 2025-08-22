package host

import (
	"fmt"
	"sort"
	"time"

	"GWatch/internal/entity"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Collector implements host-level metric collection using gopsutil.
type Collector struct{}

func New() *Collector { return &Collector{} }

func (c *Collector) GetCPUPercent() (float64, error) {
	percent, err := cpu.Percent(0, true)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, p := range percent {
		total += p
	}
	if len(percent) == 0 {
		return 0, nil
	}
	return total / float64(len(percent)), nil
}

func (c *Collector) GetMemoryUsage() (float64, uint64, uint64, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, 0, err
	}
	return vm.UsedPercent, vm.Used / 1024 / 1024, vm.Total / 1024 / 1024, nil
}

func (c *Collector) GetDiskUsage() (float64, uint64, uint64, error) {
	usage, err := disk.Usage("/")
	if err != nil {
		return 0, 0, 0, err
	}
	return usage.UsedPercent, usage.Used / 1024 / 1024 / 1024, usage.Total / 1024 / 1024 / 1024, nil
}

var lastNetIO *net.IOCountersStat
var lastNetTime time.Time

func init() {
	counters, err := net.IOCounters(false)
	if err == nil && len(counters) > 0 {
		lastNetIO = &counters[0]
	}
	lastNetTime = time.Now()
}

func (c *Collector) GetNetworkRate() (float64, float64, error) {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return 0, 0, err
	}
	now := time.Now()
	curr := counters[0]
	elapsed := now.Sub(lastNetTime).Seconds()
	if elapsed <= 0 || lastNetIO == nil {
		lastNetIO = &curr
		lastNetTime = now
		return 0, 0, nil
	}
	bytesRecv := float64(curr.BytesRecv - lastNetIO.BytesRecv)
	bytesSent := float64(curr.BytesSent - lastNetIO.BytesSent)
	dl := bytesRecv / elapsed / 1024
	ul := bytesSent / elapsed / 1024
	lastNetIO = &curr
	lastNetTime = now
	return dl, ul, nil
}

// GetTopProcesses returns top N processes by CPU and Memory
func (c *Collector) GetTopProcesses(n int) ([]entity.ProcessInfo, []entity.ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, nil, fmt.Errorf("无法获取 PID 列表: %w", err)
	}
	var processesList []*process.Process
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil || p == nil {
			continue
		}
		processesList = append(processesList, p)
	}
	var cpuList, memList []entity.ProcessInfo
	// sampling interval
	time.Sleep(300 * time.Millisecond)
	for _, p := range processesList {
		if p == nil {
			continue
		}
		cpuPercent, err := p.CPUPercent()
		if err != nil {
			continue
		}
		memInfo, err := p.MemoryInfo()
		if err != nil {
			continue
		}
		memPercent, err := p.MemoryPercent()
		if err != nil {
			continue
		}
		name, err := p.Name()
		if err != nil {
			name = "unknown"
		}
		info := entity.ProcessInfo{PID: p.Pid, Name: name, CPUPercent: cpuPercent, MemPercent: memPercent, MemRSS: memInfo.RSS / 1024 / 1024}
		if cpuPercent > 0.1 {
			cpuList = append(cpuList, info)
		}
		if memPercent > 0.1 {
			memList = append(memList, info)
		}
	}
	sort.Slice(cpuList, func(i, j int) bool { return cpuList[i].CPUPercent > cpuList[j].CPUPercent })
	sort.Slice(memList, func(i, j int) bool { return memList[i].MemPercent > memList[j].MemPercent })
	if len(cpuList) > n {
		cpuList = cpuList[:n]
	}
	if len(memList) > n {
		memList = memList[:n]
	}
	return cpuList, memList, nil
}

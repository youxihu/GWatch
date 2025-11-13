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

// 添加磁盘IO统计变量
var lastDiskIO *disk.IOCountersStat
var lastDiskTime time.Time

func init() {
	counters, err := net.IOCounters(false)
	if err == nil && len(counters) > 0 {
		lastNetIO = &counters[0]
	}
	lastNetTime = time.Now()
	
	// 初始化磁盘IO统计
	diskCounters, err := disk.IOCounters()
	if err == nil && len(diskCounters) > 0 {
		// 获取第一个磁盘的统计信息（通常是系统盘）
		for name, counter := range diskCounters {
			if name == "sda" || name == "nvme0n1" || name == "vda" {
				lastDiskIO = &counter
				break
			}
		}
		// 如果没有找到特定名称的磁盘，使用第一个
		if lastDiskIO == nil {
			for _, counter := range diskCounters {
				lastDiskIO = &counter
				break
			}
		}
	}
	lastDiskTime = time.Now()
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

// GetDiskIORate returns disk read/write rate in KB/s
// 该方法需要至少1秒的时间间隔才能准确计算速率
// 如果时间间隔不足，会返回上次的速率或0
func (c *Collector) GetDiskIORate() (float64, float64, error) {
	diskCounters, err := disk.IOCounters()
	if err != nil || len(diskCounters) == 0 {
		return 0, 0, err
	}
	
	now := time.Now()
	var curr disk.IOCountersStat
	
	// 尝试找到与上次相同的磁盘
	if lastDiskIO != nil {
		for _, counter := range diskCounters {
			if counter.Name == lastDiskIO.Name {
				curr = counter
				break
			}
		}
	}
	
	// 如果没找到，使用第一个磁盘
	if curr.Name == "" {
		for _, counter := range diskCounters {
			curr = counter
			break
		}
	}
	
	elapsed := now.Sub(lastDiskTime).Seconds()
	
	// 如果这是第一次调用，或者时间间隔不足1秒，更新基准值并返回0
	// 需要至少1秒的时间间隔才能准确计算速率
	const minIntervalSeconds = 1.0
	if elapsed < minIntervalSeconds || lastDiskIO == nil {
		lastDiskIO = &curr
		lastDiskTime = now
		return 0, 0, nil
	}
	
	// 计算时间差内的IO变化量
	bytesRead := float64(curr.ReadBytes - lastDiskIO.ReadBytes)
	bytesWrite := float64(curr.WriteBytes - lastDiskIO.WriteBytes)
	
	// 处理计数器回绕的情况（虽然不太可能，但需要处理）
	if bytesRead < 0 {
		bytesRead = 0
	}
	if bytesWrite < 0 {
		bytesWrite = 0
	}
	
	// 基于时间差计算速率（KB/s）
	readRate := bytesRead / elapsed / 1024
	writeRate := bytesWrite / elapsed / 1024
	
	// 调试日志：如果速率为0但时间间隔足够，记录详细信息
	if readRate == 0 && writeRate == 0 && elapsed >= minIntervalSeconds {
		// 只在调试时输出，避免日志过多
		// fmt.Printf("[GetDiskIORate] 调试: elapsed=%.2fs, curr.ReadBytes=%d, lastDiskIO.ReadBytes=%d, curr.WriteBytes=%d, lastDiskIO.WriteBytes=%d, disk=%s\n",
		// 	elapsed, curr.ReadBytes, lastDiskIO.ReadBytes, curr.WriteBytes, lastDiskIO.WriteBytes, curr.Name)
	}
	
	// 更新基准值
	lastDiskIO = &curr
	lastDiskTime = now
	
	return readRate, writeRate, nil
}

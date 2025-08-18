// internal/monitor/network.go
package monitor

import (
	"time"

	"github.com/shirou/gopsutil/v3/net"
)

var lastNetIO *net.IOCountersStat
var lastNetTime time.Time

func init() {
	var err error
	counters, err := net.IOCounters(false)
	if err == nil && len(counters) > 0 {
		lastNetIO = &counters[0]
	}
	lastNetTime = time.Now()
}

// GetNetworkRate 获取网络上传/下载速率（KB/s）
func GetNetworkRate() (float64, float64, error) {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return 0, 0, err
	}
	now := time.Now()
	curr := counters[0]

	// 时间差（秒）
	elapsed := now.Sub(lastNetTime).Seconds()
	if elapsed <= 0 {
		return 0, 0, nil
	}

	// 字节差
	bytesRecv := float64(curr.BytesRecv - lastNetIO.BytesRecv)
	bytesSent := float64(curr.BytesSent - lastNetIO.BytesSent)

	// KB/s
	downloadRate := bytesRecv / elapsed / 1024
	uploadRate := bytesSent / elapsed / 1024

	// 更新
	lastNetIO = &curr
	lastNetTime = now

	return downloadRate, uploadRate, nil
}

// Package scheduled_push internal/infra/scheduled_push/file_alert_storage.go
package scheduled_push

import (
	"GWatch/internal/domain/scheduled_push"
	"GWatch/internal/entity"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileAlertStorage 文件告警存储实现
type FileAlertStorage struct {
	config *entity.ScheduledPushAlertStorageConfig
}

// NewFileAlertStorage 创建文件告警存储
func NewFileAlertStorage(config *entity.ScheduledPushAlertStorageConfig) scheduled_push.ScheduledPushAlertStorage {
	return &FileAlertStorage{
		config: config,
	}
}

// SaveScheduledPushAlert 保存全局定时推送告警信息
func (f *FileAlertStorage) SaveScheduledPushAlert(alert *entity.ScheduledPushAlertRecord) error {
	if !f.config.Enabled {
		return nil
	}

	// 根据时间生成文件路径
	filePath := f.generateFilePath(alert.Timestamp, alert.PushTime)
	
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建告警日志目录失败: %v", err)
	}

	// 打开文件（追加模式）
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开告警日志文件失败: %v", err)
	}
	defer file.Close()

	// 根据格式保存
	var line string
	switch f.config.Format {
	case "json":
		jsonData, err := json.Marshal(alert)
		if err != nil {
			return fmt.Errorf("序列化告警记录失败: %v", err)
		}
		line = string(jsonData)
	case "text":
		line = f.formatAsText(alert)
	default:
		line = f.formatAsText(alert)
	}

	// 写入文件
	_, err = file.WriteString(line + "\n")
	if err != nil {
		return fmt.Errorf("写入告警日志失败: %v", err)
	}

	return nil
}

// GetScheduledPushAlerts 获取全局定时推送告警信息
func (f *FileAlertStorage) GetScheduledPushAlerts(startTime, endTime time.Time) ([]*entity.ScheduledPushAlertRecord, error) {
	if !f.config.Enabled {
		return nil, nil
	}

	var allAlerts []*entity.ScheduledPushAlertRecord
	
	// 遍历时间范围内的所有日期
	for d := startTime; d.Before(endTime) || d.Equal(endTime); d = d.AddDate(0, 0, 1) {
		// 生成该日期的目录路径
		year := d.Year() % 100
		month := int(d.Month())
		day := d.Day()
		dirPath := fmt.Sprintf("logs/%02d/%02d/%02d", year, month, day)
		
		// 读取该目录下的所有日志文件
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue // 目录不存在或无法读取，跳过
		}
		
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasPrefix(entry.Name(), "scheduled_push-") {
				continue
			}
			
			filePath := filepath.Join(dirPath, entry.Name())
			alerts, err := f.readAlertsFromFile(filePath, startTime, endTime)
			if err != nil {
				continue // 文件读取失败，跳过
			}
			
			allAlerts = append(allAlerts, alerts...)
		}
	}

	return allAlerts, nil
}

// readAlertsFromFile 从单个文件中读取告警信息
func (f *FileAlertStorage) readAlertsFromFile(filePath string, startTime, endTime time.Time) ([]*entity.ScheduledPushAlertRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var alerts []*entity.ScheduledPushAlertRecord
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var alert entity.ScheduledPushAlertRecord
		if err := json.Unmarshal([]byte(line), &alert); err != nil {
			continue // 跳过格式错误的行
		}

		// 检查时间范围
		if alert.Timestamp.After(startTime) && alert.Timestamp.Before(endTime) {
			alerts = append(alerts, &alert)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return alerts, nil
}

// CleanupOldAlerts 清理过期告警信息
func (f *FileAlertStorage) CleanupOldAlerts() error {
	if !f.config.Enabled || f.config.RetentionDays <= 0 {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -f.config.RetentionDays)
	
	// 遍历所有可能的日期目录
	baseDir := "logs"
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}
	
	for _, yearEntry := range entries {
		if !yearEntry.IsDir() {
			continue
		}
		
		yearPath := filepath.Join(baseDir, yearEntry.Name())
		monthEntries, err := os.ReadDir(yearPath)
		if err != nil {
			continue
		}
		
		for _, monthEntry := range monthEntries {
			if !monthEntry.IsDir() {
				continue
			}
			
			monthPath := filepath.Join(yearPath, monthEntry.Name())
			dayEntries, err := os.ReadDir(monthPath)
			if err != nil {
				continue
			}
			
			for _, dayEntry := range dayEntries {
				if !dayEntry.IsDir() {
					continue
				}
				
				dayPath := filepath.Join(monthPath, dayEntry.Name())
				
				// 解析日期
				var year, month, day int
				if _, err := fmt.Sscanf(dayEntry.Name(), "%02d", &day); err != nil {
					continue
				}
				if _, err := fmt.Sscanf(filepath.Base(monthPath), "%02d", &month); err != nil {
					continue
				}
				if _, err := fmt.Sscanf(filepath.Base(yearPath), "%02d", &year); err != nil {
					continue
				}
				
				// 年份需要加上2000
				fullYear := 2000 + year
				
				// 检查日期是否过期
				dirDate := time.Date(fullYear, time.Month(month), day, 0, 0, 0, 0, time.Local)
				if dirDate.Before(cutoffTime) {
					// 删除整个目录
					os.RemoveAll(dayPath)
				}
			}
		}
	}

	return nil
}

// generateFilePath 根据时间生成文件路径
func (f *FileAlertStorage) generateFilePath(timestamp time.Time, pushTime string) string {
	// 解析推送时间，获取小时和分钟
	hour := timestamp.Hour()
	minute := timestamp.Minute()
	
	// 如果推送时间格式为 "HH:MM"，则使用推送时间的小时和分钟
	if len(pushTime) >= 5 && pushTime[2] == ':' {
		if h, err := fmt.Sscanf(pushTime[:2], "%d", &hour); err == nil && h == 1 {
			// 成功解析小时
		}
		if m, err := fmt.Sscanf(pushTime[3:5], "%d", &minute); err == nil && m == 1 {
			// 成功解析分钟
		}
	}
	
	// 生成文件名：scheduled_push-HHMM.log
	fileName := fmt.Sprintf("scheduled_push-%02d%02d.log", hour, minute)
	
	// 使用模板生成路径：logs/年/月/日/scheduled_push-HHMM.log
	// 年份取后两位，月份和日期补零
	year := timestamp.Year() % 100
	month := int(timestamp.Month())
	day := timestamp.Day()
	
	// 替换模板中的占位符
	path := f.config.AlertLogPathTemplate
	path = strings.ReplaceAll(path, "%y", fmt.Sprintf("%02d", year))
	path = strings.ReplaceAll(path, "%m", fmt.Sprintf("%02d", month))
	path = strings.ReplaceAll(path, "%d", fmt.Sprintf("%02d", day))
	path = strings.ReplaceAll(path, "%H", fmt.Sprintf("%02d", hour))
	path = strings.ReplaceAll(path, "%M", fmt.Sprintf("%02d", minute))
	path = strings.ReplaceAll(path, "%s", fileName)
	
	return path
}

// formatAsText 将告警记录格式化为文本
func (f *FileAlertStorage) formatAsText(alert *entity.ScheduledPushAlertRecord) string {
	return fmt.Sprintf("[%s] %s - %s (推送时间: %s)",
		alert.Timestamp.Format("2006-01-02 15:04:05"),
		alert.Title,
		alert.Message,
		alert.PushTime,
	)
}

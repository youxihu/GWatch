// Package entity internal/entity/java_dump_result.go
package entity

// ScriptResult 脚本返回结果类型
type ScriptResult string

// 脚本返回的标准结果
const (
	ResultSuccess   ScriptResult = "success"    // 成功创建 dump
	ResultFileExist ScriptResult = "file_exist" // 文件已存在，跳过
	ResultFailed    ScriptResult = "failed"     // 执行失败
)

// ScriptResultText 脚本结果对应的中文描述（用于通知展示）
var ScriptResultText = map[ScriptResult]string{
	ResultSuccess:   "✅ Java堆转储已创建",
	ResultFileExist: "ℹ️ 堆转储文件已存在，跳过生成",
	ResultFailed:    "❌ Java堆转储创建失败",
}

// String 返回中文描述
func (r ScriptResult) String() string {
	if text, exists := ScriptResultText[r]; exists {
		return text
	}
	return "❓ 脚本返回未知结果"
}

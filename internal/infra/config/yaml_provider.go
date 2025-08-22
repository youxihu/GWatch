package configimpl

import (
	domain "GWatch/internal/domain/config"
	"GWatch/internal/entity"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLProvider 从指定路径加载 YAML 配置
type YAMLProvider struct {
	path string
	cfg  *entity.Config
}

func NewYAMLProvider(path string) (*YAMLProvider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}
	var c entity.Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}
	return &YAMLProvider{path: path, cfg: &c}, nil
}

func (p *YAMLProvider) GetConfig() *entity.Config { return p.cfg }

// 实现领域接口
var _ domain.Provider = (*YAMLProvider)(nil)

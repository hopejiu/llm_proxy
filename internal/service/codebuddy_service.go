package service

import (
	"encoding/json"
	"fmt"
	"llm-proxy/internal/model"
	"log/slog"
	"os"
	"path/filepath"
)

// CodeBuddyResult SetupCodeBuddy 的返回结果
type CodeBuddyResult struct {
	Message string
	Path    string
	Exists  bool
	Added   bool
	Models  int
}

// SetupCodeBuddy 配置 CodeBuddy 的 models.json
func SetupCodeBuddy(targetURL string) (*CodeBuddyResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("获取用户目录失败", "error", err)
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}

	codebuddyDir := filepath.Join(homeDir, ".codebuddy")
	modelsFilePath := filepath.Join(codebuddyDir, "models.json")

	if err := os.MkdirAll(codebuddyDir, 0755); err != nil {
		slog.Error("创建.codebuddy目录失败", "error", err)
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	var config model.CodeBuddyConfig

	if data, err := os.ReadFile(modelsFilePath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			slog.Warn("解析models.json失败", "error", err)
			config = model.CodeBuddyConfig{Models: []model.CodeBuddyModel{}}
		}
	} else {
		config = model.CodeBuddyConfig{Models: []model.CodeBuddyModel{}}
	}

	exists := false
	for _, m := range config.Models {
		if m.URL == targetURL {
			exists = true
			break
		}
	}

	if !exists {
		newModel := model.CodeBuddyModel{
			ID:                "astron-code-latest",
			Name:              "大模型",
			Vendor:            "自定义大模型",
			APIKey:            "miyao",
			URL:               targetURL,
			SupportsToolCall:  true,
			SupportsReasoning: true,
			Temperature:       0.1,
			MaxInputTokens:    128000,
		}
		config.Models = append(config.Models, newModel)
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(modelsFilePath, data, 0644); err != nil {
		slog.Error("写入models.json失败", "error", err)
		return nil, fmt.Errorf("写入配置文件失败: %w", err)
	}

	message := "配置文件已创建"
	if exists {
		message = "配置已存在，无需添加"
	}

	return &CodeBuddyResult{
		Message: message,
		Path:    modelsFilePath,
		Exists:  exists,
		Added:   !exists,
		Models:  len(config.Models),
	}, nil
}

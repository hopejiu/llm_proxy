package model

// CodeBuddyModel CodeBuddy models.json 中的模型结构
type CodeBuddyModel struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Vendor            string  `json:"vendor"`
	APIKey            string  `json:"apiKey"`
	URL               string  `json:"url"`
	SupportsToolCall  bool    `json:"supportsToolCall"`
	SupportsReasoning bool    `json:"supportsReasoning"`
	Temperature       float64 `json:"temperature"`
	MaxInputTokens    int     `json:"maxInputTokens"`
}

// CodeBuddyConfig CodeBuddy models.json 配置结构
type CodeBuddyConfig struct {
	Models []CodeBuddyModel `json:"models"`
}

package model

type History struct {
	AssistantID string    `json:"assistant_id"`
	Messages    []Message `json:"messages"`
}

type Message struct {
	Input     Input  `json:"input"`
	Output    Output `json:"output"`
	Usage     Usage  `json:"usage"`
	GmtCreate string `json:"gmt_create"`
}

type Input struct {
	Prompt string `json:"prompt"`
	Send   string `json:"send"`
}

type Output struct {
	FinishReason string `json:"finish_reason"`
	Content      string `json:"content"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

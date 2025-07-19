package model

type Assistant struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	GmtCreate   string `json:"gmt_create"`
	GmtModified string `json:"gmt_modified"`
	TimeStamp   string `json:"time_stamp"`
}

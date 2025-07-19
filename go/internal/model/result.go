package model

type Result struct {
	Success bool        `json:"success"`
	Msg     string      `json:"msg"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
}

package model

// APIResponse 服务器API响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// IsSuccess 检查响应是否成功
func (r *APIResponse) IsSuccess() bool {
	return r.Code == 200
}

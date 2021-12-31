package device

//断开连接请求
type DisconnectReq struct {
	ProductKey string `json:"productKey,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

//断开连接响应
type DisconnectRes struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

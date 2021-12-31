package shadow

import (
	"encoding/json"
	"time"
)

// 更新前的影子 & 更新后的影子
type UpdateDocument struct {
	Previous *ShadowData `json:"previous,omitempty"`
	Current  *ShadowData `json:"current,omitempty"`
}

// 设备影子Data
type ShadowData struct {
	Pid         string              `json:"-" bson:"pid"`
	Did         string              `json:"-" bson:"did"`
	State       DeviceState         `json:"state,omitempty" bson:"state"`
	MetaData    DeviceStateMetaData `json:"metadata,omitempty" bson:"metadata"`
	Timestamp   int64               `json:"timestamp,omitempty" bson:"timestamp"`
	ClientToken string              `json:"clientToken,omitempty" bson:"clientToken"`
	Version     int64               `json:"version,omitempty" json:"version"`
}

func (s ShadowData) String() string {
	str, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(str)
}

func NewShadowData() *ShadowData {
	return &ShadowData{Timestamp: time.Now().Unix()}
}

// State 设备状态
type DeviceState struct {
	// 期望状态
	Desired map[string]interface{} `json:"desired,omitempty"`
	// 上报状态
	Reported map[string]interface{} `json:"reported,omitempty"`
}

// 设备状态元数据/属性
type DeviceStateMetaData struct {
	// 期望状态属性
	Desired map[string]interface{} `json:"desired,omitempty"`
	// 上报状态属性
	Reported map[string]interface{} `json:"reported,omitempty"`
}

// 更新结果响应
type UpdateResp struct {
	// 响应结果错误码
	Code int32 `json:"code"`
	// 结果信息
	Message string `json:"message,omitempty"`
	// 响应时间
	Timestamp int64 `json:"timestamp,omitempty"`
	// 仅当客户端令牌用于向 /update 主题发布有效 JSON 时存在
	ClientToken string `json:"clientToken,omitempty"`
}

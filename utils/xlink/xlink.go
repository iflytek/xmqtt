package xlink

import "git.iflytek.com/HY_XIoT/core/utils/xferror"

const V1 = "1.0"
const (
	ActionUpload           = "upload"
	ActionUploadReply      = "upload_reply"
	ActionGetProperty      = "property.get"
	ActionGetPropertyReply = "property.get_reply"
	ActionSetProperty      = "property.set"
	ActionSetPropertyReply = "property.set_reply"
	ActionInvoke           = "invoke"
	ActionInvokeReply      = "invoke_reply"
)
const (
	ObjThingProperty = "thing.property"
	ObjThingEvent    = "thing.event"
	ObjThingService  = "thing.service"
)

const GroupId = "groupId"
const Mid = "mid"
const Version = "version"
const Obj = "obj"
const PKey = "pKey"
const DName = "dName"
const Action = "action"
const Event = "event"
const Timestamp = "timestamp"
const Identifer = "identifer"
const Payload = "payload"
const Data = "data"
const Params = "params"
const Ret = "ret"
const Msg = "msg"

//XLink协议
type XLink struct {
	Mid       string      `json:"mid,omitempty"`
	Version   string      `json:"version,omitempty"`
	Timestamp int64       `json:"timestamp"`
	PKey      string      `json:"pKey,omitempty"`
	DName     string      `json:"dName,omitempty"`
	Obj       string      `json:"obj,omitempty"`
	Action    string      `json:"action,omitempty"`
	Data      interface{} `json:"data"`
}

//data-上报属性
type UploadProperty struct {
	Params map[string]interface{} `json:"params,omitempty"`
}

//data-上报属性响应
type UploadPropertyReply struct {
	Ret int64  `json:"ret"`
	Msg string `json:"msg,omitempty"`
}

//data-上报事件
type UploadEvent struct {
	Event  string                 `json:"event,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
}

//data-上报事件响应
type UploadEventReply struct {
	Event string `json:"event,omitempty"`
	Ret   int64  `json:"ret"`
	Msg   string `json:"msg,omitempty"`
}

//data-获取设备属性
type ServiceGetProperty struct {
	Params []string `json:"params,omitempty"`
}

//data-获取设备属性响应
type ServiceGetPropertyReply struct {
	Ret     int64                  `json:"ret"`
	Msg     string                 `json:"msg,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

//data-设置设备属性
type ServiceSetProperty struct {
	Params map[string]interface{} `json:"params,omitempty"`
}

//data-设置设备属性响应
type ServiceSetPropertyReply struct {
	Ret int64  `json:"ret"`
	Msg string `json:"msg,omitempty"`
}

//data-调用指定服务
type ServiceInvoke struct {
	Identifer string                 `json:"identifer,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

//data-调用指定服务响应
type ServiceInvokeReply struct {
	Identifer string                 `json:"identifer,omitempty"`
	Ret       int64                  `json:"ret"`
	Msg       string                 `json:"msg,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

//协议解析
func (this *XLink) Parse(val []byte) (*XLink, int, error) {
	return nil, xerror.XCodeOK, nil
}

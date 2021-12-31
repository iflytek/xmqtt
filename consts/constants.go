package consts

//组件名称
const xmqtt = "xmqtt"

//设备行为日志
const LogType = "logType"
const LogContent = "content"
const LogReason = "reason"

const (
	LogTypeValDevice = "device"
)

const (
	LogReasonValTickOffDevice         = "被使用相同设备证书的设备踢下线或者设备端主动重连导致设备剔下线"
	LogReasonValServerClose           = "物联网平台服务端主动关闭连接"
	LogReasonValDeviceDisconnect      = "设备主动发送MQTT断开连接请求"
	LogReasonValAckTimeout            = "下发消息ACK超时"
	LogReasonValKeepaliveTimeout      = "心跳超时，物联网平台服务端关闭连接"
	LogReasonValConnectionResetByPeer = "对等端重置TCP连接"
)

package cnt

// ComponentList
const GateWay = "gateway"
const Dispatch = "dispatch"
const Push = "push"
const AuthServer = "authServer"
const UploadServer = "uploadServer"
const PostBack = "postback"
const MsgTransform = "msgTransform"
const LoadServer = "loadServer"
const XdnsServer = "xdnsServer"
const RemoteConfigServer = "remoteConfigServer"
const GroupServer = "groupServer"

// EventLog Common
const ComponentName = "componentName"
const IoT = "iot"
const MessageLevel = "messageLevel"
const ERR = "ERROR"
const INFO = "INFO"

// index
const (
	IndexRulePKeyAndDName       = "pKey|dName"
	IndexRuleMidAndPKeyAndDName = "mid|pKey|dName"
)

const Sid = "sid"

const GroupId = "groupId"

const AccountId = "accountId"

const ClientAddr = "clientAddr"
const ClientId = "clientId"
const Password = "password"
const NodeId = "nodeId"
const Route = "route"
const DeviceAuth = "auth"
const DeviceLogin = "login"
const DeviceLogout = "logout"

const Topic = "topic"

// EventLog xdnsServer
const ProtocolVersion = "protocolVersion"
const CVersion = "clientVersion"
const NetType = "netType"
const Operator = "operator"
const DNSSvc = "dnsSvc"
const HttpDNS = "httpDNS"

// EventLog Ret
const Ret = "ret"

// OSuffix output suffix
const OSuffix = "::output"

// ISuffix input suffix
const ISuffix = "::input"

// SPACE space
const SPACE = " "

// ThgSrv thing.service
const ThgSrv = "thing.service"

// Invoke
const Invoke = "thing.invoke"

var ActionDict = map[string]string{
	"invoke":       "invoke_reply",
	"property.get": "property.get_reply",
	"property.set": "property.set_reply",
	"upload":       "upload_reply",
}

// VERSION protocol version
const VERSION = "1.0"

// OBJ xlink protocol obj field
const OBJ = "thing.service"

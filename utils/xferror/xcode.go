package xerror

// Domain
const (
	XDomainDiagnose           = "x_domain_diagnose"
	XDomainGateway            = "x_domain_gateway"
	XDomainFeedback           = "x_domain_feedback"
	XDomainLoadServer         = "x_domain_loadserver"
	XDomainOTAServer          = "x_domain_otaserver"
	XDomainAuthServer         = "xf_domain_authServer"
	XDomainxmqtt           = "xf_domain_xmqtt"
	XDomainUploadServer       = "xf_domain_uploadServer"
	XDomainGroupServer        = "xf_domain_groupServer"
	XDomaindispatchServer     = "xf_domain_dispatchServer"
	XDomainpushServer         = "xf_domain_pushServer"
	XDomainxdnsServer         = "xf_domain_xdnsServer"
	XDomainmsgTransform       = "xf_domain_msgTransform"
	XDomainRemoteConfigServer = "xf_domain_remoteConfigServer"
	XDomainMetadataUpload     = "xf_domain_metadataUpload"
)

// Code
const (
	// Base
	XCodeFailure    = -1 // 请求失败
	XCodeFailureMsg = "Failure"

	XCodeOK    = 0 // 请求成功
	XCodeOKMsg = "Success"

	XCodeBadRequest    = 400 // 请求错误
	XCodeBadRequestMsg = "Bad Request"

	XCodeUnauthorized    = 401 // 请求认证错误，未授权
	XCodeUnauthorizedMsg = "Unauthorized"

	XCodeForbidden    = 403 // 请求被禁止
	XCodeForbiddenMsg = "Forbidden"

	XCodeNotFound    = 404 // 服务未找到
	XCodeNotFoundMsg = "Not Found"

	XCodeParamErr    = 460 // 请求参数错误
	XCodeParamErrMsg = "Request Parameter Error"

	XCodeParamExistToBeEmpty    = 461 // 请求参数存在为空
	XCodeParamExistToBeEmptyMsg = "Request Parameter Exist To Be Empty"

	XCodeParamTimestampErr = 462 // 请求参数时间戳错误

	XCodeServerError    = 500 // 服务端错误
	XCodeServerErrorMsg = "Server Error"

	XCodeServiceUnavailable    = 503 // 服务不可用
	XCodeServiceUnavailableMsg = "Service Unavailable"

	XCodeProductInactivated    = 600 // 产品未激活
	XCodeProductInactivatedMsg = "Product Inactivated"

	XCodeProductOrDeviceNotExist    = 601 // 产品或产品设备不存在
	XCodeProductOrDeviceNotExistMsg = "Product Or Device Does Not Exist"

	XCodeDeviceIsNull = 602 // 设备秘钥为空

	XCodeDeviceGroupIdNotExist    = 603 // 设备群组ID不存在
	XCodeDeviceGroupIdNotExistMsg = "Device GroupId Does Not Exist"

	XCodeSignatureMismatch    = 700 // 签名值不匹配
	XCodeSignatureMismatchMsg = "Signature Mismatch"

	XCodeNoData    = 800 // 无数据
	XCodeNoDataMsg = "No Data"

	XCodeOffline    = 801 // 无数据
	XCodeOfflineMsg = "Device Offline"

	XCodeNoMqttACK    = 802
	XCodeNoMqttACKMsg = "No Mqtt Ack"

	XCodeXLinkCheckErr    = 900
	XCodeXLinkCheckErrMsg = "协议解析失败"

	XCodeNotLogin    = 1001 // 用户未登录
	XCodeNotLoginMsg = "用户未登录，请登录后重试"

	XCodeTimeout    = 10000 // 请求超时
	XCodeTimeoutMsg = "Service Timeout"

	// RPC
	XCodeRPCRefused    = 11000 // RPC 连接被拒绝
	XCodeRPCRefusedMsg = "RPC Connection Refused"

	// MongoDB
	XCodeMgoRefused    = 12000 // MongoDB 连接被拒绝
	XCodeMgoRefusedMsg = "MongoDB Connection Refused"

	XCodeMgoQueryErr    = 12001 // MongoDB 查询错误
	XCodeMgoQueryErrMsg = "MongoDB Query Error"

	XCodeMgoInsertErr    = 12002 // MongoDB 插入错误
	XCodeMgoInsertErrMsg = "MongoDB Insert Error"

	XCodeMgoUpdateErr    = 12003 // MongoDB 更新错误
	XCodeMgoUpdateErrMsg = "MongoDB Update Error"

	XCodeMgoRemoveErr    = 12004 // MongoDB 删除错误
	XCodeMgoRemoveErrMsg = "MongoDB Remove Error"

	// JSON
	XCodeJSONErr    = 13000 // JSON相关错误码
	XCodeJSONErrMsg = "JSON Error"

	XCodeJSONSchemaErr    = 13001 // JSONSchema相关错误
	XCodeJSONSchemaErrMsg = "JSONSchema Error"

	XCodeSchemaRefused    = 13002 // 初始化Schema错误
	XCodeSchemaRefusedMsg = "Schema Connection Refused"

	// Redis
	XCodeRedisRefused    = 14000 // Redis 连接拒绝
	XCodeRedisRefusedMsg = "Redis Connection Refused"

	XCodeRedisGetErr    = 14001 // Redis 查询错误
	XCodeRedisGetErrMsg = "Redis Get Error"

	XCodeRedisSetErr    = 14002 // Redis 插入错误
	XCodeRedisSetErrMsg = "Redis Set Error"

	// NSQ
	XCodeNSQRefused    = 15000 // NSQ 连接拒绝
	XCodeNSQRefusedMsg = "NSQ Connection Refused"

	XCodeNSQProductErr    = 15001 // NSQ 生产错误
	XCodeNSQProductErrMsg = "NSQ Product Error"

	XCodeNSQConsumeErr    = 15002 // NSQ 消费错误
	XCodeNSQConsumeErrMsg = "NSQ Consume Error"

	// ProtocolBuffer
	XCodePBErr    = 16000 // ProtocolBuffer相关错误码
	XCodePBErrMsg = "ProtocolBuffer Error"

	// MySQL
	XCodeMySQLRefused    = 17000 // MySQL连接错误
	XCodeMySQLRefusedMsg = "MySQL Connection Refused"

	XCodeMySQLQueryErr    = 17001 // MySQL 查询错误
	XCodeMySQLQueryErrMsg = "MySQL Query Error"

	XCodeMySQLInsertErr    = 17002 // MySQL 插入错误
	XCodeMySQLInsertErrMsg = "MySQL Insert Error"

	XCodeMySQLUpdateErr    = 17003 // MySQL 更新错误
	XCodeMySQLUpdateErrMsg = "MySQL Update Error"

	// Mqtt protocol payload
	XCodePayloadAttributeInvalid    = 19000
	XCodePayloadAttributeInvalidMsg = "Mqtt Payload Attibute Invalid"

	// 调用计量服务异常
	RemoteOtherService = 202001

	//设备授权处于禁用状态，不能操作
	DeviceAuthStatusClose    = 202002
	DeviceAuthStatusCloseMsg = "设备是禁用状态，不能操作"
)

// Message
const (
	ProductKeyNotNull       = "产品key不能为空"
	DeviceNameNotNull       = "设备名称不能为空"
	MSG_DEVICE_NO_EXISTS    = "设备不存在"
	PARAMETER_IS_MISSING    = "参数缺失"
	OPERATETYPE_IS_EMPTY    = "请求类型不能为空"
	ORDERID_IS_EMPTY        = "订单ID不能为空"
	PRODUCTKEY_IS_EMPTY     = "产品key不能为空"
	CHANNEL_IS_EMPTY        = "AI能力不能为空"
	ORDERTIME_IS_EMPTY      = "订单时间不能为空"
	FUNCTION_IS_EMPTY       = "功能类型不能为空"
	ADDCOUNT_IS_EMPTY       = "授权时长不能为空"
	INVALIDTIME_IS_EMPTY    = "授权截止日期不能为空"
	MISS_DEVICE_AUTH        = "没有查到设备授权信息"
	MISS_APP_AUTH           = "没有查到应用授权信息"
	ADD_DEVICE_AUTH         = "新增设备授权信息异常"
	ADD_APP_AUTH            = "新增应用授权信息异常"
	ADD_AUTH_ERROR          = "新增权限记录异常"
	CHANNEL_MISS            = "没有查询到AI能力"
	DELETE_DEVICE_AUTH      = "删除设备权限异常"
	PRODUCT_DEVICE_NO_EQUAL = "产品和设备不匹配"
)

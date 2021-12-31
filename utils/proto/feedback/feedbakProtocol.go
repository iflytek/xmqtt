package feedback

type NotifyFeedbackRemoteConfigServerMessage struct {
	PKey        string      `json:"pKey,omitempty"`
	DName       string      `json:"dName,omitempty"`
	Transaction string      `json:"transaction,omityempty"`
	Payload     interface{} `json:"payload,omityempty"`
	Timestamp   int64       `json:"timestamp,omityempty"`
}

type PayloadMessage struct {
	Mid     string             `json:"mid,omitempty"`
	Version string             `json:"version,omitempty"`
	PKey    string             `json:"pKey,omitempty"`
	DName   string             `json:"dName,omitempty"`
	Obj     string             `json:"obj,omitempty"`
	Action  string             `json:"action,omitempty"`
	Data    DeviceDataAndEvent `json:"data"`
}

type DeviceDataAndEvent struct {
	Event     string                 `json:"event,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
}

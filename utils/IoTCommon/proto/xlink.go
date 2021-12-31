package proto

type XLink struct {
	Mid       string      `json:"mid,omitempty" bson:"mid"`
	Version   string      `json:"version,omitempty" bson:"version"`
	PKey      string      `json:"pKey,omitempty" bson:"productKey"`
	DName     string      `json:"dName,omitempty" bson:"deviceName"`
	Obj       string      `json:"obj,omitempty" bson:"obj"`
	Action    string      `json:"action,omitempty" bson:"action"`
	Timestamp int64       `json:"timestamp,omitempty"`
	Data      interface{} `json:"data,omitempty" bson:"data"`
}

// DBody mean download xlink body
type DBody struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

type RespProcess struct {
	Mid       string                 `from:"mid" json:"mid"`
	Version   string                 `from:"version" json:"version"`
	PKey      string                 `from:"pKey" json:"pKey"`
	DName     string                 `from:"dName" json:"dName"`
	Obj       string                 `from:"obj" json:"obj"`
	Action    string                 `from:"action" json:"action"`
	Timestamp int64                  `from:"timestamp" json:"timestamp"`
	Data      map[string]interface{} `from:"data" json:"data"`
}

package dispatch

import "encoding/json"

type Header struct {
	Mid     string          `name=mid json:"mid"`
	Version string          `name=version json:"version"`
	PKey    string          `json:"pKey"`
	DName   string          `json:"dName"`
	Obj     string          `json:"obj"`
	Action  string          `json:"action"`
	Data    json.RawMessage `json:"data"`
}

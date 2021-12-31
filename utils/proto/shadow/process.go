package shadow

import (
	"encoding/json"

	"git.iflytek.com/HY_XIoT/core/utils/log"
)

// 解析影子Data
func ParseShadowData(in []byte) (*ShadowData, error) {
	log.Debugf("parseData | data:%s\n", string(in))
	var data ShadowData
	err := json.Unmarshal(in, &data)
	if err != nil {
		log.Debugf("parseData | %v", err)
	}
	return &data, err
}

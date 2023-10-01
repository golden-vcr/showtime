package alerts

import (
	"encoding/json"
)

const (
	AlertTypeFollow = "follow"
	AlertTypeRaid   = "raid"
)

type Alert struct {
	Type string    `json:"type"`
	Data AlertData `json:"data"`
}

type AlertData struct {
	Follow *AlertDataFollow
	Raid   *AlertDataRaid
}

type AlertDataFollow struct {
	Username string `json:"username"`
}

type AlertDataRaid struct {
	Username   string `json:"username"`
	NumViewers int    `json:"numViewers"`
}

func (ad AlertData) MarshalJSON() ([]byte, error) {
	if ad.Follow != nil {
		return json.Marshal(ad.Follow)
	}
	if ad.Raid != nil {
		return json.Marshal(ad.Raid)
	}
	return json.Marshal(nil)
}

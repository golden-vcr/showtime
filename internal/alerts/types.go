package alerts

import (
	"encoding/json"
)

const (
	AlertTypeFollow          = "follow"
	AlertTypeRaid            = "raid"
	AlertTypeGeneratedImages = "generated-images"
)

type Alert struct {
	Type string    `json:"type"`
	Data AlertData `json:"data"`
}

type AlertData struct {
	Follow          *AlertDataFollow
	Raid            *AlertDataRaid
	GeneratedImages *AlertDataGeneratedImages
}

type AlertDataFollow struct {
	Username string `json:"username"`
}

type AlertDataRaid struct {
	Username   string `json:"username"`
	NumViewers int    `json:"numViewers"`
}

type AlertDataGeneratedImages struct {
	Username    string   `json:"username"`
	Description string   `json:"description"`
	Urls        []string `json:"urls"`
}

func (ad AlertData) MarshalJSON() ([]byte, error) {
	if ad.Follow != nil {
		return json.Marshal(ad.Follow)
	}
	if ad.Raid != nil {
		return json.Marshal(ad.Raid)
	}
	if ad.GeneratedImages != nil {
		return json.Marshal(ad.GeneratedImages)
	}
	return json.Marshal(nil)
}

package server

import (
	"encoding/json"

	"github.com/nicklaw5/helix/v2"
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

func (ev *Event) ToAlert() (*Alert, error) {
	switch ev.Type {
	case helix.EventSubTypeChannelFollow:
		return &Alert{
			Type: AlertTypeFollow,
			Data: AlertData{
				Follow: &AlertDataFollow{
					Username: ev.ChannelFollow.UserName,
				},
			},
		}, nil
	case helix.EventSubTypeChannelRaid:
		return &Alert{
			Type: AlertTypeRaid,
			Data: AlertData{
				Raid: &AlertDataRaid{
					Username:   ev.ChannelRaid.UserName,
					NumViewers: ev.ChannelRaid.Viewers,
				},
			},
		}, nil
	}
	return nil, nil
}

package eventsub

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/nicklaw5/helix/v2"
)

// Subscription declares a specific event type that we must subscribe to in order to
// support the showtime API via the connected Twitch channel
type Subscription struct {
	Type           string
	Version        string
	Condition      helix.EventSubCondition
	RequiredScopes []string
}

type ExistingSubscription struct {
	Value    helix.EventSubSubscription
	Required Subscription
}

// ConditionParams defines the template values that can be used when specifying the
// EventSubCondition values for required subscriptions
type ConditionParams struct {
	ChannelUserId string
}

type ReconcileResult struct {
	ToDelete []helix.EventSubSubscription
	ToCreate []Subscription
	Existing []ExistingSubscription
}

// format takes a helix.EventSubCondition struct whose values may contain template
// strings that references values from ConditionParams, and it transforms those
// template strings to the actual values contained in c.
func (c *ConditionParams) format(conditionWithTemplateStrings *helix.EventSubCondition) (*helix.EventSubCondition, error) {
	// Serialize the original struct to JSON and treat it as a Go template
	data, err := json.Marshal(conditionWithTemplateStrings)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("condition").Parse(string(data))
	if err != nil {
		return nil, err
	}

	// Execute the template, substituting the referenced values from our receiver
	var resultData bytes.Buffer
	if err := tmpl.Execute(&resultData, c); err != nil {
		return nil, err
	}

	// Deserialize from JSON to get a new struct with the final values
	var result helix.EventSubCondition
	if err := json.Unmarshal(resultData.Bytes(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

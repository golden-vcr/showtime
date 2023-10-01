package events

import (
	"bytes"
	"encoding/json"
	"html/template"
	"sort"

	"github.com/nicklaw5/helix/v2"
)

// RequiredSubscription declares a specific event type that we must subscribe to in
// order to support the showtime API via the connected Twitch channel
type RequiredSubscription struct {
	Type               string
	Version            string
	TemplatedCondition helix.EventSubCondition
	RequiredScopes     []string
}

// RequiredSubscriptionConditionParams defines the template values that can be used when
// specifying the EventSubCondition values for required subscriptions
type RequiredSubscriptionConditionParams struct {
	ChannelUserId string
}

// Format takes a helix.EventSubCondition struct whose values may contain template
// strings that references values from RequiredSubscriptionConditionParams, and it
// transforms those template strings to the actual values contained in c.
func (c *RequiredSubscriptionConditionParams) Format(conditionWithTemplateStrings *helix.EventSubCondition) (*helix.EventSubCondition, error) {
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

// GetRequiredUserScopes returns a list of all OAuth scopes that the connected Twitch
// user (i.e. the broadcaster) must authorize in order for our app to manage the
// required set of EventSub subscriptions on their behalf: we'll only be able to create
// those subscriptions via the EventSub API (using our app access token) once user
// access has been granted
func GetRequiredUserScopes(required []RequiredSubscription) []string {
	scopes := make(map[string]struct{})
	for i := range required {
		for _, scope := range required[i].RequiredScopes {
			scopes[scope] = struct{}{}
		}
	}
	scopesArray := make([]string, 0, len(scopes))
	for k := range scopes {
		scopesArray = append(scopesArray, k)
	}
	sort.Strings(scopesArray)
	return scopesArray
}

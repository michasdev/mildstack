package http

import (
	"fmt"
	"sort"

	snscontracts "github.com/michasdev/mildstack/core/internal/resources/sns/contracts"
)

// SNSRegistrySpec describes a known SNS action in the native adapter.
type SNSRegistrySpec struct {
	Action         string
	Version        string
	Supported      bool
	DomainDeferred bool
}

// SNSRegistry stores SNS action metadata and lookup index.
type SNSRegistry struct {
	ordered []SNSRegistrySpec
	byName  map[string]SNSRegistrySpec
}

func NewSNSRegistry() SNSRegistry {
	actions := snscontracts.ActionNames()
	sort.Strings(actions)

	ordered := make([]SNSRegistrySpec, 0, len(actions))
	byName := make(map[string]SNSRegistrySpec, len(actions))

	for _, action := range actions {
		supported := isSNSTopicSubscriptionAction(action)
		spec := SNSRegistrySpec{
			Action:         action,
			Version:        snsAPIVersion,
			Supported:      supported,
			DomainDeferred: !supported,
		}
		ordered = append(ordered, spec)
		byName[action] = spec
	}

	return SNSRegistry{ordered: ordered, byName: byName}
}

func (r SNSRegistry) Entries() []SNSRegistrySpec {
	return append([]SNSRegistrySpec(nil), r.ordered...)
}

func (r SNSRegistry) Lookup(action string) (SNSRegistrySpec, bool) {
	spec, ok := r.byName[action]
	return spec, ok
}

func (r SNSRegistry) Resolve(ctx SNSRequestContext) (SNSRegistrySpec, error) {
	spec, ok := r.Lookup(ctx.Action)
	if !ok {
		return SNSRegistrySpec{}, ErrSNSInvalidAction
	}
	if ctx.Version != spec.Version {
		return SNSRegistrySpec{}, ErrSNSInvalidVersion
	}
	return spec, nil
}

func (r SNSRegistry) String() string {
	return fmt.Sprintf("sns registry: %d actions", len(r.ordered))
}

func isSNSTopicSubscriptionAction(action string) bool {
	switch action {
	case "CreateTopic",
		"DeleteTopic",
		"GetTopicAttributes",
		"SetTopicAttributes",
		"ListTopics",
		"Publish",
		"PublishBatch",
		"Subscribe",
		"ConfirmSubscription",
		"Unsubscribe",
		"GetSubscriptionAttributes",
		"SetSubscriptionAttributes",
		"ListSubscriptions",
		"ListSubscriptionsByTopic":
		return true
	default:
		return false
	}
}

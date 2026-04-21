package http

import (
	"errors"
	"fmt"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
)

type SQSRegistrySpec struct {
	Action           string
	Scope            contracts.Scope
	Version          string
	Supported        bool
	DomainDeferred   bool
	MessageSurface   bool
	ReturnsQueueURL  bool
	UsesQueueContext bool
}

type SQSRegistry struct {
	ordered []SQSRegistrySpec
	byName  map[string]SQSRegistrySpec
}

func NewSQSRegistry() SQSRegistry {
	specs := contracts.Catalog()
	ordered := make([]SQSRegistrySpec, 0, len(specs))
	byName := make(map[string]SQSRegistrySpec, len(specs))

	for _, spec := range specs {
		supported := isQueueLifecycleAction(spec.Action) || isQueueGovernanceAction(spec.Action) || isQueueRedriveAction(spec.Action) || spec.MessageSurface
		entry := SQSRegistrySpec{
			Action:           spec.Action,
			Scope:            spec.Scope,
			Version:          spec.Version,
			Supported:        supported,
			DomainDeferred:   !supported,
			MessageSurface:   spec.MessageSurface,
			ReturnsQueueURL:  spec.ReturnsQueueURL,
			UsesQueueContext: spec.UsesQueueContext,
		}
		ordered = append(ordered, entry)
		byName[entry.Action] = entry
	}

	return SQSRegistry{
		ordered: ordered,
		byName:  byName,
	}
}

func (r SQSRegistry) Entries() []SQSRegistrySpec {
	return append([]SQSRegistrySpec(nil), r.ordered...)
}

func (r SQSRegistry) Lookup(action string) (SQSRegistrySpec, bool) {
	spec, ok := r.byName[action]
	return spec, ok
}

func (r SQSRegistry) Resolve(ctx SQSRequestContext) (SQSRegistrySpec, error) {
	spec, ok := r.Lookup(ctx.Action)
	if !ok {
		return SQSRegistrySpec{}, ErrSQSInvalidAction
	}
	if err := validateSQSRequestContext(ctx, spec); err != nil {
		return SQSRegistrySpec{}, err
	}
	return spec, nil
}

func (r SQSRegistry) SupportedActions() []string {
	actions := make([]string, 0, len(r.ordered))
	for _, spec := range r.ordered {
		if spec.Supported {
			actions = append(actions, spec.Action)
		}
	}
	return actions
}

func (r SQSRegistry) UnsupportedActions() []string {
	actions := make([]string, 0)
	for _, spec := range r.ordered {
		if !spec.Supported {
			actions = append(actions, spec.Action)
		}
	}
	return actions
}

func (r SQSRegistry) String() string {
	return fmt.Sprintf("sqs registry: %d actions", len(r.ordered))
}

func isQueueLifecycleAction(action string) bool {
	switch action {
	case "CreateQueue", "DeleteQueue", "GetQueueAttributes", "GetQueueUrl", "ListQueues", "PurgeQueue", "SetQueueAttributes":
		return true
	default:
		return false
	}
}

func isQueueGovernanceAction(action string) bool {
	switch action {
	case "AddPermission", "RemovePermission", "TagQueue", "UntagQueue", "ListQueueTags":
		return true
	default:
		return false
	}
}

func isQueueRedriveAction(action string) bool {
	switch action {
	case "ListDeadLetterSourceQueues", "StartMessageMoveTask", "CancelMessageMoveTask", "ListMessageMoveTasks":
		return true
	default:
		return false
	}
}

func isSQSErrorCode(err error, target error) bool {
	return errors.Is(err, target)
}

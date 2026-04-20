package contracts

type Scope string

const (
	ScopeRoot  Scope = "root"
	ScopeQueue Scope = "queue"
)

type ActionSpec struct {
	Action           string
	Scope            Scope
	Version          string
	ReturnsQueueURL  bool
	UsesQueueContext bool
}

var catalog = []ActionSpec{
	{Action: "AddPermission", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "CancelMessageMoveTask", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "ChangeMessageVisibility", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "ChangeMessageVisibilityBatch", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "CreateQueue", Scope: ScopeRoot, Version: "2012-11-05", ReturnsQueueURL: true},
	{Action: "DeleteMessage", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "DeleteMessageBatch", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "DeleteQueue", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "GetQueueAttributes", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "GetQueueUrl", Scope: ScopeRoot, Version: "2012-11-05", ReturnsQueueURL: true},
	{Action: "ListDeadLetterSourceQueues", Scope: ScopeQueue, Version: "2012-11-05", ReturnsQueueURL: true, UsesQueueContext: true},
	{Action: "ListMessageMoveTasks", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "ListQueues", Scope: ScopeRoot, Version: "2012-11-05", ReturnsQueueURL: true},
	{Action: "ListQueueTags", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "PurgeQueue", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "ReceiveMessage", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "RemovePermission", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "SendMessage", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "SendMessageBatch", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "SetQueueAttributes", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "StartMessageMoveTask", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "TagQueue", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
	{Action: "UntagQueue", Scope: ScopeQueue, Version: "2012-11-05", UsesQueueContext: true},
}

func Catalog() []ActionSpec {
	return append([]ActionSpec(nil), catalog...)
}

func ActionNames() []string {
	specs := Catalog()
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Action)
	}
	return names
}

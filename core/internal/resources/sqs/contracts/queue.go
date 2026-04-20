package contracts

// QueueAttributesView is the shared queue attribute response contract used by
// the application service and native transport scaffolding.
type QueueAttributesView struct {
	QueueName  string
	QueueURL   string
	QueueARN   string
	Attributes map[string]string
}

package domain

import (
	"sort"
	"strings"
	"time"
)

const StateKey = "services/sqs"

type State struct {
	Service          string
	Queues           []Queue
	Messages         []Message
	RecoveryMetadata map[string]RecoveryMetadata
}

type Queue struct {
	Name         string
	URL          string
	Attributes   map[string]string
	OrderingHint string
	Recovery     QueueRecovery
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type QueueRecovery struct {
	DeadLetterQueue string
	Policy          map[string]string
}

type Message struct {
	Queue                 string
	MessageID             string
	Body                  string
	Attributes            map[string]string
	Metadata              map[string]string
	Tags                  []string
	ReceiptKeys           []string
	MessageGroupID        string
	SequenceNumber        int64
	BatchID               string
	BatchEntryID          string
	BatchEntryIndex       int
	BatchEntryCount       int
	DeadLetterQueue       string
	DeadLetterSourceQueue string
	DeadLetteredAt        time.Time
	SentAt                time.Time
	AvailableAt           time.Time
	ReceivedAt            time.Time
	Recovery              MessageRecovery
}

type MessageRecovery struct {
	Attempts int
	Detail   map[string]string
}

type RecoveryMetadata struct {
	Queue   string
	Message string
	Detail  map[string]string
}

func NewState() State {
	return State{
		Service:          "sqs",
		Queues:           []Queue{},
		Messages:         []Message{},
		RecoveryMetadata: map[string]RecoveryMetadata{},
	}
}

func (s State) Snapshot() map[string]any {
	queues := make([]any, 0, len(s.Queues))
	for _, queue := range s.ListQueues() {
		queues = append(queues, map[string]any{
			"name":          queue.Name,
			"url":           queue.URL,
			"attributes":    cloneStringMapAny(queue.Attributes),
			"ordering_hint": queue.OrderingHint,
			"recovery": map[string]any{
				"dead_letter_queue": queue.Recovery.DeadLetterQueue,
				"policy":            cloneStringMapAny(queue.Recovery.Policy),
			},
			"created_at": snapshotTime(queue.CreatedAt),
			"updated_at": snapshotTime(queue.UpdatedAt),
		})
	}

	messages := make([]any, 0, len(s.Messages))
	for _, message := range s.ListMessages() {
		messages = append(messages, map[string]any{
			"queue":                    message.Queue,
			"message_id":               message.MessageID,
			"body":                     message.Body,
			"attributes":               cloneStringMapAny(message.Attributes),
			"metadata":                 cloneStringMapAny(message.Metadata),
			"tags":                     append([]string(nil), message.Tags...),
			"receipt_keys":             append([]string(nil), message.ReceiptKeys...),
			"message_group_id":         message.MessageGroupID,
			"sequence_number":          message.SequenceNumber,
			"batch_id":                 message.BatchID,
			"batch_entry_id":           message.BatchEntryID,
			"batch_entry_index":        message.BatchEntryIndex,
			"batch_entry_count":        message.BatchEntryCount,
			"dead_letter_queue":        message.DeadLetterQueue,
			"dead_letter_source_queue": message.DeadLetterSourceQueue,
			"dead_lettered_at":         snapshotTime(message.DeadLetteredAt),
			"sent_at":                  snapshotTime(message.SentAt),
			"available_at":             snapshotTime(message.AvailableAt),
			"received_at":              snapshotTime(message.ReceivedAt),
			"recovery": map[string]any{
				"attempts": message.Recovery.Attempts,
				"detail":   cloneStringMapAny(message.Recovery.Detail),
			},
		})
	}

	recovery := make(map[string]any, len(s.RecoveryMetadata))
	keys := make([]string, 0, len(s.RecoveryMetadata))
	for key := range s.RecoveryMetadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		metadata := s.RecoveryMetadata[key]
		recovery[key] = map[string]any{
			"queue":   metadata.Queue,
			"message": metadata.Message,
			"detail":  cloneStringMapAny(metadata.Detail),
		}
	}

	return map[string]any{
		"service":           s.Service,
		"queues":            queues,
		"messages":          messages,
		"recovery_metadata": recovery,
	}
}

func (s State) Clone() State {
	cloned := State{
		Service:          s.Service,
		Queues:           make([]Queue, len(s.Queues)),
		Messages:         make([]Message, len(s.Messages)),
		RecoveryMetadata: make(map[string]RecoveryMetadata, len(s.RecoveryMetadata)),
	}
	copy(cloned.Queues, s.Queues)
	for i := range cloned.Queues {
		cloned.Queues[i].Attributes = cloneStringMap(cloned.Queues[i].Attributes)
		cloned.Queues[i].Recovery.Policy = cloneStringMap(cloned.Queues[i].Recovery.Policy)
	}
	copy(cloned.Messages, s.Messages)
	for i := range cloned.Messages {
		cloned.Messages[i].Attributes = cloneStringMap(cloned.Messages[i].Attributes)
		cloned.Messages[i].Metadata = cloneStringMap(cloned.Messages[i].Metadata)
		cloned.Messages[i].Tags = append([]string(nil), cloned.Messages[i].Tags...)
		cloned.Messages[i].ReceiptKeys = append([]string(nil), cloned.Messages[i].ReceiptKeys...)
		cloned.Messages[i].Recovery.Detail = cloneStringMap(cloned.Messages[i].Recovery.Detail)
	}
	for key, value := range s.RecoveryMetadata {
		cloned.RecoveryMetadata[key] = RecoveryMetadata{
			Queue:   value.Queue,
			Message: value.Message,
			Detail:  cloneStringMap(value.Detail),
		}
	}
	return cloned
}

func (s State) ListQueues() []Queue {
	queues := make([]Queue, len(s.Queues))
	copy(queues, s.Queues)
	for i := range queues {
		queues[i].Attributes = cloneStringMap(queues[i].Attributes)
		queues[i].Recovery.Policy = cloneStringMap(queues[i].Recovery.Policy)
	}
	sort.SliceStable(queues, func(i, j int) bool {
		return queues[i].Name < queues[j].Name
	})
	return queues
}

func (s State) ListMessages() []Message {
	messages := make([]Message, len(s.Messages))
	copy(messages, s.Messages)
	for i := range messages {
		messages[i].Attributes = cloneStringMap(messages[i].Attributes)
		messages[i].Metadata = cloneStringMap(messages[i].Metadata)
		messages[i].Tags = append([]string(nil), messages[i].Tags...)
		messages[i].ReceiptKeys = append([]string(nil), messages[i].ReceiptKeys...)
		messages[i].Recovery.Detail = cloneStringMap(messages[i].Recovery.Detail)
	}
	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].Queue == messages[j].Queue {
			return messages[i].MessageID < messages[j].MessageID
		}
		return messages[i].Queue < messages[j].Queue
	})
	return messages
}

func snapshotTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneStringMapAny(values map[string]string) map[string]any {
	if values == nil {
		return nil
	}

	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func normalizeName(value string) string {
	return strings.TrimSpace(value)
}

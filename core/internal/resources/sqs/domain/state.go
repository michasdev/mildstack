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
	QueueTags        map[string]map[string]string
	QueuePermissions map[string]map[string]QueuePermission
	MoveTasks        map[string]map[string]MessageMoveTask
}

type Queue struct {
	Name         string
	URL          string
	Attributes   map[string]string
	OrderingHint string
	Recovery     QueueRecovery
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    time.Time
	PurgedAt     time.Time
}

type QueueRecovery struct {
	DeadLetterQueue string
	Policy          map[string]string
}

type QueuePermission struct {
	Label         string
	AWSAccountIDs []string
	Actions       []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type MessageMoveTask struct {
	TaskHandle                       string
	SourceQueue                      string
	SourceArn                        string
	DestinationArn                   string
	MaxNumberOfMessagesPerSecond     int
	ApproximateNumberOfMessagesMoved int64
	Status                           string
	StartedAt                        time.Time
	UpdatedAt                        time.Time
	CancelledAt                      time.Time
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
		QueueTags:        map[string]map[string]string{},
		QueuePermissions: map[string]map[string]QueuePermission{},
		MoveTasks:        map[string]map[string]MessageMoveTask{},
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
			"deleted_at": snapshotTime(queue.DeletedAt),
			"purged_at":  snapshotTime(queue.PurgedAt),
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

	queueTags := make(map[string]any, len(s.QueueTags))
	tagQueueNames := make([]string, 0, len(s.QueueTags))
	for queueName := range s.QueueTags {
		tagQueueNames = append(tagQueueNames, queueName)
	}
	sort.Strings(tagQueueNames)
	for _, queueName := range tagQueueNames {
		queueTags[queueName] = cloneStringMapAny(s.QueueTags[queueName])
	}

	queuePermissions := make(map[string]any, len(s.QueuePermissions))
	permissionQueueNames := make([]string, 0, len(s.QueuePermissions))
	for queueName := range s.QueuePermissions {
		permissionQueueNames = append(permissionQueueNames, queueName)
	}
	sort.Strings(permissionQueueNames)
	for _, queueName := range permissionQueueNames {
		labels := make([]string, 0, len(s.QueuePermissions[queueName]))
		for label := range s.QueuePermissions[queueName] {
			labels = append(labels, label)
		}
		sort.Strings(labels)
		entries := make([]any, 0, len(labels))
		for _, label := range labels {
			permission := s.QueuePermissions[queueName][label]
			entries = append(entries, map[string]any{
				"label":           permission.Label,
				"aws_account_ids": append([]string(nil), permission.AWSAccountIDs...),
				"actions":         append([]string(nil), permission.Actions...),
				"created_at":      snapshotTime(permission.CreatedAt),
				"updated_at":      snapshotTime(permission.UpdatedAt),
			})
		}
		queuePermissions[queueName] = entries
	}

	moveTasks := make(map[string]any, len(s.MoveTasks))
	moveQueueNames := make([]string, 0, len(s.MoveTasks))
	for queueName := range s.MoveTasks {
		moveQueueNames = append(moveQueueNames, queueName)
	}
	sort.Strings(moveQueueNames)
	for _, queueName := range moveQueueNames {
		handles := make([]string, 0, len(s.MoveTasks[queueName]))
		for handle := range s.MoveTasks[queueName] {
			handles = append(handles, handle)
		}
		sort.Strings(handles)
		entries := make([]any, 0, len(handles))
		for _, handle := range handles {
			task := s.MoveTasks[queueName][handle]
			entries = append(entries, map[string]any{
				"task_handle":                          task.TaskHandle,
				"source_queue":                         task.SourceQueue,
				"source_arn":                           task.SourceArn,
				"destination_arn":                      task.DestinationArn,
				"max_number_of_messages_per_second":    task.MaxNumberOfMessagesPerSecond,
				"approximate_number_of_messages_moved": task.ApproximateNumberOfMessagesMoved,
				"status":                               task.Status,
				"started_at":                           snapshotTime(task.StartedAt),
				"updated_at":                           snapshotTime(task.UpdatedAt),
				"cancelled_at":                         snapshotTime(task.CancelledAt),
			})
		}
		moveTasks[queueName] = entries
	}

	return map[string]any{
		"service":           s.Service,
		"queues":            queues,
		"messages":          messages,
		"recovery_metadata": recovery,
		"queue_tags":        queueTags,
		"queue_permissions": queuePermissions,
		"move_tasks":        moveTasks,
	}
}

func (s State) Clone() State {
	cloned := State{
		Service:          s.Service,
		Queues:           make([]Queue, len(s.Queues)),
		Messages:         make([]Message, len(s.Messages)),
		RecoveryMetadata: make(map[string]RecoveryMetadata, len(s.RecoveryMetadata)),
		QueueTags:        make(map[string]map[string]string, len(s.QueueTags)),
		QueuePermissions: make(map[string]map[string]QueuePermission, len(s.QueuePermissions)),
		MoveTasks:        make(map[string]map[string]MessageMoveTask, len(s.MoveTasks)),
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
	for queueName, tags := range s.QueueTags {
		cloned.QueueTags[queueName] = cloneStringMap(tags)
	}
	for queueName, permissions := range s.QueuePermissions {
		cloned.QueuePermissions[queueName] = make(map[string]QueuePermission, len(permissions))
		for label, permission := range permissions {
			cloned.QueuePermissions[queueName][label] = QueuePermission{
				Label:         permission.Label,
				AWSAccountIDs: append([]string(nil), permission.AWSAccountIDs...),
				Actions:       append([]string(nil), permission.Actions...),
				CreatedAt:     permission.CreatedAt,
				UpdatedAt:     permission.UpdatedAt,
			}
		}
	}
	for queueName, tasks := range s.MoveTasks {
		cloned.MoveTasks[queueName] = make(map[string]MessageMoveTask, len(tasks))
		for handle, task := range tasks {
			cloned.MoveTasks[queueName][handle] = MessageMoveTask{
				TaskHandle:                       task.TaskHandle,
				SourceQueue:                      task.SourceQueue,
				SourceArn:                        task.SourceArn,
				DestinationArn:                   task.DestinationArn,
				MaxNumberOfMessagesPerSecond:     task.MaxNumberOfMessagesPerSecond,
				ApproximateNumberOfMessagesMoved: task.ApproximateNumberOfMessagesMoved,
				Status:                           task.Status,
				StartedAt:                        task.StartedAt,
				UpdatedAt:                        task.UpdatedAt,
				CancelledAt:                      task.CancelledAt,
			}
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

package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
	"github.com/michasdev/mildstack/core/internal/resources/sqs/infrastructure"
)

var _ orchestrator.Service = (*Service)(nil)

type Service struct {
	state     domain.State
	policy    orchestrator.EmulationPolicy
	repo      Repository
	stateHook orchestrator.StateHook
	clock     Clock
	worker    *worker
	cancel    context.CancelFunc
	done      chan struct{}
	mu        sync.Mutex
}

const (
	defaultVisibilityTimeout      = 30 * time.Second
	maxLongPollWait               = 20 * time.Second
	workerPollInterval            = 50 * time.Millisecond
	leaseVisibilityTimeoutMetaKey = "visibility_timeout_seconds"
)

var (
	errQueueNotFound           = errors.New("sqs: queue not found")
	errReceiptHandleMismatch   = errors.New("sqs: receipt handle does not match active lease")
	errInvalidVisibilityWindow = errors.New("sqs: visibility timeout must be non-negative")
)

func New() *Service {
	return newService(domain.NewState(), nil)
}

func newService(state domain.State, repo Repository) *Service {
	return &Service{
		state: state.Clone(),
		repo:  repo,
		clock: realClock{},
		policy: orchestrator.NewEmulationPolicy(
			orchestrator.FidelityExemplar,
			contracts.ActionNames(),
			nil,
			"sqs",
		),
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.worker != nil {
		return nil
	}

	s.worker = newWorker(s, s.clock)
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})

	go func(done chan struct{}, runCtx context.Context, w *worker) {
		defer close(done)
		w.run(runCtx)
	}(s.done, runCtx, s.worker)

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.cancel = nil
	s.worker = nil
	s.done = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		waitCtx := ctx
		if _, ok := waitCtx.Deadline(); !ok {
			var cancelWait context.CancelFunc
			waitCtx, cancelWait = context.WithTimeout(waitCtx, 2*time.Second)
			defer cancelWait()
		}
		select {
		case <-done:
		case <-waitCtx.Done():
			return fmt.Errorf("sqs: worker shutdown timed out")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.repo == nil {
		return nil
	}

	if err := s.repo.Close(); err != nil {
		return fmt.Errorf("sqs: close repository: %w", err)
	}
	s.repo = nil
	return nil
}

func (s *Service) Metadata() orchestrator.Metadata {
	return orchestrator.Metadata{
		Name:        "sqs",
		Description: "MildStack SQS real service",
		Version:     "v1",
		Tags:        []string{"aws", "messaging", "queue", "real-service"},
	}
}

func (s *Service) Policy() orchestrator.EmulationPolicy {
	return s.policy.Clone()
}

func (s *Service) RegisterRoutes(registrar orchestrator.RouteRegistrar) error {
	for _, route := range infrastructure.Routes() {
		if err := registrar.Register(route); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) AttachState(hook orchestrator.StateHook) error {
	if hook == nil {
		return fmt.Errorf("sqs: nil state hook")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stateHook = hook
	s.publishSnapshotLocked()
	return nil
}

func (s *Service) publishSnapshotLocked() {
	if s.stateHook == nil {
		return
	}

	s.stateHook.Set(domain.StateKey, s.state.Snapshot())
}

func newServiceWithClock(state domain.State, repo Repository, clock Clock) *Service {
	service := newService(state, repo)
	if clock != nil {
		service.clock = clock
	}
	return service
}

func (s *Service) ReceiveMessage(queueName string, maxMessages int, waitTime time.Duration) ([]domain.Message, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return nil, fmt.Errorf("sqs: queue name is required")
	}
	if maxMessages <= 0 {
		maxMessages = 1
	}
	if waitTime < 0 {
		waitTime = 0
	}
	if waitTime > maxLongPollWait {
		waitTime = maxLongPollWait
	}

	deadline := s.clock.Now().Add(waitTime)
	for {
		now := s.clock.Now()
		messages, err := s.receiveReadyMessagesLocked(queueName, maxMessages, now)
		if err != nil {
			return nil, err
		}
		if len(messages) > 0 || waitTime == 0 || !now.Before(deadline) {
			return messages, nil
		}

		remaining := deadline.Sub(now)
		sleep := workerPollInterval
		if remaining < sleep {
			sleep = remaining
		}
		if sleep <= 0 {
			return messages, nil
		}
		s.clock.Sleep(sleep)
	}
}

func (s *Service) DeleteMessage(queueName string, receiptHandle string) error {
	queueName = trimName(queueName)
	receiptHandle = trimName(receiptHandle)
	if queueName == "" {
		return fmt.Errorf("sqs: queue name is required")
	}
	if receiptHandle == "" {
		return fmt.Errorf("sqs: receipt handle is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.findMessageByReceiptLocked(queueName, receiptHandle)
	if !ok {
		return errReceiptHandleMismatch
	}

	s.state.Messages = append(s.state.Messages[:idx], s.state.Messages[idx+1:]...)
	return s.commitStateLocked()
}

func (s *Service) ChangeMessageVisibility(queueName string, receiptHandle string, visibility time.Duration) error {
	queueName = trimName(queueName)
	receiptHandle = trimName(receiptHandle)
	if queueName == "" {
		return fmt.Errorf("sqs: queue name is required")
	}
	if receiptHandle == "" {
		return fmt.Errorf("sqs: receipt handle is required")
	}
	if visibility < 0 {
		return errInvalidVisibilityWindow
	}
	if visibility > 12*time.Hour {
		visibility = 12 * time.Hour
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.findMessageByReceiptLocked(queueName, receiptHandle)
	if !ok {
		return errReceiptHandleMismatch
	}

	message := &s.state.Messages[idx]
	if message.Metadata == nil {
		message.Metadata = map[string]string{}
	}
	if visibility == 0 {
		message.ReceivedAt = time.Time{}
		message.AvailableAt = s.clock.Now()
		message.Metadata[leaseVisibilityTimeoutMetaKey] = "0"
	} else {
		message.ReceivedAt = s.clock.Now()
		message.Metadata[leaseVisibilityTimeoutMetaKey] = strconv.FormatInt(int64(visibility/time.Second), 10)
	}

	return s.commitStateLocked()
}

func (s *Service) commitStateLocked() error {
	if s.repo != nil {
		if err := s.repo.Save(s.state.Clone()); err != nil {
			return fmt.Errorf("sqs: save repository: %w", err)
		}
	}
	s.publishSnapshotLocked()
	return nil
}

func (s *Service) queueByNameLocked(name string) (domain.Queue, bool) {
	for _, queue := range s.state.Queues {
		if queue.Name == name {
			return queue, true
		}
	}
	return domain.Queue{}, false
}

func (s *Service) receiveReadyMessagesLocked(queueName string, maxMessages int, now time.Time) ([]domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.queueByNameLocked(queueName)
	if !ok {
		return nil, errQueueNotFound
	}

	indices := make([]int, 0, len(s.state.Messages))
	for idx, message := range s.state.Messages {
		if !messageVisibleInQueue(message, queueName) {
			continue
		}
		if IsDelayed(message, now) || IsInvisible(message, queue, now) {
			continue
		}
		if IsFIFOQueue(queue) && fifoDeliveryBlocked(s.state.Messages, idx, queueName) {
			continue
		}
		indices = append(indices, idx)
	}
	sortMessagesByDelivery(s.state.Messages, indices, queue)

	selected := make([]domain.Message, 0, maxMessages)
	for _, idx := range indices {
		message := &s.state.Messages[idx]
		if message.Metadata == nil {
			message.Metadata = map[string]string{}
		}
		timeout := queueVisibilityTimeout(queue, *message)
		message.ReceivedAt = now
		message.Recovery.Attempts++
		message.Metadata[leaseVisibilityTimeoutMetaKey] = strconv.FormatInt(int64(timeout/time.Second), 10)
		message.ReceiptKeys = append(message.ReceiptKeys, nextReceiptHandle(queueName, *message))
		selected = append(selected, cloneMessage(*message))
		if len(selected) == maxMessages {
			break
		}
	}

	if len(selected) == 0 {
		return nil, nil
	}
	if err := s.commitStateLocked(); err != nil {
		return nil, err
	}

	return selected, nil
}

func (s *Service) findMessageByReceiptLocked(queueName, receiptHandle string) (int, bool) {
	for idx, message := range s.state.Messages {
		if !messageVisibleInQueue(message, queueName) {
			continue
		}
		if CurrentReceiptHandle(message) == receiptHandle {
			return idx, true
		}
	}
	return -1, false
}

func sortMessagesByDelivery(messages []domain.Message, indices []int, queue domain.Queue) {
	sort.SliceStable(indices, func(i, j int) bool {
		left := messages[indices[i]]
		right := messages[indices[j]]
		if IsFIFOQueue(queue) {
			if left.MessageGroupID != right.MessageGroupID {
				return left.MessageGroupID < right.MessageGroupID
			}
			if left.SequenceNumber != right.SequenceNumber && left.SequenceNumber > 0 && right.SequenceNumber > 0 {
				return left.SequenceNumber < right.SequenceNumber
			}
		}
		if !left.SentAt.Equal(right.SentAt) {
			return left.SentAt.Before(right.SentAt)
		}
		if !left.AvailableAt.Equal(right.AvailableAt) {
			return left.AvailableAt.Before(right.AvailableAt)
		}
		return left.MessageID < right.MessageID
	})
}

func cloneMessage(message domain.Message) domain.Message {
	cloned := message
	cloned.Attributes = cloneMap(message.Attributes)
	cloned.Metadata = cloneMap(message.Metadata)
	cloned.Tags = append([]string(nil), message.Tags...)
	cloned.ReceiptKeys = append([]string(nil), message.ReceiptKeys...)
	cloned.Recovery.Detail = cloneMap(message.Recovery.Detail)
	return cloned
}

func queueVisibilityTimeout(queue domain.Queue, message domain.Message) time.Duration {
	if timeout := parseMessageVisibilityTimeout(message.Metadata[leaseVisibilityTimeoutMetaKey]); timeout > 0 {
		return timeout
	}
	if queue.Attributes != nil {
		if timeout := parseMessageVisibilityTimeout(queue.Attributes["VisibilityTimeout"]); timeout > 0 {
			return timeout
		}
	}
	return defaultVisibilityTimeout
}

func parseMessageVisibilityTimeout(raw string) time.Duration {
	raw = trimName(raw)
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func nextReceiptHandle(queueName string, message domain.Message) string {
	return fmt.Sprintf("%s/%s/%d", queueName, message.MessageID, len(message.ReceiptKeys)+1)
}

func trimName(value string) string {
	return strings.TrimSpace(value)
}

func cloneMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func messageVisibleInQueue(message domain.Message, queueName string) bool {
	if message.Queue != queueName {
		return false
	}
	if trimName(message.DeadLetterSourceQueue) == queueName && trimName(message.DeadLetterQueue) != queueName {
		return false
	}
	return true
}

func fifoDeliveryBlocked(messages []domain.Message, candidateIndex int, queueName string) bool {
	candidate := messages[candidateIndex]
	groupID := trimName(candidate.MessageGroupID)
	if groupID == "" {
		return false
	}

	for idx, other := range messages {
		if idx == candidateIndex || !messageVisibleInQueue(other, queueName) {
			continue
		}
		if trimName(other.MessageGroupID) != groupID {
			continue
		}
		if compareFIFOMessageOrder(other, candidate) < 0 {
			return true
		}
	}
	return false
}

func compareFIFOMessageOrder(left, right domain.Message) int {
	if left.SequenceNumber > 0 && right.SequenceNumber > 0 && left.SequenceNumber != right.SequenceNumber {
		if left.SequenceNumber < right.SequenceNumber {
			return -1
		}
		return 1
	}
	if !left.SentAt.Equal(right.SentAt) {
		if left.SentAt.Before(right.SentAt) {
			return -1
		}
		return 1
	}
	if !left.AvailableAt.Equal(right.AvailableAt) {
		if left.AvailableAt.Before(right.AvailableAt) {
			return -1
		}
		return 1
	}
	switch {
	case left.MessageID < right.MessageID:
		return -1
	case left.MessageID > right.MessageID:
		return 1
	default:
		return 0
	}
}

func queueOrderingMode(queue domain.Queue) string {
	if IsFIFOQueue(queue) {
		return "fifo"
	}
	return "standard"
}

func deadLetterThresholdFromQueue(queue domain.Queue) int {
	return deadLetterThreshold(queue)
}

func (s *Service) deadLetterEligibleLocked(message domain.Message, queue domain.Queue, now time.Time) bool {
	return IsDeadLetterEligible(message, queue, now)
}

func (s *Service) sweepDeadLettersLocked(now time.Time) int {
	mutated := 0
	for idx := range s.state.Messages {
		message := s.state.Messages[idx]
		queue, ok := s.queueByNameLocked(message.Queue)
		if !ok {
			continue
		}
		if !s.deadLetterEligibleLocked(message, queue, now) {
			continue
		}
		if s.moveMessageToDeadLetterLocked(idx, queue, now) {
			mutated++
		}
	}
	if mutated > 0 {
		_ = s.commitStateLocked()
	}
	return mutated
}

func (s *Service) moveMessageToDeadLetterLocked(index int, sourceQueue domain.Queue, now time.Time) bool {
	message := &s.state.Messages[index]
	dlqName := trimName(sourceQueue.Recovery.DeadLetterQueue)
	if dlqName == "" || trimName(message.DeadLetterQueue) == dlqName {
		return false
	}

	message.DeadLetterQueue = dlqName
	message.DeadLetterSourceQueue = sourceQueue.Name
	message.DeadLetteredAt = now
	message.ReceivedAt = time.Time{}
	message.AvailableAt = now
	if message.Metadata == nil {
		message.Metadata = map[string]string{}
	}
	message.Metadata["dead_letter_queue"] = dlqName
	message.Metadata["dead_letter_source_queue"] = sourceQueue.Name
	message.Metadata["dead_lettered_at"] = now.UTC().Format(time.RFC3339Nano)

	if dlqQueue, ok := s.queueByNameLocked(dlqName); ok {
		message.Queue = dlqQueue.Name
	}

	if s.state.RecoveryMetadata == nil {
		s.state.RecoveryMetadata = map[string]domain.RecoveryMetadata{}
	}
	s.state.RecoveryMetadata[sourceQueue.Name+"/"+message.MessageID] = domain.RecoveryMetadata{
		Queue:   dlqName,
		Message: message.MessageID,
		Detail: map[string]string{
			"source_queue":       sourceQueue.Name,
			"dead_letter_queue":  dlqName,
			"attempts":           strconv.Itoa(message.Recovery.Attempts),
			"dead_lettered_at":   now.UTC().Format(time.RFC3339Nano),
			"dead_letter_reason": "max_receive_count",
			"ordering_mode":      queueOrderingMode(sourceQueue),
		},
	}
	return true
}

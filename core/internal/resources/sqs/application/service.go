package application

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
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
	queueLifecycleCooldown        = 60 * time.Second
)

var (
	errQueueNotFound           = errors.New("sqs: queue not found")
	errReceiptHandleMismatch   = errors.New("sqs: receipt handle does not match active lease")
	errInvalidVisibilityWindow = errors.New("sqs: visibility timeout must be non-negative")
	errEmptyBatchRequest       = errors.New("sqs: batch request is empty")
	errTooManyBatchEntries     = errors.New("sqs: batch request contains more than 10 entries")
	errDuplicateBatchEntryIDs  = errors.New("sqs: batch request contains duplicate entry IDs")
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

func (s *Service) QueueURL(queueName string) string {
	return queueURLForAccount(queueName, "")
}

func (s *Service) QueueARN(queueName string) string {
	return queueARNForAccount(queueName, "")
}

func (s *Service) CreateQueue(queueName string, attributes map[string]string) (domain.Queue, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return domain.Queue{}, fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now()
	normalizedAttributes := cloneMap(attributes)
	if normalizedAttributes == nil {
		normalizedAttributes = map[string]string{}
	}
	recovery := queueRecoveryFromAttributes(normalizedAttributes)

	if index, queue, ok := s.queueRecordByNameLocked(queueName); ok {
		if queue.DeletedAt.IsZero() {
			if equalStringMaps(queue.Attributes, normalizedAttributes) {
				return s.queueResponse(queueName, queue, normalizedAttributes), nil
			}
			return domain.Queue{}, fmt.Errorf("sqs: queue %q already exists with different attributes", queueName)
		}
		if now.Sub(queue.DeletedAt) < queueLifecycleCooldown {
			return domain.Queue{}, fmt.Errorf("sqs: queue %q is still in delete cooldown", queueName)
		}

		queue.URL = s.QueueURL(queueName)
		queue.Attributes = normalizedAttributes
		queue.Recovery = recovery
		queue.OrderingHint = orderingHintFromAttributes(normalizedAttributes, queue.OrderingHint)
		queue.CreatedAt = now
		queue.UpdatedAt = now
		queue.DeletedAt = time.Time{}
		queue.PurgedAt = time.Time{}
		s.state.Queues[index] = queue
		if err := s.commitStateLocked(); err != nil {
			return domain.Queue{}, err
		}
		return s.queueResponse(queueName, queue, normalizedAttributes), nil
	}

	queue := domain.Queue{
		Name:         queueName,
		URL:          s.QueueURL(queueName),
		Attributes:   normalizedAttributes,
		Recovery:     recovery,
		OrderingHint: orderingHintFromAttributes(normalizedAttributes, ""),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.state.Queues = append(s.state.Queues, queue)
	if err := s.commitStateLocked(); err != nil {
		return domain.Queue{}, err
	}
	return s.queueResponse(queueName, queue, normalizedAttributes), nil
}

func (s *Service) DeleteQueue(queueName string) error {
	queueName = trimName(queueName)
	if queueName == "" {
		return fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, queue, ok := s.queueRecordByNameLocked(queueName)
	if !ok {
		return errQueueNotFound
	}
	now := s.clock.Now()
	if !queue.DeletedAt.IsZero() {
		if now.Sub(queue.DeletedAt) < queueLifecycleCooldown {
			return fmt.Errorf("sqs: queue %q is still in delete cooldown", queueName)
		}
		return errQueueNotFound
	}

	queue.DeletedAt = now
	queue.UpdatedAt = now
	queue.PurgedAt = time.Time{}
	s.state.Queues[index] = queue
	s.removeMessagesForQueueLocked(queueName)
	return s.commitStateLocked()
}

func (s *Service) GetQueueUrl(queueName, ownerAccountID string) (string, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return "", fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.activeQueueByNameLocked(queueName)
	if !ok || !ownerAccountMatches(ownerAccountID) {
		return "", errQueueNotFound
	}
	if queue.URL == "" {
		return queueURLForAccount(queueName, ownerAccountID), nil
	}
	return queue.URL, nil
}

func (s *Service) ListQueues(queueNamePrefix string, maxResults int, nextToken, ownerAccountID string) ([]domain.Queue, string, error) {
	queueNamePrefix = trimName(queueNamePrefix)
	nextToken = trimName(nextToken)
	ownerAccountID = trimName(ownerAccountID)
	if maxResults < 0 {
		maxResults = 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !ownerAccountMatches(ownerAccountID) {
		return nil, "", errQueueNotFound
	}

	activeQueues := make([]domain.Queue, 0, len(s.state.Queues))
	for _, queue := range s.state.ListQueues() {
		if !queue.DeletedAt.IsZero() {
			continue
		}
		if queueNamePrefix != "" && !strings.HasPrefix(queue.Name, queueNamePrefix) {
			continue
		}
		if queue.URL == "" {
			queue.URL = s.QueueURL(queue.Name)
		}
		activeQueues = append(activeQueues, queue)
	}

	startIndex := 0
	if nextToken != "" {
		startIndex = len(activeQueues)
		for i, queue := range activeQueues {
			if queue.Name > nextToken {
				startIndex = i
				break
			}
			if queue.Name == nextToken {
				startIndex = i + 1
			}
		}
	}
	if startIndex > len(activeQueues) {
		startIndex = len(activeQueues)
	}

	endIndex := len(activeQueues)
	if maxResults > 0 && startIndex+maxResults < endIndex {
		endIndex = startIndex + maxResults
	}

	page := append([]domain.Queue(nil), activeQueues[startIndex:endIndex]...)
	nextPageToken := ""
	if endIndex < len(activeQueues) {
		nextPageToken = activeQueues[endIndex-1].Name
	}

	return page, nextPageToken, nil
}

func (s *Service) PurgeQueue(queueName string) error {
	queueName = trimName(queueName)
	if queueName == "" {
		return fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, queue, ok := s.queueRecordByNameLocked(queueName)
	if !ok || !queue.DeletedAt.IsZero() {
		return errQueueNotFound
	}
	now := s.clock.Now()
	if !queue.PurgedAt.IsZero() && now.Sub(queue.PurgedAt) < queueLifecycleCooldown {
		return fmt.Errorf("sqs: queue %q is still in purge cooldown", queueName)
	}

	s.removeMessagesForQueueLocked(queueName)
	queue.PurgedAt = now
	queue.UpdatedAt = now
	s.state.Queues[index] = queue
	return s.commitStateLocked()
}

func (s *Service) GetQueueAttributes(queueName string, attributeNames []string, ownerAccountID string) (contracts.QueueAttributesView, error) {
	queueName = trimName(queueName)
	ownerAccountID = trimName(ownerAccountID)
	if queueName == "" {
		return contracts.QueueAttributesView{}, fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.activeQueueByNameLocked(queueName)
	if !ok || !ownerAccountMatches(ownerAccountID) {
		return contracts.QueueAttributesView{}, errQueueNotFound
	}

	attributes := selectQueueAttributes(queue.Attributes, attributeNames, s.QueueARN(queueName))
	if attributes == nil {
		attributes = map[string]string{}
	}

	return contracts.QueueAttributesView{
		QueueName:  queueName,
		QueueURL:   queueURLForAccount(queueName, ownerAccountID),
		QueueARN:   queueARNForAccount(queueName, ownerAccountID),
		Attributes: attributes,
	}, nil
}

func (s *Service) SetQueueAttributes(queueName string, attributes map[string]string) (contracts.QueueAttributesView, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return contracts.QueueAttributesView{}, fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, queue, ok := s.queueRecordByNameLocked(queueName)
	if !ok || !queue.DeletedAt.IsZero() {
		return contracts.QueueAttributesView{}, errQueueNotFound
	}

	normalizedAttributes := cloneMap(queue.Attributes)
	if normalizedAttributes == nil {
		normalizedAttributes = map[string]string{}
	}
	for key, value := range attributes {
		normalizedAttributes[trimName(key)] = value
	}

	queue.Attributes = normalizedAttributes
	queue.Recovery = queueRecoveryFromAttributes(normalizedAttributes)
	queue.OrderingHint = orderingHintFromAttributes(normalizedAttributes, queue.OrderingHint)
	queue.UpdatedAt = s.clock.Now()
	s.state.Queues[index] = queue

	if err := s.commitStateLocked(); err != nil {
		return contracts.QueueAttributesView{}, err
	}

	return contracts.QueueAttributesView{
		QueueName:  queueName,
		QueueURL:   queueURLForAccount(queueName, ""),
		QueueARN:   queueARNForAccount(queueName, ""),
		Attributes: cloneMap(normalizedAttributes),
	}, nil
}

func (s *Service) TagQueue(queueName string, tags map[string]string) error {
	queueName = trimName(queueName)
	if queueName == "" {
		return fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.activeQueueByNameLocked(queueName)
	if !ok {
		return errQueueNotFound
	}

	if s.state.QueueTags == nil {
		s.state.QueueTags = map[string]map[string]string{}
	}
	current := cloneMap(s.state.QueueTags[queueName])
	if current == nil {
		current = map[string]string{}
	}
	for key, value := range tags {
		key = trimName(key)
		if key == "" {
			continue
		}
		current[key] = value
	}
	s.state.QueueTags[queue.Name] = current
	queue.UpdatedAt = s.clock.Now()
	if index, _, ok := s.queueRecordByNameLocked(queueName); ok {
		s.state.Queues[index] = queue
	}
	return s.commitStateLocked()
}

func (s *Service) UntagQueue(queueName string, tagKeys []string) error {
	queueName = trimName(queueName)
	if queueName == "" {
		return fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.activeQueueByNameLocked(queueName)
	if !ok {
		return errQueueNotFound
	}

	current := cloneMap(s.state.QueueTags[queueName])
	if current == nil {
		current = map[string]string{}
	}
	for _, tagKey := range tagKeys {
		tagKey = trimName(tagKey)
		if tagKey == "" {
			continue
		}
		delete(current, tagKey)
	}
	s.state.QueueTags[queue.Name] = current
	queue.UpdatedAt = s.clock.Now()
	if index, _, ok := s.queueRecordByNameLocked(queueName); ok {
		s.state.Queues[index] = queue
	}
	return s.commitStateLocked()
}

func (s *Service) AddPermission(queueName, label string, awsAccountIDs, actions []string) error {
	queueName = trimName(queueName)
	label = trimName(label)
	if queueName == "" {
		return fmt.Errorf("sqs: queue name is required")
	}
	if label == "" {
		return fmt.Errorf("sqs: permission label is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.activeQueueByNameLocked(queueName)
	if !ok {
		return errQueueNotFound
	}

	if s.state.QueuePermissions == nil {
		s.state.QueuePermissions = map[string]map[string]domain.QueuePermission{}
	}
	permissions := s.state.QueuePermissions[queueName]
	if permissions == nil {
		permissions = map[string]domain.QueuePermission{}
	}

	now := s.clock.Now()
	permissions[label] = domain.QueuePermission{
		Label:         label,
		AWSAccountIDs: uniqueSortedTrimmedStrings(awsAccountIDs),
		Actions:       uniqueSortedTrimmedStrings(actions),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.state.QueuePermissions[queue.Name] = permissions
	queue.UpdatedAt = now
	if index, _, ok := s.queueRecordByNameLocked(queueName); ok {
		s.state.Queues[index] = queue
	}
	return s.commitStateLocked()
}

func (s *Service) RemovePermission(queueName, label string) error {
	queueName = trimName(queueName)
	label = trimName(label)
	if queueName == "" {
		return fmt.Errorf("sqs: queue name is required")
	}
	if label == "" {
		return fmt.Errorf("sqs: permission label is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.activeQueueByNameLocked(queueName)
	if !ok {
		return errQueueNotFound
	}

	permissions := cloneQueuePermissionMap(s.state.QueuePermissions[queueName])
	delete(permissions, label)
	if s.state.QueuePermissions == nil {
		s.state.QueuePermissions = map[string]map[string]domain.QueuePermission{}
	}
	s.state.QueuePermissions[queue.Name] = permissions
	queue.UpdatedAt = s.clock.Now()
	if index, _, ok := s.queueRecordByNameLocked(queueName); ok {
		s.state.Queues[index] = queue
	}
	return s.commitStateLocked()
}

func (s *Service) ListQueueTags(queueName string) (map[string]string, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return map[string]string{}, fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeQueueByNameLocked(queueName); !ok {
		return map[string]string{}, errQueueNotFound
	}
	return cloneMap(s.state.QueueTags[queueName]), nil
}

func (s *Service) ListDeadLetterSourceQueues(queueName string) ([]string, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return nil, fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeQueueByNameLocked(queueName); !ok {
		return nil, errQueueNotFound
	}

	sources := make([]string, 0)
	for _, queue := range s.state.ListQueues() {
		if queue.DeletedAt.IsZero() && queue.Recovery.DeadLetterQueue == queueName {
			sources = append(sources, queue.Name)
		}
	}
	sort.Strings(sources)
	return sources, nil
}

func (s *Service) StartMessageMoveTask(sourceArn, destinationArn string, maxNumberOfMessagesPerSecond int) (string, error) {
	sourceArn = trimName(sourceArn)
	destinationArn = trimName(destinationArn)
	if sourceArn == "" {
		return "", fmt.Errorf("sqs: source ARN is required")
	}
	if maxNumberOfMessagesPerSecond < 0 {
		return "", fmt.Errorf("sqs: max number of messages per second must be non-negative")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sourceQueue, ok := s.queueByARNLocked(sourceArn)
	if !ok {
		return "", errQueueNotFound
	}
	for _, task := range s.state.MoveTasks[sourceQueue.Name] {
		if strings.EqualFold(task.Status, "RUNNING") {
			return "", fmt.Errorf("sqs: a message move task is already running for %s", sourceQueue.Name)
		}
	}

	hasSourceQueue := false
	for _, queue := range s.state.ListQueues() {
		if queue.DeletedAt.IsZero() && queue.Recovery.DeadLetterQueue == sourceQueue.Name {
			hasSourceQueue = true
			break
		}
	}
	if !hasSourceQueue {
		return "", fmt.Errorf("sqs: source queue is not configured as a dead-letter queue")
	}

	if s.state.MoveTasks == nil {
		s.state.MoveTasks = map[string]map[string]domain.MessageMoveTask{}
	}
	tasks := s.state.MoveTasks[sourceQueue.Name]
	if tasks == nil {
		tasks = map[string]domain.MessageMoveTask{}
	}

	now := s.clock.Now()
	task := domain.MessageMoveTask{
		TaskHandle:                   sourceArn + "|" + uuid.NewString(),
		SourceQueue:                  sourceQueue.Name,
		SourceArn:                    sourceArn,
		DestinationArn:               destinationArn,
		MaxNumberOfMessagesPerSecond: maxNumberOfMessagesPerSecond,
		Status:                       "RUNNING",
		StartedAt:                    now,
		UpdatedAt:                    now,
	}
	tasks[task.TaskHandle] = task
	s.state.MoveTasks[sourceQueue.Name] = tasks
	if index, _, ok := s.queueRecordByNameLocked(sourceQueue.Name); ok {
		sourceQueue.UpdatedAt = now
		s.state.Queues[index] = sourceQueue
	}
	if err := s.commitStateLocked(); err != nil {
		return "", err
	}
	return task.TaskHandle, nil
}

func (s *Service) CancelMessageMoveTask(taskHandle string) (int64, error) {
	taskHandle = trimName(taskHandle)
	if taskHandle == "" {
		return 0, fmt.Errorf("sqs: task handle is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queueName, task, ok := s.findMessageMoveTaskLocked(taskHandle)
	if !ok {
		return 0, errQueueNotFound
	}
	now := s.clock.Now()
	task.Status = "CANCELLED"
	task.CancelledAt = now
	task.UpdatedAt = now
	if s.state.MoveTasks == nil {
		s.state.MoveTasks = map[string]map[string]domain.MessageMoveTask{}
	}
	tasks := cloneMessageMoveTaskMap(s.state.MoveTasks[queueName])
	tasks[taskHandle] = task
	s.state.MoveTasks[queueName] = tasks
	if err := s.commitStateLocked(); err != nil {
		return 0, err
	}
	return task.ApproximateNumberOfMessagesMoved, nil
}

func (s *Service) ListMessageMoveTasks(queueName string) ([]domain.MessageMoveTask, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return nil, fmt.Errorf("sqs: queue name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeQueueByNameLocked(queueName); !ok {
		return nil, errQueueNotFound
	}

	tasks := make([]domain.MessageMoveTask, 0, len(s.state.MoveTasks[queueName]))
	for _, task := range s.state.MoveTasks[queueName] {
		tasks = append(tasks, task)
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].StartedAt.Equal(tasks[j].StartedAt) {
			return tasks[i].TaskHandle < tasks[j].TaskHandle
		}
		return tasks[i].StartedAt.After(tasks[j].StartedAt)
	})
	return tasks, nil
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

func (s *Service) SendMessage(queueName string, request contracts.SendMessageRequest) (contracts.SendMessageResult, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return contracts.SendMessageResult{}, fmt.Errorf("sqs: queue name is required")
	}
	if trimName(request.MessageBody) == "" {
		return contracts.SendMessageResult{}, fmt.Errorf("sqs: message body is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.activeQueueByNameLocked(queueName)
	if !ok {
		return contracts.SendMessageResult{}, errQueueNotFound
	}

	message, result, err := s.enqueueMessageLocked(queueName, queue, request, "", "", 0, 1)
	if err != nil {
		return contracts.SendMessageResult{}, err
	}
	s.state.Messages = append(s.state.Messages, message)
	if err := s.commitStateLocked(); err != nil {
		return contracts.SendMessageResult{}, err
	}
	return result, nil
}

func (s *Service) SendMessageBatch(queueName string, request contracts.SendMessageBatchRequest) (contracts.SendMessageBatchResult, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return contracts.SendMessageBatchResult{}, fmt.Errorf("sqs: queue name is required")
	}
	if len(request.Entries) == 0 {
		return contracts.SendMessageBatchResult{}, errEmptyBatchRequest
	}
	if len(request.Entries) > 10 {
		return contracts.SendMessageBatchResult{}, errTooManyBatchEntries
	}
	if hasDuplicateBatchEntryIDsByID(request.Entries, func(entry contracts.SendMessageBatchRequestEntry) string { return entry.Id }) {
		return contracts.SendMessageBatchResult{}, errDuplicateBatchEntryIDs
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	queue, ok := s.activeQueueByNameLocked(queueName)
	if !ok {
		return contracts.SendMessageBatchResult{}, errQueueNotFound
	}

	result := contracts.SendMessageBatchResult{
		Successful: make([]contracts.SendMessageBatchResultEntry, 0, len(request.Entries)),
		Failed:     make([]contracts.BatchResultErrorEntry, 0),
	}
	pending := make([]domain.Message, 0, len(request.Entries))
	for index, entry := range request.Entries {
		entryResult, message, err := s.enqueueBatchMessageLocked(queueName, queue, entry, index, len(request.Entries))
		if err != nil {
			result.Failed = append(result.Failed, batchFailureEntry(entry.Id, err.Error(), true))
			continue
		}
		pending = append(pending, message)
		result.Successful = append(result.Successful, entryResult)
	}

	if len(pending) > 0 {
		s.state.Messages = append(s.state.Messages, pending...)
		if err := s.commitStateLocked(); err != nil {
			return contracts.SendMessageBatchResult{}, err
		}
	}

	return result, nil
}

func (s *Service) DeleteMessageBatch(queueName string, request contracts.DeleteMessageBatchRequest) (contracts.DeleteMessageBatchResult, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return contracts.DeleteMessageBatchResult{}, fmt.Errorf("sqs: queue name is required")
	}
	if len(request.Entries) == 0 {
		return contracts.DeleteMessageBatchResult{}, errEmptyBatchRequest
	}
	if len(request.Entries) > 10 {
		return contracts.DeleteMessageBatchResult{}, errTooManyBatchEntries
	}
	if hasDuplicateBatchEntryIDsByID(request.Entries, func(entry contracts.DeleteMessageBatchRequestEntry) string { return entry.Id }) {
		return contracts.DeleteMessageBatchResult{}, errDuplicateBatchEntryIDs
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeQueueByNameLocked(queueName); !ok {
		return contracts.DeleteMessageBatchResult{}, errQueueNotFound
	}

	result := contracts.DeleteMessageBatchResult{
		Successful: make([]contracts.DeleteMessageBatchResultEntry, 0, len(request.Entries)),
		Failed:     make([]contracts.BatchResultErrorEntry, 0),
	}
	deleted := false
	for _, entry := range request.Entries {
		id := trimName(entry.Id)
		receiptHandle := trimName(entry.ReceiptHandle)
		if id == "" || receiptHandle == "" {
			result.Failed = append(result.Failed, batchFailureEntry(entry.Id, "receipt handle is required", true))
			continue
		}

		idx, ok := s.findMessageByReceiptLocked(queueName, receiptHandle)
		if !ok {
			result.Failed = append(result.Failed, batchFailureEntry(entry.Id, "receipt handle is invalid", true))
			continue
		}

		s.state.Messages = append(s.state.Messages[:idx], s.state.Messages[idx+1:]...)
		deleted = true
		result.Successful = append(result.Successful, contracts.DeleteMessageBatchResultEntry{Id: id})
	}

	if deleted {
		if err := s.commitStateLocked(); err != nil {
			return contracts.DeleteMessageBatchResult{}, err
		}
	}

	return result, nil
}

func (s *Service) ChangeMessageVisibilityBatch(queueName string, request contracts.ChangeMessageVisibilityBatchRequest) (contracts.ChangeMessageVisibilityBatchResult, error) {
	queueName = trimName(queueName)
	if queueName == "" {
		return contracts.ChangeMessageVisibilityBatchResult{}, fmt.Errorf("sqs: queue name is required")
	}
	if len(request.Entries) == 0 {
		return contracts.ChangeMessageVisibilityBatchResult{}, errEmptyBatchRequest
	}
	if len(request.Entries) > 10 {
		return contracts.ChangeMessageVisibilityBatchResult{}, errTooManyBatchEntries
	}
	if hasDuplicateBatchEntryIDsByID(request.Entries, func(entry contracts.ChangeMessageVisibilityBatchRequestEntry) string { return entry.Id }) {
		return contracts.ChangeMessageVisibilityBatchResult{}, errDuplicateBatchEntryIDs
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.activeQueueByNameLocked(queueName); !ok {
		return contracts.ChangeMessageVisibilityBatchResult{}, errQueueNotFound
	}

	result := contracts.ChangeMessageVisibilityBatchResult{
		Successful: make([]contracts.ChangeMessageVisibilityBatchResultEntry, 0, len(request.Entries)),
		Failed:     make([]contracts.BatchResultErrorEntry, 0),
	}
	changed := false
	for _, entry := range request.Entries {
		id := trimName(entry.Id)
		receiptHandle := trimName(entry.ReceiptHandle)
		if id == "" || receiptHandle == "" {
			result.Failed = append(result.Failed, batchFailureEntry(entry.Id, "receipt handle is required", true))
			continue
		}

		if entry.VisibilityTimeout < 0 {
			result.Failed = append(result.Failed, batchFailureEntry(entry.Id, "visibility timeout must be non-negative", true))
			continue
		}

		if err := s.changeMessageVisibilityLocked(queueName, receiptHandle, time.Duration(entry.VisibilityTimeout)*time.Second); err != nil {
			result.Failed = append(result.Failed, batchFailureEntry(entry.Id, err.Error(), true))
			continue
		}

		changed = true
		result.Successful = append(result.Successful, contracts.ChangeMessageVisibilityBatchResultEntry{Id: id})
	}

	if changed {
		if err := s.commitStateLocked(); err != nil {
			return contracts.ChangeMessageVisibilityBatchResult{}, err
		}
	}

	return result, nil
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

func (s *Service) queueResponse(queueName string, queue domain.Queue, attributes map[string]string) domain.Queue {
	queue.Name = trimName(queue.Name)
	if queue.Name == "" {
		queue.Name = trimName(queueName)
	}
	queue.URL = queueURLForAccount(queue.Name, "")
	if len(attributes) == 0 && queue.Attributes == nil {
		queue.Attributes = map[string]string{}
		return queue
	}
	queue.Attributes = cloneMap(attributes)
	if queue.Attributes == nil {
		queue.Attributes = map[string]string{}
	}
	return queue
}

func (s *Service) queueRecordByNameLocked(name string) (int, domain.Queue, bool) {
	for idx, queue := range s.state.Queues {
		if queue.Name == name {
			return idx, queue, true
		}
	}
	return -1, domain.Queue{}, false
}

func (s *Service) queueByARNLocked(queueARN string) (domain.Queue, bool) {
	queueARN = trimName(queueARN)
	for _, queue := range s.state.Queues {
		if queue.DeletedAt.IsZero() && queueARNForAccount(queue.Name, "") == queueARN {
			return queue, true
		}
	}
	return domain.Queue{}, false
}

func (s *Service) findMessageMoveTaskLocked(taskHandle string) (string, domain.MessageMoveTask, bool) {
	for queueName, tasks := range s.state.MoveTasks {
		if task, ok := tasks[taskHandle]; ok {
			return queueName, task, true
		}
	}
	return "", domain.MessageMoveTask{}, false
}

func (s *Service) queueByNameLocked(name string) (domain.Queue, bool) {
	queue, ok := s.activeQueueByNameLocked(name)
	return queue, ok
}

func (s *Service) activeQueueByNameLocked(name string) (domain.Queue, bool) {
	for _, queue := range s.state.Queues {
		if queue.Name == name && queue.DeletedAt.IsZero() {
			return queue, true
		}
	}
	return domain.Queue{}, false
}

func (s *Service) removeMessagesForQueueLocked(queueName string) {
	filtered := s.state.Messages[:0]
	for _, message := range s.state.Messages {
		if trimName(message.Queue) == queueName {
			continue
		}
		filtered = append(filtered, message)
	}
	s.state.Messages = filtered

	if len(s.state.RecoveryMetadata) == 0 {
		return
	}
	for key, metadata := range s.state.RecoveryMetadata {
		if metadata.Queue == queueName || strings.HasPrefix(key, queueName+"/") {
			delete(s.state.RecoveryMetadata, key)
		}
	}
}

func ownerAccountMatches(ownerAccountID string) bool {
	ownerAccountID = trimName(ownerAccountID)
	if ownerAccountID == "" {
		return true
	}
	return ownerAccountID == awscontext.Default().AccountID
}

func selectQueueAttributes(attributes map[string]string, attributeNames []string, queueARN string) map[string]string {
	if len(attributes) == 0 && len(attributeNames) == 0 {
		return map[string]string{"QueueArn": queueARN}
	}

	selected := map[string]string{}
	includeAll := len(attributeNames) == 0
	for _, name := range attributeNames {
		if strings.EqualFold(trimName(name), "All") {
			includeAll = true
			break
		}
	}
	if includeAll {
		for key, value := range attributes {
			selected[key] = value
		}
	} else {
		allowed := map[string]struct{}{}
		for _, name := range attributeNames {
			allowed[trimName(name)] = struct{}{}
		}
		for key, value := range attributes {
			if _, ok := allowed[key]; ok {
				selected[key] = value
			}
		}
	}
	selected["QueueArn"] = queueARN
	return selected
}

func orderingHintFromAttributes(attributes map[string]string, fallback string) string {
	if strings.EqualFold(trimName(attributes["FifoQueue"]), "true") {
		return "fifo"
	}
	if strings.EqualFold(fallback, "fifo") {
		return "fifo"
	}
	return "standard"
}

func equalStringMaps(left, right map[string]string) bool {
	if len(left) != len(right) {
		if len(left) == 0 && len(right) == 0 {
			return true
		}
		return false
	}
	for key, leftValue := range left {
		if right[key] != leftValue {
			return false
		}
	}
	return true
}

func uniqueSortedTrimmedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = trimName(value)
		if value == "" {
			continue
		}
		seen[value] = struct{}{}
	}

	ordered := make([]string, 0, len(seen))
	for value := range seen {
		ordered = append(ordered, value)
	}
	sort.Strings(ordered)
	return ordered
}

func cloneQueuePermissionMap(values map[string]domain.QueuePermission) map[string]domain.QueuePermission {
	if values == nil {
		return map[string]domain.QueuePermission{}
	}

	cloned := make(map[string]domain.QueuePermission, len(values))
	for label, permission := range values {
		cloned[label] = domain.QueuePermission{
			Label:         permission.Label,
			AWSAccountIDs: append([]string(nil), permission.AWSAccountIDs...),
			Actions:       append([]string(nil), permission.Actions...),
			CreatedAt:     permission.CreatedAt,
			UpdatedAt:     permission.UpdatedAt,
		}
	}
	return cloned
}

func cloneMessageMoveTaskMap(values map[string]domain.MessageMoveTask) map[string]domain.MessageMoveTask {
	if values == nil {
		return map[string]domain.MessageMoveTask{}
	}

	cloned := make(map[string]domain.MessageMoveTask, len(values))
	for handle, task := range values {
		cloned[handle] = domain.MessageMoveTask{
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
	return cloned
}

func (s *Service) receiveReadyMessagesLocked(queueName string, maxMessages int, now time.Time) ([]domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sweepDeadLettersLocked(now)

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
		if trimName(message.Metadata["approximate_first_receive_timestamp"]) == "" {
			message.Metadata["approximate_first_receive_timestamp"] = strconv.FormatInt(now.UnixMilli(), 10)
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

func (s *Service) enqueueMessageLocked(queueName string, queue domain.Queue, request contracts.SendMessageRequest, batchID string, batchEntryID string, batchEntryIndex int, batchEntryCount int) (domain.Message, contracts.SendMessageResult, error) {
	messageGroupID := trimName(request.MessageGroupId)
	if IsFIFOQueue(queue) && messageGroupID == "" {
		return domain.Message{}, contracts.SendMessageResult{}, fmt.Errorf("sqs: message group id is required for fifo queues")
	}

	now := s.clock.Now()
	message := domain.Message{
		Queue:           queueName,
		MessageID:       uuid.NewString(),
		Body:            request.MessageBody,
		Attributes:      messageAttributesToStrings(request.MessageAttributes),
		Metadata:        messageSystemAttributesToStrings(request.MessageSystemAttributes),
		MessageGroupID:  messageGroupID,
		BatchID:         trimName(batchID),
		BatchEntryID:    trimName(batchEntryID),
		BatchEntryIndex: batchEntryIndex,
		BatchEntryCount: batchEntryCount,
		SentAt:          now,
	}

	effectiveDelaySeconds := request.DelaySeconds
	if effectiveDelaySeconds <= 0 {
		effectiveDelaySeconds = parseDelaySeconds(queue.Attributes["DelaySeconds"])
	}
	if effectiveDelaySeconds > 0 {
		message.AvailableAt = now.Add(time.Duration(effectiveDelaySeconds) * time.Second)
	}
	if message.Metadata == nil {
		message.Metadata = map[string]string{}
	}
	if request.MessageDeduplicationId != "" {
		message.Metadata["MessageDeduplicationId"] = request.MessageDeduplicationId
	}

	if IsFIFOQueue(queue) {
		message.SequenceNumber = s.nextSequenceNumberLocked(queueName, messageGroupID)
	}

	result := contracts.SendMessageResult{
		MD5OfMessageBody: md5OfString(request.MessageBody),
		MessageId:        message.MessageID,
	}
	if message.SequenceNumber > 0 {
		result.SequenceNumber = strconv.FormatInt(message.SequenceNumber, 10)
	}
	if len(request.MessageAttributes) > 0 {
		result.MD5OfMessageAttributes = md5OfMap(message.Attributes)
	}
	if len(request.MessageSystemAttributes) > 0 {
		result.MD5OfMessageSystemAttributes = md5OfMap(message.Metadata)
	}

	return message, result, nil
}

func (s *Service) enqueueBatchMessageLocked(queueName string, queue domain.Queue, entry contracts.SendMessageBatchRequestEntry, batchIndex int, batchCount int) (contracts.SendMessageBatchResultEntry, domain.Message, error) {
	if trimName(entry.Id) == "" {
		return contracts.SendMessageBatchResultEntry{}, domain.Message{}, fmt.Errorf("sqs: batch entry id is required")
	}
	if trimName(entry.MessageBody) == "" {
		return contracts.SendMessageBatchResultEntry{}, domain.Message{}, fmt.Errorf("sqs: message body is required")
	}

	message, result, err := s.enqueueMessageLocked(queueName, queue, contracts.SendMessageRequest{
		DelaySeconds:            entry.DelaySeconds,
		MessageAttributes:       entry.MessageAttributes,
		MessageBody:             entry.MessageBody,
		MessageDeduplicationId:  entry.MessageDeduplicationId,
		MessageGroupId:          entry.MessageGroupId,
		MessageSystemAttributes: entry.MessageSystemAttributes,
	}, "", entry.Id, batchIndex, batchCount)
	if err != nil {
		return contracts.SendMessageBatchResultEntry{}, domain.Message{}, err
	}

	return contracts.SendMessageBatchResultEntry{
		Id:                           trimName(entry.Id),
		MD5OfMessageAttributes:       result.MD5OfMessageAttributes,
		MD5OfMessageBody:             result.MD5OfMessageBody,
		MD5OfMessageSystemAttributes: result.MD5OfMessageSystemAttributes,
		MessageId:                    result.MessageId,
		SequenceNumber:               result.SequenceNumber,
	}, message, nil
}

func (s *Service) changeMessageVisibilityLocked(queueName, receiptHandle string, visibility time.Duration) error {
	if visibility < 0 {
		return errInvalidVisibilityWindow
	}
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
		return nil
	}

	message.ReceivedAt = s.clock.Now()
	message.Metadata[leaseVisibilityTimeoutMetaKey] = strconv.FormatInt(int64(visibility/time.Second), 10)
	return nil
}

func (s *Service) nextSequenceNumberLocked(queueName, groupID string) int64 {
	var maxSequence int64
	for _, message := range s.state.Messages {
		if trimName(message.Queue) != queueName {
			continue
		}
		if trimName(groupID) != "" && trimName(message.MessageGroupID) != trimName(groupID) {
			continue
		}
		if message.SequenceNumber > maxSequence {
			maxSequence = message.SequenceNumber
		}
	}
	return maxSequence + 1
}

func md5OfString(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func md5OfMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	builder := strings.Builder{}
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(values[key])
		builder.WriteString(";")
	}
	return md5OfString(builder.String())
}

func messageAttributesToStrings(attributes map[string]contracts.MessageAttributeValue) map[string]string {
	if len(attributes) == 0 {
		return nil
	}

	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make(map[string]string, len(keys))
	for _, key := range keys {
		value := attributes[key]
		switch {
		case value.StringValue != "":
			values[key] = value.StringValue
		case len(value.BinaryValue) > 0:
			values[key] = string(value.BinaryValue)
		case value.DataType != "":
			values[key] = value.DataType
		default:
			values[key] = ""
		}
	}
	return values
}

func messageSystemAttributesToStrings(attributes map[string]contracts.MessageAttributeValue) map[string]string {
	if len(attributes) == 0 {
		return nil
	}

	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make(map[string]string, len(keys))
	for _, key := range keys {
		value := attributes[key]
		switch {
		case value.StringValue != "":
			values[key] = value.StringValue
		case len(value.BinaryValue) > 0:
			values[key] = string(value.BinaryValue)
		case value.DataType != "":
			values[key] = value.DataType
		default:
			values[key] = ""
		}
	}
	return values
}

func hasDuplicateBatchEntryIDsByID[T any](entries []T, getID func(T) string) bool {
	seen := map[string]struct{}{}
	for _, entry := range entries {
		id := trimName(getID(entry))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}

func batchFailureEntry(id, message string, senderFault bool) contracts.BatchResultErrorEntry {
	return contracts.BatchResultErrorEntry{
		Code:        "InvalidParameterValue",
		Id:          trimName(id),
		Message:     message,
		SenderFault: senderFault,
	}
}

func (s *Service) findMessageByReceiptLocked(queueName, receiptHandle string) (int, bool) {
	if _, ok := s.activeQueueByNameLocked(queueName); !ok {
		return -1, false
	}
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

func parseDelaySeconds(raw string) int {
	raw = trimName(raw)
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return 0
	}
	return seconds
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

func queueRecoveryFromAttributes(attributes map[string]string) domain.QueueRecovery {
	recovery := domain.QueueRecovery{
		Policy: map[string]string{},
	}
	if len(attributes) == 0 {
		return recovery
	}

	rawPolicy := trimName(attributes["RedrivePolicy"])
	if rawPolicy == "" {
		return recovery
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(rawPolicy), &parsed); err != nil {
		recovery.Policy["raw"] = rawPolicy
		return recovery
	}

	for key, value := range parsed {
		normalizedKey := camelToSnake(key)
		recovery.Policy[normalizedKey] = fmt.Sprint(value)
	}
	if targetArn := trimName(recovery.Policy["dead_letter_target_arn"]); targetArn != "" {
		if queueName, _, err := queueNameAndAccountFromARN(targetArn); err == nil {
			recovery.DeadLetterQueue = queueName
		}
	}
	return recovery
}

func camelToSnake(value string) string {
	if value == "" {
		return ""
	}

	var out strings.Builder
	for i, r := range value {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out.WriteByte('_')
		}
		out.WriteRune(r)
	}
	return strings.ToLower(out.String())
}

func queueNameAndAccountFromARN(raw string) (string, string, error) {
	trimmed := trimName(raw)
	if trimmed == "" {
		return "", "", fmt.Errorf("sqs: arn is required")
	}

	parts := strings.Split(trimmed, ":")
	if len(parts) < 6 || parts[0] != "arn" {
		return "", "", fmt.Errorf("sqs: invalid arn: %s", raw)
	}

	accountID := trimName(parts[len(parts)-2])
	queueName := trimName(parts[len(parts)-1])
	if accountID == "" || queueName == "" {
		return "", "", fmt.Errorf("sqs: invalid arn: %s", raw)
	}
	return queueName, accountID, nil
}

func queueURLForAccount(queueName, ownerAccountID string) string {
	queueName = trimName(queueName)
	if queueName == "" {
		return ""
	}

	aws := awscontext.Default()
	if ownerAccountID = trimName(ownerAccountID); ownerAccountID != "" {
		aws = aws.WithAccountID(ownerAccountID)
	}
	endpoint := strings.TrimRight(strings.TrimSpace(aws.Endpoint), "/")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:4566"
	}
	return fmt.Sprintf("%s/%s/%s", endpoint, aws.AccountID, queueName)
}

func queueARNForAccount(queueName, ownerAccountID string) string {
	queueName = trimName(queueName)
	if queueName == "" {
		return ""
	}

	aws := awscontext.Default()
	if ownerAccountID = trimName(ownerAccountID); ownerAccountID != "" {
		aws = aws.WithAccountID(ownerAccountID)
	}
	return aws.ServiceARN("sqs", queueName)
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

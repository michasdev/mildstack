package application

import (
	"context"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/domain"
)

type worker struct {
	service *Service
	clock   Clock
	tick    time.Duration
}

type leaseSnapshot struct {
	Queue   string
	Message string
	ReadyIn time.Duration
}

func newWorker(service *Service, clock Clock) *worker {
	if clock == nil {
		clock = realClock{}
	}
	return &worker{
		service: service,
		clock:   clock,
		tick:    workerPollInterval,
	}
}

func (w *worker) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		now := w.clock.Now()
		wait := w.poll(now)
		if wait <= 0 || wait > w.tick {
			wait = w.tick
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		w.advance(w.clock.Now())
	}
}

func (w *worker) poll(now time.Time) time.Duration {
	w.service.mu.Lock()
	defer w.service.mu.Unlock()

	next := w.tick
	for _, message := range w.service.state.Messages {
		queue, ok := w.service.queueByNameLocked(message.Queue)
		if !ok {
			continue
		}

		if IsDelayed(message, now) {
			if wait := message.AvailableAt.Sub(now); wait >= 0 && wait < next {
				next = wait
			}
			continue
		}

		if IsInvisible(message, queue, now) {
			if wait := leaseDeadline(message, queue).Sub(now); wait >= 0 && wait < next {
				next = wait
			}
		}
	}

	if next < 0 {
		return 0
	}
	return next
}

func (w *worker) lease(now time.Time) []leaseSnapshot {
	w.service.mu.Lock()
	defer w.service.mu.Unlock()

	leases := make([]leaseSnapshot, 0)
	for _, message := range w.service.state.Messages {
		queue, ok := w.service.queueByNameLocked(message.Queue)
		if !ok {
			continue
		}
		if !IsInvisible(message, queue, now) {
			continue
		}
		readyIn := leaseDeadline(message, queue).Sub(now)
		if readyIn < 0 {
			readyIn = 0
		}
		leases = append(leases, leaseSnapshot{
			Queue:   message.Queue,
			Message: message.MessageID,
			ReadyIn: readyIn,
		})
	}
	return leases
}

func (w *worker) redeliver(now time.Time) []domain.Message {
	w.service.mu.Lock()
	defer w.service.mu.Unlock()

	redeliverable := make([]domain.Message, 0)
	for _, message := range w.service.state.Messages {
		queue, ok := w.service.queueByNameLocked(message.Queue)
		if !ok {
			continue
		}
		if CanRedeliver(message, queue, now) && !IsDeadLetterEligible(message, queue, now) {
			redeliverable = append(redeliverable, cloneMessage(message))
		}
	}
	return redeliverable
}

func (w *worker) advance(now time.Time) int {
	w.service.mu.Lock()
	defer w.service.mu.Unlock()
	return w.service.sweepDeadLettersLocked(now)
}

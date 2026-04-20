package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func Routes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/sqs/queues",
			Name:   "sqs.queues.index",
		},
		{
			Method: "POST",
			Path:   "/sqs/queues",
			Name:   "sqs.queues.create",
		},
		{
			Method: "GET",
			Path:   "/sqs/queues/:queue",
			Name:   "sqs.queues.show",
		},
		{
			Method: "DELETE",
			Path:   "/sqs/queues/:queue",
			Name:   "sqs.queues.delete",
		},
		{
			Method: "GET",
			Path:   "/sqs/queues/:queue/messages",
			Name:   "sqs.messages.receive",
		},
		{
			Method: "POST",
			Path:   "/sqs/queues/:queue/messages",
			Name:   "sqs.messages.send",
		},
		{
			Method: "DELETE",
			Path:   "/sqs/queues/:queue/messages/:receiptHandle",
			Name:   "sqs.messages.delete",
		},
	}
}

package contracts

import "errors"

// ErrSQSOperationDeferred marks queue operations that are routed but not yet
// behaviorally implemented in the current phase.
var ErrSQSOperationDeferred = errors.New("sqs: operation deferred")

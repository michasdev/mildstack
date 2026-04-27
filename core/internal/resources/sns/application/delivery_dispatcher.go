package application

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
	sqscontracts "github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
)

func (s *Service) dispatchDelivery(message domain.PublishedMessage, hasTopic bool, topic domain.Topic) error {
	if !hasTopic {
		return nil
	}

	nextToken := ""
	for {
		subscriptions, responseNextToken, err := s.subscriptionRepository().ListByTopic(topic.TenantKey, topic.ARN, nextToken, 100)
		if err != nil {
			return err
		}

		for _, subscription := range subscriptions {
			if !subscription.IsConfirmed() {
				continue
			}
			if err := s.dispatchToSubscription(message, subscription); err != nil {
				return err
			}
		}

		nextToken = strings.TrimSpace(responseNextToken)
		if nextToken == "" {
			return nil
		}
	}
}

func (s *Service) dispatchToSubscription(message domain.PublishedMessage, subscription domain.Subscription) error {
	startedAt := time.Now().UTC()

	attempt, err := domain.NewDeliveryAttempt(
		message.MessageID,
		message.TenantKey,
		subscription.ARN,
		"",
		subscription.Protocol,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}

	matched, err := domain.EvaluateSubscriptionFilter(subscription.FilterPolicy(), subscription.FilterPolicyScope(), message)
	if err != nil {
		attempt, _ = attempt.MarkFailed("InvalidParameter", err.Error(), time.Now().UTC())
		attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{
			"endpoint":          subscription.Endpoint,
			"filterPolicy":      subscription.FilterPolicy(),
			"filterPolicyScope": subscription.FilterPolicyScope(),
		})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}
	if !matched {
		attempt, _ = attempt.MarkFilteredOut(time.Now().UTC())
		attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{
			"endpoint":          subscription.Endpoint,
			"filterPolicy":      subscription.FilterPolicy(),
			"filterPolicyScope": subscription.FilterPolicyScope(),
		})
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{"status": domain.DeliveryAttemptStatusFilteredOut})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}

	if strings.EqualFold(subscription.Protocol, "sqs") {
		return s.dispatchToSQSSubscription(message, subscription, attempt, startedAt)
	}

	if !strings.EqualFold(subscription.Protocol, "http") && !strings.EqualFold(subscription.Protocol, "https") {
		attempt, _ = attempt.MarkSkipped("ProtocolDeferred", "Protocol delivery is simulated in local runtime.", time.Now().UTC())
		attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{"endpoint": subscription.Endpoint})
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{"status": domain.DeliveryAttemptStatusSkipped})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}

	if !isLocalEndpoint(subscription.Endpoint) {
		attempt, _ = attempt.MarkSkipped("EndpointDeferred", "Non-local endpoints are simulated in local runtime.", time.Now().UTC())
		attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{"endpoint": subscription.Endpoint})
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{"status": domain.DeliveryAttemptStatusSkipped})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}

	payload, contentType, err := snsDeliveryPayload(subscription, message)
	if err != nil {
		attempt, _ = attempt.MarkFailed("InvalidParameter", err.Error(), time.Now().UTC())
		attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{"endpoint": subscription.Endpoint})
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{"status": domain.DeliveryAttemptStatusFailed})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}

	statusCode, responseBody, err := dispatchLocalHTTPSDelivery(subscription.Endpoint, payload, contentType, nil)
	attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{
		"endpoint":           subscription.Endpoint,
		"protocol":           subscription.Protocol,
		"rawMessageDelivery": subscription.RawMessageDeliveryEnabled(),
		"contentType":        contentType,
		"payload":            string(payload),
	})

	if err != nil {
		attempt, _ = attempt.MarkFailed("EndpointConnectionError", err.Error(), time.Now().UTC())
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{
			"statusCode": 0,
			"error":      err.Error(),
		})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}

	attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{
		"statusCode": statusCode,
		"body":       responseBody,
	})
	if statusCode >= 200 && statusCode < 300 {
		attempt, _ = attempt.MarkDelivered(time.Now().UTC())
	} else {
		attempt, _ = attempt.MarkFailed("EndpointResponse", fmt.Sprintf("endpoint returned status %d", statusCode), time.Now().UTC())
	}
	return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
}

func (s *Service) dispatchToSQSSubscription(message domain.PublishedMessage, subscription domain.Subscription, attempt domain.DeliveryAttempt, startedAt time.Time) error {
	queueName := queueNameFromARN(subscription.Endpoint)
	if queueName == "" {
		attempt, _ = attempt.MarkFailed("InvalidParameterException", "Invalid SQS endpoint ARN.", time.Now().UTC())
		attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{"endpoint": subscription.Endpoint})
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{"status": domain.DeliveryAttemptStatusFailed})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}

	if s.sqsBridge == nil {
		attempt, _ = attempt.MarkSkipped("ProtocolDeferred", "SQS bridge is not available in this runtime.", time.Now().UTC())
		attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{"endpoint": subscription.Endpoint, "queue": queueName})
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{"status": domain.DeliveryAttemptStatusSkipped})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}

	payload, _, err := snsDeliveryPayload(subscription, message)
	if err != nil {
		attempt, _ = attempt.MarkFailed("InvalidParameterException", err.Error(), time.Now().UTC())
		attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{"endpoint": subscription.Endpoint, "queue": queueName})
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{"status": domain.DeliveryAttemptStatusFailed})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}

	request := sqscontracts.SendMessageRequest{
		MessageBody:            string(payload),
		MessageGroupId:         strings.TrimSpace(message.MessageGroupID),
		MessageDeduplicationId: strings.TrimSpace(message.MessageDeduplicationID),
	}
	result, err := s.sqsBridge.SendMessage(queueName, request)
	attempt.RequestSnapshotJSON = mustMarshalJSON(map[string]any{
		"endpoint":           subscription.Endpoint,
		"queue":              queueName,
		"rawMessageDelivery": subscription.RawMessageDeliveryEnabled(),
		"payload":            string(payload),
	})
	if err != nil {
		attempt, _ = attempt.MarkFailed("EndpointConnectionError", err.Error(), time.Now().UTC())
		attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{
			"statusCode": 0,
			"error":      err.Error(),
		})
		return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
	}
	attempt.ResponseSnapshotJSON = mustMarshalJSON(map[string]any{
		"status":    domain.DeliveryAttemptStatusDelivered,
		"messageId": result.MessageId,
	})
	attempt, _ = attempt.MarkDelivered(time.Now().UTC())
	return s.persistDeliveryAttempt(message.MessageID, subscription.Endpoint, attempt, startedAt)
}

func (s *Service) persistDeliveryAttempt(messageID, endpoint string, attempt domain.DeliveryAttempt, startedAt time.Time) error {
	err := s.publishRepository().SaveDeliveryAttempt(attempt)
	if s != nil && s.observability != nil {
		s.observability.recordDelivery(attempt.Status, attempt.Protocol, attempt.FailureCode, time.Since(startedAt), err)
		s.syncObservabilitySnapshot()
	}

	log.Printf(
		"sns delivery attempt message_id=%s subscription_arn=%s endpoint=%s status=%s failure_code=%s persisted=%t",
		strings.TrimSpace(messageID),
		strings.TrimSpace(attempt.SubscriptionARN),
		strings.TrimSpace(endpoint),
		strings.TrimSpace(attempt.Status),
		strings.TrimSpace(attempt.FailureCode),
		err == nil,
	)

	return err
}

func snsDeliveryPayload(subscription domain.Subscription, message domain.PublishedMessage) ([]byte, string, error) {
	if subscription.RawMessageDeliveryEnabled() {
		return []byte(message.Payload), "text/plain; charset=utf-8", nil
	}

	envelope := map[string]any{
		"Type":             "Notification",
		"MessageId":        message.MessageID,
		"TopicArn":         message.TopicARN,
		"Message":          message.Payload,
		"Timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
		"SignatureVersion": "1",
	}
	if strings.TrimSpace(message.Subject) != "" {
		envelope["Subject"] = message.Subject
	}
	if len(message.MessageAttributes) > 0 {
		attributes := map[string]map[string]string{}
		for key, value := range message.MessageAttributes {
			entryValue := strings.TrimSpace(value.StringValue)
			if entryValue == "" {
				entryValue = strings.TrimSpace(value.BinaryValue)
			}
			attributes[key] = map[string]string{
				"Type":  value.DataType,
				"Value": entryValue,
			}
		}
		envelope["MessageAttributes"] = attributes
	}

	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, "", fmt.Errorf("sns: marshal delivery payload: %w", err)
	}
	return encoded, "application/json", nil
}

func dispatchLocalHTTPSDelivery(endpoint string, payload []byte, contentType string, headers map[string]string) (int, string, error) {
	requestURL, authHeader := sanitizeEndpointForRequest(endpoint)
	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	request.Header.Set("Content-Type", contentType)
	if strings.TrimSpace(authHeader) != "" {
		request.Header.Set("Authorization", authHeader)
	}
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		request.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body), nil
}

func isLocalEndpoint(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func queueNameFromARN(rawARN string) string {
	parts := strings.Split(strings.TrimSpace(rawARN), ":")
	if len(parts) < 6 || parts[0] != "arn" {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func sanitizeEndpointForRequest(endpoint string) (string, string) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed == nil {
		return endpoint, ""
	}

	authHeader := ""
	if parsed.User != nil {
		username := parsed.User.Username()
		password, _ := parsed.User.Password()
		parsed.User = nil
		token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		authHeader = "Basic " + token
	}
	return parsed.String(), authHeader
}

func (s *Service) deliverSubscriptionConfirmation(subscription domain.Subscription) error {
	if !strings.EqualFold(subscription.Protocol, "http") && !strings.EqualFold(subscription.Protocol, "https") {
		return nil
	}
	if !isLocalEndpoint(subscription.Endpoint) {
		return nil
	}

	confirmURL := fmt.Sprintf("https://sns.localhost/?Action=ConfirmSubscription&TopicArn=%s&Token=%s", url.QueryEscape(subscription.TopicARN), url.QueryEscape(subscription.Token))
	payload, err := json.Marshal(map[string]any{
		"Type":         "SubscriptionConfirmation",
		"MessageId":    subscription.ARN,
		"Token":        subscription.Token,
		"TopicArn":     subscription.TopicARN,
		"Message":      "You have chosen to subscribe to the topic.",
		"SubscribeURL": confirmURL,
		"Timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return err
	}

	headers := map[string]string{
		"x-amz-sns-message-type": "SubscriptionConfirmation",
		"x-amz-sns-topic-arn":    subscription.TopicARN,
	}
	_, _, err = dispatchLocalHTTPSDelivery(subscription.Endpoint, payload, "text/plain; charset=UTF-8", headers)
	return err
}

func mustMarshalJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

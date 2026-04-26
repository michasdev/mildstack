package http

import (
	"errors"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

// SNSNativeService describes the minimum SNS service contract required by
// the native AWS Query/XML adapter lifecycle.
type SNSNativeService interface {
	Policy() orchestrator.EmulationPolicy
	Metadata() orchestrator.Metadata
	CreateTopic(name string, attributes map[string]string) (domain.Topic, error)
	DeleteTopic(topicARN string) error
	GetTopicAttributes(topicARN string) (map[string]string, error)
	SetTopicAttributes(topicARN, attributeName, attributeValue string) (map[string]string, error)
	ListTopics(nextToken string) ([]domain.Topic, string, error)
	Subscribe(topicARN, protocol, endpoint string, attributes map[string]string, returnSubscriptionARN bool) (domain.SubscribeOutput, error)
	ConfirmSubscription(topicARN, token string) (domain.Subscription, error)
	Unsubscribe(subscriptionARN string) error
	GetSubscriptionAttributes(subscriptionARN string) (map[string]string, error)
	SetSubscriptionAttributes(subscriptionARN, attributeName, attributeValue string) (map[string]string, error)
	ListSubscriptions(nextToken string) ([]domain.Subscription, string, error)
	ListSubscriptionsByTopic(topicARN, nextToken string) ([]domain.Subscription, string, error)
	Publish(request domain.PublishRequest) (domain.PublishResult, error)
	PublishBatch(request domain.PublishBatchRequest) (domain.PublishBatchResult, error)
}

type servicesHandler struct {
	snapshotter Snapshotter
	registrar   *Registrar
}

var errServiceRoutesNotRegistered = errors.New("service routes not registered")

type servicesResponse struct {
	Services []serviceSummary `json:"services"`
}

type serviceResponse struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Tags        []string       `json:"tags"`
	Routes      []serviceRoute `json:"routes"`
}

type serviceSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags"`
	RouteCount  int      `json:"route_count"`
}

type serviceRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}

func newServicesHandler(snapshotter Snapshotter, registrar *Registrar) servicesHandler {
	return servicesHandler{
		snapshotter: snapshotter,
		registrar:   registrar,
	}
}

func (h servicesHandler) handleIndex(c *gin.Context) {
	snapshot := h.snapshotter.Snapshot(c.Request.Context())
	summaries, err := h.buildSummaries(snapshot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servicesResponse{Services: summaries})
}

func (h servicesHandler) handleService(c *gin.Context) {
	serviceName := c.Param("service")
	snapshot := h.snapshotter.Snapshot(c.Request.Context())
	metadata, ok := serviceMetadataByName(snapshot.Services, serviceName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}

	entry, err := h.serviceRoutesForName(serviceName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, serviceResponse{
		Name:        metadata.Name,
		Description: metadata.Description,
		Version:     metadata.Version,
		Tags:        append([]string(nil), metadata.Tags...),
		Routes:      copyServiceRoutes(entry.Routes),
	})
}

func (h servicesHandler) buildSummaries(snapshot runtime.Snapshot) ([]serviceSummary, error) {
	metadata := cloneAndSortMetadata(snapshot.Services)
	summaries := make([]serviceSummary, 0, len(metadata))
	for _, service := range metadata {
		entry, err := h.serviceRoutesForName(service.Name)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, serviceSummary{
			Name:        service.Name,
			Description: service.Description,
			Version:     service.Version,
			Tags:        append([]string(nil), service.Tags...),
			RouteCount:  len(entry.Routes),
		})
	}
	return summaries, nil
}

func (h servicesHandler) serviceRoutesForName(serviceName string) (ServiceCatalogEntry, error) {
	entry, ok := h.registrar.Service(serviceName)
	if !ok {
		return ServiceCatalogEntry{}, errServiceRoutesNotRegistered
	}
	return entry, nil
}

func cloneAndSortMetadata(services []orchestrator.Metadata) []orchestrator.Metadata {
	copied := make([]orchestrator.Metadata, len(services))
	for i, service := range services {
		copied[i] = orchestrator.Metadata{
			Name:        service.Name,
			Description: service.Description,
			Version:     service.Version,
			Tags:        append([]string(nil), service.Tags...),
		}
	}

	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].Name < copied[j].Name
	})
	return copied
}

func serviceMetadataByName(services []orchestrator.Metadata, name string) (orchestrator.Metadata, bool) {
	for _, service := range services {
		if service.Name == name {
			return orchestrator.Metadata{
				Name:        service.Name,
				Description: service.Description,
				Version:     service.Version,
				Tags:        append([]string(nil), service.Tags...),
			}, true
		}
	}
	return orchestrator.Metadata{}, false
}

func copyServiceRoutes(routes []RegisteredRoute) []serviceRoute {
	copied := make([]serviceRoute, len(routes))
	for i, route := range routes {
		copied[i] = serviceRoute{
			Method: route.Method,
			Path:   route.Path,
			Name:   route.Name,
		}
	}
	return copied
}

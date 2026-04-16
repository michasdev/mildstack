package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

type runtimeHandler struct {
	snapshotter Snapshotter
}

type runtimeResponse struct {
	Services []runtimeService `json:"services"`
	Ports    []int            `json:"ports"`
}

type runtimeService struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags"`
}

func newRuntimeHandler(snapshotter Snapshotter) runtimeHandler {
	return runtimeHandler{snapshotter: snapshotter}
}

func (h runtimeHandler) handleInfo(c *gin.Context) {
	snapshot := h.snapshotter.Snapshot(c.Request.Context())
	c.JSON(http.StatusOK, runtimeResponse{
		Services: copyRuntimeServices(snapshot.Services),
		Ports:    append([]int(nil), snapshot.Ports...),
	})
}

func copyRuntimeServices(services []orchestrator.Metadata) []runtimeService {
	copied := make([]runtimeService, len(services))
	for i, service := range services {
		copied[i] = runtimeService{
			Name:        service.Name,
			Description: service.Description,
			Version:     service.Version,
			Tags:        append([]string(nil), service.Tags...),
		}
	}
	return copied
}

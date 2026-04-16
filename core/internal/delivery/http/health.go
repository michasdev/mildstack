package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type healthHandler struct {
	snapshotter Snapshotter
}

type healthResponse struct {
	Status string `json:"status"`
}

type readinessResponse struct {
	Status   string           `json:"status"`
	Services []runtimeService `json:"services"`
	Ports    []int            `json:"ports"`
}

func newHealthHandler(snapshotter Snapshotter) healthHandler {
	return healthHandler{snapshotter: snapshotter}
}

func (h healthHandler) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}

func (h healthHandler) handleReady(c *gin.Context) {
	snapshot := h.snapshotter.Snapshot(c.Request.Context())
	response := readinessResponse{
		Status:   "not_ready",
		Services: copyRuntimeServices(snapshot.Services),
		Ports:    copyRuntimePorts(snapshot.Ports),
	}

	if len(snapshot.Ports) > 0 {
		response.Status = "ready"
		c.JSON(http.StatusOK, response)
		return
	}

	c.JSON(http.StatusServiceUnavailable, response)
}

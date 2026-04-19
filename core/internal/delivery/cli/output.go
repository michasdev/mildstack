package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type OutputMode string

const (
	OutputModeHuman OutputMode = "human"
	OutputModeJSON  OutputMode = "json"
)

func resolveOutputMode(cmd *cobra.Command) OutputMode {
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil || !jsonOutput {
		return OutputModeHuman
	}
	return OutputModeJSON
}

type statusPayload struct {
	State     string            `json:"state"`
	Services  []servicePayload  `json:"services"`
	Instances []instancePayload `json:"instances"`
	Ports     []int             `json:"ports"`
}

type servicePayload struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags"`
}

type portsPayload struct {
	Ports []int `json:"ports"`
}

type instancePayload struct {
	InstanceID string `json:"instanceId,omitempty"`
	Port       int    `json:"port"`
	PID        int    `json:"pid,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

func renderJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\"error\":%q}\n", err.Error())
	}
	return string(data) + "\n"
}

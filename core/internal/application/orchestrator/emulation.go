package orchestrator

import "fmt"

type Fidelity string

const (
	FidelityExemplar    Fidelity = "exemplar"
	FidelityPartial     Fidelity = "partial"
	FidelityUnsupported Fidelity = "unsupported"
)

type EmulationPolicy struct {
	Fidelity    Fidelity
	Supported   []string
	Unsupported []string
	ErrorPrefix string
}

func NewEmulationPolicy(fidelity Fidelity, supported, unsupported []string, errorPrefix string) EmulationPolicy {
	return EmulationPolicy{
		Fidelity:    fidelity,
		Supported:   cloneStrings(supported),
		Unsupported: cloneStrings(unsupported),
		ErrorPrefix: errorPrefix,
	}
}

func (p EmulationPolicy) Clone() EmulationPolicy {
	return EmulationPolicy{
		Fidelity:    p.Fidelity,
		Supported:   cloneStrings(p.Supported),
		Unsupported: cloneStrings(p.Unsupported),
		ErrorPrefix: p.ErrorPrefix,
	}
}

func UnsupportedError(policy EmulationPolicy, operation string) error {
	return fmt.Errorf("%s: unsupported operation %s", policy.ErrorPrefix, operation)
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

package cli

import "charm.land/lipgloss/v2"

type Theme struct {
	TitleLabel     string
	StateLabel     string
	ServicesLabel  string
	InstancesLabel string
	PortsLabel     string
	EmptyLabel     string
	Indent         string
	TitleStyle     lipgloss.Style
	LabelStyle     lipgloss.Style
	SectionStyle   lipgloss.Style
	AccentStyle    lipgloss.Style
	EmptyStyle     lipgloss.Style
}

func DefaultTheme() Theme {
	violet := lipgloss.Color("#8b5cf6")
	softViolet := lipgloss.Color("#a78bfa")
	muted := lipgloss.Color("#9ca3af")

	return Theme{
		TitleLabel:     "Runtime Status",
		StateLabel:     "State",
		ServicesLabel:  "Services",
		InstancesLabel: "Instances",
		PortsLabel:     "Ports",
		EmptyLabel:     "(none)",
		Indent:         "  ",
		TitleStyle:     lipgloss.NewStyle().Bold(true).Foreground(violet),
		LabelStyle:     lipgloss.NewStyle().Bold(true).Foreground(softViolet),
		SectionStyle:   lipgloss.NewStyle().Bold(true).Foreground(violet),
		AccentStyle:    lipgloss.NewStyle().Bold(true).Foreground(softViolet),
		EmptyStyle:     lipgloss.NewStyle().Foreground(muted),
	}
}

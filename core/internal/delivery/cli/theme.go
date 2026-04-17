package cli

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	TitleLabel    string
	StateLabel    string
	ServicesLabel string
	PortsLabel    string
	EmptyLabel    string
	Indent        string
	TitleStyle    lipgloss.Style
	LabelStyle    lipgloss.Style
	SectionStyle  lipgloss.Style
	AccentStyle   lipgloss.Style
	EmptyStyle    lipgloss.Style
}

func DefaultTheme() Theme {
	softGreen := lipgloss.Color("120")
	green := lipgloss.Color("114")
	muted := lipgloss.Color("245")

	return Theme{
		TitleLabel:    "Runtime Status",
		StateLabel:    "State",
		ServicesLabel: "Services",
		PortsLabel:    "Ports",
		EmptyLabel:    "(none)",
		Indent:        "  ",
		TitleStyle:    lipgloss.NewStyle().Bold(true).Foreground(softGreen),
		LabelStyle:    lipgloss.NewStyle().Bold(true).Foreground(green),
		SectionStyle:  lipgloss.NewStyle().Bold(true).Foreground(softGreen),
		AccentStyle:   lipgloss.NewStyle().Bold(true).Foreground(green),
		EmptyStyle:    lipgloss.NewStyle().Foreground(muted),
	}
}

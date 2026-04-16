package cli

type Theme struct {
	TitleLabel    string
	StateLabel    string
	ServicesLabel string
	PortsLabel    string
	EmptyLabel    string
	Indent        string
}

func DefaultTheme() Theme {
	return Theme{
		TitleLabel:    "Runtime Status",
		StateLabel:    "State",
		ServicesLabel: "Services",
		PortsLabel:    "Ports",
		EmptyLabel:    "(none)",
		Indent:        "  ",
	}
}

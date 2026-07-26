package core

type CommandMatch struct {
	Name                 string
	MatchKey             string                  `json:"-"`
	ShowOnlyWhenSelected bool                    `json:"-"`
	Placeholder          string                  `json:"-"`
	Description          string                  `json:"description"`
	CompleteDescription  string                  `json:"completeDescription"`
	NUsed                int                     `json:"nUsed"`
	SubCommand           map[string]CommandMatch `json:"subCommand"`
}

func NewCommandMatch(name, description string) CommandMatch {
	return CommandMatch{
		Name:                 name,
		MatchKey:             name,
		ShowOnlyWhenSelected: false,
		Description:          description,
		CompleteDescription:  "",
		NUsed:                0,
		SubCommand:           make(map[string]CommandMatch),
	}
}

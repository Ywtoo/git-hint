package core

// NodeKind classifies how a node relates to its parent in the command tree.
type NodeKind string

const (
	// KindSubcommand marks an exclusive child: selecting it consumes the
	// current level and may trigger a new help invocation during crawling.
	KindSubcommand NodeKind = "subcommand"
	// KindOption marks a combinable child (a flag): selecting it does not
	// consume the level, and it never triggers a new help invocation.
	KindOption NodeKind = "option"
	// KindArgument marks a leaf value (e.g. <message>, <keyid>). Never
	// triggers a new help invocation.
	KindArgument NodeKind = "argument"
)

type CommandMatch struct {
	Name                 string
	Kind                 NodeKind `json:"kind"`
	MatchKey             string   `json:"-"`
	ShowOnlyWhenSelected bool     `json:"-"`
	Placeholder          string   `json:"-"`
	ExpansionControl     bool     `json:"-"`
	ExitControl          bool     `json:"-"`
	ExpandedGroup        bool     `json:"-"`

	// Path is only populated in index.json entries, to record where the
	// command came from (PATH binary vs shell builtin). Empty/omitted
	// everywhere else.
	Path                string `json:"path,omitempty"`
	Description         string `json:"description"`
	CompleteDescription string `json:"completeDescription"`
	NUsed               int    `json:"nUsed"`

	// SubCommand holds exclusive children. Selecting one represents moving
	// to the next command level.
	SubCommand map[string]CommandMatch `json:"subCommand"`

	// Options holds combinable children (flags). Selecting one does not
	// remove the others from consideration.
	Options map[string]CommandMatch `json:"options,omitempty"`

	// Requires lists child keys (typically an Options or Argument key)
	// that become mandatory once this node is selected. For example,
	// "-m" requires "<message>".
	Requires []string `json:"requires,omitempty"`
}

func NewCommandMatch(name, description string) CommandMatch {
	return CommandMatch{
		Name:                 name,
		Kind:                 KindSubcommand,
		MatchKey:             name,
		ShowOnlyWhenSelected: false,
		Description:          description,
		CompleteDescription:  "",
		NUsed:                0,
		SubCommand:           make(map[string]CommandMatch),
		Options:              make(map[string]CommandMatch),
	}
}

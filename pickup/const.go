package pickup

const (
	// Pair is an identifier for queuing with a team of 2
	Pair = iota
	// Quad is an identifier for queuing with a team of 4
	Quad
	// Private is an identifier for queuing for private battles
	Private
)

const (
	// RoleSearchPair is the Discord role for "Searching for Pair"
	RoleSearchPair = "374995164222980097"
	// RoleSearchQuad is the Discord role for "Searching for Quad"
	RoleSearchQuad = "380166544560357377"
	// RoleSearchPrivate is the Discord role for "Searching for Private"
	RoleSearchPrivate = "380166602563518474"
	// RoleInProgress is the Discord role for "In Progress"
	RoleInProgress = "374995237342281732"
	// SearchChannelID is the ID for the channel where the search commands are used
	SearchChannelID = "374996197561073665"
)

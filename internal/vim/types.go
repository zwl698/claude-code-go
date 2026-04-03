package vim

// =============================================================================
// Vim Mode Types
// =============================================================================

// Operator represents a vim operator.
type Operator string

const (
	OperatorDelete Operator = "delete"
	OperatorChange Operator = "change"
	OperatorYank   Operator = "yank"
)

// FindType represents a find motion type.
type FindType string

const (
	FindTypeF      FindType = "f" // Find next
	FindTypeFUpper FindType = "F" // Find previous
	FindTypeT      FindType = "t" // To next
	FindTypeTUpper FindType = "T" // To previous
)

// TextObjScope represents a text object scope.
type TextObjScope string

const (
	TextObjScopeInner  TextObjScope = "inner"
	TextObjScopeAround TextObjScope = "around"
)

// =============================================================================
// Vim State
// =============================================================================

// VimState represents the complete vim state.
type VimState struct {
	Mode         string        `json:"mode"` // INSERT or NORMAL
	Command      *CommandState `json:"command,omitempty"`
	InsertedText string        `json:"insertedText,omitempty"`
}

// CommandState represents the command state machine for NORMAL mode.
type CommandState struct {
	Type   string       `json:"type"`
	Op     Operator     `json:"op,omitempty"`
	Count  int          `json:"count,omitempty"`
	Digits string       `json:"digits,omitempty"`
	Find   FindType     `json:"find,omitempty"`
	Scope  TextObjScope `json:"scope,omitempty"`
	Dir    string       `json:"dir,omitempty"` // for indent: > or <
}

// PersistentState represents state that survives across commands.
type PersistentState struct {
	LastChange         *RecordedChange `json:"lastChange,omitempty"`
	LastFind           *FindInfo       `json:"lastFind,omitempty"`
	Register           string          `json:"register"`
	RegisterIsLinewise bool            `json:"registerIsLinewise"`
}

// FindInfo represents last find command info.
type FindInfo struct {
	Type FindType `json:"type"`
	Char string   `json:"char"`
}

// RecordedChange represents a recorded change for dot-repeat.
type RecordedChange struct {
	Type      string       `json:"type"`
	Text      string       `json:"text,omitempty"`
	Op        Operator     `json:"op,omitempty"`
	Motion    string       `json:"motion,omitempty"`
	Count     int          `json:"count,omitempty"`
	ObjType   string       `json:"objType,omitempty"`
	Scope     TextObjScope `json:"scope,omitempty"`
	Find      FindType     `json:"find,omitempty"`
	Char      string       `json:"char,omitempty"`
	Dir       string       `json:"dir,omitempty"`
	Direction string       `json:"direction,omitempty"` // for openLine
}

// =============================================================================
// State Factories
// =============================================================================

// NewVimState creates an initial vim state (INSERT mode).
func NewVimState() *VimState {
	return &VimState{
		Mode:         "INSERT",
		InsertedText: "",
	}
}

// NewPersistentState creates an initial persistent state.
func NewPersistentState() *PersistentState {
	return &PersistentState{
		LastChange:         nil,
		LastFind:           nil,
		Register:           "",
		RegisterIsLinewise: false,
	}
}

// NewCommandState creates an idle command state.
func NewCommandState() *CommandState {
	return &CommandState{
		Type: "idle",
	}
}

// =============================================================================
// Operators and Keys
// =============================================================================

// IsOperatorKey checks if a key is an operator.
func IsOperatorKey(key string) bool {
	operators := map[string]Operator{
		"d": OperatorDelete,
		"c": OperatorChange,
		"y": OperatorYank,
	}
	_, ok := operators[key]
	return ok
}

// GetOperator returns the operator for a key.
func GetOperator(key string) Operator {
	operators := map[string]Operator{
		"d": OperatorDelete,
		"c": OperatorChange,
		"y": OperatorYank,
	}
	return operators[key]
}

// IsSimpleMotion checks if a key is a simple motion.
func IsSimpleMotion(key string) bool {
	motions := map[string]bool{
		"h": true, "l": true, "j": true, "k": true, // Basic movement
		"w": true, "b": true, "e": true, // Word motions
		"W": true, "B": true, "E": true,
		"0": true, "^": true, "$": true, // Line positions
		"G": true, "gg": true, // File navigation
	}
	return motions[key]
}

// IsFindKey checks if a key is a find key.
func IsFindKey(key string) bool {
	finds := map[string]bool{
		"f": true, "F": true, "t": true, "T": true,
	}
	return finds[key]
}

// IsTextObjScopeKey checks if a key is a text object scope.
func IsTextObjScopeKey(key string) bool {
	scopes := map[string]bool{
		"i": true, // inner
		"a": true, // around
	}
	return scopes[key]
}

// GetTextObjScope returns the text object scope for a key.
func GetTextObjScope(key string) TextObjScope {
	scopes := map[string]TextObjScope{
		"i": TextObjScopeInner,
		"a": TextObjScopeAround,
	}
	return scopes[key]
}

// IsTextObjType checks if a key is a text object type.
func IsTextObjType(key string) bool {
	types := map[string]bool{
		"w": true, "W": true, // Word/WORD
		"\"": true, "'": true, "`": true, // Quotes
		"(": true, ")": true, "b": true, // Parens
		"[": true, "]": true, // Brackets
		"{": true, "}": true, "B": true, // Braces
		"<": true, ">": true, // Angle brackets
	}
	return types[key]
}

// MaxVimCount is the maximum count for vim commands.
const MaxVimCount = 10000

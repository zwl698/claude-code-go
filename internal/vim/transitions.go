package vim

// =============================================================================
// Vim State Transitions
// =============================================================================

// Transition handles vim state transitions.
type Transition struct {
	state      *VimState
	persistent *PersistentState
}

// NewTransition creates a new transition handler.
func NewTransition(state *VimState, persistent *PersistentState) *Transition {
	return &Transition{
		state:      state,
		persistent: persistent,
	}
}

// HandleKey handles a key press in the current state.
func (t *Transition) HandleKey(key string) (action string, err error) {
	switch t.state.Mode {
	case "INSERT":
		return t.handleInsertKey(key)
	case "NORMAL":
		return t.handleNormalKey(key)
	default:
		return "", nil
	}
}

// handleInsertKey handles keys in INSERT mode.
func (t *Transition) handleInsertKey(key string) (string, error) {
	// Escape switches to NORMAL mode
	if key == "<Esc>" || key == "\x1b" {
		// Save inserted text for dot-repeat
		if t.state.InsertedText != "" {
			t.persistent.LastChange = &RecordedChange{
				Type: "insert",
				Text: t.state.InsertedText,
			}
		}
		t.state.Mode = "NORMAL"
		t.state.Command = NewCommandState()
		t.state.InsertedText = ""
		return "mode:normal", nil
	}

	// Record inserted text
	t.state.InsertedText += key
	return "insert:" + key, nil
}

// handleNormalKey handles keys in NORMAL mode.
func (t *Transition) handleNormalKey(key string) (string, error) {
	cmd := t.state.Command
	if cmd == nil {
		cmd = NewCommandState()
		t.state.Command = cmd
	}

	switch cmd.Type {
	case "idle":
		return t.handleIdleKey(key)
	case "count":
		return t.handleCountKey(key)
	case "operator":
		return t.handleOperatorKey(key)
	case "operatorCount":
		return t.handleOperatorCountKey(key)
	case "find":
		return t.handleFindKey(key)
	case "operatorFind":
		return t.handleOperatorFindKey(key)
	case "operatorTextObj":
		return t.handleOperatorTextObjKey(key)
	case "g":
		return t.handleGKey(key)
	case "replace":
		return t.handleReplaceKey(key)
	case "indent":
		return t.handleIndentKey(key)
	default:
		return "", nil
	}
}

// handleIdleKey handles keys in idle state.
func (t *Transition) handleIdleKey(key string) (string, error) {
	// Switch to INSERT mode
	if key == "i" {
		t.state.Mode = "INSERT"
		t.state.InsertedText = ""
		return "mode:insert", nil
	}

	// Switch to INSERT mode at beginning of line
	if key == "I" {
		t.state.Mode = "INSERT"
		t.state.InsertedText = ""
		return "mode:insert:start", nil
	}

	// Switch to INSERT mode at end of line
	if key == "A" {
		t.state.Mode = "INSERT"
		t.state.InsertedText = ""
		return "mode:insert:end", nil
	}

	// Switch to INSERT mode on new line below
	if key == "o" {
		t.state.Mode = "INSERT"
		t.state.InsertedText = ""
		t.persistent.LastChange = &RecordedChange{
			Type:      "openLine",
			Direction: "below",
		}
		return "insert:line_below", nil
	}

	// Switch to INSERT mode on new line above
	if key == "O" {
		t.state.Mode = "INSERT"
		t.state.InsertedText = ""
		t.persistent.LastChange = &RecordedChange{
			Type:      "openLine",
			Direction: "above",
		}
		return "insert:line_above", nil
	}

	// Operator keys
	if IsOperatorKey(key) {
		t.state.Command = &CommandState{
			Type:  "operator",
			Op:    GetOperator(key),
			Count: 1,
		}
		return "", nil
	}

	// Count
	if isDigit(key) && key != "0" {
		t.state.Command = &CommandState{
			Type:   "count",
			Digits: key,
		}
		return "", nil
	}

	// Find keys
	if IsFindKey(key) {
		t.state.Command = &CommandState{
			Type:  "find",
			Find:  FindType(key),
			Count: 1,
		}
		return "", nil
	}

	// g commands
	if key == "g" {
		t.state.Command = &CommandState{
			Type:  "g",
			Count: 1,
		}
		return "", nil
	}

	// Replace
	if key == "r" {
		t.state.Command = &CommandState{
			Type:  "replace",
			Count: 1,
		}
		return "", nil
	}

	// Indent
	if key == ">" || key == "<" {
		t.state.Command = &CommandState{
			Type:  "indent",
			Dir:   key,
			Count: 1,
		}
		return "", nil
	}

	// Simple motions
	if IsSimpleMotion(key) {
		return "motion:" + key, nil
	}

	// Delete character
	if key == "x" {
		t.persistent.LastChange = &RecordedChange{
			Type:  "x",
			Count: 1,
		}
		return "delete:char", nil
	}

	// Toggle case
	if key == "~" {
		t.persistent.LastChange = &RecordedChange{
			Type:  "toggleCase",
			Count: 1,
		}
		return "toggle:case", nil
	}

	// Join lines
	if key == "J" {
		t.persistent.LastChange = &RecordedChange{
			Type:  "join",
			Count: 1,
		}
		return "join:lines", nil
	}

	// Dot repeat
	if key == "." {
		if t.persistent.LastChange != nil {
			return "repeat", nil
		}
		return "", nil
	}

	// Paste
	if key == "p" {
		return "paste:after", nil
	}
	if key == "P" {
		return "paste:before", nil
	}

	// Undo
	if key == "u" {
		return "undo", nil
	}

	// Redo
	if key == "\x12" { // Ctrl-R
		return "redo", nil
	}

	return "", nil
}

// handleCountKey handles keys in count state.
func (t *Transition) handleCountKey(key string) (string, error) {
	cmd := t.state.Command

	if isDigit(key) {
		cmd.Digits += key
		// Check for overflow
		if len(cmd.Digits) > 4 {
			cmd.Type = "idle"
		}
		return "", nil
	}

	// Convert digits to count
	count := parseCount(cmd.Digits)
	if count > MaxVimCount {
		count = MaxVimCount
	}

	// Operator with count
	if IsOperatorKey(key) {
		cmd.Type = "operator"
		cmd.Op = GetOperator(key)
		cmd.Count = count
		cmd.Digits = ""
		return "", nil
	}

	// Find with count
	if IsFindKey(key) {
		cmd.Type = "find"
		cmd.Find = FindType(key)
		cmd.Count = count
		cmd.Digits = ""
		return "", nil
	}

	// g with count
	if key == "g" {
		cmd.Type = "g"
		cmd.Count = count
		cmd.Digits = ""
		return "", nil
	}

	// Motion with count
	if IsSimpleMotion(key) {
		cmd.Type = "idle"
		return "motion:" + key, nil
	}

	// Unknown key, reset to idle
	cmd.Type = "idle"
	return "", nil
}

// handleOperatorKey handles keys in operator state.
func (t *Transition) handleOperatorKey(key string) (string, error) {
	cmd := t.state.Command

	// Double operator (e.g., dd, cc, yy)
	if IsOperatorKey(key) && GetOperator(key) == cmd.Op {
		// Line-wise operation
		action := string(cmd.Op) + ":line"
		t.persistent.LastChange = &RecordedChange{
			Type:   "operator",
			Op:     cmd.Op,
			Motion: "line",
			Count:  cmd.Count,
		}
		cmd.Type = "idle"
		return action, nil
	}

	// Count for operator
	if isDigit(key) {
		cmd.Type = "operatorCount"
		cmd.Digits = key
		return "", nil
	}

	// Find motion
	if IsFindKey(key) {
		cmd.Type = "operatorFind"
		cmd.Find = FindType(key)
		return "", nil
	}

	// Text object scope
	if IsTextObjScopeKey(key) {
		cmd.Type = "operatorTextObj"
		cmd.Scope = GetTextObjScope(key)
		return "", nil
	}

	// Simple motion
	if IsSimpleMotion(key) {
		action := string(cmd.Op) + ":" + key
		t.persistent.LastChange = &RecordedChange{
			Type:   "operator",
			Op:     cmd.Op,
			Motion: key,
			Count:  cmd.Count,
		}
		cmd.Type = "idle"
		return action, nil
	}

	// g motion
	if key == "g" {
		cmd.Type = "operatorG"
		return "", nil
	}

	// Unknown key, reset to idle
	cmd.Type = "idle"
	return "", nil
}

// handleOperatorCountKey handles keys in operatorCount state.
func (t *Transition) handleOperatorCountKey(key string) (string, error) {
	cmd := t.state.Command

	if isDigit(key) {
		cmd.Digits += key
		if len(cmd.Digits) > 4 {
			cmd.Type = "idle"
		}
		return "", nil
	}

	// Apply count to operator count
	count := parseCount(cmd.Digits)
	totalCount := cmd.Count * count
	if totalCount > MaxVimCount {
		totalCount = MaxVimCount
	}

	// Motion with combined count
	if IsSimpleMotion(key) {
		action := string(cmd.Op) + ":" + key
		t.persistent.LastChange = &RecordedChange{
			Type:   "operator",
			Op:     cmd.Op,
			Motion: key,
			Count:  totalCount,
		}
		cmd.Type = "idle"
		return action, nil
	}

	cmd.Type = "idle"
	return "", nil
}

// handleFindKey handles keys in find state.
func (t *Transition) handleFindKey(key string) (string, error) {
	cmd := t.state.Command

	// Find character
	t.persistent.LastFind = &FindInfo{
		Type: cmd.Find,
		Char: key,
	}
	action := "find:" + string(cmd.Find) + ":" + key
	cmd.Type = "idle"
	return action, nil
}

// handleOperatorFindKey handles keys in operatorFind state.
func (t *Transition) handleOperatorFindKey(key string) (string, error) {
	cmd := t.state.Command

	t.persistent.LastFind = &FindInfo{
		Type: cmd.Find,
		Char: key,
	}
	action := string(cmd.Op) + ":find:" + string(cmd.Find) + ":" + key
	t.persistent.LastChange = &RecordedChange{
		Type:  "operatorFind",
		Op:    cmd.Op,
		Find:  cmd.Find,
		Char:  key,
		Count: cmd.Count,
	}
	cmd.Type = "idle"
	return action, nil
}

// handleOperatorTextObjKey handles keys in operatorTextObj state.
func (t *Transition) handleOperatorTextObjKey(key string) (string, error) {
	cmd := t.state.Command

	if IsTextObjType(key) {
		action := string(cmd.Op) + ":textobj:" + key
		t.persistent.LastChange = &RecordedChange{
			Type:    "operatorTextObj",
			Op:      cmd.Op,
			ObjType: key,
			Scope:   cmd.Scope,
			Count:   cmd.Count,
		}
		cmd.Type = "idle"
		return action, nil
	}

	cmd.Type = "idle"
	return "", nil
}

// handleGKey handles keys in g state.
func (t *Transition) handleGKey(key string) (string, error) {
	cmd := t.state.Command

	if key == "g" {
		// gg - go to first line
		action := "goto:first_line"
		cmd.Type = "idle"
		return action, nil
	}

	cmd.Type = "idle"
	return "", nil
}

// handleReplaceKey handles keys in replace state.
func (t *Transition) handleReplaceKey(key string) (string, error) {
	cmd := t.state.Command

	action := "replace:" + key
	t.persistent.LastChange = &RecordedChange{
		Type:  "replace",
		Char:  key,
		Count: cmd.Count,
	}
	cmd.Type = "idle"
	return action, nil
}

// handleIndentKey handles keys in indent state.
func (t *Transition) handleIndentKey(key string) (string, error) {
	cmd := t.state.Command

	// Double indent (e.g., >>, <<)
	if key == cmd.Dir {
		action := "indent:" + cmd.Dir
		t.persistent.LastChange = &RecordedChange{
			Type:  "indent",
			Dir:   cmd.Dir,
			Count: cmd.Count,
		}
		cmd.Type = "idle"
		return action, nil
	}

	cmd.Type = "idle"
	return "", nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func isDigit(s string) bool {
	return len(s) == 1 && s[0] >= '0' && s[0] <= '9'
}

func parseCount(s string) int {
	count := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			count = count*10 + int(ch-'0')
		}
	}
	return count
}

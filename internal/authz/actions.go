package authz

// Action represents the type of operation being performed on a resource
type Action string

// Authorization actions
const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionDelete Action = "delete"
	ActionGrant  Action = "grant"
)

// HTTPMethodToAction maps HTTP methods to authorization actions
var HTTPMethodToAction = map[string]Action{
	"GET":    ActionRead,
	"POST":   ActionWrite,
	"PUT":    ActionWrite,
	"PATCH":  ActionWrite,
	"DELETE": ActionDelete,
}

// GetActionFromMethod returns the authorization action for a given HTTP method
func GetActionFromMethod(method string) Action {
	if action, exists := HTTPMethodToAction[method]; exists {
		return action
	}
	return ActionRead // Default to read for unknown methods
}

// IsWriteAction returns true if the action involves modifying data
func (a Action) IsWriteAction() bool {
	return a == ActionWrite || a == ActionDelete || a == ActionGrant
}

// IsReadAction returns true if the action only involves reading data
func (a Action) IsReadAction() bool {
	return a == ActionRead
}

// String returns the string representation of the action
func (a Action) String() string {
	return string(a)
}

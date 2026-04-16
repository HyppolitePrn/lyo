package auth

// Role represents a user's permission level.
type Role string

const (
	RoleAnonymous   Role = "anonymous"
	RoleUser        Role = "user"
	RoleBroadcaster Role = "broadcaster"
	RoleAdmin       Role = "admin"
)

// weight maps roles to numeric levels for hierarchy comparisons.
var weight = map[Role]int{
	RoleAnonymous:   0,
	RoleUser:        1,
	RoleBroadcaster: 2,
	RoleAdmin:       3,
}

// AtLeast reports whether r has at least the same privilege level as min.
func (r Role) AtLeast(min Role) bool {
	return weight[r] >= weight[min]
}

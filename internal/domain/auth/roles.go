package auth

const (
	RoleAdmin   = "admin"
	RoleGeneral = "general"
	RoleDemo    = "demo"
)

func IsRole(role string) bool { return role == RoleAdmin || role == RoleGeneral || role == RoleDemo }

// Accounts created before roles were introduced retain ordinary library access.
// Unknown role values fail closed; no account is implicitly made an admin.
func (u *User) EffectiveRole() string {
	if u == nil {
		return ""
	}
	if u.Role == "" {
		return RoleGeneral
	}
	return u.Role
}
func (u *User) IsAdmin() bool { return u != nil && u.EffectiveRole() == RoleAdmin }
func (u *User) CanWrite() bool {
	return u != nil && (u.EffectiveRole() == RoleAdmin || u.EffectiveRole() == RoleGeneral)
}

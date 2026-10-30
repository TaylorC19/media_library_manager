package auth

import "testing"

func TestRolesFailClosedAndLegacyAccountsAreGeneral(t *testing.T) {
	for _, test := range []struct {
		role         string
		admin, write bool
	}{
		{"", false, true}, {RoleGeneral, false, true}, {RoleAdmin, true, true}, {RoleDemo, false, false}, {"unrecognized", false, false},
	} {
		user := &User{Role: test.role}
		if user.IsAdmin() != test.admin || user.CanWrite() != test.write {
			t.Errorf("wrong permissions for role %q", test.role)
		}
	}
	var user *User
	if user.IsAdmin() || user.CanWrite() {
		t.Error("nil user must have no permissions")
	}
}

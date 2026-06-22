package application

func rolePrecedence(role string) int {
	switch role {
	case "owner":
		return 3
	case "admin":
		return 2
	case "member":
		return 1
	}
	return 0
}

func RoleAtLeast(role, min string) bool {
	return rolePrecedence(role) >= rolePrecedence(min)
}

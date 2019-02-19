package bot

// role IDs
var adminRoles = []string{""}

// IsUserIDAdmin checkes the given Discord ID to check if the user has an admin role
func IsUserIDAdmin(userID string) (bool, error) {
	m, e := data.Member(userID)
	if e != nil {
		return false, e
	}

	for _, ur := range m.Roles {
		for _, role := range adminRoles {
			if ur == role {
				return true, nil
			}
		}
	}

	return false, nil
}

package bot

// role IDs
var adminRoles = []string{"439874952112504833", "316736287065243660"}

var verifiedRole = "439875158610542592"

// IsUserIDAdmin checks the given Discord ID to check if the user has an admin role as defined in the bot package
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

// IsUserIDVerified checks if the Discord user with the id given has the verified role as defined in the bot package
func IsUserIDVerified(userID string) (bool, error) {

	m, e := data.Member(userID)
	if e != nil {
		return false, e
	}

	for _, ur := range m.Roles {
		if ur == verifiedRole {
			return true, nil
		}
	}

	return false, nil
}

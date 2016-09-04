package bot

var AdminChannel = "damnbot-admin"

func IsUserIDAdmin(userID string) (bool, error) {
	switch userID {
	case "U03RTQMPB": // demitriousk
		return true, nil
	case "U0381RKM9": // admwrightmeow
		return true, nil
	case "U0920GS5S": // bobthelob
		return true, nil
	}
	for _, channel := range data.UserGroups(userID) {
		if channel.Name == AdminChannel {
			return true, nil
		}
	}
	return false, nil
}

func IsUsernameAdmin(username string) (bool, error) {
	if user, err := data.UserByName(username); err != nil {
		return false, err
	} else {
		for _, channel := range data.UserGroups(user.ID) {
			if channel.Name == AdminChannel {
				return true, nil
			}
		}
	}
	return false, nil
}

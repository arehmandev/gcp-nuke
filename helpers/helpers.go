package helpers

// SliceContains - check if slice contains value
func SliceContains(inputList []string, input string) bool {
	for _, value := range inputList {
		if value == input {
			return true
		}
	}
	return false
}

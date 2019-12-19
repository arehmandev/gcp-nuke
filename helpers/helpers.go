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

// MapKeys -
func MapKeys(input map[string]interface{}) []string {
	keys := []string{}
	for k := range input {
		keys = append(keys, k)
	}
	return keys
}

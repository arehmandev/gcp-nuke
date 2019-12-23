package helpers

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"golang.org/x/sync/syncmap"
)

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

// SortedSyncMapKeys -
func SortedSyncMapKeys(parMap *syncmap.Map) (output []string) {

	parMap.Range(func(key, value interface{}) bool {
		strkey := fmt.Sprintf("%v", key)
		output = append(output, strkey)
		return true
	})

	sort.Sort(sort.StringSlice(output))
	return output
}

// SetupCloseHandler - allows manual termination
func SetupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal - premature termination")
		os.Exit(1)
	}()
}

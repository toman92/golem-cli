package ui

import (
	"fmt"
	"time"
)

// ShowSpinner displays a simple terminal spinner while the provided action runs.
func ShowSpinner(msg string, action func()) {
	done := make(chan bool)
	go func() {
		chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-done:
				fmt.Print("\r\033[K") // clear line
				return
			default:
				fmt.Printf("\r\033[36m%s\033[0m %s", chars[i], msg)
				i = (i + 1) % len(chars)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	action()
	done <- true
}

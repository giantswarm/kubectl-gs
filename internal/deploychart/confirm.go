package deploychart

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// AskConfirmation prints a prompt and reads y/N from the reader.
// Returns true if the user confirms with "y" or "yes".
func AskConfirmation(prompt string, in io.Reader, out io.Writer) (bool, error) {
	_, _ = fmt.Fprintf(out, "%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return response == "y" || response == "yes", nil
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

package confirm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// AskYesNo prompts the user with a yes/no question via stdin.
// It accepts Turkish (E/hayır) and English (Y/no) variations.
// Pressing enter directly (empty input) defaults to true (Yes).
func AskYesNo(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (E/h): ", prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		trimmed := strings.ToLower(strings.TrimSpace(input))
		// Default to Yes on empty input
		if trimmed == "" || trimmed == "e" || trimmed == "evet" || trimmed == "y" || trimmed == "yes" {
			return true, nil
		}
		if trimmed == "h" || trimmed == "hayır" || trimmed == "n" || trimmed == "no" {
			return false, nil
		}
		fmt.Println("Geçersiz giriş. Lütfen 'E' (Evet) veya 'H' (Hayır) giriniz.")
	}
}

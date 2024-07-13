package text

const (
	RED    string = "\033[31m"
	GREEN  string = "\033[32m"
	YELLOW string = "\033[33m"
	RESET  string = "\033[0m"
)

// Paint will wrap the given colour to the given text.
func Foreground(colour string, text string) string {
	return colour + text + RESET
}

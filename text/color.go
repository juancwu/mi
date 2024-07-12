package text

const (
	RED   string = "\033[31m"
	RESET string = "\033[0m"
)

// Paint will wrap the given colour to the given text.
func Foreground(colour string, text string) string {
	return colour + text + RESET
}

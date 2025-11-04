package utils

// Color represents a lipgloss color ID
type Color string

const (
	// Standard colors
	ColorInfo    Color = "6"    // Cyan (ANSI 36)
	ColorDebug   Color = "248"  // Light gray (ANSI 90)
	ColorSuccess Color = "46"   // Bright green (ANSI 32)
	ColorWarning Color = "220"  // Yellow/Orange (ANSI 33)
	ColorError   Color = "1"    // Red (ANSI 31)
	
	// Gray scale
	ColorDarkGray  Color = "240" // Dark gray
	ColorLightGray Color = "248" // Light gray
)

// String returns the color ID as a string
func (c Color) String() string {
	return string(c)
}


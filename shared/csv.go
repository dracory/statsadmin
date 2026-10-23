package shared

// CSVSafe neutralizes spreadsheet formula injection (CSV injection). Values
// whose first character could be interpreted as a formula by Excel,
// LibreOffice, etc. (=, +, -, @) or that start with a tab or carriage return
// are prefixed with a single quote so they are treated as literal text.
// CWE-1236: Improper Neutralization of Formula Elements in a CSV File.
func CSVSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

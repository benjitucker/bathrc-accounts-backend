package email

// LogoPNG returns the club logo used in the emails, for use elsewhere (such as web pages).
func LogoPNG() ([]byte, error) {
	return assetsFS.ReadFile("assets/logo.png")
}

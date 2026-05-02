package repo

// nullStr converts an empty string to nil, suitable for nullable DB columns.
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

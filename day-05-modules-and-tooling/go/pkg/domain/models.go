package domain

// User represents a core domain entity.
// Exported fields (Capitalized) are accessible to any package that imports this.
type User struct {
	ID   string
	Name string
}

// Note: If we had a field 'secretKey string', it would be unexported (lowercase)
// and inaccessible to the 'main' package or the 'service' package.

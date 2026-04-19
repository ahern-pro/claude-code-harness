package memory

type MemoryHeader struct {
	Path        string
	Title       string
	Description string
	ModifiedAt  int64
	MemoryType  string
	BodyPreview string
}

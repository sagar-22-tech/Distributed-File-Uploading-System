package models

type UploadSession struct {
	FileID      string
	FileName    string
	TotalSize   int64
	ChunkSize   int64
	TotalChunks int
}

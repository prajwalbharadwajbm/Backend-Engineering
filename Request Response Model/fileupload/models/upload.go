package models

type UploadRequest struct {
	Filename string `json:"filename"`
	FileSize int64  `json:"filesize"`
}

type UploadInfo struct {
	ID             string
	Filename       string
	TotalSize      int64
	UploadedSize   int64
	ChunksReceived map[int]bool
}

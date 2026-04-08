package repository

import "context"

// StorageRepository defines the contract for file storage operations.
type StorageRepository interface {
	DownloadFile(ctx context.Context, directoryPath, fileName string) ([]byte, error)
}

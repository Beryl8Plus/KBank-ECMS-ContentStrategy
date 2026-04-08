package repository

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"

	domainrepo "kbank-ecms/internal/domain/repository"
)

// AzureStorageRepository implements domain repository.StorageRepository using Azure Files.
type AzureStorageRepository struct {
	cred        *azidentity.DefaultAzureCredential
	fileClient  *service.Client
	vaultClient *azsecrets.Client
}

// Compile-time interface check.
var _ domainrepo.StorageRepository = (*AzureStorageRepository)(nil)

// NewAzureStorageRepository initializes Azure credentials and clients.
func NewAzureStorageRepository(ctx context.Context) (*AzureStorageRepository, error) {
	accountName := os.Getenv("AZACCOUNTNAME")
	fileServiceURL := fmt.Sprintf("https://%s.file.core.windows.net/", accountName)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("could not get credentials: %w", err)
	}

	// Create File Client with FileRequestIntent for OAuth authentication
	options := &service.ClientOptions{
		FileRequestIntent: to.Ptr(service.ShareTokenIntentBackup),
	}
	fileClient, err := service.NewClient(fileServiceURL, cred, options)
	if err != nil {
		return nil, fmt.Errorf("could not create file client: %w", err)
	}

	return &AzureStorageRepository{
		cred:       cred,
		fileClient: fileClient,
	}, nil
}

// DownloadFile downloads a file from Azure Files share and returns its content.
// directoryPath: path to the directory within the share (e.g., "folder1/folder2" or "" for root)
// fileName: name of the file to download (e.g., "config.yaml")
func (r *AzureStorageRepository) DownloadFile(ctx context.Context, directoryPath, fileName string) ([]byte, error) {
	if r.fileClient == nil {
		return nil, fmt.Errorf("file client not initialized")
	}

	SHARENAME := os.Getenv("SHARENAME")

	// Get share client
	shareClient := r.fileClient.NewShareClient(SHARENAME)

	// Get directory client (use root if directoryPath is empty)
	dirClient := shareClient.NewRootDirectoryClient()
	if directoryPath != "" {
		dirClient = shareClient.NewDirectoryClient(directoryPath)
	}

	// Get file client
	fileClient := dirClient.NewFileClient(fileName)

	// Download file
	resp, err := fileClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Read all content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, nil
}

// GetVaultClient returns the Azure Key Vault client (may be nil if not initialized).
func (r *AzureStorageRepository) GetVaultClient() *azsecrets.Client {
	return r.vaultClient
}

// GetCredential returns the Azure credential.
func (r *AzureStorageRepository) GetCredential() *azidentity.DefaultAzureCredential {
	return r.cred
}

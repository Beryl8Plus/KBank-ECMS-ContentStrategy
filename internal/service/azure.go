package service

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
)

var (
	AzureCred *azidentity.DefaultAzureCredential
	// ctx         = context.Background() // Removed global context
	Fileclient  *service.Client
	VaultClient *azsecrets.Client
)

func InitAzure(ctx context.Context) error {
	var err error

	// Auth with Workload Identity (Automatic detection)
	// Passing 'nil' tells the SDK to look for the Workload Identity environment variables

	accountName := os.Getenv("AZACCOUNTNAME")
	fileServiceURL := fmt.Sprintf("https://%s.file.core.windows.net/", accountName)

	AzureCred, err = azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("could not get credentials: %w", err)
	}

	// Create File Client with FileRequestIntent for OAuth authentication
	options := &service.ClientOptions{
		FileRequestIntent: to.Ptr(service.ShareTokenIntentBackup),
	}
	Fileclient, err = service.NewClient(fileServiceURL, AzureCred, options)
	if err != nil {
		return fmt.Errorf("could not create file client: %w", err)
	}

	// Create Key Vault Client
	// keyVaultName := os.Getenv("KEYVAULTACCOUNTNAME") // kvkbankcontgwapptest02
	// keyVaultfileServiceURL := fmt.Sprintf("https://%s.vault.azure.net/", keyVaultName)
	// if keyVaultfileServiceURL != "" {
	// 	VaultClient, err = azsecrets.NewClient(keyVaultfileServiceURL, AzureCred, nil)
	// 	if err != nil {
	// 		return fmt.Errorf("could not create key vault client: %w", err)
	// 	}
	// }

	return nil
}

// FileDownload downloads a file from Azure Files share and returns its content as a string
// shareName: name of the file share (e.g., "myshare")
// directoryPath: path to the directory within the share (e.g., "folder1/folder2" or "" for root)
// fileName: name of the file to download (e.g., "config.yaml")
func FileDownload(ctx context.Context, directoryPath, fileName string) ([]byte, error) {
	if Fileclient == nil {
		return nil, fmt.Errorf("file client not initialized")
	}

	// share-kbank-contgw-share-test-001
	SHARENAME := os.Getenv("SHARENAME")

	// Get share client
	shareClient := Fileclient.NewShareClient(SHARENAME)

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

// GetSecret retrieves a secret from Azure Key Vault by name
// func GetSecret(ctx context.Context, secretName string) (string, error) {
// 	if VaultClient == nil {
// 		return "", fmt.Errorf("vault client not initialized")
// 	}

// 	resp, err := VaultClient.GetSecret(ctx, secretName, "", nil)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to get secret %s: %w", secretName, err)
// 	}

// 	return *resp.Value, nil
// }

package private_channel

import (
	"os"
	"path/filepath"
)

var (
	// Directory paths
	UserDataDir = "./user_data"
	DownloadDir = "download"
	UploadDir   = "upload"
	PublicDir   = "public"

	// UDP server address
	UDPServerAddress = "165.227.25.123:8099"
)

func GetUserDownloadDir(userId string) string {
	return filepath.Join(UserDataDir, userId, DownloadDir)
}

func GetUserUploadDir(userId string) string {
	return filepath.Join(UserDataDir, userId, UploadDir)
}

func GetPublicDir() string {
	return PublicDir
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

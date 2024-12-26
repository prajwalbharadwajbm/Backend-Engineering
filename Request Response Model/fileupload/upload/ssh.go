package upload

import (
	"fmt"
	"os"

	"encoding/json"
	"io"
	"net/http"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	KeyFile    string `json:"keyFile,omitempty"`
	RemoteDir  string `json:"remoteDir"`
	AuthMethod string `json:"authMethod"`
}

func TestSSHConnection(config SSHConfig) error {
	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if config.AuthMethod == "password" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(config.Password))
	} else {
		// Handle SSH key authentication
		key, err := os.ReadFile(config.KeyFile)
		if err != nil {
			return fmt.Errorf("unable to read private key: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("unable to parse private key: %v", err)
		}

		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", config.Host, config.Port), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()

	return nil
}

func UploadFileViaSSH(config SSHConfig, localFilePath string, originalFilename string) error {
	// Setup SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
	}

	// Set up authentication
	if config.AuthMethod == "password" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(config.Password))
	} else {
		key, err := os.ReadFile(config.KeyFile)
		if err != nil {
			return fmt.Errorf("unable to read private key: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("unable to parse private key: %v", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	// Connect to SSH server
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", config.Host, config.Port), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %v", err)
	}
	defer client.Close()

	// Create new SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	// Open local file
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %v", err)
	}
	defer localFile.Close()

	// Get file info for size
	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Create remote directory if it doesn't exist
	err = sftpClient.MkdirAll(config.RemoteDir)
	if err != nil {
		return fmt.Errorf("failed to create remote directory: %v", err)
	}

	// Create remote file
	remoteFilePath := filepath.Join(config.RemoteDir, originalFilename)
	remoteFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %v", err)
	}
	defer remoteFile.Close()

	// Create a buffer for copying
	buf := make([]byte, 32*1024) // 32KB buffer
	totalWritten := int64(0)

	// Copy file with progress tracking
	for {
		n, err := localFile.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading local file: %v", err)
		}
		if n == 0 {
			break
		}

		_, err = remoteFile.Write(buf[:n])
		if err != nil {
			return fmt.Errorf("error writing to remote file: %v", err)
		}

		totalWritten += int64(n)
		progress := float64(totalWritten) / float64(fileInfo.Size()) * 100
		fmt.Printf("\rUploading... %.2f%%", progress)
	}

	fmt.Println("\nUpload complete!")

	// Verify file size
	remoteFileInfo, err := sftpClient.Stat(remoteFilePath)
	if err != nil {
		return fmt.Errorf("failed to get remote file info: %v", err)
	}

	if remoteFileInfo.Size() != fileInfo.Size() {
		return fmt.Errorf("file size mismatch: local %d != remote %d", fileInfo.Size(), remoteFileInfo.Size())
	}

	return nil
}

// Add a handler function for the HTTP endpoint
func HandleSSHUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error getting file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save original filename
	originalFilename := header.Filename

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "ssh-upload-*")
	if err != nil {
		http.Error(w, "Error creating temp file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy uploaded file to temp file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Error saving temp file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get SSH config from request
	var config SSHConfig
	configStr := r.FormValue("sshConfig")
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		http.Error(w, "Error parsing SSH config: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Upload file via SSH
	err = UploadFileViaSSH(config, tempFile.Name(), originalFilename)
	if err != nil {
		http.Error(w, "Error uploading via SSH: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("File %s uploaded successfully via SSH", originalFilename)))
}

func HandleSSHTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config SSHConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := TestSSHConnection(config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("SSH connection successful"))
}

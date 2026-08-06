package partner

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// LoadSecretRef resolves a SecretRef to credential material.
// Order: env named by SecretRef, then PARTNER_SFTP_SECRET_<SecretRef>, then file path if SecretRef is absolute.
func LoadSecretRef(secretRef string) (string, error) {
	secretRef = strings.TrimSpace(secretRef)
	if secretRef == "" {
		return "", fmt.Errorf("empty_secret_ref")
	}
	if v := strings.TrimSpace(os.Getenv(secretRef)); v != "" {
		return v, nil
	}
	envKey := "PARTNER_SFTP_SECRET_" + strings.ToUpper(strings.ReplaceAll(secretRef, "-", "_"))
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(os.Getenv("PARTNER_SFTP_SECRET_" + secretRef)); v != "" {
		return v, nil
	}
	as2Key := "PARTNER_AS2_SECRET_" + strings.ToUpper(strings.ReplaceAll(secretRef, "-", "_"))
	if v := strings.TrimSpace(os.Getenv(as2Key)); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(os.Getenv("PARTNER_AS2_SECRET_" + secretRef)); v != "" {
		return v, nil
	}
	if strings.HasPrefix(secretRef, "/") || strings.HasPrefix(secretRef, "./") {
		b, err := os.ReadFile(secretRef)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	return "", fmt.Errorf("secret_not_found")
}

// UploadSFTP pushes a local file to the configured remote directory.
func UploadSFTP(ctx context.Context, cfg SftpConfig, secret, localPath, remoteName string) error {
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.Username) == "" {
		return fmt.Errorf("sftp_config_incomplete")
	}
	port := cfg.Port
	if port <= 0 {
		port = 22
	}
	auth, err := sftpAuthMethods(secret)
	if err != nil {
		return err
	}
	sshCfg := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // enterprise partners often use password/key; pin later
		Timeout:         20 * time.Second,
	}
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", port))
	conn, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return fmt.Errorf("ssh_dial: %w", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("sftp_client: %w", err)
	}
	defer client.Close()

	remoteDir := strings.TrimSpace(cfg.RemoteDir)
	if remoteDir == "" {
		remoteDir = "/"
	}
	remotePath := path.Join(remoteDir, remoteName)

	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("sftp_create: %w", err)
	}
	defer dst.Close()

	done := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(dst, src)
		done <- copyErr
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case copyErr := <-done:
		return copyErr
	}
}

// SftpFileInfo is a remote file listing entry.
type SftpFileInfo struct {
	Name string
	Size int64
	Dir  bool
}

func dialSFTP(cfg SftpConfig, secret string) (*ssh.Client, *sftp.Client, error) {
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.Username) == "" {
		return nil, nil, fmt.Errorf("sftp_config_incomplete")
	}
	port := cfg.Port
	if port <= 0 {
		port = 22
	}
	auth, err := sftpAuthMethods(secret)
	if err != nil {
		return nil, nil, err
	}
	sshCfg := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         20 * time.Second,
	}
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", port))
	conn, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh_dial: %w", err)
	}
	client, err := sftp.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("sftp_client: %w", err)
	}
	return conn, client, nil
}

func joinRemote(base, name string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "/"
	}
	return path.Join(base, name)
}

// ListSFTP lists non-directory files under remoteDir (absolute or under RemoteDir).
func ListSFTP(ctx context.Context, cfg SftpConfig, secret, remoteDir string) ([]SftpFileInfo, error) {
	conn, client, err := dialSFTP(cfg, secret)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer client.Close()

	dir := strings.TrimSpace(remoteDir)
	if dir == "" {
		dir = joinRemote(cfg.RemoteDir, cfg.InboundDir)
	}
	entries, err := client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("sftp_readdir: %w", err)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	out := make([]SftpFileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out = append(out, SftpFileInfo{Name: e.Name(), Size: e.Size(), Dir: false})
	}
	return out, nil
}

// DownloadSFTP reads a remote file into memory.
func DownloadSFTP(ctx context.Context, cfg SftpConfig, secret, remotePath string) ([]byte, error) {
	conn, client, err := dialSFTP(cfg, secret)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer client.Close()

	f, err := client.Open(remotePath)
	if err != nil {
		return nil, fmt.Errorf("sftp_open: %w", err)
	}
	defer f.Close()
	done := make(chan struct{})
	var data []byte
	var copyErr error
	go func() {
		data, copyErr = io.ReadAll(io.LimitReader(f, 8<<20))
		close(done)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return data, copyErr
	}
}

// RenameSFTP moves a remote file (archive after ingest).
func RenameSFTP(ctx context.Context, cfg SftpConfig, secret, fromPath, toPath string) error {
	conn, client, err := dialSFTP(cfg, secret)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	_ = client.MkdirAll(path.Dir(toPath))
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := client.Rename(fromPath, toPath); err != nil {
		return fmt.Errorf("sftp_rename: %w", err)
	}
	return nil
}

// UploadSFTPToDir uploads to an explicit remote directory (outbound EDI).
func UploadSFTPToDir(ctx context.Context, cfg SftpConfig, secret, remoteDir, localPath, remoteName string) error {
	cfg2 := cfg
	cfg2.RemoteDir = remoteDir
	return UploadSFTP(ctx, cfg2, secret, localPath, remoteName)
}

func sftpAuthMethods(secret string) ([]ssh.AuthMethod, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("empty_secret")
	}
	// Try private key first when PEM-like.
	if strings.Contains(secret, "BEGIN") && strings.Contains(secret, "PRIVATE KEY") {
		signer, err := ssh.ParsePrivateKey([]byte(secret))
		if err != nil {
			return nil, fmt.Errorf("parse_private_key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	return []ssh.AuthMethod{ssh.Password(secret)}, nil
}

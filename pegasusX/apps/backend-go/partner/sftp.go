package partner

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ValidatePartnerSecretRef verifies that a partner SecretRef is a safe logical key name
// rather than an arbitrary filesystem path, OS env var name, or URL scheme.
func ValidatePartnerSecretRef(secretRef string) error {
	ref := strings.TrimSpace(secretRef)
	if ref == "" {
		return fmt.Errorf("empty_secret_ref")
	}
	if len(ref) > 128 || ref[0] == '.' || ref[0] == '-' {
		return fmt.Errorf("invalid_secret_ref")
	}
	if strings.Contains(ref, "..") {
		return fmt.Errorf("invalid_secret_ref")
	}
	for _, r := range ref {
		isOK := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.'
		if !isOK {
			return fmt.Errorf("invalid_secret_ref")
		}
	}
	return nil
}

func sanitizePartnerEnvSegment(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(raw)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return strings.Trim(b.String(), "_")
}

// LoadTenantSecretRef resolves a validated SecretRef using tenant-scoped PARTNER_{SFTP,AS2}_SECRET_<TENANT>_<KEY>
// environment variables first, failing closed in production when only cross-tenant keys exist.
func LoadTenantSecretRef(tenantType, tenantID, secretRef string) (string, error) {
	if err := ValidatePartnerSecretRef(secretRef); err != nil {
		return "", err
	}
	ref := strings.TrimSpace(secretRef)
	upperRef := strings.ReplaceAll(strings.ToUpper(ref), "-", "_")
	tenantPrefix := sanitizePartnerEnvSegment(tenantType) + "_" + sanitizePartnerEnvSegment(tenantID)
	tenantPrefix = strings.Trim(tenantPrefix, "_")
	if tenantPrefix != "" {
		for _, pfx := range []string{"PARTNER_SFTP_SECRET_", "PARTNER_AS2_SECRET_"} {
			for _, candidate := range []string{pfx + tenantPrefix + "_" + upperRef, pfx + tenantPrefix + "_" + ref} {
				if v := strings.TrimSpace(os.Getenv(candidate)); v != "" {
					return v, nil
				}
			}
		}
	}
	if auth.IsProduction() || envTruthy("PARTNER_STRICT_TENANT_SECRETS") {
		return "", fmt.Errorf("secret_not_found")
	}
	return LoadSecretRef(ref)
}

func resolveWorkerSecret(customLoader func(string) (string, error), tenantType, tenantID, secretRef string) (string, error) {
	if customLoader != nil {
		if err := ValidatePartnerSecretRef(secretRef); err != nil {
			return "", err
		}
		return customLoader(secretRef)
	}
	return LoadTenantSecretRef(tenantType, tenantID, secretRef)
}

// LoadSecretRef resolves a validated SecretRef to credential material strictly under
// PARTNER_SFTP_SECRET_<KEY> or PARTNER_AS2_SECRET_<KEY> (never arbitrary OS env or filesystem paths).
func LoadSecretRef(secretRef string) (string, error) {
	if err := ValidatePartnerSecretRef(secretRef); err != nil {
		return "", err
	}
	secretRef = strings.TrimSpace(secretRef)
	upperKey := strings.ReplaceAll(strings.ToUpper(secretRef), "-", "_")
	for _, k := range []string{
		"PARTNER_SFTP_SECRET_" + upperKey,
		"PARTNER_SFTP_SECRET_" + secretRef,
		"PARTNER_AS2_SECRET_" + upperKey,
		"PARTNER_AS2_SECRET_" + secretRef,
	} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("secret_not_found")
}

// PartnerSFTPStrictHostKey returns true when SSH host-key pinning is mandatory (default true;
// can only be disabled in non-production via PARTNER_SFTP_ALLOW_INSECURE_HOSTKEY=true).
func PartnerSFTPStrictHostKey() bool {
	if envTruthy("PARTNER_SFTP_STRICT_HOSTKEY") {
		return true
	}
	if !auth.IsProduction() && envTruthy("PARTNER_SFTP_ALLOW_INSECURE_HOSTKEY") {
		return false
	}
	return true
}

func sftpHostKeyCallback(cfg SftpConfig) (ssh.HostKeyCallback, error) {
	pin := strings.TrimSpace(cfg.HostKeySHA256)
	pin = strings.TrimPrefix(pin, "SHA256:")
	if pin != "" {
		return func(_ string, _ net.Addr, key ssh.PublicKey) error {
			sum := sha256.Sum256(key.Marshal())
			got := base64.RawStdEncoding.EncodeToString(sum[:])
			alt := base64.StdEncoding.EncodeToString(sum[:])
			hexGot := fmt.Sprintf("%x", sum[:])
			if subtleConstEq(pin, got) || subtleConstEq(pin, alt) || subtleConstEq(strings.ToLower(pin), hexGot) {
				return nil
			}
			return fmt.Errorf("sftp_host_key_mismatch")
		}, nil
	}
	if PartnerSFTPStrictHostKey() {
		return nil, fmt.Errorf("sftp_host_key_required")
	}
	return ssh.InsecureIgnoreHostKey(), nil //nolint:gosec // allowed only when strict mode off and pin empty
}

func subtleConstEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// ValidateSftpConfigPaths validates RemoteDir and relative subdirectory names against path traversal.
func ValidateSftpConfigPaths(cfg SftpConfig) error {
	for _, field := range []string{cfg.RemoteDir, cfg.InboundDir, cfg.OutboundDir, cfg.ArchiveDir} {
		if strings.ContainsRune(field, 0) || strings.Contains(field, "\\") {
			return fmt.Errorf("sftp_invalid_path")
		}
		for _, seg := range strings.Split(field, "/") {
			if seg == ".." {
				return fmt.Errorf("sftp_path_traversal_forbidden")
			}
		}
	}
	for _, sub := range []string{cfg.InboundDir, cfg.OutboundDir, cfg.ArchiveDir} {
		t := strings.TrimSpace(sub)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "/") || strings.Contains(t, "..") {
			return fmt.Errorf("sftp_subdir_must_be_relative")
		}
	}
	return nil
}

func validateSafeRemoteFilename(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, 0) || strings.Contains(name, "..") {
		return fmt.Errorf("sftp_invalid_filename")
	}
	return nil
}

// UploadSFTP pushes a local file to the configured remote directory.
func UploadSFTP(ctx context.Context, cfg SftpConfig, secret, localPath, remoteName string) error {
	if err := validateSafeRemoteFilename(remoteName); err != nil {
		return err
	}
	conn, client, err := dialSFTP(cfg, secret)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	remoteDir := strings.TrimSpace(cfg.RemoteDir)
	if remoteDir == "" {
		remoteDir = "/"
	}
	remotePath := joinRemote(remoteDir, remoteName)

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
	if err := ValidateHostNotPrivate(cfg.Host, DefaultWebhookURLPolicy()); err != nil {
		return nil, nil, fmt.Errorf("sftp_host_forbidden: %w", err)
	}
	if err := ValidateSftpConfigPaths(cfg); err != nil {
		return nil, nil, err
	}
	port := cfg.Port
	if port <= 0 {
		port = 22
	}
	auth, err := sftpAuthMethods(secret)
	if err != nil {
		return nil, nil, err
	}
	hk, err := sftpHostKeyCallback(cfg)
	if err != nil {
		return nil, nil, err
	}
	sshCfg := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            auth,
		HostKeyCallback: hk,
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
	cleanBase := path.Clean(base)
	cleanChild := strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(name)), "/")
	if cleanChild == "" || cleanChild == "." {
		return cleanBase
	}
	return path.Join(cleanBase, cleanChild)
}

func validateExistingRemotePath(p string) error {
	if strings.ContainsRune(p, 0) || strings.Contains(p, "\\") {
		return fmt.Errorf("sftp_invalid_path")
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return fmt.Errorf("sftp_path_traversal_forbidden")
		}
	}
	return nil
}

// ListSFTP lists non-directory files under remoteDir (absolute or under RemoteDir).
func ListSFTP(ctx context.Context, cfg SftpConfig, secret, remoteDir string) ([]SftpFileInfo, error) {
	dir := strings.TrimSpace(remoteDir)
	if dir == "" {
		dir = joinRemote(cfg.RemoteDir, cfg.InboundDir)
	}
	if err := validateExistingRemotePath(dir); err != nil {
		return nil, err
	}
	conn, client, err := dialSFTP(cfg, secret)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer client.Close()

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
	if err := validateExistingRemotePath(remotePath); err != nil {
		return nil, err
	}
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
	if err := validateExistingRemotePath(fromPath); err != nil {
		return err
	}
	if err := validateExistingRemotePath(toPath); err != nil {
		return err
	}
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

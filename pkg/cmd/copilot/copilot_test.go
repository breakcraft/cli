package copilot

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
)

func TestNewCmdCopilot(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	f := &cmdutil.Factory{
		IOStreams: ios,
	}

	cmd := NewCmdCopilot(f)

	if cmd.Use != "copilot [flags]" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short != "Run the GitHub Copilot CLI" {
		t.Errorf("unexpected Short: %s", cmd.Short)
	}

	if !cmd.DisableFlagParsing {
		t.Error("expected DisableFlagParsing to be true")
	}
}

func TestRemoveCopilot(t *testing.T) {
	t.Run("removes existing install directory", func(t *testing.T) {
		ios, _, _, stderr := iostreams.Test()

		// Create a temporary directory to simulate the install directory
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "copilot")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}
		// Create a dummy file in the directory
		dummyFile := filepath.Join(installDir, "copilot")
		if err := os.WriteFile(dummyFile, []byte("test"), 0755); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		err := removeCopilotFromDir(ios, installDir)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if _, err := os.Stat(installDir); !os.IsNotExist(err) {
			t.Error("expected install directory to be removed")
		}

		if stderr.String() != "Copilot CLI removed successfully\n" {
			t.Errorf("unexpected stderr: %s", stderr.String())
		}
	})

	t.Run("handles non-existent directory", func(t *testing.T) {
		ios, _, _, stderr := iostreams.Test()

		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "copilot")

		err := removeCopilotFromDir(ios, installDir)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if stderr.String() != "Copilot CLI is not installed\n" {
			t.Errorf("unexpected stderr: %s", stderr.String())
		}
	})
}

func TestSanitizeArchivePath(t *testing.T) {
	tests := []struct {
		name      string
		destDir   string
		entryName string
		wantErr   bool
	}{
		{
			name:      "valid simple path",
			destDir:   "/tmp/dest",
			entryName: "file.txt",
			wantErr:   false,
		},
		{
			name:      "valid nested path",
			destDir:   "/tmp/dest",
			entryName: "subdir/file.txt",
			wantErr:   false,
		},
		{
			name:      "path traversal attack",
			destDir:   "/tmp/dest",
			entryName: "../etc/passwd",
			wantErr:   true,
		},
		{
			name:      "nested path traversal attack",
			destDir:   "/tmp/dest",
			entryName: "subdir/../../etc/passwd",
			wantErr:   true,
		},
		{
			name:      "absolute path becomes relative",
			destDir:   "/tmp/dest",
			entryName: "/etc/passwd",
			wantErr:   false, // filepath.Join makes "/tmp/dest" + "/etc/passwd" -> "/tmp/dest/etc/passwd"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sanitizeArchivePath(tt.destDir, tt.entryName)
			if (err != nil) != tt.wantErr {
				t.Errorf("sanitizeArchivePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// createTarGz creates a tar.gz archive in memory with the given files.
func createTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("failed to write tar content: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	return buf.Bytes()
}

// createZip creates a zip archive in memory with the given files.
func createZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range files {
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry: %v", err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatalf("failed to write zip content: %v", err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestExtractTarGz(t *testing.T) {
	t.Run("extracts files correctly", func(t *testing.T) {
		destDir := t.TempDir()
		content := []byte("hello world")
		archive := createTarGz(t, map[string][]byte{
			"copilot": content,
		})

		err := extractTarGz(bytes.NewReader(archive), destDir)
		if err != nil {
			t.Fatalf("extractTarGz() error = %v", err)
		}

		extracted, err := os.ReadFile(filepath.Join(destDir, "copilot"))
		if err != nil {
			t.Fatalf("failed to read extracted file: %v", err)
		}
		if !bytes.Equal(extracted, content) {
			t.Errorf("extracted content = %q, want %q", extracted, content)
		}
	})

	t.Run("extracts nested files", func(t *testing.T) {
		destDir := t.TempDir()
		content := []byte("nested content")
		archive := createTarGz(t, map[string][]byte{
			"subdir/file.txt": content,
		})

		err := extractTarGz(bytes.NewReader(archive), destDir)
		if err != nil {
			t.Fatalf("extractTarGz() error = %v", err)
		}

		extracted, err := os.ReadFile(filepath.Join(destDir, "subdir", "file.txt"))
		if err != nil {
			t.Fatalf("failed to read extracted file: %v", err)
		}
		if !bytes.Equal(extracted, content) {
			t.Errorf("extracted content = %q, want %q", extracted, content)
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		destDir := t.TempDir()
		// Manually create a malicious tar.gz with path traversal
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)

		hdr := &tar.Header{
			Name: "../evil.txt",
			Mode: 0755,
			Size: 4,
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write([]byte("evil"))
		_ = tw.Close()
		_ = gw.Close()

		err := extractTarGz(bytes.NewReader(buf.Bytes()), destDir)
		if err == nil {
			t.Error("expected error for path traversal, got nil")
		}
	})

	t.Run("handles invalid gzip", func(t *testing.T) {
		destDir := t.TempDir()
		err := extractTarGz(bytes.NewReader([]byte("not valid gzip")), destDir)
		if err == nil {
			t.Error("expected error for invalid gzip, got nil")
		}
	})
}

func TestExtractZip(t *testing.T) {
	t.Run("extracts files correctly", func(t *testing.T) {
		destDir := t.TempDir()
		content := []byte("hello world")
		archive := createZip(t, map[string][]byte{
			"copilot.exe": content,
		})

		err := extractZip(bytes.NewReader(archive), destDir)
		if err != nil {
			t.Fatalf("extractZip() error = %v", err)
		}

		extracted, err := os.ReadFile(filepath.Join(destDir, "copilot.exe"))
		if err != nil {
			t.Fatalf("failed to read extracted file: %v", err)
		}
		if !bytes.Equal(extracted, content) {
			t.Errorf("extracted content = %q, want %q", extracted, content)
		}
	})

	t.Run("extracts nested files", func(t *testing.T) {
		destDir := t.TempDir()
		content := []byte("nested content")
		archive := createZip(t, map[string][]byte{
			"subdir/file.txt": content,
		})

		err := extractZip(bytes.NewReader(archive), destDir)
		if err != nil {
			t.Fatalf("extractZip() error = %v", err)
		}

		extracted, err := os.ReadFile(filepath.Join(destDir, "subdir", "file.txt"))
		if err != nil {
			t.Fatalf("failed to read extracted file: %v", err)
		}
		if !bytes.Equal(extracted, content) {
			t.Errorf("extracted content = %q, want %q", extracted, content)
		}
	})

	t.Run("handles invalid zip", func(t *testing.T) {
		destDir := t.TempDir()
		err := extractZip(bytes.NewReader([]byte("not valid zip")), destDir)
		if err == nil {
			t.Error("expected error for invalid zip, got nil")
		}
	})
}

func TestFetchExpectedChecksum(t *testing.T) {
	t.Run("parses checksums file correctly", func(t *testing.T) {
		reg := &httpmock.Registry{}
		checksums := "abc123def456  copilot-linux-x64.tar.gz\n789xyz  copilot-darwin-arm64.tar.gz\n"
		reg.Register(
			httpmock.MatchAny,
			httpmock.StringResponse(checksums),
		)

		client := &http.Client{Transport: reg}
		checksum, err := fetchExpectedChecksum(client, "https://example.com/checksums", "copilot-linux-x64.tar.gz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if checksum != "abc123def456" {
			t.Errorf("expected checksum abc123def456, got %s", checksum)
		}
	})

	t.Run("returns error for missing archive", func(t *testing.T) {
		reg := &httpmock.Registry{}
		checksums := "abc123  copilot-linux-x64.tar.gz\n"
		reg.Register(
			httpmock.MatchAny,
			httpmock.StringResponse(checksums),
		)

		client := &http.Client{Transport: reg}
		_, err := fetchExpectedChecksum(client, "https://example.com/checksums", "copilot-windows-x64.zip")
		if err == nil {
			t.Fatal("expected error for missing archive")
		}
		if err.Error() != "checksum not found for copilot-windows-x64.zip" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("handles single space separator", func(t *testing.T) {
		reg := &httpmock.Registry{}
		checksums := "abc123 copilot-darwin-x64.tar.gz\n"
		reg.Register(
			httpmock.MatchAny,
			httpmock.StringResponse(checksums),
		)

		client := &http.Client{Transport: reg}
		checksum, err := fetchExpectedChecksum(client, "https://example.com/checksums", "copilot-darwin-x64.tar.gz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if checksum != "abc123" {
			t.Errorf("expected checksum abc123, got %s", checksum)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		reg := &httpmock.Registry{}
		reg.Register(
			httpmock.MatchAny,
			httpmock.StatusStringResponse(http.StatusNotFound, "not found"),
		)

		client := &http.Client{Transport: reg}
		_, err := fetchExpectedChecksum(client, "https://example.com/checksums", "copilot-linux-x64.tar.gz")
		if err == nil {
			t.Fatal("expected error for HTTP 404")
		}
	})
}

func archString() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		return "x64"
	}
	return arch
}

func TestDownloadCopilot(t *testing.T) {
	// Skip on unsupported architectures
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("skipping test on unsupported architecture")
	}

	t.Run("downloads and extracts tar.gz with valid checksum", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping tar.gz test on windows")
		}

		ios, _, _, stderr := iostreams.Test()
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "copilot")
		localPath := filepath.Join(installDir, "copilot")

		// Create mock archive with copilot binary
		binaryContent := []byte("#!/bin/sh\necho copilot")
		archive := createTarGz(t, map[string][]byte{
			"copilot": binaryContent,
		})

		// Calculate checksum
		checksum := sha256.Sum256(archive)
		checksumHex := hex.EncodeToString(checksum[:])
		archiveName := fmt.Sprintf("copilot-%s-%s.tar.gz", runtime.GOOS, archString())
		checksumFile := fmt.Sprintf("%s  %s\n", checksumHex, archiveName)

		reg := &httpmock.Registry{}
		// Register checksum endpoint
		reg.Register(
			httpmock.REST("GET", "github/copilot-cli/releases/latest/download/SHA256SUMS.txt"),
			httpmock.StringResponse(checksumFile),
		)
		// Register archive endpoint
		reg.Register(
			httpmock.REST("GET", fmt.Sprintf("github/copilot-cli/releases/latest/download/%s", archiveName)),
			httpmock.BinaryResponse(archive),
		)

		httpClient := func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		}

		path, err := downloadCopilot(httpClient, ios, installDir, localPath)
		if err != nil {
			t.Fatalf("downloadCopilot() error = %v", err)
		}

		if path != localPath {
			t.Errorf("downloadCopilot() path = %q, want %q", path, localPath)
		}

		// Verify binary was extracted
		extracted, err := os.ReadFile(localPath)
		if err != nil {
			t.Fatalf("failed to read extracted binary: %v", err)
		}
		if !bytes.Equal(extracted, binaryContent) {
			t.Errorf("extracted content = %q, want %q", extracted, binaryContent)
		}

		// Verify output messages
		if !bytes.Contains(stderr.Bytes(), []byte("Downloading Copilot CLI")) {
			t.Error("expected download message in stderr")
		}
		if !bytes.Contains(stderr.Bytes(), []byte("installed successfully")) {
			t.Error("expected success message in stderr")
		}
	})

	t.Run("fails with checksum mismatch", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping tar.gz test on windows")
		}

		ios, _, _, _ := iostreams.Test()
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "copilot")
		localPath := filepath.Join(installDir, "copilot")

		binaryContent := []byte("#!/bin/sh\necho copilot")
		archive := createTarGz(t, map[string][]byte{
			"copilot": binaryContent,
		})

		// Use wrong checksum
		archiveName := fmt.Sprintf("copilot-%s-%s.tar.gz", runtime.GOOS, archString())
		checksumFile := fmt.Sprintf("%s  %s\n", "0000000000000000000000000000000000000000000000000000000000000000", archiveName)

		reg := &httpmock.Registry{}
		reg.Register(
			httpmock.REST("GET", "github/copilot-cli/releases/latest/download/SHA256SUMS.txt"),
			httpmock.StringResponse(checksumFile),
		)
		reg.Register(
			httpmock.REST("GET", fmt.Sprintf("github/copilot-cli/releases/latest/download/%s", archiveName)),
			httpmock.BinaryResponse(archive),
		)

		httpClient := func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		}

		_, err := downloadCopilot(httpClient, ios, installDir, localPath)
		if err == nil {
			t.Fatal("expected error for checksum mismatch, got nil")
		}
		if !bytes.Contains([]byte(err.Error()), []byte("checksum mismatch")) {
			t.Errorf("expected checksum mismatch error, got: %v", err)
		}
	})

	t.Run("handles HTTP error on archive download", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping tar.gz test on windows")
		}

		ios, _, _, _ := iostreams.Test()
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "copilot")
		localPath := filepath.Join(installDir, "copilot")

		archiveName := fmt.Sprintf("copilot-%s-%s.tar.gz", runtime.GOOS, archString())
		checksumFile := fmt.Sprintf("%s  %s\n", "abc123", archiveName)

		reg := &httpmock.Registry{}
		reg.Register(
			httpmock.REST("GET", "github/copilot-cli/releases/latest/download/SHA256SUMS.txt"),
			httpmock.StringResponse(checksumFile),
		)
		reg.Register(
			httpmock.REST("GET", fmt.Sprintf("github/copilot-cli/releases/latest/download/%s", archiveName)),
			httpmock.StatusStringResponse(http.StatusNotFound, "not found"),
		)

		httpClient := func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		}

		_, err := downloadCopilot(httpClient, ios, installDir, localPath)
		if err == nil {
			t.Fatal("expected error for HTTP 404, got nil")
		}
		if !bytes.Contains([]byte(err.Error()), []byte("download failed")) {
			t.Errorf("expected error to contain 'download failed', got: %v", err)
		}
	})

	t.Run("handles missing binary after extraction", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping tar.gz test on windows")
		}

		ios, _, _, _ := iostreams.Test()
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "copilot")
		localPath := filepath.Join(installDir, "copilot")

		// Create archive without the expected binary name
		archive := createTarGz(t, map[string][]byte{
			"wrong-name": []byte("content"),
		})

		checksum := sha256.Sum256(archive)
		checksumHex := hex.EncodeToString(checksum[:])
		archiveName := fmt.Sprintf("copilot-%s-%s.tar.gz", runtime.GOOS, archString())
		checksumFile := fmt.Sprintf("%s  %s\n", checksumHex, archiveName)

		reg := &httpmock.Registry{}
		reg.Register(
			httpmock.REST("GET", "github/copilot-cli/releases/latest/download/SHA256SUMS.txt"),
			httpmock.StringResponse(checksumFile),
		)
		reg.Register(
			httpmock.REST("GET", fmt.Sprintf("github/copilot-cli/releases/latest/download/%s", archiveName)),
			httpmock.BinaryResponse(archive),
		)

		httpClient := func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		}

		_, err := downloadCopilot(httpClient, ios, installDir, localPath)
		if err == nil {
			t.Fatal("expected error for missing binary, got nil")
		}
		if !bytes.Contains([]byte(err.Error()), []byte("not found after extraction")) {
			t.Errorf("expected 'not found after extraction' error, got: %v", err)
		}
	})

	t.Run("downloads and extracts zip on windows", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("skipping zip test on non-windows")
		}

		ios, _, _, _ := iostreams.Test()
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "copilot")
		localPath := filepath.Join(installDir, "copilot.exe")

		binaryContent := []byte("MZ fake exe content")
		archive := createZip(t, map[string][]byte{
			"copilot.exe": binaryContent,
		})

		checksum := sha256.Sum256(archive)
		checksumHex := hex.EncodeToString(checksum[:])
		archiveName := fmt.Sprintf("copilot-%s-%s.zip", runtime.GOOS, archString())
		checksumFile := fmt.Sprintf("%s  %s\n", checksumHex, archiveName)

		reg := &httpmock.Registry{}
		reg.Register(
			httpmock.REST("GET", "github/copilot-cli/releases/latest/download/SHA256SUMS.txt"),
			httpmock.StringResponse(checksumFile),
		)
		reg.Register(
			httpmock.REST("GET", fmt.Sprintf("github/copilot-cli/releases/latest/download/%s", archiveName)),
			httpmock.BinaryResponse(archive),
		)

		httpClient := func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		}

		path, err := downloadCopilot(httpClient, ios, installDir, localPath)
		if err != nil {
			t.Fatalf("downloadCopilot() error = %v", err)
		}

		if path != localPath {
			t.Errorf("downloadCopilot() path = %q, want %q", path, localPath)
		}
	})
}

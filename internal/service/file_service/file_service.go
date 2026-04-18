package file_service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type fileServiceImpl struct{}

var FileService = new(fileServiceImpl)

type UploadFileResult struct {
	FileKey  string `json:"fileKey"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	FileType string `json:"fileType"`
}

const (
	storageRootDir   = "storage"
	storageDocsDir   = "documents"
	defaultPDFType   = "application/pdf"
	uploadFieldName  = "file"
)

func (s *fileServiceImpl) UploadPDF(fileHeader *multipart.FileHeader) (*UploadFileResult, error) {
	if fileHeader == nil {
		return nil, fmt.Errorf("file is required")
	}

	origName := strings.TrimSpace(fileHeader.Filename)
	if origName == "" {
		return nil, fmt.Errorf("fileName is required")
	}

	ext := strings.ToLower(path.Ext(origName))
	if ext != ".pdf" {
		return nil, fmt.Errorf("only pdf file is supported")
	}

	contentType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	if contentType == "" || !strings.Contains(contentType, "pdf") {
		return nil, fmt.Errorf("only pdf file is supported")
	}

	now := time.Now()
	yyyy := now.Format("2006")
	mm := now.Format("01")

	relDir := path.Join(storageDocsDir, yyyy, mm)
	absDir := filepath.Join(storageRootDir, filepath.FromSlash(relDir))
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("save file failed: %w", err)
	}

	randHex, err := randomHex(6)
	if err != nil {
		return nil, fmt.Errorf("save file failed: %w", err)
	}

	safeOrig := sanitizeFileName(origName)
	fileName := fmt.Sprintf("%d_%s_%s", time.Now().UnixMilli(), randHex, safeOrig)
	fileKey := path.Join(relDir, fileName)
	absPath := filepath.Join(storageRootDir, filepath.FromSlash(fileKey))

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("save file failed: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("save file failed: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		_ = os.Remove(absPath)
		return nil, fmt.Errorf("save file failed: %w", err)
	}
	if written <= 0 {
		_ = os.Remove(absPath)
		return nil, fmt.Errorf("uploaded file is empty")
	}

	return &UploadFileResult{
		FileKey:  filepath.ToSlash(fileKey),
		FileName: origName,
		FileSize: written,
		FileType: defaultPDFType,
	}, nil
}

func (s *fileServiceImpl) AbsPathFromFileKey(fileKey string) string {
	fileKey = strings.TrimSpace(fileKey)
	return filepath.Join(storageRootDir, filepath.FromSlash(fileKey))
}

// BuildVersionFileKey 生成签署版本 PDF 的 fileKey（相对 storage 根目录，使用正斜杠）。
// 规则：documents/workflows/{workflowId}/{documentId}/versions/v{versionNo}.pdf
func (s *fileServiceImpl) BuildVersionFileKey(workflowID uint, documentID uint, versionNo int) string {
	ws := strconv.FormatUint(uint64(workflowID), 10)
	ds := strconv.FormatUint(uint64(documentID), 10)
	return path.Join(storageDocsDir, "workflows", ws, ds, "versions", fmt.Sprintf("v%d.pdf", versionNo))
}

// EnsureParentDirsForFileKey 为给定 fileKey 创建上级目录（不创建文件本身）。
func (s *fileServiceImpl) EnsureParentDirsForFileKey(fileKey string) error {
	fileKey = strings.TrimSpace(fileKey)
	if fileKey == "" {
		return fmt.Errorf("fileKey is empty")
	}
	absDir := filepath.Dir(filepath.Join(storageRootDir, filepath.FromSlash(fileKey)))
	return os.MkdirAll(absDir, 0o755)
}

// OpenDocumentByFileKey 将 fileKey 转为绝对路径并校验文件存在且为普通文件。
func (s *fileServiceImpl) OpenDocumentByFileKey(fileKey string) (string, error) {
	fileKey = strings.TrimSpace(fileKey)
	if fileKey == "" {
		return "", fmt.Errorf("fileKey is empty")
	}
	absPath := s.AbsPathFromFileKey(fileKey)
	st, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("stored file not found")
		}
		return "", err
	}
	if st.IsDir() {
		return "", fmt.Errorf("stored file not found")
	}
	return absPath, nil
}

// ResolveDownloadFileName 优先使用业务文件名，若为空则回退 fileKey 的 basename。
func (s *fileServiceImpl) ResolveDownloadFileName(preferredName string, fileKey string) string {
	name := strings.TrimSpace(preferredName)
	if name != "" {
		return name
	}
	base := strings.TrimSpace(path.Base(strings.TrimSpace(fileKey)))
	if base != "" && base != "." && base != "/" {
		return base
	}
	return "document.pdf"
}

// BuildAttachmentContentDisposition 构建附件下载头（兼容常见浏览器 UTF-8 文件名）。
func (s *fileServiceImpl) BuildAttachmentContentDisposition(fileName string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "document.pdf"
	}
	safeQuoted := strings.ReplaceAll(name, "\"", "'")
	escaped := url.PathEscape(name)
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", safeQuoted, escaped)
}

func randomHex(nBytes int) (string, error) {
	if nBytes <= 0 {
		return "", fmt.Errorf("invalid random length")
	}
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}


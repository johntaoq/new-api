package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	imagePlaygroundBlobPrefix     = "image-playground/users"
	imagePlaygroundDefaultMaxUser = 50
	azureBlobAPIVersion           = "2020-10-02"
	imagePlaygroundBlobTimeout    = 20 * time.Second
)

var imagePlaygroundBlobHTTPClient = &http.Client{
	Timeout: imagePlaygroundBlobTimeout,
}

type imagePlaygroundBlobConfig struct {
	AccountName   string
	AccountKey    string
	Container     string
	Endpoint      string
	PublicBaseURL string
	MaxPerUser    int
	Enabled       bool
}

type azureBlobListResponse struct {
	Blobs []azureBlobListItem `xml:"Blobs>Blob"`
}

type azureBlobListItem struct {
	Name       string `xml:"Name"`
	Properties struct {
		LastModified string `xml:"Last-Modified"`
	} `xml:"Properties"`
	lastModifiedTime time.Time
}

func ArchivePlaygroundImageResponse(c *gin.Context, responseBody []byte) []byte {
	if len(responseBody) == 0 {
		return responseBody
	}

	cfg, ok := loadImagePlaygroundBlobConfig()
	if !ok {
		return responseBody
	}

	userID := c.GetInt("id")
	if userID <= 0 {
		return responseBody
	}

	var imageResponse dto.ImageResponse
	if err := json.Unmarshal(responseBody, &imageResponse); err != nil || len(imageResponse.Data) == 0 {
		return responseBody
	}

	changed := false
	for index := range imageResponse.Data {
		imageBytes, contentType, err := getImagePlaygroundBytes(imageResponse.Data[index])
		if err != nil {
			common.SysLog(fmt.Sprintf("image playground archive skipped: %s", err.Error()))
			continue
		}

		blobURL, err := uploadPlaygroundImageBlob(cfg, userID, index, imageBytes, contentType)
		if err != nil {
			common.SysLog(fmt.Sprintf("image playground archive upload failed: %s", err.Error()))
			continue
		}

		imageResponse.Data[index].Url = blobURL
		imageResponse.Data[index].B64Json = ""
		changed = true
	}

	if !changed {
		return responseBody
	}

	if err := cleanupPlaygroundImageBlobs(cfg, userID); err != nil {
		common.SysLog(fmt.Sprintf("image playground archive cleanup failed: %s", err.Error()))
	}

	archivedBody, err := json.Marshal(imageResponse)
	if err != nil {
		return responseBody
	}
	return archivedBody
}

func loadImagePlaygroundBlobConfig() (imagePlaygroundBlobConfig, bool) {
	if strings.EqualFold(os.Getenv("IMAGE_PLAYGROUND_BLOB_ENABLED"), "false") {
		return imagePlaygroundBlobConfig{}, false
	}

	values := parseAzureConnectionString(os.Getenv("AZURE_STORAGE_CONNECTION_STRING"))
	cfg := imagePlaygroundBlobConfig{
		AccountName:   firstNonEmpty(values["AccountName"], os.Getenv("AZURE_STORAGE_ACCOUNT"), os.Getenv("IMAGE_PLAYGROUND_BLOB_ACCOUNT"), os.Getenv("IMAGE_STUDIO_BLOB_ACCOUNT")),
		AccountKey:    firstNonEmpty(values["AccountKey"], os.Getenv("AZURE_STORAGE_KEY"), os.Getenv("IMAGE_PLAYGROUND_BLOB_KEY"), os.Getenv("IMAGE_STUDIO_BLOB_KEY")),
		Container:     firstNonEmpty(os.Getenv("IMAGE_PLAYGROUND_BLOB_CONTAINER"), os.Getenv("AZURE_STORAGE_CONTAINER"), os.Getenv("IMAGE_STUDIO_BLOB_CONTAINER")),
		Endpoint:      strings.TrimRight(firstNonEmpty(values["BlobEndpoint"], os.Getenv("AZURE_STORAGE_BLOB_ENDPOINT")), "/"),
		PublicBaseURL: strings.TrimRight(os.Getenv("IMAGE_PLAYGROUND_BLOB_PUBLIC_BASE_URL"), "/"),
		MaxPerUser:    imagePlaygroundDefaultMaxUser,
		Enabled:       true,
	}

	if cfg.Endpoint == "" && cfg.AccountName != "" {
		cfg.Endpoint = fmt.Sprintf("https://%s.blob.core.windows.net", cfg.AccountName)
	}

	if rawMax := os.Getenv("IMAGE_PLAYGROUND_BLOB_MAX_PER_USER"); rawMax != "" {
		if maxPerUser, err := strconv.Atoi(rawMax); err == nil && maxPerUser > 0 {
			cfg.MaxPerUser = maxPerUser
		}
	}

	if cfg.AccountName == "" || cfg.AccountKey == "" || cfg.Container == "" || cfg.Endpoint == "" {
		return imagePlaygroundBlobConfig{}, false
	}
	return cfg, true
}

func parseAzureConnectionString(connectionString string) map[string]string {
	values := map[string]string{}
	for _, part := range strings.Split(connectionString, ";") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func getImagePlaygroundBytes(imageData dto.ImageData) ([]byte, string, error) {
	if imageData.B64Json != "" {
		contentType, cleanBase64, err := DecodeBase64FileData(imageData.B64Json)
		if err != nil {
			return nil, "", err
		}
		imageBytes, err := base64.StdEncoding.DecodeString(cleanBase64)
		if err != nil {
			return nil, "", err
		}
		return imageBytes, normalizeImageContentType(contentType, imageBytes), nil
	}

	if imageData.Url != "" {
		contentType, cleanBase64, err := GetImageFromUrl(imageData.Url)
		if err != nil {
			return nil, "", err
		}
		imageBytes, err := base64.StdEncoding.DecodeString(cleanBase64)
		if err != nil {
			return nil, "", err
		}
		return imageBytes, normalizeImageContentType(contentType, imageBytes), nil
	}

	return nil, "", fmt.Errorf("image response has neither url nor b64_json")
}

func normalizeImageContentType(contentType string, imageBytes []byte) string {
	if strings.HasPrefix(contentType, "image/") {
		return contentType
	}
	detected := http.DetectContentType(imageBytes)
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	return "image/png"
}

func uploadPlaygroundImageBlob(cfg imagePlaygroundBlobConfig, userID int, index int, imageBytes []byte, contentType string) (string, error) {
	blobName := fmt.Sprintf(
		"%s/%d/%d-%02d-%s%s",
		imagePlaygroundBlobPrefix,
		userID,
		time.Now().UTC().UnixNano(),
		index+1,
		uuid.NewString(),
		imageExtension(contentType),
	)

	uploadURL := buildAzureBlobURL(cfg, blobName, nil)
	req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(imageBytes))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(imageBytes))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", strconv.Itoa(len(imageBytes)))
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-date", azureTimeNow())
	req.Header.Set("x-ms-version", azureBlobAPIVersion)

	if err := signAzureBlobRequest(req, cfg, strconv.Itoa(len(imageBytes))); err != nil {
		return "", err
	}

	resp, err := imagePlaygroundBlobHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		if err := createPlaygroundImageContainer(cfg); err != nil {
			return "", err
		}
		return uploadPlaygroundImageBlob(cfg, userID, index, imageBytes, contentType)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("azure blob upload failed: HTTP %d %s", resp.StatusCode, string(body))
	}

	return buildPlaygroundBlobPublicURL(cfg, blobName), nil
}

func createPlaygroundImageContainer(cfg imagePlaygroundBlobConfig) error {
	query := url.Values{}
	query.Set("restype", "container")

	req, err := http.NewRequest(http.MethodPut, buildAzureBlobURL(cfg, "", query), nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-ms-date", azureTimeNow())
	req.Header.Set("x-ms-version", azureBlobAPIVersion)
	req.Header.Set("x-ms-blob-public-access", "blob")
	if err := signAzureBlobRequest(req, cfg, ""); err != nil {
		return err
	}

	resp, err := imagePlaygroundBlobHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("azure blob create container failed: HTTP %d %s", resp.StatusCode, string(body))
	}
	return nil
}

func cleanupPlaygroundImageBlobs(cfg imagePlaygroundBlobConfig, userID int) error {
	items, err := listPlaygroundImageBlobs(cfg, userID)
	if err != nil {
		return err
	}
	if len(items) <= cfg.MaxPerUser {
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].lastModifiedTime.Equal(items[j].lastModifiedTime) {
			return items[i].lastModifiedTime.Before(items[j].lastModifiedTime)
		}
		return items[i].Name < items[j].Name
	})

	for _, item := range items[:len(items)-cfg.MaxPerUser] {
		if err := deletePlaygroundImageBlob(cfg, item.Name); err != nil {
			common.SysLog(fmt.Sprintf("image playground archive delete failed for %s: %s", item.Name, err.Error()))
		}
	}
	return nil
}

func listPlaygroundImageBlobs(cfg imagePlaygroundBlobConfig, userID int) ([]azureBlobListItem, error) {
	query := url.Values{}
	query.Set("restype", "container")
	query.Set("comp", "list")
	query.Set("prefix", fmt.Sprintf("%s/%d/", imagePlaygroundBlobPrefix, userID))

	req, err := http.NewRequest(http.MethodGet, buildAzureBlobURL(cfg, "", query), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-ms-date", azureTimeNow())
	req.Header.Set("x-ms-version", azureBlobAPIVersion)
	if err := signAzureBlobRequest(req, cfg, ""); err != nil {
		return nil, err
	}

	resp, err := imagePlaygroundBlobHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("azure blob list failed: HTTP %d %s", resp.StatusCode, string(body))
	}

	var listResponse azureBlobListResponse
	if err := xml.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		return nil, err
	}

	for index := range listResponse.Blobs {
		if parsed, err := http.ParseTime(listResponse.Blobs[index].Properties.LastModified); err == nil {
			listResponse.Blobs[index].lastModifiedTime = parsed
		}
	}
	return listResponse.Blobs, nil
}

func deletePlaygroundImageBlob(cfg imagePlaygroundBlobConfig, blobName string) error {
	req, err := http.NewRequest(http.MethodDelete, buildAzureBlobURL(cfg, blobName, nil), nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-ms-date", azureTimeNow())
	req.Header.Set("x-ms-version", azureBlobAPIVersion)
	if err := signAzureBlobRequest(req, cfg, ""); err != nil {
		return err
	}

	resp, err := imagePlaygroundBlobHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("azure blob delete failed: HTTP %d %s", resp.StatusCode, string(body))
	}
	return nil
}

func buildAzureBlobURL(cfg imagePlaygroundBlobConfig, blobName string, query url.Values) string {
	blobURL, _ := url.Parse(cfg.Endpoint)
	blobURL.Path = "/" + cfg.Container
	if blobName != "" {
		blobURL.Path += "/" + blobName
	}
	blobURL.RawQuery = query.Encode()
	return blobURL.String()
}

func buildPlaygroundBlobPublicURL(cfg imagePlaygroundBlobConfig, blobName string) string {
	base := cfg.PublicBaseURL
	if base != "" {
		blobURL, err := url.Parse(base)
		if err != nil {
			return strings.TrimRight(base, "/") + "/" + escapeBlobPath(blobName)
		}
		blobURL.Path = strings.TrimRight(blobURL.Path, "/") + "/" + escapeBlobPath(blobName)
		return blobURL.String()
	}

	blobURL, err := url.Parse(fmt.Sprintf("%s/%s/%s", strings.TrimRight(cfg.Endpoint, "/"), cfg.Container, escapeBlobPath(blobName)))
	if err != nil {
		return strings.TrimRight(cfg.Endpoint, "/") + "/" + cfg.Container + "/" + escapeBlobPath(blobName)
	}
	blobURL.RawQuery = buildPlaygroundBlobSASQuery(cfg, blobName).Encode()
	return blobURL.String()
}

func buildPlaygroundBlobSASQuery(cfg imagePlaygroundBlobConfig, blobName string) url.Values {
	start := time.Now().UTC().Add(-5 * time.Minute)
	expiry := start.Add(30 * 24 * time.Hour)
	resource := fmt.Sprintf("/blob/%s/%s/%s", cfg.AccountName, cfg.Container, blobName)
	params := url.Values{}
	params.Set("sp", "r")
	params.Set("st", start.Format(time.RFC3339))
	params.Set("se", expiry.Format(time.RFC3339))
	params.Set("spr", "https")
	params.Set("sv", azureBlobAPIVersion)
	params.Set("sr", "b")
	params.Set("sig", signPlaygroundBlobSAS(cfg, resource, params))
	return params
}

func signPlaygroundBlobSAS(cfg imagePlaygroundBlobConfig, canonicalizedResource string, params url.Values) string {
	decodedKey, err := base64.StdEncoding.DecodeString(cfg.AccountKey)
	if err != nil {
		return ""
	}

	stringToSign := strings.Join([]string{
		params.Get("sp"),
		params.Get("st"),
		params.Get("se"),
		canonicalizedResource,
		"",
		"",
		params.Get("spr"),
		params.Get("sv"),
		params.Get("sr"),
		"",
		"",
		"",
		"",
		"",
		"",
	}, "\n")

	mac := hmac.New(sha256.New, decodedKey)
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func escapeBlobPath(blobName string) string {
	parts := strings.Split(blobName, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}

func imageExtension(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func signAzureBlobRequest(req *http.Request, cfg imagePlaygroundBlobConfig, contentLength string) error {
	decodedKey, err := base64.StdEncoding.DecodeString(cfg.AccountKey)
	if err != nil {
		return err
	}

	stringToSign := strings.Join([]string{
		req.Method,
		"",
		"",
		contentLength,
		"",
		req.Header.Get("Content-Type"),
		"",
		"",
		"",
		"",
		"",
		"",
		canonicalizedAzureHeaders(req),
		canonicalizedAzureResource(req, cfg.AccountName),
	}, "\n")

	mac := hmac.New(sha256.New, decodedKey)
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	req.Header.Set("Authorization", fmt.Sprintf("SharedKey %s:%s", cfg.AccountName, signature))
	return nil
}

func canonicalizedAzureHeaders(req *http.Request) string {
	var names []string
	values := map[string]string{}
	for name, headerValues := range req.Header {
		lowerName := strings.ToLower(name)
		if !strings.HasPrefix(lowerName, "x-ms-") {
			continue
		}
		names = append(names, lowerName)
		values[lowerName] = strings.Join(headerValues, ",")
	}
	sort.Strings(names)

	var builder strings.Builder
	for _, name := range names {
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(name)
		builder.WriteByte(':')
		builder.WriteString(strings.TrimSpace(values[name]))
	}
	return builder.String()
}

func canonicalizedAzureResource(req *http.Request, accountName string) string {
	var builder strings.Builder
	builder.WriteByte('/')
	builder.WriteString(accountName)
	builder.WriteString(req.URL.EscapedPath())

	query := req.URL.Query()
	var queryNames []string
	for name := range query {
		queryNames = append(queryNames, strings.ToLower(name))
	}
	sort.Strings(queryNames)
	for _, name := range queryNames {
		values := query[name]
		sort.Strings(values)
		builder.WriteByte('\n')
		builder.WriteString(name)
		builder.WriteByte(':')
		builder.WriteString(strings.Join(values, ","))
	}
	return builder.String()
}

func azureTimeNow() string {
	return time.Now().UTC().Format(http.TimeFormat)
}

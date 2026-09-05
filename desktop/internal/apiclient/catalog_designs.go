package apiclient

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CatalogPart struct {
	ID            string `json:"id"`
	CatalogItemID string `json:"catalog_item_id"`
	Name          string `json:"name"`
	Quantity      int    `json:"quantity"`
	Notes         string `json:"notes"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type CatalogPartInput struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Notes    string `json:"notes"`
}

type DesignVersion struct {
	ID                   string       `json:"id"`
	CatalogPartID        string       `json:"catalog_part_id"`
	Version              string       `json:"version"`
	Notes                string       `json:"notes"`
	Origin               string       `json:"origin"`
	SourceURL            *string      `json:"source_url"`
	OriginalAuthor       string       `json:"original_author"`
	LicenseName          string       `json:"license_name"`
	CommercialUseAllowed *bool        `json:"commercial_use_allowed"`
	AttributionRequired  bool         `json:"attribution_required"`
	AttributionText      string       `json:"attribution_text"`
	CreatedBy            string       `json:"created_by"`
	CreatedAt            string       `json:"created_at"`
	Files                []DesignFile `json:"files"`
}

type DesignVersionInput struct {
	Version              string  `json:"version"`
	Notes                string  `json:"notes"`
	Origin               string  `json:"origin"`
	SourceURL            *string `json:"source_url"`
	OriginalAuthor       string  `json:"original_author"`
	LicenseName          string  `json:"license_name"`
	CommercialUseAllowed *bool   `json:"commercial_use_allowed"`
	AttributionRequired  bool    `json:"attribution_required"`
	AttributionText      string  `json:"attribution_text"`
}

type DesignFile struct {
	FileID       string `json:"file_id"`
	Role         string `json:"role"`
	OriginalName string `json:"original_name"`
	ContentType  string `json:"content_type"`
	SizeBytes    int64  `json:"size_bytes"`
	SHA256       string `json:"sha256"`
	CreatedAt    string `json:"created_at"`
}

func (client *Client) ListCatalogParts(ctx context.Context, token, itemID string) ([]CatalogPart, error) {
	var result struct {
		Parts []CatalogPart `json:"parts"`
	}
	path := catalogItemsPath + "/" + url.PathEscape(strings.TrimSpace(itemID)) + "/parts"
	if err := client.catalogJSON(ctx, http.MethodGet, path, token, nil, &result); err != nil {
		return nil, err
	}
	if result.Parts == nil {
		return nil, invalidResponseError("Server returned invalid catalog parts", nil)
	}
	for _, part := range result.Parts {
		if !validCatalogPart(part) {
			return nil, invalidResponseError("Server returned invalid catalog parts", nil)
		}
	}
	return result.Parts, nil
}

func (client *Client) CreateCatalogPart(ctx context.Context, token, itemID string, input CatalogPartInput) (CatalogPart, error) {
	var part CatalogPart
	path := catalogItemsPath + "/" + url.PathEscape(strings.TrimSpace(itemID)) + "/parts"
	if err := client.catalogJSON(ctx, http.MethodPost, path, token, input, &part); err != nil {
		return CatalogPart{}, err
	}
	if !validCatalogPart(part) {
		return CatalogPart{}, invalidResponseError("Server returned invalid catalog part", nil)
	}
	return part, nil
}

func (client *Client) ListDesignVersions(ctx context.Context, token, partID string) ([]DesignVersion, error) {
	var result struct {
		Versions []DesignVersion `json:"versions"`
	}
	path := "/api/v1/catalog/parts/" + url.PathEscape(strings.TrimSpace(partID)) + "/design-versions"
	if err := client.catalogJSON(ctx, http.MethodGet, path, token, nil, &result); err != nil {
		return nil, err
	}
	if result.Versions == nil {
		return nil, invalidResponseError("Server returned invalid design history", nil)
	}
	for _, version := range result.Versions {
		if !validDesignVersion(version) {
			return nil, invalidResponseError("Server returned invalid design history", nil)
		}
	}
	return result.Versions, nil
}

func (client *Client) CreateDesignVersion(ctx context.Context, token, partID string, input DesignVersionInput) (DesignVersion, error) {
	var version DesignVersion
	path := "/api/v1/catalog/parts/" + url.PathEscape(strings.TrimSpace(partID)) + "/design-versions"
	if err := client.catalogJSON(ctx, http.MethodPost, path, token, input, &version); err != nil {
		return DesignVersion{}, err
	}
	if !validDesignVersion(version) {
		return DesignVersion{}, invalidResponseError("Server returned invalid design version", nil)
	}
	return version, nil
}

func (client *Client) AttachDesignFile(ctx context.Context, token, versionID, fileID, role string) (DesignFile, error) {
	var file DesignFile
	path := "/api/v1/catalog/design-versions/" + url.PathEscape(strings.TrimSpace(versionID)) + "/files"
	input := struct {
		FileID string `json:"file_id"`
		Role   string `json:"role"`
	}{FileID: fileID, Role: role}
	if err := client.catalogJSON(ctx, http.MethodPost, path, token, input, &file); err != nil {
		return DesignFile{}, err
	}
	if !validDesignFile(file) {
		return DesignFile{}, invalidResponseError("Server returned invalid design file", nil)
	}
	return file, nil
}

func validCatalogPart(part CatalogPart) bool {
	if strings.TrimSpace(part.ID) == "" || strings.TrimSpace(part.CatalogItemID) == "" || strings.TrimSpace(part.Name) == "" || part.Quantity < 1 {
		return false
	}
	if _, err := time.Parse(time.RFC3339, part.CreatedAt); err != nil {
		return false
	}
	_, err := time.Parse(time.RFC3339, part.UpdatedAt)
	return err == nil
}

func validDesignVersion(version DesignVersion) bool {
	if strings.TrimSpace(version.ID) == "" || strings.TrimSpace(version.CatalogPartID) == "" || strings.TrimSpace(version.Version) == "" || strings.TrimSpace(version.CreatedBy) == "" || version.Files == nil {
		return false
	}
	if _, err := time.Parse(time.RFC3339, version.CreatedAt); err != nil {
		return false
	}
	switch version.Origin {
	case "original", "customer", "remix", "third_party", "unknown":
	default:
		return false
	}
	for _, file := range version.Files {
		if !validDesignFile(file) {
			return false
		}
	}
	return true
}

func validDesignFile(file DesignFile) bool {
	if strings.TrimSpace(file.FileID) == "" || len(file.SHA256) != 64 || file.SizeBytes < 0 {
		return false
	}
	if _, err := hex.DecodeString(file.SHA256); err != nil {
		return false
	}
	if _, err := time.Parse(time.RFC3339, file.CreatedAt); err != nil {
		return false
	}
	switch file.Role {
	case "source", "mesh", "print", "preview", "documentation", "other":
		return true
	default:
		return false
	}
}

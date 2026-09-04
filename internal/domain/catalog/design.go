package catalog

import (
	"errors"
	"time"
)

var (
	ErrPartNotFound          = errors.New("catalog part not found")
	ErrDesignVersionNotFound = errors.New("design version not found")
	ErrDesignVersionConflict = errors.New("design version already exists")
	ErrDesignFileConflict    = errors.New("design file link already exists")
	ErrDesignFileNotFound    = errors.New("design version or file not found")
	ErrDesignHistoryExists   = errors.New("immutable design history exists")
)

type Part struct {
	ID            string
	CatalogItemID string
	Name          string
	Quantity      int
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PartValues struct {
	Name     string
	Quantity int
	Notes    string
}

type DesignOrigin string

const (
	DesignOriginOriginal   DesignOrigin = "original"
	DesignOriginCustomer   DesignOrigin = "customer"
	DesignOriginRemix      DesignOrigin = "remix"
	DesignOriginThirdParty DesignOrigin = "third_party"
	DesignOriginUnknown    DesignOrigin = "unknown"
)

type DesignVersion struct {
	ID                   string
	CatalogPartID        string
	Version              string
	Notes                string
	Origin               DesignOrigin
	SourceURL            *string
	OriginalAuthor       string
	LicenseName          string
	CommercialUseAllowed *bool
	AttributionRequired  bool
	AttributionText      string
	CreatedBy            string
	CreatedAt            time.Time
	Files                []DesignFile
}

type DesignVersionValues struct {
	Version              string
	Notes                string
	Origin               DesignOrigin
	SourceURL            *string
	OriginalAuthor       string
	LicenseName          string
	CommercialUseAllowed *bool
	AttributionRequired  bool
	AttributionText      string
}

type DesignFileRole string

const (
	DesignFileSource        DesignFileRole = "source"
	DesignFileMesh          DesignFileRole = "mesh"
	DesignFilePrint         DesignFileRole = "print"
	DesignFilePreview       DesignFileRole = "preview"
	DesignFileDocumentation DesignFileRole = "documentation"
	DesignFileOther         DesignFileRole = "other"
)

type DesignFile struct {
	FileID       string
	Role         DesignFileRole
	OriginalName string
	ContentType  string
	SizeBytes    int64
	SHA256Hex    string
	CreatedAt    time.Time
}

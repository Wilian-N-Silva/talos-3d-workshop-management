// Package catalog contains catalog domain definitions.
package catalog

import (
	"errors"
	"time"
)

// ErrItemNotFound indicates that a catalog item does not exist.
var ErrItemNotFound = errors.New("catalog item not found")

// Purpose classifies why an item exists in the workshop.
type Purpose string

const (
	PurposeProduct   Purpose = "product"
	PurposePrototype Purpose = "prototype"
	PurposeTooling   Purpose = "tooling"
	PurposeTest      Purpose = "test"
	PurposeInternal  Purpose = "internal"
	PurposePersonal  Purpose = "personal"
)

// Status controls whether an item remains active without deleting its history.
type Status string

const (
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

// Item is one generic workshop catalog entry.
type Item struct {
	ID          string
	Name        string
	SKU         *string
	Description string
	Purpose     Purpose
	Sellable    bool
	Tags        []string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Values contains the mutable fields of a catalog item.
type Values struct {
	Name        string
	SKU         *string
	Description string
	Purpose     Purpose
	Sellable    bool
	Tags        []string
	Status      Status
}

// ListFilter defines parameterized catalog list filters and pagination.
type ListFilter struct {
	Purpose  *Purpose
	Status   *Status
	Sellable *bool
	Tag      string
	Query    string
	Limit    int
	Offset   int
}

// Page is one catalog list result with total matching rows.
type Page struct {
	Items  []Item
	Total  int64
	Limit  int
	Offset int
}

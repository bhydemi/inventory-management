package models

// Item represents a single piece of inventory stored in the database.
type Item struct {
	ID    string  `json:"id" gorm:"type:uuid;primaryKey"`
	Name  string  `json:"name" gorm:"not null;index"`
	Stock int     `json:"stock" gorm:"not null"`
	Price float64 `json:"price" gorm:"not null"`
}

// ItemInput is the shape clients submit when creating or updating an item.
// Keeping it separate from Item stops a client from setting its own ID.
type ItemInput struct {
	Name  string  `json:"name" binding:"required"`
	Stock int     `json:"stock" binding:"gte=0"`
	Price float64 `json:"price" binding:"gte=0"`
}

// AllowedSortFields whitelists which columns a client may sort by, so a
// sort_by query parameter can never be used to inject arbitrary SQL.
var AllowedSortFields = map[string]string{
	"name":  "name",
	"stock": "stock",
	"price": "price",
}

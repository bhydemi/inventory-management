package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inventory_management/database"
	"inventory_management/models"
)

// GetItems handles GET /inventory. It supports pagination (limit, offset),
// filtering (name, min_stock), and sorting (sort_by, order).
func GetItems(c *gin.Context) {
	query := database.DB.Model(&models.Item{})

	// --- filtering ---
	if name := c.Query("name"); name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}
	if minStockStr := c.Query("min_stock"); minStockStr != "" {
		minStock, err := strconv.Atoi(minStockStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "min_stock must be an integer"})
			return
		}
		query = query.Where("stock >= ?", minStock)
	}

	// total matches the filter, before pagination is applied -- lets a
	// client page through a filtered result set correctly.
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count items"})
		return
	}

	// --- sorting ---
	sortBy := strings.ToLower(c.DefaultQuery("sort_by", "name"))
	column, ok := models.AllowedSortFields[sortBy]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sort_by must be one of: name, stock, price"})
		return
	}
	order := strings.ToLower(c.DefaultQuery("order", "asc"))
	if order != "asc" && order != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order must be 'asc' or 'desc'"})
		return
	}
	query = query.Order(column + " " + order)

	// --- pagination ---
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be a non-negative integer"})
		return
	}

	var items []models.Item
	if err := query.Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetItem handles GET /inventory/:id.
func GetItem(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var item models.Item
	if err := database.DB.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// CreateItem handles POST /inventory.
func CreateItem(c *gin.Context) {
	var input models.ItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := models.Item{
		ID:    uuid.NewString(),
		Name:  input.Name,
		Stock: input.Stock,
		Price: input.Price,
	}

	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create item"})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// UpdateItem handles PUT /inventory/:id.
func UpdateItem(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var item models.Item
	if err := database.DB.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	var input models.ItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item.Name = input.Name
	item.Stock = input.Stock
	item.Price = input.Price

	if err := database.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteItem handles DELETE /inventory/:id.
func DeleteItem(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var item models.Item
	if err := database.DB.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	if err := database.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item deleted", "id": id})
}

// parseID reads the :id path param and rejects anything that isn't a
// well-formed UUID before it ever reaches a database query.
func parseID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id: must be a UUID"})
		return "", false
	}
	return id, true
}

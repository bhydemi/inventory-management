package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inventory_management/cache"
	"inventory_management/database"
	"inventory_management/models"
)

// itemCache holds short-lived copies of GET responses. Any write (create,
// update, delete) flushes it, so a cached response can never outlive the
// data it was built from by more than its TTL.
var itemCache = cache.New(10 * time.Second)

// GetItems godoc
// @Summary      List inventory items
// @Description  Returns a paginated, filterable, sortable list of items.
// @Tags         inventory
// @Produce      json
// @Param        limit      query     int     false  "max items to return"      default(10)
// @Param        offset     query     int     false  "items to skip"            default(0)
// @Param        sort_by    query     string  false  "name, stock, or price"    default(name)
// @Param        order      query     string  false  "asc or desc"              default(asc)
// @Param        name       query     string  false  "substring filter on name"
// @Param        min_stock  query     int     false  "minimum stock level"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Router       /inventory [get]
func GetItems(c *gin.Context) {
	cacheKey := "list:" + c.Request.URL.RawQuery
	if cached, ok := itemCache.Get(cacheKey); ok {
		c.Data(http.StatusOK, "application/json; charset=utf-8", cached)
		return
	}

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

	body := gin.H{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	if encoded, err := json.Marshal(body); err == nil {
		itemCache.Set(cacheKey, encoded)
	}

	c.JSON(http.StatusOK, body)
}

// GetItem godoc
// @Summary      Get a single inventory item
// @Description  Returns one item by its UUID.
// @Tags         inventory
// @Produce      json
// @Param        id   path      string  true  "Item ID (UUID)"
// @Success      200  {object}  models.Item
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /inventory/{id} [get]
func GetItem(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	cacheKey := "item:" + id
	if cached, ok := itemCache.Get(cacheKey); ok {
		c.Data(http.StatusOK, "application/json; charset=utf-8", cached)
		return
	}

	var item models.Item
	if err := database.DB.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	if encoded, err := json.Marshal(item); err == nil {
		itemCache.Set(cacheKey, encoded)
	}

	c.JSON(http.StatusOK, item)
}

// CreateItem godoc
// @Summary      Create an inventory item
// @Description  Adds a new item to the inventory. Requires a bearer token.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        item  body      models.ItemInput  true  "New item"
// @Success      201   {object}  models.Item
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Router       /inventory [post]
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

	itemCache.Flush()
	c.JSON(http.StatusCreated, item)
}

// UpdateItem godoc
// @Summary      Update an inventory item
// @Description  Replaces an existing item's fields. Requires a bearer token.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string            true  "Item ID (UUID)"
// @Param        item  body      models.ItemInput  true  "Updated item"
// @Success      200   {object}  models.Item
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /inventory/{id} [put]
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

	itemCache.Flush()
	c.JSON(http.StatusOK, item)
}

// DeleteItem godoc
// @Summary      Delete an inventory item
// @Description  Removes an item from the inventory. Requires a bearer token.
// @Tags         inventory
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Item ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /inventory/{id} [delete]
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

	itemCache.Flush()
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

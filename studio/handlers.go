package studio

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handlers holds the API handler dependencies
type Handlers struct {
	DB     *gorm.DB
	Models []interface{}
	Schema *SchemaInfo
}

// NewHandlers creates a new Handlers instance
func NewHandlers(db *gorm.DB, models []interface{}) (*Handlers, error) {
	schema, err := IntrospectSchema(db, models)
	if err != nil {
		return nil, fmt.Errorf("creating handlers: %w", err)
	}

	return &Handlers{
		DB:     db,
		Models: models,
		Schema: schema,
	}, nil
}

// quoteIdent quotes an identifier (table/column name) to prevent SQL injection.
// The identifier is already validated against the schema, but quoting adds defense-in-depth.
func quoteIdent(dialect, name string) string {
	switch dialect {
	case "mysql":
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	default:
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
}

// qi is a shorthand for quoteIdent using the handler's dialect.
func (h *Handlers) qi(name string) string {
	return quoteIdent(h.DB.Dialector.Name(), name)
}

// getTableInfo returns the TableInfo for a given table name.
func (h *Handlers) getTableInfo(tableName string) *TableInfo {
	for i := range h.Schema.Tables {
		if strings.EqualFold(h.Schema.Tables[i].Name, tableName) {
			return &h.Schema.Tables[i]
		}
	}
	return nil
}

// hasSoftDelete returns true if the table has a deleted_at column (GORM soft delete).
func (h *Handlers) hasSoftDelete(tableName string) bool {
	ti := h.getTableInfo(tableName)
	if ti == nil {
		return false
	}
	for _, col := range ti.Columns {
		if strings.EqualFold(col.Name, "deleted_at") {
			return true
		}
	}
	return false
}

// GetSchema returns the full database schema
func (h *Handlers) GetSchema(c *gin.Context) {
	for i := range h.Schema.Tables {
		var count int64
		h.DB.Table(h.Schema.Tables[i].Name).Count(&count)
		h.Schema.Tables[i].RowCount = count
	}
	c.JSON(http.StatusOK, h.Schema)
}

// RefreshSchema re-introspects the database
func (h *Handlers) RefreshSchema(c *gin.Context) {
	schema, err := IntrospectSchema(h.DB, h.Models)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("refreshing schema: %s", err.Error())})
		return
	}
	h.Schema = schema
	c.JSON(http.StatusOK, h.Schema)
}

// GetRows returns paginated, filtered rows from a table
func (h *Handlers) GetRows(c *gin.Context) {
	tableName := c.Param("table")

	tableInfo := h.getTableInfo(tableName)
	if tableInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrTableNotFound{Table: tableName}).Error()})
		return
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	// Sorting
	sortBy := c.DefaultQuery("sort_by", "")
	sortOrder := c.DefaultQuery("sort_order", "asc")
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	// Build query
	query := h.DB.Table(tableName)

	// Soft delete: by default hide deleted rows unless show_deleted=true
	if h.hasSoftDelete(tableName) {
		showDeleted := c.DefaultQuery("show_deleted", "false")
		if showDeleted != "true" {
			query = query.Where(h.qi("deleted_at") + " IS NULL")
		}
	}

	// Filtering: ?filter_<column>=<value>
	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "filter_") {
			column := strings.TrimPrefix(key, "filter_")
			if isValidColumn(h.Schema, tableName, column) {
				value := values[0]
				if strings.Contains(value, "%") {
					query = query.Where(h.qi(column)+" LIKE ?", value)
				} else {
					query = query.Where(h.qi(column)+" = ?", value)
				}
			}
		}
	}

	// Search across all text columns
	search := c.Query("search")
	if search != "" {
		var conditions []string
		var args []interface{}
		for _, col := range tableInfo.Columns {
			colType := strings.ToLower(col.Type)
			if isTextType(colType) {
				conditions = append(conditions, h.qi(col.Name)+" LIKE ?")
				args = append(args, "%"+search+"%")
			}
		}
		if len(conditions) > 0 {
			query = query.Where(strings.Join(conditions, " OR "), args...)
		}
	}

	// Count total
	var total int64
	countQuery := query.Session(&gorm.Session{})
	countQuery.Count(&total)

	// Sorting (validated + quoted)
	if sortBy != "" && isValidColumn(h.Schema, tableName, sortBy) {
		query = query.Order(h.qi(sortBy) + " " + sortOrder)
	}

	// Execute
	var rows []map[string]interface{}
	result := query.Offset(offset).Limit(pageSize).Find(&rows)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rows":        rows,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"pages":       (total + int64(pageSize) - 1) / int64(pageSize),
		"soft_delete": h.hasSoftDelete(tableName),
	})
}

// GetRow returns a single row by primary key
func (h *Handlers) GetRow(c *gin.Context) {
	tableName := c.Param("table")
	id := c.Param("id")

	tableInfo := h.getTableInfo(tableName)
	if tableInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrTableNotFound{Table: tableName}).Error()})
		return
	}

	pks := getPrimaryKeys(h.Schema, tableName)
	if len(pks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": (&ErrNoPrimaryKey{Table: tableName}).Error()})
		return
	}

	query := h.DB.Table(tableName)
	query = applyCompositePK(query, h, pks, id)

	var row map[string]interface{}
	result := query.First(&row)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrRowNotFound{Table: tableName, ID: id}).Error()})
		return
	}

	c.JSON(http.StatusOK, row)
}

// CreateRow inserts a new row
func (h *Handlers) CreateRow(c *gin.Context) {
	tableName := c.Param("table")

	if h.getTableInfo(tableName) == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrTableNotFound{Table: tableName}).Error()})
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filtered := filterValidColumns(h.Schema, tableName, data)

	result := h.DB.Table(tableName).Create(filtered)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "created", "data": filtered})
}

// UpdateRow updates a row by primary key
func (h *Handlers) UpdateRow(c *gin.Context) {
	tableName := c.Param("table")
	id := c.Param("id")

	if h.getTableInfo(tableName) == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrTableNotFound{Table: tableName}).Error()})
		return
	}

	pks := getPrimaryKeys(h.Schema, tableName)
	if len(pks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": (&ErrNoPrimaryKey{Table: tableName}).Error()})
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Remove all primary keys from update data
	for _, pk := range pks {
		delete(data, pk)
	}
	filtered := filterValidColumns(h.Schema, tableName, data)

	query := h.DB.Table(tableName)
	query = applyCompositePK(query, h, pks, id)

	result := query.Updates(filtered)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrRowNotFound{Table: tableName, ID: id}).Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated", "rows_affected": result.RowsAffected})
}

// DeleteRow deletes a row by primary key
func (h *Handlers) DeleteRow(c *gin.Context) {
	tableName := c.Param("table")
	id := c.Param("id")

	if h.getTableInfo(tableName) == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrTableNotFound{Table: tableName}).Error()})
		return
	}

	pks := getPrimaryKeys(h.Schema, tableName)
	if len(pks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": (&ErrNoPrimaryKey{Table: tableName}).Error()})
		return
	}

	query := h.DB.Table(tableName)
	query = applyCompositePK(query, h, pks, id)

	result := query.Delete(nil)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrRowNotFound{Table: tableName, ID: id}).Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted", "rows_affected": result.RowsAffected})
}

// BulkDelete deletes multiple rows
func (h *Handlers) BulkDelete(c *gin.Context) {
	tableName := c.Param("table")

	if h.getTableInfo(tableName) == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrTableNotFound{Table: tableName}).Error()})
		return
	}

	pk := getPrimaryKey(h.Schema, tableName)
	if pk == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": (&ErrNoPrimaryKey{Table: tableName}).Error()})
		return
	}

	var body struct {
		IDs []interface{} `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.DB.Table(tableName).Where(h.qi(pk)+" IN ?", body.IDs).Delete(nil)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted", "rows_affected": result.RowsAffected})
}

// GetRelatedRows returns rows from a related table
func (h *Handlers) GetRelatedRows(c *gin.Context) {
	tableName := c.Param("table")
	id := c.Param("id")
	relName := c.Param("relation")

	if h.getTableInfo(tableName) == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrTableNotFound{Table: tableName}).Error()})
		return
	}

	// Find the relation
	var relation *RelationInfo
	for _, t := range h.Schema.Tables {
		if t.Name == tableName {
			for _, r := range t.Relations {
				if strings.EqualFold(r.Name, relName) {
					rel := r
					relation = &rel
					break
				}
			}
		}
	}

	if relation == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrRelationNotFound{Table: tableName, Relation: relName}).Error()})
		return
	}

	var rows []map[string]interface{}
	var result *gorm.DB

	switch relation.Type {
	case "has_one", "has_many":
		result = h.DB.Table(relation.Table).Where(h.qi(relation.ForeignKey)+" = ?", id).Find(&rows)
	case "belongs_to":
		pk := getPrimaryKey(h.Schema, tableName)
		var sourceRow map[string]interface{}
		h.DB.Table(tableName).Where(h.qi(pk)+" = ?", id).First(&sourceRow)
		if fkVal, ok := sourceRow[relation.ForeignKey]; ok {
			result = h.DB.Table(relation.Table).Where(h.qi(relation.ReferenceKey)+" = ?", fkVal).Find(&rows)
		}
	case "many_to_many":
		if relation.JoinTable != "" {
			pk := getPrimaryKey(h.Schema, tableName)
			refPK := getPrimaryKey(h.Schema, relation.Table)
			joinSQL := fmt.Sprintf("JOIN %s ON %s.%s = %s.%s",
				h.qi(relation.JoinTable),
				h.qi(relation.JoinTable), h.qi(relation.ForeignKey),
				h.qi(relation.Table), h.qi(refPK))
			whereSQL := fmt.Sprintf("%s.%s = ?", h.qi(relation.JoinTable), h.qi(pk))
			result = h.DB.Table(relation.Table).
				Joins(joinSQL).
				Where(whereSQL, id).
				Find(&rows)
		}
	}

	if result != nil && result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rows":     rows,
		"relation": relation,
		"total":    len(rows),
	})
}

// ExecuteSQL runs a raw SQL query
func (h *Handlers) ExecuteSQL(c *gin.Context) {
	var body struct {
		Query string `json:"query" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := strings.TrimSpace(body.Query)

	// Determine if it's a read or write query
	upperQuery := strings.ToUpper(query)
	isRead := strings.HasPrefix(upperQuery, "SELECT") ||
		strings.HasPrefix(upperQuery, "EXPLAIN") ||
		strings.HasPrefix(upperQuery, "PRAGMA") ||
		strings.HasPrefix(upperQuery, "SHOW") ||
		strings.HasPrefix(upperQuery, "DESCRIBE")

	if isRead {
		var rows []map[string]interface{}
		result := h.DB.Raw(query).Find(&rows)
		if result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
			return
		}

		var columns []string
		if len(rows) > 0 {
			for key := range rows[0] {
				columns = append(columns, key)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"rows":          rows,
			"total":         len(rows),
			"columns":       columns,
			"rows_affected": result.RowsAffected,
			"type":          "read",
		})
	} else {
		result := h.DB.Exec(query)
		if result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"rows_affected": result.RowsAffected,
			"message":       "query executed successfully",
			"type":          "write",
		})
	}
}

// ExportTable exports table data as CSV or JSON
func (h *Handlers) ExportTable(c *gin.Context) {
	tableName := c.Param("table")
	format := c.DefaultQuery("format", "json")

	tableInfo := h.getTableInfo(tableName)
	if tableInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": (&ErrTableNotFound{Table: tableName}).Error()})
		return
	}

	// Fetch all rows (with optional soft delete filtering)
	query := h.DB.Table(tableName)
	if h.hasSoftDelete(tableName) {
		showDeleted := c.DefaultQuery("show_deleted", "false")
		if showDeleted != "true" {
			query = query.Where(h.qi("deleted_at") + " IS NULL")
		}
	}

	var rows []map[string]interface{}
	result := query.Find(&rows)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", tableName))

		writer := csv.NewWriter(c.Writer)

		// Header row
		colNames := make([]string, len(tableInfo.Columns))
		for i, col := range tableInfo.Columns {
			colNames[i] = col.Name
		}
		writer.Write(colNames)

		// Data rows
		for _, row := range rows {
			record := make([]string, len(tableInfo.Columns))
			for i, col := range tableInfo.Columns {
				val := row[col.Name]
				if val == nil {
					record[i] = ""
				} else {
					record[i] = fmt.Sprintf("%v", val)
				}
			}
			writer.Write(record)
		}
		writer.Flush()

	case "json":
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", tableName))

		encoder := json.NewEncoder(c.Writer)
		encoder.SetIndent("", "  ")
		encoder.Encode(gin.H{
			"table": tableName,
			"total": len(rows),
			"rows":  rows,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format, use 'json' or 'csv'"})
	}
}

// GetDBStats returns database connection pool statistics
func (h *Handlers) GetDBStats(c *gin.Context) {
	sqlDB, err := h.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("getting db stats: %s", err.Error())})
		return
	}

	stats := sqlDB.Stats()
	c.JSON(http.StatusOK, gin.H{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration_ms":     stats.WaitDuration.Milliseconds(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	})
}

// Helper functions

func isValidColumn(schema *SchemaInfo, tableName, columnName string) bool {
	for _, t := range schema.Tables {
		if t.Name == tableName {
			for _, c := range t.Columns {
				if c.Name == columnName {
					return true
				}
			}
		}
	}
	return false
}

func getPrimaryKey(schema *SchemaInfo, tableName string) string {
	pks := getPrimaryKeys(schema, tableName)
	if len(pks) > 0 {
		return pks[0]
	}
	return "id" // fallback
}

func getPrimaryKeys(schema *SchemaInfo, tableName string) []string {
	for _, t := range schema.Tables {
		if t.Name == tableName {
			if len(t.PrimaryKeys) > 0 {
				return t.PrimaryKeys
			}
			var pks []string
			for _, c := range t.Columns {
				if c.IsPrimaryKey {
					pks = append(pks, c.Name)
				}
			}
			if len(pks) > 0 {
				return pks
			}
		}
	}
	return nil
}

// applyCompositePK builds a WHERE clause for composite primary keys.
// For single PKs: id is used directly.
// For composite PKs: id is expected as "val1,val2" matching the PK order.
func applyCompositePK(query *gorm.DB, h *Handlers, pks []string, id string) *gorm.DB {
	if len(pks) == 1 {
		return query.Where(h.qi(pks[0])+" = ?", id)
	}

	// Composite PK: split id by comma
	parts := strings.SplitN(id, ",", len(pks))
	for i, pk := range pks {
		if i < len(parts) {
			query = query.Where(h.qi(pk)+" = ?", parts[i])
		}
	}
	return query
}

func filterValidColumns(schema *SchemaInfo, tableName string, data map[string]interface{}) map[string]interface{} {
	filtered := make(map[string]interface{})
	for key, value := range data {
		if isValidColumn(schema, tableName, key) {
			filtered[key] = value
		}
	}
	return filtered
}

func isTextType(colType string) bool {
	textTypes := []string{"text", "varchar", "char", "string", "nvarchar", "ntext", "clob"}
	for _, t := range textTypes {
		if strings.Contains(colType, t) {
			return true
		}
	}
	return false
}

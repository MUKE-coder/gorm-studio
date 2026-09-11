package studio

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ImportData handles POST /api/import/data
func (h *Handlers) ImportData(c *gin.Context) {
	if h.ReadOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "import not allowed in read-only mode"})
		return
	}

	h.limitImportBody(c)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": fmt.Sprintf("import file exceeds the maximum of %d bytes", h.MaxImportBytes)})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		if isBodyTooLarge(err) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": fmt.Sprintf("import file exceeds the maximum of %d bytes", h.MaxImportBytes)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	tableName := c.PostForm("table")
	ext := strings.ToLower(filepath.Ext(header.Filename))
	dryRun := isDryRun(c)

	db, cancel := h.importDB(c)
	defer cancel()

	var rowsInserted int64
	var tablesAffected []string

	switch ext {
	case ".json":
		rowsInserted, tablesAffected, err = h.importDataJSON(db, content, tableName, dryRun)
	case ".csv":
		if tableName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "table parameter is required for CSV imports"})
			return
		}
		rowsInserted, err = h.importDataCSV(db, content, tableName, dryRun)
		tablesAffected = []string{tableName}
	case ".sql":
		rowsInserted, tablesAffected, err = h.importDataSQL(db, string(content), dryRun)
	case ".xlsx":
		if tableName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "table parameter is required for Excel imports"})
			return
		}
		rowsInserted, err = h.importDataExcel(db, content, tableName, dryRun)
		tablesAffected = []string{tableName}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format: " + ext + ". Use .json, .csv, .sql, or .xlsx"})
		return
	}

	if err != nil {
		h.audit(c, AuditEvent{Action: "import_data", Table: tableName, Success: false, Err: err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if dryRun {
		c.JSON(http.StatusOK, gin.H{
			"message":         "dry run: no changes applied",
			"dry_run":         true,
			"rows_to_insert":  rowsInserted,
			"tables_affected": tablesAffected,
		})
		return
	}

	// Refresh schema to update row counts
	schema, serr := IntrospectSchema(h.DB, h.Models)
	if serr == nil {
		h.Schema = schema
	}

	h.audit(c, AuditEvent{Action: "import_data", Tables: tablesAffected, Rows: rowsInserted, Success: true})
	c.JSON(http.StatusOK, gin.H{
		"message":         "data imported successfully",
		"rows_inserted":   rowsInserted,
		"tables_affected": tablesAffected,
	})
}

// isDryRun reports whether the request asked for a preview instead of applying
// the import. Accepts ?dry_run=true or a dry_run form field.
func isDryRun(c *gin.Context) bool {
	return c.Query("dry_run") == "true" || c.PostForm("dry_run") == "true"
}

func (h *Handlers) importDataJSON(db *gorm.DB, data []byte, tableName string, dryRun bool) (int64, []string, error) {
	// Try multi-table format: { "table_name": [ {row}, ... ], ... }
	var multiTable map[string][]map[string]interface{}
	if err := json.Unmarshal(data, &multiTable); err == nil && len(multiTable) > 0 {
		var totalRows int64
		var tables []string
		for tName, rows := range multiTable {
			if err := h.tableWritable(tName); err != nil {
				return 0, nil, err
			}
			for _, row := range rows {
				if h.rowLimited(totalRows) {
					return 0, nil, h.errImportTooManyRows()
				}
				if dryRun {
					totalRows++
					continue
				}
				filtered := filterValidColumns(h.Schema, tName, row)
				if err := db.Table(tName).Create(&filtered).Error; err != nil {
					continue
				}
				totalRows++
			}
			tables = append(tables, tName)
		}
		if len(tables) > 0 {
			return totalRows, tables, nil
		}
	}

	// Try single-table format: [ {row}, ... ]
	if tableName == "" {
		return 0, nil, fmt.Errorf("for single-table JSON arrays, the 'table' parameter is required")
	}
	if err := h.tableWritable(tableName); err != nil {
		return 0, nil, err
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return 0, nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	var count int64
	for _, row := range rows {
		if h.rowLimited(count) {
			return 0, nil, h.errImportTooManyRows()
		}
		if dryRun {
			count++
			continue
		}
		filtered := filterValidColumns(h.Schema, tableName, row)
		if err := db.Table(tableName).Create(&filtered).Error; err != nil {
			continue
		}
		count++
	}
	return count, []string{tableName}, nil
}

func (h *Handlers) importDataCSV(db *gorm.DB, data []byte, tableName string, dryRun bool) (int64, error) {
	if err := h.tableWritable(tableName); err != nil {
		return 0, err
	}

	reader := csv.NewReader(bytes.NewReader(data))
	headers, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("reading CSV headers: %w", err)
	}

	// Map header indices to valid column names
	type headerMapping struct {
		index int
		name  string
	}
	var validHeaders []headerMapping
	for i, h2 := range headers {
		name := strings.TrimSpace(h2)
		if isValidColumn(h.Schema, tableName, name) {
			validHeaders = append(validHeaders, headerMapping{index: i, name: name})
		}
	}

	if len(validHeaders) == 0 {
		return 0, fmt.Errorf("no valid columns found in CSV headers")
	}

	var count int64
	for {
		if h.rowLimited(count) {
			return 0, h.errImportTooManyRows()
		}
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		row := make(map[string]interface{})
		for _, hm := range validHeaders {
			if hm.index < len(record) {
				val := record[hm.index]
				if val != "" {
					row[hm.name] = val
				}
			}
		}

		if len(row) > 0 {
			if dryRun {
				count++
				continue
			}
			if err := db.Table(tableName).Create(&row).Error; err != nil {
				continue
			}
			count++
		}
	}
	return count, nil
}

func (h *Handlers) importDataSQL(db *gorm.DB, content string, dryRun bool) (int64, []string, error) {
	// Strip comments and split with quote/paren awareness so a value like
	// '(' or an embedded ';' can't smuggle a second statement past the
	// INSERT-only check below.
	rawStmts := splitStatements(removeComments(content))
	tablesSet := make(map[string]bool)

	// Validate every statement is an INSERT *before* executing any of them, so a
	// non-INSERT statement can never take effect (fail closed). Without this a
	// leading INSERT would commit before a later DELETE was rejected.
	var stmts []string
	for _, stmt := range rawStmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		upper := strings.ToUpper(stmt)
		if !strings.HasPrefix(upper, "INSERT") {
			fields := strings.Fields(upper)
			kw := "statement"
			if len(fields) > 0 {
				kw = fields[0]
			}
			return 0, nil, fmt.Errorf("only INSERT statements are allowed in SQL data imports; found %s", kw)
		}

		// Extract table name from INSERT INTO
		tableRe := strings.NewReplacer("`", "", "\"", "", "'", "")
		cleaned := tableRe.Replace(upper)
		parts := strings.Fields(cleaned)
		if len(parts) >= 3 && parts[0] == "INSERT" && parts[1] == "INTO" {
			tablesSet[strings.ToLower(parts[2])] = true
		}
		stmts = append(stmts, stmt)
	}

	if h.MaxImportRows >= 0 && len(stmts) > h.MaxImportRows {
		return 0, nil, h.errImportTooManyRows()
	}

	// Refuse the whole import if it targets a hidden or read-only table.
	for t := range tablesSet {
		if err := h.tableWritable(t); err != nil {
			return 0, nil, err
		}
	}

	var tables []string
	for t := range tablesSet {
		tables = append(tables, t)
	}

	if dryRun {
		return int64(len(stmts)), tables, nil
	}

	// Execute inside a transaction so a mid-batch failure rolls back cleanly.
	var count int64
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range stmts {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("executing INSERT: %w", err)
			}
			count++
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}

	return count, tables, nil
}

func (h *Handlers) importDataExcel(db *gorm.DB, fileBytes []byte, tableName string, dryRun bool) (int64, error) {
	if err := h.tableWritable(tableName); err != nil {
		return 0, err
	}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return 0, fmt.Errorf("opening Excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)

	// Stream rows rather than materializing the whole sheet, so a file whose
	// decompressed size dwarfs its upload size can't exhaust memory.
	it, err := f.Rows(sheetName)
	if err != nil {
		return 0, fmt.Errorf("reading Excel sheet: %w", err)
	}
	defer it.Close()

	if !it.Next() {
		return 0, fmt.Errorf("Excel file must have a header row and at least one data row")
	}
	headers, err := it.Columns()
	if err != nil {
		return 0, fmt.Errorf("reading Excel headers: %w", err)
	}

	type headerMapping struct {
		index int
		name  string
	}
	var validHeaders []headerMapping
	for i, h2 := range headers {
		name := strings.TrimSpace(h2)
		if isValidColumn(h.Schema, tableName, name) {
			validHeaders = append(validHeaders, headerMapping{index: i, name: name})
		}
	}

	if len(validHeaders) == 0 {
		return 0, fmt.Errorf("no valid columns found in Excel headers")
	}

	var count int64
	for it.Next() {
		if h.rowLimited(count) {
			return 0, h.errImportTooManyRows()
		}
		row, err := it.Columns()
		if err != nil {
			continue
		}
		data := make(map[string]interface{})
		for _, hm := range validHeaders {
			if hm.index < len(row) {
				val := row[hm.index]
				if val != "" {
					data[hm.name] = val
				}
			}
		}

		if len(data) > 0 {
			if dryRun {
				count++
				continue
			}
			if err := db.Table(tableName).Create(&data).Error; err != nil {
				continue
			}
			count++
		}
	}
	return count, nil
}

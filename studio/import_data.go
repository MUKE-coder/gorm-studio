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
)

// ImportData handles POST /api/import/data
func (h *Handlers) ImportData(c *gin.Context) {
	if h.ReadOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "import not allowed in read-only mode"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	tableName := c.PostForm("table")
	ext := strings.ToLower(filepath.Ext(header.Filename))

	var rowsInserted int64
	var tablesAffected []string

	switch ext {
	case ".json":
		rowsInserted, tablesAffected, err = h.importDataJSON(content, tableName)
	case ".csv":
		if tableName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "table parameter is required for CSV imports"})
			return
		}
		rowsInserted, err = h.importDataCSV(content, tableName)
		tablesAffected = []string{tableName}
	case ".sql":
		rowsInserted, tablesAffected, err = h.importDataSQL(string(content))
	case ".xlsx":
		if tableName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "table parameter is required for Excel imports"})
			return
		}
		rowsInserted, err = h.importDataExcel(content, tableName)
		tablesAffected = []string{tableName}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format: " + ext + ". Use .json, .csv, .sql, or .xlsx"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Refresh schema to update row counts
	schema, serr := IntrospectSchema(h.DB, h.Models)
	if serr == nil {
		h.Schema = schema
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "data imported successfully",
		"rows_inserted":   rowsInserted,
		"tables_affected": tablesAffected,
	})
}

func (h *Handlers) importDataJSON(data []byte, tableName string) (int64, []string, error) {
	// Try multi-table format: { "table_name": [ {row}, ... ], ... }
	var multiTable map[string][]map[string]interface{}
	if err := json.Unmarshal(data, &multiTable); err == nil && len(multiTable) > 0 {
		var totalRows int64
		var tables []string
		for tName, rows := range multiTable {
			if h.getTableInfo(tName) == nil {
				continue
			}
			for _, row := range rows {
				filtered := filterValidColumns(h.Schema, tName, row)
				if err := h.DB.Table(tName).Create(&filtered).Error; err != nil {
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
	if h.getTableInfo(tableName) == nil {
		return 0, nil, fmt.Errorf("table not found: %s", tableName)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return 0, nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	var count int64
	for _, row := range rows {
		filtered := filterValidColumns(h.Schema, tableName, row)
		if err := h.DB.Table(tableName).Create(&filtered).Error; err != nil {
			continue
		}
		count++
	}
	return count, []string{tableName}, nil
}

func (h *Handlers) importDataCSV(data []byte, tableName string) (int64, error) {
	if h.getTableInfo(tableName) == nil {
		return 0, fmt.Errorf("table not found: %s", tableName)
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
			if err := h.DB.Table(tableName).Create(&row).Error; err != nil {
				continue
			}
			count++
		}
	}
	return count, nil
}

func (h *Handlers) importDataSQL(content string) (int64, []string, error) {
	stmts := splitStatements(content)
	tablesSet := make(map[string]bool)
	var count int64

	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		upper := strings.ToUpper(stmt)

		// Only allow INSERT statements for safety
		if !strings.HasPrefix(upper, "INSERT") {
			continue
		}

		// Extract table name from INSERT INTO
		tableRe := strings.NewReplacer("`", "", "\"", "", "'", "")
		cleaned := tableRe.Replace(upper)
		parts := strings.Fields(cleaned)
		if len(parts) >= 3 && parts[0] == "INSERT" && parts[1] == "INTO" {
			tablesSet[strings.ToLower(parts[2])] = true
		}

		if err := h.DB.Exec(stmt).Error; err != nil {
			continue
		}
		count++
	}

	var tables []string
	for t := range tablesSet {
		tables = append(tables, t)
	}
	return count, tables, nil
}

func (h *Handlers) importDataExcel(fileBytes []byte, tableName string) (int64, error) {
	if h.getTableInfo(tableName) == nil {
		return 0, fmt.Errorf("table not found: %s", tableName)
	}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return 0, fmt.Errorf("opening Excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, fmt.Errorf("reading Excel sheet: %w", err)
	}

	if len(rows) < 2 {
		return 0, fmt.Errorf("Excel file must have a header row and at least one data row")
	}

	headers := rows[0]
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
	for _, row := range rows[1:] {
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
			if err := h.DB.Table(tableName).Create(&data).Error; err != nil {
				continue
			}
			count++
		}
	}
	return count, nil
}

package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"

	"github.com/jbeck018/howlerops/backend-go/pkg/storage"
)

// ExportFormat defines supported export formats
type ExportFormat string

const (
	FormatCSV   ExportFormat = "csv"
	FormatExcel ExportFormat = "xlsx"
	FormatPDF   ExportFormat = "pdf"
)

// ExportRequest represents a request to export report data
type ExportRequest struct {
	ReportID     string
	Format       ExportFormat
	ComponentIDs []string // Empty = all components
	FilterValues map[string]interface{}
	Options      ExportOptions
}

// ExportOptions contains export-specific configuration
type ExportOptions struct {
	IncludeCharts   bool
	IncludeMetadata bool
	PageSize        string // For PDF: "A4", "Letter"
	Orientation     string // "portrait" or "landscape"
	Title           string
	Author          string
}

// ExportResult contains the exported file data
type ExportResult struct {
	Filename string
	MimeType string
	Data     []byte
	Size     int64
}

// ChartImage represents a chart rendered as an image
type ChartImage struct {
	ComponentID string
	ImageData   []byte
	Width       int
	Height      int
}

// ReportComponentResult mirrors services.ReportComponentResult
type ReportComponentResult struct {
	ComponentID string
	Type        storage.ReportComponentType
	Columns     []string
	Rows        [][]interface{}
	RowCount    int64
	DurationMS  int64
	Content     string
	Metadata    map[string]any
	Error       string
	CacheHit    bool
	TotalRows   int64
	LimitedRows int
}

// Exporter handles report data export to various formats
type Exporter struct {
	logger *logrus.Logger
}

// NewExporter creates a new exporter instance
func NewExporter(logger *logrus.Logger) *Exporter {
	return &Exporter{
		logger: logger,
	}
}

// Export executes the export based on the request format
func (e *Exporter) Export(report *storage.Report, results []ReportComponentResult, options ExportOptions) (*ExportResult, error) {
	if report == nil {
		return nil, fmt.Errorf("report cannot be nil")
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to export")
	}

	// Set default title if not provided
	if options.Title == "" {
		options.Title = report.Name
	}

	// Determine export format from filename extension if present
	format := FormatCSV // default

	return e.ExportWithFormat(format, report, results, options)
}

// ExportWithFormat exports data in the specified format
func (e *Exporter) ExportWithFormat(format ExportFormat, report *storage.Report, results []ReportComponentResult, options ExportOptions) (*ExportResult, error) {
	switch format {
	case FormatCSV:
		return e.ExportCSV(report, results, options)
	case FormatExcel:
		return e.ExportExcel(report, results, options)
	case FormatPDF:
		return e.ExportPDF(report, results, options)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// ExportCSV exports results as CSV file(s)
func (e *Exporter) ExportCSV(report *storage.Report, results []ReportComponentResult, options ExportOptions) (*ExportResult, error) {
	var buf bytes.Buffer

	// Write UTF-8 BOM for Excel compatibility
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(&buf)

	// If multiple components, create separate CSV files in a single stream
	// For simplicity, we'll concatenate with component separators
	for idx, result := range results {
		if result.Error != "" {
			e.logger.WithFields(logrus.Fields{
				"component_id": result.ComponentID,
				"error":        result.Error,
			}).Warn("Skipping component with error in CSV export")
			continue
		}

		// Skip LLM components for CSV (they don't have tabular data)
		if result.Type == storage.ReportComponentLLM {
			continue
		}

		// Write component header if multiple components
		if len(results) > 1 {
			if idx > 0 {
				writer.Write([]string{}) // blank line separator
			}
			componentName := e.getComponentName(report, result.ComponentID)
			writer.Write([]string{fmt.Sprintf("# Component: %s", componentName)})
		}

		// Write column headers
		if len(result.Columns) > 0 {
			writer.Write(result.Columns)
		}

		// Write data rows
		for _, row := range result.Rows {
			stringRow := make([]string, len(row))
			for i, cell := range row {
				stringRow[i] = formatCellValue(cell)
			}
			if err := writer.Write(stringRow); err != nil {
				return nil, fmt.Errorf("failed to write CSV row: %w", err)
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	data := buf.Bytes()
	filename := sanitizeFilename(report.Name) + ".csv"

	return &ExportResult{
		Filename: filename,
		MimeType: "text/csv; charset=utf-8",
		Data:     data,
		Size:     int64(len(data)),
	}, nil
}

// ExportExcel exports results as Excel XLSX file
func (e *Exporter) ExportExcel(report *storage.Report, results []ReportComponentResult, options ExportOptions) (*ExportResult, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Delete default Sheet1
	f.DeleteSheet("Sheet1")

	sheetIndex := 1
	for _, result := range results {
		if result.Error != "" {
			e.logger.WithFields(logrus.Fields{
				"component_id": result.ComponentID,
				"error":        result.Error,
			}).Warn("Skipping component with error in Excel export")
			continue
		}

		// Skip LLM components for Excel (they don't have tabular data)
		if result.Type == storage.ReportComponentLLM {
			continue
		}

		componentName := e.getComponentName(report, result.ComponentID)
		sheetName := sanitizeSheetName(componentName, sheetIndex)

		// Create sheet
		index, err := f.NewSheet(sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
		}

		// Set as active sheet for first component
		if sheetIndex == 1 {
			f.SetActiveSheet(index)
		}

		// Write headers
		if len(result.Columns) > 0 {
			for colIdx, colName := range result.Columns {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
				f.SetCellValue(sheetName, cell, colName)
			}

			// Format header row (bold)
			headerStyle, _ := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true},
				Fill: excelize.Fill{Type: "pattern", Color: []string{"#F0F0F0"}, Pattern: 1},
			})
			endCol, _ := excelize.CoordinatesToCellName(len(result.Columns), 1)
			f.SetCellStyle(sheetName, "A1", endCol, headerStyle)

			// Freeze header row
			f.SetPanes(sheetName, &excelize.Panes{
				Freeze:      true,
				XSplit:      0,
				YSplit:      1,
				TopLeftCell: "A2",
			})
		}

		// Write data rows
		for rowIdx, row := range result.Rows {
			for colIdx, cell := range row {
				cellName, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2) // +2 because headers are row 1
				f.SetCellValue(sheetName, cellName, cell)
			}
		}

		// Auto-size columns (approximate)
		for colIdx := range result.Columns {
			col, _ := excelize.ColumnNumberToName(colIdx + 1)
			f.SetColWidth(sheetName, col, col, 15) // Default width
		}

		sheetIndex++
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	data := buf.Bytes()
	filename := sanitizeFilename(report.Name) + ".xlsx"

	return &ExportResult{
		Filename: filename,
		MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Data:     data,
		Size:     int64(len(data)),
	}, nil
}

// ExportPDF exports results as PDF file
func (e *Exporter) ExportPDF(report *storage.Report, results []ReportComponentResult, options ExportOptions) (*ExportResult, error) {
	// Set defaults
	if options.PageSize == "" {
		options.PageSize = "Letter"
	}
	if options.Orientation == "" {
		options.Orientation = "P" // Portrait
	} else if options.Orientation == "landscape" {
		options.Orientation = "L"
	} else {
		options.Orientation = "P"
	}

	pdf := gofpdf.New(options.Orientation, "mm", options.PageSize, "")
	pdf.SetAutoPageBreak(true, 15)

	// Title page
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 24)
	pdf.CellFormat(0, 20, options.Title, "", 1, "C", false, 0, "")

	// Metadata
	pdf.SetFont("Arial", "", 12)
	if options.Author != "" {
		pdf.CellFormat(0, 10, fmt.Sprintf("Author: %s", options.Author), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, 10, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")), "", 1, "L", false, 0, "")

	if report.Description != "" {
		pdf.Ln(5)
		pdf.SetFont("Arial", "", 10)
		pdf.MultiCell(0, 5, report.Description, "", "L", false)
	}

	// Add components
	for _, result := range results {
		if result.Error != "" {
			e.logger.WithFields(logrus.Fields{
				"component_id": result.ComponentID,
				"error":        result.Error,
			}).Warn("Skipping component with error in PDF export")
			continue
		}

		pdf.AddPage()

		componentName := e.getComponentName(report, result.ComponentID)

		// Component title
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(0, 10, componentName, "", 1, "L", false, 0, "")
		pdf.Ln(5)

		// Handle different component types
		if result.Type == storage.ReportComponentLLM {
			// LLM content as text
			pdf.SetFont("Arial", "", 10)
			pdf.MultiCell(0, 5, result.Content, "", "L", false)
		} else {
			// Tabular data
			if len(result.Columns) > 0 && len(result.Rows) > 0 {
				e.renderTablePDF(pdf, result.Columns, result.Rows)
			}
		}
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	data := buf.Bytes()
	filename := sanitizeFilename(report.Name) + ".pdf"

	return &ExportResult{
		Filename: filename,
		MimeType: "application/pdf",
		Data:     data,
		Size:     int64(len(data)),
	}, nil
}

// renderTablePDF renders a table in the PDF
func (e *Exporter) renderTablePDF(pdf *gofpdf.Fpdf, columns []string, rows [][]interface{}) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(240, 240, 240)

	// Calculate column widths (simple equal distribution)
	pageWidth, _ := pdf.GetPageSize()
	leftMargin, _, rightMargin, _ := pdf.GetMargins()
	usableWidth := pageWidth - leftMargin - rightMargin
	colWidth := usableWidth / float64(len(columns))

	// Limit column width for readability
	if colWidth > 50 {
		colWidth = 50
	}
	if colWidth < 15 {
		colWidth = 15
	}

	// Headers
	for _, col := range columns {
		pdf.CellFormat(colWidth, 7, truncateString(col, 20), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Data rows (alternating colors)
	pdf.SetFont("Arial", "", 8)
	for idx, row := range rows {
		if idx%2 == 0 {
			pdf.SetFillColor(255, 255, 255)
		} else {
			pdf.SetFillColor(245, 245, 245)
		}

		for _, cell := range row {
			cellStr := formatCellValue(cell)
			pdf.CellFormat(colWidth, 6, truncateString(cellStr, 25), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)

		// Limit rows to prevent excessive PDF size
		if idx >= 100 {
			pdf.SetFont("Arial", "I", 8)
			pdf.CellFormat(0, 6, fmt.Sprintf("... and %d more rows", len(rows)-100), "", 1, "L", false, 0, "")
			break
		}
	}
}

// Helper functions

func (e *Exporter) getComponentName(report *storage.Report, componentID string) string {
	for _, comp := range report.Definition.Components {
		if comp.ID == componentID {
			if comp.Title != "" {
				return comp.Title
			}
			return componentID
		}
	}
	return componentID
}

func formatCellValue(cell interface{}) string {
	if cell == nil {
		return ""
	}

	switch v := cell.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func sanitizeFilename(name string) string {
	// Replace invalid filename characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Limit length
	if len(result) > 200 {
		result = result[:200]
	}
	return result
}

func sanitizeSheetName(name string, index int) string {
	// Excel sheet names have restrictions
	invalid := []string{":", "\\", "/", "?", "*", "[", "]"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Limit to 31 characters (Excel limit)
	if len(result) > 31 {
		result = result[:28] + fmt.Sprintf("%03d", index)
	}
	if result == "" {
		result = fmt.Sprintf("Sheet%d", index)
	}
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

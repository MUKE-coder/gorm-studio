package studio

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"math"

	"github.com/fogleman/gg"
	"github.com/go-pdf/fpdf"
)

const (
	erdTableWidth  = 240.0
	erdRowHeight   = 26.0
	erdHeaderHeight = 40.0
	erdPadding     = 50.0
	erdGapX        = 100.0
	erdGapY        = 60.0
	erdFontSize    = 13.0
)

// RenderERDPNG renders the schema as an ERD diagram PNG.
func RenderERDPNG(schema *SchemaInfo, w io.Writer) error {
	if len(schema.Tables) == 0 {
		dc := gg.NewContext(400, 200)
		dc.SetRGB(1, 1, 1)
		dc.Clear()
		dc.SetRGB(0.5, 0.5, 0.5)
		dc.DrawStringAnchored("No tables in schema", 200, 100, 0.5, 0.5)
		return dc.EncodePNG(w)
	}

	positions, canvasW, canvasH := layoutERDTables(schema.Tables)
	dc := gg.NewContext(canvasW, canvasH)

	// Background
	dc.SetRGB(0.97, 0.97, 0.98)
	dc.Clear()

	// Draw relationship lines first (behind tables)
	drawERDRelations(dc, schema, positions)

	// Draw tables
	for _, table := range schema.Tables {
		pos := positions[table.Name]
		drawERDTable(dc, table, float64(pos.X), float64(pos.Y))
	}

	return dc.EncodePNG(w)
}

// RenderERDPDF renders the schema as a PDF with the ERD diagram embedded.
func RenderERDPDF(schema *SchemaInfo, w io.Writer) error {
	// Render PNG first
	var buf bytes.Buffer
	if err := RenderERDPNG(schema, &buf); err != nil {
		return err
	}

	// Decode PNG to get dimensions
	img, err := png.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	imgW := float64(bounds.Dx())
	imgH := float64(bounds.Dy())

	// Create PDF in landscape if wider than tall
	orientation := "P"
	if imgW > imgH {
		orientation = "L"
	}
	pdf := fpdf.New(orientation, "mm", "A3", "")
	pdf.SetTitle("Database ERD - GORM Studio", true)
	pdf.AddPage()

	pageW, pageH := pdf.GetPageSize()
	margin := 10.0
	availW := pageW - 2*margin
	availH := pageH - 2*margin

	// Scale to fit
	scaleW := availW / (imgW * 0.264583) // px to mm at 96 DPI
	scaleH := availH / (imgH * 0.264583)
	scale := math.Min(scaleW, scaleH)
	if scale > 1 {
		scale = 1
	}

	finalW := imgW * 0.264583 * scale
	finalH := imgH * 0.264583 * scale

	// Register image
	opt := fpdf.ImageOptions{ImageType: "PNG"}
	pdf.RegisterImageOptionsReader("erd", opt, bytes.NewReader(buf.Bytes()))
	pdf.ImageOptions("erd", margin, margin, finalW, finalH, false, opt, 0, "")

	return pdf.Output(w)
}

// layoutERDTables calculates grid positions for each table.
func layoutERDTables(tables []TableInfo) (map[string]image.Point, int, int) {
	positions := make(map[string]image.Point)
	n := len(tables)
	if n == 0 {
		return positions, 400, 200
	}

	cols := int(math.Ceil(math.Sqrt(float64(n))))
	if cols < 2 {
		cols = 2
	}

	// Calculate max table height per row
	rows := (n + cols - 1) / cols
	rowMaxHeights := make([]float64, rows)
	for i, table := range tables {
		row := i / cols
		h := erdHeaderHeight + float64(len(table.Columns))*erdRowHeight + 16
		if h > rowMaxHeights[row] {
			rowMaxHeights[row] = h
		}
	}

	for i, table := range tables {
		col := i % cols
		row := i / cols

		x := erdPadding + float64(col)*(erdTableWidth+erdGapX)
		y := erdPadding
		for r := 0; r < row; r++ {
			y += rowMaxHeights[r] + erdGapY
		}

		positions[table.Name] = image.Point{X: int(x), Y: int(y)}
		_ = table
	}

	// Calculate canvas size
	canvasW := int(erdPadding*2 + float64(cols)*(erdTableWidth+erdGapX) - erdGapX)
	totalH := erdPadding * 2
	for _, rh := range rowMaxHeights {
		totalH += rh + erdGapY
	}
	canvasH := int(totalH - erdGapY)

	return positions, canvasW, canvasH
}

// drawERDTable draws a single table box.
func drawERDTable(dc *gg.Context, table TableInfo, x, y float64) {
	totalHeight := erdHeaderHeight + float64(len(table.Columns))*erdRowHeight + 16

	// Shadow
	dc.SetRGBA(0, 0, 0, 0.08)
	drawRoundedRect(dc, x+3, y+3, erdTableWidth, totalHeight, 8)
	dc.Fill()

	// Background
	dc.SetRGB(1, 1, 1)
	drawRoundedRect(dc, x, y, erdTableWidth, totalHeight, 8)
	dc.Fill()

	// Border
	dc.SetRGB(0.82, 0.82, 0.87)
	dc.SetLineWidth(1)
	drawRoundedRect(dc, x, y, erdTableWidth, totalHeight, 8)
	dc.Stroke()

	// Header background (accent purple)
	dc.SetRGB(0.424, 0.361, 0.906) // #6c5ce7
	drawRoundedRectTop(dc, x, y, erdTableWidth, erdHeaderHeight, 8)
	dc.Fill()

	// Header line
	dc.SetRGB(0.82, 0.82, 0.87)
	dc.DrawLine(x, y+erdHeaderHeight, x+erdTableWidth, y+erdHeaderHeight)
	dc.Stroke()

	// Table name
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(table.Name, x+erdTableWidth/2, y+erdHeaderHeight/2, 0.5, 0.5)

	// Columns
	for i, col := range table.Columns {
		rowY := y + erdHeaderHeight + float64(i)*erdRowHeight + erdRowHeight/2 + 8

		// Alternating row background
		if i%2 == 1 {
			dc.SetRGBA(0, 0, 0, 0.02)
			dc.DrawRectangle(x+1, y+erdHeaderHeight+float64(i)*erdRowHeight+8, erdTableWidth-2, erdRowHeight)
			dc.Fill()
		}

		// PK/FK indicator
		if col.IsPrimaryKey {
			dc.SetRGB(0.91, 0.73, 0.25) // gold
			dc.DrawStringAnchored("PK", x+16, rowY, 0.5, 0.5)
		} else if col.IsForeignKey {
			dc.SetRGB(0.27, 0.55, 0.96) // blue
			dc.DrawStringAnchored("FK", x+16, rowY, 0.5, 0.5)
		}

		// Column name
		dc.SetRGB(0.15, 0.15, 0.2)
		dc.DrawStringAnchored(col.Name, x+36, rowY, 0, 0.5)

		// Column type (right-aligned)
		colType := col.Type
		if len(colType) > 16 {
			colType = colType[:16]
		}
		dc.SetRGB(0.5, 0.5, 0.6)
		dc.DrawStringAnchored(colType, x+erdTableWidth-12, rowY, 1, 0.5)
	}
}

// drawERDRelations draws relationship lines between related tables.
func drawERDRelations(dc *gg.Context, schema *SchemaInfo, positions map[string]image.Point) {
	dc.SetLineWidth(1.5)

	for _, table := range schema.Tables {
		for _, col := range table.Columns {
			if !col.IsForeignKey || col.ForeignTable == "" {
				continue
			}
			fromPos, ok1 := positions[table.Name]
			toPos, ok2 := positions[col.ForeignTable]
			if !ok1 || !ok2 {
				continue
			}

			// Calculate connection points
			fromTableH := erdHeaderHeight + float64(len(table.Columns))*erdRowHeight + 16
			// Find column index for positioning
			colIdx := 0
			for i, c := range table.Columns {
				if c.Name == col.Name {
					colIdx = i
					break
				}
			}
			fromY := float64(fromPos.Y) + erdHeaderHeight + float64(colIdx)*erdRowHeight + erdRowHeight/2 + 8
			toY := float64(toPos.Y) + erdHeaderHeight/2

			_ = fromTableH

			// Determine which side to connect
			fromRight := float64(fromPos.X) + erdTableWidth
			toLeft := float64(toPos.X)

			var startX, endX float64
			if fromRight < toLeft {
				startX = fromRight
				endX = toLeft
			} else {
				startX = float64(fromPos.X)
				endX = float64(toPos.X) + erdTableWidth
			}

			// Draw curved line
			dc.SetRGBA(0.424, 0.361, 0.906, 0.5) // semi-transparent accent
			dc.SetDash(6, 3)
			midX := (startX + endX) / 2
			dc.MoveTo(startX, fromY)
			dc.CubicTo(midX, fromY, midX, toY, endX, toY)
			dc.Stroke()
			dc.SetDash() // reset dash

			// Draw small circle at endpoints
			dc.SetRGBA(0.424, 0.361, 0.906, 0.8)
			dc.DrawCircle(startX, fromY, 3)
			dc.Fill()
			dc.DrawCircle(endX, toY, 3)
			dc.Fill()
		}
	}
}

// drawRoundedRect draws a rounded rectangle path.
func drawRoundedRect(dc *gg.Context, x, y, w, h, r float64) {
	dc.NewSubPath()
	dc.DrawArc(x+r, y+r, r, gg.Radians(180), gg.Radians(270))
	dc.LineTo(x+w-r, y)
	dc.DrawArc(x+w-r, y+r, r, gg.Radians(270), gg.Radians(360))
	dc.LineTo(x+w, y+h-r)
	dc.DrawArc(x+w-r, y+h-r, r, gg.Radians(0), gg.Radians(90))
	dc.LineTo(x+r, y+h)
	dc.DrawArc(x+r, y+h-r, r, gg.Radians(90), gg.Radians(180))
	dc.ClosePath()
}

// drawRoundedRectTop draws a rectangle with rounded top corners only.
func drawRoundedRectTop(dc *gg.Context, x, y, w, h, r float64) {
	dc.NewSubPath()
	dc.DrawArc(x+r, y+r, r, gg.Radians(180), gg.Radians(270))
	dc.LineTo(x+w-r, y)
	dc.DrawArc(x+w-r, y+r, r, gg.Radians(270), gg.Radians(360))
	dc.LineTo(x+w, y+h)
	dc.LineTo(x, y+h)
	dc.ClosePath()
}

// helper for SchemaInfo JSON/YAML serialization
type schemaExport struct {
	Tables   []tableExport `json:"tables" yaml:"tables"`
	Database string        `json:"database" yaml:"database"`
	Driver   string        `json:"driver" yaml:"driver"`
}

type tableExport struct {
	Name        string        `json:"name" yaml:"name"`
	Columns     []ColumnInfo  `json:"columns" yaml:"columns"`
	Relations   []RelationInfo `json:"relations,omitempty" yaml:"relations,omitempty"`
	PrimaryKeys []string      `json:"primary_keys,omitempty" yaml:"primary_keys,omitempty"`
}

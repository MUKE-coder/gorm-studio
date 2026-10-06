package studio

import (
	"archive/zip"
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// craftNegativeSharedStringXLSX builds a structurally valid .xlsx and then
// patches a worksheet cell to reference shared-string index -1, the condition
// behind GO-2026-6452 (a panic deep in excelize's row reader).
func craftNegativeSharedStringXLSX(t *testing.T) []byte {
	t.Helper()

	f := excelize.NewFile()
	// Two shared-string rows so sharedStrings.xml exists and cells use t="s".
	_ = f.SetCellValue("Sheet1", "A1", "header")
	_ = f.SetCellValue("Sheet1", "A2", "value")
	var orig bytes.Buffer
	if err := f.Write(&orig); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(orig.Bytes()), int64(orig.Len()))
	if err != nil {
		t.Fatalf("open xlsx zip: %v", err)
	}

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	patched := false
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			t.Fatalf("open %s: %v", zf.Name, err)
		}
		var data bytes.Buffer
		if _, err := data.ReadFrom(rc); err != nil {
			t.Fatalf("read %s: %v", zf.Name, err)
		}
		rc.Close()

		content := data.Bytes()
		if strings.Contains(zf.Name, "worksheets/sheet1.xml") {
			// Point a shared-string cell at index -1.
			s := strings.Replace(string(content), `t="s"><v>0</v>`, `t="s"><v>-1</v>`, 1)
			content = []byte(s)
			patched = true
		}

		w, err := zw.Create(zf.Name)
		if err != nil {
			t.Fatalf("create %s: %v", zf.Name, err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatalf("write %s: %v", zf.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if !patched {
		t.Skip("could not locate the shared-string cell to patch; excelize layout changed")
	}
	return out.Bytes()
}

// Regression for #8: a crafted .xlsx that would panic inside excelize must be
// contained and surface as a clean client error, never a 500 or a process crash.
// The router here has no gin.Recovery middleware, so an unrecovered panic would
// crash the test outright.
func TestImportExcel_MalformedDoesNotPanic(t *testing.T) {
	router, db := setupLimitRouter(t, Config{})
	before := rowCount(t, db, "test_users")

	payload := craftNegativeSharedStringXLSX(t)

	w := doMultipartRequest(router, "/studio/api/import/data", "file", "evil.xlsx",
		string(payload), map[string]string{"table": "test_users"})

	if w.Code == http.StatusInternalServerError {
		t.Fatalf("malformed Excel should not yield a 500, got: %s", w.Body.String())
	}
	if got := rowCount(t, db, "test_users"); got != before {
		t.Errorf("malformed Excel import changed row count: %d -> %d", before, got)
	}
}

package scanner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClamAV_EICAR(t *testing.T) {
	sc := NewClamAVScanner("localhost", "3310", 10*time.Second)

	eicar := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
	res, err := sc.Scan(context.Background(), strings.NewReader(eicar))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	t.Logf("Raw EICAR Result: clean=%v, virus=%s", res.IsClean, res.VirusName)

	pdf := "%PDF-1.4\n1 0 obj\n<< /Length 68 >>\nstream\n" + eicar + "\nendstream\nendobj\n%%EOF"
	resPdf, errPdf := sc.Scan(context.Background(), strings.NewReader(pdf))
	if errPdf != nil {
		t.Fatalf("Scan PDF error: %v", errPdf)
	}
	t.Logf("PDF with EICAR Result: clean=%v, virus=%s", resPdf.IsClean, resPdf.VirusName)
}

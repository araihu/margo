package deck

import "testing"

func TestParsePDFMediaBoxes(t *testing.T) {
	pdf := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Page /MediaBox [0 0 960 540] >>\nendobj\n2 0 obj\n<< /MediaBox [0.0 0 960.5 540.25] >>\nendobj")
	boxes, err := ParsePDFMediaBoxes(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if len(boxes) != 2 {
		t.Fatalf("boxes = %#v", boxes)
	}
	if boxes[0].Index != 0 || boxes[0].RightMicrometers != 338667 || boxes[0].TopMicrometers != 190500 {
		t.Fatalf("first box = %#v", boxes[0])
	}
	if boxes[1].Index != 1 || boxes[1].RightMicrometers != 338843 || boxes[1].TopMicrometers != 190588 {
		t.Fatalf("second box = %#v", boxes[1])
	}
}

func TestParsePDFMediaBoxesRejectsMissingBox(t *testing.T) {
	if _, err := ParsePDFMediaBoxes([]byte("%PDF-1.4")); err == nil {
		t.Fatal("expected missing media box error")
	}
}

func TestPDFArtifactEvidenceUsesFourEdgesAndCanonicalDigest(t *testing.T) {
	report, err := BuildPDFArtifactReport(1280, 720, 1, []PDFMediaBoxMicrometers{{Index: 0, RightMicrometers: 338667, TopMicrometers: 190500}})
	if err != nil {
		t.Fatal(err)
	}
	if report.EvidenceBytes != 247 || report.EvidenceSHA256 != "f503f15c13c8e0c405731c71bc65c6b0d5f4f80403076c6536f868a8bec55867" {
		t.Fatalf("report evidence = %#v", report)
	}
	if !report.Valid {
		t.Fatal("matching PDF evidence is not valid")
	}
}

func TestPDFArtifactEvidenceRejectsPageCountAndEdgeMismatch(t *testing.T) {
	report, err := BuildPDFArtifactReport(1280, 720, 2, []PDFMediaBoxMicrometers{{Index: 0, RightMicrometers: 338800, TopMicrometers: 190500}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Valid {
		t.Fatal("mismatched page evidence unexpectedly valid")
	}
}

func TestPDFArtifactEvidenceToleratesChromiumMediaBoxRounding(t *testing.T) {
	report, err := BuildPDFArtifactReport(1440, 900, 1, []PDFMediaBoxMicrometers{{Index: 0, RightMicrometers: 381000, TopMicrometers: 238167}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Valid {
		t.Fatal("Chromium media box rounding unexpectedly invalid")
	}

	report, err = BuildPDFArtifactReport(1440, 900, 1, []PDFMediaBoxMicrometers{{Index: 0, RightMicrometers: 381000, TopMicrometers: 238300}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Valid {
		t.Fatal("material media box mismatch unexpectedly valid")
	}
}

func TestComposedPDFArtifactEvidenceIncludesCompositionIdentity(t *testing.T) {
	spec, err := ResolveComposition("hero")
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildPDFArtifactReportWithComposition(1280, 720, 1, []PDFMediaBoxMicrometers{{Index: 0, RightMicrometers: 338667, TopMicrometers: 190500}}, []CompositionSpec{spec})
	if err != nil {
		t.Fatal(err)
	}
	if report.CompositionCatalogVersion != CompositionCatalogVersion {
		t.Fatalf("catalog version = %q", report.CompositionCatalogVersion)
	}
	if len(report.Compositions) != 1 || report.Compositions[0].Name != "hero" || report.Compositions[0].Variant != "hero" {
		t.Fatalf("composition identity = %#v", report.Compositions)
	}
}

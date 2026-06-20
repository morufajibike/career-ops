package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseApplicationsUsesTrackerNumberColumn(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	applications := `# Applications Tracker

| # | Date | Company | Role | Score | Status | PDF | Report | Notes |
|---|------|---------|------|-------|--------|-----|--------|-------|
| 140 | 2026-04-16 | Arize AI | AI Engineer, Instrumentation | 4.7/5 | Evaluated | ✅ | [140](reports/140-arize-ai-engineer-instrumentation-2026-04-16.md) | Strong fit |
| 143 | 2026-04-16 | Arize AI | AI Sales Engineer, US | 4.1/5 | Evaluated | ❌ | [143](reports/143-arize-ai-sales-engineer-us-2026-04-16.md) | Good fit |
`

	applicationsPath := filepath.Join(dataDir, "applications.md")
	if err := os.WriteFile(applicationsPath, []byte(applications), 0o644); err != nil {
		t.Fatalf("failed to write applications tracker: %v", err)
	}

	apps := ParseApplications(tempDir)
	if len(apps) != 2 {
		t.Fatalf("expected 2 parsed applications, got %d", len(apps))
	}

	if apps[0].Number != 140 {
		t.Fatalf("expected first application number to be 140, got %d", apps[0].Number)
	}
	if apps[1].Number != 143 {
		t.Fatalf("expected second application number to be 143, got %d", apps[1].Number)
	}
	if apps[0].ReportNumber != "140" || apps[1].ReportNumber != "143" {
		t.Fatalf("expected report numbers to stay aligned with tracker IDs, got %q and %q", apps[0].ReportNumber, apps[1].ReportNumber)
	}
}

// TestParseApplicationsStrategy3ZeroPaddedReportNum exercises the strategy-3 URL
// enrichment path: a short batch report number (e.g. "42" in batch-state.tsv) must
// resolve the tracker's 3-digit zero-padded report number ("042"). The report file
// deliberately omits **URL:** and **Batch ID:** so strategies 1 and 2 do not fire.
func TestParseApplicationsStrategy3ZeroPaddedReportNum(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	reportsDir := filepath.Join(tempDir, "reports")
	batchDir := filepath.Join(tempDir, "batch")
	for _, dir := range []string{dataDir, reportsDir, batchDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	const wantURL = "https://example.com/jobs/data-engineer"

	// Tracker uses the 3-digit zero-padded report number (042).
	applications := `# Applications Tracker

| # | Date | Company | Role | Score | Status | PDF | Report | Notes |
|---|------|---------|------|-------|--------|-----|--------|-------|
| 42 | 2026-04-16 | Acme | Data Engineer | 4.5/5 | Evaluated | ✅ | [042](reports/042-acme-data-engineer-2026-04-16.md) | Strong fit |
`
	if err := os.WriteFile(filepath.Join(dataDir, "applications.md"), []byte(applications), 0o644); err != nil {
		t.Fatalf("failed to write applications tracker: %v", err)
	}

	// Report has no **URL:** and no **Batch ID:**, forcing the strategy-3 lookup.
	report := `# 042 Acme - Data Engineer

**Score:** 4.5/5

Some evaluation body without a URL or batch id header.
`
	if err := os.WriteFile(filepath.Join(reportsDir, "042-acme-data-engineer-2026-04-16.md"), []byte(report), 0o644); err != nil {
		t.Fatalf("failed to write report: %v", err)
	}

	// batch-input.tsv: id \t url \t source \t notes
	batchInput := "id\turl\tsource\tnotes\n" +
		"7\thttps://source.example.com/raw\tscan\tData Engineer @ Acme | 95% | " + wantURL + "\n"
	if err := os.WriteFile(filepath.Join(batchDir, "batch-input.tsv"), []byte(batchInput), 0o644); err != nil {
		t.Fatalf("failed to write batch-input: %v", err)
	}

	// batch-state.tsv: id \t url \t status \t ? \t ? \t report_num.
	// report_num is the short "42"; the zero-padded "042" key must be indexed too.
	batchState := "id\turl\tstatus\tworker\tts\treport_num\n" +
		"7\thttps://source.example.com/raw\tcompleted\tw1\t2026-04-16\t42\n"
	if err := os.WriteFile(filepath.Join(batchDir, "batch-state.tsv"), []byte(batchState), 0o644); err != nil {
		t.Fatalf("failed to write batch-state: %v", err)
	}

	apps := ParseApplications(tempDir)
	if len(apps) != 1 {
		t.Fatalf("expected 1 parsed application, got %d", len(apps))
	}
	if apps[0].ReportNumber != "042" {
		t.Fatalf("expected report number 042, got %q", apps[0].ReportNumber)
	}
	if apps[0].JobURL != wantURL {
		t.Fatalf("expected strategy-3 to resolve job URL %q via zero-padded lookup, got %q", wantURL, apps[0].JobURL)
	}
}

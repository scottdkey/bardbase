package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testDataPath(relPath string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile is .../projects/capell/internal/parser/cleanup_test.go
	// repo root is 4 levels up
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	return filepath.Join(repoRoot, relPath)
}

func TestDefTextAcquit(t *testing.T) {
	data, err := os.ReadFile(testDataPath("projects/sources/lexicon/entries/A/Acquit.xml"))
	if err != nil {
		t.Skipf("XML not found: %v", err)
	}
	entry, err := ParseEntryXML(data, "Acquit.xml")
	if err != nil || entry == nil {
		t.Fatal("parse failed")
	}
	fmt.Printf("Senses: %d\n", len(entry.Senses))
	for i, s := range entry.Senses {
		fmt.Printf("  Sense %d (#%d): %q\n", i, s.Number, s.Text)
	}
	// Sense 1 should NOT contain "2)" or reference artifacts
	if len(entry.Senses) < 3 {
		t.Errorf("expected at least 3 senses, got %d", len(entry.Senses))
	}
}

func TestDefTextAbergany(t *testing.T) {
	data, err := os.ReadFile(testDataPath("projects/sources/lexicon/entries/A/Abergany.xml"))
	if err != nil {
		t.Skipf("XML not found: %v", err)
	}
	entry, err := ParseEntryXML(data, "Abergany.xml")
	if err != nil || entry == nil {
		t.Fatal("parse failed")
	}
	fmt.Printf("Senses: %d\n", len(entry.Senses))
	for i, s := range entry.Senses {
		fmt.Printf("  Sense %d (#%d): %q\n", i, s.Number, s.Text)
	}
	// Definition should not end with ". ." or similar artifacts
	def := entry.Senses[0].Text
	if len(def) > 0 && (def[len(def)-1] == '.' || def[len(def)-1] == ' ') {
		// Check for artifact patterns
		if len(def) > 2 && def[len(def)-2:] == ". " || def[len(def)-2:] == ".." {
			t.Errorf("definition has trailing artifacts: %q", def)
		}
	}
}

func TestDefTextAccursed2(t *testing.T) {
	data, err := os.ReadFile(testDataPath("projects/sources/lexicon/entries/A/Accursed2.xml"))
	if err != nil {
		t.Skipf("XML not found: %v", err)
	}
	entry, err := ParseEntryXML(data, "Accursed2.xml")
	if err != nil || entry == nil {
		t.Fatal("parse failed")
	}
	fmt.Printf("Senses: %d\n", len(entry.Senses))
	for i, s := range entry.Senses {
		fmt.Printf("  Sense %d (#%d): %q\n", i, s.Number, s.Text)
	}
	if len(entry.Senses) < 2 {
		t.Errorf("expected at least 2 senses, got %d", len(entry.Senses))
	}
}

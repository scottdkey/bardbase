package parser

import (
	"os"
	"path/filepath"
	"fmt"
	"testing"
)

func TestCheckJCGlobeNumbers(t *testing.T) {
	perseusDir := filepath.Join("..", "..", "..", "sources", "perseus-plays")
	data, err := os.ReadFile(filepath.Join(perseusDir, "1999.03.0027.xml"))
	if err != nil { t.Skipf("file not found: %v", err) }

	lines, err := ParsePerseusTEI(data)
	if err != nil { t.Fatalf("parse error: %v", err) }

	// Show Globe milestones for ALL of Act 3
	fmt.Println("=== JC Act 3 — ALL Globe milestones ===")
	fmt.Printf("%-6s %-6s %-7s %s\n", "Scene", "N", "Globe", "Text (first 50)")
	n := 0
	for _, l := range lines {
		if l.Act == 3 {
			n++
			if l.GlobeLine > 0 {
				text := l.Text
				if len(text) > 50 { text = text[:50] }
				fmt.Printf("%-6d %-6d %-7d %s\n", l.Scene, n, l.GlobeLine, text)
			}
		}
	}
	fmt.Printf("\nTotal Act 3 lines: %d\n", n)
}

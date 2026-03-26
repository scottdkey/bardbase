package parser

import (
	"os"
	"path/filepath"
	"fmt"
	"testing"
)

func TestCheckJCGlobeNumbers(t *testing.T) {
	perseusDir := filepath.Join("..", "..", "..", "sources", "perseus-plays")
	data, _ := os.ReadFile(filepath.Join(perseusDir, "1999.03.0042.xml"))
	lines, _ := ParsePerseusTEI(data)

	type key struct{ a, s int }
	counts := map[key]int{}
	for _, l := range lines { counts[key{l.Act, l.Scene}]++ }

	fmt.Println("=== Troilus parser output ===")
	fmt.Printf("%-8s %-8s %-8s\n", "Act", "Scene", "Lines")
	total := 0
	for a := 1; a <= 5; a++ {
		for s := 1; s <= 15; s++ {
			if c, ok := counts[key{a, s}]; ok {
				fmt.Printf("%-8d %-8d %-8d\n", a, s, c)
				total += c
			}
		}
	}
	fmt.Printf("Total: %d\n", total)
	
	// Now check: XML has Act 2 Scene 4 — OSS doesn't have that scene
	// What does OSS have for this play?
	fmt.Println("\nNote: XML Act 2 has scenes 1,2,3,4 but OSS may split differently")
}

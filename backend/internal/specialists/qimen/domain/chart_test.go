package domain

import "testing"

func TestChartValidateRejectsCrossSystemSymbols(t *testing.T) {
	tests := map[string]Cell{
		"middle door":         {Palace: "坎", Door: "中门", God: "值符"},
		"middle abbreviation": {Palace: "坎", Door: "中", God: "值符"},
		"太常":                  {Palace: "坎", Door: "休", God: "太常"},
		"勾陈":                  {Palace: "坎", Door: "休", God: "勾陈"},
		"朱雀":                  {Palace: "坎", Door: "休", God: "朱雀"},
	}
	for name, cell := range tests {
		t.Run(name, func(t *testing.T) {
			if err := (Chart{Cells: []Cell{cell}}).Validate(); err == nil {
				t.Fatal("invalid rotating_8 symbol was accepted")
			}
		})
	}

	if err := (Chart{Cells: []Cell{{Palace: "中", Door: "休", God: "值符"}}}).Validate(); err != nil {
		t.Fatalf("middle palace must remain valid: %v", err)
	}
}

package checks

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		days int
		warn int
		crit int
		want string
	}{
		{name: "caducado", days: -1, warn: 30, crit: 7, want: "CADUCADO"},
		{name: "crítico", days: 6, warn: 30, crit: 7, want: "CRÍTICO"},
		{name: "límite crítico", days: 7, warn: 30, crit: 7, want: "AVISO"},
		{name: "aviso", days: 29, warn: 30, crit: 7, want: "AVISO"},
		{name: "límite aviso", days: 30, warn: 30, crit: 7, want: "OK"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Classify(test.days, test.warn, test.crit); got != test.want {
				t.Errorf("Classify(%d, %d, %d) = %q, want %q", test.days, test.warn, test.crit, got, test.want)
			}
		})
	}
}

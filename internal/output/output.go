package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"vigia/internal/checks"
	"vigia/internal/probe"
)

type Options struct {
	WarnDays int
	CritDays int
}

type jsonResult struct {
	State    string `json:"state"`
	Target   string `json:"target"`
	DaysLeft *int   `json:"days_left,omitempty"`
	NotAfter string `json:"not_after,omitempty"`
	KeyInfo  string `json:"key_info,omitempty"`
	Issuer   string `json:"issuer,omitempty"`
	Verify   string `json:"verification,omitempty"`
	Error    string `json:"error,omitempty"`
}

func Write(w io.Writer, format string, results []probe.Result, options Options) error {
	ordered := append([]probe.Result(nil), results...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if (a.Err != nil) != (b.Err != nil) {
			return a.Err != nil
		}
		return a.DaysLeft < b.DaysLeft
	})

	switch format {
	case "table":
		return writeTable(w, ordered, options)
	case "json":
		return writeJSON(w, ordered, options)
	default:
		return fmt.Errorf("formato no admitido: %s", format)
	}
}

func writeTable(w io.Writer, results []probe.Result, options Options) error {
	table := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "ESTADO\tOBJETIVO\tDÍAS\tCADUCA\tCLAVE\tEMISOR\tVALIDACIÓN"); err != nil {
		return err
	}
	for _, result := range results {
		if result.Err != nil {
			if _, err := fmt.Fprintf(table, "ERROR\t%s\t-\t-\t-\t-\t%s\n", result.Target, result.Error); err != nil {
				return err
			}
			continue
		}
		state := checks.Classify(result.DaysLeft, options.WarnDays, options.CritDays)
		if _, err := fmt.Fprintf(table, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
			state, result.Target, result.DaysLeft, result.NotAfter.Format("2006-01-02"), result.KeyInfo, result.Issuer, result.Verify); err != nil {
			return err
		}
	}
	return table.Flush()
}

func writeJSON(w io.Writer, results []probe.Result, options Options) error {
	encoded := make([]jsonResult, 0, len(results))
	for _, result := range results {
		item := jsonResult{Target: result.Target}
		if result.Err != nil {
			item.State = "ERROR"
			item.Error = result.Error
		} else {
			days := result.DaysLeft
			item.State = checks.Classify(result.DaysLeft, options.WarnDays, options.CritDays)
			item.DaysLeft = &days
			item.NotAfter = result.NotAfter.Format("2006-01-02")
			item.KeyInfo = result.KeyInfo
			item.Issuer = result.Issuer
			item.Verify = result.Verify
		}
		encoded = append(encoded, item)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(encoded)
}

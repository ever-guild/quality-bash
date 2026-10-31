package quality

import (
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
)

type LineCoverage struct {
	File      string `json:"file"`
	Covered   uint64 `json:"covered"`
	Total     uint64 `json:"total"`
	Uncovered []int  `json:"uncovered"`
}
type CoverageReport struct {
	Score Ratio          `json:"score"`
	Files []LineCoverage `json:"files"`
}
type cobertura struct {
	XMLName xml.Name `xml:"coverage"`
	Sources []string `xml:"sources>source"`
	Classes []struct {
		Filename string `xml:"filename,attr"`
		Lines    []struct {
			Number int   `xml:"number,attr"`
			Hits   int64 `xml:"hits,attr"`
		} `xml:"lines>line"`
	} `xml:"packages>package>classes>class"`
}

func ReadCoverage(reader io.Reader, root string, selected []string) (CoverageReport, error) {
	var report CoverageReport
	var doc cobertura
	if err := xml.NewDecoder(reader).Decode(&doc); err != nil {
		return report, fmt.Errorf("invalid Cobertura: %w", err)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return report, err
	}
	wanted := map[string]string{}
	lines := map[string]map[int]bool{}
	for _, file := range selected {
		abs := filepath.Join(root, file)
		if filepath.IsAbs(file) {
			abs = filepath.Clean(file)
		}
		if _, exists := wanted[abs]; exists {
			return report, fmt.Errorf("duplicate coverage file: %s", file)
		}
		wanted[abs] = file
		lines[file] = map[int]bool{}
	}
	for _, class := range doc.Classes {
		candidates := map[string]bool{}
		if filepath.IsAbs(class.Filename) {
			candidates[filepath.Clean(class.Filename)] = true
		} else {
			candidates[filepath.Join(root, class.Filename)] = true
			for _, source := range doc.Sources {
				if !filepath.IsAbs(source) {
					source = filepath.Join(root, source)
				}
				candidates[filepath.Join(source, class.Filename)] = true
			}
		}
		match := ""
		for candidate := range candidates {
			if name, ok := wanted[candidate]; ok {
				if match != "" && match != name {
					return report, fmt.Errorf("ambiguous Cobertura filename: %s", class.Filename)
				}
				match = name
			}
		}
		if match == "" {
			continue
		}
		for _, line := range class.Lines {
			if line.Number <= 0 || line.Hits < 0 {
				return report, fmt.Errorf("invalid line data for %s", match)
			}
			lines[match][line.Number] = lines[match][line.Number] || line.Hits > 0
		}
	}
	for _, file := range selected {
		entry := LineCoverage{File: file, Uncovered: []int{}}
		if len(lines[file]) == 0 {
			return report, fmt.Errorf("no traced lines for selected script %s", file)
		}
		for number, hit := range lines[file] {
			entry.Total++
			if hit {
				entry.Covered++
			} else {
				entry.Uncovered = append(entry.Uncovered, number)
			}
		}
		sort.Ints(entry.Uncovered)
		report.Files = append(report.Files, entry)
		report.Score.Numerator += entry.Covered
		report.Score.Denominator += entry.Total
	}
	if report.Score.Denominator == 0 || report.Score.Numerator == 0 {
		return report, fmt.Errorf("empty coverage or no evidence of tracing")
	}
	sort.Slice(report.Files, func(i, j int) bool { return report.Files[i].File < report.Files[j].File })
	return report, nil
}

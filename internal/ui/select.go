package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func PromptSelectIndices(prompt string, options []string, allowMulti bool) ([]int, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options to select")
	}

	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("%2d) %s\n", i+1, opt)
	}

	if allowMulti {
		fmt.Print("Select (e.g., 1,3-5) or press Enter to cancel: ")
	} else {
		fmt.Print("Select (e.g., 1) or press Enter to cancel: ")
	}

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("selection cancelled")
	}

	indices, err := parseSelection(line, len(options), allowMulti)
	if err != nil {
		return nil, err
	}
	return indices, nil
}

func parseSelection(input string, max int, allowMulti bool) ([]int, error) {
	var result []int
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			if !allowMulti {
				return nil, fmt.Errorf("range selection not allowed")
			}
			bounds := strings.SplitN(part, "-", 2)
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid selection: %s", part)
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid selection: %s", part)
			}
			if start < 1 || end < 1 || start > max || end > max || start > end {
				return nil, fmt.Errorf("selection out of range: %s", part)
			}
			for i := start; i <= end; i++ {
				result = append(result, i-1)
			}
		} else {
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid selection: %s", part)
			}
			if idx < 1 || idx > max {
				return nil, fmt.Errorf("selection out of range: %d", idx)
			}
			result = append(result, idx-1)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no selection made")
	}
	if !allowMulti && len(result) > 1 {
		return nil, fmt.Errorf("multiple selections not allowed")
	}

	return dedupe(result), nil
}

func dedupe(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

package evalharness

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadCases(path string) ([]Case, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cases []Case
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var c Case
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		if c.ID == "" {
			return nil, fmt.Errorf("%s:%d: id is required", path, lineNo)
		}
		cases = append(cases, c)
	}
	return cases, scanner.Err()
}

func LoadObservations(path string) (map[string]Observation, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	out := map[string]Observation{}
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var obs Observation
		if err := json.Unmarshal([]byte(line), &obs); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		if obs.ID == "" {
			return nil, fmt.Errorf("%s:%d: id is required", path, lineNo)
		}
		out[obs.ID] = obs
	}
	return out, scanner.Err()
}

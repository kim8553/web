//go:build ignore

// Copied to a temporary CI-only directory. This adapter loads only synthetic
// INI fixtures for isolated purchase preflight tests. It is NOT the production
// resource loader and MUST NOT be used in the running server.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type currentShopConditionAuthority struct {
	definitions map[int32]exchangeConditionSpec
	audit       shopExchangeConditionCapabilityAudit
}

func loadINISections(path string) (map[string]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sections := make(map[string]map[string]string)
	var section map[string]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			if name == "" {
				return nil, fmt.Errorf("empty INI section")
			}
			section = make(map[string]string)
			sections[name] = section
			continue
		}
		if section == nil {
			return nil, fmt.Errorf("INI field outside section")
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("INI field missing delimiter")
		}
		section[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return sections, scanner.Err()
}

func iniValue(fields map[string]string, key string) string { return fields[key] }

func iniInt(fields map[string]string, key string) int32 {
	value, _ := strconv.ParseInt(strings.TrimSpace(fields[key]), 10, 32)
	return int32(value)
}

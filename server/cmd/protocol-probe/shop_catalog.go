package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var defaultShopINIPath = filepath.Join(defaultModernShareRoot, "trade", "shop.ini")

type shopCatalogItem struct {
	configID     string
	amount       int32
	priceMode    int32
	exchangeData int32
	price        int32
	page         int32
	position     int32
}

func loadShopCatalogSection(path, shopID string) ([]shopCatalogItem, int32, int32, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("open shop catalog %s: %w", path, err)
	}
	defer file.Close()
	wantedHeader := "[" + shopID + "]"
	inSection := false
	shopType := int32(0)
	pageCount := int32(0)
	maxPageKey := int32(-1)
	items := make([]shopCatalogItem, 0, 10)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if inSection {
				break
			}
			inSection = line == wantedHeader
			continue
		}
		if !inSection {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "Type" {
			parsed, parseErr := strconv.ParseInt(value, 10, 32)
			if parseErr != nil {
				return nil, 0, 0, fmt.Errorf("shop %s line %d: invalid Type %q: %w", shopID, lineNumber, value, parseErr)
			}
			shopType = int32(parsed)
			continue
		}
		if key == "PageInfo" {
			count := int32(0)
			for _, pageName := range strings.Split(value, ",") {
				if strings.TrimSpace(pageName) != "" {
					count++
				}
			}
			if count > 0 {
				pageCount = count
			}
			continue
		}
		parsedPage, parseErr := strconv.ParseInt(key, 10, 32)
		if parseErr != nil {
			continue
		}
		page := int32(parsedPage)
		parts := strings.Split(value, ",")
		if len(parts) < 7 {
			return nil, 0, 0, fmt.Errorf("shop %s line %d: expected at least 7 item columns, got %d", shopID, lineNumber, len(parts))
		}
		parseColumn := func(column int, name string) (int32, error) {
			parsed, parseErr := strconv.ParseInt(strings.TrimSpace(parts[column]), 10, 32)
			if parseErr != nil {
				return 0, fmt.Errorf("shop %s line %d: invalid %s %q: %w", shopID, lineNumber, name, parts[column], parseErr)
			}
			return int32(parsed), nil
		}
		amount, parseErr := parseColumn(1, "amount")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		priceMode, parseErr := parseColumn(2, "price mode")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		price, parseErr := parseColumn(3, "price")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		position, parseErr := parseColumn(6, "position")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		exchangeData := int32(0)
		if len(parts) > 7 && strings.TrimSpace(parts[7]) != "" {
			exchangeData, parseErr = parseColumn(7, "exchange data")
			if parseErr != nil {
				return nil, 0, 0, parseErr
			}
		}
		if page < 0 || position < 0 {
			return nil, 0, 0, fmt.Errorf("shop %s line %d: invalid page=%d position=%d", shopID, lineNumber, page, position)
		}
		items = append(items, shopCatalogItem{configID: strings.TrimSpace(parts[0]), amount: amount, priceMode: priceMode, price: price, page: page, position: position, exchangeData: exchangeData})
		if page > maxPageKey {
			maxPageKey = page
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("read shop catalog %s: %w", path, err)
	}
	if !inSection {
		return nil, 0, 0, fmt.Errorf("shop catalog %s has no section %q", path, shopID)
	}
	if len(items) == 0 {
		return nil, 0, 0, fmt.Errorf("shop catalog %s section %q has no items", path, shopID)
	}
	// PageInfo can underdeclare an authored numeric page (the exact RAR has
	// Shop_special_001: PageInfo=Page1 but an item at page key 1). The
	// current client's PageCount must include every authored item page;
	// never synthesize products or alter their page/position coordinates.
	if authoredPageCount := maxPageKey + 1; authoredPageCount > pageCount {
		pageCount = authoredPageCount
	}
	return items, shopType, pageCount, nil
}

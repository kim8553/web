package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type itemModels struct {
	male   string
	female string
}
type equipmentResolver struct {
	artByItem map[string]string
	models    map[string]itemModels
}

func loadEquipmentResolver() *equipmentResolver {
	root := os.Getenv("NINEYIN_SHARE_ROOT")
	if root == "" {
		root = filepath.Join(defaultModernShareRoot, "item")
	}
	r := &equipmentResolver{artByItem: make(map[string]string), models: make(map[string]itemModels)}
	parseINI(filepath.Join(root, "equipment.ini"), func(section, key, value string) {
		if key == "ArtPack" {
			r.artByItem[section] = value
		}
	})
	parseINI(filepath.Join(root, "itemartstatic.ini"), func(section, key, value string) {
		models := r.models[section]
		switch key {
		case "MaleModel":
			models.male = value
		case "FemaleModel":
			models.female = value
		}
		r.models[section] = models
	})
	return r
}
func parseINI(path string, consume func(string, string, string)) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	section := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		if section == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		consume(section, strings.TrimSpace(key), strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return
	}
}
func (r *equipmentResolver) resolve(config string, sex uint8) string {
	if config == "" || strings.Contains(config, "\\") {
		return config
	}
	art := r.artByItem[config]
	models := r.models[art]
	if sex == 1 && models.female != "" {
		return models.female
	}
	if models.male != "" {
		return models.male
	}
	return config
}

type roleVisual struct {
	sex       uint8
	photo     string
	face      string
	hair      string
	cloth     string
	pants     string
	shoes     string
	actionSet string
}

func resolveRoleVisual(appearance []string) roleVisual {
	value := func(index int, fallback string) string {
		if index < len(appearance) && appearance[index] != "" {
			return appearance[index]
		}
		return fallback
	}
	photo := value(1, `icon\players\boy01_create.png`)
	sex := uint8(0)
	if strings.Contains(strings.ToLower(photo), "girl") {
		sex = 1
	}
	face := value(2, "333333333333333333333333333333333333333333X38\x01")
	hair := value(4, `obj\char\b_hair\b_hair1`)
	cloth := equipmentModels.resolve(value(5, "cloth_b0001"), sex)
	pants := equipmentModels.resolve(value(6, "pants_b0001"), sex)
	shoes := equipmentModels.resolve(value(7, "shoes_b0001"), sex)
	return roleVisual{sex: sex, photo: photo, face: face, hair: hair, cloth: cloth, pants: pants, shoes: shoes, actionSet: "0h"}
}

var equipmentModels = loadEquipmentResolver()

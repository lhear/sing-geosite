package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
)

type RuleUrl struct {
	Name string
	Url  string
}

var (
	CLASH_RULE_URL = []RuleUrl{
		{"gfw.srs", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/gfw.txt"},
		{"ads.srs", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/reject.txt"},
	}
)

func main() {
	for _, e := range CLASH_RULE_URL {
		src, err := ConvertYAMLToJSON(e.Url)
		if err != nil {
			log.Fatalln(err.Error())
		}
		os.MkdirAll("rule-set", os.ModePerm)
		compileRuleSet(src, "rule-set/"+e.Name)
	}
}

// clash规则转json
func ConvertYAMLToJSON(yamlPath string) ([]byte, error) {
	var yamlData []byte
	var err error

	if strings.HasPrefix(yamlPath, "https://") {
		response, err := http.Get(yamlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch YAML data: %v", err)
		}
		defer response.Body.Close()

		yamlData, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read YAML data: %v", err)
		}
	} else {
		yamlData, err = ioutil.ReadFile(yamlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read YAML file: %v", err)
		}
	}

	yamlMap := make(map[string]interface{})
	err = yaml.Unmarshal(yamlData, &yamlMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML data: %v", err)
	}

	jsonData := map[string]interface{}{
		"version": 1,
		"rules": []map[string]interface{}{
			{
				"domain_suffix": func() []string {
					var domains []string
					for _, domain := range yamlMap["payload"].([]interface{}) {
						domains = append(domains, strings.Replace(domain.(string), "+", "", -1))
					}
					return domains
				}(),
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(jsonData, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSON data: %v", err)
	}

	return jsonBytes, nil
}

func compileRuleSet(sourceJson []byte, outputPath string) error {
	var err error
	content := sourceJson
	plainRuleSet, err := sjson.UnmarshalExtended[option.PlainRuleSetCompat](content)
	if err != nil {
		return err
	}
	ruleSet := plainRuleSet.Upgrade()
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	err = srs.Write(outputFile, ruleSet)
	if err != nil {
		outputFile.Close()
		os.Remove(outputPath)
		return err
	}
	outputFile.Close()
	return nil
}

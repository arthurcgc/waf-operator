package rules

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/arthurcgc/waf-operator/api/v1"
)

var CustomRulesKey = "CUSTOM-RULES.conf"

type WAFRule map[string]string

func renderRules() (WAFRule, error) {
	var rules WAFRule = make(map[string]string)
	root, err := os.Getwd()
	if err != nil {
		return WAFRule{}, err
	}

	rootPath := fmt.Sprintf("%s/rules", root)
	err = filepath.Walk(rootPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			pathName := fmt.Sprintf("%s/%s", rootPath, info.Name())
			content, err := ioutil.ReadFile(pathName)
			if err != nil {
				return err
			}

			rules[info.Name()] = string(content)
			return nil
		})
	if err != nil {
		return WAFRule{}, err
	}

	// include custom rules as empty string
	rules[CustomRulesKey] = ""
	return rules, nil
}

func mergeRulesAfter(currentRules map[string]string, rulesAfter *v1.RulesAfter) WAFRule {
	for _, ruleId := range rulesAfter.RemoveById {
		currentRules["RESPONSE-999-EXCLUSION-RULES-AFTER-CRS.conf"] = strings.Join(
			[]string{currentRules["RESPONSE-999-EXCLUSION-RULES-AFTER-CRS.conf"],
				fmt.Sprintf("SecRuleRemoveById %s\n", ruleId)}, "")
	}

	for _, ruleTag := range rulesAfter.RemoveByTag {
		currentRules["RESPONSE-999-EXCLUSION-RULES-AFTER-CRS.conf"] = strings.Join(
			[]string{currentRules["RESPONSE-999-EXCLUSION-RULES-AFTER-CRS.conf"],
				fmt.Sprintf("SecRuleRemoveByTag \"%s\"\n", ruleTag)}, "")
	}

	return currentRules
}

func mergeRules(currentRules WAFRule, instanceRules v1.Rules) (WAFRule, error) {
	if len(instanceRules.CustomRules) > 0 {
		currentRules[CustomRulesKey] = strings.Join(instanceRules.CustomRules, "\n")
	}
	if instanceRules.RulesAfter != nil {
		currentRules = mergeRulesAfter(currentRules, instanceRules.RulesAfter)
	}
	// if instanceRules.EnableDefaultHoneyPot {
	//	If enabled we set the following rule inside REQUEST-910-IP-REPUTATION.conf:
	// 	This rule checks the client IP address against a list of recent IPs captured
	//  from the SpiderLabs web honeypot systems (last 48 hours).
	//
	// SecRule TX:REAL_IP "@ipMatchFromFile ip_blacklist.data" \
	//     "id:910110,\
	//     phase:2,\
	//     block,\
	//     t:none,\
	//     msg:'Client IP in Trustwave SpiderLabs IP Reputation Blacklist',\
	//     tag:'application-multi',\
	//     tag:'language-multi',\
	//     tag:'platform-multi',\
	//     tag:'attack-reputation-ip',\
	//     tag:'paranoia-level/1',\
	//     severity:'CRITICAL',\
	//     setvar:'tx.anomaly_score_pl1=+%{tx.critical_anomaly_score}',\
	//     setvar:'ip.reput_block_flag=1',\
	//     setvar:'ip.reput_block_reason=%{rule.msg}',\
	//     expirevar:'ip.reput_block_flag=%{tx.reput_block_duration}'"
	// }

	return currentRules, nil

}

func RenderRules(instance *v1.Waf) (WAFRule, error) {
	rules, err := renderRules()
	if err != nil {
		return nil, err
	}

	return mergeRules(rules, instance.Spec.Rules)
}

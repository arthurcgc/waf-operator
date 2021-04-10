/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

type Rules struct {
	// Used to add/edit rules in RESPONSE-999-EXCLUSION-RULES-AFTER-CRS.conf
	RulesAfter `json:"removeAfter,omitempty"`

	// Possible user created rules
	// key == file name to be created for the rule
	// value == contents of the rule
	CustomRules map[string]string `json:"customRules,omitempty"`

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
	EnableDefaultHoneyPot bool `json:"defaultHoney,omitempty"`
}

type RulesAfter struct {
	// Example Exclusion Rule: To unconditionally disable a rule ID
	// ModSecurity Rule Exclusion: 942100 SQL Injection Detected via libinjection
	// SecRuleRemoveById 942100
	RemoveById []string `json:"removeById,omitempty"`

	// Example Exclusion Rule: Remove a group of rules
	// ModSecurity Rule Exclusion: Disable PHP injection rules
	// SecRuleRemoveByTag "attack-injection-php"
	RemoveByTag []string `json:"removeByTag,omitempty"`

	// In Anomaly Mode (default in CRS3), the rules in REQUEST-949-BLOCKING-EVALUATION.conf
	// and RESPONSE-959-BLOCKING-EVALUATION.conf check the accumulated attack scores
	// against your policy. To apply a disruptive action, they overwrite the default
	// actions specified in SecDefaultAction (setup.conf) with a 'deny' action.
	// This 'deny' is by default paired with a 'status:403' action.
	DisruptiveAction DisruptiveAction `json:"disruptiveAction,omitempty"`
}

type DisruptiveAction struct {
	// Example: redirect back to the homepage on blocking
	// SecRuleUpdateActionById 949110 "t:none,redirect:'http://%{request_headers.host}/'"
	// SecRuleUpdateActionById 959100 "t:none,redirect:'http://%{request_headers.host}/'"

	// In the example above, we would set Action = redirect and RedirectURL = "http://%{request_headers.host}/"
	Action      string `json:"action,omitempty"`
	RedirectURL string `json:"redirect,omitempty"`

	// Example: drop the connection (best for DoS attacks)
	// SecRuleUpdateActionById 949110 "t:none,drop"
	// SecRuleUpdateActionById 959100 "t:none,drop"
	// When enabled we should add a DisruptiveAction to drop the connection
	Dos bool `json:"dos,omitempty"`
}

package rules

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type WAFRule map[string]string

func RenderRules() (WAFRule, error) {
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

	return rules, nil
}

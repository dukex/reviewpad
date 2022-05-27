// Copyright 2022 Explore.dev Unipessoal Lda. All Rights Reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package engine

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"

	"gopkg.in/yaml.v3"
)

type LoadEnv struct {
	Visited map[string]bool
	Stack   map[string]bool
}

func hash(data []byte) string {
	dataHash := sha256.Sum256(data)
	dHash := fmt.Sprintf("%x", dataHash)
	return dHash
}

func Load(data []byte) (*RevyFile, error) {
	file, err := parse(data)
	if err != nil {
		return nil, err
	}

	transformedFile := transform(file)

	dHash := hash(data)

	visited := make(map[string]bool)
	stack := make(map[string]bool)
	visited[dHash] = true
	stack[dHash] = true

	env := &LoadEnv{
		Visited: visited,
		Stack:   stack,
	}

	return inlineImports(transformedFile, env)
}

func parse(data []byte) (*RevyFile, error) {
	file := RevyFile{}
	err := yaml.Unmarshal([]byte(data), &file)
	if err != nil {
		return nil, err
	}

	return &file, nil
}

func transform(file *RevyFile) *RevyFile {
	var transformedProtectionGates []PadProtectionGate
	for _, gate := range file.ProtectionGates {
		var transformedPatchRules []PatchRule
		for _, patchRule := range gate.PatchRules {
			var transformedExtraActions []string
			for _, extraAction := range patchRule.ExtraActions {
				transformedExtraActions = append(transformedExtraActions, transformActionStr(extraAction))
			}

			transformedPatchRules = append(transformedPatchRules, PatchRule{
				Rule:         patchRule.Rule,
				ExtraActions: transformedExtraActions,
			})
		}

		var transformedActions []string
		for _, action := range gate.Actions {
			transformedActions = append(transformedActions, transformActionStr(action))
		}

		transformedProtectionGates = append(transformedProtectionGates, PadProtectionGate{
			Name:        gate.Name,
			Description: gate.Description,
			PatchRules:  transformedPatchRules,
			Actions:     transformedActions,
			AlwaysRun:   gate.AlwaysRun,
		})
	}

	return &RevyFile{
		Version:         file.Version,
		Mode:            file.Mode,
		Imports:         file.Imports,
		Groups:          file.Groups,
		Rules:           file.Rules,
		Labels:          file.Labels,
		ProtectionGates: transformedProtectionGates,
	}
}

func loadImport(revyImport PadImport) (*RevyFile, string, error) {
	resp, err := http.Get(revyImport.Url)
	if err != nil {
		return nil, "", err
	}

	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	file, err := parse(content)
	if err != nil {
		return nil, "", err
	}

	transformedFile := transform(file)

	return transformedFile, hash(content), nil
}

// InlineImports inlines the imports files into the current revy file
// Post-condition: RevyFile without import statements
func inlineImports(file *RevyFile, env *LoadEnv) (*RevyFile, error) {
	for _, revyImport := range file.Imports {
		iFile, idHash, err := loadImport(revyImport)
		if err != nil {
			return nil, err
		}

		// check for cycles
		if _, ok := env.Stack[idHash]; ok {
			return nil, fmt.Errorf("loader: cyclic dependency")
		}

		// optimize visits
		if _, ok := env.Visited[idHash]; ok {
			continue
		}

		// DFS call inline imports
		// update the environment
		env.Stack[idHash] = true
		env.Visited[idHash] = true

		subTreeFile, err := inlineImports(iFile, env)
		if err != nil {
			return nil, err
		}

		// remove from the stack
		delete(env.Stack, idHash)

		// append labels, rules and protection gates
		file.appendLabels(subTreeFile)
		file.appendRules(subTreeFile)
		file.appendProtectionGates(subTreeFile)
	}

	// reset all imports
	file.Imports = []PadImport{}

	return file, nil
}
/*
Copyright © 2019 Itay Shakury @itaysk

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
package neat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"
)

// Clean gets a YAML or JSON string and de-clutters it to make it more readable.
func Clean(in string) (string, error) {
	// Try to detect if input is JSON or YAML
	trimmed := strings.TrimSpace(in)
	if trimmed == "" {
		return "", nil
	}

	// If it starts with { or [, assume it's JSON
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		cleaned, err := Neat(in)
		if err != nil {
			return in, err
		}
		return cleaned, nil
	}

	// Otherwise, assume it's YAML - convert to JSON, clean, convert back
	return cleanYAML(in)
}

// CleanYAMLToJSON converts YAML to cleaned JSON
func CleanYAMLToJSON(yamlStr string) (string, error) {
	if strings.TrimSpace(yamlStr) == "" {
		return "", nil
	}

	// Parse YAML to interface{}
	var obj interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &obj); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert to JSON for processing
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to convert to JSON: %w", err)
	}

	// Clean the JSON using kubectl-neat logic
	return Neat(string(jsonBytes))
}

func cleanYAML(yamlStr string) (string, error) {
	// Convert YAML to JSON, clean it, then convert back
	cleaned, err := CleanYAMLToJSON(yamlStr)
	if err != nil {
		return yamlStr, err
	}

	// Convert cleaned JSON back to YAML
	var obj interface{}
	if err := json.Unmarshal([]byte(cleaned), &obj); err != nil {
		return yamlStr, err
	}

	yamlBytes, err := yaml.Marshal(obj)
	if err != nil {
		return yamlStr, err
	}

	return string(yamlBytes), nil
}

// Neat gets a Kubernetes resource json as string and de-clutters it to make it more readable.
// This is the main kubectl-neat implementation
func Neat(in string) (string, error) {
	var err error
	draft := in

	if in == "" {
		return draft, fmt.Errorf("error in neat, input json is empty")
	}
	if !gjson.Valid(in) {
		return draft, fmt.Errorf("error in neat, input is not a valid json: %s", in[:20])
	}

	kind := gjson.Get(in, "kind").String()

	// handle list
	if kind == "List" {
		items := gjson.Get(draft, "items").Array()
		for i, item := range items {
			itemNeat, err := Neat(item.String())
			if err != nil {
				continue
			}
			draft, err = sjson.SetRaw(draft, fmt.Sprintf("items.%d", i), itemNeat)
			if err != nil {
				continue
			}
		}
		// general neating
		draft, err = neatMetadata(draft)
		if err != nil {
			return draft, fmt.Errorf("error in neatMetadata : %v", err)
		}
		return draft, nil
	}

	// defaults neating - this is the key part that removes things like clusterIP, internalTrafficPolicy, etc.
	draft, err = neatDefaults(draft)
	if err != nil {
		return draft, fmt.Errorf("error in neatDefaults : %v", err)
	}

	// controllers neating
	draft, err = neatScheduler(draft)
	if err != nil {
		return draft, fmt.Errorf("error in neatScheduler : %v", err)
	}
	if kind == "Pod" {
		draft, err = neatServiceAccount(draft)
		if err != nil {
			return draft, fmt.Errorf("error in neatServiceAccount : %v", err)
		}
	}

	// general neating
	draft, err = neatMetadata(draft)
	if err != nil {
		return draft, fmt.Errorf("error in neatMetadata : %v", err)
	}
	draft, err = neatStatus(draft)
	if err != nil {
		return draft, fmt.Errorf("error in neatStatus : %v", err)
	}
	draft, err = neatEmpty(draft)
	if err != nil {
		return draft, fmt.Errorf("error in neatEmpty : %v", err)
	}

	return draft, nil
}

func neatMetadata(in string) (string, error) {
	var err error
	in, err = sjson.Delete(in, `metadata.annotations.kubectl\.kubernetes\.io/last-applied-configuration`)
	if err != nil {
		return in, fmt.Errorf("error deleting last-applied-configuration : %v", err)
	}
	// TODO: prettify this. gjson's @pretty is ok but setRaw the pretty code gives unwanted result
	newMeta := gjson.Get(in, "{metadata.name,metadata.namespace,metadata.labels,metadata.annotations}")
	in, err = sjson.Set(in, "metadata", newMeta.Value())
	if err != nil {
		return in, fmt.Errorf("error setting new metadata : %v", err)
	}
	return in, nil
}

func neatStatus(in string) (string, error) {
	return sjson.Delete(in, "status")
}

func neatScheduler(in string) (string, error) {
	return sjson.Delete(in, "spec.nodeName")
}

func neatServiceAccount(in string) (string, error) {
	var err error
	// keep an eye open on https://github.com/tidwall/sjson/issues/11
	// when it's implemented, we can do:
	// sjson.delete(in, "spec.volumes.#(name%default-token-*)")
	// sjson.delete(in, "spec.containers.#.volumeMounts.#(name%default-token-*)")

	for vi, v := range gjson.Get(in, "spec.volumes").Array() {
		vname := v.Get("name").String()
		if strings.HasPrefix(vname, "default-token-") {
			in, err = sjson.Delete(in, fmt.Sprintf("spec.volumes.%d", vi))
			if err != nil {
				continue
			}
		}
	}
	for ci, c := range gjson.Get(in, "spec.containers").Array() {
		for vmi, vm := range c.Get("volumeMounts").Array() {
			vmname := vm.Get("name").String()
			if strings.HasPrefix(vmname, "default-token-") {
				in, err = sjson.Delete(in, fmt.Sprintf("spec.containers.%d.volumeMounts.%d", ci, vmi))
				if err != nil {
					continue
				}
			}
		}
	}
	in, _ = sjson.Delete(in, "spec.serviceAccount") //Deprecated: Use serviceAccountName instead

	return in, nil
}

// neatEmpty removes all zero length elements in the json
func neatEmpty(in string) (string, error) {
	var err error
	jsonResult := gjson.Parse(in)
	var empties []string
	findEmptyPathsRecursive(jsonResult, "", &empties)
	for _, emptyPath := range empties {
		// if we just delete emptyPath, it may create empty parents
		// so we walk the path and re-check for emptiness at every level
		emptyPathParts := strings.Split(emptyPath, ".")
		for i := len(emptyPathParts); i > 0; i-- {
			curPath := strings.Join(emptyPathParts[:i], ".")
			cur := gjson.Get(in, curPath)
			if isResultEmpty(cur) {
				in, err = sjson.Delete(in, curPath)
				if err != nil {
					continue
				}
			}
		}
	}
	return in, nil
}

// findEmptyPathsRecursive builds a list of paths that point to zero length elements
// cur is the current element to look at
// path is the path to cur
// res is a pointer to a list of empty paths to populate
func findEmptyPathsRecursive(cur gjson.Result, path string, res *[]string) {
	if isResultEmpty(cur) {
		*res = append(*res, path[1:]) //remove '.' from start
		return
	}
	if !(cur.IsArray() || cur.IsObject()) {
		return
	}
	// sjson's ForEach doesn't put track index when iterating arrays, hence the index variable
	index := -1
	cur.ForEach(func(k gjson.Result, v gjson.Result) bool {
		var newPath string
		if cur.IsArray() {
			index++
			newPath = fmt.Sprintf("%s.%d", path, index)
		} else {
			newPath = fmt.Sprintf("%s.%s", path, k.Str)
		}
		findEmptyPathsRecursive(v, newPath, res)
		return true
	})
}

func isResultEmpty(j gjson.Result) bool {
	v := j.Value()
	switch vt := v.(type) {
	// empty string != lack of string. keep empty strings as it's meaningful data
	// case string:
	// 	return vt == ""
	case []interface{}:
		return len(vt) == 0
	case map[string]interface{}:
		return len(vt) == 0
	}
	return false
}

// neatDefaults - simplified version without Kubernetes scheme dependency
// This handles the most common service defaults that cause clutter
func neatDefaults(in string) (string, error) {
	kind := gjson.Get(in, "kind").String()

	switch kind {
	case "Service":
		return neatServiceDefaults(in)
	case "Deployment":
		return neatDeploymentDefaults(in)
	default:
		return neatCommonDefaults(in)
	}
}

func neatServiceDefaults(in string) (string, error) {
	var err error
	result := in

	// Remove service-specific defaults that cause clutter
	serviceDefaults := map[string]interface{}{
		"spec.type": "ClusterIP",
		"spec.sessionAffinity": "None",
		"spec.internalTrafficPolicy": "Cluster",
		"spec.ipFamilyPolicy": "SingleStack",
	}

	for path, defaultValue := range serviceDefaults {
		if gjson.Get(result, path).Value() == defaultValue {
			result, err = sjson.Delete(result, path)
			if err != nil {
				continue
			}
		}
	}

	// Remove default ipFamilies if it's just ["IPv4"]
	ipFamilies := gjson.Get(result, "spec.ipFamilies")
	if ipFamilies.IsArray() {
		families := ipFamilies.Array()
		if len(families) == 1 && families[0].String() == "IPv4" {
			result, _ = sjson.Delete(result, "spec.ipFamilies")
		}
	}

	// Remove clusterIP and clusterIPs as they're assigned by cluster
	result, _ = sjson.Delete(result, "spec.clusterIP")
	result, _ = sjson.Delete(result, "spec.clusterIPs")

	// Clean ports - remove default protocol TCP
	ports := gjson.Get(result, "spec.ports")
	if ports.IsArray() {
		var cleanedPorts []interface{}
		for _, port := range ports.Array() {
			portMap := port.Map()
			cleanedPort := make(map[string]interface{})

			for key, value := range portMap {
				// Skip default protocol
				if key == "protocol" && value.String() == "TCP" {
					continue
				}
				cleanedPort[key] = value.Value()
			}
			cleanedPorts = append(cleanedPorts, cleanedPort)
		}

		if len(cleanedPorts) > 0 {
			result, _ = sjson.Set(result, "spec.ports", cleanedPorts)
		}
	}

	return result, nil
}

func neatDeploymentDefaults(in string) (string, error) {
	var err error
	result := in

	// Remove deployment-specific defaults
	deploymentDefaults := map[string]interface{}{
		"spec.progressDeadlineSeconds": float64(600),
		"spec.revisionHistoryLimit": float64(10),
	}

	for path, defaultValue := range deploymentDefaults {
		if gjson.Get(result, path).Value() == defaultValue {
			result, err = sjson.Delete(result, path)
			if err != nil {
				continue
			}
		}
	}

	// Handle strategy type
	if gjson.Get(result, "spec.strategy.type").String() == "RollingUpdate" {
		result, _ = sjson.Delete(result, "spec.strategy.type")
	}

	return neatPodDefaults(result)
}

func neatPodDefaults(in string) (string, error) {
	var err error
	result := in

	// Clean pod spec defaults (works for pods, deployments, etc.)
	podSpecPath := "spec"
	if gjson.Get(result, "spec.template.spec").Exists() {
		podSpecPath = "spec.template.spec"
	}

	podDefaults := map[string]interface{}{
		podSpecPath + ".restartPolicy": "Always",
		podSpecPath + ".dnsPolicy": "ClusterFirst",
		podSpecPath + ".terminationGracePeriodSeconds": float64(30),
	}

	for path, defaultValue := range podDefaults {
		if gjson.Get(result, path).Value() == defaultValue {
			result, err = sjson.Delete(result, path)
			if err != nil {
				continue
			}
		}
	}

	// Clean containers
	containers := gjson.Get(result, podSpecPath+".containers")
	if containers.IsArray() {
		var cleanedContainers []interface{}
		for _, container := range containers.Array() {
			cleanedContainer := cleanContainer(container.Raw)
			var containerObj interface{}
			if err := json.Unmarshal([]byte(cleanedContainer), &containerObj); err == nil {
				cleanedContainers = append(cleanedContainers, containerObj)
			}
		}

		if len(cleanedContainers) > 0 {
			result, _ = sjson.Set(result, podSpecPath+".containers", cleanedContainers)
		}
	}

	return result, nil
}

func cleanContainer(containerJSON string) string {
	result := containerJSON

	containerDefaults := map[string]interface{}{
		"imagePullPolicy": "Always",
		"terminationMessagePath": "/dev/termination-log",
		"terminationMessagePolicy": "File",
	}

	// Also handle IfNotPresent as default for imagePullPolicy
	if gjson.Get(result, "imagePullPolicy").String() == "IfNotPresent" {
		result, _ = sjson.Delete(result, "imagePullPolicy")
	}

	for path, defaultValue := range containerDefaults {
		if gjson.Get(result, path).Value() == defaultValue {
			result, _ = sjson.Delete(result, path)
		}
	}

	return result
}

func neatCommonDefaults(in string) (string, error) {
	// Handle common defaults for other resource types
	return neatPodDefaults(in)
}
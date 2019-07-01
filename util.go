
package main

import (
	/* Only for utils: */
	"regexp"
)

/* Util functions: */

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func kubeLabelsToPrometheusLabels(labels map[string]string) ([]string, []string) {
	labelKeys := make([]string, len(labels))
	labelValues := make([]string, len(labels))
	i := 0
	for k, v := range labels {
		labelKeys[i] = "label_" + sanitizeLabelName(k)
		labelValues[i] = v
		i++
	}
	return labelKeys, labelValues
}

func sanitizeLabelName(s string) string {
	invalidLabelCharRE        := regexp.MustCompile(`[^a-zA-Z0-9_]`)
  
	return invalidLabelCharRE.ReplaceAllString(s, "_")
}
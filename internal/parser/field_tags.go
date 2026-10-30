// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package parser

import "maps"

type parsedFieldTags struct {
	Default    string
	Metadata   map[string]string
	Modifiers  []string
	Checks     map[string]string
	BareChecks map[string]bool
}

func parseFieldTags(tagStr string) parsedFieldTags {
	var out parsedFieldTags

	if genTag := ParseTag(tagStr, "optgen"); genTag != "" {
		genDefault, genMeta := ParseOptGen(genTag)
		out.Default = genDefault
		if len(genMeta) > 0 {
			out.Metadata = maps.Clone(genMeta)
		}
	}

	if valTag := ParseTag(tagStr, "optval"); valTag != "" {
		out.Modifiers = ParseOptModifiers(valTag)
	}

	out.Checks = parseOptChecks(ParseTag(tagStr, "optcheck"))
	out.BareChecks = bareKeys(smartSplit(ParseTag(tagStr, "optcheck")))

	return out
}

// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/altessa-s/go-optgen/internal/application"
	"github.com/altessa-s/go-optgen/model"
)

// reportingSource reports parsed packages in verbose mode.
type reportingSource struct {
	source  application.Source
	out     io.Writer
	verbose bool
}

func (r reportingSource) Parse(ctx context.Context, dir, name string, all bool) ([]model.SourcePackage, error) {
	packages, err := r.source.Parse(ctx, dir, name, all)
	if err != nil || !r.verbose {
		return packages, err
	}
	_, _ = fmt.Fprintf(r.out, "Found %d package(s)\n", len(packages))
	for _, pkg := range packages {
		_, _ = fmt.Fprintf(r.out, "Processing package: %s\n", pkg.Name)
		if len(pkg.Fields) == 0 {
			_, _ = fmt.Fprintf(r.out, "No opt fields found in package %s\n", pkg.Name)
			continue
		}
		if pkg.GenericInfo.IsGeneric() {
			_, _ = fmt.Fprintf(r.out, "Generic type parameters: %s\n", pkg.GenericInfo.TypeParamsDecl())
		}
		if pkg.NormalizationMethod != "" {
			_, _ = fmt.Fprintf(r.out, "Found normalization method: %s\n", pkg.NormalizationMethod)
		}
	}
	return packages, nil
}

// reportingPreparer reports prepared fields, with their final option names,
// in verbose mode.
type reportingPreparer struct {
	preparer   application.Preparer
	out        io.Writer
	verbose    bool
	funcPrefix string
}

func (r reportingPreparer) Prepare(fields []model.OptField) ([]model.OptField, error) {
	prepared, err := r.preparer.Prepare(fields)
	if err != nil || !r.verbose {
		return prepared, err
	}
	_, _ = fmt.Fprintf(r.out, "Found %d field(s) with opt tags:\n", len(prepared))
	for _, field := range prepared {
		_, _ = fmt.Fprintf(r.out, "  - %s (%s) -> %s%s\n", field.FieldName, field.Type, r.funcPrefix, field.OptionName)
	}
	_, _ = fmt.Fprintln(r.out, "Generating code...")
	return prepared, nil
}

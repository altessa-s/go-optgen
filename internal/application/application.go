// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package application

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/altessa-s/go-optgen/model"
)

// Source reads declarations from a target directory.
type Source interface {
	Parse(context.Context, string, string, bool) ([]model.SourcePackage, error)
}

// Preparer validates declarations and resolves policies.
type Preparer interface {
	Prepare([]model.OptField) ([]model.OptField, error)
}

// Renderer converts prepared declarations to source code.
type Renderer interface {
	Generate(model.GenerateInput) ([]byte, error)
}

// Output formats and publishes generated source.
type Output interface {
	Write(context.Context, string, []byte, string) error
}

// Request is one generation operation, independent of CLI flags and YAML.
// Settings.TypeName selects the struct to generate options for.
type Request struct {
	Directory string
	Output    string
	AllFields bool
	DryRun    bool
	Formatter string
	Settings  model.RenderSettings
}

// Artifact is a rendered file and its option count.
type Artifact struct {
	Path    string
	Content []byte
	Options int
}

// Service coordinates parsing, preparation, rendering, and publication.
type Service struct {
	source   Source
	preparer Preparer
	renderer Renderer
	output   Output
}

// New assembles the use case from required ports.
func New(source Source, preparer Preparer, renderer Renderer, output Output) *Service {
	return &Service{source: source, preparer: preparer, renderer: renderer, output: output}
}

// Generate executes one request without depending on CLI or filesystem APIs.
func (s *Service) Generate(ctx context.Context, req Request) ([]Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	packages, err := s.source.Parse(ctx, req.Directory, req.Settings.TypeName, req.AllFields)
	if err != nil {
		return nil, err
	}
	var artifacts []Artifact
	for _, pkg := range packages {
		if len(pkg.Fields) == 0 {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fields, err := s.preparer.Prepare(pkg.Fields)
		if err != nil {
			return nil, err
		}
		content, err := s.renderer.Generate(model.GenerateInput{
			RenderSettings:      req.Settings,
			PackageName:         pkg.Name,
			Fields:              fields,
			Imports:             pkg.Imports,
			NormalizationMethod: pkg.NormalizationMethod,
			GenericInfo:         pkg.GenericInfo,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate: %w", err)
		}
		artifacts = append(artifacts, Artifact{Path: filepath.Join(req.Directory, req.Output), Content: content, Options: len(fields)})
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("no struct with name '%s' and opt tags found", req.Settings.TypeName)
	}
	if len(artifacts) > 1 {
		return nil, fmt.Errorf("multiple packages contain target struct %q", req.Settings.TypeName)
	}
	if !req.DryRun {
		for _, artifact := range artifacts {
			if err := s.output.Write(ctx, artifact.Path, artifact.Content, req.Formatter); err != nil {
				return nil, err
			}
		}
	}
	return artifacts, nil
}

// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/application"
	"github.com/altessa-s/go-optgen/model"
)

type sourceFunc func(context.Context, string, string, bool) ([]model.SourcePackage, error)

func (f sourceFunc) Parse(c context.Context, d, n string, a bool) ([]model.SourcePackage, error) {
	return f(c, d, n, a)
}

type prepareFunc func([]model.OptField) ([]model.OptField, error)

func (f prepareFunc) Prepare(fields []model.OptField) ([]model.OptField, error) { return f(fields) }

type renderFunc func(model.GenerateInput) ([]byte, error)

func (f renderFunc) Generate(in model.GenerateInput) ([]byte, error) { return f(in) }

type outputFunc func(context.Context, string, []byte, string) error

func (f outputFunc) Write(c context.Context, p string, b []byte, formatter string) error {
	return f(c, p, b, formatter)
}

func TestGenerateUsesPortsAndHonorsDryRun(t *testing.T) {
	t.Parallel()
	for _, dry := range []bool{false, true} {
		t.Run(map[bool]string{false: "write", true: "preview"}[dry], func(t *testing.T) {
			t.Parallel()
			var calls []string
			service := application.New(
				sourceFunc(func(ctx context.Context, dir, name string, all bool) ([]model.SourcePackage, error) {
					require.Equal(t, t.Context(), ctx)
					require.Equal(t, "dir", dir)
					require.Equal(t, "options", name)
					require.True(t, all)
					calls = append(calls, "source")
					return []model.SourcePackage{{Name: "sample", Fields: []model.OptField{{OptionName: "Count"}}}}, nil
				}),
				prepareFunc(func(fields []model.OptField) ([]model.OptField, error) {
					calls = append(calls, "prepare")
					return fields, nil
				}),
				renderFunc(func(in model.GenerateInput) ([]byte, error) {
					calls = append(calls, "render")
					require.Equal(t, "Count", in.Fields[0].OptionName)
					require.Equal(t, "options", in.TypeName)
					require.Equal(t, "sample", in.PackageName)
					return []byte("generated"), nil
				}),
				outputFunc(func(_ context.Context, path string, data []byte, formatter string) error {
					calls = append(calls, "write")
					require.Equal(t, "dir/options_gen.go", path)
					require.Equal(t, "generated", string(data))
					require.Equal(t, "gofmt -w", formatter)
					return nil
				}),
			)
			artifacts, err := service.Generate(t.Context(), application.Request{Directory: "dir", Output: "options_gen.go", AllFields: true, DryRun: dry, Formatter: "gofmt -w", Settings: model.RenderSettings{TypeName: "options"}})
			require.NoError(t, err)
			require.Len(t, artifacts, 1)
			want := []string{"source", "prepare", "render"}
			if !dry {
				want = append(want, "write")
			}
			require.Equal(t, want, calls)
		})
	}
}

func TestGenerateStopsOnPortFailure(t *testing.T) {
	t.Parallel()
	failure := errors.New("source failed")
	service := application.New(sourceFunc(func(context.Context, string, string, bool) ([]model.SourcePackage, error) { return nil, failure }), nil, nil, nil)
	_, err := service.Generate(t.Context(), application.Request{})
	require.ErrorIs(t, err, failure)
}

func TestGenerateReportsMissingAndAmbiguousTargets(t *testing.T) {
	t.Parallel()
	identity := prepareFunc(func(fields []model.OptField) ([]model.OptField, error) { return fields, nil })
	render := renderFunc(func(model.GenerateInput) ([]byte, error) { return []byte("x"), nil })
	source := func(pkgs ...model.SourcePackage) application.Source {
		return sourceFunc(func(context.Context, string, string, bool) ([]model.SourcePackage, error) { return pkgs, nil })
	}
	req := application.Request{Settings: model.RenderSettings{TypeName: "options"}, DryRun: true}
	field := []model.OptField{{OptionName: "A"}}

	_, err := application.New(source(model.SourcePackage{Name: "a"}), identity, render, nil).Generate(t.Context(), req)
	require.ErrorContains(t, err, "no struct with name 'options'")

	_, err = application.New(source(model.SourcePackage{Name: "a", Fields: field}, model.SourcePackage{Name: "b", Fields: field}), identity, render, nil).
		Generate(t.Context(), req)
	require.ErrorContains(t, err, "multiple packages contain target struct")
}

func TestGenerateStopsOnPreparationRenderingAndOutputFailures(t *testing.T) {
	t.Parallel()
	failure := errors.New("failed")
	source := sourceFunc(func(context.Context, string, string, bool) ([]model.SourcePackage, error) {
		return []model.SourcePackage{{Name: "a", Fields: []model.OptField{{OptionName: "A"}}}}, nil
	})
	identity := prepareFunc(func(fields []model.OptField) ([]model.OptField, error) { return fields, nil })
	render := renderFunc(func(model.GenerateInput) ([]byte, error) { return []byte("x"), nil })
	req := application.Request{Settings: model.RenderSettings{TypeName: "options"}}

	_, err := application.New(source, prepareFunc(func([]model.OptField) ([]model.OptField, error) { return nil, failure }), render, nil).Generate(t.Context(), req)
	require.ErrorIs(t, err, failure)
	_, err = application.New(source, identity, renderFunc(func(model.GenerateInput) ([]byte, error) { return nil, failure }), nil).Generate(t.Context(), req)
	require.ErrorIs(t, err, failure)
	_, err = application.New(source, identity, render, outputFunc(func(context.Context, string, []byte, string) error { return failure })).Generate(t.Context(), req)
	require.ErrorIs(t, err, failure)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = application.New(source, identity, render, nil).Generate(ctx, req)
	require.ErrorIs(t, err, context.Canceled)
}

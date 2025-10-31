package jq

import (
	"errors"
	"fmt"

	"github.com/itchyny/gojq"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

// function represents a JQ function with its implementation.
type function struct {
	name     string
	minarity int
	maxarity int
	impl     func(any, []any) any
}

// variable represents a JQ variable with its name and value.
type variable struct {
	name  string
	value any
}

// Engine represents a JQ execution engine.
type Engine struct {
	code      *gojq.Code
	functions []function
	variables []variable
}

// Option is a generic option for Engine.
type Option = util.Option[Engine]

// Options is a struct-based option that can set multiple engine options at once.
type Options struct {
	// Functions are custom JQ functions to register with the engine.
	Functions []function

	// Variables are JQ variables to make available during execution.
	Variables []variable
}

// ApplyTo applies the JQ engine options to the target engine.
func (opts Options) ApplyTo(target *Engine) {
	target.functions = opts.Functions
	target.variables = opts.Variables
}

// WithFunction adds a custom function to the JQ engine.
func WithFunction(name string, minarity, maxarity int, impl func(any, []any) any) Option {
	return util.FunctionalOption[Engine](func(e *Engine) {
		e.functions = append(e.functions, function{
			name:     name,
			minarity: minarity,
			maxarity: maxarity,
			impl:     impl,
		})
	})
}

// WithVariable adds a variable to the JQ engine.
func WithVariable(name string, value any) Option {
	return util.FunctionalOption[Engine](func(e *Engine) {
		// Ensure variable name starts with $
		if name[0] != '$' {
			name = "$" + name
		}

		e.variables = append(e.variables, variable{
			name:  name,
			value: value,
		})
	})
}

// NewEngine creates a new JQ engine with the given expression and options.
func NewEngine(expression string, opts ...Option) (*Engine, error) {
	e := &Engine{
		functions: make([]function, 0),
		variables: make([]variable, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt.ApplyTo(e)
	}

	// Parse the query
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JQ expression: %w", err)
	}

	// Create compiler options for functions
	compilerOpts := make([]gojq.CompilerOption, 0, len(e.functions))
	for _, fn := range e.functions {
		compilerOpts = append(compilerOpts, gojq.WithFunction(fn.name, fn.minarity, fn.maxarity, fn.impl))
	}

	vars := make([]string, len(e.variables))
	for i, v := range e.variables {
		vars[i] = v.name
	}

	compilerOpts = append(compilerOpts, gojq.WithVariables(vars))

	// Compile the query with function options
	code, err := gojq.Compile(query, compilerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JQ expression: %w", err)
	}

	e.code = code

	return e, nil
}

// Run executes the JQ expression on the given input and returns a single value or an error.
// It expects the JQ expression to return exactly one value.
func (e *Engine) Run(input any) (any, error) {
	// Create a map of variables for this execution
	vars := make([]any, len(e.variables))
	for i, v := range e.variables {
		vars[i] = v.value
	}

	// Run the JQ program with variables
	iter := e.code.Run(input, vars...)

	v, ok := iter.Next()
	if !ok {
		return nil, errors.New("jq: no results returned")
	}

	if err, ok := v.(error); ok {
		return nil, fmt.Errorf("jq: error during execution: %w", err)
	}

	// Check if there are more values
	if _, ok := iter.Next(); ok {
		return nil, errors.New("jq: multiple results returned")
	}

	return v, nil
}

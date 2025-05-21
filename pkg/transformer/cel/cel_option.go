package cel

import (
	"github.com/google/cel-go/cel"
)

type Option func(*options)

type options struct {
	envOptions []cel.EnvOption
}

func WithLibrary(lib cel.Library) Option {
	return func(o *options) {
		o.envOptions = append(o.envOptions, cel.Lib(lib))
	}
}

func WithFunction(name string, overloads ...cel.FunctionOpt) Option {
	return func(o *options) {
		o.envOptions = append(o.envOptions, cel.Function(name, overloads...))
	}
}

func WithVariable(name string, typ *cel.Type) Option {
	return func(o *options) {
		o.envOptions = append(o.envOptions, cel.Variable(name, typ))
	}
}

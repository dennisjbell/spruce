package main

import (
	"fmt"
)

// InjectOperator ...
type InjectOperator struct{}

// Setup ...
func (InjectOperator) Setup() error {
	return nil
}

// Phase ...
func (InjectOperator) Phase() OperatorPhase {
	return MergePhase
}

// Dependencies ...
func (InjectOperator) Dependencies(_ *Evaluator, args []*Expr, locs []*Cursor) []*Cursor {
	l := []*Cursor{}

	for _, arg := range args {
		if arg.Type != Reference {
			continue
		}

		for _, other := range locs {
			if other.Under(arg.Reference) {
				l = append(l, other)
			}
		}
	}

	return l
}

// Run ...
func (InjectOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( inject ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( inject ... )) operation at $%s\n", ev.Here)

	var vals []map[interface{}]interface{}

	for i, arg := range args {
		v, err := arg.Resolve(ev.Tree)
		if err != nil {
			DEBUG("  arg[%d]: failed to resolve expression to a concrete value", i)
			DEBUG("     [%d]: error was: %s", i, err)
			return nil, err
		}
		switch v.Type {
		case Literal:
			DEBUG("  arg[%d]: found string literal '%s'", i, v.Literal)
			DEBUG("           (inject operator only handles references to other parts of the YAML tree)")
			return nil, fmt.Errorf("inject operator only accepts key reference arguments")

		case Reference:
			DEBUG("  arg[%d]: trying to resolve reference $.%s", i, v.Reference)
			s, err := v.Reference.Resolve(ev.Tree)
			if err != nil {
				DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
				return nil, err
			}

			m, ok := s.(map[interface{}]interface{})
			if !ok {
				DEBUG("     [%d]: resolved to something that is not a map.  that is unacceptable.", i)
				return nil, fmt.Errorf("%s is not a map", v.Reference)
			}

			DEBUG("     [%d]: resolved to a map; appending to the list of maps to merge/inject", i)
			vals = append(vals, m)

		default:
			DEBUG("  arg[%d]: I don't know what to do with '%v'", i, arg)
			return nil, fmt.Errorf("inject operator only accepts key reference arguments")
		}
		DEBUG("")
	}

	switch len(vals) {
	case 0:
		DEBUG("  no arguments supplied to (( inject ... )) operation.  oops.")
		return nil, fmt.Errorf("no arguments specified to (( inject ... ))")

	default:
		DEBUG("  merging found maps into a single map to be injected")
		merged, err := Merge(vals...)
		if err != nil {
			DEBUG("  failed: %s\n", err)
			return nil, err
		}
		return &Response{
			Type:  Inject,
			Value: merged,
		}, nil
	}
}

func init() {
	RegisterOp("inject", InjectOperator{})
}

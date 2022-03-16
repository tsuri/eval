// Implementation of Evaluation Graphs

package egraph

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"go.starlark.net/starlark"
)

func Load() {

	type entry struct {
		globals starlark.StringDict
		err     error
	}

	cache := make(map[string]*entry)

	var load func(_ *starlark.Thread, module string) (starlark.StringDict, error)
	load = func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
		e, ok := cache[module]
		if e == nil {
			if ok {
				// request for package whose loading is in progress
				return nil, fmt.Errorf("cycle in load graph")
			}

			// Add a placeholder to indicate "load in progress".
			cache[module] = nil

			// Load and initialize the module in a new thread.
			//data := fakeFilesystem[module]
			thread := &starlark.Thread{Name: "exec " + module, Load: load}
			globals, err := starlark.ExecFile(thread, module, nil, nil)
			e = &entry{globals, err}

			// Update the cache.
			cache[module] = e
		}
		fmt.Printf("========[ %v ]===========\n", module)
		fmt.Println(e.globals["_components"])
		fmt.Printf("f: %v\n", e.globals["f"])
		fmt.Printf("m: %v\n", e.globals["m"])
		fmt.Printf("c: %v\n", e.globals["c"])
		fmt.Println("======================")

		return e.globals, e.err
	}

	_, err := load(nil, "evaluations/graphs/comparison.star")
	if err != nil {
		log.Fatal(err)
	}

}

func loadEvalDefnition() {
	const data = `
print(greeting + ", beautiful world")
print(repeat("one"))
print(repeat("mur", 2))
squares = [x*x for x in range(10)]
`

	// repeat(str, n=1) is a Go function called from Starlark.
	// It behaves like the 'string * int' operation.
	repeat := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var s string
		var n int = 1
		if err := starlark.UnpackArgs(b.Name(), args, kwargs, "s", &s, "n?", &n); err != nil {
			return nil, err
		}
		return starlark.String(strings.Repeat(s, n)), nil
	}

	// The Thread defines the behavior of the built-in 'print' function.
	thread := &starlark.Thread{
		Name:  "example",
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
		Load: func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
			fmt.Println("Loading %v", module)
			return nil, nil
		},
	}

	// This dictionary defines the pre-declared environment.
	predeclared := starlark.StringDict{
		"components": starlark.NewList(nil),
		"greeting":   starlark.String("hello"),
		"repeat":     starlark.NewBuiltin("repeat", repeat),
	}

	// Execute a program.
	fmt.Println("------------------")
	globals, err := starlark.ExecFile(thread, "evaluations/graphs/comparison.star", nil, predeclared)
	//	globals, err := starlark.ExecFile(thread, "apparent/filename1.star", data, predeclared)
	fmt.Println("done exec")
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			log.Fatal(evalErr.Backtrace())
		}
		log.Fatal(err)
	}

	// var load func(_ *starlark.Thread, module string) (starlark.StringDict, error)
	// load = func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
	// 	e, ok := cache[module]
	// 	if e == nil {
	// 		if ok {
	// 			// request for package whose loading is in progress
	// 			return nil, fmt.Errorf("cycle in load graph")
	// 		}

	// 		// Add a placeholder to indicate "load in progress".
	// 		cache[module] = nil

	// 		// Load and initialize the module in a new thread.
	// 		data := fakeFilesystem[module]
	// 		thread := &starlark.Thread{Name: "exec " + module, Load: load}
	// 		globals, err := starlark.ExecFile(thread, module, data, nil)
	// 		e = &entry{globals, err}

	// 		// Update the cache.
	// 		cache[module] = e
	// 	}
	// 	return e.globals, e.err
	// }
	// gobals, err = load(nil, "graphs/ab_comparison.star")

	fmt.Println("------------------")
	files, err := ioutil.ReadDir("./")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Println(f.Name())
	}
	graph, err := os.ReadFile("evaluations/graphs/comparison.star")
	if err != nil {
		fmt.Println("Failure ", err)
	} else {
		fmt.Println("Success")
		fmt.Println(string(graph))
	}

	// Print the global environment.
	fmt.Println("\nGlobals:")
	for _, name := range globals.Keys() {
		v := globals[name]
		fmt.Printf("%s (%s) = %s\n", name, v.Type(), v.String())
	}

}

func ImportEvaluationGraph() bool {

	//	loadEvalDefnition()
	Load()
	return true
}

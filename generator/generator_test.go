package generator_test

import (
	"bytes"
	"io"
	"regexp"
	"testing"

	"github.com/susji/typestringer/generator"
	"golang.org/x/exp/slices"
)

type MultiBuffer struct {
	b       bytes.Buffer
	history []string
}

func (cb *MultiBuffer) Close() error {
	cb.history = append(cb.history, cb.b.String())
	cb.b = bytes.Buffer{}
	return nil
}

func (cb *MultiBuffer) Write(p []byte) (int, error) {
	return cb.b.Write(p)
}

func (cb *MultiBuffer) String() string {
	return cb.b.String()
}

func TestOne(t *testing.T) {
	base := &generator.Generator{
		Patterns: []string{"./testdata/one"},
		Includes: nil,
		Ignores:  nil,
		Format:   "%s,%s\n",
		Header:   "// the header\n",
		Preamble: `import (
    "fmt"
    "os"
)`,
	}
	t.Run("accept all", func(t *testing.T) {
		cb := &MultiBuffer{}
		g := *base
		g.WriteCloserCreator = func(path, mod string) (io.WriteCloser, error) {
			return cb, nil
		}
		if err := g.Generate(); err != nil {
			t.Error(err)
		}
		want := []string{
			`// the header
package one

import (
    "fmt"
    "os"
)

Int,Int
String,String
Struct,Struct
`}
		if !slices.Equal(want, cb.history) {
			t.Error(cb.history)
		}
	})
	t.Run("ignore some", func(t *testing.T) {
		cb := &MultiBuffer{}
		g := *base
		g.WriteCloserCreator = func(path, mod string) (io.WriteCloser, error) {
			return cb, nil
		}
		g.Ignores = []*regexp.Regexp{
			regexp.MustCompile("ring"),
			regexp.MustCompile("^Struct$"),
		}
		if err := g.Generate(); err != nil {
			t.Error(err)
		}
		want := []string{
			`// the header
package one

import (
    "fmt"
    "os"
)

Int,Int
`}
		if !slices.Equal(want, cb.history) {
			t.Error(cb.history)
		}
	})
	t.Run("include and ignore", func(t *testing.T) {
		cb := &MultiBuffer{}
		g := *base
		g.WriteCloserCreator = func(path, mod string) (io.WriteCloser, error) {
			return cb, nil
		}
		// Ignore should take precedence if both match.
		g.Includes = []*regexp.Regexp{
			regexp.MustCompile("String"),
			regexp.MustCompile("^Int$"),
		}
		g.Ignores = []*regexp.Regexp{
			regexp.MustCompile("String"),
		}
		if err := g.Generate(); err != nil {
			t.Error(err)
		}
		want := []string{
			`// the header
package one

import (
    "fmt"
    "os"
)

Int,Int
`}
		if !slices.Equal(want, cb.history) {
			t.Error(cb.history)
		}
	})
}

func TestTwo(t *testing.T) {
	base := &generator.Generator{
		Patterns: []string{"./testdata/two/two1.go", "./testdata/two/two2.go"},
		Includes: nil,
		Ignores:  nil,
		Format:   "%s,%s\n",
	}
	t.Run("only one file", func(t *testing.T) {
		cb := &MultiBuffer{}
		g := *base
		g.Patterns = g.Patterns[:1]
		g.WriteCloserCreator = func(path, mod string) (io.WriteCloser, error) {
			return cb, nil
		}
		if err := g.Generate(); err != nil {
			t.Error(err)
		}
		want := []string{
			`package two

FIRST,FIRST
`}
		if !slices.Equal(want, cb.history) {
			t.Error(cb.history)
		}
	})
	t.Run("both files file", func(t *testing.T) {
		called := false
		cb := &MultiBuffer{}
		g := *base
		g.WriteCloserCreator = func(path, mod string) (io.WriteCloser, error) {
			if called {
				t.Error("called more than once")
			}
			called = true
			return cb, nil
		}
		if err := g.Generate(); err != nil {
			t.Error(err)
		}
		want := []string{
			`package two

FIRST,FIRST
SECOND,SECOND
`}
		if !slices.Equal(want, cb.history) {
			t.Error(cb.history)
		}
	})
}

func TestThree(t *testing.T) {
	base := &generator.Generator{
		Patterns: []string{"./testdata/three/threeone", "./testdata/three/threetwo"},
		Includes: nil,
		Ignores:  nil,
		Format:   "%s,%s\n",
	}
	t.Run("both subpackages", func(t *testing.T) {
		cb := &MultiBuffer{}
		g := *base
		g.WriteCloserCreator = func(path, mod string) (io.WriteCloser, error) {
			return cb, nil
		}
		if err := g.Generate(); err != nil {
			t.Error(err)
		}
		want := []string{`package threeone

THREEONE,THREEONE
`,
			`package threetwo

THREETWO,THREETWO
`}
		if !slices.Equal(want, cb.history) {
			t.Error(want, cb.history)
		}
	})
}

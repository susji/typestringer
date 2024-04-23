package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/tools/go/packages"
)

// Generator contains the configuration for String() generation based on
// package's type names.
type Generator struct {
	// Patterns passed to packages.Load.
	Patterns []string
	// List of regular expressions to determine which types are included.
	// Empty list means to include all types by default.
	Includes []*regexp.Regexp
	// List of regular expressions to determine which types are ignored.
	// Ignores takes precedence over Inludes.
	Ignores []*regexp.Regexp
	// Format string for writing out the type's String() receiver.
	Format string
	// Function used to create the output Writer for generated files. If
	// left empty, the generated output is directed to a file in the target
	// package with its filename determined by FormatFilename.
	WriteCloserCreator WriteCloserCreator
	// Determines where generated output is written. If set to nil,
	// WriteCloserCreator is used.
	Output io.WriteCloser
	// Determines where Generator diagnostic output is written. If set to
	// nil, os.Stderr will be used. If output should be discarded, something
	// like io.Discard may be used.
	DiagnosticOutput io.Writer
	// If set true, the output stream will not be closed after package's
	// code generation.
	NoClose bool
	// Format string for writing out the header of generated files. The
	// format operand is the package name. May be left empty.
	Header string
	// String to write after the generated file's package has been declared
	// and before the type-specific part begins. Useful for declaring things
	// such as imports.
	Preamble string
	// If set true, generation will not output "package <name>".
	NoPackage bool
}
type WriteCloserCreator func(filepath string, module string) (io.WriteCloser, error)

func (g *Generator) Generate() error {
	if g.DiagnosticOutput == nil {
		g.DiagnosticOutput = os.Stderr
	}
	if g.WriteCloserCreator == nil {
		g.WriteCloserCreator = g.defaultwg
	}
	cfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax,
	}
	ps, err := packages.Load(cfg, g.Patterns...)
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		fmt.Fprintln(g.DiagnosticOutput, "no packages loaded")
		return errors.New("no packages loaded")
	}
	var reterr error
	for i, p := range ps {
		fmt.Fprintln(g.DiagnosticOutput, "package with pattern", g.Patterns[i])
		if len(p.Errors) > 0 {
			fmt.Fprintln(g.DiagnosticOutput, "found package errors, not continuing")
			for _, err := range p.Errors {
				reterr = errors.Join(reterr, err)
				fmt.Fprintln(g.DiagnosticOutput, err)
			}
			continue
		}
		if err := g.HandlePackage(p); err != nil {
			fmt.Fprintln(g.DiagnosticOutput, "generate error:", err)
			reterr = errors.Join(reterr, err)
		}
	}
	return reterr
}

func (g *Generator) defaultwg(path, mod string) (io.WriteCloser, error) {
	fn := filepath.Join(path, fmt.Sprintf(FormatFilename, mod))
	w, err := os.Create(fn)
	if err != nil {
		fmt.Fprintln(g.DiagnosticOutput, err)
		return nil, err
	}
	fmt.Fprintln(g.DiagnosticOutput, "writing file:", fn)
	return w, nil
}

func (g *Generator) HandlePackage(p *packages.Package) error {
	if len(p.GoFiles) == 0 {
		return errors.New("no Go files in package")
	}
	typenames := []string{}
	var packagename string
	for _, a := range p.Syntax {
		packagename = a.Name.Name
		for _, decl := range a.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if gd.Tok != token.TYPE {
				continue
			}
		decl:
			for _, sp := range gd.Specs {
				ts := sp.(*ast.TypeSpec)
				tn := ts.Name.Name
				for _, r := range g.Ignores {
					if r.MatchString(tn) {
						fmt.Fprintln(g.DiagnosticOutput, "ignoring:", tn)
						continue decl
					}
				}
				if len(g.Includes) > 0 {
					found := false
					for _, r := range g.Includes {
						if r.MatchString(tn) {
							found = true
							break
						}
					}
					if !found {
						fmt.Fprintln(g.DiagnosticOutput, "not included:", tn)
						continue
					}
				}
				fmt.Fprintln(g.DiagnosticOutput, "including:", tn)
				typenames = append(typenames, ts.Name.Name)
			}
		}
	}
	var w io.WriteCloser
	if g.Output != nil {
		w = g.Output
	} else {
		var err error
		w, err = g.WriteCloserCreator(filepath.Dir(p.GoFiles[0]), packagename)
		if err != nil {
			return err
		}
		if w == nil {
			panic(errors.New("nil WriteCloser"))
		}
	}
	if len(g.Header) > 0 {
		fmt.Fprint(w, g.Header)
	}
	if !g.NoPackage {
		fmt.Fprintf(w, "package %s\n\n", packagename)
	}
	if len(g.Preamble) > 0 {
		fmt.Fprint(w, g.Preamble, "\n\n")
	}
	for _, tn := range typenames {
		fmt.Fprintf(w, g.Format, tn, tn)
	}
	if !g.NoClose {
		w.Close()
	}
	return nil
}

var (
	// Format string for writing out the type-specific receiver. May of
	// course be set to something completely different. The formatted
	// operands are the type name passed twice.
	FormatReceiver = "func (t %s) String() string { return \"%s\" }\n"
	// Format string for determining the generated filenames.
	FormatFilename = "%s_strings.go"
)

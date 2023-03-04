package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

const (
	REQS_PATH     = "requirements.txt"
	DEV_REQS_PATH = "dev-requirements.txt"
)

type PackageSpec struct {
	Name   string
	Extras string
}

// pip show schema
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

var (
	extrasRgx *regexp.Regexp
)

func main() {
	// TODO: Match python spec
	extrasRgx = regexp.MustCompile(`\[(?:[^\d\W]\w*)(?:,[^\d\W]\w*)*\]`)

	app := &cli.App{
		Name:  "pips",
		Usage: "install pip packages and add to requirements.txt",
		Commands: []*cli.Command{
			{
				Name:    "install",
				Aliases: []string{"i"},
				Usage:   "installs a package and updates " + REQS_PATH,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "dev",
						Aliases: []string{"d"},
						Value:   false,
						Usage:   "save to " + DEV_REQS_PATH,
					},
				},
				Action: InstallCmd,
			},
			{
				Name:    "remove",
				Aliases: []string{"rm"},
				Usage:   "uninstalls a package",
				Action: func(ctx *cli.Context) error {
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func InstallCmd(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("got 0 arguments")
	}

	// Install packages

	args := []string{"-m", "pip", "install"}
	args = append(args, ctx.Args().Slice()...)

	stdout, stderr, err := RunCommand("python", args...)
	if err != nil {
		return err
	}
	if stderr.Len() > 0 {
		return fmt.Errorf(stderr.String())
	}

	io.Copy(os.Stdout, &stdout) // Log pip install output

	// Add to [dev-]requirements.txt

	ps := []PackageSpec{}

	for _, arg := range ctx.Args().Slice() {
		name, extras := arg, ""

		if strings.Contains(name, "[") {
			name = extrasRgx.ReplaceAllLiteralString(arg, "")
		}

		if s := extrasRgx.FindString(arg); s != "" {
			extras = s
		}

		ps = append(ps, PackageSpec{name, extras})
	}

	stdout, stderr, err = RunCommand("python", "-m", "pip", "list", "--format=json")
	if err != nil {
		return err
	}
	if stderr.Len() > 0 {
		return fmt.Errorf(stderr.String())
	}

	pipShowPkgs := make([]Package, 0)
	decoder := json.NewDecoder(&stdout)

	err = decoder.Decode(&pipShowPkgs)
	if err != nil {
		return err
	}

	path := REQS_PATH
	if ctx.Bool("dev") {
		path = DEV_REQS_PATH
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	for _, p := range ps {
		normalized := NormalizePkgName(p.Name)

		idx, ok := slices.BinarySearchFunc(pipShowPkgs, normalized, func(p Package, name string) int {
			return Cmp(p.Name, name)
		})

		if !ok {
			fmt.Printf("package %s not found\n", normalized)
			continue
		}

		_, err := f.WriteString(fmt.Sprintf("%s%s==%s\n", normalized, p.Extras, pipShowPkgs[idx].Version))
		if err != nil {
			fmt.Printf("could not add %s to %s: %v\n", normalized, path, err)
			continue
		}
	}

	return nil
}

// RunCommand runs exec.Command
// and returns buffers containing stdout and stderr.
func RunCommand(name string, arg ...string) (stdout, stderr bytes.Buffer, err error) {
	cmd := exec.Command("python", arg...)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return
	}

	return
}

// NormalizePkgName attempts to normalize
// a PyPi package name.
// TODO: Follow spec
func NormalizePkgName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}

func Cmp[T constraints.Ordered](a, b T) int {
	if a == b {
		return 0
	} else if a > b {
		return 1
	} else {
		return -1
	}
}

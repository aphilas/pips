package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

const (
	REQS_PATH     = "requirements.txt"
	DEV_REQS_PATH = "dev-requirements.txt"
)

type PackageSpec struct {
	Name    string `json:"name"`
	Extras  string
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
				Name:    "uninstall",
				Aliases: []string{"remove", "rm", "u"},
				Usage:   "uninstalls a package",
				Action:  UninstallCmd,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		// log.Fatal(err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func InstallCmd(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("got 0 arguments")
	}

	// Install packages

	args := []string{"-m", "pip", "install"}
	args = append(args, ctx.Args().Slice()...)

	cmd := exec.Command("python", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	// Add to [dev-]requirements.txt

	pkgExtras := map[string]string{}
	pkgs := []string{}
	for _, arg := range ctx.Args().Slice() {
		if strings.Contains(arg, "[") {
			name := NormalizePkgName(extrasRgx.ReplaceAllLiteralString(arg, ""))
			pkgs = append(pkgs, name)
			pkgExtras[name] = extrasRgx.FindString(arg)
		} else {
			pkgs = append(pkgs, arg)
		}
	}

	inspection, err := PipInspect()
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

	for _, p := range pkgs {
		var item *InspectReportItem

		// TODO: Improve perf
		for _, it := range inspection.Installed {
			if NormalizePkgName(it.Metadata.Name) == p {
				item = &it
			}
		}

		if item == nil {
			fmt.Printf("package %s not found\n", p)
			continue
		}

		extras := ""
		if v, ok := pkgExtras[p]; ok {
			extras = v
		} else if len(item.Metadata.ProvidesExtra) > 0 {
			extras = fmt.Sprintf("[%s]", strings.Join(item.Metadata.ProvidesExtra, ","))
		}

		_, err := f.WriteString(fmt.Sprintf("%s%s==%s\n", p, extras, item.Metadata.Version))
		if err != nil {
			fmt.Printf("could not add %s to %s: %v\n", p, path, err)
			continue
		}
	}

	return nil
}

func UninstallCmd(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("got 0 arguments")
	}

	// Uninstall packages

	args := []string{"-m", "pip", "uninstall"}
	args = append(args, ctx.Args().Slice()...)

	cmd := exec.Command("python", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// PipInsect runs pip inspect
// and decodes the output.
func PipInspect() (*PipInspection, error) {
	var stdout bytes.Buffer

	cmd := exec.Command("python", "-m", "pip", "inspect")
	// cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	pi := new(PipInspection)
	dec := json.NewDecoder(&stdout)
	if err := dec.Decode(pi); err != nil {
		return nil, err
	}

	return pi, nil
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

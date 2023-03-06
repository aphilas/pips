package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// TODO: Match python spec - https://peps.python.org/pep-0508/#names
	// See name regexp - https://peps.python.org/pep-0508/#names
	extrasRgx    *regexp.Regexp = regexp.MustCompile(`\[(?:[^\d\W]\w*)(?:,[^\d\W]\w*)*\]`)
	normalizeRgx *regexp.Regexp = regexp.MustCompile(`[-_.]+`)
)

func main() {
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
				Action: installCmd,
			},
			{
				Name:    "save",
				Aliases: []string{"s"},
				Usage:   "saves an installed package to " + REQS_PATH,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "dev",
						Aliases: []string{"d"},
						Value:   false,
						Usage:   "save to " + DEV_REQS_PATH,
					},
				},
				Action: saveCmd,
			},
			{
				Name:    "uninstall",
				Aliases: []string{"remove", "rm", "u"},
				Usage:   "uninstalls a package and updates " + REQS_PATH,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "dev",
						Aliases: []string{"d"},
						Value:   false,
						Usage:   "remove from " + DEV_REQS_PATH,
					},
				},
				Action: uninstallCmd,
			},
			{
				Name:    "delete",
				Aliases: []string{"d"},
				Usage:   "deletes a package from " + REQS_PATH,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "dev",
						Aliases: []string{"d"},
						Value:   false,
						Usage:   "delete from " + DEV_REQS_PATH,
					},
				},
				Action: unsaveCmd,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		// log.Fatal(err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func installCmd(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("got 0 arguments")
	}

	args := ctx.Args().Slice()
	execArgs := []string{"-m", "pip", "install"}
	execArgs = append(execArgs, args...)

	cmd := exec.Command("python", execArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	path := REQS_PATH
	if ctx.Bool("dev") {
		path = DEV_REQS_PATH
	}

	inspection, err := pipInspect()
	if err != nil {
		return err
	}

	pkgs := parseArgs(args)
	if err := savePkgs(path, inspection, pkgs); err != nil {
		fmt.Println(err)
	}

	return nil
}

func saveCmd(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("got 0 arguments")
	}

	path := REQS_PATH
	if ctx.Bool("dev") {
		path = DEV_REQS_PATH
	}

	inspection, err := pipInspect()
	if err != nil {
		return err
	}

	pkgs := parseArgs(ctx.Args().Slice())
	if err := savePkgs(path, inspection, pkgs); err != nil {
		return err
	}

	return nil
}

// TODO: Skip truncating requirements file
// if uninstall fails
func uninstallCmd(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("got 0 arguments")
	}

	args := []string{"-m", "pip", "uninstall"}
	args = append(args, ctx.Args().Slice()...)

	cmd := exec.Command("python", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	path := REQS_PATH
	if ctx.Bool("dev") {
		path = DEV_REQS_PATH
	}

	pkgs := parseArgs(ctx.Args().Slice())
	if err := removePkgs(path, pkgs); err != nil {
		return err
	}

	return nil
}

func unsaveCmd(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("got 0 arguments")
	}

	path := REQS_PATH
	if ctx.Bool("dev") {
		path = DEV_REQS_PATH
	}

	pkgs := parseArgs(ctx.Args().Slice())
	if err := removePkgs(path, pkgs); err != nil {
		return err
	}

	return nil
}

// pipInsect runs pip inspect
// and decodes the output.
func pipInspect() (*PipInspection, error) {
	var stdout bytes.Buffer

	// Run pip inspect. Ignore stderr and block stdin.
	cmd := exec.Command("python", "-m", "pip", "inspect")
	cmd.Stdout = &stdout

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

// parseArgs parses a list of requirement specifiers
// returning a map of normalizedName -> extras.
func parseArgs(specifiers []string) map[string]string {
	pkgs := map[string]string{}

	for _, arg := range specifiers {
		name, extras, _ := parseSpecifier(arg)
		pkgs[name] = extras
	}

	return pkgs
}

// parseSpecifier parses a requirement specifier
// in the format packageName[extra1,extra2]==0.0.1
// naively to return package-name,[extras],versionspecifier.
// TODO: Parse version specifiers.
// See PEP508 - https://peps.python.org/pep-0508/.
func parseSpecifier(s string) (string, string, string) {
	var name, extras, version string

	if s == "" {
		return "", "", ""
	}

	splt := strings.SplitN(s, "==", 2)
	if len(splt) > 1 {
		version = splt[1]
	}

	ne := splt[0]
	if strings.Contains(ne, "[") {
		loc := extrasRgx.FindStringIndex(s)
		if loc != nil {
			extras = ne[loc[0]:loc[1]]
			name = normalizePkgName(ne[:loc[0]])
		}
	} else {
		name = normalizePkgName(ne)
	}

	return name, extras, version
}

// removePkgs deletes all lines in the requirements file specified by path
// which are requirements specifications for any package in pkgs.
func removePkgs(path string, pkgs map[string]string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	var buf bytes.Buffer
	var e error
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		name, _, _ := parseSpecifier(line)
		if _, ok := pkgs[name]; !ok {
			_, err := buf.Write(scanner.Bytes())
			if err != nil {
				e = errors.Join(err, err)
				continue
			}
			_, err = buf.WriteString("\n")
			if err != nil {
				e = errors.Join(err, err)
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Join(err, e)
	}

	f.Truncate(0)
	f.Seek(0, io.SeekStart)
	_, err = io.Copy(f, &buf)
	if err != nil {
		return errors.Join(err, e)
	}

	return e
}

// savePkgs adds pkgs to a requirements file.
func savePkgs(path string, inspection *PipInspection, pkgs map[string]string) error {
	var e error

	inspHash := make(map[string]*InspectReportItem)
	for i, it := range inspection.Installed {
		inspHash[normalizePkgName(it.Metadata.Name)] = &inspection.Installed[i]
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(e)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	dups := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		name, _, _ := parseSpecifier(line)
		if _, ok := pkgs[name]; ok {
			dups[name] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Join(err, err)
	}

	for p, extras := range pkgs {
		item, ok := inspHash[p]
		if !ok {
			e = errors.Join(e, fmt.Errorf("package %s not found", p))
			continue
		}

		if dup := dups[p]; dup {
			e = errors.Join(e, fmt.Errorf("package %s duplicate found", p))
		} else {
			_, err := f.WriteString(fmt.Sprintf("%s%s==%s\n", p, extras, item.Metadata.Version))
			if err != nil {
				errors.Join(err, fmt.Errorf("could not add %s to %s: %v", p, path, err))
				continue
			}
		}
	}

	return e
}

// normalizePkgName normalizes a package name.
// See - https://peps.python.org/pep-0503/#normalized-names.
// Regex test - https://go.dev/play/p/4SazgOPaJmm
func normalizePkgName(name string) string {
	return strings.ToLower(normalizeRgx.ReplaceAllLiteralString(name, "-"))
}

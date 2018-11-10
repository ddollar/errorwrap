package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		if strings.HasPrefix(path, "vendor/") {
			return nil
		}

		fmt.Printf("processing: %s\n", path)

		if err := wrapFile(path); err != nil {
			return err
		}

		return nil
	})
}

func wrapFile(path string) error {
	// if path != "context.go" {
	//   return nil
	// }

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		wl, err := wrapLine(line)
		if err != nil {
			return err
		}

		lines[i] = wl

		// fmt.Printf("%d: %s\n", i, lines[i])
	}

	if err := ioutil.WriteFile(path, []byte(strings.Join(lines, "\n")), info.Mode()); err != nil {
		return err
	}

	if err := exec.Command("goimports", "-w", path).Run(); err != nil {
		return err
	}

	return nil
}

func wrapLine(line string) (string, error) {
	trim := strings.TrimSpace(line)

	if !strings.HasPrefix(trim, "return ") {
		return line, nil
	}

	args := tokenizeArgs(strings.TrimPrefix(trim, "return "))

	// args := strings.Split(strings.TrimPrefix(trim, "return "), ", ")

	for i, arg := range args {
		if wrappable(arg) {
			args[i] = fmt.Sprintf("errors.WithStack(%s)", arg)
		}
	}

	parts := strings.Split(line, "return ")

	return fmt.Sprintf("%sreturn %s", parts[0], strings.Join(args, ", ")), nil
}

func tokenizeArgs(args string) []string {
	tokens := []string{""}
	i := 0
	depth := 0

	for _, r := range args {
		if r == ',' && depth == 0 {
			tokens = append(tokens, "")
			i += 1
			continue
		}

		if r == '(' {
			depth += 1
		}

		if r == ')' {
			depth -= 1
		}

		tokens[i] += string(r)
	}

	for i := range tokens {
		tokens[i] = strings.TrimSpace(tokens[i])
	}

	return tokens
}

func wrappable(arg string) bool {
	if arg == "err" {
		return true
	}

	if strings.HasPrefix(arg, "errors.New") {
		return true
	}

	if strings.HasPrefix(arg, "fmt.Errorf") {
		return true
	}

	if strings.HasPrefix(arg, "log.Error") {
		return true
	}

	return false
}

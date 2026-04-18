package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("mk_migration", flag.ContinueOnError)
	fs.SetOutput(stdout)

	name := fs.String("name", "", "migration name, e.g. bootstrap_migration")
	dbInstance := fs.String("db-instance", "", "database instance name, e.g. test")
	all := fs.Bool("all", false, "create files for mysql/postgres/dm")

	if err := fs.Parse(args); err != nil {
		return err
	}

	migrationName := strings.TrimSpace(*name)
	if migrationName == "" {
		return errors.New("usage: go run ./cmd/mk_migration --db-instance test --name bootstrap_migration --all")
	}
	instanceName := strings.TrimSpace(*dbInstance)
	if instanceName == "" {
		return errors.New("--db-instance is required")
	}
	if !*all {
		return errors.New("--all is required in this version")
	}

	dialects := []string{"mysql", "postgres", "dm"}
	next, err := nextVersion(filepath.Join("migrations", instanceName, "postgres"))
	if err != nil {
		return fmt.Errorf("get next version failed: %w", err)
	}

	for _, d := range dialects {
		dir := filepath.Join("migrations", instanceName, d)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}

		up := filepath.Join(dir, fmt.Sprintf("%s_%s.up.sql", next, migrationName))
		down := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", next, migrationName))

		if err := writeFile(up, "-- write your UP migration here\n"); err != nil {
			return err
		}
		if err := writeFile(down, "-- write your DOWN migration here\n"); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "created:", up)
		fmt.Fprintln(stdout, "created:", down)
	}

	return nil
}

func nextVersion(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	re := regexp.MustCompile(`^(\d{6})_.*\.up\.sql$`)
	var nums []int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := re.FindStringSubmatch(e.Name())
		if len(m) != 2 {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return "", err
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return "000001", nil
	}
	sort.Ints(nums)
	return fmt.Sprintf("%06d", nums[len(nums)-1]+1), nil
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

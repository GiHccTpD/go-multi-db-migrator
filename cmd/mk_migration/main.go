package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

var (
	name = flag.String("name", "", "migration name, e.g. bootstrap_migration")
	all  = flag.Bool("all", false, "create files for mysql/postgres/dm")
)

func main() {
	flag.Parse()

	if *name == "" {
		fmt.Println("usage: go run ./cmd/mk_migration --name bootstrap_migration --all")
		os.Exit(1)
	}

	dialects := []string{"mysql", "postgres", "dm"}
	if !*all {
		fmt.Println("--all is required in this version")
		os.Exit(1)
	}

	next, err := nextVersion("migrations/postgres")
	if err != nil {
		fmt.Printf("get next version failed: %v\n", err)
		os.Exit(1)
	}

	for _, d := range dialects {
		dir := filepath.Join("migrations", d)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}

		up := filepath.Join(dir, fmt.Sprintf("%s_%s.up.sql", next, *name))
		down := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", next, *name))

		mustWrite(up, "-- write your UP migration here\n")
		mustWrite(down, "-- write your DOWN migration here\n")
		fmt.Println("created:", up)
		fmt.Println("created:", down)
	}
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

func mustWrite(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		panic(err)
	}
}
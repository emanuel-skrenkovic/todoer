package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const goExtension = ".go"

type Todo struct {
	filePath string
	content  string
}

func findGoFilePaths(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	goFilePaths := make([]string, 0)

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				continue
			}

			foundFiles, err := findGoFilePaths(info.Name())
			if err != nil {
				return nil, err
			}

			goFilePaths = append(goFilePaths, foundFiles...)
		} else if strings.HasSuffix(info.Name(), goExtension) {
			goFilePaths = append(goFilePaths, info.Name())
		}
	}

	return goFilePaths, nil
}

func getFileTodos(path string) ([]Todo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	// TODO: read file line-by-line and
	// create todos.

	todos := make([]Todo, 0)

	insideComment := false

	rd := bufio.NewReader(f)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Fatalf("read file line error: %v", err)
			break
		}

		/*
		  TODO: clear up this crap
		*/

		trimmedLine := strings.TrimSpace(line)

		if !insideComment {
			insideComment = strings.HasPrefix(trimmedLine, "/*")
		}

		shortComment := false
		if !insideComment {
			shortComment = strings.HasPrefix(trimmedLine, "//")
		}

		if (insideComment || shortComment) && strings.Contains(line, "TODO") {
			todo := Todo{content: line}
			todos = append(todos, todo)
		}

		if strings.HasSuffix(trimmedLine, "*/") {
			insideComment = false
		}
	}

	return todos, nil
}

func main() {
	path := os.Args[1]
	paths, err := findGoFilePaths(path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	todos := make([]Todo, 0)

	for _, path := range paths {
		fmt.Println(path)

		t, err := getFileTodos(path)
		if err != nil {
			fmt.Printf("error occurred getting todos for file %s: %s", path, err.Error())
			continue
		}

		todos = append(todos, t...)
	}

	for _, todo := range todos {
		fmt.Println("===========================")
		fmt.Printf("Found TODO:\n")
		fmt.Printf("Content:\n")
		fmt.Println(todo.content)
		fmt.Println()
	}
}

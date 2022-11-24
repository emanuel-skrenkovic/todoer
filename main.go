package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const goExtension = ".go"

type Todo struct {
	filePath  string
	content   string
	lineStart int
	lineEnd   int
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

			foundFiles, err := findGoFilePaths(filepath.Join(path, info.Name()))
			if err != nil {
				return nil, err
			}

			goFilePaths = append(goFilePaths, foundFiles...)

		} else if strings.HasSuffix(info.Name(), goExtension) {
			goFilePaths = append(goFilePaths, filepath.Join(path, info.Name()))
		}
	}

	return goFilePaths, nil
}

func getFileTodos(path string) ([]Todo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// TODO: single line TODO example

	defer f.Close()

	// TODO: multi line todo with
	// double slashes.

	todos := make([]Todo, 0)

	insideComment := false
	insideTodo := false

	var current *Todo

	rd := bufio.NewReader(f)

	lineNum := 1
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Fatalf("read file line error: %v", err)
			break
		}

		if !insideTodo && current != nil {
			current.lineEnd = lineNum - 1
			todos = append(todos, *current)
			current = nil
		}

		/* TODO: single line comment with slash star */

		/*
		   TODO: multi line comment with
		   slash star
		*/

		trimmedLine := strings.TrimSpace(line)

		if !insideComment {
			insideComment = strings.HasPrefix(trimmedLine, "/*")
		}

		shortComment := false
		if !insideComment {
			shortComment = strings.HasPrefix(trimmedLine, "//")
			if !shortComment {
				insideTodo = false
			}
		}

		if (insideComment || shortComment) && strings.Contains(line, "TODO") {
			insideTodo = true
			current = &Todo{filePath: path, lineStart: lineNum}
		}

		if insideTodo {
			current.content += line
		}

		if strings.HasSuffix(trimmedLine, "*/") {
			insideComment = false
			insideTodo = false
		}

		lineNum += 1
	}

	if current != nil {
		current.lineEnd = lineNum - 1
		todos = append(todos, *current)
		current = nil
	}

	return todos, nil
}

func main() {
	path := os.Args[1]
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	paths, err := findGoFilePaths(absolutePath)
	if err != nil {
		log.Fatal(err)
	}

	todos := make([]Todo, 0)

	for _, path := range paths {
		t, err := getFileTodos(path)
		if err != nil {
			fmt.Printf("error occurred getting todos for file %s: %s", path, err.Error())
			continue
		}

		todos = append(todos, t...)
	}

	for _, todo := range todos {
		fmt.Println("===========================")
		fmt.Printf(
			"Found TODO in file '%s' starting at: %d, ending at: %d:\n",
			todo.filePath,
			todo.lineStart,
			todo.lineEnd,
		)
		fmt.Printf("Content:\n")
		fmt.Println(todo.content)
		fmt.Println()
	}
}

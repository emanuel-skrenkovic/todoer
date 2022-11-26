package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
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

func match(reader *bufio.Reader, nextN int, check string) bool {
	next, err := reader.Peek(nextN)
	if err != nil {
		return false
	}

	if string(next[:]) != check {
		return false
	}

	for i := 0; i < nextN; i++ {
		_, _ = reader.ReadByte()
	}

	return true
}

func getFileTodos(path string) ([]Todo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// TODO: single line TODO example

	defer func() {
		if err := f.Close(); err != nil {
			log.Print(err)
		}
	}()

	// TODO: multi line todo with
	// double slashes.

	todos := make([]Todo, 0) // TODO: handle todos inline with code

	/* TODO: handle todos inline with code with slash star */

	var current *Todo /* TODO: handle todos starting inline
	but ending on a different line */

	/*
	   TODO: multi line comment with
	   slash star
	*/

	rd := bufio.NewReader(f) // TODO: inline todo

	singleLineComment := false
	insideComment := false
	insideTodo := false

	lineNum := 1

MainLoop:
	for {
		char, _, err := rd.ReadRune()
		if err != nil {
			if err == io.EOF {
				break MainLoop
			}

			return nil, err
		}

		if insideTodo && !insideComment {
			isLineComment := false

			lastChar := " "
			for string(lastChar) == " " || string(lastChar) == "\t" {
				lastRune, _, err := rd.ReadRune()
				if err != nil {
					return nil, err
				}

				lastChar = string(lastRune)
			}

			err = rd.UnreadRune()
			if err != nil {
				return nil, err
			}

			isLineComment = match(rd, 2, "//")
			if isLineComment {
				singleLineComment = true
				insideComment = true
			}

			if !isLineComment {
				current.lineEnd = lineNum - 1
				todos = append(todos, *current)
				current = nil
				insideTodo = false
			}
		}

		if char == '*' {
			next, err := rd.Peek(1)
			if err != nil {
				return nil, err
			}

			nextChar, _ := utf8.DecodeRune(next)
			if insideComment && nextChar == '/' {
				insideComment = false

				if insideTodo {
					current.lineEnd = lineNum
					todos = append(todos, *current)
					current = nil
				}

				insideTodo = false
			}
		}

		if char == '/' {
			next, err := rd.Peek(1)
			if err != nil {
				return nil, err
			}

			nextChar, _ := utf8.DecodeRune(next)

			if !insideComment {
				singleLineComment = nextChar == '/'
			}

			multiLineComment := nextChar == '*'
			insideComment = singleLineComment || multiLineComment
		}

		if insideComment && char == 'T' {
			next, err := rd.Peek(3)
			if err != nil {
				return nil, err
			}

			nextThree := string(next)
			if nextThree == "ODO" {
				insideTodo = true
			}
		}

		if insideTodo {
			if current == nil {
				current = &Todo{filePath: path, lineStart: lineNum}
			}

			current.content += string(char)
		}

		if char == '\n' {
			lineNum += 1

			if singleLineComment {
				insideComment = false

				// TODO: move up
				//if insideTodo {
				//	current.lineEnd = lineNum - 1
				//	todos = append(todos, *current)
				//	current = nil
				//	insideTodo = false
				//}
			}

			singleLineComment = false
		}
	}

	if current != nil {
		current.lineEnd = lineNum - 1
		todos = append(todos, *current)
		current = nil
	}

	return todos, nil
}

func main() {
	path := "./"

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	paths, err := findGoFilePaths(absolutePath)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	var lock sync.RWMutex

	fileTodos := make(map[string][]Todo)

	for _, path := range paths {
		wg.Add(1)

		go func(wg *sync.WaitGroup, path string) {
			defer wg.Done()

			todos, err := getFileTodos(path)
			if err != nil {
				log.Printf("error occurred getting todos for file %s: %s\n", path, err.Error())
			} else {
				lock.Lock()
				defer lock.Unlock()
				fileTodos[path] = todos
			}
		}(&wg, path)
	}

	wg.Wait()

	// fmt.Printf("Found %d TODOs in %d file/s.\n", len(todos), len(paths))

	for path, todos := range fileTodos {
		gitBlameBytes, err := exec.Command("git", "--no-pager", "blame", path).
			Output()
		if err != nil {
			log.Print(err)
			continue
		}

		gitBlame := strings.Split(string(gitBlameBytes), "\n")

		for _, todo := range todos {
			blamePart := gitBlame[todo.lineStart]
			start := strings.Index(blamePart, "(") + 1
			end := strings.Index(blamePart, ")")

			commitInfo := blamePart[start:end]
			commitInfoParts := strings.Split(commitInfo, " ")
			committer := commitInfoParts[0] + " " + commitInfoParts[1]
			committedAt := commitInfoParts[2]

			fmt.Println("===========================")
			fmt.Printf("TODO by: %s\n", committer)
			fmt.Printf("Committed at: %s\n", committedAt)
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
}

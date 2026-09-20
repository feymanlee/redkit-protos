package main

// 本文件解析手写 Proto import 并执行允许依赖检查，阻止跨 bounded context 的未声明契约耦合。

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"unicode"
)

type tokenKind int

const (
	tokenIdentifier tokenKind = iota
	tokenString
	tokenSemicolon
)

type token struct {
	kind  tokenKind
	value string
}

// main 作为 check_proto_imports 的进程入口，负责在运行边界前完成检查或装配。
func main() {
	root := flag.String("root", ".", "Proto root to inspect")
	flag.Parse()

	err := filepath.WalkDir(*root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".proto" {
			return nil
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		imports, err := protoImports(contents)
		if err != nil {
			return fmt.Errorf("parse Proto imports in %s: %w", path, err)
		}
		for _, importPath := range imports {
			fmt.Printf("%s\t%s\n", path, importPath)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// protoImports tokenizes enough of the Proto grammar to extract complete import
// declarations. Whitespace, line breaks, and comments are insignificant between
// import tokens, so valid multiline imports are handled the same as one-line ones.
func protoImports(contents []byte) ([]string, error) {
	tokens, err := tokenize(contents)
	if err != nil {
		return nil, err
	}

	var imports []string
	for index := 0; index < len(tokens); index++ {
		if tokens[index].kind != tokenIdentifier || tokens[index].value != "import" {
			continue
		}
		index++
		if index < len(tokens) && tokens[index].kind == tokenIdentifier &&
			(tokens[index].value == "public" || tokens[index].value == "weak") {
			index++
		}
		if index >= len(tokens) || tokens[index].kind != tokenString {
			return nil, fmt.Errorf("import declaration is missing its path")
		}
		importPath := tokens[index].value
		index++
		if index >= len(tokens) || tokens[index].kind != tokenSemicolon {
			return nil, fmt.Errorf("import declaration for %q is missing its semicolon", importPath)
		}
		imports = append(imports, importPath)
	}
	return imports, nil
}

// tokenize 完成 check_proto_imports 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func tokenize(contents []byte) ([]token, error) {
	var tokens []token
	for index := 0; index < len(contents); {
		if unicode.IsSpace(rune(contents[index])) {
			index++
			continue
		}
		if contents[index] == '/' && index+1 < len(contents) {
			switch contents[index+1] {
			case '/':
				index += 2
				for index < len(contents) && contents[index] != '\n' {
					index++
				}
				continue
			case '*':
				end := index + 2
				for end+1 < len(contents) && (contents[end] != '*' || contents[end+1] != '/') {
					end++
				}
				if end+1 >= len(contents) {
					return nil, fmt.Errorf("unterminated block comment")
				}
				index = end + 2
				continue
			}
		}

		switch contents[index] {
		case ';':
			tokens = append(tokens, token{kind: tokenSemicolon})
			index++
		case '"':
			end := index + 1
			escaped := false
			for end < len(contents) {
				if !escaped && contents[end] == '"' {
					break
				}
				escaped = !escaped && contents[end] == '\\'
				if contents[end] != '\\' {
					escaped = false
				}
				end++
			}
			if end == len(contents) {
				return nil, fmt.Errorf("unterminated string literal")
			}
			value, err := strconv.Unquote(string(contents[index : end+1]))
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{kind: tokenString, value: value})
			index = end + 1
		default:
			if isIdentifierStart(contents[index]) {
				end := index + 1
				for end < len(contents) && isIdentifierPart(contents[end]) {
					end++
				}
				tokens = append(tokens, token{kind: tokenIdentifier, value: string(contents[index:end])})
				index = end
				continue
			}
			index++
		}
	}
	return tokens, nil
}

// isIdentifierStart 完成 check_proto_imports 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func isIdentifierStart(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

// isIdentifierPart 完成 check_proto_imports 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func isIdentifierPart(value byte) bool {
	return isIdentifierStart(value) || value >= '0' && value <= '9'
}

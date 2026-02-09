// Package main предоставляет утилиту для генерации методов Reset() для структур.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func main() {
	var (
		path    = flag.String("path", ".", "Path to scan")
		verbose = flag.Bool("v", false, "Verbose output")
		dryRun  = flag.Bool("dry-run", false, "Dry run (don't write files)")
	)
	flag.Parse()

	if err := run(*path, *verbose, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(path string, verbose, dryRun bool) error {
	if verbose {
		fmt.Printf("Scanning path: %s\n", path)
	}

	// Собираем все Go файлы рекурсивно
	var goFiles []string
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() &&
			strings.HasSuffix(filePath, ".go") &&
			!strings.HasSuffix(filePath, "_test.go") &&
			!strings.HasSuffix(filePath, ".gen.go") &&
			!strings.Contains(filePath, "/vendor/") {
			goFiles = append(goFiles, filePath)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d Go files\n", len(goFiles))
	}

	// Обрабатываем каждый файл
	totalGenerated := 0
	processedPackages := make(map[string]bool)

	for _, file := range goFiles {
		// Проверяем быстро наличие комментария
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if !bytes.Contains(content, []byte("generate:reset")) {
			continue
		}

		pkgPath := filepath.Dir(file)
		if processedPackages[pkgPath] {
			continue
		}

		// Обрабатываем пакет
		generated, err := processPackage(pkgPath, verbose, dryRun)
		if err != nil {
			return fmt.Errorf("failed to process package %s: %w", pkgPath, err)
		}

		totalGenerated += generated
		processedPackages[pkgPath] = true
	}

	if totalGenerated == 0 {
		fmt.Println("No structures with // generate:reset found")
	} else if verbose {
		fmt.Printf("Generated %d Reset() methods\n", totalGenerated)
	}

	return nil
}

func processPackage(pkgPath string, verbose, dryRun bool) (int, error) {
	fset := token.NewFileSet()
	var structs []StructInfo

	// Собираем все .go файлы в пакете
	files, err := os.ReadDir(pkgPath)
	if err != nil {
		return 0, err
	}

	// Парсим каждый файл пакета
	for _, fileInfo := range files {
		if fileInfo.IsDir() ||
			!strings.HasSuffix(fileInfo.Name(), ".go") ||
			strings.HasSuffix(fileInfo.Name(), "_test.go") ||
			strings.HasSuffix(fileInfo.Name(), ".gen.go") {
			continue
		}

		filePath := filepath.Join(pkgPath, fileInfo.Name())
		f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: failed to parse %s: %v\n", filePath, err)
			}
			continue
		}

		// Ищем структуры с комментарием generate:reset
		foundStructs := findResetableStructsInFile(f, fset, filePath, verbose)
		structs = append(structs, foundStructs...)
	}

	if len(structs) == 0 {
		return 0, nil
	}

	if verbose {
		fmt.Printf("Found %d resetable structure(s) in package %s\n", len(structs), pkgPath)
	}

	// Генерируем код
	generatedCode := generateResetFile(structs, pkgPath, verbose)
	if generatedCode == "" {
		return 0, nil
	}

	// Форматируем код
	formattedCode, err := format.Source([]byte(generatedCode))
	if err != nil {
		return 0, fmt.Errorf("failed to format code: %w", err)
	}

	// Определяем путь для выходного файла
	outputPath := filepath.Join(pkgPath, "reset.gen.go")

	if dryRun {
		fmt.Printf("\n=== DRY RUN: Would write to %s ===\n", outputPath)
		fmt.Println(string(formattedCode))
		fmt.Println("=== END DRY RUN ===")
		return len(structs), nil
	}

	// Записываем файл
	if err := os.WriteFile(outputPath, formattedCode, 0644); err != nil {
		return 0, fmt.Errorf("failed to write file: %w", err)
	}

	if verbose {
		fmt.Printf("Generated %s\n", outputPath)
	}

	return len(structs), nil
}

func findResetableStructsInFile(f *ast.File, fset *token.FileSet, filePath string, verbose bool) []StructInfo {
	var structs []StructInfo

	ast.Inspect(f, func(node ast.Node) bool {
		typeSpec, ok := node.(*ast.TypeSpec)
		if !ok {
			return true
		}

		// Проверяем, что это структура
		_, isStruct := typeSpec.Type.(*ast.StructType)
		if !isStruct {
			return true
		}

		// Ищем комментарий generate:reset
		hasResetComment := false

		// 1. Проверяем комментарии у самого TypeSpec
		if typeSpec.Doc != nil {
			for _, comment := range typeSpec.Doc.List {
				if strings.Contains(comment.Text, "generate:reset") {
					hasResetComment = true
					break
				}
			}
		}

		// 2. Проверяем комментарии в файле перед структурой
		if !hasResetComment {
			structPos := fset.Position(typeSpec.Pos())

			// Ищем комментарии, которые находятся прямо перед структурой
			for _, commentGroup := range f.Comments {
				commentPos := fset.Position(commentGroup.End())

				// Комментарий заканчивается на строке прямо перед структурой
				if commentPos.Line+1 == structPos.Line {
					for _, comment := range commentGroup.List {
						if strings.Contains(comment.Text, "generate:reset") {
							hasResetComment = true
							break
						}
					}
				}
				if hasResetComment {
					break
				}
			}
		}

		if hasResetComment {
			structs = append(structs, StructInfo{
				Name:   typeSpec.Name.Name,
				Struct: typeSpec.Type.(*ast.StructType),
				File:   f,
			})

			if verbose {
				fmt.Printf("  Found: %s in %s\n", typeSpec.Name.Name, filepath.Base(filePath))
			}
		}

		return true
	})

	return structs
}

type StructInfo struct {
	Name   string
	Struct *ast.StructType
	File   *ast.File
}

func generateResetFile(structs []StructInfo, pkgPath string, verbose bool) string {
	if len(structs) == 0 {
		return ""
	}

	// Берем информацию о пакете из первого файла
	pkgName := structs[0].File.Name.Name

	var buf bytes.Buffer

	// Заголовок файла
	buf.WriteString("// Code generated by reset tool. DO NOT EDIT.\n")
	buf.WriteString("//go:generate go run ./cmd/reset/main.go .\n\n")
	buf.WriteString("package " + pkgName + "\n\n")

	// Генерируем методы для каждой структуры
	for _, info := range structs {
		methodCode := generateResetMethod(info)
		buf.WriteString(methodCode)
		buf.WriteString("\n\n")
	}

	return buf.String()
}

func generateResetMethod(info StructInfo) string {
	var buf bytes.Buffer

	structName := info.Name
	receiver := getReceiverName(structName)

	// Заголовок метода
	buf.WriteString(fmt.Sprintf("// Reset сбрасывает состояние %s к начальным значениям.\n", structName))
	buf.WriteString(fmt.Sprintf("func (%s *%s) Reset() {\n", receiver, structName))
	buf.WriteString(fmt.Sprintf("\tif %s == nil {\n", receiver))
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n\n")

	// Генерируем сброс для каждого поля
	for _, field := range info.Struct.Fields.List {
		if len(field.Names) == 0 {
			continue // Анонимные поля
		}

		for _, fieldName := range field.Names {
			fieldCode := generateFieldReset(receiver, fieldName.Name, field.Type, 1)
			buf.WriteString(fieldCode)
		}
	}

	buf.WriteString("}\n")

	return buf.String()
}

func getReceiverName(structName string) string {
	if len(structName) == 0 {
		return "r"
	}

	firstChar := rune(structName[0])
	return string(unicode.ToLower(firstChar))
}

func generateFieldReset(receiver, fieldName string, expr ast.Expr, indent int) string {
	var buf bytes.Buffer

	indentStr := strings.Repeat("\t", indent)
	fieldAccess := fmt.Sprintf("%s.%s", receiver, fieldName)

	switch t := expr.(type) {
	case *ast.Ident:
		// Простой тип
		typeName := t.Name
		zeroValue := getZeroValue(typeName)
		if zeroValue != "" {
			buf.WriteString(fmt.Sprintf("%s%s = %s\n", indentStr, fieldAccess, zeroValue))
		} else {
			// Пользовательский тип - предполагаем структуру
			buf.WriteString(fmt.Sprintf("%s// Field %s of type %s - may need manual reset\n",
				indentStr, fieldName, typeName))
		}

	case *ast.StarExpr:
		// Указатель
		buf.WriteString(fmt.Sprintf("%sif %s != nil {\n", indentStr, fieldAccess))

		switch x := t.X.(type) {
		case *ast.Ident:
			typeName := x.Name
			zeroValue := getZeroValue(typeName)
			if zeroValue != "" {
				buf.WriteString(fmt.Sprintf("%s\t*%s = %s\n", indentStr, fieldAccess, zeroValue))
			} else {
				// Пользовательский тип - проверяем наличие Reset
				buf.WriteString(fmt.Sprintf("%s\tif resetter, ok := %s.(interface{ Reset() }); ok {\n", indentStr, fieldAccess))
				buf.WriteString(fmt.Sprintf("%s\t\tresetter.Reset()\n", indentStr))
				buf.WriteString(fmt.Sprintf("%s\t}\n", indentStr))
			}
		default:
			buf.WriteString(fmt.Sprintf("%s\t// Complex pointer type\n", indentStr))
		}

		buf.WriteString(fmt.Sprintf("%s}\n", indentStr))

	case *ast.ArrayType:
		// Массив или слайс
		if t.Len == nil {
			// Слайс
			buf.WriteString(fmt.Sprintf("%s%s = %s[:0]\n", indentStr, fieldAccess, fieldAccess))
		} else {
			// Массив - обнуляем каждый элемент
			buf.WriteString(fmt.Sprintf("%sfor i := range %s {\n", indentStr, fieldAccess))

			// Получаем нулевое значение для типа элемента
			elemZeroValue := getElementZeroValue(t.Elt)
			buf.WriteString(fmt.Sprintf("%s\t%s[i] = %s\n", indentStr, fieldAccess, elemZeroValue))

			buf.WriteString(fmt.Sprintf("%s}\n", indentStr))
		}

	case *ast.MapType:
		// Мапа
		buf.WriteString(fmt.Sprintf("%sclear(%s)\n", indentStr, fieldAccess))

	case *ast.StructType:
		// Вложенная структура (инлайн)
		buf.WriteString(fmt.Sprintf("%s// Inline struct field %s\n", indentStr, fieldName))
		buf.WriteString(fmt.Sprintf("%s// May need manual Reset() implementation\n", indentStr))

	case *ast.SelectorExpr:
		// Тип из другого пакета, например: time.Time
		buf.WriteString(fmt.Sprintf("%s// Field %s of type %v\n", indentStr, fieldName, t))
		buf.WriteString(fmt.Sprintf("%s// May need manual Reset() implementation\n", indentStr))

	default:
		buf.WriteString(fmt.Sprintf("%s// TODO: Implement reset for field %s of type %T\n",
			indentStr, fieldName, expr))
	}

	return buf.String()
}

func getZeroValue(typeName string) string {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"uintptr", "byte", "rune", "float32", "float64",
		"complex64", "complex128":
		return "0"
	case "string":
		return `""`
	case "bool":
		return "false"
	case "error":
		return "nil"
	default:
		// Проверяем распространенные типы
		if typeName == "time.Duration" {
			return "0"
		}
		// Для остальных возвращаем пустую структуру
		return typeName + "{}"
	}
}

func getElementZeroValue(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return getZeroValue(t.Name)
	case *ast.StarExpr:
		// Для указателей возвращаем nil
		return "nil"
	default:
		return "nil"
	}
}

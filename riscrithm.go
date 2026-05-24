package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const DEFAULT_HEADER string = ".section .text"
const ASM_INTERRUPT_U string = "uret"
const ASM_INTERRUPT_S string = "sret"
const ASM_INTERRUPT_M string = "mret"
const ASM_HALT string = "ecall"
const ASM_DEBUGGER_TRAP string = "ebreak"
const ASM_WAIT string = "wfi"
const ASM_PASS string = "nop"
const ASM_JUMP string = "j "

func ExtractHeader(lineContent string, lineNumber int) (string, error) {
	trimmedLineContent := strings.TrimSpace(lineContent)

	if strings.HasPrefix(trimmedLineContent, "header") {
		chosenHeader := strings.TrimSpace(strings.TrimPrefix(trimmedLineContent, "header"))

		if chosenHeader == "default" {
			return DEFAULT_HEADER, nil
		} else if chosenHeader != "" {
			return chosenHeader, nil
		} else {
			return "", fmt.Errorf("SyntaxError on line %d: Expected a header (e.g. 'header default'), got nothing.", lineNumber)
		}
	}
	return "", nil
}

func ExtractEntryPoint(lineContent string, lineNumber int) (string, error) {
	trimmedLineContent := strings.TrimSpace(lineContent)

	if strings.HasPrefix(trimmedLineContent, "entrypoint") {
		chosenEntrypoint := strings.TrimSpace(strings.TrimPrefix(trimmedLineContent, "entrypoint"))

		if chosenEntrypoint != "" {
			return chosenEntrypoint, nil
		} else {
			return "", fmt.Errorf("SyntaxError on line %d: Expected an entrypoint (e.g. 'entrypoint main') got nothing.", lineNumber)
		}
	}
	return "", nil
}

func RegisterMacrosAndExpansions(lineContent string, lineNumber int, macrosAndExpansions *map[string]string) error {
	trimmedLineContent := strings.TrimSpace(lineContent)
	partsOfLineContent := strings.SplitN(trimmedLineContent, "=", 2)

	if len(partsOfLineContent) == 2 {
		macro := strings.TrimLeft(RemoveComments(strings.TrimSpace(partsOfLineContent[0])), "define")
		expension := RemoveComments(strings.TrimSpace(partsOfLineContent[1]))

		macro = strings.TrimSpace(macro)
		expension = strings.TrimSpace(expension)

		if macro == "" {
			return fmt.Errorf("SyntaxError on line %d: Macro cannot be empty.", lineNumber)
		}
		if expension == "" {
			return fmt.Errorf("SyntaxError on line %d: Expansion cannot be empty.", lineNumber)
		}

		(*macrosAndExpansions)[macro] = expension
	} else {
		return fmt.Errorf("SyntaxError on line %d: Definition requires a macro and an expansion (e.g. 'define <macro> = <expansion>').", lineNumber)
	}
	return nil
}

func ReplaceMacros(lineContent string, macrosAndExpansions *map[string]string) string {
	expandedLineContent := lineContent

	for macro, expansion := range *macrosAndExpansions {
		expandedLineContent = strings.ReplaceAll(expandedLineContent, macro, expansion)
	}

	return expandedLineContent
}

func ExpandShorthands(lineContent string) string {
	smallShorthandRegex := regexp.MustCompile(`(\w+)\s*(\+\+|--|\^\^)`)
	expandedShorthandsLine := smallShorthandRegex.ReplaceAllStringFunc(lineContent, func(match string) string {
		submatches := smallShorthandRegex.FindStringSubmatch(match)
		variable := submatches[1]
		operator := submatches[2]

		switch operator {
		case "++":
			return variable + " = " + variable + " + 1"
		case "--":
			return variable + " = " + variable + " - 1"
		case "^^":
			return variable + " = " + variable + " ^ " + variable
		default:
			return lineContent
		}
	})

	bigShorthandRegex := regexp.MustCompile(`(\w+)\s*([-+/\*%^&|]=|<<=|>>=)\s*([^;\n]+)`)
	expandedShorthandsLine = bigShorthandRegex.ReplaceAllStringFunc(expandedShorthandsLine, func(match string) string {
		submatches := bigShorthandRegex.FindStringSubmatch(match)
		variable := submatches[1]
		operator := submatches[2]
		value := submatches[3]

		simpleOperator := operator[:1]
		if strings.Contains(expandedShorthandsLine, ">>") {
			return variable + " = " + variable + " >> " + value
		}
		if strings.Contains(expandedShorthandsLine, "<<") {
			return variable + " = " + variable + " << " + value
		}
		return variable + " = " + variable + " " + simpleOperator + " " + value
	})

	return expandedShorthandsLine
}

func TransformToAsm(lineContent string) string {
	operationRegex := regexp.MustCompile(`^(\w+)\s*=\s*(\w+)\s*([+\-&|^|\*|/|%]|<<|>>)\s*(\w+)$`)
	if !operationRegex.MatchString(strings.TrimSpace(lineContent)) {
		return lineContent
	}

	matches := operationRegex.FindStringSubmatch(strings.TrimSpace(lineContent))
	destinationRegister := matches[1]
	sourceRegister := matches[2]
	operator := matches[3]
	secondOperand := matches[4]

	isImmediate := false
	if _, err := strconv.Atoi(secondOperand); err == nil {
		isImmediate = true
	}

	var instruction string
	switch operator {
	case "+":
		if isImmediate {
			instruction = "addi"
		} else {
			instruction = "add"
		}
	case "-":
		if isImmediate {
			instruction = "addi"
			if number, err := strconv.Atoi(secondOperand); err == nil {
				secondOperand = strconv.Itoa(-number)
			}
		} else {
			instruction = "sub"
		}
	case "&":
		if isImmediate {
			instruction = "andi"
		} else {
			instruction = "and"
		}
	case "|":
		if isImmediate {
			instruction = "ori"
		} else {
			instruction = "or"
		}
	case "^":
		if isImmediate {
			instruction = "xori"
		} else {
			instruction = "xor"
		}
	case ">>":
		if isImmediate {
			instruction = "srli"
		} else {
			instruction = "srl"
		}
	case "<<":
		if isImmediate {
			instruction = "slli"
		} else {
			instruction = "sll"
		}
	case "*":
		instruction = "mul"
	case "/":
		instruction = "div"
	case "%":
		instruction = "rem"
	default:
		return lineContent
	}

	return instruction + " " + destinationRegister + ", " + sourceRegister + ", " + secondOperand
}

func TransformLoadToAsm(lineContent string) string {
	loadRegex := regexp.MustCompile(`^load\s*(\w+)\s*=\s*(\w+)$`)
	if !loadRegex.MatchString(strings.TrimSpace(lineContent)) {
		return lineContent
	}

	matches := loadRegex.FindStringSubmatch(strings.TrimSpace(lineContent))
	destinationRegister := matches[1]
	sourceValue := matches[2]

	if _, err := strconv.Atoi(sourceValue); err == nil {
		return "li " + destinationRegister + ", " + sourceValue
	}

	return lineContent
}

func TransformMoveToAsm(lineContent string) string {
	loadRegex := regexp.MustCompile(`^move\s*(\w+)\s*=\s*(\w+)$`)
	if !loadRegex.MatchString(strings.TrimSpace(lineContent)) {
		return lineContent
	}

	matches := loadRegex.FindStringSubmatch(strings.TrimSpace(lineContent))
	destinationRegister := matches[1]
	sourceValue := matches[2]

	if _, err := strconv.Atoi(sourceValue); err == nil {
		return "mv " + destinationRegister + ", " + sourceValue
	}

	return lineContent
}

func TransformSwapToAsm(lineContent string) []string {
	swapRegex := regexp.MustCompile(`^(\w+)\s*swap\s*(\w+)$`)
	if !swapRegex.MatchString(strings.TrimSpace(lineContent)) {
		return []string{lineContent}
	}

	matches := swapRegex.FindStringSubmatch(strings.TrimSpace(lineContent))
	firstRegister := matches[1]
	secondRegister := matches[2]

	if firstRegister == secondRegister {
		return []string{lineContent}
	}

	firstLine := "xor " + firstRegister + ", " + firstRegister + ", " + secondRegister
	secondLine := "xor " + secondRegister + ", " + firstRegister + ", " + secondRegister
	thirdLine := "xor " + firstRegister + ", " + firstRegister + ", " + secondRegister

	return []string{firstLine, secondLine, thirdLine}
}

func TransformConditionalToAsm(lineContent string) []string {
	conditionalRegex := regexp.MustCompile(`^if\s*(\w+)\s*(==|!=|>=|<=|>|<)\s*(\w+)\s*@(\w+)\s*else\s*@(\w+)$`)
	if !conditionalRegex.MatchString(strings.TrimSpace(lineContent)) {
		return []string{lineContent}
	}

	matches := conditionalRegex.FindStringSubmatch(strings.TrimSpace(lineContent))
	firstRegister := matches[1]
	operator := matches[2]
	secondRegister := matches[3]
	trueLabel := matches[4]
	falseLabel := matches[5]

	var branchInstruction string
	switch operator {
	case "==":
		branchInstruction = "beq"
	case "!=":
		branchInstruction = "bne"
	case "<":
		branchInstruction = "blt"
	case ">=":
		branchInstruction = "bge"
	case ">":
		branchInstruction = "blt"
		firstRegister, secondRegister = secondRegister, firstRegister
	case "<=":
		branchInstruction = "bge"
		firstRegister, secondRegister = secondRegister, firstRegister
	}

	conditionLine := branchInstruction + " " + firstRegister + ", " + secondRegister + ", " + trueLabel
	jumpFalseLine := "j " + falseLabel

	return []string{conditionLine, jumpFalseLine}
}

func TransformStackOperationToAsm(lineContent string) []string {
	trimmedLine := strings.TrimSpace(lineContent)

	pushRegex := regexp.MustCompile(`^(\w+)\s*->\s*stack\.(b|w|d)$`)
	popRegex := regexp.MustCompile(`^(\w+)\s*<-\s*stack\.(b|w|d)$`)
	peekRegex := regexp.MustCompile(`^(\w+)\s*=\s*stack\.(b|w|d)$`)

	if pushRegex.MatchString(trimmedLine) {
		matches := pushRegex.FindStringSubmatch(trimmedLine)
		sourceRegister := matches[1]
		sizeExtension := matches[2]

		var byteSize string
		var storeInstruction string
		switch sizeExtension {
		case "b":
			byteSize = "1"
			storeInstruction = "sb"
		case "w":
			byteSize = "4"
			storeInstruction = "sw"
		case "d":
			byteSize = "8"
			storeInstruction = "sd"
		}

		decrementStackPointer := "addi sp, sp, -" + byteSize
		pushValue := storeInstruction + " " + sourceRegister + ", 0(sp)"
		return []string{decrementStackPointer, pushValue}
	}

	if popRegex.MatchString(trimmedLine) {
		matches := popRegex.FindStringSubmatch(trimmedLine)
		destinationRegister := matches[1]
		sizeExtension := matches[2]

		var byteSize string
		var loadInstruction string
		switch sizeExtension {
		case "b":
			byteSize = "1"
			loadInstruction = "lb"
		case "w":
			byteSize = "4"
			loadInstruction = "lw"
		case "d":
			byteSize = "8"
			loadInstruction = "ld"
		}

		popValue := loadInstruction + " " + destinationRegister + ", 0(sp)"
		incrementStackPointer := "addi sp, sp, " + byteSize
		return []string{popValue, incrementStackPointer}
	}

	if peekRegex.MatchString(trimmedLine) {
		matches := peekRegex.FindStringSubmatch(trimmedLine)
		destinationRegister := matches[1]
		sizeExtension := matches[2]

		var loadInstruction string
		switch sizeExtension {
		case "b":
			loadInstruction = "lb"
		case "w":
			loadInstruction = "lw"
		case "d":
			loadInstruction = "ld"
		}

		peekValue := loadInstruction + " " + destinationRegister + ", 0(sp)"
		return []string{peekValue}
	}

	return []string{lineContent}
}

func TransformHeapOperationToAsm(lineContent string) []string {
	trimmedLine := strings.TrimSpace(lineContent)

	heapLoadRegex := regexp.MustCompile(`^(\w+)\s*<-\s*heap\.(b|w|d)\s+from\s+&\s*(\w+)$`)
	heapStoreRegex := regexp.MustCompile(`^(\w+)\s*->\s*heap\.(b|w|d)\s+from\s+&\s*(\w+)$`)

	if heapLoadRegex.MatchString(trimmedLine) {
		matches := heapLoadRegex.FindStringSubmatch(trimmedLine)
		destinationRegister := matches[1]
		sizeExtension := matches[2]
		addressRegister := matches[3]

		var loadInstruction string
		switch sizeExtension {
		case "b":
			loadInstruction = "lb"
		case "w":
			loadInstruction = "lw"
		case "d":
			loadInstruction = "ld"
		}

		return []string{loadInstruction + " " + destinationRegister + ", 0(" + addressRegister + ")"}
	}

	if heapStoreRegex.MatchString(trimmedLine) {
		matches := heapStoreRegex.FindStringSubmatch(trimmedLine)
		sourceRegister := matches[1]
		sizeExtension := matches[2]
		addressRegister := matches[3]

		var storeInstruction string
		switch sizeExtension {
		case "b":
			storeInstruction = "sb"
		case "w":
			storeInstruction = "sw"
		case "d":
			storeInstruction = "sd"
		}

		return []string{storeInstruction + " " + sourceRegister + ", 0(" + addressRegister + ")"}
	}

	return []string{lineContent}
}

func OptimizeLineContent(lineContent string, previousLineContent string) string {
	currentLineTrimmed := strings.TrimSpace(lineContent)
	previousLineTrimmed := strings.TrimSpace(previousLineContent)

	if currentLineTrimmed == "..." {
		return currentLineTrimmed
	}

	if currentLineTrimmed == previousLineTrimmed {
		assignmentLoadRegex := regexp.MustCompile(`^load\s*\w+\s*=\s*[^;\n]+$`)
		if assignmentLoadRegex.MatchString(currentLineTrimmed) {
			return ""
		}
	}
	if currentLineTrimmed == previousLineTrimmed {
		assignmentMoveRegex := regexp.MustCompile(`^move\s*\w+\s*=\s*[^;\n]+$`)
		if assignmentMoveRegex.MatchString(currentLineTrimmed) {
			return ""
		}
	}

	identityMathRegex := regexp.MustCompile(`^(\w+)\s*=\s*(\w+)\s*([-+/\*])\s*(\d+)$`)
	if identityMathRegex.MatchString(currentLineTrimmed) {
		matches := identityMathRegex.FindStringSubmatch(currentLineTrimmed)
		destinationRegister := strings.TrimSpace(matches[1])
		sourceRegister := strings.TrimSpace(matches[2])
		operator := strings.TrimSpace(matches[3])
		numericValue, _ := strconv.Atoi(matches[4])

		isIdentity := false
		if (operator == "+" || operator == "-") && numericValue == 0 {
			isIdentity = true
		} else if (operator == "*" || operator == "/") && numericValue == 1 {
			isIdentity = true
		}

		if isIdentity {
			if destinationRegister == sourceRegister {
				return ""
			}
			return lineContent
		}
	}

	powerOfTwoMultiplyRegex := regexp.MustCompile(`^(\w+)\s*=\s*(\w+)\s*\*\s*(\d+)$`)
	if powerOfTwoMultiplyRegex.MatchString(currentLineTrimmed) {
		matches := powerOfTwoMultiplyRegex.FindStringSubmatch(currentLineTrimmed)
		destinationRegister := matches[1]
		sourceRegister := matches[2]
		multiplierValue, _ := strconv.Atoi(matches[3])

		if multiplierValue > 0 && (multiplierValue&(multiplierValue-1)) == 0 {
			shiftAmount := 0
			for multiplierValue > 1 {
				multiplierValue >>= 1
				shiftAmount++
			}
			if shiftAmount == 0 {
				return lineContent
			}
			return destinationRegister + " = " + sourceRegister + " << " + strconv.Itoa(shiftAmount)
		}
	}

	powerOfTwoDivideRegex := regexp.MustCompile(`^(\w+)\s*=\s*(\w+)\s*/\s*(\d+)$`)
	if powerOfTwoDivideRegex.MatchString(currentLineTrimmed) {
		matches := powerOfTwoDivideRegex.FindStringSubmatch(currentLineTrimmed)
		destinationRegister := matches[1]
		sourceRegister := matches[2]
		divisorValue, _ := strconv.Atoi(matches[3])

		if divisorValue > 0 && (divisorValue&(divisorValue-1)) == 0 {
			shiftAmount := 0
			for divisorValue > 1 {
				divisorValue >>= 1
				shiftAmount++
			}
			if shiftAmount == 0 {
				return lineContent
			}
			return destinationRegister + " = " + sourceRegister + " >> " + strconv.Itoa(shiftAmount)
		}
	}

	return lineContent
}

func ReadAndValidateSourceFile(filePath string) ([]string, error) {
	file, openingError := os.Open(filePath)
	if openingError != nil {
		return nil, openingError
	}
	defer file.Close()

	var scriptLines []string
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	insideLabelBlock := false

	for scanner.Scan() {
		lineNumber++
		rawLine := scanner.Text()
		trimmedLine := strings.TrimSpace(rawLine)

		if trimmedLine == "" {
			continue
		}

		isHeader := strings.HasPrefix(rawLine, "header")
		isEntryPoint := strings.HasPrefix(rawLine, "entrypoint")
		isDefinition := strings.HasPrefix(rawLine, "define")
		isComment := strings.HasPrefix(rawLine, "#")

		if isComment {
			continue
		}

		if isHeader || isEntryPoint || isDefinition {
			normalizedLine := NormalizeLineWhitespace(rawLine)
			scriptLines = append(scriptLines, normalizedLine)
			continue
		}

		isLabel := strings.HasSuffix(trimmedLine, ":")
		hasIndentation := strings.HasPrefix(RemoveComments(rawLine), " ") || strings.HasPrefix(RemoveComments(rawLine), "\t")

		if isLabel {
			if hasIndentation {
				return nil, fmt.Errorf("SyntaxError on line %d: labels must not be indented", lineNumber)
			}
			insideLabelBlock = true
			normalizedLine := NormalizeLineWhitespace(rawLine)
			scriptLines = append(scriptLines, normalizedLine)
			continue
		}

		if insideLabelBlock {
			if !hasIndentation {
				return nil, fmt.Errorf("SyntaxError on line %d: instructions inside a label block must be indented", lineNumber)
			}
			normalizedLine := NormalizeLineWhitespace(rawLine)
			scriptLines = append(scriptLines, normalizedLine)
			continue
		}

		return nil, fmt.Errorf("SyntaxError on line %d: instruction found outside of label block", lineNumber)
	}

	if scanningError := scanner.Err(); scanningError != nil {
		return nil, scanningError
	}

	return scriptLines, nil
}

func WriteAsmFile(filePath string, scriptLines []string) error {
	file, creationError := os.Create(filePath)
	if creationError != nil {
		return creationError
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for index, line := range scriptLines {
		var formattedLine string

		if index < 2 {
			formattedLine = line
		} else if strings.HasSuffix(line, ":") {
			formattedLine = line
		} else {
			formattedLine = "   " + line
		}

		_, writingError := writer.WriteString(formattedLine + "\n")
		if writingError != nil {
			return writingError
		}
	}

	flushingError := writer.Flush()
	if flushingError != nil {
		return flushingError
	}

	return nil
}

func NormalizeLineWhitespace(line string) string {
	var resultBuilder strings.Builder
	insideQuotes := false
	previousWasWhitespace := false

	trimmedLine := strings.TrimSpace(line)

	for _, character := range trimmedLine {
		if character == '"' {
			insideQuotes = !insideQuotes
			resultBuilder.WriteRune(character)
			previousWasWhitespace = false
			continue
		}

		if insideQuotes {
			resultBuilder.WriteRune(character)
			continue
		}

		if unicode.IsSpace(character) {
			if !previousWasWhitespace {
				resultBuilder.WriteRune(' ')
				previousWasWhitespace = true
			}
		} else {
			resultBuilder.WriteRune(character)
			previousWasWhitespace = false
		}
	}

	return RemoveComments(resultBuilder.String())
}

func RemoveComments(lineContent string) string {
	NotCommentedLine, _, _ := strings.Cut(lineContent, "#")

	return NotCommentedLine
}

func Parse(scriptLines []string, optimizationFlag bool) ([]string, string, string) {
	var header string
	var entrypoint string
	var err error
	var macrosAndExpansions map[string]string
	var parsedScriptLines []string
	var enteredRawAssemblyLabel bool = false
	var previousLineContent string

	macrosAndExpansions = make(map[string]string)

	for lineNumber, lineContent := range scriptLines {
		if strings.HasPrefix(lineContent, "!!") && strings.HasSuffix(lineContent, ":") {
			enteredRawAssemblyLabel = true
			lineContent = strings.TrimLeft(lineContent, "! ")
		}
		if !strings.HasPrefix(lineContent, "!!") && strings.HasSuffix(lineContent, ":") {
			enteredRawAssemblyLabel = false
		}
		if enteredRawAssemblyLabel {
			parsedScriptLines = append(parsedScriptLines, lineContent)
			continue
		}
		if lineNumber == 0 {
			header, err = ExtractHeader(lineContent, lineNumber)
			if err != nil {
				fmt.Println(err)
				goto out
			}
			continue
		}

		if lineNumber == 1 {
			entrypoint, err = ExtractEntryPoint(lineContent, lineNumber)
			if err != nil {
				fmt.Println(err)
				goto out
			}
			continue
		}

		if strings.HasPrefix(lineContent, "define") {
			RegisterMacrosAndExpansions(lineContent, lineNumber, &macrosAndExpansions)
			continue
		}

		lineContent = ReplaceMacros(lineContent, &macrosAndExpansions)

		if strings.Contains(lineContent, "if") && strings.Contains(lineContent, "else") {
			conditionalStatementToAsm := TransformConditionalToAsm(lineContent)
			parsedScriptLines = append(parsedScriptLines, conditionalStatementToAsm...)
			continue
		}
		if strings.Contains(lineContent, "swap") {
			swapStatementToAsm := TransformSwapToAsm(lineContent)
			parsedScriptLines = append(parsedScriptLines, swapStatementToAsm...)
			continue
		}
		if strings.Contains(lineContent, "stack") && (strings.Contains(lineContent, ".w") || strings.Contains(lineContent, ".b") || strings.Contains(lineContent, ".d")) {
			stackStatementToAsm := TransformStackOperationToAsm(lineContent)
			parsedScriptLines = append(parsedScriptLines, stackStatementToAsm...)
			continue
		}
		if strings.Contains(lineContent, "heap") && (strings.Contains(lineContent, ".w") || strings.Contains(lineContent, ".b") || strings.Contains(lineContent, ".d")) {
			heapStatementToAsm := TransformHeapOperationToAsm(lineContent)
			parsedScriptLines = append(parsedScriptLines, heapStatementToAsm...)
			continue
		}

		lineContent = ExpandShorthands(lineContent)
		if optimizationFlag {
			lineContent = OptimizeLineContent(lineContent, previousLineContent)
		}
		if lineContent != "" {
			previousLineContent = lineContent
		}
		lineContent = strings.ReplaceAll(lineContent, "interrupt.u", ASM_INTERRUPT_U)
		lineContent = strings.ReplaceAll(lineContent, "interrupt.s", ASM_INTERRUPT_S)
		lineContent = strings.ReplaceAll(lineContent, "interrupt.m", ASM_INTERRUPT_M)
		lineContent = strings.ReplaceAll(lineContent, "halt", ASM_HALT)
		lineContent = strings.ReplaceAll(lineContent, "trap", ASM_DEBUGGER_TRAP)
		lineContent = strings.ReplaceAll(lineContent, "wait", ASM_WAIT)
		lineContent = strings.ReplaceAll(lineContent, "...", ASM_PASS)
		lineContent = strings.ReplaceAll(lineContent, "@", ASM_JUMP)
		lineContent = TransformToAsm(lineContent)
		lineContent = TransformLoadToAsm(lineContent)
		lineContent = TransformMoveToAsm(lineContent)

		if lineContent != "" {
			parsedScriptLines = append(parsedScriptLines, lineContent)
		}
	}

	return parsedScriptLines, header, ".globl " + entrypoint

out:
	_ = header
	_ = entrypoint
	return []string{""}, "", ""
}

func main() {
	var header string
	var entrypoint string
	var parsedScriptLines []string

	if len(os.Args) >= 3 {
		filePathSourceCode := os.Args[1]
		filePathAsmFile := os.Args[2]
		scriptLines, readingError := ReadAndValidateSourceFile(filePathSourceCode)
		if readingError == nil {
			if len(os.Args) == 4 && (slices.Contains(os.Args, "-o") || slices.Contains(os.Args, "--optimize")) {
				parsedScriptLines, header, entrypoint = Parse(scriptLines, true)
			} else if len(os.Args) == 3 && !(slices.Contains(os.Args, "-o") || slices.Contains(os.Args, "--optimize")) {
				parsedScriptLines, header, entrypoint = Parse(scriptLines, false)
			} else {
				fmt.Println("CommandLineError: Expected a source code file, an assembly target file and an optional '--optimize'/'-o' flag.")
				goto out
			}
			fullAsmCode := []string{header, entrypoint}
			fullAsmCode = append(fullAsmCode, parsedScriptLines...)
			if fullAsmCode[0] != "" {
				writtingError := WriteAsmFile(filePathAsmFile, fullAsmCode)
				if writtingError != nil {
					fmt.Println(writtingError.Error())
					goto out
				}
			} else {
				goto out
			}
		} else {
			fmt.Println(readingError.Error())
			goto out
		}
	} else {
		fmt.Println("CommandLineError: Expected a source code file, an assembly target file and an optional '--optimize'/'-o' flag.")
	}
out:
	return
}

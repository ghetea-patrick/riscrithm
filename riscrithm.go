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
const ASM_INTERRUPT_M string = "mret"
const ASM_INTERRUPT_S string = "sret"
const ASM_INTERRUPT_U string = "uret"
const ASM_HALT string = "ecall"
const ASM_RETURN string = "ret"
const ASM_DEBUGGER_TRAP string = "ebreak"
const ASM_WAIT string = "wfi"
const ASM_JUMP string = "j "
const ASM_PASS string = "nop"

func extractHeader(lineContent string, lineNumber int) (string, error) {
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
	} else {
		return "", fmt.Errorf("SyntaxError on line %d: Expected a header (e.g. 'header default').", lineNumber)
	}
}

func extractEntryPoint(lineContent string, lineNumber int) (string, error) {
	trimmedLineContent := strings.TrimSpace(lineContent)

	if strings.HasPrefix(trimmedLineContent, "entrypoint") {
		chosenEntrypoint := strings.TrimSpace(strings.TrimPrefix(trimmedLineContent, "entrypoint"))

		if chosenEntrypoint != "" {
			return chosenEntrypoint, nil
		} else {
			return "", fmt.Errorf("SyntaxError on line %d: Expected an entrypoint (e.g. 'entrypoint main') got nothing.", lineNumber)
		}
	} else {
		return "", fmt.Errorf("SyntaxError on line %d: Expected an entrypoint (e.g. 'entrypoint main').", lineNumber)
	}
}

func registerMacrosAndExpansions(lineContent string, lineNumber int, macrosAndExpansions *map[string]string) error {
	trimmedLineContent := strings.TrimSpace(lineContent)
	partsOfLineContent := strings.SplitN(trimmedLineContent, "=", 2)

	if len(partsOfLineContent) == 2 {
		macro := strings.TrimLeft(removeComments(strings.TrimSpace(partsOfLineContent[0])), "define")
		expension := removeComments(strings.TrimSpace(partsOfLineContent[1]))

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

func replaceMacros(lineContent string, macrosAndExpansions *map[string]string) string {
	expandedLineContent := lineContent

	for macro, expansion := range *macrosAndExpansions {
		expandedLineContent = strings.ReplaceAll(expandedLineContent, macro, expansion)
	}

	return expandedLineContent
}

func expandShorthands(lineContent string) string {
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

	bigShorthandRegex := regexp.MustCompile(`(\w+)\s*([-+/\*%^&|]|<<|>>)=\s*([^;\n]+)`)
	expandedShorthandsLine = bigShorthandRegex.ReplaceAllStringFunc(expandedShorthandsLine, func(match string) string {
		submatches := bigShorthandRegex.FindStringSubmatch(match)
		variable := submatches[1]
		operator := submatches[2]
		value := submatches[3]

		return variable + " = " + variable + " " + operator + " " + value
	})

	return expandedShorthandsLine
}

func extractBlocks(lines []string, targets []string) []string {
	var results []string

	targetMap := make(map[string]bool)
	for _, target := range targets {
		targetMap[target] = true
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		cleanLine := strings.TrimPrefix(trimmed, "!!")
		cleanLine = strings.TrimSuffix(cleanLine, ":")
		cleanLine = strings.TrimSpace(cleanLine)

		if targetMap[cleanLine] {
			var block []string = []string{trimmed}

			for j := i + 1; j < len(lines); j++ {
				nextLine := lines[j]

				if !strings.HasSuffix(nextLine, ":") {
					block = append(block, strings.TrimSpace(nextLine))
					i = j
				} else {
					break
				}
			}

			if len(block) > 1 {
				results = append(results, block...)
			}
		}
	}

	return results
}

func extractJumps(lineContent string) []string {
	var jumps []string

	jumpRegex := regexp.MustCompile(`@(\S+)`)
	matches := jumpRegex.FindAllStringSubmatch(lineContent, -1)

	for _, match := range matches {
		jumps = append(jumps, match[1])
	}

	return jumps
}

func transformToAsm(lineContent string) string {
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

func transformLoadToAsm(lineContent string) string {
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

func transformMoveToAsm(lineContent string) string {
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

func transformSwapToAsm(lineContent string) []string {
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

func transformConditionalToAsm(lineContent string) []string {
	var containsElse bool
	var branchInstruction string
	var matches []string

	conditionalRegexWithElse := regexp.MustCompile(`^if\s*(\w+)\s*(==|!=|>=|<=|>|<)\s*(\w+)\s*@(\w+)\s*else\s*@(\w+)$`)
	conditionalRegexWithoutElse := regexp.MustCompile(`^if\s*(\w+)\s*(==|!=|>=|<=|>|<)\s*(\w+)\s*@(\w+)$`)

	if conditionalRegexWithElse.MatchString(strings.TrimSpace(lineContent)) {
		matches = conditionalRegexWithElse.FindStringSubmatch(strings.TrimSpace(lineContent))
		containsElse = true
	} else if conditionalRegexWithoutElse.MatchString(strings.TrimSpace(lineContent)) {
		matches = conditionalRegexWithoutElse.FindStringSubmatch(strings.TrimSpace(lineContent))
		containsElse = false
	} else {
		return []string{lineContent}
	}
	firstRegister := matches[1]
	operator := matches[2]
	secondRegister := matches[3]
	trueLabel := matches[4]

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

	if containsElse {
		return []string{conditionLine, "j " + matches[5]}
	} else {
		return []string{conditionLine}
	}
}

func transformStackOperationToAsm(lineContent string) []string {
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

func transformHeapOperationToAsm(lineContent string) []string {
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

func optimizeLineContent(lineContent string, previousLineContent string) string {
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
			} else {
				if (operator == "+" || operator == "-") && numericValue == 0 {
					return "mv " + destinationRegister + ", " + sourceRegister
				} else if (operator == "*" || operator == "/") && numericValue == 1 {
					return "mv " + destinationRegister + ", " + sourceRegister
				}
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

func hasDuplicates(seenLabels []string) bool {
	seen := make(map[string]bool)
	for _, label := range seenLabels {
		if seen[label] {
			fmt.Printf("SyntaxError: Label '%s' was defined more than once.", label)
			return true
		}
		seen[label] = true
	}
	return false
}

func isSubset(seenJumps []string, seenLabels []string) bool {
	seenInLabels := make(map[string]bool)
	for _, label := range seenLabels {
		seenInLabels[label] = true
	}

	for _, jump := range seenJumps {
		if !seenInLabels[jump] {
			fmt.Printf("SyntaxError: Jumping to '%s' requires the label to be defined", jump)
			return false
		}
	}

	return true
}

func validateLabelsAndJumps(seenLabels []string, seenJumps []string) bool {
	if hasDuplicates(seenLabels) {
		return false
	}
	if !isSubset(seenJumps, seenLabels) {
		return false
	}

	return true
}

func readAndValidateSourceFile(filePath string) ([]string, error) {
	file, openingError := os.Open(filePath)
	if openingError != nil {
		return nil, openingError
	}
	defer file.Close()

	var scriptLines []string
	var seenImportPaths []string
	var seenFromImportLabels []string
	var scriptLinesOfImportPaths []string
	var scriptLinesOfFromImportPaths []string

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
		isImport := strings.HasPrefix(rawLine, "import")
		isFromImport := strings.HasPrefix(rawLine, "from")
		isComment := strings.HasPrefix(rawLine, "#")

		uncommentedLine := removeComments(rawLine)
		normalizedLine := normalizeLineWhitespace(removeComments(rawLine))

		if isComment {
			continue
		}

		importRegex := regexp.MustCompile(`^import\s+['"]([^'"]+)['"]\s*$`)

		if isImport && importRegex.MatchString(normalizedLine) {
			importPath := importRegex.FindStringSubmatch(normalizedLine)[1]

			if slices.Contains(seenImportPaths, importPath) {
				return nil, fmt.Errorf("SyntaxError on line %d: Path %s is imported more than once.", lineNumber, importPath)
			}

			seenImportPaths = append(seenImportPaths, importPath)
			scriptLinesOfImportPath, errorReadingAndValidatingSourceFile := readAndValidateSourceFile(importPath)

			if errorReadingAndValidatingSourceFile != nil {
				return nil, errorReadingAndValidatingSourceFile
			}
			scriptLinesOfImportPaths = append(scriptLinesOfImportPaths, scriptLinesOfImportPath...)
			continue
		}

		if isHeader || isEntryPoint || isDefinition {
			scriptLines = append(scriptLines, normalizedLine)
			continue
		}

		fromImportRegex := regexp.MustCompile(`^from\s+['"]([^'"]+)['"]\s+import\s+(.+)$`)

		if isFromImport && fromImportRegex.MatchString(normalizedLine) {
			fromImportPath := fromImportRegex.FindStringSubmatch(normalizedLine)[1]
			fromImportLabelsAsString := fromImportRegex.FindStringSubmatch(normalizedLine)[2]
			fromImportLabelsAsSlice := strings.Split(fromImportLabelsAsString, ",")

			for _, fromImportLabel := range fromImportLabelsAsSlice {
				normalizedFromImportLabel := strings.TrimSpace(fromImportLabel)

				if slices.Contains(seenFromImportLabels, normalizedFromImportLabel) {
					return nil, fmt.Errorf("SyntaxError on line %d: Label '%s' is imported more than once.", lineNumber, normalizedFromImportLabel)
				} else {
					seenFromImportLabels = append(seenFromImportLabels, normalizedFromImportLabel)
				}
			}

			scriptLinesOfFromImportPath, errorReadingAndValidatingSourceFile := readAndValidateSourceFile(fromImportPath)

			if errorReadingAndValidatingSourceFile != nil {
				return nil, errorReadingAndValidatingSourceFile
			}
			scriptLinesOfFromImportPaths = append(scriptLinesOfImportPaths, scriptLinesOfFromImportPath...)
			continue
		}

		if isHeader || isEntryPoint || isDefinition {
			scriptLines = append(scriptLines, normalizedLine)
			continue
		}

		isLabel := strings.HasSuffix(trimmedLine, ":")
		hasIndentation := strings.HasPrefix(uncommentedLine, " ") || strings.HasPrefix(uncommentedLine, "\t")

		if isLabel {
			if hasIndentation {
				return nil, fmt.Errorf("SyntaxError on line %d: labels must not be indented", lineNumber)
			}
			if normalizedLine != "" {
				scriptLines = append(scriptLines, normalizedLine)
			}
			insideLabelBlock = true
			continue
		}

		if insideLabelBlock {
			if !hasIndentation {
				return nil, fmt.Errorf("SyntaxError on line %d: instructions inside a label block must be indented", lineNumber)
			}
			if normalizedLine != "" {
				scriptLines = append(scriptLines, normalizedLine)
			}
			continue
		}

		return nil, fmt.Errorf("SyntaxError on line %d: instruction found outside of label block", lineNumber)
	}

	if scanningError := scanner.Err(); scanningError != nil {
		return nil, scanningError
	}

	return append(scriptLines, append(scriptLinesOfImportPaths, extractBlocks(scriptLinesOfFromImportPaths, seenFromImportLabels)...)...), nil
}

func writeAsmFile(filePath string, scriptLines []string) error {
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

func normalizeLineWhitespace(line string) string {
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

	return removeComments(resultBuilder.String())
}

func removeComments(lineContent string) string {
	NotCommentedLine, _, _ := strings.Cut(lineContent, "#")

	return NotCommentedLine
}

func parse(scriptLines []string, optimizationFlag bool) ([]string, string, string) {
	var header string
	var entrypoint string
	var err error
	var macrosAndExpansions map[string]string
	var parsedScriptLines []string
	var enteredRawAssemblyLabel bool
	var labelAndJumpsAreInSync bool
	var previousLineContent string
	var seenLabels []string
	var seenJumps []string
	var seenReturn bool

	macrosAndExpansions = make(map[string]string)

	for lineNumber, lineContent := range scriptLines {
		if strings.HasPrefix(lineContent, "!!") && !strings.HasSuffix(lineContent, ":") {
			parsedScriptLines = append(parsedScriptLines, strings.TrimLeft(lineContent, "!"))
			continue
		}
		if strings.HasPrefix(lineContent, "header") || strings.HasSuffix(lineContent, ":") || lineContent == "" || lineContent == "\n" {
			seenReturn = false
		} else {
			if seenReturn {
				fmt.Printf("SyntaxError on line %d: Cannot have any actions after a return.", lineNumber)
				goto out
			}
			seenReturn = false
		}
		if strings.HasSuffix(strings.TrimSpace(lineContent), ":") {
			if strings.HasPrefix(lineContent, "!!") {
				enteredRawAssemblyLabel = true
			} else {
				enteredRawAssemblyLabel = false
			}
			lineContent = strings.Trim(lineContent, "! ")
			seenLabels = append(seenLabels, strings.Trim(lineContent, ":"))
		}
		if enteredRawAssemblyLabel {
			parsedScriptLines = append(parsedScriptLines, lineContent)
			continue
		}

		if lineNumber == 0 {
			header, err = extractHeader(lineContent, lineNumber)
			if err != nil {
				fmt.Println(err)
				goto out
			}
			continue
		}
		if lineNumber == 1 {
			entrypoint, err = extractEntryPoint(lineContent, lineNumber)
			if err != nil {
				fmt.Println(err)
				goto out
			}
			continue
		}

		if strings.HasPrefix(lineContent, "define") {
			registerMacrosAndExpansions(lineContent, lineNumber, &macrosAndExpansions)
			continue
		}

		lineContent = replaceMacros(lineContent, &macrosAndExpansions)

		if strings.Contains(lineContent, "swap") {
			swapStatementToAsm := transformSwapToAsm(lineContent)
			parsedScriptLines = append(parsedScriptLines, swapStatementToAsm...)
			continue
		}
		if strings.Contains(lineContent, "if") {
			conditionalStatementToAsm := transformConditionalToAsm(lineContent)
			parsedScriptLines = append(parsedScriptLines, conditionalStatementToAsm...)
			continue
		}
		if strings.Contains(lineContent, "stack") {
			stackStatementToAsm := transformStackOperationToAsm(lineContent)
			parsedScriptLines = append(parsedScriptLines, stackStatementToAsm...)
			continue
		}
		if strings.Contains(lineContent, "heap") {
			heapStatementToAsm := transformHeapOperationToAsm(lineContent)
			parsedScriptLines = append(parsedScriptLines, heapStatementToAsm...)
			continue
		}

		lineContent = expandShorthands(lineContent)

		if optimizationFlag {
			lineContent = optimizeLineContent(lineContent, previousLineContent)
		}
		if lineContent != "" {
			previousLineContent = lineContent
		}
		if strings.Contains(lineContent, "return") {
			seenReturn = true
		}
		if strings.Contains(lineContent, "@") {
			seenJumps = append(seenJumps, extractJumps(lineContent)...)
		}

		lineContent = strings.ReplaceAll(lineContent, "interrupt.m", ASM_INTERRUPT_M)
		lineContent = strings.ReplaceAll(lineContent, "interrupt.s", ASM_INTERRUPT_S)
		lineContent = strings.ReplaceAll(lineContent, "interrupt.u", ASM_INTERRUPT_U)
		lineContent = strings.ReplaceAll(lineContent, "halt", ASM_HALT)
		lineContent = strings.ReplaceAll(lineContent, "return", ASM_RETURN)
		lineContent = strings.ReplaceAll(lineContent, "trap", ASM_DEBUGGER_TRAP)
		lineContent = strings.ReplaceAll(lineContent, "wait", ASM_WAIT)
		lineContent = strings.ReplaceAll(lineContent, "@", ASM_JUMP)
		lineContent = strings.ReplaceAll(lineContent, "...", ASM_PASS)
		lineContent = transformToAsm(lineContent)
		lineContent = transformLoadToAsm(lineContent)
		lineContent = transformMoveToAsm(lineContent)

		if lineContent != "" {
			parsedScriptLines = append(parsedScriptLines, lineContent)
		}
	}
	labelAndJumpsAreInSync = validateLabelsAndJumps(seenLabels, seenJumps)
	if !labelAndJumpsAreInSync {
		goto out
	}
	if !slices.Contains(seenLabels, entrypoint) {
		fmt.Printf("SyntaxError: Entrypoint specifies '%s' but '%s' label is not implemented.", entrypoint, entrypoint)
		goto out
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
		scriptLines, readingError := readAndValidateSourceFile(filePathSourceCode)
		if readingError == nil {
			if len(os.Args) == 4 && (slices.Contains(os.Args, "-o") || slices.Contains(os.Args, "--optimize")) {
				parsedScriptLines, header, entrypoint = parse(scriptLines, true)
			} else if len(os.Args) == 3 && !(slices.Contains(os.Args, "-o") || slices.Contains(os.Args, "--optimize")) {
				parsedScriptLines, header, entrypoint = parse(scriptLines, false)
			} else {
				fmt.Println("CommandLineError: Expected a source code file, an assembly target file and an optional '--optimize'/'-o' flag.")
				goto out
			}
			fullAsmCode := []string{header, entrypoint}
			fullAsmCode = append(fullAsmCode, parsedScriptLines...)
			if fullAsmCode[0] != "" {
				writtingError := writeAsmFile(filePathAsmFile, fullAsmCode)
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

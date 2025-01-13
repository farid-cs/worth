package main

import "fmt"
import "os"
import "os/exec"
import "strconv"

const (
	OP_PLUS = iota
	OP_MINUS
	OP_PUSH
	OP_DUMP
)

type Token struct {
	word string
	line int
	column int
}

type Operation struct {
	kind int
	arg int64
}

func isspace(char byte) bool {
	switch char {
	case '\t':
		fallthrough
	case '\n':
		fallthrough
	case '\v':
		fallthrough
	case '\f':
		fallthrough
	case '\r':
		fallthrough
	case ' ':
		return true
	}
	return false
}

func lex_text(text string) []Token {
	var tokens []Token
	var begin = 0
	var line = 1
	var column = 1

	for begin != len(text) {
		if !isspace(text[begin]) {
			var token Token
			var end = begin

			for end != len(text) && !isspace(text[end]) {
				end += 1
			}

			token.word = text[begin:end]
			token.column = column
			token.line = line
			tokens = append(tokens, token)

			begin = end
			continue
		}

		if text[begin] == '\n' {
			line += 1
			column = 1
		}

		begin += 1
	}

	return tokens
}

func word_to_operation(word string) Operation {
	var op Operation
	var err error

	switch word {
	case "+":
		op.kind = OP_PLUS
	case "-":
		op.kind = OP_MINUS
	case ".":
		op.kind = OP_DUMP
	default:
		op.kind = OP_PUSH
		op.arg, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			panic(err)
		}
	}

	return op
}

func generate_program(tokens []Token) []Operation {
	var program []Operation

	for _, token := range tokens {
		program = append(program, word_to_operation(token.word))
	}

	return program
}

func compile(filepath string) {
	var source []byte
	var program []Operation
	var tokens []Token
	var out *os.File
	var err error

	source, err = os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	for i := range source {
		if source[i] > 127 {
			panic("invalid ascii")
		}
	}

	tokens = lex_text(string(source))

	program = generate_program(tokens)

	out, err = os.Create("a.s")
	if err != nil {
		panic(err)
	}

	defer out.Close()

	fmt.Fprintf(out, "format ELF64 executable\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "entry _start\n")
	fmt.Fprintf(out, "\n")

	fmt.Fprintf(out, "dump:\n")
	fmt.Fprintf(out, "	mov rax, rdi\n")
	fmt.Fprintf(out, "	mov r10, 0\n")
	fmt.Fprintf(out, "	dec rsp\n")
	fmt.Fprintf(out, "	mov byte [rsp], 10\n")
	fmt.Fprintf(out, "	inc r10\n")
	fmt.Fprintf(out, ".prepend_digit:\n")
	fmt.Fprintf(out, "	mov rdx, 0\n")
	fmt.Fprintf(out, "	mov rbx, 10\n")
	fmt.Fprintf(out, "	div rbx\n")
	fmt.Fprintf(out, "	add rdx, 48\n")
	fmt.Fprintf(out, "	dec rsp\n")
	fmt.Fprintf(out, "	mov [rsp], dl\n")
	fmt.Fprintf(out, "	inc r10\n")
	fmt.Fprintf(out, "	cmp rax, 0\n")
	fmt.Fprintf(out, "	jne .prepend_digit\n")
	fmt.Fprintf(out, ".print_digit:\n")
	fmt.Fprintf(out, "	mov rax, 1\n")
	fmt.Fprintf(out, "	mov rdi, 1\n")
	fmt.Fprintf(out, "	mov rsi, rsp\n")
	fmt.Fprintf(out, "	mov rdx, r10\n")
	fmt.Fprintf(out, "	syscall\n")
	fmt.Fprintf(out, "	add rsp, r10\n")
	fmt.Fprintf(out, "	ret\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "_start:\n")

	for _, op := range program {
		switch op.kind {
		case OP_PLUS:
			fmt.Fprintf(out, "	pop rdi\n")
			fmt.Fprintf(out, "	pop rax\n")
			fmt.Fprintf(out, "	add rax, rdi\n")
			fmt.Fprintf(out, "	push rax\n")
		case OP_MINUS:
			fmt.Fprintf(out, "	pop rdi\n")
			fmt.Fprintf(out, "	pop rax\n")
			fmt.Fprintf(out, "	sub rax, rdi\n")
			fmt.Fprintf(out, "	push rax\n")
		case OP_PUSH:
			fmt.Fprintf(out, "	push %d\n", op.arg)
		case OP_DUMP:
			fmt.Fprintf(out, "	pop rdi\n")
			fmt.Fprintf(out, "	call dump\n")
		}
	}

	fmt.Fprintf(out, "	mov rax, 60\n")
	fmt.Fprintf(out, "	mov rdi, 0\n")
	fmt.Fprintf(out, "	syscall\n")

}

func main() {
	fasm := exec.Command("fasm", "a.s", "a.out")

	if len(os.Args) < 2 {
		fmt.Printf("usage: %s <filepath>\n", os.Args[0])
		os.Exit(1)
	}

	compile(os.Args[1])
	fasm.Run()
}

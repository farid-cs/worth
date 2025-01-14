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
	OP_EQUAL
	OP_IF
	OP_FI
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

			column += end - begin
			begin = end
			continue
		}

		if text[begin] == '\n' {
			line += 1
			column = 0
		}

		begin += 1
		column += 1
	}

	return tokens
}

func token_to_operation(tok Token) Operation {
	var op Operation
	var err error

	switch tok.word {
	case "+":
		op.kind = OP_PLUS
	case "-":
		op.kind = OP_MINUS
	case ".":
		op.kind = OP_DUMP
	case "=":
		op.kind = OP_EQUAL
	case "if":
		op.kind = OP_IF
	case "fi":
		op.kind = OP_FI
	default:
		op.kind = OP_PUSH
		op.arg, err = strconv.ParseInt(tok.word, 10, 64)
		if err != nil {
			fmt.Printf("%d:%d: %s\n", tok.line, tok.column, err)
			os.Exit(1)
		}
	}

	return op
}

func generate_program(tokens []Token) []Operation {
	var program []Operation

	for _, token := range tokens {
		program = append(program, token_to_operation(token))
	}

	return program
}

func compile(filepath string) {
	var source []byte
	var program []Operation
	var tokens []Token
	var out *os.File
	var branch_count = 0
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

	out.WriteString("format ELF64 executable\n")
	out.WriteString("\n")
	out.WriteString("entry _start\n")
	out.WriteString("\n")

	out.WriteString("dump:\n")
	out.WriteString("	mov rax, rdi\n")
	out.WriteString("	mov r10, 0\n")
	out.WriteString("	dec rsp\n")
	out.WriteString("	mov byte [rsp], 10\n")
	out.WriteString("	inc r10\n")
	out.WriteString(".prepend_digit:\n")
	out.WriteString("	mov rdx, 0\n")
	out.WriteString("	mov rbx, 10\n")
	out.WriteString("	div rbx\n")
	out.WriteString("	add rdx, 48\n")
	out.WriteString("	dec rsp\n")
	out.WriteString("	mov [rsp], dl\n")
	out.WriteString("	inc r10\n")
	out.WriteString("	cmp rax, 0\n")
	out.WriteString("	jne .prepend_digit\n")
	out.WriteString(".print_digit:\n")
	out.WriteString("	mov rax, 1\n")
	out.WriteString("	mov rdi, 1\n")
	out.WriteString("	mov rsi, rsp\n")
	out.WriteString("	mov rdx, r10\n")
	out.WriteString("	syscall\n")
	out.WriteString("	add rsp, r10\n")
	out.WriteString("	ret\n")
	out.WriteString("\n")
	out.WriteString("_start:\n")

	for _, op := range program {
		switch op.kind {
		case OP_PLUS:
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	add rax, rdi\n")
			out.WriteString("	push rax\n")

		case OP_MINUS:
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	sub rax, rdi\n")
			out.WriteString("	push rax\n")

		case OP_PUSH:
			fmt.Fprintf(out, "	push %d\n", op.arg)

		case OP_DUMP:
			out.WriteString("	pop rdi\n")
			out.WriteString("	call dump\n")

		case OP_EQUAL:
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rdx\n")
			out.WriteString("	xor rax, rax\n")
			out.WriteString("	cmp rdx, rdi\n")
			out.WriteString("	sete al\n")
			out.WriteString("	push rax\n")

		case OP_IF:
			out.WriteString("	pop rdi\n")
			out.WriteString("	test rdi, rdi\n")
			fmt.Fprintf(out, "	je .L%d\n", branch_count)

		case OP_FI:
			fmt.Fprintf(out, ".L%d:\n", branch_count)
			branch_count++
		}
	}

	out.WriteString("	mov rax, 60\n")
	out.WriteString("	mov rdi, 0\n")
	out.WriteString("	syscall\n")

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

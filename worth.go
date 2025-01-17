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
	OP_ELSE
	OP_FI
	OP_DUP
	OP_GT
	OP_WHILE
	OP_DO
	OP_DONE
)

type Token struct {
	kind int
	push int64
	line int
	column int
}

type Operation struct {
	kind int
	arg int64
	line int
	column int
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

func NewToken(word string, line int, column int) Token {
	var tok Token
	var err error

	tok.line = line
	tok.column = column

	switch word {
	case "+":
		tok.kind = OP_PLUS
	case "-":
		tok.kind = OP_MINUS
	case ".":
		tok.kind = OP_DUMP
	case "=":
		tok.kind = OP_EQUAL
	case "if":
		tok.kind = OP_IF
	case "else":
		tok.kind = OP_ELSE
	case "fi":
		tok.kind = OP_FI
	case "dup":
		tok.kind = OP_DUP
	case ">":
		tok.kind = OP_GT
	case "while":
		tok.kind = OP_WHILE
	case "do":
		tok.kind = OP_DO
	case "done":
		tok.kind = OP_DONE
	default:
		tok.push, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			fmt.Printf("%d:%d: %s\n", tok.line, tok.column, err)
			os.Exit(1)
		}
		tok.kind = OP_PUSH
	}
	return tok
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

			token = NewToken(text[begin:end], line, column)
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

func generate_program(tokens []Token) []Operation {
	var program []Operation
	var branch_count int64 = 0
	var stack []int64

	for _, tok := range tokens {
		op := Operation{arg: -1}

		op.kind = tok.kind
		op.line = tok.line
		op.column = tok.column
		if op.kind == OP_PUSH {
			op.arg = tok.push
		}

		program = append(program, op)
	}

	for i := 0; i != len(program); i++ {
		switch program[i].kind {
		case OP_IF:
			program[i].arg = branch_count
			stack = append(stack, branch_count)
			branch_count++

		case OP_ELSE:
			if len(stack) == 0 {
				fmt.Printf("%d:%d: `else` of non-existent if block\n", program[i].line, program[i].column)
				os.Exit(1)
			}
			program[i].arg = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			stack = append(stack, branch_count)
			branch_count++

		case OP_FI:
			if len(stack) == 0 {
				fmt.Printf("%d:%d: `fi` of a non-existent if block\n", program[i].line, program[i].column)
				os.Exit(1)
			}
			program[i].arg = stack[len(stack)-1]
			stack = stack[:len(stack)-1]

		case OP_WHILE:
			program[i].arg = branch_count
			stack = append(stack, branch_count)
			branch_count++

		case OP_DO:
			program[i].arg = branch_count
			stack = append(stack, branch_count)
			branch_count++

		case OP_DONE:
			if len(stack) < 2 {
				fmt.Printf("%d:%d: `done` of a non-existent `while` or `do` block\n", program[i].line, program[i].column)
				os.Exit(1)
			}
			program[i].arg = stack[len(stack)-2]
			stack = stack[:len(stack)-2]
		}
	}

	if len(stack) > 0 {
		fmt.Println("unterminated while or if block")
		os.Exit(1)
	}

	return program
}

func translate_to_assembly(program []Operation) {
	out, err := os.Create("a.s")
	if err != nil {
		panic(err)
	}

	defer out.Close()

	out.WriteString("format ELF64 executable\n")
	out.WriteString("\n")
	out.WriteString("entry _start\n")
	out.WriteString("\n")

	out.WriteString("dump:\n")
	out.WriteString("	mov	rax, rdi\n")
	out.WriteString("	mov	r10, 0\n")
	out.WriteString("	dec	rsp\n")
	out.WriteString("	mov	byte [rsp], 10\n")
	out.WriteString("	inc	r10\n")
	out.WriteString(".prepend_digit:\n")
	out.WriteString("	mov	rdx, 0\n")
	out.WriteString("	mov	rbx, 10\n")
	out.WriteString("	div	rbx\n")
	out.WriteString("	add	rdx, 48\n")
	out.WriteString("	dec	rsp\n")
	out.WriteString("	mov	[rsp], dl\n")
	out.WriteString("	inc	r10\n")
	out.WriteString("	cmp	rax, 0\n")
	out.WriteString("	jne	.prepend_digit\n")
	out.WriteString(".print_digit:\n")
	out.WriteString("	mov	rax, 1\n")
	out.WriteString("	mov	rdi, 1\n")
	out.WriteString("	mov	rsi, rsp\n")
	out.WriteString("	mov	rdx, r10\n")
	out.WriteString("	syscall\n")
	out.WriteString("	add	rsp, r10\n")
	out.WriteString("	ret\n")
	out.WriteString("\n")
	out.WriteString("_start:\n")

	for _, op := range program {
		switch op.kind {
		case OP_PLUS:
			out.WriteString("	;; -- add --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	pop	rax\n")
			out.WriteString("	add	rax, rdi\n")
			out.WriteString("	push	rax\n")

		case OP_MINUS:
			out.WriteString("	;; -- subtract --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	pop	rax\n")
			out.WriteString("	sub	rax, rdi\n")
			out.WriteString("	push	rax\n")

		case OP_PUSH:
			out.WriteString("	;; -- push --\n")
			fmt.Fprintf(out, "	push	%d\n", op.arg)

		case OP_DUMP:
			out.WriteString("	;; -- dump --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	call	dump\n")

		case OP_EQUAL:
			out.WriteString("	;; -- equal --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	pop	rdx\n")
			out.WriteString("	xor	rax, rax\n")
			out.WriteString("	cmp	rdx, rdi\n")
			out.WriteString("	sete	al\n")
			out.WriteString("	push	rax\n")

		case OP_IF:
			out.WriteString("	;; -- if --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	test	rdi, rdi\n")
			fmt.Fprintf(out, "	je	.L%d\n", op.arg)

		case OP_ELSE:
			out.WriteString("	;; -- else --\n")
			fmt.Fprintf(out, "	jmp .L%d\n", op.arg + 1)
			fmt.Fprintf(out, ".L%d:\n", op.arg)

		case OP_FI:
			out.WriteString("	;; -- fi --\n")
			fmt.Fprintf(out, ".L%d:\n", op.arg)

		case OP_DUP:
			out.WriteString("	;; -- dup --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	push	rdi\n")
			out.WriteString("	push	rdi\n")

		case OP_GT:
			out.WriteString("	;; -- greater --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	pop	rdx\n")
			out.WriteString("	xor	rax, rax\n")
			out.WriteString("	cmp	rdx, rdi\n")
			out.WriteString("	setg	al\n")
			out.WriteString("	push	rax\n")

		case OP_WHILE:
			out.WriteString("	;; -- while --\n")
			fmt.Fprintf(out, ".L%d:\n", op.arg)

		case OP_DO:
			out.WriteString("	;; -- do --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	test	rdi, rdi\n")
			fmt.Fprintf(out, "	je	.L%d\n", op.arg)

		case OP_DONE:
			out.WriteString("	;; -- done --\n")
			fmt.Fprintf(out, "	jmp .L%d\n", op.arg)
			fmt.Fprintf(out, ".L%d:\n", op.arg + 1)
		}
	}

	out.WriteString("	mov	rax, 60\n")
	out.WriteString("	mov	rdi, 0\n")
	out.WriteString("	syscall\n")
}

func compile(filepath string) {
	var source []byte
	var tokens []Token
	var program []Operation
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

	translate_to_assembly(program)
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

package main

import "fmt"
import "os"
import "os/exec"
import "strconv"

const (
	TOK_WORD = iota
	TOK_INT
)

const (
	OP_PLUS = iota
	OP_MINUS
	OP_PUSH
	OP_DUMP
	OP_DROP
	OP_MEM
	OP_LOAD
	OP_STORE
	OP_SYSCALL0
	OP_SYSCALL1
	OP_SYSCALL2
	OP_SYSCALL3
	OP_SYSCALL4
	OP_SYSCALL5
	OP_SYSCALL6
	OP_EQUAL
	OP_IF
	OP_ELSE
	OP_FI
	OP_DUP
	OP_GT
	OP_WHILE
	OP_DO
	OP_DONE
	OP_QUIT
)

const MEM_CAPACITY = 600_000

type Token struct {
	kind int
	ivalue int
	wvalue string
	line int
	column int
}

type Operation struct {
	kind int
	arg int
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

func lex_word(text string, line int, column int) (Token, string, int, int) {
	var tok Token
	var toklen int
	var err error

	for toklen != len(text) && !isspace(text[toklen]) {
		toklen += 1
	}

	tok.ivalue, err = strconv.Atoi(text[:toklen])
	if err == nil {
		column += toklen
		tok.kind = TOK_INT
		return tok, text[toklen:], line, column
	}

	column += toklen
	tok.kind = TOK_WORD
	tok.wvalue = text[:toklen]
	return tok, text[toklen:], line, column
}

func lex_text(text string) []Token {
	var tokens []Token
	var token Token
	var line = 1
	var column = 1

	for len(text) != 0 {
		if isspace(text[0]) {
			if text[0] == '\n' {
				line += 1
				column = 0
			}
			text = text[1:]
			column += 1
			continue
		}

		token, text, line, column = lex_word(text, line, column)
		tokens = append(tokens, token)
	}

	return tokens
}

func token_to_operation(tok Token) Operation {
	op := Operation{line: tok.line, column: tok.column}

	if tok.kind == TOK_INT {
		op.kind = OP_PUSH
		op.arg = tok.ivalue
		return op
	}

	switch tok.wvalue {
	case "+":
		op.kind = OP_PLUS
	case "-":
		op.kind = OP_MINUS
	case "dump":
		op.kind = OP_DUMP
	case "=":
		op.kind = OP_EQUAL
	case "if":
		op.kind = OP_IF
	case "else":
		op.kind = OP_ELSE
	case "fi":
		op.kind = OP_FI
	case "dup":
		op.kind = OP_DUP
	case ">":
		op.kind = OP_GT
	case "while":
		op.kind = OP_WHILE
	case "do":
		op.kind = OP_DO
	case "done":
		op.kind = OP_DONE
	case "drop":
		op.kind = OP_DROP
	case "mem":
		op.kind = OP_MEM
	case ",":
		op.kind = OP_LOAD
	case ".":
		op.kind = OP_STORE
	case "syscall0":
		op.kind = OP_SYSCALL0
	case "syscall1":
		op.kind = OP_SYSCALL1
	case "syscall2":
		op.kind = OP_SYSCALL2
	case "syscall3":
		op.kind = OP_SYSCALL3
	case "syscall4":
		op.kind = OP_SYSCALL4
	case "syscall5":
		op.kind = OP_SYSCALL5
	case "syscall6":
		op.kind = OP_SYSCALL6
	case "quit":
		op.kind = OP_QUIT
	}
	return op
}

func generate_program(tokens []Token) []Operation {
	var program []Operation
	var stack []int

	for _, tok := range tokens {
		program = append(program, token_to_operation(tok))
	}

	for addr, op := range program {
		switch op.kind {
		case OP_IF:
			stack = append(stack, addr)

		case OP_ELSE:
			if len(stack) < 1 {
				fmt.Fprintf(os.Stderr, "%d:%d: `else` of non-existent if block\n", op.line, op.column)
				os.Exit(1)
			}
			program[stack[len(stack)-1]].arg = addr
			stack = stack[:len(stack)-1]
			stack = append(stack, addr)

		case OP_FI:
			if len(stack) < 1 {
				fmt.Fprintf(os.Stderr, "%d:%d: `fi` of a non-existent if block\n", op.line, op.column)
				os.Exit(1)
			}
			program[stack[len(stack)-1]].arg = addr
			stack = stack[:len(stack)-1]

		case OP_WHILE:
			stack = append(stack, addr)

		case OP_DO:
			stack = append(stack, addr)

		case OP_DONE:
			if len(stack) < 2 {
				fmt.Fprintf(os.Stderr, "%d:%d: `done` of a non-existent `while` or `do` block\n", op.line, op.column)
				os.Exit(1)
			}
			program[stack[len(stack)-1]].arg = addr;
			program[addr].arg = stack[len(stack)-2];
			stack = stack[:len(stack)-2]
		}
	}

	if len(stack) > 0 {
		fmt.Fprintln(os.Stderr, "unterminated while or if block")
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
	out.WriteString("segment readable executable\n")
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

	for addr, op := range program {
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

		case OP_DROP:
			out.WriteString("	;; -- drop --\n")
			out.WriteString("	pop	rdi\n")

		case OP_MEM:
			out.WriteString("	;; -- mem --\n")
			out.WriteString("	push	mem\n")

		case OP_LOAD:
			out.WriteString("	;; -- load --\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	xor rdi, rdi\n")
			out.WriteString("	mov dil, [rax]\n")
			out.WriteString("	push rdi\n")

		case OP_STORE:
			out.WriteString("	;; -- store --\n")
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	mov [rax], dil\n")

		case OP_SYSCALL0:
			out.WriteString("	;; -- syscall1 --\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	syscall\n")

		case OP_SYSCALL1:
			out.WriteString("	;; -- syscall2 --\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	pop rdi\n")
			out.WriteString("	syscall\n")

		case OP_SYSCALL2:
			out.WriteString("	;; -- syscall3 --\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rsi\n")
			out.WriteString("	syscall\n")

		case OP_SYSCALL3:
			out.WriteString("	;; -- syscall4 --\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rsi\n")
			out.WriteString("	pop rdx\n")
			out.WriteString("	syscall\n")

		case OP_SYSCALL4:
			out.WriteString("	;; -- syscall5 --\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rsi\n")
			out.WriteString("	pop rdx\n")
			out.WriteString("	pop r10\n")
			out.WriteString("	syscall\n")

		case OP_SYSCALL5:
			out.WriteString("	;; -- syscall6 --\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rsi\n")
			out.WriteString("	pop rdx\n")
			out.WriteString("	pop r10\n")
			out.WriteString("	pop r8\n")
			out.WriteString("	syscall\n")

		case OP_SYSCALL6:
			out.WriteString("	;; -- syscall7 --\n")
			out.WriteString("	pop rax\n")
			out.WriteString("	pop rdi\n")
			out.WriteString("	pop rsi\n")
			out.WriteString("	pop rdx\n")
			out.WriteString("	pop r10\n")
			out.WriteString("	pop r8\n")
			out.WriteString("	pop r9\n")
			out.WriteString("	syscall\n")

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
			fmt.Fprintf(out, "	je	.addr_%d\n", op.arg)

		case OP_ELSE:
			out.WriteString("	;; -- else --\n")
			fmt.Fprintf(out, "	jmp .addr_%d\n", op.arg)
			fmt.Fprintf(out, ".addr_%d:\n", addr)

		case OP_FI:
			out.WriteString("	;; -- fi --\n")
			fmt.Fprintf(out, ".addr_%d:\n", addr)

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
			fmt.Fprintf(out, ".addr_%d:\n", addr)

		case OP_DO:
			out.WriteString("	;; -- do --\n")
			out.WriteString("	pop	rdi\n")
			out.WriteString("	test	rdi, rdi\n")
			fmt.Fprintf(out, "	je	.addr_%d\n", op.arg)

		case OP_DONE:
			out.WriteString("	;; -- done --\n")
			fmt.Fprintf(out, "	jmp .addr_%d\n", op.arg)
			fmt.Fprintf(out, ".addr_%d:\n", addr)

		case OP_QUIT:
			out.WriteString("	;; -- quit --\n")
			out.WriteString("	mov	rax, 60\n")
			out.WriteString("	mov	rdi, 0\n")
			out.WriteString("	syscall\n")
		}
	}

	out.WriteString("\n")
	fmt.Fprintf(out, "segment readable writable\n")
	fmt.Fprintf(out, "mem: rb %d\n", MEM_CAPACITY)
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
			fmt.Fprintf(os.Stderr, "error: invalid ascii\n")
			os.Exit(1)
		}
	}

	tokens = lex_text(string(source))

	program = generate_program(tokens)

	translate_to_assembly(program)
}

func main() {
	fasm := exec.Command("fasm", "a.s", "a.out")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <filepath>\n", os.Args[0])
		os.Exit(1)
	}

	compile(os.Args[1])
	if fasm.Run() != nil {
		fmt.Fprintf(os.Stderr, "error: fasm failed\n")
		os.Exit(1)
	}
}

use std::fs::File;
use std::io::Write;
use std::process::Command;

enum Operation {
	Plus,
	Minus,
	Push(i64),
	Dump,
}

fn compile(program: &[Operation], filepath: &str) {
	let mut out = File::create(filepath).expect("create file");

	writeln!(out, "format ELF64 executable");
	writeln!(out, "");
	writeln!(out, "entry _start");
	writeln!(out, "");

	writeln!(out, "dump:");
	writeln!(out, "	mov rax, rdi");
	writeln!(out, "	mov r10, 0");
	writeln!(out, "	dec rsp");
	writeln!(out, "	mov byte [rsp], 10");
	writeln!(out, "	inc r10");
	writeln!(out, ".prepend_digit:");
	writeln!(out, "	mov rdx, 0");
	writeln!(out, "	mov rbx, 10");
	writeln!(out, "	div rbx");
	writeln!(out, "	add rdx, 48");
	writeln!(out, "	dec rsp");
	writeln!(out, "	mov [rsp], dl");
	writeln!(out, "	inc r10");
	writeln!(out, "	cmp rax, 0");
	writeln!(out, "	jne .prepend_digit");
	writeln!(out, ".print_digit:");
	writeln!(out, "	mov rax, 1");
	writeln!(out, "	mov rdi, 1");
	writeln!(out, "	mov rsi, rsp");
	writeln!(out, "	mov rdx, r10");
	writeln!(out, "	syscall");
	writeln!(out, "	add rsp, r10");
	writeln!(out, "	ret");
        writeln!(out, "");
	writeln!(out, "_start:");

	for operation in program {
		match operation {
			Operation::Plus => {
				writeln!(out, "	pop rax");
				writeln!(out, "	pop rdi");
				writeln!(out, "	add rax, rdi");
				writeln!(out, "	push rax");
			}
			Operation::Minus => {
				writeln!(out, "	pop rdi");
				writeln!(out, "	pop rax");
				writeln!(out, "	sub rax, rdi");
				writeln!(out, "	push rax");
			}
			Operation::Push(x) => {
				writeln!(out, "	push {}", x);
			}
			Operation::Dump => {
				writeln!(out, "	pop rdi");
				writeln!(out, "	call dump");
			}
		}
	}

	writeln!(out, "	mov rax, 60");
	writeln!(out, "	mov rdi, 0");
	writeln!(out, "	syscall");
}

fn main() {
	let program = [
		Operation::Push(10),
		Operation::Push(15),
		Operation::Plus,
		Operation::Dump,
	];

	compile(&program, "sample.s");

	Command::new("fasm")
		.arg("sample.s")
		.spawn()
		.expect("failed to assemble");
}

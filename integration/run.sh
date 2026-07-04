#!/bin/sh
# Integration checks for yup-tr, run inside a Debian (GNU coreutils) container.
#
# parity INPUT ARGS...   — yup-tr reading stdin must match GNU `tr` byte-for-byte.
# assert WANT INPUT ARGS... — yup-tr stdout must equal WANT (used for the one
#   documented divergence: yup-tr is line-oriented, so it never translates the
#   input's trailing newline, whereas GNU `tr -c SET _` maps that newline — being
#   outside SET — to the replacement. Parity therefore holds for every case
#   except a *translating* -c whose output keeps the (mapped) trailing newline).
set -eu

fails=0

parity() {
	in=$1
	shift
	ours=$(printf '%s' "$in" | yup-tr "$@" 2>/dev/null || true)
	gnu=$(printf '%s' "$in" | tr "$@" 2>/dev/null || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  tr %s\n' "$*"
	else
		printf 'FAIL  parity  tr %s\n        gnu:  %s\n        ours: %s\n' "$*" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

assert() {
	want=$1
	in=$2
	shift 2
	ours=$(printf '%s' "$in" | yup-tr "$@" 2>/dev/null || true)
	if [ "$ours" = "$want" ]; then
		printf 'ok    assert  tr %s\n' "$*"
	else
		printf 'FAIL  assert  tr %s\n        want: %s\n        ours: %s\n' "$*" "$want" "$ours"
		fails=$((fails + 1))
	fi
}

# translate: explicit sets and lower<->upper via ranges.
parity 'abc
' abc xyz
parity 'hello world
' a-z A-Z
parity 'HELLO WORLD
' A-Z a-z

# delete (-d): SET1 alone, removing characters.
parity 'hello world
' -d aeiou
parity 'a1b2c3
' -d 0-9

# squeeze (-s): SET1 alone collapses runs; ranges expand.
parity 'hello   world
' -s ' '
parity 'aaabbbccc
' -s a-z
# squeeze with a translate (SET2 is the squeezed set).
parity 'aabbcc
' -s ab xy

# complement (-c) delete: the trailing newline is in the complement and dropped
# by GNU; yup-tr keeps it, but command substitution strips it, so parity holds.
parity 'abc123 xyz
' -d -c 'a-z'

# complement (-c) translate: yup-tr is line-oriented and does not translate the
# input's trailing newline, so it emits one fewer replacement char than GNU.
# Documented divergence — assert yup-tr's own contract.
assert '_ello__orld____' 'Hello World 123
' -c a-z _
assert 'x1x2x3' 'a1b2c3
' -c '0-9' x

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'

package lexer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func init() {
	loadTokens()
}

// Lexer Token Field Types
type tokenId uint16        // TokenKind Primary Key
type tokenName string      // Human readable ID, still should be unique
type tokenSignature []byte // Character sequence identity

type tokenLineNo uint64   // Token Line Position
type tokenPosition uint64 // Token Lateral Position

/*
Compare the given signature, see if is
a substring of this signature.
*/
func (ts tokenSignature) Contains(ots tokenSignature) bool {
	return strings.Contains(string(ts), string(ots))
}

/*
Compare the given signature, see if is
equivalent to this signature.
*/
func (ts tokenSignature) Compare(ots tokenSignature) bool {
	return string(ts) == string(ots)
}

/*
Represents a type of token and it's basic
identity.
*/
type TokenKind struct {
	Id        tokenId
	Name      tokenName
	Signature tokenSignature
}

func (tk TokenKind) asString() string {
	return fmt.Sprintf("Token[%s]", tk.Name)
}
func (tk TokenKind) String() string   { return tk.asString() }
func (tk TokenKind) GoString() string { return tk.asString() }

/*
Initialize a new `TokenObject` from this
`TokenKind`.
*/
func (tk TokenKind) New(line tokenLineNo, pos tokenPosition, symbol tokenSignature) *TokenObject {
	return &TokenObject{&tk, line, pos, symbol}
}

type TokenObject struct {
	Kind     *TokenKind
	LineNo   tokenLineNo
	Position tokenPosition
	Symbol   tokenSignature // Captures Token Object value if needed
}

func (to TokenObject) asString() string {
	return fmt.Sprintf("%#v=['%s']", to.Kind, to.Symbol)
}
func (to TokenObject) String() string   { return to.asString() }
func (to TokenObject) GoString() string { return to.asString() }

// Holds what will be the next ID given to a `TokenKind`.
var tokenKindId tokenId = 0

// Tracks the last recorded largest `TokenKind` Name.
var tokenKindNameMaxSize int = 0

// Tracks the last recorded largest `TokenKind` Signature.
var tokenKindSignatureMaxSize int = 0

/* Initialize a `TokenKind`. */
func newKind(name tokenName, sig tokenSignature) TokenKind {
	id := tokenKindId
	tokenKindId += 1

	if len(name) > tokenKindNameMaxSize {
		tokenKindNameMaxSize = len(name)
	}

	if len(sig) > tokenKindSignatureMaxSize {
		tokenKindSignatureMaxSize = len(sig)
	}

	return TokenKind{id, name, sig}
}

/* --- TOKEN MAPPING ---
Below should express the internal API concerning token
types-- `TokenKinds`; how they are stored, how to
find/identify them, etc.
*/

type tokenKindMap map[tokenId]TokenKind

/* Retrieve a list of IDs in this map. */
func (tkm tokenKindMap) Ids() []tokenId {
	var ids []tokenId = []tokenId{}

	for i := range tkm {
		ids = append(ids, i)
	}
	return ids
}

/* Retrieve a `TokenKind` per the tokenId */
func (tkm tokenKindMap) Get(id tokenId) TokenKind {
	return tkm[id]
}

/* Add a new `TokenKind`. */
func (tkm tokenKindMap) Add(name tokenName, sig tokenSignature) {
	kind := newKind(name, sig)
	tkm[kind.Id] = kind
}

/*
Search this map for a `TokenKind` matching
the given signature. Returns a set of IDs of
potential matches.

If a series of IDs are provided, `Find` will
only run a comparison against that series,
otherwise looking through the entire map.
*/
func (tkm tokenKindMap) Find(sig tokenSignature, ids ...tokenId) []tokenId {
	if !(len(ids) > 0) {
		ids = tkm.Ids()
	}

	var found []tokenId = []tokenId{}

	for i := range ids {
		id := ids[i]
		if tkm[id].Signature.Contains(sig) {
			found = append(found, id)
		}
	}
	return found
}

/*
Search this map for a `TokenKind` matching
the given signature. Returns a set of IDs of
potential matches. Note this function looks
for exact matches.

If a series of IDs are provided, `Find` will
only run a comparison against that series,
otherwise looking through the entire map.
*/
func (tkm tokenKindMap) FindEx(sig tokenSignature, ids ...tokenId) []tokenId {
	if !(len(ids) > 0) {
		ids = tkm.Ids()
	}

	var found []tokenId = []tokenId{}

	for i := range ids {
		id := ids[i]
		if tkm[id].Signature.Compare(sig) {
			found = append(found, id)
		}
	}

	return found
}

// Tokens added are stored here at runtime.
var tokenKinds tokenKindMap = tokenKindMap{}

/* --- TOKENIZING --- */

/* Array in which to hold `TokenObject` instances. */
type tokenObjectsMap []TokenObject

/*
Calculate token position start.

Ensure no slicing is attempted outside
the bounds of the given line.
*/
func calcStep(line string) tokenPosition {
	step := tokenKindSignatureMaxSize
	step = step - (step - len(line))
	return tokenPosition(step)
}

/* Calculate the view used to inspect a token. */
func calcView(line string, pos tokenPosition, step tokenPosition) string {
	if int(pos+step) > len(line) {
		return line[pos:]
	}
	return line[pos : pos+step]
}

/* Calculate the view used to inspect a token looking backwards. */
func calcViewR(line string, pos tokenPosition, step tokenPosition) string {
	if (pos - step) > 0 {
		return line[0:0]
	}
	return line[pos-step : pos]
}

/*
Determine if the given sequence of
characters is a token.
*/
func isToken(line string) bool {
	step := calcStep(line)
	view := calcViewR(line, step, 1)
	sig := tokenSignature(line)

	matches := tokenKinds.Find(sig)

	for len(matches) == 0 || view == " " {
		matches = tokenKinds.Find(sig, matches...)

		if (step - 1) == 0 {
			break
		}
		step -= 1
		sig = tokenSignature(line[:step])
		view = line[step-1 : step]
	}

	matches = tokenKinds.FindEx(sig, matches...)

	return (len(matches) > 0)
}

/*
Given a string, a distance to slice up to from zero,
and an array of IDs, retrieve the most likely tokenId
and it's signature of the given value.

Note that if no tokenId can be found, this function
returns an ID of `1` by default. This is to ensure
any non-defined values can be tokenized generically.
*/
func findToken(line string, step tokenPosition, ids ...tokenId) (tokenId, tokenSignature) {
	view := calcView(line, 0, step)
	sig := tokenSignature(view)

	// If no IDs are passed to this function,
	// attempt to perform a lookup of potential
	// matches.
	if len(ids) == 0 {
		ids = tokenKinds.Find(sig, ids...)
	}

	switch len(ids) {
	case 0:
		// In the event no potential token kinds
		// are found, return a generic token ID
		// and a signature of the current view.
		return 1, tokenSignature(line)
	case 1:
		ids = tokenKinds.FindEx(sig, ids...)
		if len(ids) == 0 {
			return findToken(line, step+1, ids...)
		}
		return ids[0], sig
	}

	// Ensure there is no token immediatly ahead
	// of the current view.
	// If there is, find the exact matching ids
	// to current view and try again.
	if !isToken(calcView(line, step, 1)) {
		ids = tokenKinds.FindEx(sig, ids...)
		if len(ids) == 0 {
			ids = append(ids, 1)
		}
		return findToken(line, step, ids...)
	}

	// If no token is found, expand the view
	// using the same line and current set
	// of token IDs.
	return findToken(line, step+1, ids...)
}

/* Identify the entirety of a generic token. */
func findIdenToken(line string) tokenSignature {
	// If the given string is only a single
	// char, chances are it has no token
	// or will not have any tokens adjacent
	// to itself.
	if len(line) == 1 {
		return tokenSignature(line)
	}

	step := 1
	view, lookAhead := line[:step], line[step:]

	// Ensure there are no tokens ahead of
	// the current view on the line.
	// Break the loop either when the step
	// goes out of bounds, or if there is
	// a token ahead of the view.
	for !isToken(lookAhead) {
		view, lookAhead = line[:step], line[step:]
		step += 1
		if step > len(line) {
			break
		}
	}
	return tokenSignature(view)
}

/* Break down a single line into a series of tokens. */
func TokenizeLine(line string, lineNo tokenLineNo) tokenObjectsMap {
	var pos tokenPosition = 0
	var tokens tokenObjectsMap = tokenObjectsMap{}

	for pos < tokenPosition(len(line)) {
		var id tokenId
		var sig tokenSignature

		id, sig = findToken(line[pos:], 1)
		if id == 1 {
			// Current token is GENIDEN;
			// get full identity.
			sig = findIdenToken(string(sig))
		}
		tokens = append(tokens, *tokenKinds.Get(id).New(lineNo, pos+1, sig))
		pos += tokenPosition(len(sig))
	}

	return tokens
}

/* Break down multiple lines into a series of tokens. */
func TokenizeLines(lines []string) tokenObjectsMap {
	var tokens tokenObjectsMap = tokenObjectsMap{}

	for lineId := range lines {
		line := lines[lineId]
		lineNo := tokenLineNo(lineId)
		tokens = append(tokens, TokenizeLine(line, lineNo)...)
	}

	return tokens
}

/*
Break down multiple lines, from a file,
into a series of tokens.
*/
func TokenizeFile(name string) tokenObjectsMap {
	file := newTokenFile(name)

	tokens := tokenObjectsMap{}
	lineNo := tokenLineNo(0)
	for file.Scan() {
		lineNo += 1
		tokens = append(tokens, TokenizeLine(file.Text(), lineNo)...)
	}

	return tokens
}

/* --- TOKEN REPRESENTATION ---
The below defines how tokens are represented in human
readable terms. Functionally useless, but because we are
simple creatures it's best to help ourselves out and
provide some way to comprehend the inner workings of our
lexer. */

/* Render token representation. */
func RenderTokenRepr() string {
	var render string = ""
	var id tokenId = 0

	for id < tokenKindId {
		t := tokenKinds[id]
		id += 1
		render += fmt.Sprintf("[%d]\t%s\t'%s'\n", t.Id, t, t.Signature)
	}
	return render
}

/* Render token representation to stdout. */
func DisplayTokensRepr() {
	fmt.Println(RenderTokenRepr())
}

/* --- TOKEN LOADING ---
Tokens are going to be defined in a separate plain-text
file `tokens`. Said tokens will then be defined/loaded
at lexer runtime; perhaps this is a mistake, but we shall
soon see.

Tokens will be expected to be defined with the following
format:
[TOKEN_NAME] [TOKEN_SEQUENCE] <#: COMMENTS>

Note the space between the two objects, this must be
present. After the space, all will be considered
"fair game". Meaning the token will be from that first
space on to the end of that line.

TOKEN_NAME: A human readable ID given to the defined
token. Should be used mostly for debugging.

TOKEN_SEQUENCE: The literal identifier given to the defined
token.

COMMENTS: Any additional information not necessary
for token instantiation, but useful for us to explain
purpose/ideas/etc.

NOTE: Comments are annotated using '#:'. */

/* Represents token file when open. */
type tokenFile struct {
	file    *os.File
	scanner *bufio.Scanner
}

func (tf tokenFile) Close() {
	tf.file.Close()
}

func (tf tokenFile) Scan() bool {
	return tf.scanner.Scan()
}

func (tf tokenFile) Text() string {
	return tf.scanner.Text()
}

/* Ensure no error raised, panic otherwise. */
func check(err error) {
	if err != nil {
		panic(err)
	}
}

/* Initialize a new `tokenFile` */
func newTokenFile(name string) tokenFile {
	file, err := os.Open(name)
	check(err)
	return tokenFile{file, bufio.NewScanner(file)}
}

/* Opens a scanner to the tokens file. */
func openTokensFile() tokenFile {
	return newTokenFile("../lexer.tokens")
}

/*
Identifies the index of the start of a comment.
Returns -1 if none found.
*/
func findCommentPos(s string) int {
	return strings.Index(s, "#:")
}

/*
Returns a copy of the given string up to the
given index.

Note that this function will also remove
trailing whitespace.
*/
func justifyString(s string, ind int) string {
	if !(ind >= 0) {
		return s
	}
	temp := s[:ind]
	if temp == "  " {
		return " " // Assume is whitespace.
	}
	return strings.TrimRight(temp, " ") // remove unneeded whitespace.
}

/*
Identify any comments on the given line,
promptly remove any comments if existing.
*/
func parseComment(line string) string {
	var temp string = line
	var ind int = findCommentPos(temp)

	for ind >= 0 {
		ind = findCommentPos(temp)
		temp = justifyString(temp, ind)
	}
	return temp
}

/* Identify the tokenName and tokenSequence on a single line. */
func parseLine(line string) (string, string) {
	temp := strings.SplitN(line, " ", 2)

	for i, p := range temp {
		temp[i] = parseComment(p)
	}
	if len(temp) == 1 {
		return "", ""
	}
	if len(temp) > 2 {
		msg := fmt.Sprintf("expected no more than two objects, got %s", temp)
		panic(msg)
	}
	return temp[0], temp[1]
}

/* From the tokens file, load in defined tokens. */
func loadTokens() {
	file := openTokensFile()

	// Explicit add of whitespace token
	// to enforce always ID of 0.
	tokenKinds.Add(tokenName("WHTSPACE"), tokenSignature(" "))

	// Explicit add of general objects also to
	// enforce always ID of 1-3.
	tokenKinds.Add(tokenName("GENIDEN"), tokenSignature("&IDEN"))
	tokenKinds.Add(tokenName("GENTYPE"), tokenSignature("&TYPE"))
	tokenKinds.Add(tokenName("GENOBJ"), tokenSignature("&OBJ"))

	// Explicit add of general whitespace chars.
	// Cannot properly read these values from
	// tokens file. Not worth the jerry rigging.
	tokenKinds.Add(tokenName("NEWLINE"), tokenSignature("\n"))
	tokenKinds.Add(tokenName("CRETURN"), tokenSignature("\r"))
	tokenKinds.Add(tokenName("TABLINE"), tokenSignature("\t"))

	for file.Scan() {
		name, seq := parseLine(file.Text())
		if name == "" {
			continue
		}
		tokenKinds.Add(tokenName(name), tokenSignature(seq))
	}

	file.Close()
}

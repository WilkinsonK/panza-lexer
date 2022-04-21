package main

import (
	"fmt"
	"lexer/tokens"
)

func main() {
	tokens := tokens.TokenizeFile("./testfile.pz")
	for i := range tokens {
		token := tokens[i]
		fmt.Printf("line[%d]  \tpos[%d]  \tID[%d]   \t%s   \t'%s'\n", token.LineNo, token.Position, token.Kind.Id, token.Kind, token.Symbol)
	}
}

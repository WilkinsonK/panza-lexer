package lexer_test

import (
	"fmt"
	"testing"

	"github.com/WilkinsonK/panza-lexer"
)

func TestTokenizeFile(t *testing.T) {
	tokens := lexer.TokenizeFile("../testfile.pz")

	var to lexer.TokenObject
	for i := range tokens {
		to = tokens[i]
		fmt.Printf("line[%d]    \tpos[%d]    \tid[%d]    \t%s   \t'%s'\n", to.LineNo, to.Position, to.Kind.Id, to.Kind, to.Symbol)
	}
}

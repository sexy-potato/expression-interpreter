package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func patch(k string) (Token, error) {
	switch k {
		case "keyword": return Token{Number: 1,Type: Number}, nil
		default: {
			return Token{}, nil
		}
	}
}

func BenchmarkInterpret(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Interpret(`(6 / (4 - 1) >= 2) and "b5375d873f974d1eabe9784439d8fc73" matches "(?i)[\dA-F]{32}"`)
	}
}

func Test(t *testing.T) {

	It("Tokenize", func() {
		v := `6/(4-1)>=2 and "b5375d873f974d1eabe9784439d8fc73" matches "(?i)[\dA-F]{32}" and 1 in [1,2,3]`
		t := &tokenizer{
			Index: -1,
			Length: len(v),
			Input: v,
			Line: 1,
		}
		Expect(t.tokenize(abort)).To(Equal(
			[]Token{
				{ Type: Number, Number: float64(6) },
				{ Type: reserved, String: "/" },
				{ Type: reserved, String: "(" },
				{ Type: Number, Number: float64(4) },
				{ Type: reserved, String: "-" },
				{ Type: Number, Number: float64(1) },
				{ Type: reserved, String: ")" },
				{ Type: reserved, String: ">=" },
				{ Type: Number, Number: float64(2) },
				{ Type: reserved, String: "and" },
				{ Type: String, String: "b5375d873f974d1eabe9784439d8fc73" },
				{ Type: reserved, String: "matches" },
				{ Type: String, String: `(?i)[\dA-F]{32}` },
				{ Type: reserved, String: "and" },
				{ Type: Number, Number: float64(1) },
				{ Type: reserved, String: "in" },
				{ Type: List, List: []Token{
					{ Type: Number, Number: float64(1) },
					{ Type: Number, Number: float64(2) }, 
					{ Type: Number, Number: float64(3) },
					},
				},
			},
		))
	})

	It("have syntax error", func() {
		tests := []string {
			`[1`,
			`[1,2,]`,
			`[,1]`,
			`"`,
			`[`,
		}
		for i := range tests {
			_, v := Interpret(tests[i])
			Expect(v).To(HaveOccurred())
		}	
	})

	It("should work correctly", func() {
		Expect(Interpret(`(1)`)).To(Equal( float64(1) ))
		Expect(Interpret(`1`)).To(Equal(float64(1)))
		Expect(Interpret(`true`)).To(BeTrue())
		Expect(Interpret(`""`)).To(Equal(""))
		Expect(Interpret(`1>0`)).To(BeTrue())
		Expect(Interpret(`1+1`)).To(Equal(float64(2)))
		Expect(Interpret(`"b5375d873f974d1eabe9784439d8fc73" matches "(?i)[\dA-F]{32}"`)).To(BeTrue())
		Expect(Interpret(`1 in [1,2,3]`)).To(BeTrue())
		Expect(Interpret(`(6 / (4 - 1) >= 2) or "b5375d873f974d1eabe9784439d8fc73" matches "[\dA-Fa-f]{32}"`)).To(BeTrue())
		Expect(Interpret(`6/(4 - 1)`)).To(Equal(float64(2)))
		Expect(Interpret(`1<0`)).To(BeFalse())
		Expect(InterpretWith(`keyword == 1`, patch)).To(BeTrue())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "expression-interpreter")
}
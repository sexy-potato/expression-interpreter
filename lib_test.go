package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func contextOf(v string) tokenizer {
  stream := []rune(v)
  return tokenizer{
    Stream: stream,
    Index: -1,
    Length: len(stream),
    Line: 1,
  }
}

func TestByGinkgo(t *testing.T) {
  
  It("should remove line/block comment", func() {
    Expect(ready( []rune("double//a\n") )).To(Equal( contextOf("double") ))
    Expect(ready( []rune("// line\ndouble< -1 /* min */, 0.98>") )).To(Equal(	contextOf("double< -1 , 0.98>") ))
    Expect(ready( []rune("double//a") )).To(Equal( contextOf("double") ))
  })

  It("should report an error if block comment does not enclosed with */ syntax", func() {
    _, failure := ready([]rune("double/*"))
    Expect(failure).To(HaveOccurred())
  })

  It("should work correctly", func() {
    Expect(Interpret("(1)")).To(Equal( float64(1) ))
    Expect(Interpret("1")).To(Equal(float64(1)))
    Expect(Interpret("1>0")).To(BeTrue())
    Expect(Interpret("1 + 1")).To(Equal(float64(2)))
    Expect(Interpret("\"b5375d873f974d1eabe9784439d8fc73\" matches /^[\\dA-Fa-f]{32}$/")).To(BeTrue())
    Expect(Interpret("1 in [1,2,3]")).To(BeTrue())
    Expect(Interpret("(6/(4 - 1)>=2) or \"b5375d873f974d1eabe9784439d8fc73\" matches /^[\\dA-Fa-f]{32}$/")).To(BeTrue())
    Expect(Interpret("6/(4 - 1)")).To(Equal(float64(2)))
    Expect(Interpret("1<0")).To(BeFalse())
  })

  RegisterFailHandler(Fail)
  RunSpecs(t, "expression-interpreter")
}
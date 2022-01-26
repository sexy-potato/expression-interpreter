# Expression interpreter

Simple library used to evaluate expression (logical, arithmetic or comparision operation)

## Example

![GIF](https://github.com/sexy-potato/expression-interpreter/blob/main/example.gif)

## Patch keyword by callback

You can define a keyword in expression, and replace it by a callback when you calling to `InterpretWith(string, func(string)(Token,error))` function.

```go
v, e := InterpretWith(`money.amount == 1`, func(k string) (Token, error) {
  switch k {
    case "money.amount": return Token{Number: 1,Type: Number}, nil
    default: {
      return Token{}, nil
    }
  }
})
```
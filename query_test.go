//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh

import (
	"testing"
)

func TestQueryExprEval(t *testing.T) {
	labels := map[string]string{"POP": "LAX", "OS": "JUNOS"}

	exprTests := []struct {
		expr     string
		expected bool
	}{
		{"POP==LAX", true},
		{"POP!=LAX", false},
		{"POP==LAX && OS==JUNOS", true},
		{"POP==LAX && OS!=JUNOS", false},
		{"(POP==LAX || POP==BUR) && OS==JUNOS", true},
		{"OS==JUNOS && (POP==LAX || POP==BUR)", true},
		{"OS!=JUNOS && (POP==LAX || POP==BUR)", false},
		{"(OS==JUNOS) && (POP==LAX || POP==BUR)", true},
		{"((OS==JUNOS) && (POP==LAX || POP==BUR))", true},
	}

	for _, x := range exprTests {
		v, err := parseExpr(x.expr)
		if err != nil {
			t.Fatal(err)
		}

		ok, err := exprEval(v, labels)
		if err != nil {
			t.Fatal(err)
		}

		if ok != x.expected {
			t.Fatalf("expect %t, got %t", x.expected, ok)
		}
	}

	_, err := parseExpr("OS=JUNOS")
	if err == nil {
		t.Fatal("expect error but got nil")
	}

	v, err := parseExpr("OS")
	if err != nil {
		t.Fatal("expect error but got nil")
	}
	_, err = exprEval(v, labels)
	if err == nil {
		t.Fatal("expect error but got nil")
	}

	// not support operator
	ops := []string{"&", "+", "<=", "<"}
	for _, op := range ops {
		v, _ := parseExpr("OS == JUNOS " + op + " POP == LAX")
		_, err = exprEval(v, labels)
		if err == nil {
			t.Fatal("expect error but got nil")
		}
	}
}

func BenchmarkQueryExprEval(b *testing.B) {
	labels := map[string]string{"POP": "LAX", "OS": "JUNOS"}
	expr := "POP==LAX"

	for i := 0; i < b.N; i++ {

		v, err := parseExpr(expr)
		if err != nil {
			b.Fatal(err)
		}

		_, err = exprEval(v, labels)
		if err != nil {
			b.Fatal(err)
		}

	}

}

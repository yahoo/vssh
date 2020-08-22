//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"strconv"
	"sync"
	"time"
)

const (
	idLogic = iota
	idOp
	idName
	idValue
	idEvald

	opAnd = 34
	opOr  = 35
)

var errNotSupportOperator = errors.New("operator doesn't support")
var errQuery = errors.New("query error")

type query struct {
	cmd           string
	ctx           context.Context
	respChan      chan *Response
	respTimeout   time.Duration
	stmt          string
	compiledQuery *visitor
	limitReadOut  int64
	limitReadErr  int64
}

type visitor struct {
	idents []ident
}

type ident struct {
	value string
	vType int
}

func (q *query) errResp(id string, err error) {
	err = fmt.Errorf("client [%s]: %v", id, err)
	q.respChan <- &Response{id: id, err: err}
}

func (q *query) run(v *VSSH) {
	var wg sync.WaitGroup
	for client := range v.clients.enum() {

		client := client

		wg.Add(1)
		go func() {
			defer wg.Done()

			if len(q.stmt) > 0 && !client.labelMatch(q.compiledQuery) {
				return
			}

			if v.mode {
				client.connect()
			}

			if client.getErr() != nil {
				client.RLock()
				q.respChan <- &Response{id: client.addr, err: client.err}
				client.RUnlock()
				return
			}

			client.run(q)

			if v.mode {
				client.close()
			}
		}()
	}

	wg.Wait()
	close(q.respChan)
}

func (f *visitor) Visit(n ast.Node) ast.Visitor {
	switch d := n.(type) {
	case *ast.Ident:
		f.idents = append(f.idents, ident{d.Name, idName})
	case *ast.BasicLit:
		f.idents = append(f.idents, ident{d.Value, idValue})
	case *ast.BinaryExpr:
		if d.Op == opAnd || d.Op == opOr {
			f.idents = append(f.idents, ident{d.Op.String(), idLogic})
		} else {
			f.idents = append(f.idents, ident{d.Op.String(), idOp})
		}

	}

	return f
}

func parseExpr(expr string) (*visitor, error) {
	e, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, err
	}

	var v visitor
	ast.Walk(&v, e)

	return &v, nil
}

func exprEval(v *visitor, labels map[string]string) (bool, error) {
	results := []ident{}
	i := 0

	for {
		if len(v.idents[i:]) < 3 {
			break
		}

		if v.idents[i].vType == 0 {
			results = append(results, v.idents[i])
			i++
			continue
		}

		if relOpEval(v.idents[i:i+3], labels) {
			results = append(results, ident{"true", idEvald})
		} else {
			results = append(results, ident{"false", idEvald})
		}

		if i+3 < len(v.idents) {
			i = i + 3
		} else {
			break
		}
	}

	if len(results) < 1 {
		return false, errQuery
	}

	ok, err := binOpEval(&results)
	if err != nil {
		return ok, err
	}

	return ok, nil
}

func binOpEval(idents *[]ident) (bool, error) {
	i := 0
	for {
		if len((*idents)) == 1 {
			r, _ := strconv.ParseBool((*idents)[0].value)
			return r, nil
		}

		if (*idents)[i].vType == 0 && (*idents)[i+1].vType == 0 {
			i++
			continue
		}

		if len((*idents)) > 2 && (*idents)[i].vType == idLogic &&
			(*idents)[i+1].vType == idEvald && (*idents)[i+2].vType == idEvald {

			r1, _ := strconv.ParseBool((*idents)[i+1].value)
			r2, _ := strconv.ParseBool((*idents)[i+2].value)

			if (*idents)[i].value == "&&" {
				(*idents)[i].value = strconv.FormatBool(r1 && r2)
			} else if (*idents)[i+0].value == "||" {
				(*idents)[i].value = strconv.FormatBool(r1 || r2)
			} else {
				return false, errNotSupportOperator
			}

			(*idents)[i].vType = idEvald
			if len((*idents)) > 2 {
				(*idents) = append((*idents)[:i+1], (*idents)[i+3:]...)
				i = 0
				continue
			} else {
				tmp := (*idents)[0:1]
				*idents = tmp
			}
		}

		if len((*idents)) > 4 {
			i += 2
		} else {
			return false, errNotSupportOperator
		}
	}
}

func relOpEval(idents []ident, labels map[string]string) bool {
	if idents[0].value == "==" {
		if _, ok := labels[idents[1].value]; ok {
			return labels[idents[1].value] == idents[2].value
		}
	} else {
		if _, ok := labels[idents[1].value]; ok {
			return labels[idents[1].value] != idents[2].value
		}
	}

	return false
}

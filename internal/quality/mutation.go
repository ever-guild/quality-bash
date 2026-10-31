package quality

import (
	"crypto/sha256"
	"fmt"
	"mvdan.cc/sh/v3/syntax"
	"sort"
	"strings"
)

const MutationMethod = "bash-ast-v1"

type Mutation struct {
	ID       string `json:"id"`
	File     string `json:"file"`
	Operator string `json:"operator"`
	Offset   uint   `json:"offset"`
	Line     uint   `json:"line"`
	Column   uint   `json:"column"`
	Before   string `json:"before"`
	After    string `json:"after"`
	Result   string `json:"result"`
}

func GenerateMutations(file, source string) ([]Mutation, error) {
	tree, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(source), file)
	if err != nil {
		return nil, err
	}
	var mutations []Mutation
	add := func(pos syntax.Pos, operator, before, after string) {
		offset := pos.Offset()
		if int(offset)+len(before) > len(source) || source[offset:uint(int(offset)+len(before))] != before {
			return
		}
		id := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%s\x00%s", MutationMethod, file, offset, operator, before, after))))
		mutations = append(mutations, Mutation{ID: id, File: file, Operator: operator, Offset: offset, Line: pos.Line(), Column: pos.Col(), Before: before, After: after, Result: "pending"})
	}
	syntax.Walk(tree, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.BinaryTest:
			before := n.Op.String()
			after := map[string]string{"==": "!=", "!=": "==", "-eq": "-ne", "-ne": "-eq", "-lt": "-ge", "-ge": "-lt", "-gt": "-le", "-le": "-gt", "&&": "||", "||": "&&"}[before]
			if after != "" {
				add(n.OpPos, "condition", before, after)
			}
		case *syntax.BinaryArithm:
			before := n.Op.String()
			after := map[string]string{"+": "-", "-": "+", "*": "+", "<": ">=", ">=": "<", ">": "<=", "<=": ">", "==": "!=", "!=": "=="}[before]
			if after != "" {
				add(n.OpPos, "arithmetic", before, after)
			}
		case *syntax.CallExpr:
			if len(n.Args) == 2 && (n.Args[0].Lit() == "exit" || n.Args[0].Lit() == "return") {
				value := n.Args[1].Lit()
				if value == "0" {
					add(n.Args[1].Pos(), "return-code", "0", "1")
				} else if value == "1" {
					add(n.Args[1].Pos(), "return-code", "1", "0")
				}
			}
		}
		return true
	})
	sort.Slice(mutations, func(i, j int) bool { return mutations[i].Offset < mutations[j].Offset })
	return mutations, nil
}
func (m Mutation) Apply(source string) (string, error) {
	end := uint64(m.Offset) + uint64(len(m.Before))
	if end > uint64(len(source)) || source[m.Offset:end] != m.Before {
		return "", fmt.Errorf("mutation %s does not match source", m.ID)
	}
	return source[:m.Offset] + m.After + source[end:], nil
}

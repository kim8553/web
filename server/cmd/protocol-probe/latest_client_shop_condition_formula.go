package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// conditionFormulaOp mirrors the exact-current condition formula node kinds:
// op0 is either a leaf condition id or a parenthesized wrapper, op1 is AND,
// and op2 is OR.  The grammar deliberately gives '&' and '|' the same
// precedence and descends on the right-hand side, so expressions are
// right-associative rather than using conventional boolean precedence.
type conditionFormulaOp uint8

const (
	conditionFormulaLeafOrWrapper conditionFormulaOp = 0
	conditionFormulaAnd           conditionFormulaOp = 1
	conditionFormulaOr            conditionFormulaOp = 2
)

type conditionFormulaNode struct {
	Op          conditionFormulaOp
	ConditionID int32
	Child       *conditionFormulaNode
	Left        *conditionFormulaNode
	Right       *conditionFormulaNode
}

type conditionFormulaParser struct {
	input string
	pos   int
}

// parseConditionFormula implements the exact-current right-descending grammar.
// Examples fixed by the native parser evidence:
//
//	1&2|3 => 1 & (2 | 3)
//	1|2&3 => 1 | (2 & 3)
//	1&2&3 => 1 & (2 & 3)
//
// Parentheses create an op0 wrapper and therefore force grouping.
func parseConditionFormula(raw string) (*conditionFormulaNode, error) {
	parser := conditionFormulaParser{input: strings.TrimSpace(raw)}
	if parser.input == "" {
		return nil, fmt.Errorf("empty condition formula")
	}
	node, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}
	parser.skipSpace()
	if parser.pos != len(parser.input) {
		return nil, fmt.Errorf("unexpected token %q at byte %d", parser.input[parser.pos], parser.pos)
	}
	return node, nil
}

func (parser *conditionFormulaParser) parseExpression() (*conditionFormulaNode, error) {
	left, err := parser.parsePrimary()
	if err != nil {
		return nil, err
	}
	parser.skipSpace()
	if parser.pos >= len(parser.input) || (parser.input[parser.pos] != '&' && parser.input[parser.pos] != '|') {
		return left, nil
	}
	token := parser.input[parser.pos]
	parser.pos++
	right, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}
	op := conditionFormulaAnd
	if token == '|' {
		op = conditionFormulaOr
	}
	return &conditionFormulaNode{Op: op, Left: left, Right: right}, nil
}

func (parser *conditionFormulaParser) parsePrimary() (*conditionFormulaNode, error) {
	parser.skipSpace()
	if parser.pos >= len(parser.input) {
		return nil, fmt.Errorf("missing condition at byte %d", parser.pos)
	}
	if parser.input[parser.pos] == '(' {
		open := parser.pos
		parser.pos++
		child, err := parser.parseExpression()
		if err != nil {
			return nil, err
		}
		parser.skipSpace()
		if parser.pos >= len(parser.input) || parser.input[parser.pos] != ')' {
			return nil, fmt.Errorf("unclosed parenthesis at byte %d", open)
		}
		parser.pos++
		return &conditionFormulaNode{Op: conditionFormulaLeafOrWrapper, Child: child}, nil
	}

	start := parser.pos
	if parser.input[parser.pos] == '+' || parser.input[parser.pos] == '-' {
		parser.pos++
	}
	digits := parser.pos
	for parser.pos < len(parser.input) && parser.input[parser.pos] >= '0' && parser.input[parser.pos] <= '9' {
		parser.pos++
	}
	if parser.pos == digits {
		return nil, fmt.Errorf("expected int32 condition id at byte %d", start)
	}
	value, err := strconv.ParseInt(parser.input[start:parser.pos], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("condition id %q is not int32: %w", parser.input[start:parser.pos], err)
	}
	return &conditionFormulaNode{Op: conditionFormulaLeafOrWrapper, ConditionID: int32(value)}, nil
}

func (parser *conditionFormulaParser) skipSpace() {
	for parser.pos < len(parser.input) && unicode.IsSpace(rune(parser.input[parser.pos])) {
		parser.pos++
	}
}

// conditionFormulaLeafEvaluator returns both the condition result and whether
// that result is authoritative. Unsupported leaves must return supported=false;
// evaluateConditionFormula then fails closed rather than inventing a value.
type conditionFormulaLeafEvaluator func(conditionID int32) (satisfied bool, supported bool)

func evaluateConditionFormula(node *conditionFormulaNode, evaluateLeaf conditionFormulaLeafEvaluator) (satisfied bool, supported bool, err error) {
	if node == nil {
		return false, false, fmt.Errorf("nil condition formula node")
	}
	switch node.Op {
	case conditionFormulaLeafOrWrapper:
		if node.Child != nil {
			if node.Left != nil || node.Right != nil {
				return false, false, fmt.Errorf("op0 wrapper contains binary children")
			}
			return evaluateConditionFormula(node.Child, evaluateLeaf)
		}
		if node.Left != nil || node.Right != nil {
			return false, false, fmt.Errorf("op0 leaf contains binary children")
		}
		if evaluateLeaf == nil {
			return false, false, fmt.Errorf("nil condition leaf evaluator")
		}
		satisfied, supported := evaluateLeaf(node.ConditionID)
		if !supported {
			return false, false, nil
		}
		return satisfied, true, nil

	case conditionFormulaAnd, conditionFormulaOr:
		if node.Child != nil || node.Left == nil || node.Right == nil {
			return false, false, fmt.Errorf("binary condition formula node is malformed")
		}
		// Evaluate both sides even when the boolean result could short-circuit.
		// SAFE classification requires every leaf to be authoritative.
		leftValue, leftSupported, err := evaluateConditionFormula(node.Left, evaluateLeaf)
		if err != nil {
			return false, false, err
		}
		rightValue, rightSupported, err := evaluateConditionFormula(node.Right, evaluateLeaf)
		if err != nil {
			return false, false, err
		}
		if !leftSupported || !rightSupported {
			return false, false, nil
		}
		if node.Op == conditionFormulaAnd {
			return leftValue && rightValue, true, nil
		}
		return leftValue || rightValue, true, nil

	default:
		return false, false, fmt.Errorf("unsupported condition formula op %d", node.Op)
	}
}

func conditionFormulaLeafIDs(node *conditionFormulaNode) ([]int32, error) {
	if node == nil {
		return nil, fmt.Errorf("nil condition formula node")
	}
	result := make([]int32, 0, 4)
	var walk func(*conditionFormulaNode) error
	walk = func(current *conditionFormulaNode) error {
		if current == nil {
			return fmt.Errorf("nil child in condition formula")
		}
		switch current.Op {
		case conditionFormulaLeafOrWrapper:
			if current.Child != nil {
				return walk(current.Child)
			}
			result = append(result, current.ConditionID)
			return nil
		case conditionFormulaAnd, conditionFormulaOr:
			if current.Left == nil || current.Right == nil {
				return fmt.Errorf("binary condition formula node is malformed")
			}
			if err := walk(current.Left); err != nil {
				return err
			}
			return walk(current.Right)
		default:
			return fmt.Errorf("unsupported condition formula op %d", current.Op)
		}
	}
	if err := walk(node); err != nil {
		return nil, err
	}
	return result, nil
}

// conditionFormulaResolver returns the authored formula for a condition id.
// found=false means the id is a terminal leaf handled by the server evaluator.
type conditionFormulaResolver func(conditionID int32) (formula string, found bool)

// evaluateConditionRoot recursively expands formula ids and evaluates terminal
// leaves. It never converts an unsupported terminal leaf into an authoritative
// false: supported=false propagates to the root and the value is fail-closed.
func evaluateConditionRoot(rootID int32, resolve conditionFormulaResolver, evaluateLeaf conditionFormulaLeafEvaluator) (satisfied bool, supported bool, err error) {
	return evaluateConditionRootStack(rootID, resolve, evaluateLeaf, make(map[int32]bool))
}

func evaluateConditionRootStack(conditionID int32, resolve conditionFormulaResolver, evaluateLeaf conditionFormulaLeafEvaluator, stack map[int32]bool) (bool, bool, error) {
	if resolve == nil {
		return false, false, fmt.Errorf("nil condition formula resolver")
	}
	formula, found := resolve(conditionID)
	if !found {
		if evaluateLeaf == nil {
			return false, false, fmt.Errorf("nil condition leaf evaluator")
		}
		value, supported := evaluateLeaf(conditionID)
		if !supported {
			return false, false, nil
		}
		return value, true, nil
	}
	if stack[conditionID] {
		return false, false, fmt.Errorf("condition formula cycle at id %d", conditionID)
	}
	stack[conditionID] = true
	defer delete(stack, conditionID)

	node, err := parseConditionFormula(formula)
	if err != nil {
		return false, false, fmt.Errorf("condition formula %d: %w", conditionID, err)
	}
	return evaluateResolvedConditionFormula(node, resolve, evaluateLeaf, stack)
}

func evaluateResolvedConditionFormula(node *conditionFormulaNode, resolve conditionFormulaResolver, evaluateLeaf conditionFormulaLeafEvaluator, stack map[int32]bool) (bool, bool, error) {
	if node == nil {
		return false, false, fmt.Errorf("nil condition formula node")
	}
	switch node.Op {
	case conditionFormulaLeafOrWrapper:
		if node.Child != nil {
			return evaluateResolvedConditionFormula(node.Child, resolve, evaluateLeaf, stack)
		}
		return evaluateConditionRootStack(node.ConditionID, resolve, evaluateLeaf, stack)
	case conditionFormulaAnd, conditionFormulaOr:
		left, leftSupported, err := evaluateResolvedConditionFormula(node.Left, resolve, evaluateLeaf, stack)
		if err != nil {
			return false, false, err
		}
		right, rightSupported, err := evaluateResolvedConditionFormula(node.Right, resolve, evaluateLeaf, stack)
		if err != nil {
			return false, false, err
		}
		if !leftSupported || !rightSupported {
			return false, false, nil
		}
		if node.Op == conditionFormulaAnd {
			return left && right, true, nil
		}
		return left || right, true, nil
	default:
		return false, false, fmt.Errorf("unsupported condition formula op %d", node.Op)
	}
}

// terminalConditionLeaves expands nested formulas and returns terminal ids in
// authored traversal order. It is used by the capability audit and therefore
// reports cycles or malformed formula rows instead of silently dropping them.
func terminalConditionLeaves(rootID int32, resolve conditionFormulaResolver) ([]int32, error) {
	result := make([]int32, 0, 4)
	stack := make(map[int32]bool)
	var expand func(int32) error
	expand = func(conditionID int32) error {
		formula, found := resolve(conditionID)
		if !found {
			result = append(result, conditionID)
			return nil
		}
		if stack[conditionID] {
			return fmt.Errorf("condition formula cycle at id %d", conditionID)
		}
		stack[conditionID] = true
		defer delete(stack, conditionID)
		node, err := parseConditionFormula(formula)
		if err != nil {
			return fmt.Errorf("condition formula %d: %w", conditionID, err)
		}
		ids, err := conditionFormulaLeafIDs(node)
		if err != nil {
			return fmt.Errorf("condition formula %d: %w", conditionID, err)
		}
		for _, childID := range ids {
			if err := expand(childID); err != nil {
				return err
			}
		}
		return nil
	}
	if resolve == nil {
		return nil, fmt.Errorf("nil condition formula resolver")
	}
	if err := expand(rootID); err != nil {
		return nil, err
	}
	return result, nil
}

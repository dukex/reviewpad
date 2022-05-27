// Copyright 2022 Explore.dev Unipessoal Lda. All Rights Reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package aladino

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v42/github"
	"github.com/reviewpad/reviewpad/engine"
	"github.com/reviewpad/reviewpad/utils/fmtio"
	"github.com/shurcooL/githubv4"
)

type Interpreter struct {
	env *EvalEnv
}

func (i *Interpreter) ProcessGroup(groupName string, kind engine.GroupKind, typeOf engine.GroupType, expr, paramExpr, whereExpr string) error {
	exprAST, err := buildGroupAST(typeOf, expr, paramExpr, whereExpr)
	value, err := evalGroup(i.env, exprAST)

	i.env.RegisterMap[groupName] = value

	return err
}

func buildGroupAST(typeOf engine.GroupType, expr, paramExpr, whereExpr string) (Expr, error) {
	if typeOf == engine.GroupTypeFilter {
		whereExprAST, err := Parse(whereExpr)
		if err != nil {
			return nil, err
		}

		return buildFilter(paramExpr, whereExprAST)
	} else {
		return Parse(expr)
	}
}

func evalGroup(env *EvalEnv, expr Expr) (Value, error) {
	exprType, err := TypeInference(env, expr)
	if err != nil {
		return nil, err
	}

	if exprType.Kind() != ARRAY_TYPE && exprType.Kind() != ARRAY_OF_TYPE {
		return nil, fmt.Errorf("expression is not a valid group")
	}

	return Eval(env, expr)
}

func (i *Interpreter) EvalExpr(kind, expr string) (bool, error) {
	exprAST, err := Parse(expr)
	if err != nil {
		return false, err
	}

	exprType, err := TypeInference(i.env, exprAST)
	if err != nil {
		return false, err
	}

	if exprType.Kind() != "BoolType" {
		return false, fmt.Errorf("expression %v is not a condition", expr)
	}

	return EvalCondition(i.env, exprAST)
}

func execLog(val string) {
	log.Println(fmtio.Sprint("aladino", val))
}

func execLogf(format string, a ...interface{}) {
	log.Println(fmtio.Sprintf("aladino", format, a...))
}

func (i *Interpreter) ExecActions(program *[]string) error {
	execLog("executing actions:")

	for _, statRaw := range *program {
		statAST, err := Parse(statRaw)
		if err != nil {
			return err
		}

		execStatAST, err := TypeCheckExec(i.env, statAST)
		if err != nil {
			return err
		}

		err = ExecAction(i.env, execStatAST)
		if err != nil {
			return err
		}

		execLogf("\taction %v executed", statRaw)
	}

	execLog("execution done")

	return nil
}

func NewInterpreter(
	ctx context.Context,
	client *github.Client,
	clientGQL *githubv4.Client,
	pullRequest *github.PullRequest,
	builtIns *BuiltIns,
) (engine.Interpreter, error) {
	evalEnv, err := NewEvalEnv(ctx, client, clientGQL, pullRequest, builtIns)
	if err != nil {
		return nil, err
	}

	return &Interpreter{
		env: evalEnv,
	}, nil
}
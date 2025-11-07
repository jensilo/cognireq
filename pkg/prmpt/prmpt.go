package prmpt

import (
	"bytes"
	_ "embed"
	"fmt"
	"sync"
	"text/template"
)

//go:embed iga.go.prmpt
var igaPrompt string

type promptTpl struct {
	init   sync.Once
	tpl    *template.Template
	tplErr error
}

type PromptService struct {
	igaPrompt promptTpl
}

func NewPromptService() *PromptService {
	return &PromptService{}
}

type IgaPromptParams struct {
	UserInstruction string
}

func (ps *PromptService) IgaPrompt(params IgaPromptParams) (string, error) {
	ps.igaPrompt.init.Do(func() {
		ps.igaPrompt.tpl, ps.igaPrompt.tplErr = template.New("igaPrompt").Parse(igaPrompt)
	})
	if ps.igaPrompt.tplErr != nil {
		return "", fmt.Errorf("can not parse iga prompt: %w", ps.igaPrompt.tplErr)
	}

	buf := bytes.Buffer{}
	err := ps.igaPrompt.tpl.Execute(&buf, params)
	if err != nil {
		return "", fmt.Errorf("can not execute iga prompt template: %w, with params %+v", err, params)
	}

	if buf.Len() < 1 {
		return "", fmt.Errorf("iga prompt should not be empty! with params: %+v", params)
	}

	return buf.String(), nil
}

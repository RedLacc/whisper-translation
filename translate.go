package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/ollama/ollama/api"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var (
	timeMatcher  *regexp.Regexp
	thinkMatcher *regexp.Regexp
)

func init() {
	timeMatcher = regexp.MustCompile(`\[\d{2}:\d{2}:\d{2}\.\d{3}\s-->\s\d{2}:\d{2}:\d{2}\.\d{3}\]`)
	thinkMatcher = regexp.MustCompile(`(?s)<think>.*?</think>`)

}

type Translate struct {
	OllamaModel       string
	client            *api.Client
	Prompt            string
	contextSize       int
	batchSize         int
	Histories         []*Line
	enableTranslation bool
}

func NewTranslate(prompt, ollamaModel string, contextSize, batchSize int) *Translate {
	ollamaUrl, _ := url.Parse("http://localhost:11434")
	return &Translate{
		OllamaModel:       ollamaModel,
		client:            api.NewClient(ollamaUrl, http.DefaultClient),
		Prompt:            prompt,
		contextSize:       contextSize,
		batchSize:         batchSize,
		Histories:         make([]*Line, 0, contextSize),
		enableTranslation: false,
	}
}

func (w *Translate) process(scanner *bufio.Scanner) {
	lines := make([]*Line, 0, w.contextSize)
	for scanner.Scan() {
		/*
			[00:02:02.440 --> 00:02:05.440]   Lando Norris leads the British Grand Prix.
			[00:02:05.440 --> 00:02:06.440]   It's in there.
			[00:02:06.440 --> 00:02:07.440]   Let's go, boys.
		*/
		line := scanner.Text()
		if timeMatcher.MatchString(strings.TrimSpace(line)) && len(line) > 34 {
			lines = append(lines, &Line{
				Original:   line,
				Start:      line[1:13],
				End:        line[18:30],
				Text:       line[34:],
				Translated: "",
			})
			if len(lines) >= w.batchSize {
				w.translate(lines)
				lines = lines[:0]
			}
		}
	}
	if w.enableTranslation && len(lines) > 0 {
		w.translate(lines)
	}
}

func (w *Translate) translate(texts []*Line) error {
	previous := w.getContext()
	b := strings.Builder{}
	for _, text := range texts {
		b.WriteString(text.Text)
		b.WriteString("\n")
	}
	rawText := b.String()

	segment1 := `%s
请将以上内容逐行翻译为中文，严格保证翻译结果的行数与原文行数相等，保持与上下文的连贯性，不要添加任何解释或其他内容，只回复翻译的结果。为了保证翻译结果的准确性，我给你提供了以下信息：`
	if previous != "" {
		segment1 += `
上文以及翻译结果：
` + previous
	}
	if w.Prompt != "" {
		segment1 += `

这些内容的有关信息：
` + w.Prompt
	}
	prompt := fmt.Sprintf(segment1, rawText)
	req := &api.GenerateRequest{
		Model:  w.OllamaModel,
		Prompt: prompt,
		Stream: &[]bool{false}[0],
	}
	err := w.client.Generate(context.Background(), req, func(resp api.GenerateResponse) error {
		translated := strings.Split(strings.ReplaceAll(strings.TrimSpace(thinkMatcher.ReplaceAllString(resp.Response, "")), "\n\n", "\n"), "\n")
		if len(translated) != len(texts) {
			return errors.Errorf("ollama response line count doesn't match:\n%s", resp.Response)
		}
		for i, trans := range translated {
			trans = strings.TrimSpace(trans)
			texts[i].Translated = trans
		}
		w.Histories = append(w.Histories, texts...)
		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}
	for _, line := range texts {
		fmt.Println(line)
	}
	return nil
}

func (w *Translate) getContext() string {
	idx := 0
	if len(w.Histories) > w.contextSize {
		idx = len(w.Histories) - w.contextSize
	}
	b := &strings.Builder{}
	for _, text := range w.Histories[idx:] {
		b.WriteString(text.Text)
		b.WriteString(text.Translated)
		b.WriteString("\n")
	}
	return b.String()
}

func readTxt() {
	file, err := os.Open(`C:\Users\sgy16\GolandProjects\transper\2025年7月12日000113.txt`)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	t := NewTranslate("这是由 Sky Sports 解说的2024 F1 巴林大奖赛，内容为口语，所以你可以翻译得不那么正式", "qwen3:14b", 2, 8)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t.process(scanner)
	}

}

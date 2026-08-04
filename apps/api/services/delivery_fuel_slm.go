package services

// delivery_fuel_slm.go — SLM fallback for reading the pump price off a page.
//
// Scraping is brittle: when the markup changes the regex silently finds nothing
// and the fuel signal degrades to neutral without anyone noticing. The
// support-platform SLM (Qwen 2.5 1.5B, OpenAI-compatible) reads the number out
// of the page text instead, which survives a layout change.
//
// Strictly bounded: cron path only (never a checkout), one short call, and the
// answer is only a candidate — parseFuelPrice and the move check still decide
// whether it may move a price. The model cannot produce a fee, only a reading
// that numeric gates can reject.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/homechef/api/config"
)

const fuelSLMTimeout = 20 * time.Second

// fuelSLMPrompt asks for one bare number. Anything else fails the parse below,
// which is the intended outcome — we would rather lose a reading than guess.
const fuelSLMPrompt = `Read the petrol (gasoline) price per litre in Indian Rupees from the text below.
Reply with ONLY the number, e.g. 104.21
If the text does not clearly state a petrol price per litre, reply exactly: NONE

TEXT:
`

// fuelSLMPageBudget caps how much page text is sent. A 1.5B model on CPU has a
// small context and the price sits near the top of these pages.
const fuelSLMPageBudget = 6000

type slmChatRequest struct {
	Model       string           `json:"model"`
	Messages    []slmChatMessage `json:"messages"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens"`
}

type slmChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type slmChatResponse struct {
	Choices []struct {
		Message slmChatMessage `json:"message"`
	} `json:"choices"`
}

// slmNumberPattern pulls the first bare decimal out of the reply, tolerating a
// stray currency symbol or trailing prose from a small model.
var slmNumberPattern = regexp.MustCompile(`([0-9]{2,3}(?:\.[0-9]{1,2})?)`)

// extractFuelPriceViaSLM asks the SLM for the price. Returns (0,false) on any
// failure — unreachable service, timeout, refusal, or an unparseable reply — so
// the caller simply keeps the previous price.
func extractFuelPriceViaSLM(ctx context.Context, body string) (float64, bool) {
	endpoint := config.AppConfig.SLMInferenceURL
	if endpoint == "" {
		return 0, false
	}

	text := stripHTMLTags(body)
	if len(text) > fuelSLMPageBudget {
		text = text[:fuelSLMPageBudget]
	}

	payload, err := json.Marshal(slmChatRequest{
		Model:       config.AppConfig.SLMInferenceModel,
		Messages:    []slmChatMessage{{Role: "user", Content: fuelSLMPrompt + text}},
		Temperature: 0, // deterministic: this is an extraction, not a generation
		MaxTokens:   16,
	})
	if err != nil {
		return 0, false
	}

	ctx, cancel := context.WithTimeout(ctx, fuelSLMTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return 0, false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: fuelSLMTimeout}).Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}

	var out slmChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out.Choices) == 0 {
		return 0, false
	}
	return parseSLMFuelReply(out.Choices[0].Message.Content)
}

// parseSLMFuelReply turns the model's reply into a validated price. Pure, so the
// gate the model must clear is directly testable.
func parseSLMFuelReply(reply string) (float64, bool) {
	reply = strings.TrimSpace(reply)
	if reply == "" || strings.EqualFold(reply, "NONE") {
		return 0, false
	}
	m := slmNumberPattern.FindStringSubmatch(reply)
	if m == nil {
		return 0, false
	}
	price, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	// The same band every automatically-obtained reading must clear.
	return parseFuelPrice(strconv.FormatFloat(price, 'f', 2, 64))
}

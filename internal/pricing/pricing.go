package pricing

type MoneyPerMillionTokens struct {
	Input       float64 `json:"input"`
	CachedInput float64 `json:"cached_input"`
	Output      float64 `json:"output"`
}

type ModelPricing struct {
	Model        string                `json:"model"`
	Image        MoneyPerMillionTokens `json:"image"`
	Text         MoneyPerMillionTokens `json:"text"`
	Currency     string                `json:"currency"`
	Unit         string                `json:"unit"`
	SourceURL    string                `json:"source_url"`
	LastVerified string                `json:"last_verified"`
	Notes        []string              `json:"notes"`
}

type Table struct {
	Models []ModelPricing `json:"models"`
}

func Current() Table {
	return Table{
		Models: []ModelPricing{
			{
				Model: "gpt-image-2",
				Image: MoneyPerMillionTokens{
					Input:       8,
					CachedInput: 2,
					Output:      30,
				},
				Text: MoneyPerMillionTokens{
					Input:       5,
					CachedInput: 1.25,
				},
				Currency:     "USD",
				Unit:         "1M tokens",
				SourceURL:    "https://openai.com/api/pricing/",
				LastVerified: "2026-06-14",
				Notes: []string{
					"OpenAI bills images as tokens; the final bill depends on token accounting for the generated image.",
					"Imagecut sends one grid image request and then cuts the returned image locally.",
				},
			},
		},
	}
}

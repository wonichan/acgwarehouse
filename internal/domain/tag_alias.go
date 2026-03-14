package domain

type TagAlias struct {
	ID              int64  `json:"id"`
	TagID           int64  `json:"tag_id"`
	Label           string `json:"label"`
	NormalizedLabel string `json:"normalized_label"`
	Locale          string `json:"locale"`
	AliasType       string `json:"alias_type"`
	IsPreferred     bool   `json:"is_preferred"`
}

package discord

type embed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"` // It comes from a hex-encoded number
}

type allowedMentions struct {
	Roles []uint64 `json:"roles"`
}

type postData struct {
	Content         string           `json:"content"`
	AllowedMentions *allowedMentions `json:"allowed_mentions"`
	Embeds          []*embed         `json:"embeds"`
}

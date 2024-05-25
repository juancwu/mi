package form

type BentoForm struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	PublicKey string `json:"public_key"`
}

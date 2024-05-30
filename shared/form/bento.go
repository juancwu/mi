package form

type BentoForm struct {
	Name      string   `json:"name"`
	PublicKey string   `json:"public_key"`
	KeyVals   []string `json:"keyvals"`
}

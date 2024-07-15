package cmd

// orderBentoResponseBody represents the response body when successfully ordered bento.
type orderBentoResponseBody struct {
	Message     string       `json:"message"`
	Ingridients []ingridient `json:"ingridients"`
}

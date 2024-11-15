package cmd

// orderBentoResponseBody represents the response body when successfully ordered bento.
type orderBentoResponseBody struct {
	Message     string       `json:"message"`
	Ingridients []ingridient `json:"ingridients"`
}

type challengerType struct {
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

type revokeEditRequest struct {
	BentoId                string   `json:"bento_id"`
	Email                  string   `json:"email"`
	Challenge              string   `json:"challenge"`
	Signature              string   `json:"signature"`
	ToBeRevokedPermissions []string `json:"to_be_revoked_permissions,omitempty"`
}

type reseasonIngridientRequest struct {
	BentoId    string         `json:"bento_id"`
	Challenger challengerType `json:"challenger"`
	Name       string         `json:"name"`
	Value      string         `json:"value"`
}

package cmd

// orderBentoResponseBody represents the response body when successfully ordered bento.
type orderBentoResponseBody struct {
	Message     string       `json:"message"`
	Ingridients []ingridient `json:"ingridients"`
}

type revokeEditRequest struct {
	BentoId                string   `json:"bento_id"`
	Email                  string   `json:"email"`
	Challenge              string   `json:"challenge"`
	Signature              string   `json:"signature"`
	ToBeRevokedPermissions []string `json:"to_be_revoked_permissions,omitempty"`
}

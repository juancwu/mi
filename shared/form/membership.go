package form

type MembershipForm struct {
	Email     string `json:"email" validate:"required,email" errormgs:"Invalid email address"`
	Password  string `json:"password" validate:"required,min=12" errormgs:"Minimum password length is 12"`
	FirstName string `json:"first_name" validate:"required" errormgs:"First name is required"`
	LastName  string `json:"last_name" validate:"required" errormgs:"Last name is required"`
}

package auth

type UserDetails struct {
	Id          string `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"displayName"`
}

type UserTokens struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	Scopes       []string `json:"scopes"`
}

type AuthState struct {
	LoggedIn bool         `json:"loggedIn"`
	User     *UserDetails `json:"user,omitempty"`
	Tokens   *UserTokens  `json:"tokens,omitempty"`
	Error    string       `json:"error,omitempty"`
}

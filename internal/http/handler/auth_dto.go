package handler

type credentialsRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

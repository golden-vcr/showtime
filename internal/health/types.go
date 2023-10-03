package health

type Status struct {
	IsReady bool   `json:"isReady"`
	Message string `json:"message"`
}

package services

type AppError struct {
	Status  int
	Code    string
	Message string
}

func NewAppError(status int, code string, message string) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

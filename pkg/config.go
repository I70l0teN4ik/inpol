package pkg

type Config struct {
	Host             string   `json:"host"`
	Queue            string   `json:"queue"`
	Case             string   `json:"case"`
	Name             string   `json:"name"`
	LastName         string   `json:"lastName"`
	DateOfBirth      string   `json:"dateOfBirth"`
	MFA              string   `json:"MFA"`
	Email            string   `json:"email"`
	UserID           string   `json:"userId"`
	InpolSecret      string   `json:"inpolSecret"`
	TelegramBotToken string   `json:"telegramBotToken"`
	TelegramChatIDs  []string `json:"telegramChatIDs"`
}

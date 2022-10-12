package tokens

import (
	"encoding/base64"

	"github.com/google/uuid"
)

const USER_ID_LENGTH = 36

func CreateToken(userUuid string) (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	key := userUuid + "-" + uuid.String()
	token := base64.StdEncoding.EncodeToString([]byte(key))
	return token, nil
}

func ExtractUserUuid(token string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	decoded := string(data)
	return decoded[:USER_ID_LENGTH], nil
}

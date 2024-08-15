package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"csleiyang.com/p_server/dbutil"
)

const (
	BIZ_PROMPT       = "biz_prompt"
	BIZ_PROMPT_AUDIO = "biz_prompt_audio"
	BIZ_PROMPT_IMAGE = "biz_prompt_image"
)

type Prompt struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	Prompt  string     `json:"prompt"`
	Created time.Time  `json:"created"`
	Updated *time.Time `json:"updated"`
}

func GetPrompt(db *dbutil.MySQLDB, promptName string) (Prompt, error) {
	query := "SELECT idprompt,name, prompt FROM prompt WHERE name = ?"
	row := db.QueryRow(query, promptName)

	var prompt Prompt
	err := row.Scan(&prompt.ID, &prompt.Name, &prompt.Prompt)
	if err != nil {
		return prompt, err
	}
	return prompt, nil
}

// replacePlaceholders 替换字符串中的占位符${{key}}为jsonParams中的值
func ReplacePlaceholders(template string, jsonParams map[string]interface{}, userInput string) string {
	for key, value := range jsonParams {
		placeholder := fmt.Sprintf("${{%s}}", key)
		template = strings.ReplaceAll(template, placeholder, fmt.Sprintf("%v", value))
	}
	if len(userInput) > 0 {
		template = strings.ReplaceAll(template, "${{userInput}}", userInput)
	}
	return template
}

func GetPromptFromDbByName(db *dbutil.MySQLDB, JsonParams map[string]interface{}, promptKey string) (string, error) {
	if bizPrompt, exist := JsonParams[promptKey]; exist {
		bizPromptStr, ok := bizPrompt.(string)
		if !ok {
			return "", errors.New("Error: BIZ_PROMPT is not a string")
		}
		promtpDb, err := GetPrompt(db, bizPromptStr)
		if err != nil {
			return "", err
		}
		return ReplacePlaceholders(promtpDb.Prompt, JsonParams, ""), nil
	} else {
		return "", fmt.Errorf("%s not exist.", BIZ_PROMPT)
	}
}

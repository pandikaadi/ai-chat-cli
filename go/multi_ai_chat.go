package main

import (
	"time"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"github.com/chzyer/readline"
	"database/sql"
	_ "github.com/lib/pq"
)

var (
	claudeAPIKey       = os.Getenv("MULTI_AI_CLAUDE_API_KEY")
	claudeAPIEndpoint  = os.Getenv("MULTI_AI_CLAUDE_EP")
	geminiAPIEndpoint  = os.Getenv("MULTI_AI_GEMINI_CREDS")
)
const (
	green              = "\033[92m"
	blue               = "\033[94m"
	reset              = "\033[0m"
	blackText          = "\033[30m"
	whiteBG            = "\033[47m"
	evalPrompt	   = "Based on the previous conversation, generate a brief, descriptive title (5 words or less) that captures the main topic or theme (reply by blank ' ' if you are unable to provide yet; do not reply ' ' if you can give the title):"

)

var models = []string{"claude-3-haiku-20240307", "claude-3-5-sonnet-20240620", "gemini-1.5-flash"}

type Topic struct {
	ID int64
	Title string
	CreatedAt time.Time
}
type Model struct {
	ID int64
	Name string
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GeminiMessage struct {
	Role  string         `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

func main() {
	for {
		convo := []string{}
		connStr := "user=postgres dbname=mul_llm password=postgres host=localhost port=5432 sslmode=disable"
		db, err := sql.Open("postgres", connStr)
		var topicID int64
		var modelID int64
		topicTitle := "-"
		
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
		defer db.Close()
		err = db.Ping()
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Println("What are we going to do today? \n 1. Continue past conversations \n 2. Create new conversation")

		for {
			rl, err := readline.New("You: ")
			if err != nil {
				panic(err)
			}
			defer rl.Close()

			userInput, err := rl.Readline()
			if err != nil { // io.EOF, readline.ErrInterrupt
				break
			}
			userInput = strings.TrimSpace(userInput)
			if strings.ToLower(userInput) == "exit" || strings.ToLower(userInput) == "quit" {
				fmt.Println("Exiting the chat.")
				os.Exit(0)
			} else if strings.ToLower(userInput) == "__main__" {
				break
			}
			userInput = strings.TrimSpace(userInput)
			if userInput == "1" {
				convo, topicID = showList(db)
				break
			} else if userInput == "2" {

				query := `INSERT INTO topics (title, model_id) VALUES ($1, $2) RETURNING id`
				err = db.QueryRow(query, "-", "3").Scan(&topicID)
				
				if err != nil {
					fmt.Println(err)
				}
				break

			} else if strings.ToLower(userInput) == "exit" || strings.ToLower(userInput) == "quit" {
				fmt.Println("Exiting the chat.")
				os.Exit(0)
			}
			fmt.Println("invalid input")
		}	
		if topicID == 0 {
			continue
		}

		modelID, provider, data, headers := chooseModel(db)

		if provider == "" {
			continue
		}

		rl, err := readline.New("You: ")
		if err != nil {
			panic(err)
		}
		defer rl.Close()

		for {
			userInput, err := rl.Readline()
			if err != nil { // io.EOF, readline.ErrInterrupt
				break
			}
			userInput = strings.TrimSpace(userInput)
			if strings.ToLower(userInput) == "exit" || strings.ToLower(userInput) == "quit" {
				fmt.Println("Exiting the chat.")
				os.Exit(0)
			} else if strings.ToLower(userInput) == "__main__" {
				break
			}
			userInput = strings.TrimSpace(userInput)

			if strings.ToLower(userInput) == "exit" || strings.ToLower(userInput) == "quit" {
				fmt.Println("Exiting the chat.")
				os.Exit(0)
			} else if strings.ToLower(userInput) == "__main__" {
				break
			} else if strings.ToLower(userInput) == "change_model" {
				modelID, provider, data, headers = chooseModel(db)
				continue
			} else if strings.ToLower(userInput) == "clear_convo" {
				convo = []string{}
				continue
			}

			convo = append(convo, userInput)
			var reply string
			var err2 error

			if provider == "claude" {
				reply, err2 = sendClaudeRequest(convo, data, headers)
			} else if provider == "gemini" {
				reply, err2 = sendGeminiRequest(convo, data, headers)
			}

			if err2 == nil {
				fmt.Printf("%s%s: %s%s\n", green, provider, reply, reset)
				convo = append(convo, reply)

				if topicTitle == "-" {
					evalConvo := append(convo, evalPrompt)
					var evalReply string
					var errEval error
					if provider == "claude" {
						evalReply, errEval = sendClaudeRequest(evalConvo, data, headers)
					} else {
						evalReply, errEval = sendGeminiRequest(evalConvo, data, headers)
					}
					if errEval != nil {
						fmt.Errorf("failed to update record: %w", errEval)
					}

					if evalReply == "-" {
						continue
					}

					query := `UPDATE topics SET title = $1 WHERE id = $2`

					// Execute the query
					_, err := db.Exec(query, evalReply, topicID)
					if err != nil {
						fmt.Errorf("failed to update record: %w", err)
					}

					if (len(evalReply) > 1 ) {
						topicTitle = evalReply
					}
				}
				query := `INSERT INTO chats (content, topic_id, model_id) VALUES ($1, $2, $3)`
				
				// Start a new transaction
				tx, err := db.Begin()
				if err != nil {
				    fmt.Println("Failed to start transaction:", err)
				    continue
				}
				
				// Prepare the statement
				stmt, err := tx.Prepare(query)
				if err != nil {
				    tx.Rollback()
				    fmt.Println("Failed to prepare statement:", err)
				    continue
				}
				defer stmt.Close()
				
				// Execute the statement for user input
				_, err = stmt.Exec(userInput, topicID, modelID)
				if err != nil {
				    tx.Rollback()
				    fmt.Println("Failed to insert user input:", err)
				    continue
				}
				
				// Execute the statement for AI reply
				_, err = stmt.Exec(reply, topicID, modelID)
				if err != nil {
				    tx.Rollback()
				    fmt.Println("Failed to insert AI reply:", err)
				    continue
				}
				
				// Commit the transaction
				err = tx.Commit()
				if err != nil {
				    fmt.Println("Failed to commit transaction:", err)
				    continue
				}

			} else {
				fmt.Printf("Error: %s\n", err2)
			}
		}
	}
}

func chooseModel(db *sql.DB) (int64, string, map[string]interface{}, map[string]string) {
	var model Model
	var models []Model
	var provider string
	var data map[string]interface{}
	var headers map[string]string
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT m.id, m.name from models m")

	if err != nil {
	    fmt.Println(err)
	}

	defer rows.Close()

	fmt.Printf("%-5s | %-30s | \n", "Index", "Model Name")
	counter := 1
	for rows.Next() {
		fmt.Println(strings.Repeat("-", 40))
		var m Model
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			fmt.Println("Error scanning row:", err)
			os.Exit(0)
		}
		fmt.Printf("%-5d | %-30s |\n", counter, m.Name)
		models = append(models, m)
		counter++
	}

	if err := rows.Err(); err != nil {
		fmt.Println(err)
	}

	for model.Name == "" {
		rl, err := readline.New("Choose model: ")
		if err != nil {
			panic(err)
		}
		defer rl.Close()
		modelInput, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			os.Exit(0)
		}
		modelInput = strings.TrimSpace(modelInput)
		if strings.ToLower(modelInput) == "exit" || strings.ToLower(modelInput) == "quit" {
			fmt.Println("Exiting the chat.")
			os.Exit(0)
		} else if strings.ToLower(modelInput) == "__main__" {
			return 0, "", nil, nil
		}

		if strings.ToLower(modelInput) == "exit" || strings.ToLower(modelInput) == "quit" {
			fmt.Println("Exiting the chat.")
			os.Exit(0)
		}

		if intModelInput, err := strconv.Atoi(modelInput); err == nil {
			if intModelInput > 0 && intModelInput <= len(models) {
				model = models[intModelInput - 1]
				if strings.Contains(model.Name, "claude") {
					provider = "claude"
					data = map[string]interface{}{
						"model":      model.Name,
						"max_tokens": 300,
					}
					headers = map[string]string{
						"x-api-key":          claudeAPIKey,
						"anthropic-version":  "2023-06-01",
						"Content-Type":       "application/json",
					}
				} else {
					provider = "gemini"
					headers = map[string]string{
						"Content-Type": "application/json",
					}
					data = map[string]interface{}{
						"contents": nil,
					}
				}
			} else {
				fmt.Println("Invalid model")
			}
		} else {
			fmt.Println("Invalid model")
		}
	}

	fmt.Printf("#### provider: %s | model: %s\n", provider, model.Name)
	return model.ID, provider, data, headers
}

func showList(db *sql.DB) ([]string, int64) {
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT t.id, t.title, created_at from topics t join models m on m.id = t.model_id ORDER BY t.id")
	ids := []int64{}
	var convo = []string{}

	if err != nil {
	    fmt.Println(err)
	}

	defer rows.Close()

	fmt.Printf("%-5s | %-50s | %-40s\n", "ID", "Title", "Model Name", "Created At")
	for rows.Next() {
		fmt.Println(strings.Repeat("-", 125))
		var t Topic
		if err := rows.Scan(&t.ID, &t.Title, &t.CreatedAt); err != nil {
			fmt.Println("Error scanning row:", err)
			os.Exit(0)
		}
		fmt.Printf("%-5d | %-50s | %-40v\n", t.ID, t.Title, t.CreatedAt)
		ids = append(ids, t.ID)
	}

	if err := rows.Err(); err != nil {
		fmt.Println(err)
	}

	for {
		rl, err := readline.New("Pick topic: ")
		topicInput, err := rl.Readline()
		defer rl.Close()
		if err != nil { // io.EOF, readline.ErrInterrupt
			os.Exit(0)
		}
		topicInput = strings.TrimSpace(topicInput)
		if strings.ToLower(topicInput) == "exit" || strings.ToLower(topicInput) == "quit" {
			fmt.Println("Exiting the chat.")
			os.Exit(0)
		} else if strings.ToLower(topicInput) == "__main__" {
			return []string{}, 0
		}

		if strings.ToLower(topicInput) == "exit" || strings.ToLower(topicInput) == "quit" {
			fmt.Println("Exiting the chat.")
			os.Exit(0)
		}

		if intTopicInput, err := strconv.ParseInt(topicInput, 10, 64); err == nil && contains(ids, intTopicInput)  {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			rows, err := db.QueryContext(ctx, "SELECT c.content, m.name from chats c join topics t on t.id = c.topic_id join models m on m.id = c.model_id where c.topic_id = $1 ;", intTopicInput)
			if err != nil {
				fmt.Println(err)
			}
			defer rows.Close()
			counter := 1
			for rows.Next() {
				var content string
				var modelName string
				if err := rows.Scan(&content, &modelName); err != nil {
				    fmt.Println("Error scanning row:", err)
				    os.Exit(0)
				}

				if counter % 2 == 0 {
					fmt.Printf("%s%s: %s%s\n", green, modelName, content, reset)
				} else {
					fmt.Printf("you: %s\n", content)
				}
				convo = append(convo, content)
				counter++
			}
			return convo, intTopicInput
		}
		fmt.Println("Invalid pick")
	}
}

func contains(ids []int64, target int64) bool {
    for _, id := range ids {
        if id == target {
            return true
        }
    }
    return false
}

func sendClaudeRequest(convo []string, data map[string]interface{}, headers map[string]string) (string, error) {
	formattedConvo := []Message{}
	for i, conv := range convo {
		role := "assistant"
		if i%2 == 0 {
			role = "user"
		}
		formattedConvo = append(formattedConvo, Message{Role: role, Content: conv})
	}
	data["messages"] = formattedConvo

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", claudeAPIEndpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", fmt.Errorf("API request failed with status code: %d", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result["content"].([]interface{})[0].(map[string]interface{})["text"].(string)), nil
}

func sendGeminiRequest(convo []string, data map[string]interface{}, headers map[string]string) (string, error) {
	formattedConvo := []GeminiMessage{}
	for i, conv := range convo {
		role := "model"
		if i%2 == 0 {
			role = "user"
		}
		formattedConvo = append(formattedConvo, GeminiMessage{Role: role, Parts: []GeminiPart{{Text: conv}}})
	}
	data["contents"] = formattedConvo

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", geminiAPIEndpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", fmt.Errorf("API request failed with status code: %d", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result["candidates"].([]interface{})[0].(map[string]interface{})["content"].(map[string]interface{})["parts"].([]interface{})[0].(map[string]interface{})["text"].(string)), nil
}

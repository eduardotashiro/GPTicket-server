package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/genai"
)

// Estructura que llega del front
type AnalyzeRequest struct {
	Content string `json:"content"`
}

// Estructura enviada al frontend
type AnalyzeResponse struct {
	Intention  string   `json:"intention"`
	Summary    []string `json:"summary"`
	Suggestion string   `json:"suggestion"`
}

func main() {
	http.HandleFunc("/analyze", handleAnalyze)

	fmt.Println("Servidor iniciado na porta 8080")

	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		log.Fatal(err)
	}
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	// Permite solicitudes desde el navegador
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Maneja solicitudes preflight HTTP (CORS)
	if r.Method == http.MethodOptions {
		fmt.Println("Recebi preflight do navegador")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Decodifica el JSON recibido en el body
	var req AnalyzeRequest
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&req)

	if err != nil {
		http.Error(w, "body inválido", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content vazio", http.StatusBadRequest)
		return
	}

	result, err := callGemini(req.Content)
	if err != nil {
		log.Println("erro ao chamar Gemini:", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	encodeErr := json.NewEncoder(w).Encode(result)

	if encodeErr != nil {
		log.Println(err)
	}
}

func callGemini(ticketContent string) (AnalyzeResponse, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		25*time.Second,
	)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return AnalyzeResponse{}, err
	}

	prompt := fmt.Sprintf(`Você é um assistente de suporte da Tuna Pagamentos.
Analise o histórico do ticket abaixo e responda APENAS com JSON válido neste formato:
{
  "intention": "uma frase descrevendo o problema principal do cliente",
  "summary": ["ponto 1", "ponto 2", "ponto 3"],
  "suggestion": "sugestão de resposta ao cliente em português"
}
Não inclua explicações, markdown ou texto fora do JSON.

Histórico:
%s`, ticketContent)

	resp, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return AnalyzeResponse{}, err
	}

	// Convierte el JSON devuelto por Gemini a la estructura de respuesta
	var result AnalyzeResponse

	err = json.Unmarshal([]byte(resp.Text()), &result)

	if err != nil {
		return AnalyzeResponse{}, err
	}

	return result, nil
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/grafikarsa/backend/internal/config"
	"github.com/grafikarsa/backend/internal/dto"
)

type AIHandler struct {
	cfg *config.Config
}

func NewAIHandler(cfg *config.Config) *AIHandler {
	return &AIHandler{cfg: cfg}
}

// GenerateProjectIdeas generates project ideas using Google Gemini AI
func (h *AIHandler) GenerateProjectIdeas(c *fiber.Ctx) error {
	var req dto.GenerateProjectIdeasRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("AI Handler - BodyParser error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
		))
	}

	log.Printf("AI Handler - Request received: jurusan=%s, interests=%v, type=%s, difficulty=%s", 
		req.Jurusan, req.Interests, req.ProjectType, req.Difficulty)

	// Validate API key
	if h.cfg.AI.GeminiAPIKey == "" {
		log.Printf("AI Handler - API key not configured")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"AI_SERVICE_NOT_CONFIGURED",
			"AI service not configured",
		))
	}

	log.Printf("AI Handler - Initializing Gemini client...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize Gemini client
	client, err := genai.NewClient(ctx, option.WithAPIKey(h.cfg.AI.GeminiAPIKey))
	if err != nil {
		log.Printf("AI Handler - Failed to create client: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"AI_CLIENT_ERROR",
			"Failed to initialize AI client",
		))
	}
	defer client.Close()

	log.Printf("AI Handler - Client initialized, generating content...")

	// Use Gemini model (use gemini-pro for stable API)
	model := client.GenerativeModel("gemini-3.1-flash-lite-preview")
	
	// Configure model for JSON output
	model.ResponseMIMEType = "application/json"

	// Build prompt
	prompt := h.buildPrompt(req)

	// Generate content
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("AI Handler - Generation error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"AI_GENERATION_ERROR",
			"Failed to generate ideas",
		))
	}

	log.Printf("AI Handler - Content generated, parsing response...")

	// Parse response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("AI Handler - Empty response from AI")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"AI_EMPTY_RESPONSE",
			"AI returned empty response",
		))
	}

	// Extract text from response
	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			responseText += string(txt)
		}
	}

	log.Printf("AI Handler - Response text length: %d", len(responseText))

	// Parse JSON response
	var aiResponse dto.GenerateProjectIdeasResponse
	if err := json.Unmarshal([]byte(responseText), &aiResponse); err != nil {
		log.Printf("AI Handler - JSON parse error: %v, response: %s", err, responseText)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"AI_PARSE_ERROR",
			"Failed to parse AI response",
		))
	}

	log.Printf("AI Handler - Success! Generated %d ideas", len(aiResponse.Ideas))

	return c.JSON(dto.SuccessResponse(aiResponse, "Project ideas generated successfully"))
}

func (h *AIHandler) buildPrompt(req dto.GenerateProjectIdeasRequest) string {
	interestsStr := strings.Join(req.Interests, ", ")
	
	prompt := fmt.Sprintf(`You are an expert educational advisor for vocational high school students in Indonesia, specifically for SMKN 4 Malang.

Generate 5 creative and practical project ideas for a student with the following profile:
- Jurusan (Major): %s
- Interests: %s
- Project Type: %s
- Difficulty Level: %s

Requirements:
1. Each project should be realistic and achievable for a high school student
2. Projects should align with the student's major and interests
3. Include modern technologies and best practices
4. Provide clear learning goals
5. Estimate realistic completion time
6. Make projects portfolio-worthy

Return ONLY a valid JSON object with this exact structure (no markdown, no code blocks, just raw JSON):
{
  "ideas": [
    {
      "title": "Project title in Indonesian",
      "description": "Detailed description in Indonesian (2-3 sentences explaining what the project is about and its purpose)",
      "technologies": ["Tech1", "Tech2", "Tech3"],
      "difficulty": "beginner|intermediate|advanced",
      "estimated_time": "X weeks/months",
      "learning_goals": ["Goal 1", "Goal 2", "Goal 3"]
    }
  ]
}

Make sure all text fields are in Indonesian language and relevant to Indonesian vocational education context.`, 
		req.Jurusan, 
		interestsStr, 
		req.ProjectType, 
		req.Difficulty,
	)

	return prompt
}

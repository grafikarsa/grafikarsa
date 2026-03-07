package dto

// GenerateProjectIdeasRequest represents the request to generate project ideas
type GenerateProjectIdeasRequest struct {
	Jurusan     string   `json:"jurusan" validate:"required"`
	Interests   []string `json:"interests" validate:"required,min=1"`
	ProjectType string   `json:"project_type" validate:"required"`
	Difficulty  string   `json:"difficulty" validate:"required,oneof=beginner intermediate advanced"`
}

// ProjectIdea represents a single generated project idea
type ProjectIdea struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Technologies []string `json:"technologies"`
	Difficulty  string   `json:"difficulty"`
	EstimatedTime string `json:"estimated_time"`
	LearningGoals []string `json:"learning_goals"`
}

// GenerateProjectIdeasResponse represents the response with generated ideas
type GenerateProjectIdeasResponse struct {
	Ideas []ProjectIdea `json:"ideas"`
}

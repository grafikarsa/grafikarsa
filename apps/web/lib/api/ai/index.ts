import api from '../client';

export interface ProjectIdea {
  title: string;
  description: string;
  technologies: string[];
  difficulty: string;
  estimated_time: string;
  learning_goals: string[];
}

export interface GenerateProjectIdeasRequest {
  jurusan: string;
  interests: string[];
  project_type: string;
  difficulty: string;
}

export interface GenerateProjectIdeasResponse {
  ideas: ProjectIdea[];
}

// Backend wrapper response
interface BackendResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

export const aiApi = {
  generateProjectIdeas: async (data: GenerateProjectIdeasRequest) => {
    return api.post<BackendResponse<GenerateProjectIdeasResponse>>('/ai/generate-project-ideas', data);
  },
};

# AI Features Configuration

This document explains how to configure and enable/disable AI features in Grafikarsa.

## Overview

Grafikarsa includes an AI-powered Project Ideas Generator that uses Google Gemini AI to help students generate creative project ideas based on their major, interests, and skill level.

## Environment Variables

### Backend (Go)

Add to your `.env` file:

```env
# Google Gemini API Key (required for AI features)
GOOGLE_GEMINI_API_KEY=your_api_key_here
```

### Frontend (Next.js)

Add to your `.env` or `.env.local` file:

```env
# Enable/disable AI features in the UI
NEXT_PUBLIC_AI_FEATURES_ENABLED=true
```

## Getting a Gemini API Key

1. Go to [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Sign in with your Google account
3. Click "Create API Key"
4. Copy the generated API key
5. Add it to your `.env` file

## Enabling/Disabling AI Features

### Development

To enable AI features in development:

```bash
# In .env or .env.local
GOOGLE_GEMINI_API_KEY=your_api_key_here
NEXT_PUBLIC_AI_FEATURES_ENABLED=true
```

To disable AI features:

```bash
# Set to false or remove the variable
NEXT_PUBLIC_AI_FEATURES_ENABLED=false
```

### Production (Docker)

The AI feature configuration is automatically passed to Docker containers through environment variables.

**docker-compose.yml** (Development):
```yaml
backend:
  environment:
    - GOOGLE_GEMINI_API_KEY=${GOOGLE_GEMINI_API_KEY}

# No changes needed for web in dev mode
```

**docker-compose.prod.yml** (Production Simulation):
```yaml
backend:
  environment:
    - GOOGLE_GEMINI_API_KEY=${GOOGLE_GEMINI_API_KEY}

web:
  build:
    args:
      - NEXT_PUBLIC_AI_FEATURES_ENABLED=${NEXT_PUBLIC_AI_FEATURES_ENABLED}
  environment:
    - NEXT_PUBLIC_AI_FEATURES_ENABLED=${NEXT_PUBLIC_AI_FEATURES_ENABLED}
```

**docker-compose.deploy.yml** (Server Deployment):
```yaml
backend:
  environment:
    - GOOGLE_GEMINI_API_KEY=${GOOGLE_GEMINI_API_KEY}

web:
  environment:
    - NEXT_PUBLIC_AI_FEATURES_ENABLED=${NEXT_PUBLIC_AI_FEATURES_ENABLED}
```

### Server Deployment

When deploying to your VPS:

1. SSH into your server
2. Navigate to the project directory
3. Update the `.env` file:
   ```bash
   nano .env
   ```
4. Add or update the AI configuration:
   ```env
   GOOGLE_GEMINI_API_KEY=your_api_key_here
   NEXT_PUBLIC_AI_FEATURES_ENABLED=true
   ```
5. Rebuild and restart containers:
   ```bash
   docker compose -f docker-compose.deploy.yml down
   docker compose -f docker-compose.deploy.yml pull
   docker compose -f docker-compose.deploy.yml up -d
   ```

## UI Behavior

When `NEXT_PUBLIC_AI_FEATURES_ENABLED=true`:
- The "AI Ide Proyek" menu item appears in the student sidebar
- Users can access the AI Project Ideas Generator at `/ai-ideas`

When `NEXT_PUBLIC_AI_FEATURES_ENABLED=false` or not set:
- The "AI Ide Proyek" menu item is hidden
- The `/ai-ideas` page is still accessible via direct URL but won't be linked in the UI

## API Endpoint

The AI feature exposes one endpoint:

```
POST /api/v1/ai/generate-project-ideas
```

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "jurusan": "Rekayasa Perangkat Lunak",
  "interests": ["Web Development", "UI/UX Design"],
  "project_type": "web_app",
  "difficulty": "intermediate"
}
```

**Response**:
```json
{
  "success": true,
  "message": "Project ideas generated successfully",
  "data": {
    "ideas": [
      {
        "title": "Sistem Manajemen Perpustakaan Digital",
        "description": "Aplikasi web untuk mengelola koleksi buku...",
        "technologies": ["React", "Node.js", "PostgreSQL"],
        "difficulty": "intermediate",
        "estimated_time": "4-6 minggu",
        "learning_goals": [
          "Memahami CRUD operations",
          "Implementasi autentikasi",
          "Desain database relational"
        ]
      }
    ]
  }
}
```

## Cost Considerations

Google Gemini API has a free tier with generous limits:
- **Free tier**: 15 requests per minute, 1,500 requests per day
- **Pricing**: Check [Google AI Pricing](https://ai.google.dev/pricing) for current rates

For a school project with moderate usage, the free tier should be sufficient.

## Troubleshooting

### AI menu not showing
- Check that `NEXT_PUBLIC_AI_FEATURES_ENABLED=true` in your environment
- Restart the Next.js dev server or rebuild the Docker container
- Clear browser cache

### API errors
- Verify `GOOGLE_GEMINI_API_KEY` is set correctly
- Check API key is valid and not expired
- Ensure you haven't exceeded rate limits
- Check backend logs: `docker logs grafikarsa-backend`

### Empty or invalid responses
- The AI model might be temporarily unavailable
- Check your internet connection
- Verify the API key has proper permissions

## Security Notes

1. **Never commit API keys** to version control
2. The `.env` file is in `.gitignore` by default
3. Use different API keys for development and production
4. Rotate API keys periodically
5. Monitor API usage in Google Cloud Console

## Feature Limitations

- Requires active internet connection
- Subject to Google Gemini API rate limits
- Responses are in Indonesian language
- Generated ideas are AI-generated and should be reviewed by teachers
- Results are stored in browser localStorage (not in database)

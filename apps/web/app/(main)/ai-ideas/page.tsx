'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { publicApi } from '@/lib/api';
import { aiApi, type ProjectIdea, type GenerateProjectIdeasRequest } from '@/lib/api/ai';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Sparkles, X, Clock, Target, Lightbulb, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';

const PROJECT_TYPES = [
  { value: 'web_app', label: 'Aplikasi Web' },
  { value: 'mobile_app', label: 'Aplikasi Mobile' },
  { value: 'desktop_app', label: 'Aplikasi Desktop' },
  { value: 'design_project', label: 'Proyek Desain' },
  { value: 'animation', label: 'Animasi' },
  { value: 'video_editing', label: 'Video Editing' },
  { value: 'game', label: 'Game' },
  { value: 'iot', label: 'IoT Project' },
  { value: 'other', label: 'Lainnya' },
];

const DIFFICULTY_LEVELS = [
  { value: 'beginner', label: 'Pemula', color: 'bg-green-500/10 text-green-700 dark:text-green-400' },
  { value: 'intermediate', label: 'Menengah', color: 'bg-yellow-500/10 text-yellow-700 dark:text-yellow-400' },
  { value: 'advanced', label: 'Lanjutan', color: 'bg-red-500/10 text-red-700 dark:text-red-400' },
];

const INTEREST_SUGGESTIONS = [
  'Web Development',
  'Mobile Development',
  'UI/UX Design',
  'Graphic Design',
  '3D Modeling',
  'Animation',
  'Video Editing',
  'Photography',
  'Game Development',
  'Machine Learning',
  'IoT',
  'Networking',
  'Database',
  'Cloud Computing',
];

export default function AIIdeasPage() {
  const [formData, setFormData] = useState<GenerateProjectIdeasRequest>({
    jurusan: '',
    interests: [],
    project_type: '',
    difficulty: 'intermediate',
  });
  const [interestInput, setInterestInput] = useState('');
  const [savedIdeas, setSavedIdeas] = useState<ProjectIdea[]>([]);

  // Load saved ideas from localStorage on mount
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('ai_project_ideas');
      if (saved) {
        try {
          setSavedIdeas(JSON.parse(saved));
        } catch (e) {
          console.error('Failed to parse saved ideas:', e);
        }
      }
    }
  }, []);

  // Fetch jurusan list
  const { data: jurusanData } = useQuery({
    queryKey: ['jurusan'],
    queryFn: () => publicApi.getJurusan(),
  });

  const jurusanList = jurusanData?.data || [];

  // Generate ideas mutation
  const generateMutation = useMutation({
    mutationFn: (data: GenerateProjectIdeasRequest) => aiApi.generateProjectIdeas(data),
    onSuccess: (response) => {
      // Backend returns: { success: true, data: { ideas: [...] }, message: "..." }
      // Axios wraps it in response.data
      const newIdeas = response.data.data.ideas;
      setSavedIdeas(newIdeas);
      // Save to localStorage
      localStorage.setItem('ai_project_ideas', JSON.stringify(newIdeas));
      toast.success('Ide proyek berhasil dibuat!');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error?.message || 'Gagal membuat ide proyek');
    },
  });

  const handleAddInterest = () => {
    if (interestInput.trim() && !formData.interests.includes(interestInput.trim())) {
      setFormData({
        ...formData,
        interests: [...formData.interests, interestInput.trim()],
      });
      setInterestInput('');
    }
  };

  const handleRemoveInterest = (interest: string) => {
    setFormData({
      ...formData,
      interests: formData.interests.filter((i) => i !== interest),
    });
  };

  const handleAddSuggestion = (suggestion: string) => {
    if (!formData.interests.includes(suggestion)) {
      setFormData({
        ...formData,
        interests: [...formData.interests, suggestion],
      });
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.jurusan) {
      toast.error('Pilih jurusan terlebih dahulu');
      return;
    }
    if (formData.interests.length === 0) {
      toast.error('Tambahkan minimal 1 minat');
      return;
    }
    if (!formData.project_type) {
      toast.error('Pilih tipe proyek terlebih dahulu');
      return;
    }

    generateMutation.mutate(formData);
  };

  const handleClearSaved = () => {
    setSavedIdeas([]);
    localStorage.removeItem('ai_project_ideas');
    toast.success('Ide tersimpan berhasil dihapus');
  };

  const getDifficultyColor = (difficulty: string) => {
    return DIFFICULTY_LEVELS.find((d) => d.value === difficulty)?.color || '';
  };

  return (
    <div className="container mx-auto px-4 pb-24 pt-20 md:px-12 lg:px-16 md:pt-24">
      {/* Header */}
      <div className="mx-auto mb-8 max-w-3xl text-center">
        <div className="mb-4 inline-flex items-center gap-2 rounded-full border bg-muted/50 px-4 py-2 text-sm">
          <Sparkles className="h-4 w-4 text-primary" />
          <span className="font-medium">AI-Powered</span>
        </div>
        <h1 className="mb-3 text-3xl font-bold md:text-4xl">Generator Ide Proyek</h1>
        <p className="text-muted-foreground">
          Dapatkan inspirasi ide proyek portfolio yang sesuai dengan jurusan dan minatmu menggunakan AI
        </p>
      </div>

      <div className="mx-auto grid max-w-6xl gap-8 lg:grid-cols-2">
        {/* Form Section */}
        <div>
          <Card className="p-6">
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Jurusan */}
              <div className="space-y-2">
                <Label htmlFor="jurusan">Jurusan</Label>
                <Select
                  value={formData.jurusan}
                  onValueChange={(value) => setFormData({ ...formData, jurusan: value })}
                >
                  <SelectTrigger id="jurusan">
                    <SelectValue placeholder="Pilih jurusan" />
                  </SelectTrigger>
                  <SelectContent>
                    {jurusanList.map((j) => (
                      <SelectItem key={j.id} value={j.nama}>
                        {j.nama}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Interests */}
              <div className="space-y-2">
                <Label htmlFor="interests">Minat & Keahlian</Label>
                <div className="flex gap-2">
                  <Input
                    id="interests"
                    placeholder="Tambahkan minat..."
                    value={interestInput}
                    onChange={(e) => setInterestInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        handleAddInterest();
                      }
                    }}
                  />
                  <Button type="button" onClick={handleAddInterest} variant="outline">
                    Tambah
                  </Button>
                </div>
                
                {/* Selected Interests */}
                {formData.interests.length > 0 && (
                  <div className="flex flex-wrap gap-2 pt-2">
                    {formData.interests.map((interest) => (
                      <Badge key={interest} variant="secondary" className="gap-1 pr-1">
                        {interest}
                        <button
                          type="button"
                          onClick={() => handleRemoveInterest(interest)}
                          className="ml-1 rounded-sm hover:bg-muted"
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </Badge>
                    ))}
                  </div>
                )}

                {/* Suggestions */}
                <div className="space-y-2 pt-2">
                  <p className="text-xs text-muted-foreground">Saran:</p>
                  <div className="flex flex-wrap gap-2">
                    {INTEREST_SUGGESTIONS.filter(
                      (s) => !formData.interests.includes(s)
                    ).slice(0, 8).map((suggestion) => (
                      <Badge
                        key={suggestion}
                        variant="outline"
                        className="cursor-pointer hover:bg-muted"
                        onClick={() => handleAddSuggestion(suggestion)}
                      >
                        {suggestion}
                      </Badge>
                    ))}
                  </div>
                </div>
              </div>

              {/* Project Type */}
              <div className="space-y-2">
                <Label htmlFor="project_type">Tipe Proyek</Label>
                <Select
                  value={formData.project_type}
                  onValueChange={(value) => setFormData({ ...formData, project_type: value })}
                >
                  <SelectTrigger id="project_type">
                    <SelectValue placeholder="Pilih tipe proyek" />
                  </SelectTrigger>
                  <SelectContent>
                    {PROJECT_TYPES.map((type) => (
                      <SelectItem key={type.value} value={type.value}>
                        {type.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Difficulty */}
              <div className="space-y-2">
                <Label htmlFor="difficulty">Tingkat Kesulitan</Label>
                <Select
                  value={formData.difficulty}
                  onValueChange={(value) => setFormData({ ...formData, difficulty: value })}
                >
                  <SelectTrigger id="difficulty">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {DIFFICULTY_LEVELS.map((level) => (
                      <SelectItem key={level.value} value={level.value}>
                        {level.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Submit Button */}
              <Button
                type="submit"
                className="w-full"
                disabled={generateMutation.isPending}
              >
                {generateMutation.isPending ? (
                  <>
                    <Sparkles className="mr-2 h-4 w-4 animate-spin" />
                    Membuat Ide...
                  </>
                ) : (
                  <>
                    <Sparkles className="mr-2 h-4 w-4" />
                    Buat Ide Proyek
                  </>
                )}
              </Button>
            </form>
          </Card>
        </div>

        {/* Results Section */}
        <div>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-semibold">
              {savedIdeas?.length > 0 ? 'Ide Proyek' : 'Hasil akan muncul di sini'}
            </h2>
            {savedIdeas?.length > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClearSaved}
                className="gap-2 text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="h-4 w-4" />
                Hapus Semua
              </Button>
            )}
          </div>

          {generateMutation.isPending ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <Card key={i} className="p-6">
                  <Skeleton className="mb-3 h-6 w-3/4" />
                  <Skeleton className="mb-4 h-20 w-full" />
                  <div className="flex gap-2">
                    <Skeleton className="h-6 w-20" />
                    <Skeleton className="h-6 w-20" />
                    <Skeleton className="h-6 w-20" />
                  </div>
                </Card>
              ))}
            </div>
          ) : savedIdeas?.length > 0 ? (
            <div className="space-y-4">
              {savedIdeas?.map((idea, index) => (
                <Card key={index} className="p-6 transition-all hover:shadow-md">
                  <div className="mb-3 flex items-start justify-between gap-4">
                    <h3 className="text-lg font-semibold">{idea.title}</h3>
                    <Badge className={cn('shrink-0', getDifficultyColor(idea.difficulty))}>
                      {DIFFICULTY_LEVELS.find((d) => d.value === idea.difficulty)?.label}
                    </Badge>
                  </div>

                  <p className="mb-4 text-sm text-muted-foreground">{idea.description}</p>

                  {/* Technologies */}
                  <div className="mb-4">
                    <div className="mb-2 flex items-center gap-2 text-xs font-medium text-muted-foreground">
                      <Lightbulb className="h-3.5 w-3.5" />
                      Teknologi
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {idea.technologies.map((tech, i) => (
                        <Badge key={i} variant="outline" className="text-xs">
                          {tech}
                        </Badge>
                      ))}
                    </div>
                  </div>

                  {/* Estimated Time */}
                  <div className="mb-4 flex items-center gap-2 text-sm">
                    <Clock className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Estimasi:</span>
                    <span className="font-medium">{idea.estimated_time}</span>
                  </div>

                  {/* Learning Goals */}
                  <div>
                    <div className="mb-2 flex items-center gap-2 text-xs font-medium text-muted-foreground">
                      <Target className="h-3.5 w-3.5" />
                      Tujuan Pembelajaran
                    </div>
                    <ul className="space-y-1 text-sm">
                      {idea.learning_goals.map((goal, i) => (
                        <li key={i} className="flex gap-2">
                          <span className="text-muted-foreground">•</span>
                          <span>{goal}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                </Card>
              ))}
            </div>
          ) : (
            <Card className="flex min-h-[400px] items-center justify-center p-12">
              <div className="text-center">
                <Sparkles className="mx-auto mb-4 h-12 w-12 text-muted-foreground/50" />
                <p className="text-muted-foreground">
                  Isi form di sebelah kiri untuk membuat ide proyek
                </p>
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

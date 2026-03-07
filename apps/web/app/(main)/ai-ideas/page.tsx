'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { publicApi } from '@/lib/api';
import { aiApi, type ProjectIdea, type GenerateProjectIdeasRequest } from '@/lib/api/ai';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Card } from '@/components/ui/card';
import { Sparkles, ChevronLeft, ChevronRight, RotateCcw, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { WizardProgress } from '@/components/ai/wizard-progress';
import { InterestCombobox } from '@/components/ai/interest-combobox';
import { IdeaCarousel } from '@/components/ai/idea-carousel';
import { LoadingProgress } from '@/components/ai/loading-progress';
import { EmptyState } from '@/components/ai/empty-state';
import { IdeasHistory } from '@/components/ai/ideas-history';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

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
  { value: 'beginner', label: 'Pemula', description: 'Cocok untuk yang baru mulai' },
  { value: 'intermediate', label: 'Menengah', description: 'Sudah punya pengalaman dasar' },
  { value: 'advanced', label: 'Lanjutan', description: 'Untuk yang sudah mahir' },
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

const WIZARD_STEPS = [
  { label: 'Profil', description: 'Jurusan & Minat' },
  { label: 'Proyek', description: 'Tipe & Kesulitan' },
  { label: 'Generate', description: 'Buat Ide' },
];

const FORM_STORAGE_KEY = 'ai_ideas_form_draft';
const IDEAS_STORAGE_KEY = 'ai_project_ideas';
const HISTORY_STORAGE_KEY = 'ai_ideas_history';

interface GenerationHistory {
  id: string;
  timestamp: number;
  formData: GenerateProjectIdeasRequest;
  ideas: ProjectIdea[];
}

export default function AIIdeasPage() {
  const [currentStep, setCurrentStep] = useState(1);
  const [formData, setFormData] = useState<GenerateProjectIdeasRequest>({
    jurusan: '',
    interests: [],
    project_type: '',
    difficulty: 'intermediate',
  });
  const [savedIdeas, setSavedIdeas] = useState<ProjectIdea[]>([]);
  const [showClearDialog, setShowClearDialog] = useState(false);
  const [hasGenerated, setHasGenerated] = useState(false);
  const [showEmptyState, setShowEmptyState] = useState(true);
  const [generationHistory, setGenerationHistory] = useState<GenerationHistory[]>([]);

  // Load saved form and ideas from localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      // Load form draft
      const savedForm = localStorage.getItem(FORM_STORAGE_KEY);
      if (savedForm) {
        try {
          const parsed = JSON.parse(savedForm);
          setFormData(parsed);
        } catch (e) {
          console.error('Failed to parse saved form:', e);
        }
      }

      // Load saved ideas
      const savedIdeasStr = localStorage.getItem(IDEAS_STORAGE_KEY);
      if (savedIdeasStr) {
        try {
          const parsed = JSON.parse(savedIdeasStr);
          setSavedIdeas(parsed);
          if (parsed.length > 0) {
            setHasGenerated(true);
            setCurrentStep(3);
            setShowEmptyState(false);
          }
        } catch (e) {
          console.error('Failed to parse saved ideas:', e);
        }
      }

      // Load history
      const historyStr = localStorage.getItem(HISTORY_STORAGE_KEY);
      if (historyStr) {
        try {
          const parsed = JSON.parse(historyStr);
          setGenerationHistory(parsed);
        } catch (e) {
          console.error('Failed to parse history:', e);
        }
      }
    }
  }, []);

  // Auto-save form data
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(FORM_STORAGE_KEY, JSON.stringify(formData));
    }
  }, [formData]);

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
      const newIdeas = response.data.data.ideas;
      setSavedIdeas(newIdeas);
      localStorage.setItem(IDEAS_STORAGE_KEY, JSON.stringify(newIdeas));
      
      // Save to history
      const historyItem: GenerationHistory = {
        id: Date.now().toString(),
        timestamp: Date.now(),
        formData: formData,
        ideas: newIdeas,
      };
      
      const updatedHistory = [historyItem, ...generationHistory].slice(0, 10); // Keep last 10
      setGenerationHistory(updatedHistory);
      localStorage.setItem(HISTORY_STORAGE_KEY, JSON.stringify(updatedHistory));
      
      setHasGenerated(true);
      toast.success('Ide proyek berhasil dibuat!');
    },
    onError: (error: any) => {
      const errorMessage = error.response?.data?.error?.message || 'Gagal membuat ide proyek';
      toast.error(errorMessage, {
        action: {
          label: 'Coba Lagi',
          onClick: () => handleGenerate(),
        },
      });
    },
  });

  const handleNext = () => {
    if (currentStep === 1) {
      if (!formData.jurusan) {
        toast.error('Pilih jurusan terlebih dahulu');
        return;
      }
      if (formData.interests.length === 0) {
        toast.error('Tambahkan minimal 1 minat');
        return;
      }
    } else if (currentStep === 2) {
      if (!formData.project_type) {
        toast.error('Pilih tipe proyek terlebih dahulu');
        return;
      }
    }

    if (currentStep < 3) {
      setCurrentStep(currentStep + 1);
    }
  };

  const handleBack = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleGenerate = () => {
    generateMutation.mutate(formData);
  };

  const handleStartOver = () => {
    setCurrentStep(1);
    setHasGenerated(false);
    setShowEmptyState(false);
    setFormData({
      jurusan: '',
      interests: [],
      project_type: '',
      difficulty: 'intermediate',
    });
    localStorage.removeItem(FORM_STORAGE_KEY);
  };

  const handleGetStarted = () => {
    setShowEmptyState(false);
    setCurrentStep(1);
  };

  const handleClearAll = () => {
    setSavedIdeas([]);
    setHasGenerated(false);
    setCurrentStep(1);
    setShowEmptyState(true);
    localStorage.removeItem(IDEAS_STORAGE_KEY);
    setShowClearDialog(false);
    toast.success('Semua ide berhasil dihapus');
  };

  const handleDeleteIdea = (index: number) => {
    const newIdeas = savedIdeas.filter((_, i) => i !== index);
    setSavedIdeas(newIdeas);
    
    if (newIdeas.length === 0) {
      localStorage.removeItem(IDEAS_STORAGE_KEY);
      setHasGenerated(false);
      setCurrentStep(1);
      setShowEmptyState(true);
      toast.success('Semua ide telah dihapus');
    } else {
      localStorage.setItem(IDEAS_STORAGE_KEY, JSON.stringify(newIdeas));
      toast.success('Ide berhasil dihapus');
    }
  };

  const handleLike = (index: number) => {
    toast.success('Ide ditandai sebagai favorit!');
  };

  const handleSkip = (index: number) => {
    if (savedIdeas.length > 1) {
      handleDeleteIdea(index);
    } else {
      toast.info('Ini adalah ide terakhir');
    }
  };

  const handleSave = (index: number) => {
    toast.success('Fitur simpan ke portfolio akan segera hadir!');
  };

  const handleLoadHistory = (historyItem: GenerationHistory) => {
    setSavedIdeas(historyItem.ideas);
    setFormData(historyItem.formData);
    setHasGenerated(true);
    setShowEmptyState(false);
    setCurrentStep(3);
    localStorage.setItem(IDEAS_STORAGE_KEY, JSON.stringify(historyItem.ideas));
    toast.success('Riwayat berhasil dimuat!');
  };

  const handleDeleteHistory = (id: string) => {
    const updatedHistory = generationHistory.filter((item) => item.id !== id);
    setGenerationHistory(updatedHistory);
    localStorage.setItem(HISTORY_STORAGE_KEY, JSON.stringify(updatedHistory));
    toast.success('Riwayat berhasil dihapus');
  };

  const handleClearHistory = () => {
    setGenerationHistory([]);
    localStorage.removeItem(HISTORY_STORAGE_KEY);
    toast.success('Semua riwayat berhasil dihapus');
  };

  return (
    <div className="container mx-auto px-4 pb-24 pt-4 md:px-6 lg:px-8 md:pb-6 md:pt-6">
      {/* Header */}
      <div className="mx-auto mb-6 max-w-4xl text-center md:mb-12">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full border bg-primary/5 px-3 py-1 text-xs font-medium text-primary md:px-4 md:py-1.5 md:text-sm">
          <Sparkles className="h-3.5 w-3.5 md:h-4 md:w-4" />
          <span>AI-Powered</span>
        </div>
        <h1 className="mb-3 text-2xl font-bold tracking-tight md:mb-4 md:text-3xl lg:text-4xl xl:text-5xl">
          Generator Ide Proyek
        </h1>
        <p className="text-sm text-muted-foreground md:text-base lg:text-lg">
          Dapatkan inspirasi ide proyek portfolio yang sesuai dengan jurusan dan minatmu
        </p>
      </div>

      {/* Main Content */}
      <div className="mx-auto max-w-4xl">
        {/* Empty State */}
        {showEmptyState && (
          <EmptyState onGetStarted={handleGetStarted} />
        )}

        {/* Wizard Progress */}
        {!hasGenerated && !showEmptyState && (
          <div className="mb-8">
            <WizardProgress currentStep={currentStep} totalSteps={3} steps={WIZARD_STEPS} />
          </div>
        )}

        {/* Step 1: Profile */}
        {currentStep === 1 && !hasGenerated && !showEmptyState && (
          <Card className="border-2 p-4 shadow-sm md:p-6 lg:p-8">
            <div className="mb-4 md:mb-6">
              <h2 className="text-lg font-semibold md:text-xl lg:text-2xl">Profil Kamu</h2>
              <p className="mt-1 text-sm text-muted-foreground md:mt-2 md:text-base">
                Ceritakan tentang jurusan dan minat kamu
              </p>
            </div>

            <div className="space-y-6">
              {/* Jurusan */}
              <div className="space-y-2">
                <Label htmlFor="jurusan" className="text-sm font-medium">
                  Jurusan <span className="text-destructive">*</span>
                </Label>
                <Select
                  value={formData.jurusan}
                  onValueChange={(value) => setFormData({ ...formData, jurusan: value })}
                >
                  <SelectTrigger id="jurusan" className="h-11">
                    <SelectValue placeholder="Pilih jurusan kamu" />
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
              <InterestCombobox
                selectedInterests={formData.interests}
                onInterestsChange={(interests) => setFormData({ ...formData, interests })}
                suggestions={INTEREST_SUGGESTIONS}
                maxSelections={10}
              />
            </div>

            {/* Navigation */}
            <div className="mt-6 flex justify-end md:mt-8">
              <Button onClick={handleNext} size="lg" className="w-full min-w-32 md:w-auto">
                Lanjut
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
          </Card>
        )}

        {/* Step 2: Project Details */}
        {currentStep === 2 && !hasGenerated && !showEmptyState && (
          <Card className="border-2 p-4 shadow-sm md:p-6 lg:p-8">
            <div className="mb-4 md:mb-6">
              <h2 className="text-lg font-semibold md:text-xl lg:text-2xl">Detail Proyek</h2>
              <p className="mt-1 text-sm text-muted-foreground md:mt-2 md:text-base">
                Tentukan tipe dan tingkat kesulitan proyek
              </p>
            </div>

            <div className="space-y-6">
              {/* Project Type */}
              <div className="space-y-2">
                <Label htmlFor="project_type" className="text-sm font-medium">
                  Tipe Proyek <span className="text-destructive">*</span>
                </Label>
                <Select
                  value={formData.project_type}
                  onValueChange={(value) => setFormData({ ...formData, project_type: value })}
                >
                  <SelectTrigger id="project_type" className="h-11">
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
              <div className="space-y-3">
                <Label className="text-sm font-medium">Tingkat Kesulitan</Label>
                <div className="grid gap-3 sm:grid-cols-3">
                  {DIFFICULTY_LEVELS.map((level) => (
                    <button
                      key={level.value}
                      type="button"
                      onClick={() => setFormData({ ...formData, difficulty: level.value })}
                      className={`rounded-lg border-2 p-4 text-left transition-all hover:border-primary/50 ${
                        formData.difficulty === level.value
                          ? 'border-primary bg-primary/5'
                          : 'border-border'
                      }`}
                    >
                      <p className="font-semibold">{level.label}</p>
                      <p className="mt-1 text-xs text-muted-foreground">{level.description}</p>
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Navigation */}
            <div className="mt-6 flex flex-col gap-3 md:mt-8 md:flex-row md:justify-between">
              <Button onClick={handleBack} variant="outline" size="lg" className="w-full min-w-32 md:w-auto">
                <ChevronLeft className="mr-2 h-4 w-4" />
                Kembali
              </Button>
              <Button onClick={handleNext} size="lg" className="w-full min-w-32 md:w-auto">
                Lanjut
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
          </Card>
        )}

        {/* Step 3: Generate */}
        {currentStep === 3 && !hasGenerated && !showEmptyState && (
          <Card className="border-2 p-4 shadow-sm md:p-6 lg:p-8">
            <div className="mb-4 text-center md:mb-6">
              <h2 className="text-lg font-semibold md:text-xl lg:text-2xl">Siap Generate!</h2>
              <p className="mt-1 text-sm text-muted-foreground md:mt-2 md:text-base">
                Review informasi kamu sebelum membuat ide proyek
              </p>
            </div>

            {/* Summary */}
            <div className="mb-8 space-y-4 rounded-lg border bg-muted/30 p-6">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Jurusan</p>
                <p className="mt-1 font-semibold">{formData.jurusan}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Minat & Keahlian</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {formData.interests.map((interest) => (
                    <span
                      key={interest}
                      className="rounded-full bg-primary/10 px-3 py-1 text-sm font-medium text-primary"
                    >
                      {interest}
                    </span>
                  ))}
                </div>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Tipe Proyek</p>
                  <p className="mt-1 font-semibold">
                    {PROJECT_TYPES.find((t) => t.value === formData.project_type)?.label}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Tingkat Kesulitan</p>
                  <p className="mt-1 font-semibold">
                    {DIFFICULTY_LEVELS.find((d) => d.value === formData.difficulty)?.label}
                  </p>
                </div>
              </div>
            </div>

            {/* Navigation */}
            <div className="flex flex-col gap-3 sm:flex-row sm:justify-between">
              <Button onClick={handleBack} variant="outline" size="lg" className="w-full min-w-32 sm:w-auto">
                <ChevronLeft className="mr-2 h-4 w-4" />
                Kembali
              </Button>
              <Button
                onClick={handleGenerate}
                size="lg"
                className="w-full min-w-48 sm:w-auto"
                disabled={generateMutation.isPending}
              >
                <Sparkles className="mr-2 h-5 w-5" />
                Generate Ide Proyek
              </Button>
            </div>
          </Card>
        )}

        {/* Loading State */}
        {generateMutation.isPending && <LoadingProgress />}

        {/* Results */}
        {hasGenerated && savedIdeas.length > 0 && !generateMutation.isPending && (
          <div className="space-y-6">
            {/* History */}
            {generationHistory.length > 1 && (
              <IdeasHistory
                history={generationHistory}
                onLoadHistory={handleLoadHistory}
                onDeleteHistory={handleDeleteHistory}
                onClearAll={handleClearHistory}
              />
            )}

            {/* Action Buttons */}
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-xl font-semibold md:text-2xl">Ide Proyek Kamu</h2>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleStartOver}
                  className="gap-2"
                >
                  <RotateCcw className="h-4 w-4" />
                  <span className="hidden sm:inline">Buat Baru</span>
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowClearDialog(true)}
                  className="gap-2 text-muted-foreground hover:text-destructive"
                >
                  <Trash2 className="h-4 w-4" />
                  <span className="hidden sm:inline">Hapus Semua</span>
                </Button>
              </div>
            </div>

            {/* Carousel */}
            <IdeaCarousel
              ideas={savedIdeas}
              onLike={handleLike}
              onSkip={handleSkip}
              onSave={handleSave}
              onDelete={handleDeleteIdea}
            />
          </div>
        )}
      </div>

      {/* Clear Confirmation Dialog */}
      <AlertDialog open={showClearDialog} onOpenChange={setShowClearDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Semua Ide?</AlertDialogTitle>
            <AlertDialogDescription>
              Semua ide proyek yang telah di-generate akan dihapus. Tindakan ini tidak dapat dibatalkan.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleClearAll} className="bg-destructive hover:bg-destructive/90">
              Hapus Semua
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

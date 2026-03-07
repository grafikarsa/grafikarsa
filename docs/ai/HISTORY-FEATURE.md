# AI Ideas Generator - History Feature

## 📚 Overview

Fitur History memungkinkan user untuk menyimpan dan mengakses kembali hasil generate sebelumnya. Setiap kali user generate ide proyek, hasilnya akan otomatis disimpan ke history.

**Location:** History card ditampilkan di halaman results (setelah generate), bukan di halaman form.

**Visibility:** History hanya muncul jika ada lebih dari 1 generation (karena current generation sudah ditampilkan di carousel).

---

## ✨ Features

### 1. **Auto-Save to History**
- Setiap generate otomatis disimpan
- Menyimpan form data dan hasil ideas
- Maximum 10 history items (FIFO)
- Timestamp untuk setiap entry

### 2. **History Display**
- Collapsible card untuk menghemat space
- Menampilkan info penting:
  - Waktu generate (relative time)
  - Jurusan
  - Project type
  - Jumlah ide
  - Interests (max 3 + counter)
- Hover actions untuk Lihat dan Hapus

### 3. **Load from History**
- Click "Lihat" untuk load history
- Restore form data dan ideas
- Langsung tampilkan di carousel
- Toast notification

### 4. **Delete History**
- Delete individual history item
- Delete all history
- Confirmation untuk clear all
- Toast notification

---

## 🎨 UI Design

### History Card
```
┌─────────────────────────────────────────────────┐
│ 📜 Riwayat Generate [10]    [Hapus Semua] [▼]  │
├─────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────┐ │
│ │ 2 jam lalu                    [Lihat] [🗑️]  │ │
│ │ Rekayasa Perangkat Lunak • Web App • 5 ide  │ │
│ │ [Web Dev] [UI/UX] [Database] +2             │ │
│ └─────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────┐ │
│ │ 1 hari lalu                   [Lihat] [🗑️]  │ │
│ │ Desain Grafis • Design • 5 ide              │ │
│ │ [Graphic Design] [3D] [Animation]           │ │
│ └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

### States
- **Collapsed:** Show header only
- **Expanded:** Show all history items
- **Empty:** Hide component
- **Hover:** Show action buttons

---

## 💾 Storage Structure

### localStorage Keys
```typescript
'ai_ideas_history' // Array of GenerationHistory
```

### Data Structure
```typescript
interface GenerationHistory {
  id: string;              // Unique ID (timestamp)
  timestamp: number;       // Unix timestamp
  formData: {
    jurusan: string;
    interests: string[];
    project_type: string;
    difficulty: string;
  };
  ideas: ProjectIdea[];    // Generated ideas
}
```

### Example Data
```json
[
  {
    "id": "1709812800000",
    "timestamp": 1709812800000,
    "formData": {
      "jurusan": "Rekayasa Perangkat Lunak",
      "interests": ["Web Development", "UI/UX Design"],
      "project_type": "web_app",
      "difficulty": "intermediate"
    },
    "ideas": [
      {
        "title": "Sistem Manajemen Perpustakaan",
        "description": "...",
        "technologies": ["React", "Node.js"],
        "difficulty": "intermediate",
        "estimated_time": "4-6 minggu",
        "learning_goals": ["..."]
      }
    ]
  }
]
```

---

## 🔄 User Flow

### Generate New Ideas
```
Fill Form
  ↓
Generate
  ↓
Save to History (auto)
  ↓
Display Results + History (if > 1)
```

### Load from History
```
View Results Page
  ↓
See History Card (if multiple generations)
  ↓
Click "Lihat" on history item
  ↓
Load Form Data + Ideas
  ↓
Display in Carousel
```

### Delete History
```
Click Delete Icon
  ↓
Remove from Array
  ↓
Update localStorage
  ↓
Toast Notification
```

---

## 🎯 Features Detail

### 1. Relative Time Display
```typescript
const formatDate = (timestamp: number) => {
  const diffMins = Math.floor((now - date) / 60000);
  
  if (diffMins < 1) return 'Baru saja';
  if (diffMins < 60) return `${diffMins} menit lalu`;
  if (diffHours < 24) return `${diffHours} jam lalu`;
  if (diffDays < 7) return `${diffDays} hari lalu`;
  
  return date.toLocaleDateString('id-ID');
};
```

### 2. FIFO Queue (Max 10)
```typescript
const updatedHistory = [newItem, ...history].slice(0, 10);
```

### 3. Hover Actions
- Actions hidden by default
- Show on hover (desktop)
- Always visible on mobile
- Smooth opacity transition

### 4. Collapsible UI
- Collapsed by default
- Toggle with chevron icon
- Smooth height transition
- Preserve state in session

---

## 📱 Responsive Design

### Desktop (> 1024px)
- Full width history card
- Hover actions
- 2-line layout for info

### Tablet (768px - 1024px)
- Adjusted spacing
- Visible actions
- Wrapped interests

### Mobile (< 768px)
- Stacked layout
- Always visible actions
- Compact spacing
- Touch-friendly buttons

---

## ♿ Accessibility

### Keyboard Navigation
- Tab through history items
- Enter to load history
- Delete to remove item
- Escape to collapse

### Screen Reader
- ARIA labels for actions
- Descriptive button text
- Semantic HTML structure
- Live region for updates

### Visual
- Clear hover states
- Focus indicators
- Color contrast 4.5:1
- Icon + text labels

---

## 🔧 Implementation

### Component
```typescript
<IdeasHistory
  history={generationHistory}
  onLoadHistory={handleLoadHistory}
  onDeleteHistory={handleDeleteHistory}
  onClearAll={handleClearHistory}
/>
``
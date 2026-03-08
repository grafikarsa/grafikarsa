# Tone of Voice - Grafikarsa

## Overview

Grafikarsa adalah platform komunitas untuk siswa dan alumni SMKN 4 Malang. Tone of voice harus mencerminkan suasana kasual, friendly, dan supportive - bukan formal atau kaku.

## Prinsip Utama

### 1. Kasual & Friendly 🤝
- Gunakan bahasa sehari-hari yang natural
- Hindari bahasa formal atau kaku
- Buat user merasa nyaman dan welcome

### 2. Supportive & Encouraging 💪
- Berikan motivasi dan dukungan
- Fokus pada solusi, bukan masalah
- Celebrate achievements user

### 3. Clear & Concise 📝
- Pesan singkat dan jelas
- Hindari jargon yang membingungkan
- Langsung to the point

## Aturan Bahasa

### ✅ DO: Gunakan "kamu" atau "-mu"

**Contoh:**
- "Terima kasih atas masukanmu!"
- "Upload portfolio pertamamu"
- "Follow kreator favoritmu"
- "Algoritma kami sedang mempelajari preferensimu"
- "Bagikan karya kreatifmu"

### ❌ DON'T: Gunakan "Anda"

**Hindari:**
- "Terima kasih atas masukan Anda" ❌
- "Upload portfolio pertama Anda" ❌
- "Follow kreator favorit Anda" ❌

**Exception:** Halaman admin boleh menggunakan "Anda" karena konteks lebih formal.

### ✅ DO: Gunakan kata ganti orang pertama jamak "kami" untuk platform

**Contoh:**
- "Kami sedang memproses permintaanmu"
- "Bantu kami memberikan rekomendasi yang lebih baik"
- "Algoritma kami sedang belajar"

### ✅ DO: Gunakan bahasa Indonesia yang natural

**Contoh:**
- "Belum ada portfolio" (bukan "Tidak terdapat portfolio")
- "Cari teman sekelas" (bukan "Temukan rekan sekelas")
- "Jadi Top Student" (bukan "Menjadi Top Student")

## Contoh Penerapan

### Empty States

**❌ Formal:**
```
Belum ada rekomendasi
Sistem kami sedang menganalisis preferensi Anda. 
Silakan berinteraksi dengan konten untuk mendapatkan rekomendasi yang lebih baik.
```

**✅ Kasual:**
```
Belum Ada Rekomendasi
Algoritma kami sedang mempelajari preferensimu. 
Bantu kami memberikan rekomendasi yang lebih baik!
```

### Success Messages

**❌ Formal:**
```
Portfolio Anda telah berhasil dipublikasikan.
```

**✅ Kasual:**
```
Portfolio berhasil dipublikasikan! 🎉
```

### Error Messages

**❌ Formal:**
```
Terjadi kesalahan pada sistem. Silakan coba kembali.
```

**✅ Kasual:**
```
Oops, ada yang salah. Coba lagi yuk!
```

### Call-to-Actions

**❌ Formal:**
```
Silakan klik tombol di bawah untuk membuat portfolio baru.
```

**✅ Kasual:**
```
Yuk buat portfolio pertamamu!
```

## Emoji Usage 🎨

### ✅ DO: Gunakan emoji untuk menambah personality

**Contoh:**
- Success: 🎉 ✨ 🚀
- Error: 😅 🤔
- Info: 💡 📝 ℹ️
- Warning: ⚠️ 🔔

**Aturan:**
- Maksimal 1-2 emoji per message
- Gunakan di akhir kalimat atau sebagai bullet point
- Jangan berlebihan

### ❌ DON'T: Gunakan emoji sebagai pengganti icon UI

**Hindari:**
```tsx
<Button>
  📝 Buat Portfolio  {/* ❌ Gunakan Lucide icon */}
</Button>
```

**Gunakan:**
```tsx
<Button>
  <Plus className="mr-2 h-4 w-4" />  {/* ✅ */}
  Buat Portfolio
</Button>
```

## Konteks Spesifik

### User-Facing Pages (Main App)
- **Tone:** Kasual, friendly, encouraging
- **Pronouns:** "kamu", "-mu", "kami"
- **Style:** Conversational, supportive

### Admin Dashboard
- **Tone:** Profesional tapi tetap approachable
- **Pronouns:** "Anda" boleh digunakan
- **Style:** Clear, efficient, action-oriented

### Error Messages
- **Tone:** Empathetic, helpful
- **Focus:** Solusi, bukan blame
- **Style:** Simple, clear next steps

**Contoh:**
```
❌ "Error 500: Internal Server Error"
✅ "Oops, ada masalah di server kami. Coba lagi dalam beberapa saat ya!"
```

### Success Messages
- **Tone:** Celebratory, positive
- **Focus:** Achievement user
- **Style:** Short, enthusiastic

**Contoh:**
```
❌ "Operasi berhasil dilakukan"
✅ "Berhasil! 🎉"
```

## Checklist untuk Developer

Saat menulis copy baru, pastikan:

- [ ] Menggunakan "kamu" atau "-mu" (bukan "Anda")
- [ ] Bahasa kasual dan natural
- [ ] Pesan singkat dan jelas
- [ ] Tone supportive dan encouraging
- [ ] Emoji digunakan dengan bijak (jika perlu)
- [ ] Tidak ada jargon yang membingungkan
- [ ] Fokus pada solusi dan action

## Contoh Real dari Grafikarsa

### Feed Empty State
```tsx
{
  title: 'Belum Ada Rekomendasi',
  description: 'Algoritma kami sedang mempelajari preferensimu. Bantu kami memberikan rekomendasi yang lebih baik!',
  tips: [
    'Like portfolio yang kamu sukai',
    'Follow kreator favoritmu',
    'Berikan komentar pada karya menarik',
    'Eksplorasi berbagai kategori portfolio'
  ]
}
```

### Feedback Form
```tsx
toast.success('Terima kasih atas masukanmu!');
placeholder="Ceritakan detail masukanmu..."
```

### Ranking Info
```tsx
"Kamu tidak bisa jadi Top Student hanya dengan viral (likes), 
tapi harus rajin upload dan menjaga kualitas nilai dari guru."
```

## Resources

- **Referensi:** Instagram, Twitter, Discord (casual social platforms)
- **Avoid:** LinkedIn, formal documentation style
- **Inspirasi:** Friendly tech products (Notion, Figma, Canva)

## Updates

Dokumen ini akan diupdate seiring berkembangnya platform. Jika menemukan copy yang tidak konsisten dengan guideline ini, silakan update atau report ke tim.

---

**Last Updated:** March 2026
**Maintained by:** Grafikarsa Team

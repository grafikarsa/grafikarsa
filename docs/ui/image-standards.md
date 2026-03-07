# Panduan Standar Aset Visual (Image Standards Guide)

Dokumen ini menjelaskan standar teknis untuk aset gambar di platform Grafikarsa, mencakup dimensi, rasio aspek, dan praktik terbaik aksesibilitas/responsivitas.

## 1. Foto Profil (Avatar)
Digunakan untuk representasi identitas pengguna di seluruh platform.

- **Dimensi**: 400 x 400 px (Minimal) | 800 x 800 px (Rekomendasi)
- **Rasio**: 1:1 (Square)
- **Focal Point**: Tengah (Center)
- **Masking**: Lingkaran (Circular)
- **Contoh Penggunaan**: 
    - **LinkedIn**: 400x400 px.
    - **X/Twitter**: 400x400 px.
- **Panduan**: Gunakan "Face-Centered" cropping. Area 10-15% dari tepi adalah "danger zone" yang akan terpotong oleh masking lingkaran.

## 2. Banner Profil (User Cover)
Header utama di halaman profil pengguna.

- **Dimensi**: 1500 x 500 px
- **Rasio**: 3:1
- **Responsivitas**: Center-Cropped.
- **Safe Zone**: Teks atau elemen branding utama harus berada di area tengah (pusat 900px lebar).
- **Contoh Penggunaan**:
    - **X/Twitter**: 1500x500 px.
    - **Behance**: Banner profil sangat lebar untuk menonjolkan estetika.
- **Behavior**: Pada perangkat mobile, bagian kiri dan kanan akan terpotong secara otomatis oleh CSS `object-fit: cover`.

## 3. Thumbnail Portfolio (Grid Thumbnails)
Sampul karya yang muncul di halaman beranda atau galeri.

- **Dimensi**: 1200 x 900 px
- **Rasio**: 4:3
- **Focal Point**: 1/3 Atas (Rule of Thirds)
- **Contoh Penggunaan**:
    - **Dribbble**: 1600x1200 px (4:3) adalah standar emas untuk desain digital.
    - **Behance**: Menggunakan grid yang lebih fleksibel, namun cover biasanya tetap di rasio 4:3 atau 3:2.
- **Panduan**: Rasio 4:3 memberikan ruang vertikal lebih banyak daripada 16:9, sehingga sangat cocok untuk menampilkan screenshot aplikasi atau detail tipografi.

---

### Ringkasan Ukuran (Cheat Sheet)

| Tipe Aset | Rasio | Rekomendasi (px) | Batas Maksimum |
| :--- | :--- | :--- | :--- |
| **Foto Profil** | 1:1 | 800 x 800 | 2000 x 2000 |
| **Banner Profil** | 3:1 | 1500 x 500 | 3000 x 1000 |
| **Thumb Portfolio**| 4:3 | 1200 x 900 | 1600 x 1200 |
| **Gambar Isi** | Adaptive | Minimal 1920w | 3840w (4K) |

---

### Praktik Terbaik (Best Practices)

1. **Format File**:
    - **JPEG**: Gunakan untuk foto pemandangan atau potret (kompresi 80%).
    - **PNG**: Gunakan untuk logo, ilustrasi dengan detail tajam, atau jika memerlukan transparansi.
    - **WebP**: Sangat disarankan untuk performa web (ukuran file lebih kecil 25-30% dibanding JPEG/PNG).

2. **Crop & Responsive Guide**:
    - Gunakan properti CSS `object-position: center center` untuk memastikan fokus gambar tetap di tengah saat di-*resize*.
    - Hindari meletakkan teks penting di bagian paling pinggir (atas, bawah, kiri, kanan) gambar banner.

3. **Optimasi Solusi**:
    - Selalu jalankan kompresi gambar (seperti TinyPNG atau plugin Sharp di backend) sebelum disimpan ke storage (MinIO) untuk menghemat bandwidth.

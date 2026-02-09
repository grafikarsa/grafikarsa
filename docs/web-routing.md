# Grafikarsa Frontend Routes

Dokumen ini menjelaskan struktur routing untuk frontend Grafikarsa menggunakan Next.js App Router dengan konsep **Shared Routes + Conditional Rendering**.

## Konsep Utama

### Shared Routes dengan Conditional Rendering

Alih-alih memisahkan route menjadi `(public)` dan `(dashboard)`, semua halaman utama digabungkan dalam satu route group `(main)`. URL tetap sama, tapi konten berbeda berdasarkan status autentikasi.

**Keuntungan:**
- URL konsisten dan mudah diingat
- Tidak ada redirect saat login/logout
- Komponen reusable dengan conditional logic
- SEO friendly

---

## Struktur Route Groups

```
app/
├── (main)/                      # Route group utama (guest & authenticated)
│   ├── layout.tsx               # Conditional layout (GuestNavbar vs StudentSidebar+Header)
│   ├── page.tsx                 # / (Landing vs Feed)
│   ├── login/
│   │   └── page.tsx             # /login
│   ├── users/
│   │   └── page.tsx             # /users (Daftar siswa & alumni)
│   ├── portfolios/
│   │   └── page.tsx             # /portfolios (Katalog portofolio)
│   └── [username]/              # Dynamic user routes
│       ├── page.tsx             # /:username (Profil user)
│       ├── edit/
│       │   └── page.tsx         # /:username/edit (Edit profil - owner only)
│       ├── portfolios/
│       │   └── new/
│       │       └── page.tsx     # /:username/portfolios/new (Create - owner only)
│       └── [slug]/
│           ├── page.tsx         # /:username/:slug (Detail portofolio)
│           └── edit/
│               └── page.tsx     # /:username/:slug/edit (Edit portofolio - owner only)
│
├── admin/                       # Route group admin (terpisah)
│   ├── [secret-path]/           # Configurable secret path
│   │   └── page.tsx             # /admin/:secret-path (Login admin)
│   └── (dashboard)/             # Admin dashboard (protected)
│       ├── layout.tsx           # Admin layout dengan sidebar
│       ├── page.tsx             # /admin (Dashboard overview)
│       ├── users/
│       │   └── page.tsx         # /admin/users (Kelola user - modal CRUD)
│       ├── portfolios/
│       │   └── page.tsx         # /admin/portfolios (Kelola portofolio user - modal CRUD)
│       ├── moderation/
│       │   └── page.tsx         # /admin/moderation (Moderasi portofolio pending)
│       ├── majors/
│       │   └── page.tsx         # /admin/majors (Kelola jurusan - modal CRUD)
│       ├── classes/
│       │   └── page.tsx         # /admin/classes (Kelola kelas - modal CRUD)
│       ├── academic-years/
│       │   └── page.tsx         # /admin/academic-years (Kelola tahun ajaran - modal CRUD)
│       └── tags/
│           └── page.tsx         # /admin/tags (Kelola tags - modal CRUD)
│
└── api/                         # API routes (jika perlu)
    └── auth/
        └── [...nextauth]/
            └── route.ts
```

---

## Detail Routes

### Main Routes (Guest & Authenticated)

| Route | Guest | Authenticated | Deskripsi |
|-------|-------|---------------|-----------|
| `/` | Landing page | Feed timeline | Beranda |
| `/login` | Form login | Redirect → `/` | Login user |
| `/users` | ✅ List users | ✅ List users | Daftar siswa & alumni |
| `/portfolios` | ✅ Katalog | ✅ Katalog | Explore portofolio |
| `/:username` | ✅ Profil (published) | ✅ Profil | Detail profil user |
| `/:username/edit` | 🔒 → `/login` | ✅ Owner only | Edit profil sendiri |
| `/:username/portfolios/new` | 🔒 → `/login` | ✅ Owner only | Buat portofolio baru |
| `/:username/:slug` | ✅ Detail | ✅ Detail | View portofolio |
| `/:username/:slug/edit` | 🔒 → `/login` | ✅ Owner only | Edit portofolio |

### Admin Routes

| Route | Deskripsi |
|-------|-----------|
| `/admin/:secret` | Login admin (secret path configurable) |
| `/admin` | Dashboard overview |
| `/admin/users` | Kelola user (CRUD via modal) |
| `/admin/portfolios` | Kelola semua portofolio user (CRUD via modal) |
| `/admin/moderation` | Moderasi portofolio pending_review (approve/reject) |
| `/admin/majors` | Kelola jurusan (CRUD via modal) |
| `/admin/classes` | Kelola kelas (CRUD via modal) |
| `/admin/academic-years` | Kelola tahun ajaran (CRUD via modal) |
| `/admin/tags` | Kelola tags (CRUD via modal) |

---

## Conditional Rendering Strategy

### Layout `(main)/layout.tsx`

```tsx
export default function MainLayout({ children }) {
  const { user, isLoading } = useAuth()
  
  if (isLoading) return <LoadingScreen />
  
  // Guest: Navbar only
  if (!user) {
    return (
      <>
        <GuestNavbar />
        <main>{children}</main>
        <Footer />
      </>
    )
  }
  
  // Authenticated: Sidebar + Header
  return (
    <div className="flex">
      <StudentSidebar />
      <div className="flex-1">
        <StudentHeader />
        <main>{children}</main>
      </div>
    </div>
  )
}
```

### Home Page `/page.tsx`

```tsx
export default function HomePage() {
  const { user } = useAuth()
  
  if (!user) {
    return <LandingPage />  // Hero, About, FAQ sections
  }
  
  return <FeedPage />  // Timeline portofolio terbaru
}
```

### Profile Page `/:username/page.tsx`

```tsx
export default function ProfilePage({ params }) {
  const { user } = useAuth()
  const profile = await getUser(params.username)
  const isOwner = user?.username === params.username
  
  return (
    <ProfileView 
      profile={profile} 
      isOwner={isOwner}
      showAllPortfolios={isOwner}  // Owner sees all statuses
    />
  )
}
```

---

## Navigation Components

### Guest Navbar

```
┌─────────────────────────────────────────────────────────────┐
│ [Logo]   Beranda   Siswa   Portofolio   [Theme] [Login]     │
└─────────────────────────────────────────────────────────────┘
```

Menu items:
- **Beranda** → `/`
- **Siswa** → `/users`
- **Portofolio** → `/portfolios`
- **Theme Toggle** → Dark/Light mode
- **Login** → `/login`

### Student Sidebar

```
┌──────────────────┐
│ [Logo]           │
├──────────────────┤
│ 🏠 Feed          │  → /
│ Tambah Portofolio│  → /:username/portfolios/new
│ 🔍 Search        │  → muncul popover untuk memilih pencarian: user atau portofolio, jika cari users maka ke /users, jika portofolio maka ke /portfolios
│ 👤 Profil Saya   │  → /:username
├──────────────────┤
│ [User Info]      │
│ [Logout]         │
└──────────────────┘
```

---

## URL Examples

| Aksi | URL |
|------|-----|
| Landing/Feed | `grafikarsa.com` |
| Login | `grafikarsa.com/login` |
| Daftar/list user | `grafikarsa.com/users` |
| Katalog/list portofolio | `grafikarsa.com/portfolios` |
| Profil user | `grafikarsa.com/budisantoso` |
| Edit profil | `grafikarsa.com/budisantoso/edit` |
| Buat portofolio | `grafikarsa.com/budisantoso/portfolios/new` |
| Detail portofolio | `grafikarsa.com/budisantoso/website-toko-online` |
| Edit portofolio | `grafikarsa.com/budisantoso/website-toko-online/edit` |
| Admin login | `grafikarsa.com/admin/secretpath123` |
| Admin dashboard | `grafikarsa.com/admin` |
| Admin kelola user | `grafikarsa.com/admin/users` |
| Admin kelola portofolio | `grafikarsa.com/admin/portfolios` |
| Admin moderasi | `grafikarsa.com/admin/moderation` |

---

## Protected Routes

### Middleware Protection

```tsx
// middleware.ts
export function middleware(request: NextRequest) {
  const token = request.cookies.get('token')
  const path = request.nextUrl.pathname
  
  // Protected patterns untuk user (owner-only routes)
  const ownerOnlyPatterns = [
    /^\/[^\/]+\/edit$/,              // /:username/edit
    /^\/[^\/]+\/portfolios\/new$/,   // /:username/portfolios/new
    /^\/[^\/]+\/[^\/]+\/edit$/,      // /:username/:slug/edit
  ]
  
  if (ownerOnlyPatterns.some(p => p.test(path)) && !token) {
    return NextResponse.redirect(new URL('/login', request.url))
  }
  
  // Admin routes
  if (path.startsWith('/admin') && !path.match(/^\/admin\/[^\/]+$/)) {
    // Check admin token
  }
  
  return NextResponse.next()
}
```

### Owner Validation (Server Component)

```tsx
// /:username/edit/page.tsx
export default async function EditProfilePage({ params }) {
  const session = await getSession()
  
  // Not logged in
  if (!session) {
    redirect('/login')
  }
  
  // Not owner
  if (session.user.username !== params.username) {
    notFound() // atau redirect ke profil
  }
  
  return <EditProfileForm user={session.user} />
}
```

---

## Portfolio Status & Visibility

| Status | Owner | Other Users | Guest | Admin |
|--------|-------|-------------|-------|-------|
| `draft` | ✅ View/Edit | ❌ | ❌ | ✅ View |
| `pending_review` | ✅ View | ❌ | ❌ | ✅ View/Moderate |
| `rejected` | ✅ View/Edit | ❌ | ❌ | ✅ View |
| `published` | ✅ View/Edit | ✅ View | ✅ View | ✅ View |
| `archived` | ✅ View/Edit | ❌ | ❌ | ✅ View |

---

## Admin CRUD Pattern (Modal)

Admin tidak menggunakan halaman terpisah untuk create/edit. Semua operasi CRUD dilakukan via modal popup di halaman list.

```tsx
// /admin/users/page.tsx
export default function AdminUsersPage() {
  const [isCreateOpen, setCreateOpen] = useState(false)
  const [editUser, setEditUser] = useState(null)
  
  return (
    <>
      <DataTable 
        data={users}
        onEdit={(user) => setEditUser(user)}
        onDelete={(user) => handleDelete(user)}
      />
      
      <Button onClick={() => setCreateOpen(true)}>Tambah User</Button>
      
      {/* Create Modal */}
      <Modal open={isCreateOpen} onClose={() => setCreateOpen(false)}>
        <UserForm onSubmit={handleCreate} />
      </Modal>
      
      {/* Edit Modal */}
      <Modal open={!!editUser} onClose={() => setEditUser(null)}>
        <UserForm user={editUser} onSubmit={handleUpdate} />
      </Modal>
    </>
  )
}
```

---

## Complete Route Reference

### Main Routes

| # | Route | File Path | Guest | Auth | Owner Only | Deskripsi |
|---|-------|-----------|-------|------|------------|-----------|
| 1 | `/` | `(main)/page.tsx` | Landing | Feed | - | Beranda |
| 2 | `/login` | `(main)/login/page.tsx` | ✅ | → `/` | - | Login |
| 3 | `/users` | `(main)/users/page.tsx` | ✅ | ✅ | - | Daftar user |
| 4 | `/portfolios` | `(main)/portfolios/page.tsx` | ✅ | ✅ | - | Katalog portofolio |
| 5 | `/:username` | `(main)/[username]/page.tsx` | ✅ | ✅ | - | Profil user |
| 6 | `/:username/edit` | `(main)/[username]/edit/page.tsx` | 🔒 | 🔒 | ✅ | Edit profil |
| 7 | `/:username/portfolios/new` | `(main)/[username]/portfolios/new/page.tsx` | 🔒 | 🔒 | ✅ | Create portofolio |
| 8 | `/:username/:slug` | `(main)/[username]/[slug]/page.tsx` | ✅ | ✅ | - | Detail portofolio |
| 9 | `/:username/:slug/edit` | `(main)/[username]/[slug]/edit/page.tsx` | 🔒 | 🔒 | ✅ | Edit portofolio |

### Admin Routes

| # | Route | File Path | Deskripsi |
|---|-------|-----------|-----------|
| 12 | `/admin/:secret` | `admin/[secret-path]/page.tsx` | Login admin |
| 13 | `/admin` | `admin/(dashboard)/page.tsx` | Dashboard |
| 14 | `/admin/users` | `admin/(dashboard)/users/page.tsx` | Kelola user |
| 15 | `/admin/portfolios` | `admin/(dashboard)/portfolios/page.tsx` | Kelola portofolio |
| 16 | `/admin/moderation` | `admin/(dashboard)/moderation/page.tsx` | Moderasi pending |
| 17 | `/admin/majors` | `admin/(dashboard)/majors/page.tsx` | Kelola jurusan |
| 18 | `/admin/classes` | `admin/(dashboard)/classes/page.tsx` | Kelola kelas |
| 19 | `/admin/academic-years` | `admin/(dashboard)/academic-years/page.tsx` | Kelola tahun ajaran |
| 20 | `/admin/tags` | `admin/(dashboard)/tags/page.tsx` | Kelola tags |

### Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Accessible |
| 🔒 | Protected (redirect to login) |
| → | Redirect to |
| Owner Only | Hanya bisa diakses oleh pemilik akun |

---

## Dynamic Route Parameters

| Parameter | Type | Example | Used In |
|-----------|------|---------|---------|
| `:username` | string | `budisantoso` | `/:username/*` |
| `:slug` | string | `website-toko-online` | `/:username/:slug/*` |
| `:secret` | string | `loginadmin` | `/admin/:secret` |

---
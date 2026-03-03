# Grafikarsa API Inspector - Shows raw JSON request/response
# Usage: .\scripts\api_inspect.ps1 [-BaseUrl "http://localhost:8080"]

param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$Endpoint = "",
    [switch]$All
)

$ApiUrl = "$BaseUrl/api/v1"
$Global:AccessToken = ""
$Global:OutputLines = @()

# Setup log directory and file
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$LogDir = Join-Path $ScriptDir "log"
if (-not (Test-Path $LogDir)) {
    New-Item -ItemType Directory -Path $LogDir -Force | Out-Null
}

# Find next available log filename
$LogFile = Join-Path $LogDir "log.md"
if (Test-Path $LogFile) {
    $counter = 1
    while (Test-Path (Join-Path $LogDir "log$counter.md")) {
        $counter++
    }
    $LogFile = Join-Path $LogDir "log$counter.md"
}

function Write-Log {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
    $Global:OutputLines += $Message
}

function Write-Header {
    param([string]$Title)
    Write-Log ""
    Write-Log ("=" * 80) -Color Cyan
    Write-Log "  $Title" -Color Cyan
    Write-Log ("=" * 80) -Color Cyan
}

function Write-SubHeader {
    param([string]$Title)
    Write-Log ""
    Write-Log ("-" * 60) -Color DarkGray
    Write-Log "  $Title" -Color Yellow
    Write-Log ("-" * 60) -Color DarkGray
}

function Format-Json {
    param([string]$Json)
    try {
        $obj = $Json | ConvertFrom-Json
        return $obj | ConvertTo-Json -Depth 10
    } catch {
        return $Json
    }
}

function Invoke-ApiInspect {
    param(
        [string]$Method,
        [string]$Endpoint,
        [hashtable]$Body = @{},
        [bool]$UseAuth = $false
    )
    
    $uri = "$ApiUrl$Endpoint"
    $headers = @{
        "Content-Type" = "application/json"
    }
    
    if ($UseAuth -and $Global:AccessToken) {
        $headers["Authorization"] = "Bearer $Global:AccessToken"
    }
    
    Write-SubHeader "$Method $Endpoint"
    
    # Show request
    Write-Log ""
    Write-Log "REQUEST:" -Color Green
    Write-Log "  URL: $uri" -Color Gray
    Write-Log "  Method: $Method" -Color Gray
    Write-Log "  Headers:" -Color Gray
    foreach ($key in $headers.Keys) {
        $value = $headers[$key]
        if ($key -eq "Authorization") {
            $value = "Bearer <token>"
        }
        Write-Log "    $key : $value" -Color Gray
    }
    
    if ($Body.Count -gt 0) {
        $bodyJson = $Body | ConvertTo-Json -Depth 5
        Write-Log "  Body:" -Color Gray
        Write-Log $bodyJson -Color DarkYellow
    }
    
    # Make request
    try {
        $params = @{
            Method = $Method
            Uri = $uri
            Headers = $headers
            ErrorAction = "Stop"
        }
        
        if ($Body.Count -gt 0 -and $Method -ne "GET") {
            $params["Body"] = ($Body | ConvertTo-Json -Depth 10)
        }
        
        $response = Invoke-WebRequest @params
        $statusCode = $response.StatusCode
        $content = $response.Content
        
        Write-Log ""
        Write-Log "RESPONSE:" -Color Green
        Write-Log "  Status: $statusCode" -Color $(if ($statusCode -lt 400) { "Green" } else { "Red" })
        Write-Log "  Body:" -Color Gray
        Write-Log (Format-Json $content)
        
        return @{
            StatusCode = $statusCode
            Content = $content | ConvertFrom-Json
        }
    }
    catch {
        $statusCode = 0
        $errorContent = ""
        
        if ($_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
            try {
                $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
                $errorContent = $reader.ReadToEnd()
            } catch {}
        }
        
        Write-Log ""
        Write-Log "RESPONSE:" -Color Green
        Write-Log "  Status: $statusCode" -Color Red
        if ($errorContent) {
            Write-Log "  Body:" -Color Gray
            Write-Log (Format-Json $errorContent) -Color Red
        } else {
            Write-Log "  Error: $($_.Exception.Message)" -Color Red
        }
        
        return @{
            StatusCode = $statusCode
            Content = $null
        }
    }
}

function Login {
    $result = Invoke-ApiInspect -Method "POST" -Endpoint "/auth/login" -Body @{
        username = "admin"
        password = "password"
    }
    
    if ($result.Content -and $result.Content.data.access_token) {
        $Global:AccessToken = $result.Content.data.access_token
        Write-Log ""
        Write-Log "  [Logged in successfully, token saved]" -Color Green
    }
}

function Inspect-AllEndpoints {
    Write-Header "GRAFIKARSA API INSPECTOR"
    Write-Log "Base URL: $ApiUrl" -Color Magenta
    Write-Log "Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -Color Magenta
    
    # Check server
    try {
        $null = Invoke-WebRequest -Uri "$ApiUrl/health" -TimeoutSec 5 -ErrorAction Stop
    } catch {
        Write-Log "ERROR: Cannot connect to API server" -Color Red
        exit 1
    }
    
    # ========== PUBLIC ENDPOINTS ==========
    Write-Header "PUBLIC ENDPOINTS"
    
    Invoke-ApiInspect -Method "GET" -Endpoint "/jurusan"
    Invoke-ApiInspect -Method "GET" -Endpoint "/kelas"
    Invoke-ApiInspect -Method "GET" -Endpoint "/tags"
    Invoke-ApiInspect -Method "GET" -Endpoint "/users?limit=3"
    $portfoliosResult = Invoke-ApiInspect -Method "GET" -Endpoint "/portfolios?limit=3"
    
    # GET /portfolios/{slug} - Get portfolio by slug
    if ($portfoliosResult.Content -and $portfoliosResult.Content.data -and $portfoliosResult.Content.data.Count -gt 0) {
        $testSlug = $portfoliosResult.Content.data[0].slug
        Invoke-ApiInspect -Method "GET" -Endpoint "/portfolios/$testSlug"
    }
    
    Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin"
    Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin/followers?limit=3"
    Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin/following?limit=3"
    Invoke-ApiInspect -Method "GET" -Endpoint "/search/users?q=admin"
    Invoke-ApiInspect -Method "GET" -Endpoint "/search/portfolios?q=web"
    
    # ========== AUTHENTICATION ==========
    Write-Header "AUTHENTICATION"
    
    # Login invalid
    Invoke-ApiInspect -Method "POST" -Endpoint "/auth/login" -Body @{
        username = "invalid"
        password = "wrong"
    }
    
    # Login valid
    Login
    
    # ========== AUTHENTICATED ENDPOINTS ==========
    if ($Global:AccessToken) {
        Write-Header "PROFILE (AUTHENTICATED)"
        
        Invoke-ApiInspect -Method "GET" -Endpoint "/me" -UseAuth $true
        Invoke-ApiInspect -Method "GET" -Endpoint "/me/check-username?username=testuser123" -UseAuth $true
        Invoke-ApiInspect -Method "GET" -Endpoint "/me/portfolios?limit=3" -UseAuth $true
        
        Write-Header "FEED"
        Invoke-ApiInspect -Method "GET" -Endpoint "/feed?limit=3" -UseAuth $true
        
        Write-Header "SESSIONS"
        Invoke-ApiInspect -Method "GET" -Endpoint "/auth/sessions" -UseAuth $true
        
        Write-Header "ADMIN ENDPOINTS"
        Invoke-ApiInspect -Method "GET" -Endpoint "/admin/dashboard/stats" -UseAuth $true
        Invoke-ApiInspect -Method "GET" -Endpoint "/admin/users?limit=3" -UseAuth $true
        
        Write-Header "ADMIN JURUSAN CRUD"
        Invoke-ApiInspect -Method "GET" -Endpoint "/admin/jurusan" -UseAuth $true
        $jurusanResult = Invoke-ApiInspect -Method "POST" -Endpoint "/admin/jurusan" -UseAuth $true -Body @{
            nama = "Test Jurusan Inspect"
            kode = "testinsp"
        }
        if ($jurusanResult.Content -and $jurusanResult.Content.data.id) {
            $testJurusanId = $jurusanResult.Content.data.id
            Invoke-ApiInspect -Method "PATCH" -Endpoint "/admin/jurusan/$testJurusanId" -UseAuth $true -Body @{
                nama = "Test Jurusan Updated"
            }
            Invoke-ApiInspect -Method "DELETE" -Endpoint "/admin/jurusan/$testJurusanId" -UseAuth $true
        }
        
        Write-Header "ADMIN TAHUN AJARAN CRUD"
        Invoke-ApiInspect -Method "GET" -Endpoint "/admin/tahun-ajaran" -UseAuth $true
        $tahunResult = Invoke-ApiInspect -Method "POST" -Endpoint "/admin/tahun-ajaran" -UseAuth $true -Body @{
            tahun_mulai = 2098
            is_active = $false
        }
        if ($tahunResult.Content -and $tahunResult.Content.data.id) {
            $testTahunId = $tahunResult.Content.data.id
            Invoke-ApiInspect -Method "PATCH" -Endpoint "/admin/tahun-ajaran/$testTahunId" -UseAuth $true -Body @{
                promotion_month = 8
            }
            Invoke-ApiInspect -Method "DELETE" -Endpoint "/admin/tahun-ajaran/$testTahunId" -UseAuth $true
        }
        
        Write-Header "ADMIN KELAS CRUD"
        Invoke-ApiInspect -Method "GET" -Endpoint "/admin/kelas?limit=5" -UseAuth $true
        
        Write-Header "ADMIN TAGS"
        Invoke-ApiInspect -Method "GET" -Endpoint "/admin/tags" -UseAuth $true
        
        Write-Header "ADMIN PORTFOLIOS"
        Invoke-ApiInspect -Method "GET" -Endpoint "/admin/portfolios?limit=3" -UseAuth $true
        Invoke-ApiInspect -Method "GET" -Endpoint "/admin/portfolios/pending?limit=3" -UseAuth $true
        
        Write-Header "ADMIN PORTFOLIO MODERATION (APPROVE/REJECT)"
        
        # Create portfolio for approve test
        $approveTestResult = Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
            judul = "Test Portfolio for Approve"
        }
        
        if ($approveTestResult.Content -and $approveTestResult.Content.data.id) {
            $approvePortfolioId = $approveTestResult.Content.data.id
            
            # Add thumbnail
            Invoke-ApiInspect -Method "PATCH" -Endpoint "/portfolios/$approvePortfolioId" -UseAuth $true -Body @{
                thumbnail_url = "https://picsum.photos/800/600"
            } | Out-Null
            
            # Add content block
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$approvePortfolioId/blocks" -UseAuth $true -Body @{
                block_type = "text"
                block_order = 0
                payload = @{ content = "<p>Test content</p>" }
            } | Out-Null
            
            # Submit for review
            $submitResult = Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$approvePortfolioId/submit" -UseAuth $true
            
            if ($submitResult.StatusCode -eq 200) {
                # Test APPROVE
                Invoke-ApiInspect -Method "POST" -Endpoint "/admin/portfolios/$approvePortfolioId/approve" -UseAuth $true -Body @{
                    note = "Approved via inspect script"
                }
            }
            
            # Cleanup
            Invoke-ApiInspect -Method "DELETE" -Endpoint "/portfolios/$approvePortfolioId" -UseAuth $true | Out-Null
        }
        
        # Create portfolio for reject test
        $rejectTestResult = Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
            judul = "Test Portfolio for Reject"
        }
        
        if ($rejectTestResult.Content -and $rejectTestResult.Content.data.id) {
            $rejectPortfolioId = $rejectTestResult.Content.data.id
            
            # Add thumbnail
            Invoke-ApiInspect -Method "PATCH" -Endpoint "/portfolios/$rejectPortfolioId" -UseAuth $true -Body @{
                thumbnail_url = "https://picsum.photos/800/600"
            } | Out-Null
            
            # Add content block
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$rejectPortfolioId/blocks" -UseAuth $true -Body @{
                block_type = "text"
                block_order = 0
                payload = @{ content = "<p>Test content</p>" }
            } | Out-Null
            
            # Submit for review
            $submitResult = Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$rejectPortfolioId/submit" -UseAuth $true
            
            if ($submitResult.StatusCode -eq 200) {
                # Test REJECT
                Invoke-ApiInspect -Method "POST" -Endpoint "/admin/portfolios/$rejectPortfolioId/reject" -UseAuth $true -Body @{
                    note = "Rejected via inspect script - testing purposes"
                }
            }
            
            # Cleanup
            Invoke-ApiInspect -Method "DELETE" -Endpoint "/portfolios/$rejectPortfolioId" -UseAuth $true | Out-Null
        }
        
        Write-Header "PORTFOLIO CRUD DEMO"
        
        # Create portfolio
        $createResult = Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
            judul = "Demo Portfolio untuk Inspect"
        }
        
        if ($createResult.Content -and $createResult.Content.data.id) {
            $portfolioId = $createResult.Content.data.id
            
            # Get by ID
            Invoke-ApiInspect -Method "GET" -Endpoint "/portfolios/id/$portfolioId" -UseAuth $true
            
            # Update
            Invoke-ApiInspect -Method "PATCH" -Endpoint "/portfolios/$portfolioId" -UseAuth $true -Body @{
                judul = "Demo Portfolio Updated"
            }
            
            Write-Header "CONTENT BLOCKS - ALL TYPES"
            
            # 1. TEXT block
            $textBlockResult = Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/blocks" -UseAuth $true -Body @{
                block_type = "text"
                block_order = 0
                payload = @{
                    content = "<p>Ini adalah content block text</p>"
                }
            }
            
            # 2. IMAGE block
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/blocks" -UseAuth $true -Body @{
                block_type = "image"
                block_order = 1
                payload = @{
                    url = "https://picsum.photos/800/600"
                    caption = "Screenshot aplikasi"
                }
            }
            
            # 3. TABLE block
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/blocks" -UseAuth $true -Body @{
                block_type = "table"
                block_order = 2
                payload = @{
                    headers = @("Fitur", "Deskripsi")
                    rows = @(@("Login", "Autentikasi user"), @("Dashboard", "Halaman utama"))
                }
            }
            
            # 4. YOUTUBE block
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/blocks" -UseAuth $true -Body @{
                block_type = "youtube"
                block_order = 3
                payload = @{
                    video_id = "dQw4w9WgXcQ"
                    title = "Demo Video"
                }
            }
            
            # 5. BUTTON block
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/blocks" -UseAuth $true -Body @{
                block_type = "button"
                block_order = 4
                payload = @{
                    text = "Lihat Demo"
                    url = "https://demo.example.com"
                }
            }
            
            # 6. EMBED block
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/blocks" -UseAuth $true -Body @{
                block_type = "embed"
                block_order = 5
                payload = @{
                    html = "<iframe src=`"https://codepen.io/pen`"></iframe>"
                    title = "CodePen Demo"
                }
            }
            
            # Update text block and test reorder
            if ($textBlockResult.Content -and $textBlockResult.Content.data.id) {
                $blockId = $textBlockResult.Content.data.id
                Invoke-ApiInspect -Method "PATCH" -Endpoint "/portfolios/$portfolioId/blocks/$blockId" -UseAuth $true -Body @{
                    payload = @{
                        content = "<p>Content block updated</p>"
                    }
                }
                
                # Reorder blocks (swap order of first two blocks)
                Invoke-ApiInspect -Method "PUT" -Endpoint "/portfolios/$portfolioId/blocks/reorder" -UseAuth $true -Body @{
                    block_orders = @(
                        @{ id = $blockId; order = 1 }
                    )
                }
                
                Invoke-ApiInspect -Method "DELETE" -Endpoint "/portfolios/$portfolioId/blocks/$blockId" -UseAuth $true
            }
            
            Write-Header "PORTFOLIO STATUS ENDPOINTS"
            
            # Submit (will fail - no thumbnail)
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/submit" -UseAuth $true
            
            # Archive
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/archive" -UseAuth $true
            
            # Unarchive
            Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioId/unarchive" -UseAuth $true
            
            # Delete portfolio
            Invoke-ApiInspect -Method "DELETE" -Endpoint "/portfolios/$portfolioId" -UseAuth $true
        }
        
        Write-Header "LOGOUT (ADMIN)"
        Invoke-ApiInspect -Method "POST" -Endpoint "/auth/logout" -UseAuth $true
    }
    
    # ========== STUDENT USER ENDPOINTS ==========
    Write-Header "STUDENT USER ENDPOINTS"
    
    # Get a student username
    $studentUsername = $null
    $usersResult = Invoke-ApiInspect -Method "GET" -Endpoint "/users?role=student&limit=3"
    if ($usersResult.Content -and $usersResult.Content.data -and $usersResult.Content.data.Count -gt 0) {
        $studentUsername = $usersResult.Content.data[0].username
    }
    
    if ($studentUsername) {
        Write-Log ""
        Write-Log "  [Found student user: $studentUsername]" -Color Yellow
        
        # Login as student
        $studentLoginResult = Invoke-ApiInspect -Method "POST" -Endpoint "/auth/login" -Body @{
            username = $studentUsername
            password = "password"
        }
        
        if ($studentLoginResult.Content -and $studentLoginResult.Content.data.access_token) {
            $Global:AccessToken = $studentLoginResult.Content.data.access_token
            Write-Log ""
            Write-Log "  [Logged in as student, token saved]" -Color Green
            
            Write-Header "STUDENT PROFILE"
            Invoke-ApiInspect -Method "GET" -Endpoint "/me" -UseAuth $true
            
            Write-Header "STUDENT USER ENDPOINTS"
            Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/users/$studentUsername" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin/followers?limit=3" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin/following?limit=3" -UseAuth $true
            
            Write-Header "STUDENT PORTFOLIOS"
            Invoke-ApiInspect -Method "GET" -Endpoint "/me/portfolios?limit=3" -UseAuth $true
            
            Write-Header "STUDENT FEED"
            Invoke-ApiInspect -Method "GET" -Endpoint "/feed?limit=3" -UseAuth $true
            
            Write-Header "STUDENT ADMIN ACCESS (SHOULD FAIL)"
            Invoke-ApiInspect -Method "GET" -Endpoint "/admin/dashboard/stats" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/admin/users?limit=1" -UseAuth $true
            
            Write-Header "STUDENT SOCIAL (FOLLOW/LIKE)"
            Invoke-ApiInspect -Method "POST" -Endpoint "/users/admin/follow" -UseAuth $true
            Invoke-ApiInspect -Method "DELETE" -Endpoint "/users/admin/follow" -UseAuth $true
            
            # Like a portfolio
            $portfolioResult = Invoke-ApiInspect -Method "GET" -Endpoint "/portfolios?limit=1"
            if ($portfolioResult.Content -and $portfolioResult.Content.data -and $portfolioResult.Content.data.Count -gt 0) {
                $portfolioToLike = $portfolioResult.Content.data[0].id
                Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioToLike/like" -UseAuth $true
                Invoke-ApiInspect -Method "DELETE" -Endpoint "/portfolios/$portfolioToLike/like" -UseAuth $true
            }
            
            Write-Header "STUDENT SEARCH"
            Invoke-ApiInspect -Method "GET" -Endpoint "/search/users?q=admin" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/search/portfolios?q=web" -UseAuth $true
            
            Write-Header "STUDENT PORTFOLIO CRUD"
            # Create portfolio as student
            $studentPortfolioResult = Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
                judul = "Student Demo Portfolio"
            }
            
            if ($studentPortfolioResult.Content -and $studentPortfolioResult.Content.data.id) {
                $studentPortfolioId = $studentPortfolioResult.Content.data.id
                
                # Update portfolio
                Invoke-ApiInspect -Method "PATCH" -Endpoint "/portfolios/$studentPortfolioId" -UseAuth $true -Body @{
                    judul = "Student Demo Portfolio Updated"
                }
                
                # Delete portfolio
                Invoke-ApiInspect -Method "DELETE" -Endpoint "/portfolios/$studentPortfolioId" -UseAuth $true
            }
            
            Write-Header "LOGOUT (STUDENT)"
            Invoke-ApiInspect -Method "POST" -Endpoint "/auth/logout" -UseAuth $true
        }
    } else {
        Write-Log ""
        Write-Log "  [No student users found, skipping student tests]" -Color Yellow
    }
    
    # ========== ALUMNI USER ENDPOINTS ==========
    Write-Header "ALUMNI USER ENDPOINTS"
    
    # Get an alumni username
    $alumniUsername = $null
    $alumniResult = Invoke-ApiInspect -Method "GET" -Endpoint "/users?role=alumni&limit=3"
    if ($alumniResult.Content -and $alumniResult.Content.data -and $alumniResult.Content.data.Count -gt 0) {
        $alumniUsername = $alumniResult.Content.data[0].username
    }
    
    if ($alumniUsername) {
        Write-Log ""
        Write-Log "  [Found alumni user: $alumniUsername]" -Color Yellow
        
        # Login as alumni
        $alumniLoginResult = Invoke-ApiInspect -Method "POST" -Endpoint "/auth/login" -Body @{
            username = $alumniUsername
            password = "password"
        }
        
        if ($alumniLoginResult.Content -and $alumniLoginResult.Content.data.access_token) {
            $Global:AccessToken = $alumniLoginResult.Content.data.access_token
            Write-Log ""
            Write-Log "  [Logged in as alumni, token saved]" -Color Green
            
            Write-Header "ALUMNI PROFILE"
            Invoke-ApiInspect -Method "GET" -Endpoint "/me" -UseAuth $true
            
            Write-Header "ALUMNI USER ENDPOINTS"
            Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/users/$alumniUsername" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin/followers?limit=3" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/users/admin/following?limit=3" -UseAuth $true
            
            Write-Header "ALUMNI ADMIN ACCESS (SHOULD FAIL)"
            Invoke-ApiInspect -Method "GET" -Endpoint "/admin/dashboard/stats" -UseAuth $true
            
            Write-Header "ALUMNI SOCIAL (FOLLOW/LIKE)"
            Invoke-ApiInspect -Method "POST" -Endpoint "/users/admin/follow" -UseAuth $true
            Invoke-ApiInspect -Method "DELETE" -Endpoint "/users/admin/follow" -UseAuth $true
            
            # Like a portfolio
            $portfolioResult = Invoke-ApiInspect -Method "GET" -Endpoint "/portfolios?limit=1"
            if ($portfolioResult.Content -and $portfolioResult.Content.data -and $portfolioResult.Content.data.Count -gt 0) {
                $portfolioToLike = $portfolioResult.Content.data[0].id
                Invoke-ApiInspect -Method "POST" -Endpoint "/portfolios/$portfolioToLike/like" -UseAuth $true
                Invoke-ApiInspect -Method "DELETE" -Endpoint "/portfolios/$portfolioToLike/like" -UseAuth $true
            }
            
            Write-Header "ALUMNI SEARCH"
            Invoke-ApiInspect -Method "GET" -Endpoint "/search/users?q=admin" -UseAuth $true
            Invoke-ApiInspect -Method "GET" -Endpoint "/search/portfolios?q=web" -UseAuth $true
            
            Write-Header "LOGOUT (ALUMNI)"
            Invoke-ApiInspect -Method "POST" -Endpoint "/auth/logout" -UseAuth $true
        }
    } else {
        Write-Log ""
        Write-Log "  [No alumni users found, skipping alumni tests]" -Color Yellow
    }
    
    Write-Log ""
    Write-Log ("=" * 80) -Color Cyan
    Write-Log "  INSPECTION COMPLETE" -Color Cyan
    Write-Log ("=" * 80) -Color Cyan
}

function Inspect-SingleEndpoint {
    param([string]$Path)
    
    Write-Header "SINGLE ENDPOINT INSPECTION"
    
    # Try to login first for authenticated endpoints
    $loginResult = Invoke-ApiInspect -Method "POST" -Endpoint "/auth/login" -Body @{
        username = "admin"
        password = "password"
    }
    
    if ($loginResult.Content -and $loginResult.Content.data.access_token) {
        $Global:AccessToken = $loginResult.Content.data.access_token
    }
    
    # Inspect the endpoint
    Invoke-ApiInspect -Method "GET" -Endpoint $Path -UseAuth ($Global:AccessToken -ne "")
}

# Main
if ($All -or $Endpoint -eq "") {
    Inspect-AllEndpoints
} else {
    Inspect-SingleEndpoint -Path $Endpoint
}

# Save output to log file
$mdContent = @"
# Grafikarsa API Inspection Log

Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')

``````
$($Global:OutputLines -join "`n")
``````
"@

$mdContent | Out-File -FilePath $LogFile -Encoding UTF8
Write-Host ""
Write-Host "Log saved to: $LogFile" -ForegroundColor Green

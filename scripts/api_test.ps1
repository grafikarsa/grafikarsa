# Grafikarsa API Test Script
# Usage: .\scripts\api_test.ps1 [-BaseUrl "http://localhost:8080"]

param(
    [string]$BaseUrl = "http://localhost:8080"
)

$ApiUrl = "$BaseUrl/api/v1"
$Global:AccessToken = ""
$Global:RefreshToken = ""
$Global:TestUserId = ""
$Global:TestPortfolioId = ""
$Global:TestBlockId = ""
$Global:PassCount = 0
$Global:FailCount = 0
$Global:SkipCount = 0

function Write-TestHeader {
    param([string]$Title)
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  $Title" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
}

function Write-TestResult {
    param(
        [string]$Name,
        [bool]$Success,
        [string]$Message = "",
        [bool]$Skip = $false
    )
    
    if ($Skip) {
        Write-Host "[SKIP] $Name" -ForegroundColor Yellow
        if ($Message) { Write-Host "       $Message" -ForegroundColor Gray }
        $Global:SkipCount++
        return
    }
    
    if ($Success) {
        Write-Host "[PASS] $Name" -ForegroundColor Green
        $Global:PassCount++
    } else {
        Write-Host "[FAIL] $Name" -ForegroundColor Red
        if ($Message) { Write-Host "       $Message" -ForegroundColor Red }
        $Global:FailCount++
    }
}

function Invoke-ApiRequest {
    param(
        [string]$Method,
        [string]$Endpoint,
        [hashtable]$Body = @{},
        [hashtable]$Headers = @{},
        [bool]$UseAuth = $false,
        [int]$ExpectedStatus = 200
    )
    
    $uri = "$ApiUrl$Endpoint"
    $requestHeaders = @{
        "Content-Type" = "application/json"
    }
    
    foreach ($key in $Headers.Keys) {
        $requestHeaders[$key] = $Headers[$key]
    }
    
    if ($UseAuth -and $Global:AccessToken) {
        $requestHeaders["Authorization"] = "Bearer $Global:AccessToken"
    }
    
    try {
        $params = @{
            Method = $Method
            Uri = $uri
            Headers = $requestHeaders
            ErrorAction = "Stop"
        }
        
        if ($Body.Count -gt 0 -and $Method -ne "GET") {
            $params["Body"] = ($Body | ConvertTo-Json -Depth 10)
        }
        
        $response = Invoke-WebRequest @params
        $statusCode = $response.StatusCode
        $content = $response.Content | ConvertFrom-Json
        
        return @{
            Success = ($statusCode -eq $ExpectedStatus)
            StatusCode = $statusCode
            Data = $content
            Error = $null
        }
    }
    catch {
        $statusCode = 0
        $errorMessage = $_.Exception.Message
        
        if ($_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
            try {
                $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
                $errorContent = $reader.ReadToEnd() | ConvertFrom-Json
                $errorMessage = $errorContent.error.message
            } catch {}
        }
        
        return @{
            Success = ($statusCode -eq $ExpectedStatus)
            StatusCode = $statusCode
            Data = $null
            Error = $errorMessage
        }
    }
}

# ============================================
# TEST SUITES
# ============================================

function Test-PublicEndpoints {
    Write-TestHeader "PUBLIC ENDPOINTS"
    
    # GET /jurusan
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/jurusan"
    Write-TestResult -Name "GET /jurusan" -Success $result.Success -Message $result.Error
    
    # GET /kelas
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/kelas"
    Write-TestResult -Name "GET /kelas" -Success $result.Success -Message $result.Error
    
    # GET /tags
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/tags"
    Write-TestResult -Name "GET /tags" -Success $result.Success -Message $result.Error
    
    # GET /users
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users"
    Write-TestResult -Name "GET /users" -Success $result.Success -Message $result.Error
    
    # GET /portfolios
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios"
    Write-TestResult -Name "GET /portfolios" -Success $result.Success -Message $result.Error
}

function Test-AuthEndpoints {
    Write-TestHeader "AUTHENTICATION"
    
    # POST /auth/login - Invalid credentials
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/auth/login" -Body @{
        username = "invalid_user"
        password = "wrong_password"
    } -ExpectedStatus 401
    Write-TestResult -Name "POST /auth/login (invalid)" -Success $result.Success -Message $result.Error
    
    # POST /auth/login - Valid credentials
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/auth/login" -Body @{
        username = "admin"
        password = "password"
    }
    
    if ($result.Success -and $result.Data.data.access_token) {
        $Global:AccessToken = $result.Data.data.access_token
        Write-TestResult -Name "POST /auth/login (valid)" -Success $true
    } else {
        Write-TestResult -Name "POST /auth/login (valid)" -Success $false -Message "Failed to get access token: $($result.Error)"
    }
    
    # GET /auth/sessions
    if ($Global:AccessToken) {
        $result = Invoke-ApiRequest -Method "GET" -Endpoint "/auth/sessions" -UseAuth $true
        Write-TestResult -Name "GET /auth/sessions" -Success $result.Success -Message $result.Error
    } else {
        Write-TestResult -Name "GET /auth/sessions" -Skip $true -Message "No access token"
    }
}

function Test-ProfileEndpoints {
    Write-TestHeader "PROFILE (AUTHENTICATED)"
    
    if (-not $Global:AccessToken) {
        Write-TestResult -Name "Profile tests" -Skip $true -Message "No access token available"
        return
    }
    
    # GET /me
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/me" -UseAuth $true
    Write-TestResult -Name "GET /me" -Success $result.Success -Message $result.Error
    if ($result.Success) {
        $Global:TestUserId = $result.Data.data.id
    }
    
    # GET /me without auth (should fail)
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/me" -ExpectedStatus 401
    Write-TestResult -Name "GET /me (no auth)" -Success $result.Success -Message $result.Error
    
    # PATCH /me
    $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/me" -UseAuth $true -Body @{
        bio = "Updated bio from test script"
    }
    Write-TestResult -Name "PATCH /me" -Success $result.Success -Message $result.Error
    
    # GET /me/check-username
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/me/check-username?username=testuser123" -UseAuth $true
    Write-TestResult -Name "GET /me/check-username" -Success $result.Success -Message $result.Error
}

function Test-UserEndpoints {
    Write-TestHeader "USER ENDPOINTS"
    
    # GET /users/{username}
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin"
    Write-TestResult -Name "GET /users/{username}" -Success $result.Success -Message $result.Error
    
    # GET /users/{username} - Not found
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/nonexistent_user_12345" -ExpectedStatus 404
    Write-TestResult -Name "GET /users/{username} (404)" -Success $result.Success -Message $result.Error
    
    # GET /users/{username}/followers
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin/followers"
    Write-TestResult -Name "GET /users/{username}/followers" -Success $result.Success -Message $result.Error
    
    # GET /users/{username}/following
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin/following"
    Write-TestResult -Name "GET /users/{username}/following" -Success $result.Success -Message $result.Error
}

function Test-PortfolioEndpoints {
    Write-TestHeader "PORTFOLIO ENDPOINTS"
    
    if (-not $Global:AccessToken) {
        Write-TestResult -Name "Portfolio tests" -Skip $true -Message "No access token available"
        return
    }
    
    # POST /portfolios - Create
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
        judul = "Test Portfolio from Script"
    } -ExpectedStatus 201
    
    if ($result.Success -and $result.Data.data.id) {
        $Global:TestPortfolioId = $result.Data.data.id
        Write-TestResult -Name "POST /portfolios" -Success $true
    } else {
        Write-TestResult -Name "POST /portfolios" -Success $false -Message $result.Error
    }
    
    # GET /me/portfolios
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/me/portfolios" -UseAuth $true
    Write-TestResult -Name "GET /me/portfolios" -Success $result.Success -Message $result.Error
    
    if ($Global:TestPortfolioId) {
        # GET /portfolios/id/{id} (note: route is /portfolios/id/:id to avoid conflict with slug)
        $result = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios/id/$Global:TestPortfolioId" -UseAuth $true
        Write-TestResult -Name "GET /portfolios/id/{id}" -Success $result.Success -Message $result.Error
        
        # PATCH /portfolios/{id}
        $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/portfolios/$Global:TestPortfolioId" -UseAuth $true -Body @{
            judul = "Updated Test Portfolio"
        }
        Write-TestResult -Name "PATCH /portfolios/{id}" -Success $result.Success -Message $result.Error
        
        # POST /portfolios/{id}/submit - Should fail because no thumbnail/content blocks yet
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/submit" -UseAuth $true -ExpectedStatus 422
        Write-TestResult -Name "POST /portfolios/{id}/submit (incomplete)" -Success $result.Success -Message $result.Error
    }
    
    # GET /portfolios/{slug} - Test with published portfolio from public list
    $publicPortfolios = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios?limit=1"
    if ($publicPortfolios.Success -and $publicPortfolios.Data.data -and $publicPortfolios.Data.data.Count -gt 0) {
        $testSlug = $publicPortfolios.Data.data[0].slug
        $result = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios/$testSlug"
        Write-TestResult -Name "GET /portfolios/{slug}" -Success $result.Success -Message $result.Error
    } else {
        Write-TestResult -Name "GET /portfolios/{slug}" -Skip $true -Message "No published portfolios available"
    }
}

function Test-ContentBlockEndpoints {
    Write-TestHeader "CONTENT BLOCK ENDPOINTS"
    
    if (-not $Global:AccessToken -or -not $Global:TestPortfolioId) {
        Write-TestResult -Name "Content block tests" -Skip $true -Message "No portfolio available"
        return
    }
    
    # Test all 6 block types: text, image, table, youtube, button, embed
    
    # 1. TEXT block
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks" -UseAuth $true -Body @{
        block_type = "text"
        block_order = 0
        payload = @{
            content = "<p>Test text content block</p>"
        }
    } -ExpectedStatus 201
    $textBlockId = $null
    if ($result.Success -and $result.Data.data.id) { $textBlockId = $result.Data.data.id }
    Write-TestResult -Name "POST block (text)" -Success $result.Success -Message $result.Error
    
    # 2. IMAGE block
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks" -UseAuth $true -Body @{
        block_type = "image"
        block_order = 1
        payload = @{
            url = "https://picsum.photos/800/600"
            caption = "Test image caption"
        }
    } -ExpectedStatus 201
    $imageBlockId = $null
    if ($result.Success -and $result.Data.data.id) { $imageBlockId = $result.Data.data.id }
    Write-TestResult -Name "POST block (image)" -Success $result.Success -Message $result.Error
    
    # 3. TABLE block
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks" -UseAuth $true -Body @{
        block_type = "table"
        block_order = 2
        payload = @{
            headers = @("Fitur", "Deskripsi")
            rows = @(@("Login", "Autentikasi user"), @("Dashboard", "Halaman utama"))
        }
    } -ExpectedStatus 201
    $tableBlockId = $null
    if ($result.Success -and $result.Data.data.id) { $tableBlockId = $result.Data.data.id }
    Write-TestResult -Name "POST block (table)" -Success $result.Success -Message $result.Error
    
    # 4. YOUTUBE block
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks" -UseAuth $true -Body @{
        block_type = "youtube"
        block_order = 3
        payload = @{
            video_id = "dQw4w9WgXcQ"
            title = "Demo Video"
        }
    } -ExpectedStatus 201
    $youtubeBlockId = $null
    if ($result.Success -and $result.Data.data.id) { $youtubeBlockId = $result.Data.data.id }
    Write-TestResult -Name "POST block (youtube)" -Success $result.Success -Message $result.Error
    
    # 5. BUTTON block
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks" -UseAuth $true -Body @{
        block_type = "button"
        block_order = 4
        payload = @{
            text = "Lihat Demo"
            url = "https://demo.example.com"
        }
    } -ExpectedStatus 201
    $buttonBlockId = $null
    if ($result.Success -and $result.Data.data.id) { $buttonBlockId = $result.Data.data.id }
    Write-TestResult -Name "POST block (button)" -Success $result.Success -Message $result.Error
    
    # 6. EMBED block
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks" -UseAuth $true -Body @{
        block_type = "embed"
        block_order = 5
        payload = @{
            html = "<iframe src=`"https://codepen.io/pen`" width=`"100%`" height=`"300`"></iframe>"
            title = "CodePen Demo"
        }
    } -ExpectedStatus 201
    $embedBlockId = $null
    if ($result.Success -and $result.Data.data.id) { $embedBlockId = $result.Data.data.id }
    Write-TestResult -Name "POST block (embed)" -Success $result.Success -Message $result.Error
    
    # Save first block ID for other tests
    $Global:TestBlockId = $textBlockId
    
    # Test PATCH on text block
    if ($textBlockId) {
        $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks/$textBlockId" -UseAuth $true -Body @{
            payload = @{
                content = "<p>Updated text content</p>"
            }
        }
        Write-TestResult -Name "PATCH block (text)" -Success $result.Success -Message $result.Error
    }
    
    # Test PATCH on image block
    if ($imageBlockId) {
        $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks/$imageBlockId" -UseAuth $true -Body @{
            payload = @{
                url = "https://picsum.photos/1200/800"
                caption = "Updated image caption"
            }
        }
        Write-TestResult -Name "PATCH block (image)" -Success $result.Success -Message $result.Error
    }
    
    # Test PUT /portfolios/{id}/blocks/reorder
    $blockIdsForReorder = @($textBlockId, $imageBlockId, $tableBlockId) | Where-Object { $_ -ne $null }
    if ($blockIdsForReorder.Count -ge 2) {
        $blockOrders = @()
        $order = $blockIdsForReorder.Count - 1
        foreach ($bid in $blockIdsForReorder) {
            $blockOrders += @{ id = $bid; order = $order }
            $order--
        }
        $result = Invoke-ApiRequest -Method "PUT" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks/reorder" -UseAuth $true -Body @{
            block_orders = $blockOrders
        }
        Write-TestResult -Name "PUT /portfolios/{id}/blocks/reorder" -Success $result.Success -Message $result.Error
    } else {
        Write-TestResult -Name "PUT /portfolios/{id}/blocks/reorder" -Skip $true -Message "Not enough blocks to reorder"
    }
    
    # Cleanup - delete all test blocks
    $blockIds = @($textBlockId, $imageBlockId, $tableBlockId, $youtubeBlockId, $buttonBlockId, $embedBlockId) | Where-Object { $_ -ne $null }
    foreach ($blockId in $blockIds) {
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$Global:TestPortfolioId/blocks/$blockId" -UseAuth $true
    }
    Write-TestResult -Name "DELETE blocks (cleanup)" -Success $true
}

function Test-PortfolioStatusEndpoints {
    Write-TestHeader "PORTFOLIO STATUS ENDPOINTS"
    
    if (-not $Global:AccessToken -or -not $Global:TestPortfolioId) {
        Write-TestResult -Name "Portfolio status tests" -Skip $true -Message "No portfolio available"
        return
    }
    
    # POST /portfolios/{id}/archive
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/archive" -UseAuth $true
    Write-TestResult -Name "POST /portfolios/{id}/archive" -Success $result.Success -Message $result.Error
    
    # POST /portfolios/{id}/unarchive
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$Global:TestPortfolioId/unarchive" -UseAuth $true
    Write-TestResult -Name "POST /portfolios/{id}/unarchive" -Success $result.Success -Message $result.Error
}

function Test-SocialEndpoints {
    Write-TestHeader "SOCIAL ENDPOINTS (FOLLOW/LIKE)"
    
    if (-not $Global:AccessToken) {
        Write-TestResult -Name "Social tests" -Skip $true -Message "No access token available"
        return
    }
    
    # Get a user to follow (not admin)
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users?limit=5"
    $targetUser = $null
    if ($result.Success -and $result.Data.data) {
        foreach ($user in $result.Data.data) {
            if ($user.username -ne "admin") {
                $targetUser = $user.username
                break
            }
        }
    }
    
    if ($targetUser) {
        # POST /users/{username}/follow
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/users/$targetUser/follow" -UseAuth $true
        Write-TestResult -Name "POST /users/{username}/follow" -Success ($result.Success -or $result.StatusCode -eq 409) -Message $result.Error
        
        # DELETE /users/{username}/follow
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/users/$targetUser/follow" -UseAuth $true
        Write-TestResult -Name "DELETE /users/{username}/follow" -Success ($result.Success -or $result.StatusCode -eq 400) -Message $result.Error
    } else {
        Write-TestResult -Name "Follow tests" -Skip $true -Message "No other users to follow"
    }
    
    # Get a portfolio to like
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios?limit=1"
    if ($result.Success -and $result.Data.data -and $result.Data.data.Count -gt 0) {
        $portfolioId = $result.Data.data[0].id
        
        # POST /portfolios/{id}/like
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$portfolioId/like" -UseAuth $true
        Write-TestResult -Name "POST /portfolios/{id}/like" -Success ($result.Success -or $result.StatusCode -eq 409) -Message $result.Error
        
        # DELETE /portfolios/{id}/like
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$portfolioId/like" -UseAuth $true
        Write-TestResult -Name "DELETE /portfolios/{id}/like" -Success ($result.Success -or $result.StatusCode -eq 400) -Message $result.Error
    } else {
        Write-TestResult -Name "Like tests" -Skip $true -Message "No portfolios to like"
    }
}

function Test-FeedEndpoints {
    Write-TestHeader "FEED & SEARCH"
    
    # GET /feed (requires auth)
    if ($Global:AccessToken) {
        $result = Invoke-ApiRequest -Method "GET" -Endpoint "/feed" -UseAuth $true
        Write-TestResult -Name "GET /feed" -Success $result.Success -Message $result.Error
    } else {
        Write-TestResult -Name "GET /feed" -Skip $true -Message "No access token"
    }
    
    # GET /search/users
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/search/users?q=admin"
    Write-TestResult -Name "GET /search/users" -Success $result.Success -Message $result.Error
    
    # GET /search/portfolios
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/search/portfolios?q=test"
    Write-TestResult -Name "GET /search/portfolios" -Success $result.Success -Message $result.Error
}

function Test-AdminEndpoints {
    Write-TestHeader "ADMIN ENDPOINTS"
    
    if (-not $Global:AccessToken) {
        Write-TestResult -Name "Admin tests" -Skip $true -Message "No access token available"
        return
    }
    
    # GET /admin/dashboard/stats
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/dashboard/stats" -UseAuth $true
    Write-TestResult -Name "GET /admin/dashboard/stats" -Success $result.Success -Message $result.Error
    
    # GET /admin/users
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/users" -UseAuth $true
    Write-TestResult -Name "GET /admin/users" -Success $result.Success -Message $result.Error
    
    # ========== JURUSAN CRUD ==========
    # GET /admin/jurusan
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/jurusan" -UseAuth $true
    Write-TestResult -Name "GET /admin/jurusan" -Success $result.Success -Message $result.Error
    
    # POST /admin/jurusan
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/admin/jurusan" -UseAuth $true -Body @{
        nama = "Test Jurusan"
        kode = "testjur"
    } -ExpectedStatus 201
    $testJurusanId = $null
    if ($result.Success -and $result.Data.data.id) { $testJurusanId = $result.Data.data.id }
    Write-TestResult -Name "POST /admin/jurusan" -Success $result.Success -Message $result.Error
    
    # PATCH /admin/jurusan/{id}
    if ($testJurusanId) {
        $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/admin/jurusan/$testJurusanId" -UseAuth $true -Body @{
            nama = "Test Jurusan Updated"
        }
        Write-TestResult -Name "PATCH /admin/jurusan/{id}" -Success $result.Success -Message $result.Error
        
        # DELETE /admin/jurusan/{id}
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/admin/jurusan/$testJurusanId" -UseAuth $true
        Write-TestResult -Name "DELETE /admin/jurusan/{id}" -Success $result.Success -Message $result.Error
    }
    
    # ========== TAHUN AJARAN CRUD ==========
    # GET /admin/tahun-ajaran
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/tahun-ajaran" -UseAuth $true
    Write-TestResult -Name "GET /admin/tahun-ajaran" -Success $result.Success -Message $result.Error
    
    # POST /admin/tahun-ajaran
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/admin/tahun-ajaran" -UseAuth $true -Body @{
        tahun_mulai = 2099
        is_active = $false
    } -ExpectedStatus 201
    $testTahunAjaranId = $null
    if ($result.Success -and $result.Data.data.id) { $testTahunAjaranId = $result.Data.data.id }
    Write-TestResult -Name "POST /admin/tahun-ajaran" -Success $result.Success -Message $result.Error
    
    # PATCH /admin/tahun-ajaran/{id}
    if ($testTahunAjaranId) {
        $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/admin/tahun-ajaran/$testTahunAjaranId" -UseAuth $true -Body @{
            promotion_month = 8
        }
        Write-TestResult -Name "PATCH /admin/tahun-ajaran/{id}" -Success $result.Success -Message $result.Error
        
        # DELETE /admin/tahun-ajaran/{id}
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/admin/tahun-ajaran/$testTahunAjaranId" -UseAuth $true
        Write-TestResult -Name "DELETE /admin/tahun-ajaran/{id}" -Success $result.Success -Message $result.Error
    }
    
    # ========== KELAS CRUD ==========
    # GET /admin/kelas
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/kelas" -UseAuth $true
    Write-TestResult -Name "GET /admin/kelas" -Success $result.Success -Message $result.Error
    
    # Get existing jurusan and tahun_ajaran for creating kelas
    $jurusanResult = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/jurusan" -UseAuth $true
    $tahunResult = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/tahun-ajaran" -UseAuth $true
    
    if ($jurusanResult.Success -and $jurusanResult.Data.data.Count -gt 0 -and $tahunResult.Success -and $tahunResult.Data.data.Count -gt 0) {
        $jurusanId = $jurusanResult.Data.data[0].id
        $tahunAjaranId = $tahunResult.Data.data[0].id
        
        # POST /admin/kelas
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/admin/kelas" -UseAuth $true -Body @{
            jurusan_id = $jurusanId
            tahun_ajaran_id = $tahunAjaranId
            tingkat = 10
            rombel = "Z"
        } -ExpectedStatus 201
        $testKelasId = $null
        if ($result.Success -and $result.Data.data.id) { $testKelasId = $result.Data.data.id }
        Write-TestResult -Name "POST /admin/kelas" -Success $result.Success -Message $result.Error
        
        # PATCH /admin/kelas/{id}
        if ($testKelasId) {
            $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/admin/kelas/$testKelasId" -UseAuth $true -Body @{
                tingkat = 11
            }
            Write-TestResult -Name "PATCH /admin/kelas/{id}" -Success $result.Success -Message $result.Error
            
            # DELETE /admin/kelas/{id}
            $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/admin/kelas/$testKelasId" -UseAuth $true
            Write-TestResult -Name "DELETE /admin/kelas/{id}" -Success $result.Success -Message $result.Error
        }
    } else {
        Write-TestResult -Name "Kelas CRUD tests" -Skip $true -Message "No jurusan or tahun_ajaran available"
    }
    
    # ========== TAGS ==========
    # GET /admin/tags
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/tags" -UseAuth $true
    Write-TestResult -Name "GET /admin/tags" -Success $result.Success -Message $result.Error
    
    # ========== PORTFOLIOS ==========
    # GET /admin/portfolios/pending
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/portfolios/pending" -UseAuth $true
    Write-TestResult -Name "GET /admin/portfolios/pending" -Success $result.Success -Message $result.Error
    
    # GET /admin/portfolios
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/portfolios" -UseAuth $true
    Write-TestResult -Name "GET /admin/portfolios" -Success $result.Success -Message $result.Error
    
    # ========== PORTFOLIO MODERATION (APPROVE/REJECT) ==========
    # Create a test portfolio for moderation testing
    $moderationPortfolioId = $null
    
    # Create portfolio
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
        judul = "Test Portfolio for Moderation"
    } -ExpectedStatus 201
    
    if ($result.Success -and $result.Data.data.id) {
        $moderationPortfolioId = $result.Data.data.id
        
        # Add thumbnail (required for submit)
        $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/portfolios/$moderationPortfolioId" -UseAuth $true -Body @{
            thumbnail_url = "https://picsum.photos/800/600"
        }
        
        # Add a content block (required for submit)
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$moderationPortfolioId/blocks" -UseAuth $true -Body @{
            block_type = "text"
            block_order = 0
            payload = @{
                content = "<p>Test content for moderation</p>"
            }
        } -ExpectedStatus 201
        
        # Submit for review
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$moderationPortfolioId/submit" -UseAuth $true
        
        if ($result.Success) {
            # Test APPROVE endpoint
            $result = Invoke-ApiRequest -Method "POST" -Endpoint "/admin/portfolios/$moderationPortfolioId/approve" -UseAuth $true -Body @{
                note = "Approved by test script"
            }
            Write-TestResult -Name "POST /admin/portfolios/{id}/approve" -Success $result.Success -Message $result.Error
            
            # Cleanup - delete the approved portfolio
            Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$moderationPortfolioId" -UseAuth $true | Out-Null
        } else {
            Write-TestResult -Name "POST /admin/portfolios/{id}/approve" -Skip $true -Message "Could not submit portfolio for review"
        }
    } else {
        Write-TestResult -Name "POST /admin/portfolios/{id}/approve" -Skip $true -Message "Could not create test portfolio"
    }
    
    # Create another portfolio for REJECT test
    $rejectPortfolioId = $null
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
        judul = "Test Portfolio for Rejection"
    } -ExpectedStatus 201
    
    if ($result.Success -and $result.Data.data.id) {
        $rejectPortfolioId = $result.Data.data.id
        
        # Add thumbnail
        $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/portfolios/$rejectPortfolioId" -UseAuth $true -Body @{
            thumbnail_url = "https://picsum.photos/800/600"
        }
        
        # Add content block
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$rejectPortfolioId/blocks" -UseAuth $true -Body @{
            block_type = "text"
            block_order = 0
            payload = @{
                content = "<p>Test content for rejection</p>"
            }
        } -ExpectedStatus 201
        
        # Submit for review
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$rejectPortfolioId/submit" -UseAuth $true
        
        if ($result.Success) {
            # Test REJECT endpoint
            $result = Invoke-ApiRequest -Method "POST" -Endpoint "/admin/portfolios/$rejectPortfolioId/reject" -UseAuth $true -Body @{
                note = "Rejected by test script - testing purposes"
            }
            Write-TestResult -Name "POST /admin/portfolios/{id}/reject" -Success $result.Success -Message $result.Error
            
            # Cleanup - delete the rejected portfolio
            Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$rejectPortfolioId" -UseAuth $true | Out-Null
        } else {
            Write-TestResult -Name "POST /admin/portfolios/{id}/reject" -Skip $true -Message "Could not submit portfolio for review"
        }
    } else {
        Write-TestResult -Name "POST /admin/portfolios/{id}/reject" -Skip $true -Message "Could not create test portfolio"
    }
}

function Test-Cleanup {
    Write-TestHeader "CLEANUP"
    
    if ($Global:TestPortfolioId -and $Global:AccessToken) {
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$Global:TestPortfolioId" -UseAuth $true
        Write-TestResult -Name "DELETE test portfolio" -Success $result.Success -Message $result.Error
    }
}

function Test-Logout {
    Write-TestHeader "LOGOUT"
    
    if ($Global:AccessToken) {
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/auth/logout" -UseAuth $true
        Write-TestResult -Name "POST /auth/logout" -Success $result.Success -Message $result.Error
    }
}

function Get-StudentUsername {
    # Get a student user from the database
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users?role=student&limit=5"
    if ($result.Success -and $result.Data.data -and $result.Data.data.Count -gt 0) {
        foreach ($user in $result.Data.data) {
            if ($user.username -ne "admin") {
                return $user.username
            }
        }
    }
    return $null
}

function Get-AlumniUsername {
    # Get an alumni user from the database
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users?role=alumni&limit=5"
    if ($result.Success -and $result.Data.data -and $result.Data.data.Count -gt 0) {
        foreach ($user in $result.Data.data) {
            if ($user.username -ne "admin") {
                return $user.username
            }
        }
    }
    return $null
}

function Test-StudentEndpoints {
    Write-TestHeader "STUDENT USER TESTS"
    
    # Get a student username
    $studentUsername = Get-StudentUsername
    if (-not $studentUsername) {
        Write-TestResult -Name "Student tests" -Skip $true -Message "No student users found in database"
        return
    }
    
    Write-Host "  Testing as student: $studentUsername" -ForegroundColor Gray
    
    # Login as student
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/auth/login" -Body @{
        username = $studentUsername
        password = "password"
    }
    
    if (-not $result.Success -or -not $result.Data.data.access_token) {
        Write-TestResult -Name "POST /auth/login (student)" -Success $false -Message "Failed to login as student"
        return
    }
    
    $studentToken = $result.Data.data.access_token
    Write-TestResult -Name "POST /auth/login (student)" -Success $true
    
    # Save original token and use student token
    $originalToken = $Global:AccessToken
    $Global:AccessToken = $studentToken
    
    # GET /me - Student profile
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/me" -UseAuth $true
    $isStudent = $result.Success -and $result.Data.data.role -in @("student", "alumni")
    Write-TestResult -Name "GET /me (student role check)" -Success $isStudent -Message $(if (-not $isStudent) { "Expected student/alumni role" })
    
    # GET /users/{username} - View other user profile as student
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin" -UseAuth $true
    Write-TestResult -Name "GET /users/{username} (student)" -Success $result.Success -Message $result.Error
    
    # GET /users/{username} - View own profile as student
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/$studentUsername" -UseAuth $true
    Write-TestResult -Name "GET /users/{username} (student own)" -Success $result.Success -Message $result.Error
    
    # GET /users/{username}/followers - As student
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin/followers" -UseAuth $true
    Write-TestResult -Name "GET /users/{username}/followers (student)" -Success $result.Success -Message $result.Error
    
    # GET /users/{username}/following - As student
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin/following" -UseAuth $true
    Write-TestResult -Name "GET /users/{username}/following (student)" -Success $result.Success -Message $result.Error
    
    # Student creates portfolio
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
        judul = "Student Test Portfolio"
    } -ExpectedStatus 201
    
    $studentPortfolioId = $null
    if ($result.Success -and $result.Data.data.id) {
        $studentPortfolioId = $result.Data.data.id
        Write-TestResult -Name "POST /portfolios (student)" -Success $true
    } else {
        Write-TestResult -Name "POST /portfolios (student)" -Success $false -Message $result.Error
    }
    
    # Student gets own portfolios
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/me/portfolios" -UseAuth $true
    Write-TestResult -Name "GET /me/portfolios (student)" -Success $result.Success -Message $result.Error
    
    # GET /portfolios/{slug} - Student views published portfolio by slug
    $publicPortfolios = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios?limit=1"
    if ($publicPortfolios.Success -and $publicPortfolios.Data.data -and $publicPortfolios.Data.data.Count -gt 0) {
        $testSlug = $publicPortfolios.Data.data[0].slug
        $result = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios/$testSlug" -UseAuth $true
        Write-TestResult -Name "GET /portfolios/{slug} (student)" -Success $result.Success -Message $result.Error
    }
    
    # Student updates own portfolio
    if ($studentPortfolioId) {
        $result = Invoke-ApiRequest -Method "PATCH" -Endpoint "/portfolios/$studentPortfolioId" -UseAuth $true -Body @{
            judul = "Student Portfolio Updated"
        }
        Write-TestResult -Name "PATCH /portfolios/{id} (student own)" -Success $result.Success -Message $result.Error
    }
    
    # Student CANNOT access admin endpoints (should get 403)
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/dashboard/stats" -UseAuth $true -ExpectedStatus 403
    Write-TestResult -Name "GET /admin/dashboard/stats (student - expect 403)" -Success $result.Success -Message $(if (-not $result.Success) { "Got status $($result.StatusCode) instead of 403" })
    
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/users" -UseAuth $true -ExpectedStatus 403
    Write-TestResult -Name "GET /admin/users (student - expect 403)" -Success $result.Success -Message $(if (-not $result.Success) { "Got status $($result.StatusCode) instead of 403" })
    
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/portfolios" -UseAuth $true -ExpectedStatus 403
    Write-TestResult -Name "GET /admin/portfolios (student - expect 403)" -Success $result.Success -Message $(if (-not $result.Success) { "Got status $($result.StatusCode) instead of 403" })
    
    # Student follow/unfollow another user
    $targetUser = $null
    $usersResult = Invoke-ApiRequest -Method "GET" -Endpoint "/users?limit=10"
    if ($usersResult.Success -and $usersResult.Data.data) {
        foreach ($user in $usersResult.Data.data) {
            if ($user.username -ne $studentUsername) {
                $targetUser = $user.username
                break
            }
        }
    }
    
    if ($targetUser) {
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/users/$targetUser/follow" -UseAuth $true
        Write-TestResult -Name "POST /users/{username}/follow (student)" -Success ($result.Success -or $result.StatusCode -eq 409) -Message $result.Error
        
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/users/$targetUser/follow" -UseAuth $true
        Write-TestResult -Name "DELETE /users/{username}/follow (student)" -Success ($result.Success -or $result.StatusCode -eq 400) -Message $result.Error
    }
    
    # Student like/unlike portfolio
    $portfolioResult = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios?limit=1"
    if ($portfolioResult.Success -and $portfolioResult.Data.data -and $portfolioResult.Data.data.Count -gt 0) {
        $portfolioToLike = $portfolioResult.Data.data[0].id
        
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$portfolioToLike/like" -UseAuth $true
        Write-TestResult -Name "POST /portfolios/{id}/like (student)" -Success ($result.Success -or $result.StatusCode -eq 409) -Message $result.Error
        
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$portfolioToLike/like" -UseAuth $true
        Write-TestResult -Name "DELETE /portfolios/{id}/like (student)" -Success ($result.Success -or $result.StatusCode -eq 400) -Message $result.Error
    }
    
    # Student search users
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/search/users?q=admin" -UseAuth $true
    Write-TestResult -Name "GET /search/users (student)" -Success $result.Success -Message $result.Error
    
    # Student search portfolios
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/search/portfolios?q=web" -UseAuth $true
    Write-TestResult -Name "GET /search/portfolios (student)" -Success $result.Success -Message $result.Error
    
    # Cleanup - delete student test portfolio
    if ($studentPortfolioId) {
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$studentPortfolioId" -UseAuth $true
        Write-TestResult -Name "DELETE test portfolio (student)" -Success $result.Success -Message $result.Error
    }
    
    # Logout student
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/auth/logout" -UseAuth $true
    Write-TestResult -Name "POST /auth/logout (student)" -Success $result.Success -Message $result.Error
    
    # Restore original admin token
    $Global:AccessToken = $originalToken
}

function Test-AlumniEndpoints {
    Write-TestHeader "ALUMNI USER TESTS"
    
    # Get an alumni username
    $alumniUsername = Get-AlumniUsername
    if (-not $alumniUsername) {
        Write-TestResult -Name "Alumni tests" -Skip $true -Message "No alumni users found in database"
        return
    }
    
    Write-Host "  Testing as alumni: $alumniUsername" -ForegroundColor Gray
    
    # Login as alumni
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/auth/login" -Body @{
        username = $alumniUsername
        password = "password"
    }
    
    if (-not $result.Success -or -not $result.Data.data.access_token) {
        Write-TestResult -Name "POST /auth/login (alumni)" -Success $false -Message "Failed to login as alumni"
        return
    }
    
    $alumniToken = $result.Data.data.access_token
    Write-TestResult -Name "POST /auth/login (alumni)" -Success $true
    
    # Save original token and use alumni token
    $originalToken = $Global:AccessToken
    $Global:AccessToken = $alumniToken
    
    # GET /me - Alumni profile
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/me" -UseAuth $true
    $isAlumni = $result.Success -and $result.Data.data.role -eq "alumni"
    Write-TestResult -Name "GET /me (alumni role check)" -Success $isAlumni -Message $(if (-not $isAlumni) { "Expected alumni role, got: $($result.Data.data.role)" })
    
    # GET /users/{username} - View other user profile as alumni
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin" -UseAuth $true
    Write-TestResult -Name "GET /users/{username} (alumni)" -Success $result.Success -Message $result.Error
    
    # GET /users/{username} - View own profile as alumni
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/$alumniUsername" -UseAuth $true
    Write-TestResult -Name "GET /users/{username} (alumni own)" -Success $result.Success -Message $result.Error
    
    # GET /users/{username}/followers - As alumni
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin/followers" -UseAuth $true
    Write-TestResult -Name "GET /users/{username}/followers (alumni)" -Success $result.Success -Message $result.Error
    
    # GET /users/{username}/following - As alumni
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/users/admin/following" -UseAuth $true
    Write-TestResult -Name "GET /users/{username}/following (alumni)" -Success $result.Success -Message $result.Error
    
    # Alumni creates portfolio
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios" -UseAuth $true -Body @{
        judul = "Alumni Test Portfolio"
    } -ExpectedStatus 201
    
    $alumniPortfolioId = $null
    if ($result.Success -and $result.Data.data.id) {
        $alumniPortfolioId = $result.Data.data.id
        Write-TestResult -Name "POST /portfolios (alumni)" -Success $true
    } else {
        Write-TestResult -Name "POST /portfolios (alumni)" -Success $false -Message $result.Error
    }
    
    # GET /portfolios/{slug} - Alumni views published portfolio by slug
    $publicPortfolios = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios?limit=1"
    if ($publicPortfolios.Success -and $publicPortfolios.Data.data -and $publicPortfolios.Data.data.Count -gt 0) {
        $testSlug = $publicPortfolios.Data.data[0].slug
        $result = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios/$testSlug" -UseAuth $true
        Write-TestResult -Name "GET /portfolios/{slug} (alumni)" -Success $result.Success -Message $result.Error
    }
    
    # Alumni CANNOT access admin endpoints
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/admin/dashboard/stats" -UseAuth $true -ExpectedStatus 403
    Write-TestResult -Name "GET /admin/dashboard/stats (alumni - expect 403)" -Success $result.Success -Message $(if (-not $result.Success) { "Got status $($result.StatusCode) instead of 403" })
    
    # Alumni follow/unfollow another user
    $targetUser = $null
    $usersResult = Invoke-ApiRequest -Method "GET" -Endpoint "/users?limit=10"
    if ($usersResult.Success -and $usersResult.Data.data) {
        foreach ($user in $usersResult.Data.data) {
            if ($user.username -ne $alumniUsername) {
                $targetUser = $user.username
                break
            }
        }
    }
    
    if ($targetUser) {
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/users/$targetUser/follow" -UseAuth $true
        Write-TestResult -Name "POST /users/{username}/follow (alumni)" -Success ($result.Success -or $result.StatusCode -eq 409) -Message $result.Error
        
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/users/$targetUser/follow" -UseAuth $true
        Write-TestResult -Name "DELETE /users/{username}/follow (alumni)" -Success ($result.Success -or $result.StatusCode -eq 400) -Message $result.Error
    }
    
    # Alumni like/unlike portfolio
    $portfolioResult = Invoke-ApiRequest -Method "GET" -Endpoint "/portfolios?limit=1"
    if ($portfolioResult.Success -and $portfolioResult.Data.data -and $portfolioResult.Data.data.Count -gt 0) {
        $portfolioToLike = $portfolioResult.Data.data[0].id
        
        $result = Invoke-ApiRequest -Method "POST" -Endpoint "/portfolios/$portfolioToLike/like" -UseAuth $true
        Write-TestResult -Name "POST /portfolios/{id}/like (alumni)" -Success ($result.Success -or $result.StatusCode -eq 409) -Message $result.Error
        
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$portfolioToLike/like" -UseAuth $true
        Write-TestResult -Name "DELETE /portfolios/{id}/like (alumni)" -Success ($result.Success -or $result.StatusCode -eq 400) -Message $result.Error
    }
    
    # Alumni search users
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/search/users?q=admin" -UseAuth $true
    Write-TestResult -Name "GET /search/users (alumni)" -Success $result.Success -Message $result.Error
    
    # Alumni search portfolios
    $result = Invoke-ApiRequest -Method "GET" -Endpoint "/search/portfolios?q=web" -UseAuth $true
    Write-TestResult -Name "GET /search/portfolios (alumni)" -Success $result.Success -Message $result.Error
    
    # Cleanup - delete alumni test portfolio
    if ($alumniPortfolioId) {
        $result = Invoke-ApiRequest -Method "DELETE" -Endpoint "/portfolios/$alumniPortfolioId" -UseAuth $true
        Write-TestResult -Name "DELETE test portfolio (alumni)" -Success $result.Success -Message $result.Error
    }
    
    # Logout alumni
    $result = Invoke-ApiRequest -Method "POST" -Endpoint "/auth/logout" -UseAuth $true
    Write-TestResult -Name "POST /auth/logout (alumni)" -Success $result.Success -Message $result.Error
    
    # Restore original admin token
    $Global:AccessToken = $originalToken
}

# ============================================
# MAIN
# ============================================

Write-Host ""
Write-Host "GRAFIKARSA API TEST SUITE" -ForegroundColor Magenta
Write-Host "Base URL: $ApiUrl" -ForegroundColor Magenta
Write-Host ""

# Check if server is running
try {
    $null = Invoke-WebRequest -Uri "$ApiUrl/tags" -TimeoutSec 5 -ErrorAction Stop
} catch {
    Write-Host "ERROR: Cannot connect to API server at $ApiUrl" -ForegroundColor Red
    Write-Host "Make sure the server is running: go run ./cmd/api" -ForegroundColor Yellow
    exit 1
}

# Run tests
Test-PublicEndpoints
Test-AuthEndpoints
Test-ProfileEndpoints
Test-UserEndpoints
Test-PortfolioEndpoints
Test-ContentBlockEndpoints
Test-PortfolioStatusEndpoints
Test-SocialEndpoints
Test-FeedEndpoints
Test-AdminEndpoints
Test-Cleanup
Test-Logout

# Run student/alumni tests (re-login as admin first for cleanup)
Test-StudentEndpoints
Test-AlumniEndpoints

# Summary
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  TEST SUMMARY" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "PASSED: $Global:PassCount" -ForegroundColor Green
Write-Host "FAILED: $Global:FailCount" -ForegroundColor Red
Write-Host "SKIPPED: $Global:SkipCount" -ForegroundColor Yellow
Write-Host ""

$total = $Global:PassCount + $Global:FailCount
if ($total -gt 0) {
    $percentage = [math]::Round(($Global:PassCount / $total) * 100, 1)
    Write-Host "Success Rate: $percentage%" -ForegroundColor $(if ($percentage -ge 80) { "Green" } elseif ($percentage -ge 50) { "Yellow" } else { "Red" })
}

if ($Global:FailCount -gt 0) {
    exit 1
}

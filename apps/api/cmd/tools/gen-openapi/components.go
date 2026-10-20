package main

// componentsYAML adalah bagian components: dari OpenAPI, ditulis manual
// (bukan dari routes.go) karena berisi skema domain.
const componentsYAML = `components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: >-
        Access token 15 menit dari POST /api/v1/auth/login (disimpan di memori
        FE, bukan localStorage). Refresh token ada di cookie __Host-rt HttpOnly.
  responses:
    BadRequest:
      description: >-
        Format request tidak dikenali (body JSON rusak, parameter tidak valid).
        Content-Type bisa text/plain kalau framework menolak URL-nya sebelum
        aplikasi sempat menjawab (mis. percent-encoding rusak).
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          example:
            success: false
            error: { code: VALIDATION_ERROR, message: Body JSON tidak valid. }
        text/plain:
          schema: { type: string }
          example: "400 Bad Request"
    NotFound:
      description: Sumber daya tidak ada (atau tidak terlihat oleh pemanggil)
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          example:
            success: false
            error: { code: NOT_FOUND, message: Tidak ditemukan. }
    MethodNotAllowed:
      description: >-
        Method HTTP tidak diizinkan untuk path ini. Router mengembalikan 405
        kalau path-nya dikenali tapi method-nya tidak (mis. GET ke endpoint
        yang hanya menerima POST).
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          example:
            success: false
            error: { code: METHOD_NOT_ALLOWED, message: Method tidak diizinkan. }
    Gone:
      description: >-
        Sumber daya sudah tidak tersedia. Untuk route "/storage/*" ini berarti
        URL lama v1 terdaftar, tapi file-nya sudah tidak ada (masa redirect
        6 bulan berakhir).
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          example:
            success: false
            error: { code: GONE, message: File lama sudah tidak tersedia. }
    Conflict:
      description: Konflik dengan data yang sudah ada (duplikat)
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          example:
            success: false
            error: { code: DUPLICATE_USER, message: Sudah ada. }
    Unauthorized:
      description: Belum login atau token kedaluwarsa
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          example:
            success: false
            error: { code: UNAUTHORIZED, message: Sesi berakhir. Silakan login ulang. }
    Forbidden:
      description: >-
        Tidak punya permission, di luar scope jurusan, CSRF tidak valid, atau
        wajib ganti password lebih dulu.
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          examples:
            scope:
              value:
                success: false
                error: { code: FORBIDDEN_SCOPE, message: Kamu tidak memiliki akses ini. }
            csrf:
              value:
                success: false
                error: { code: CSRF_INVALID, message: Token keamanan tidak valid. }
            password:
              value:
                success: false
                error: { code: PASSWORD_CHANGE_REQUIRED, message: Ganti password dulu sebelum melanjutkan. }
    ValidationError:
      description: Body/parameter tidak lolos validasi
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          example:
            success: false
            error:
              code: VALIDATION_ERROR
              message: Data tidak valid.
              details: [{ field: title, message: 3-200 karakter. }]
    TooManyRequests:
      description: Kena rate limit (per user/IP) atau lockout login
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
          examples:
            rate:
              value:
                success: false
                error: { code: TOO_MANY_REQUESTS, message: Terlalu banyak permintaan. Coba lagi sebentar. }
            captcha:
              value:
                success: false
                error: { code: CAPTCHA_REQUIRED, message: Verifikasi keamanan diperlukan. }
            locked:
              value:
                success: false
                error: { code: ACCOUNT_LOCKED, message: Terlalu banyak percobaan. Coba lagi 15 menit. }
    InternalError:
      description: Kesalahan server (detail hanya di log)
      content:
        application/json:
          schema: { $ref: "#/components/schemas/ErrorEnvelope" }
  schemas:
    SuccessEnvelope:
      type: object
      required: [success, data]
      properties:
        success: { type: boolean, const: true }
        data: { description: Payload spesifik endpoint }
        meta:
          $ref: "#/components/schemas/PaginationMeta"
        request_id: { type: string }
    ErrorEnvelope:
      type: object
      required: [success, error]
      properties:
        success: { type: boolean, const: false }
        error:
          type: object
          required: [code, message]
          properties:
            code:
              $ref: "#/components/schemas/ErrorCode"
            message: { type: string }
            details:
              type: array
              items:
                type: object
                properties:
                  field: { type: string }
                  message: { type: string }
        request_id: { type: string }
    ErrorCode:
      type: string
      description: Kode error stabil untuk ditangani klien
      enum:
        - UNAUTHORIZED
        - TOKEN_REVOKED
        - TOKEN_REUSE_DETECTED
        - SESSION_EXPIRED
        - INVALID_CREDENTIALS
        - ACCOUNT_LOCKED
        - CAPTCHA_REQUIRED
        - PASSWORD_CHANGE_REQUIRED
        - FORBIDDEN_SCOPE
        - CSRF_INVALID
        - VALIDATION_ERROR
        - NOT_FOUND
        - DUPLICATE_USER
        - DUPLICATE_VERSION
        - INVALID_STATUS_TRANSITION
        - INCOMPLETE_PORTFOLIO
        - QUOTA_EXCEEDED
        - FILE_TOO_LARGE
        - INVALID_CONTENT_TYPE
        - TOO_MANY_UPLOADS
        - LIKE_RATE_LIMITED
        - TOO_MANY_REQUESTS
        - TAG_IN_USE
        - METHOD_NOT_ALLOWED
        - INTERNAL_ERROR
        - ALREADY_HAS_BLOCKS
        - CANNOT_FOLLOW_SELF
        - CLASS_NOT_EMPTY
        - DUPLICATE_CLASS
        - DUPLICATE_JOB
        - DUPLICATE_MAJOR
        - DUPLICATE_ROLE
        - DUPLICATE_SLUG
        - DUPLICATE_YEAR
        - GONE
        - NO_ACTIVE_YEAR
        - OBJECT_MISSING
        - SERVICE_UNAVAILABLE
        - USERNAME_TAKEN
    PaginationMeta:
      # Nullable: endpoint yang tidak berpaginasi (leaderboard, feed ringkas)
      # mengirim meta: null. Kalau tidak nullable, contract test menandai
      # "null is not of type object" padahal itu perilaku yang disengaja.
      type: object
      nullable: true
      properties:
        page: { type: integer }
        per_page: { type: integer }
        total: { type: integer }
        total_pages: { type: integer }
    User:
      type: object
      properties:
        id: { type: string, format: uuid }
        username: { type: string }
        name: { type: string }
        email: { type: string, format: email, description: Hanya self/admin/show_email }
        role_primary: { type: string, enum: [siswa, guru, admin_jurusan, admin] }
        status: { type: string, enum: [aktif, lulus, nonaktif] }
        bio: { type: string }
        avatar_media_id: { type: string, nullable: true }
        banner_media_id: { type: string, nullable: true }
        must_change_password: { type: boolean }
    Portfolio:
      type: object
      properties:
        id: { type: string, format: uuid }
        title: { type: string }
        slug: { type: string }
        status:
          type: string
          enum: [draft, pending_review, published, rejected, changes_requested, archived]
        owner_id: { type: string, format: uuid }
        thumbnail_media_id: { type: string, nullable: true }
        tags:
          type: array
          items: { type: string }
        blocks:
          type: array
          items: { $ref: "#/components/schemas/PortfolioBlock" }
        published_at: { type: string, format: date-time, nullable: true }
    PortfolioBlock:
      type: object
      properties:
        id: { type: string, format: uuid }
        block_type:
          type: string
          enum: [text, image, table, youtube, button, embed, figma, canva, ppt, pdf, doc, link]
        position: { type: integer }
        payload: { type: object }
        asset_media_id: { type: string, nullable: true }
        is_from_series: { type: boolean }
    Media:
      type: object
      properties:
        media_id: { type: string, format: uuid }
        object_key: { type: string }
        mime: { type: string }
        width: { type: integer }
        height: { type: integer }
        purpose:
          type: string
          enum: [avatar, banner, thumbnail, block_image, document, feedback, changelog_cover]
        status: { type: string, enum: [pending, confirmed, deleted, quarantined] }
        variants:
          type: object
          description: Lebar (px) -> object key varian
          additionalProperties: { type: string }
    PortfolioStats:
      type: object
      properties:
        portfolio_id: { type: string, format: uuid }
        quality: { type: number }
        engagement: { type: number }
        trending: { type: number }
        assessment_avg: { type: number }
        computed_at: { type: string, format: date-time }
    LeaderboardEntry:
      type: object
      properties:
        rank: { type: integer }
        score: { type: number }
        delta_rank: { type: integer }
        breakdown: { type: object }
        name: { type: string }
        username: { type: string }
        title: { type: string }
        slug: { type: string }
    FeedbackThread:
      type: object
      properties:
        id: { type: string, format: uuid }
        category: { type: string, enum: [bug, idea, question, praise] }
        title: { type: string }
        body: { type: string }
        status: { type: string, enum: [open, triaged, in_progress, resolved, closed, duplicate] }
        priority: { type: string, enum: [low, medium, high, critical] }
        upvotes: { type: integer }
        sla_hours: { type: integer }
        sla_breached: { type: boolean }
        is_public: { type: boolean }
    Changelog:
      type: object
      properties:
        id: { type: string, format: uuid }
        version: { type: string }
        title: { type: string }
        summary: { type: string }
        channel: { type: string, enum: [stable, beta, hotfix] }
        audience: { type: string, enum: [all, siswa, guru, admin] }
        status: { type: string, enum: [draft, scheduled, published, archived] }
        sections:
          type: array
          items:
            type: object
            properties:
              kind: { type: string, enum: [added, improved, fixed, removed] }
              title: { type: string }
              body_md: { type: string }
    Notification:
      type: object
      properties:
        id: { type: string, format: uuid }
        type:
          type: string
          enum:
            - portfolio_liked
            - new_comment
            - reply_comment
            - new_follower
            - portfolio_approved
            - portfolio_rejected
            - portfolio_changes_requested
            - feedback_updated
            - announcement
            - digest
            - leaderboard_ranked
        title: { type: string }
        body: { type: string }
        is_read: { type: boolean }
        created_at: { type: string, format: date-time }
`
